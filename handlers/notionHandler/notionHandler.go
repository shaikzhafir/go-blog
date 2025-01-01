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

	"github.com/pkg/errors"
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
	Index() http.HandlerFunc
	GetAllPosts() http.HandlerFunc
	GetSinglePost() http.HandlerFunc
	RenderPostHTML() http.HandlerFunc
	GetReadingNowHandler() http.HandlerFunc
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
		render(w, map[string]interface{}{"BlogEntries": slugEntries}, "./templates/blogEntries.html", "./templates/slugEntry.html")
	}
}

func (n *notionHandler) RenderPostHTML() http.HandlerFunc {
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

func (n *notionHandler) GetReadingNowHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		blockID := os.Getenv("READING_NOW_BLOCK_ID")
		readingNowBlocks, err := n.GetReadingNow(r.Context(), blockID)
		if err != nil {
			log.Error("error getting reading now blocks: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// render standalone html template
		tmpl, err := template.ParseFiles("./templates/readingNow.html")
		if err != nil {
			log.Error("error parsing template: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		err = tmpl.Execute(w, map[string]interface{}{
			"Books": readingNowBlocks,
		})
		if err != nil {
			log.Error("error executing template: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

	}
}

// Get ReadingNow is custom and called to render the reading now blocks
func (n *notionHandler) GetReadingNow(ctx context.Context, blockID string) ([]models.ReadingNowBlock, error) {
	// first check if blockID exists in redis cache
	// if it does, return the cached html
	// if it doesn't, get the rawblocks from notion
	rawBlocks, err := n.cache.GetReadingNowPage(ctx, blockID)
	if err != nil {
		return nil, fmt.Errorf("error getting block children: %v", err)
	}
	// parse rawBlocks into reading book objects
	// then render the reading book objects into html
	// then return the html
	readingNowBlocks := []models.ReadingNowBlock{}
	var currentBook models.ReadingNowBlock
	for i := range rawBlocks {
		var b models.Block
		err := json.Unmarshal(rawBlocks[i], &b)
		if err != nil {
			log.Error("error unmarshalling rawblock: %v", err)
			continue
		}
		switch b.Type {
		case "divider":
			if i != 0 {
				readingNowBlocks = append(readingNowBlocks, currentBook)
			}
			currentBook = models.ReadingNowBlock{}
		case "heading_1":
			// unmarshal into heading1 block
			var heading1Block models.Heading1
			err := json.Unmarshal(rawBlocks[i], &heading1Block)
			if err != nil {
				log.Error("error unmarshalling heading1 block: %v", err)
				continue
			}
			currentBook.Title = heading1Block.Heading1.Text[0].Text.Content
		case "heading_2":
			// unmarshal into heading2 block
			var heading2Block models.Heading2
			err := json.Unmarshal(rawBlocks[i], &heading2Block)
			if err != nil {
				log.Error("error unmarshalling heading2 block: %v", err)
				continue
			}
			currentBook.Author = heading2Block.Heading2.Text[0].Text.Content
		case "heading_3":
			// unmarshal into heading3 block
			var heading3Block models.Heading3
			err := json.Unmarshal(rawBlocks[i], &heading3Block)
			if err != nil {
				log.Error("error unmarshalling heading3 block: %v", err)
				continue
			}
			currentBook.Progress = heading3Block.Heading3.Text[0].Text.Content
		case "image":
			var block models.Image
			err := json.Unmarshal(rawBlocks[i], &block)
			if err != nil {
				log.Error("error unmarshalling image block: %v", err)
				continue
			}
			currentBook.ImageURL = block.Image.File.URL
		case "paragraph":
			var block models.Paragraph
			err := json.Unmarshal(rawBlocks[i], &block)
			if err != nil {
				log.Error("error unmarshalling paragraph block: %v", err)
				continue
			}
			currentBook.Comments = block.Paragraph.RichText[0].PlainText
		}
	}
	return readingNowBlocks, nil
}

// StoreNotionImage stores the image locally and updates the rawBlock with the new image url
// first it will get the existing fresh image url from notion aws image url
// then it will download the image from the aws image url
// then it will store the image locally
func StoreNotionImage(rawBlocks []json.RawMessage, i int) (string, error) {
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
		return "", fmt.Errorf("error downloading image from s3: %v", err)
	}
	defer resp.Body.Close()

	imageBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading image bytes: %v", err)
	}
	// store locally in ./images
	// Ensure the folder exists
	absPath, err := filepath.Abs("./images")
	if err != nil {
		return "", fmt.Errorf("error getting absolute path: %v", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		os.Mkdir(absPath, os.ModePerm)
	}

	filePath := filepath.Join(absPath, "/", imageBlock.ID+".png")
	err = os.WriteFile(filePath, imageBytes, 0755)
	if err != nil {
		return "", fmt.Errorf("error writing image to file: %v", err)
	}

	imageBlock.Image.File.URL = "https://cloud.shaikzhafir.com/images/" + imageBlock.ID + ".png"
	if os.Getenv("DEV") == "true" {
		imageBlock.Image.File.URL = "/images/" + imageBlock.ID + ".png"
	}

	// update rawBlock with new image url
	rawBlocks[i], err = json.Marshal(imageBlock)
	if err != nil {
		return "", fmt.Errorf("error marshalling imageblock: %v", err)
	}
	return imageBlock.Image.File.URL, nil
}

// Index is the main page for the notion handler
func (n *notionHandler) Index() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		render(w, nil, "./templates/home.page.html")
	}
}

func render(w http.ResponseWriter, data map[string]interface{}, paths ...string) {
	paths = append(paths, "./templates/main.layout.html")
	tmpl, err := template.ParseFiles(paths...)
	if err != nil {
		http.Error(w, errors.Wrap(err, "failed to render html page").Error(), http.StatusInternalServerError)
		return
	}

	err = tmpl.ExecuteTemplate(w, "main", data)
	if err != nil {
		http.Error(w, errors.Wrap(err, "failed to render html page").Error(), http.StatusInternalServerError)
	}
}
