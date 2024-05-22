package notionHandler

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	log "htmx-blog/logging"
	"htmx-blog/models"
	"htmx-blog/services/cache"
	"htmx-blog/services/notion"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/redis/go-redis/v9"
)

type notionHandler struct {
	cache        cache.Cache
	notionClient notion.NotionClient
	redisClient  *redis.Client
}

func NewHandler(notionClient notion.NotionClient, redisClient *redis.Client) NotionHandler {
	/* if os.Getenv("DEV") == "true" {
		return &notionHandler{
			notionClient: notionClient,
			cache:        cache.NewInMemoryCache(),
		}
	} */
	return &notionHandler{
		notionClient: notionClient,
		redisClient:  redisClient,
		cache:        cache.NewCache(redisClient, notionClient),
	}
}

type NotionHandler interface {
	GetAllPosts() http.HandlerFunc
	GetSinglePost() http.HandlerFunc
	RenderPostHTML() http.HandlerFunc
}

// GetAllPosts implements NotionHandler.
func (n *notionHandler) GetAllPosts() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filter := r.PathValue("filter")
		databaseID := n.notionClient.GetDatabaseID()

		// cache logic is handled internally in the cache package
		slugEntries, err := n.cache.GetSlugEntries(r.Context(), databaseID, filter)
		if err != nil {
			log.Error("error getting slug entries: %v", err)
			w.Write([]byte("error getting slug entries"))
		}
		err = WriteNotionSlugEntriesToHTML(r.Context(), w, slugEntries)
		if err != nil {
			log.Error("error writing notion slug entries to html: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

func (n *notionHandler) RenderPostHTML() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// get post id from request
		// TODO the blockID will be the slug name, and conversion of slug name to blockID done here
		path := r.URL.Path
		segments := strings.Split(path, "/")
		blockID := segments[len(segments)-1]

		tmpl, err := template.ParseFiles("./templates/notionPost.html")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		err = tmpl.Execute(w, blockID)
		if err != nil {
			log.Error("error executing template: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
}

// GetSinglePost is called when routing to a page with a single notion post
func (n *notionHandler) GetSinglePost() http.HandlerFunc {
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

// StoreNotionImage stores the image locally and updates the rawBlock with the new image url
// first it will get the existing fresh image url from notion aws image url
// then it will download the image from the aws image url
// then it will store the image locally
func StoreNotionImage(rawBlocks []json.RawMessage, i int) error {
	var imageBlock models.Image
	err := json.Unmarshal(rawBlocks[i], &imageBlock)
	if err != nil {
		log.Error("error unmarshalling imageblock: %v", err)
	}
	awsImageURL := imageBlock.Image.File.URL
	// read and write image to r2, then update the rawBlock with the new image url
	// Download file from S3
	resp, err := http.Get(awsImageURL)
	if err != nil {
		return fmt.Errorf("error downloading image from s3: %v", err)
	}
	defer resp.Body.Close()

	imageBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading image bytes: %v", err)
	}
	// store locally in ./images
	// Ensure the folder exists
	absPath, err := filepath.Abs("./images")
	if err != nil {
		return fmt.Errorf("error getting absolute path: %v", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		os.Mkdir(absPath, os.ModePerm)
	}

	filePath := filepath.Join(absPath, "/", imageBlock.ID+".png")
	err = os.WriteFile(filePath, imageBytes, 0755)
	if err != nil {
		return fmt.Errorf("error writing image to file: %v", err)
	}

	imageBlock.Image.File.URL = "https://cloud.shaikzhafir.com/images/" + imageBlock.ID + ".png"
	if os.Getenv("DEV") == "true" {
		imageBlock.Image.File.URL = "/images/" + imageBlock.ID + ".png"
	}

	// update rawBlock with new image url
	rawBlocks[i], err = json.Marshal(imageBlock)
	if err != nil {
		return fmt.Errorf("error marshalling imageblock: %v", err)
	}
	return nil
}

func WriteNotionSlugEntriesToHTML(ctx context.Context, w http.ResponseWriter, slugEntries []notion.SlugEntry) error {
	// convert posts to html template!
	tmpl, err := template.ParseFiles("./templates/slugEntry.html")
	if err != nil {
		return err
	}
	for _, entry := range slugEntries {
		err = tmpl.Execute(w, entry)
		if err != nil {
			// dont have to return error here, just log it as maybe some posts have issues
			log.Error("error executing template for specific entry: %v", err)
		}
	}
	return nil
}
