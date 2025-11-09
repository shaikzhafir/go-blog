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
	GetSlugEntries(ctx context.Context, key string, filter string) ([]notion.SlugEntry, error)
	GetReadingNowEntries(ctx context.Context, key string, filter string) ([]notion.ReadingNow, error)
	GetPostByID(ctx context.Context, key string) ([]json.RawMessage, error)
	GetReadingNowPage(ctx context.Context, key string) ([]json.RawMessage, error)
}

func NewCache(redis *redis.Client, nc notion.NotionClient) Cache {
	/* if os.Getenv("DEV") == "true" {
		return NewInMemoryCache()
	} */
	// Initialize JSON file cache with default cache directory
	cacheDir := "./cache"
	if customDir := os.Getenv("CACHE_DIR"); customDir != "" {
		cacheDir = customDir
	}
	jsonClient := NewJSONFileClient(cacheDir)
	return &cache{redisClient: redis, notionClient: nc, jsonClient: jsonClient}
}

type inMemoryCache struct {
}

// GetReadingNowEntries implements Cache.
func (imc inMemoryCache) GetReadingNowEntries(ctx context.Context, key string, filter string) ([]notion.ReadingNow, error) {
	panic("unimplemented")
}

func NewInMemoryCache() Cache {
	return inMemoryCache{}
}

func (imc inMemoryCache) GetReadingNowPage(ctx context.Context, key string) ([]json.RawMessage, error) {
	file, err := os.Open("./local/sampleData/readingnow.json")
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
func (imc inMemoryCache) GetSlugEntries(ctx context.Context, key string, filter string) ([]notion.SlugEntry, error) {
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

type JSONClient interface {
	Get(key string) ([]byte, error)
	Set(key string, value []byte) error
}

type jsonFileClient struct {
	cacheDir string
}

func NewJSONFileClient(cacheDir string) JSONClient {
	// Create cache directory if it doesn't exist
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		os.MkdirAll(cacheDir, os.ModePerm)
	}
	return &jsonFileClient{cacheDir: cacheDir}
}

func (jc *jsonFileClient) Get(key string) ([]byte, error) {
	filePath := fmt.Sprintf("%s/%s.json", jc.cacheDir, key)
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("cache miss")
		}
		return nil, err
	}
	return data, nil
}

func (jc *jsonFileClient) Set(key string, value []byte) error {
	filePath := fmt.Sprintf("%s/%s.json", jc.cacheDir, key)
	err := os.WriteFile(filePath, value, 0644)
	if err != nil {
		return fmt.Errorf("error writing to file: %v", err)
	}
	return nil
}

type cache struct {
	redisClient  *redis.Client
	jsonClient   JSONClient
	notionClient notion.NotionClient
}

// GetReadingNowEntries implements Cache.
func (c *cache) GetReadingNowEntries(ctx context.Context, key string, filter string) ([]notion.ReadingNow, error) {
	readingNowKey := fmt.Sprintf("%s-%s", key, filter)

	// Try to get from JSON file cache first
	cachedJSON, err := c.jsonClient.Get(readingNowKey)
	if err != nil && err.Error() != "cache miss" {
		return nil, fmt.Errorf("error reading from cache: %v", err)
	}

	// If cache miss, fetch from Notion API
	if err != nil && err.Error() == "cache miss" {
		log.Info("json cache miss, fetching from notion")
		cachedJSON, err = c.UpdateReadingNowEntriesCache(ctx, key, filter)
		if err != nil {
			return nil, fmt.Errorf("error adding to cache: %v", err)
		}
	}

	var readingNowEntries []notion.ReadingNow
	err = json.Unmarshal(cachedJSON, &readingNowEntries)
	if err != nil {
		log.Error("Failed to deserialize: %v", err)
		return nil, err
	}

	// Asynchronously update cache if expired
	go func() {
		ctx := context.Background()
		shouldUpdate := c.ShouldUpdateJSONCache(ctx, readingNowKey)
		if shouldUpdate {
			log.Info("cache expired, updating")
			_, err := c.UpdateReadingNowEntriesCache(ctx, key, filter)
			if err != nil {
				log.Error("error updating cache: %v", err)
			}
		}
	}()

	return readingNowEntries, nil
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
func (c *cache) GetSlugEntries(ctx context.Context, key string, filter string) ([]notion.SlugEntry, error) {
	slugkey := fmt.Sprintf("%s-%s", key, filter)
	cachedJSON, err := c.redisClient.Get(ctx, slugkey).Bytes()
	if err != nil && err != redis.Nil {
		// key does not exist, does not matter what the error is, we have to fetch from notion API
		return nil, fmt.Errorf("error reading from cache: %v", err)
	}
	// if cache miss
	if err == redis.Nil {
		// fetch notion block
		log.Info("redis nil, fetching from notion")
		cachedJSON, err = c.UpdateSlugEntriesCache(ctx, key, filter)
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
			log.Info("expired")
			_, err := c.UpdateSlugEntriesCache(ctx, key, filter)
			if err != nil {
				log.Error("error updating cache: %v", err)
			}
		}
	}()
	return slugEntries, nil
}

