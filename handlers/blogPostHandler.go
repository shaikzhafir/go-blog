package handlers

import (
	"net/http"
	"strings"

	log "htmx-blog/logging"
	"htmx-blog/services/cache"
	"htmx-blog/services/content"
	"htmx-blog/utils"
)

type BlogPostHandler struct {
	cache         cache.Cache
	blockRenderer content.BlockRenderer
}

func NewBlogPostHandler(cache cache.Cache, renderer content.BlockRenderer) *BlogPostHandler {
	return &BlogPostHandler{
		cache:         cache,
		blockRenderer: renderer,
	}
}

func (h *BlogPostHandler) GetAllPosts() http.HandlerFunc {
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

		utils.Render(w, map[string]interface{}{"BlogEntries": postEntries}, "./templates/blogEntries.html", "./templates/slugEntry.html")
	}
}

func (h *BlogPostHandler) RenderPostHTML() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// get post id from request
		// TODO the blockID will be the slug name, and conversion of slug name to blockID done here
		path := r.URL.Path
		segments := strings.Split(path, "/")
		blockID := segments[len(segments)-1]
		postType := r.URL.Query().Get("type")
		utils.Render(w, map[string]interface{}{"BlockID": blockID, "PostType": postType}, "./templates/notionPost.html")
	}
}

// GetSinglePost is called when routing to a page with a single notion post
func (h *BlogPostHandler) GetSinglePost() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		// get post id from request
		path := r.URL.Path
		segments := strings.Split(path, "/")
		blockID := segments[len(segments)-1]
		postType := r.URL.Query().Get("type")

		blocks, err := h.cache.GetBlockChildren(ctx, blockID)
		if err != nil {
			log.Error("error getting post by id: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		for _, rawBlock := range blocks {
			err := h.blockRenderer.RenderBlock(w, rawBlock, postType)
			if err != nil {
				w.Write([]byte("error parsing block oopsie"))
			}
		}
	}
}
