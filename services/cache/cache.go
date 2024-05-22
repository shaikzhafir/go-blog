package cache

import (
	"context"
	"encoding/json"
	"fmt"
	log "htmx-blog/logging"
	"htmx-blog/models"
	"htmx-blog/services/notion"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

// so time can be mocked in UT
var CurrentTime = func() time.Time {
	return time.Now()
}

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
		return nil, fmt.Errorf("error reading from cache: %v", err)
	}
	if err == redis.Nil {
		cachedJSON, err = c.UpdateBlockChildrenCache(ctx, key)
		if err != nil {
			return nil, fmt.Errorf("error adding to cache: %v", err)
		}
	}

	var deserialized []json.RawMessage
	err = json.Unmarshal(cachedJSON, &deserialized)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize: %v", err)
	}
	go func() {
		ctx := context.Background()
		shouldUpdate := c.ShouldUpdateCache(ctx, key)
		if shouldUpdate {
			_, err := c.UpdateBlockChildrenCache(ctx, key)
			if err != nil {
				log.Error("error updating cache: %v", err)
			}
		}
	}()
	return deserialized, nil
}

// GetSlugEntries implements Cache.
func (c *cache) GetSlugEntries(ctx context.Context, key string) ([]notion.SlugEntry, error) {
	cachedJSON, err := c.redisClient.Get(ctx, key).Bytes()
	if err != nil && err != redis.Nil {
		// key does not exist, does not matter what the error is, we have to fetch from notion API
		return nil, fmt.Errorf("error reading from cache: %v", err)
	}
	// if cache miss
	if err == redis.Nil {
		// fetch notion block
		cachedJSON, err = c.UpdateSlugEntriesCache(ctx, key)
		if err != nil {
			return nil, fmt.Errorf("error adding to cache: %v", err)
		}
	}

	var slugEntries []notion.SlugEntry
	err = json.Unmarshal(cachedJSON, &slugEntries)
	if err != nil {
		log.Error("Failed to deserialize: %v", err)
	}
	go func() {
		ctx := context.Background()
		shouldUpdate := c.ShouldUpdateCache(ctx, key)
		if shouldUpdate {
			_, err := c.UpdateSlugEntriesCache(ctx, key)
			if err != nil {
				log.Error("error updating cache: %v", err)
			}
		}
	}()
	return slugEntries, nil
}

func (c *cache) UpdateBlockChildrenCache(ctx context.Context, key string) ([]byte, error) {
	rawBlocks, err := c.notionClient.GetBlockChildren(key)
	if err != nil {
		return nil, fmt.Errorf("error getting block children: %v", err)
	}
	// update the image url
	for i := range rawBlocks {
		// need to modify rawBlock if its an image block
		var b models.Block
		err := json.Unmarshal(rawBlocks[i], &b)
		if err != nil {
			log.Error("error unmarshalling rawblock: %v", err)
			continue
		}
		if b.Type == "image" {
			err = notion.StoreNotionImage(rawBlocks, i)
			if err != nil {
				log.Error("error storing notion image: %v", err)
			}
		}
	}
	// after storing images, write to redis cache
	// Serialize the slice of json.RawMessage
	cachedJSON, err := json.Marshal(rawBlocks)
	if err != nil {
		return nil, fmt.Errorf("error marshalling rawblocks: %v", err)
	}
	err = c.UpdateCache(ctx, key, cachedJSON)
	if err != nil {
		return nil, fmt.Errorf("error updating cache: %v", err)
	}
	return cachedJSON, nil
}

// UpdateSlugEntriesCache will fetch the slug entries from the notion client and update the cache
func (c *cache) UpdateSlugEntriesCache(ctx context.Context, key string) ([]byte, error) {
	// fetch notion block
	rawBlocks, err := c.notionClient.GetSlugEntries(key)
	if err != nil {
		return nil, fmt.Errorf("error getting slug entries: %v", err)
	}
	// write to redis cache
	// Serialize the slice of json.RawMessage
	cachedJSON, err := json.Marshal(rawBlocks)
	if err != nil {
		return nil, fmt.Errorf("error marshalling rawblocks: %v", err)
	}
	err = c.UpdateCache(ctx, key, cachedJSON)
	if err != nil {
		return nil, fmt.Errorf("error updating cache: %v", err)
	}
	return cachedJSON, nil
}

func (c *cache) UpdateCache(ctx context.Context, key string, value []byte) error {
	err := c.redisClient.Set(ctx, key, value, 0).Err()
	if err != nil {
		return fmt.Errorf("error setting key: %v", err)
	}
	// also store timestamp
	currentTime := CurrentTime()
	err = c.redisClient.Set(ctx, key+"-timestamp", currentTime, 0).Err()
	if err != nil {
		return fmt.Errorf("error setting key: %v", err)
	}
	return nil
}

// Get implements Cache.
func (cache) Get(key string) ([]byte, error) {
	return nil, nil
}

// Set implements Cache.
func (cache) Set(key string, value []byte) error {
	panic("unimplemented")
}

func (c *cache) ShouldUpdateCache(ctx context.Context, key string) bool {
	// handle timestamp to check whether to update cache
	timestamp, err := c.redisClient.Get(ctx, key+"-timestamp").Time()
	// if error is that the key doesn't exist, we should add it
	currentTime := CurrentTime()
	if err == redis.Nil {
		c.redisClient.Set(ctx, key+"-timestamp", currentTime, 0)
		return false
	}
	if err != nil {
		log.Error("error getting timestamp: %v", err)
		return false
	}
	// if timestamp is more than 1 hour ago, update cache
	if time.Since(timestamp) > time.Hour {
		return true
	}
	return false
}
