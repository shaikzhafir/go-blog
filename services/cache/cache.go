package cache

import (
	"context"
	"encoding/json"
	"fmt"
	log "htmx-blog/logging"
	"htmx-blog/services/notion"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

type Cache interface {
	Get(key string) ([]byte, error)
	Set(key string, value []byte) error
	GetSlugEntries(ctx context.Context, key string) ([]notion.SlugEntry, error)
	GetPostByID(ctx context.Context, key string) ([]json.RawMessage, error)
}

func NewCache(redis *redis.Client, nc notion.NotionClient) Cache {
	if os.Getenv("DEV") == "true" {
		return NewInMemoryCache()
	}
	return &cache{redisClient: redis, notionClient: nc}
}

type inMemoryCache struct {
}

func NewInMemoryCache() Cache {
	return inMemoryCache{}
}

// GetPostByID implements Cache.
func (imc inMemoryCache) GetPostByID(ctx context.Context, key string) ([]json.RawMessage, error) {
	file, err := os.Open("./local/sampleData/notionPost.json")
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var response notion.QueryBlockChildrenResponse
	err = json.NewDecoder(file).Decode(&response)
	if err != nil {
		return nil, err
	}
	return response.Results, nil
}

// Get implements Cache.
func (imc inMemoryCache) Get(key string) ([]byte, error) {
	panic("unimplemented")
}

// Get implements Cache.
func (imc inMemoryCache) GetSlugEntries(ctx context.Context, key string) ([]notion.SlugEntry, error) {
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
func (imc inMemoryCache) Set(key string, value []byte) error {
	panic("unimplemented")
}

type cache struct {
	redisClient  *redis.Client
	notionClient notion.NotionClient
}

// GetPostByID implements Cache.
func (c *cache) GetPostByID(ctx context.Context, key string) (rawBlocks []json.RawMessage, err error) {
	cachedJSON, err := c.redisClient.Get(ctx, key).Bytes()
	if err != nil && err != redis.Nil {
		return nil, err
	}
	if err == redis.Nil {
		// doesnt exist, get from notion and store in cache
		rawBlocks, err := c.notionClient.GetBlockChildren(key)
		if err != nil {
			return nil, fmt.Errorf("error getting block children: %v", err)
		}
		// write to redis cache
		// Serialize the slice of json.RawMessage
		cachedJSON, err = json.Marshal(rawBlocks)
		if err != nil {
			log.Error("error marshalling rawblocks: %v", err)
		}
		// Storing the serialized data in Redis
		err = c.redisClient.Set(ctx, key, cachedJSON, 0).Err()
		if err != nil {
			log.Error("Failed to set key: %v", err)
		}
	}

	var deserialized []json.RawMessage
	err = json.Unmarshal(cachedJSON, &deserialized)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize: %v", err)
	}
	// go c.UpdateCache(ctx, key)
	return deserialized, nil
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

	// fetch notion block
	rawBlocks, err := c.notionClient.GetSlugEntries(key)
	if err != nil {
		return nil, fmt.Errorf("error getting block children: %v", err)
	}
	// write to redis cache
	// Serialize the slice of json.RawMessage
	serialized, err := json.Marshal(rawBlocks)
	if err != nil {
		return nil, fmt.Errorf("error marshalling rawblocks: %v", err)
	}
	err = c.redisClient.Set(ctx, key, serialized, 0).Err()
	if err != nil {
		return nil, fmt.Errorf("error setting key: %v", err)
	}
	// also store timestamp
	currentTime := time.Now()
	err = c.redisClient.Set(ctx, key+"-timestamp", currentTime, 0).Err()
	if err != nil {
		return nil, fmt.Errorf("error setting key: %v", err)
	}
	return rawBlocks, nil
}

// Get implements Cache.
func (cache) Get(key string) ([]byte, error) {
	return nil, nil
}

// Set implements Cache.
func (cache) Set(key string, value []byte) error {
	panic("unimplemented")
}

func (c *cache) UpdateCache(ctx context.Context, key string) {
	// handle timestamp to check whether to update cache
	timestamp, err := c.redisClient.Get(ctx, key+"-timestamp").Time()
	// if error is that the key doesn't exist, we should add it
	if err == redis.Nil {
		c.redisClient.Set(ctx, key+"-timestamp", time.Now(), 0)
	}
	if err != nil {
		log.Error("error getting timestamp: %v", err)
	}
	// if timestamp is more than 1 hour ago, update cache
	if time.Since(timestamp) > time.Hour {
		// TODO update cache here
		return // Add return statement to fix empty branch issue
	}
}
