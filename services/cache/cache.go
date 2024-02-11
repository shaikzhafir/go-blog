package cache

import (
	"context"
	"encoding/json"
	log "htmx-blog/logging"
	"htmx-blog/services/notion"
	"os"

	"github.com/redis/go-redis/v9"
)

type Cache interface {
	Get(key string) ([]byte, error)
	Set(key string, value []byte) error
	GetSlugEntries(ctx context.Context, key string) ([]notion.SlugEntry, error)
}

func NewCache(redis *redis.Client) Cache {
	if os.Getenv("DEV") == "true" {
		return NewInMemoryCache()
	}
	return cache{redisClient: redis}
}

type inMemoryCache struct {
}

// Get implements Cache.
func (inMemoryCache) Get(key string) ([]byte, error) {
	panic("unimplemented")
}

// Get implements Cache.
func (inMemoryCache) GetSlugEntries(ctx context.Context, key string) ([]notion.SlugEntry, error) {
	file, err := os.Open("./local/sampleData/posts.json")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var dbResponse notion.QueryDBResponse
	err = json.NewDecoder(file).Decode(&dbResponse)
	if err != nil {
		return nil, err
	}

	slugEntries := []notion.SlugEntry{}

	for _, entry := range dbResponse.Results {
		// an empty RichText is not nil but an empty slice
		if entry.Properties.Slug.RichText == nil || len(entry.Properties.Slug.RichText) == 0 || len(entry.Properties.Name.Title) == 0 {
			continue
		}
		if entry.Properties.Slug.RichText[0].PlainText == "" {
			continue
		}

		slugEntry := notion.SlugEntry{
			ID:          entry.ID,
			Title:       entry.Properties.Name.Title[0].PlainText,
			CreatedTime: entry.CreatedTime,
			Slug:        entry.Properties.Slug.RichText[0].PlainText,
		}

		// append to slice
		slugEntries = append(slugEntries, slugEntry)
	}

	return slugEntries, nil

}

// Set implements Cache.
func (inMemoryCache) Set(key string, value []byte) error {
	panic("unimplemented")
}

type cache struct {
	redisClient *redis.Client
}

// GetSlugEntries implements Cache.
func (c *cache) GetSlugEntries(ctx context.Context, key string) ([]notion.SlugEntry, error) {
	exists, err := c.redisClient.Exists(ctx, key).Result()
	if err != nil {
		// key does not exist, does not matter what the error is, we have to fetch from notion API
		log.Error("error checking if key exists: %v", err)
		// cache miss, get from notion and store in cache also
	}
	if exists == 1 {
		log.Info("cache hit")
		// get cached content from redis
		cachedJSON, err := c.redisClient.Get(ctx, key).Bytes()
		if err != nil {
			log.Error("error getting cached content from redis: %v", err)
		}
		var slugEntries []notion.SlugEntry
		err = json.Unmarshal(cachedJSON, &slugEntries)
		if err != nil {
			log.Error("Failed to deserialize: %v", err)
		}

		return slugEntries, nil

		// check expiry of cached content
		// if expired, update cache
		// if not expired, do nothing

		// if timestamp is more than 1 hour ago, update cache
	}
}

// Get implements Cache.
func (cache) Get(key string) ([]byte, error) {
	return nil, nil
}

// Set implements Cache.
func (cache) Set(key string, value []byte) error {
	panic("unimplemented")
}

func NewInMemoryCache() Cache {
	return inMemoryCache{}
}
