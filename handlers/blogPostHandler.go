package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	log "htmx-blog/logging"
	notionModels "htmx-blog/models"
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

func (h *BlogPostHandler) GetSinglePost() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		path := r.URL.Path
		segments := strings.Split(path, "/")
		blockID := segments[len(segments)-1]

		deserialized, err := h.cache.GetPostByID(ctx, blockID)
		if err != nil {
			log.Error("error getting post by id: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		//New approach to rendering the page, loop through raw blocks and append the html to a string builder
		var htmlBuilder strings.Builder
		for _, rawBlock := range deserialized {
			var b notionModels.Block
			err = json.Unmarshal(rawBlock, &b)
			if err != nil {
				log.Error("error unmarshalling rawblock: %v", err)
				continue
			}
			err = h.notionClient.ParseAndWriteNotionBlock(&htmlBuilder, rawBlock)
			if err != nil {
				w.Write([]byte("error parsing block oopsie"))
				return
			}
		}

		//Write the complete HTML to the response, using the notionPost template
		utils.Render(w, map[string]interface{}{"Content": htmlBuilder.String()}, "./templates/blogEntries.html")
	}
}
