package handlers

import (
	"net/http"
	"strings"

	log "htmx-blog/logging"
	"htmx-blog/services/cache"
	"htmx-blog/services/notion"
	"htmx-blog/utils"
)

type BlogPostHandler struct {
	cache        cache.Cache
	notionClient notion.NotionClient
}

func NewBlogPostHandler(notionClient notion.NotionClient, cache cache.Cache) *BlogPostHandler {
	return &BlogPostHandler{
		cache:        cache,
		notionClient: notionClient,
	}
}

func (h *BlogPostHandler) GetAllPosts() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filter := r.PathValue("filter")
		databaseID := h.notionClient.GetDatabaseID()

		slugEntries, err := h.cache.GetSlugEntries(r.Context(), databaseID, filter)
		if err != nil {
			log.Error("error getting slug entries: %v", err)
			w.Write([]byte("error getting slug entries"))
			return
		}

		utils.Render(w, map[string]interface{}{"BlogEntries": slugEntries}, "./templates/blogEntries.html", "./templates/slugEntry.html")
	}
}

func (n *BlogPostHandler) RenderPostHTML() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// get post id from request
		// TODO the blockID will be the slug name, and conversion of slug name to blockID done here
		path := r.URL.Path
		segments := strings.Split(path, "/")
		blockID := segments[len(segments)-1]
		render(w, map[string]interface{}{"BlockID": blockID}, "./templates/notionPost.html")
	}
}

// GetSinglePost is called when routing to a page with a single notion post
func (n *BlogPostHandler) GetSinglePost() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		// get post id from request
		path := r.URL.Path
		segments := strings.Split(path, "/")
		blockID := segments[len(segments)-1]

		deserialized, err := n.cache.GetPostByID(ctx, blockID)
		if err != nil {
			log.Error("error getting post by id: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		for _, rawBlock := range deserialized {
			err := n.notionClient.ParseAndWriteNotionBlock(w, rawBlock)
			if err != nil {
				w.Write([]byte("error parsing block oopsie"))
			}
		}

		// get post from notion
		// it should return a list of rawblocks

		// first check if blockID exists in redis cache
		// if it does, return the cached html
		// if it doesn't, get the rawblocks from notion

	}
}
