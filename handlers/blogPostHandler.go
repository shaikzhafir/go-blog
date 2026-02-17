package handlers

import (
	"errors"
	"net/http"
	"strings"

	log "htmx-blog/logging"
	"htmx-blog/services/cache"
	"htmx-blog/services/content"
	"htmx-blog/utils"
)

type BlogPostHandler struct {
	cache         cache.Cache
	pageRenderer content.PageRenderer
}

// NewBlogPostHandler creates a handler that uses cache for list views and
// pageRenderer for rendering single posts. Both are backed by the content
// abstraction, so the data source (Notion, Markdown, etc.) can be swapped.
func NewBlogPostHandler(cache cache.Cache, pageRenderer content.PageRenderer) *BlogPostHandler {
	return &BlogPostHandler{
		cache:         cache,
		pageRenderer: pageRenderer,
	}
}

// ListPosts returns a handler that renders the list of posts for the given filter.
func (h *BlogPostHandler) ListPosts() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filter := r.PathValue("filter")
		collectionID := h.cache.GetSource().GetDefaultCollectionID()

		postEntries, err := h.cache.GetPostEntries(r.Context(), collectionID, filter)
		if err != nil {
			log.Error("error getting post entries: %v", err)
			w.Write([]byte("error getting post entries"))
			return
		}

		if filter != "" {
			for i := range postEntries {
				postEntries[i].PostType = filter
			}
		}

		utils.Render(w, map[string]interface{}{"BlogEntries": postEntries}, "./templates/pages/notion-list.html", "./templates/partials/post-entry.html")
	}
}

// GetPostPage returns a handler that serves the post page shell (used when navigating to a post by subtitle/slug;
// actual content is loaded via htmx from GetPostContent). Renders a 404 page if the slug does not exist.
func (h *BlogPostHandler) GetPostPage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		segments := strings.Split(path, "/")
		subtitle := segments[len(segments)-1]
		postType := r.URL.Query().Get("type")

		collectionID := h.cache.GetSource().GetDefaultCollectionID()
		_, err := h.cache.GetBlockIDBySlug(r.Context(), collectionID, subtitle, postType)
		if err != nil {
			if errors.Is(err, cache.ErrSlugNotFound) {
				w.WriteHeader(http.StatusNotFound)
				utils.Render(w, nil, "./templates/pages/not-found.html")
				return
			}
			log.Error("error resolving slug for post page: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("error loading post"))
			return
		}

		utils.Render(w, map[string]interface{}{"Slug": subtitle, "PostType": postType}, "./templates/pages/notion-post.html")
	}
}

// GetPostContent returns a handler that renders a single post's content (used by htmx to swap
// into the post page). The URL segment is the post subtitle (slug); it is resolved to a block ID
// via the cache. Uses the content PageRenderer interface, so the backend is interchangeable.
func (h *BlogPostHandler) GetPostContent() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		segments := strings.Split(path, "/")
		slug := segments[len(segments)-1]
		postType := r.URL.Query().Get("type")

		collectionID := h.cache.GetSource().GetDefaultCollectionID()
		blockID, err := h.cache.GetBlockIDBySlug(r.Context(), collectionID, slug, postType)
		if err != nil {
			if errors.Is(err, cache.ErrSlugNotFound) {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte("post not found"))
				return
			}
			log.Error("error resolving slug: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("error loading post"))
			return
		}

		err = h.pageRenderer.RenderPage(r.Context(), w, blockID, content.RenderOptions{PostType: postType})
		if err != nil {
			log.Error("error rendering post: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("error rendering post"))
			return
		}
	}
}
