package notionHandler

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	log "htmx-blog/logging"
	"htmx-blog/models"
	"htmx-blog/services/notion"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type notionHandler struct {
	notionClient notion.NotionClient
	redisClient  *redis.Client
}

func NewHandler(notionClient notion.NotionClient, redisClient *redis.Client) NotionHandler {
	return &notionHandler{
		notionClient: notionClient,
		redisClient:  redisClient,
	}
}

type NotionHandler interface {
	GetAllPosts() http.HandlerFunc
	GetSinglePost() http.HandlerFunc
	RenderSinglePostPage() http.HandlerFunc
}

// GetAllPosts implements NotionHandler.
func (n *notionHandler) GetAllPosts() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		databaseID := n.notionClient.GetDatabaseID()
		// check if slugentries exist in redis cache
		// if it does, return the cached html
		exists, err := n.redisClient.Exists(r.Context(), databaseID).Result()
		if err != nil {
			log.Error("error checking if key exists: %v", err)
		}
		if exists == 1 {
			log.Info("cache hit")
			// get cached content from redis
			cachedJSON, err := n.redisClient.Get(r.Context(), databaseID).Bytes()
			if err != nil {
				log.Error("error getting cached content from redis: %v", err)
			}
			var slugEntries []notion.SlugEntry
			err = json.Unmarshal(cachedJSON, &slugEntries)
			if err != nil {
				log.Error("Failed to deserialize: %v", err)
			}
			tmpl, err := template.ParseFiles("./templates/slugEntry.html")
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			for _, entry := range slugEntries {
				err = tmpl.Execute(w, entry)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
			}
			// check expiry of cached content
			// if expired, update cache
			// if not expired, do nothing
			timestamp, err := n.redisClient.Get(r.Context(), databaseID+"-timestamp").Time()
			if err != nil {
				log.Error("error getting timestamp: %v", err)
				return
			}
			// if timestamp is more than 1 hour ago, update cache
			if time.Since(timestamp) > time.Hour {
				// update cache
				_, err = n.StoreOrUpdateCacheQueryDB(r.Context(), databaseID)
				if err != nil {
					log.Error("error storing or updating cache: %v", err)
				}
				return
			}
		}

		// cache miss, get from notion and store in cache also
		slugEntries, err := n.StoreOrUpdateCacheQueryDB(r.Context(), databaseID)
		if err != nil {
			log.Error("error getting slug entries: %v", err)
		}
		// convert posts to html template!
		tmpl, err := template.ParseFiles("./templates/slugEntry.html")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		for _, entry := range slugEntries {
			err = tmpl.Execute(w, entry)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}
	}
}

func (n *notionHandler) RenderSinglePostPage() http.HandlerFunc {
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

// GetSinglePost
func (n *notionHandler) GetSinglePost() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		// get post id from request
		path := r.URL.Path
		segments := strings.Split(path, "/")
		blockID := segments[len(segments)-1]
		// get post from notion
		// it should return a list of rawblocks

		// first check if blockID exists in redis cache
		// if it does, return the cached html
		// if it doesn't, get the rawblocks from notion
		exists, err := n.redisClient.Exists(ctx, blockID).Result()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if exists == 1 {
			// get cached content from redis
			cachedJSON, err := n.redisClient.Get(ctx, blockID).Bytes()
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			var deserialized []json.RawMessage
			err = json.Unmarshal(cachedJSON, &deserialized)
			if err != nil {
				log.Error("Failed to deserialize: %v", err)
			}

			for _, rawBlock := range deserialized {
				err := n.notionClient.ParseAndWriteNotionBlock(w, rawBlock)
				if err != nil {
					w.Write([]byte("error parsing block oopsie"))
				}
			}

			// check expiry of cached content
			// if expired, update cache
			// if not expired, do nothing
			timestamp, err := n.redisClient.Get(ctx, blockID+"-timestamp").Time()
			if err != nil {
				log.Error("error getting timestamp: %v", err)
				return
			}
			// if timestamp is more than 1 hour ago, update cache
			if time.Since(timestamp) > time.Hour {
				// update cache
				err = n.StoreOrUpdateCacheBlockChildren(ctx, blockID)
				if err != nil {
					log.Error("error storing or updating cache: %v", err)
				}
				return
			}
			return
		}

		rawBlocks, err := n.notionClient.GetBlockChildren(blockID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// convert post to html template!
		for i := range rawBlocks {
			// need to modify rawBlock if its an image block
			var b models.Block
			err := json.Unmarshal(rawBlocks[i], &b)
			if err != nil {
				log.Error("error unmarshalling rawblock: %v", err)
			}
			if b.Type == "image" {
				StoreNotionImage(rawBlocks, i)
			}

			err = n.notionClient.ParseAndWriteNotionBlock(w, rawBlocks[i])
			if err != nil {
				w.Write([]byte("error parsing block oopsie"))
			}
		}
		// write to redis cache
		// Serialize the slice of json.RawMessage
		serialized, err := json.Marshal(rawBlocks)
		if err != nil {
			log.Error("error marshalling rawblocks: %v", err)
		}

		// Storing the serialized data in Redis
		err = n.redisClient.Set(ctx, blockID, serialized, 0).Err()
		if err != nil {
			log.Error("Failed to set key: %v", err)
		}
	}
}

// update or store in redis cache
func (n *notionHandler) StoreOrUpdateCacheBlockChildren(ctx context.Context, key string) error {
	// fetch notion block
	rawBlocks, err := n.notionClient.GetBlockChildren(key)
	if err != nil {
		log.Error("error getting block children: %v", err)
	}
	// update the image url
	// convert post to html template!
	for i := range rawBlocks {
		// need to modify rawBlock if its an image block
		var b models.Block
		err := json.Unmarshal(rawBlocks[i], &b)
		if err != nil {
			log.Error("error unmarshalling rawblock: %v", err)
		}
		if b.Type == "image" {
			StoreNotionImage(rawBlocks, i)
		}

	}
	// write to redis cache
	// Serialize the slice of json.RawMessage
	serialized, err := json.Marshal(rawBlocks)
	if err != nil {
		log.Error("error marshalling rawblocks: %v", err)
	}
	err = n.redisClient.Set(ctx, key, serialized, 0).Err()
	if err != nil {
		return err
	}
	// also store timestamp
	currentTime := time.Now()
	err = n.redisClient.Set(ctx, key+"-timestamp", currentTime, 0).Err()
	if err != nil {
		return err
	}
	return nil
}

// update or store in redis cache
func (n *notionHandler) StoreOrUpdateCacheQueryDB(ctx context.Context, key string) ([]notion.SlugEntry, error) {
	// fetch notion block
	rawBlocks, err := n.notionClient.GetSlugEntries(key)
	if err != nil {
		return nil, fmt.Errorf("error getting block children: %v", err)
	}
	// write to redis cache
	// Serialize the slice of json.RawMessage
	serialized, err := json.Marshal(rawBlocks)
	if err != nil {
		return nil, fmt.Errorf("error marshalling rawblocks: %v", err)
	}
	err = n.redisClient.Set(ctx, key, serialized, 0).Err()
	if err != nil {
		return nil, fmt.Errorf("error setting key: %v", err)
	}
	// also store timestamp
	currentTime := time.Now()
	err = n.redisClient.Set(ctx, key+"-timestamp", currentTime, 0).Err()
	if err != nil {
		return nil, fmt.Errorf("error setting key: %v", err)
	}
	return rawBlocks, nil
}

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