// try getting from cache
// if cannot, then make via notion client
func (c *cache) GetReadingNowPage(ctx context.Context, key string) ([]json.RawMessage, error) {
	cachedJSON, err := c.redisClient.Get(ctx, key).Bytes()
	if err != nil && err != redis.Nil {
		// key does not exist, does not matter what the error is, we have to fetch from notion API
		return nil, fmt.Errorf("error reading from cache: %v", err)
	}
	if err == redis.Nil {
		// key does not exist, does not matter what the error is, we have to fetch from notion API
		// fetch notion block
		log.Info("redis nil, fetching from notion")
		cachedJSON, err = c.UpdateBlockChildrenCache(ctx, key)
		if err != nil {
			return nil, fmt.Errorf("error adding to cache: %v", err)
		}
	}
	var deserialized []json.RawMessage
	err = json.Unmarshal(cachedJSON, &deserialized)
	if err != nil {
		return nil, err
	}
	go func() {
		ctx := context.Background()
		shouldUpdate := c.ShouldUpdateCache(ctx, key)
		if shouldUpdate {
			log.Info("expired")
			_, err := c.UpdateBlockChildrenCache(ctx, key)
			if err != nil {
				log.Error("error updating cache: %v", err)
			}
		}
		log.Info("not expired, so not updating cache")
	}()
	return deserialized, nil
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
func (c *cache) UpdateSlugEntriesCache(ctx context.Context, key string, filter string) ([]byte, error) {
	// fetch notion block
	log.Info("fetching from notion")
	rawBlocks, err := c.notionClient.GetSlugEntries(key, filter)
	if err != nil {
		return nil, fmt.Errorf("error getting slug entries: %v", err)
	}
	// write to redis cache
	// Serialize the slice of json.RawMessage
	cachedJSON, err := json.Marshal(rawBlocks)
	if err != nil {
		return nil, fmt.Errorf("error marshalling rawblocks: %v", err)
	}
	slugkey := fmt.Sprintf("%s-%s", key, filter)
	err = c.UpdateCache(ctx, slugkey, cachedJSON)
	if err != nil {
		return nil, fmt.Errorf("error updating cache: %v", err)
	}
	return cachedJSON, nil
}

// UpdateReadingNowEntriesCache will fetch the reading now entries from the notion client and update the JSON cache
func (c *cache) UpdateReadingNowEntriesCache(ctx context.Context, key string, filter string) ([]byte, error) {
	log.Info("fetching reading now entries from notion")
	readingNowEntries, err := c.notionClient.GetReadingNowEntries(key, filter)
	if err != nil {
		return nil, fmt.Errorf("error getting reading now entries: %v", err)
	}

	// Serialize the entries
	cachedJSON, err := json.Marshal(readingNowEntries)
	if err != nil {
		return nil, fmt.Errorf("error marshalling reading now entries: %v", err)
	}

	readingNowKey := fmt.Sprintf("%s-%s", key, filter)
	err = c.UpdateJSONCache(ctx, readingNowKey, cachedJSON)
	if err != nil {
		return nil, fmt.Errorf("error updating json cache: %v", err)
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

// UpdateJSONCache updates the JSON file cache with the given key and value
func (c *cache) UpdateJSONCache(ctx context.Context, key string, value []byte) error {
	err := c.jsonClient.Set(key, value)
	if err != nil {
		return fmt.Errorf("error setting json cache key: %v", err)
	}

	// Store timestamp separately
	currentTime := CurrentTime()
	timestampData, err := json.Marshal(currentTime)
	if err != nil {
		return fmt.Errorf("error marshalling timestamp: %v", err)
	}

	err = c.jsonClient.Set(key+"-timestamp", timestampData)
	if err != nil {
		return fmt.Errorf("error setting json cache timestamp: %v", err)
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
	// if timestamp is more than 1 min ago, update cache
	if time.Since(timestamp) > time.Minute*1 {
		return true
	}
	return false
}

// ShouldUpdateJSONCache checks if the JSON cache should be updated based on timestamp
func (c *cache) ShouldUpdateJSONCache(ctx context.Context, key string) bool {
	timestampData, err := c.jsonClient.Get(key + "-timestamp")
	currentTime := CurrentTime()

	// If error (cache miss or other), create timestamp and return false
	if err != nil {
		timestampBytes, _ := json.Marshal(currentTime)
		c.jsonClient.Set(key+"-timestamp", timestampBytes)
		return false
	}

	var timestamp time.Time
	err = json.Unmarshal(timestampData, &timestamp)
	if err != nil {
		log.Error("error unmarshalling timestamp: %v", err)
		return false
	}

	// If timestamp is more than 1 min ago, update cache
	if time.Since(timestamp) > time.Minute*1 {
		return true
	}
	return false
}
