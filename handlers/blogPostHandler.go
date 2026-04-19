package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	log "htmx-blog/logging"
	"htmx-blog/services/cache"
	"htmx-blog/services/content"
	"htmx-blog/utils"
)

const defaultPageSize = 10
const maxPageSize = 100

// buildPageList returns page numbers to show in the nav; 0 means ellipsis.
func buildPageList(current, totalPages int) []int {
	if totalPages <= 0 {
		return nil
	}
	if totalPages <= 7 {
		out := make([]int, totalPages)
		for i := range out {
			out[i] = i + 1
		}
		return out
	}
	var out []int
	out = append(out, 1)
	// window around current: current-2 to current+2, clamped
	lo := current - 2
	if lo < 2 {
		lo = 2
	}
	hi := current + 2
	if hi > totalPages-1 {
		hi = totalPages - 1
	}
	if lo > 2 {
		out = append(out, 0)
	}
	for p := lo; p <= hi; p++ {
		out = append(out, p)
	}
	if hi < totalPages-1 {
		out = append(out, 0)
	}
	if totalPages > 1 {
		out = append(out, totalPages)
	}
	return out
}

// sectionTitle converts a URL filter slug into a human-readable heading.
func sectionTitle(filter string) string {
	titles := map[string]string{
		"book-reviews": "Book Reviews",
		"engineering":  "Coding",
		"travel":       "Travel",
		"speaking":     "Speaking",
	}
	if t, ok := titles[filter]; ok {
		return t
	}
	// Fallback: capitalize and replace hyphens
	if filter == "" {
		return "Posts"
	}
	return strings.ToUpper(filter[:1]) + strings.ReplaceAll(filter[1:], "-", " ")
}

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

// Pagination holds data for list pagination.
// Pages is the list of page numbers to show; 0 means ellipsis.
type Pagination struct {
	Page       int
	Limit      int
	TotalCount int
	TotalPages int
	HasPrev    bool
	HasNext    bool
	PrevPage   int
	NextPage   int
	Filter     string
	Pages      []int
}

// ListPosts returns a handler that renders the list of posts for the given filter with pagination.
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

		page := 1
		if p := r.URL.Query().Get("page"); p != "" {
			if n, err := strconv.Atoi(p); err == nil && n > 0 {
				page = n
			}
		}
		limit := defaultPageSize
		if l := r.URL.Query().Get("limit"); l != "" {
			if n, err := strconv.Atoi(l); err == nil && n > 0 {
				if n > maxPageSize {
					n = maxPageSize
				}
				limit = n
			}
		}

		total := len(postEntries)
		totalPages := 1
		if limit > 0 {
			totalPages = (total + limit - 1) / limit
		}
		if totalPages < 1 {
			totalPages = 1
		}
		if page > totalPages {
			page = totalPages
		}

		start := (page - 1) * limit
		end := start + limit
		if start > total {
			start = total
		}
		if end > total {
			end = total
		}
		pageEntries := postEntries[start:end]

		pagination := Pagination{
			Page:       page,
			Limit:      limit,
			TotalCount: total,
			TotalPages: totalPages,
			HasPrev:    page > 1,
			HasNext:    page < totalPages,
			PrevPage:   page - 1,
			NextPage:   page + 1,
			Filter:     filter,
			Pages:      buildPageList(page, totalPages),
		}

		utils.Render(w, map[string]interface{}{
			"BlogEntries":  pageEntries,
			"Pagination":   pagination,
			"SectionTitle": sectionTitle(filter),
		}, "./templates/pages/notion-list.html", "./templates/partials/post-entry.html")
	}
}

// GetPostPage returns a handler that serves the post page shell (used when navigating to a post by subtitle/slug;
// actual content is loaded via htmx from GetPostContent). Renders a 404 page if the slug does not exist.
func (h *BlogPostHandler) GetPostPage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		subtitle := r.PathValue("slug")
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
		slug := r.PathValue("slug")
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
