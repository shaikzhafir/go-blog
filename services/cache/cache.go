package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	log "htmx-blog/logging"
	"htmx-blog/services/content"
	"os"
	"time"
)

// Sentinel errors for cache operations
var (
	ErrCacheMiss = errors.New("cache miss")
)

// CurrentTime is a function that returns the current time, can be mocked in tests
var CurrentTime = func() time.Time {
	return time.Now()
}

// CacheTTL is the duration after which cache entries are considered stale
const CacheTTL = time.Minute * 1

// Cache provides caching functionality for content data.
// It wraps a content.Source and adds caching capabilities.
type Cache interface {
	// GetBlockChildren returns cached block children for a given block/page ID.
	// If not cached, fetches from the content source and caches the result.
	GetBlockChildren(ctx context.Context, blockID string) ([]json.RawMessage, error)

	// GetPostEntries returns cached post entries for a collection with optional filter.
	GetPostEntries(ctx context.Context, collectionID, filter string) ([]content.PostEntry, error)

	// GetReadingEntries returns cached reading entries for a collection with filter.
	GetReadingEntries(ctx context.Context, collectionID, filter string) ([]content.ReadingEntry, error)

	// GetSource returns the underlying content source for direct access when needed.
	GetSource() content.Source
}

// CacheEntry represents a cached item with its data and timestamp
type CacheEntry struct {
	Data      json.RawMessage `json:"data"`
	Timestamp time.Time       `json:"timestamp"`
}

// JSONClient handles low-level JSON file cache operations
type JSONClient interface {
	Get(key string) (*CacheEntry, error)
	Set(key string, entry *CacheEntry) error
}

// NewCache creates a new Cache instance that wraps a content source
func NewCache(source content.Source) Cache {
	cacheDir := "./cache"
	if customDir := os.Getenv("CACHE_DIR"); customDir != "" {
		cacheDir = customDir
	}
	jsonClient := NewJSONFileClient(cacheDir)
	return &cache{
		source:     source,
		jsonClient: jsonClient,
	}
}

// cache is the main implementation of Cache interface
type cache struct {
	jsonClient JSONClient
	source     content.Source
}

// GetSource returns the underlying content source
func (c *cache) GetSource() content.Source {
	return c.source
}

// GetBlockChildren retrieves block children from cache or fetches from source
func (c *cache) GetBlockChildren(ctx context.Context, blockID string) ([]json.RawMessage, error) {
	cacheEntry, err := c.jsonClient.Get(blockID)
	if err != nil {
		if errors.Is(err, ErrCacheMiss) {
			log.Info("cache miss for block %s, fetching from source", blockID)
			return c.fetchAndCacheBlockChildren(ctx, blockID)
		}
		return nil, fmt.Errorf("error reading from cache: %w", err)
	}

	var blocks []json.RawMessage
	if err := json.Unmarshal(cacheEntry.Data, &blocks); err != nil {
		return nil, fmt.Errorf("failed to deserialize cached blocks: %w", err)
	}

	// Asynchronously refresh cache if stale
	c.refreshIfStale(blockID, func(ctx context.Context) error {
		_, err := c.fetchAndCacheBlockChildren(ctx, blockID)
		return err
	})

	return blocks, nil
}

// GetPostEntries retrieves post entries from cache or fetches from source
func (c *cache) GetPostEntries(ctx context.Context, collectionID, filter string) ([]content.PostEntry, error) {
	cacheKey := buildCacheKey(collectionID, filter)

	cacheEntry, err := c.jsonClient.Get(cacheKey)
	if err != nil {
		if errors.Is(err, ErrCacheMiss) {
			log.Info("cache miss for post entries, fetching from source")
			return c.fetchAndCachePostEntries(ctx, collectionID, filter)
		}
		return nil, fmt.Errorf("error reading from cache: %w", err)
	}

	var entries []content.PostEntry
	if err := json.Unmarshal(cacheEntry.Data, &entries); err != nil {
		return nil, fmt.Errorf("failed to deserialize cached post entries: %w", err)
	}

	// Asynchronously refresh cache if stale
	c.refreshIfStale(cacheKey, func(ctx context.Context) error {
		_, err := c.fetchAndCachePostEntries(ctx, collectionID, filter)
		return err
	})

	return entries, nil
}

// GetReadingEntries retrieves reading entries from cache or fetches from source
func (c *cache) GetReadingEntries(ctx context.Context, collectionID, filter string) ([]content.ReadingEntry, error) {
	cacheKey := buildCacheKey(collectionID, filter)

	cacheEntry, err := c.jsonClient.Get(cacheKey)
	if err != nil {
		if errors.Is(err, ErrCacheMiss) {
			log.Info("cache miss for reading entries, fetching from source")
			return c.fetchAndCacheReadingEntries(ctx, collectionID, filter)
		}
		return nil, fmt.Errorf("error reading from cache: %w", err)
	}

	var entries []content.ReadingEntry
	if err := json.Unmarshal(cacheEntry.Data, &entries); err != nil {
		return nil, fmt.Errorf("failed to deserialize cached reading entries: %w", err)
	}

	// Asynchronously refresh cache if stale
	c.refreshIfStale(cacheKey, func(ctx context.Context) error {
		_, err := c.fetchAndCacheReadingEntries(ctx, collectionID, filter)
		return err
	})

	return entries, nil
}

// fetchAndCacheBlockChildren fetches block children from source and caches them
func (c *cache) fetchAndCacheBlockChildren(ctx context.Context, blockID string) ([]json.RawMessage, error) {
	rawBlocks, err := c.source.GetBlockChildren(ctx, blockID)
	if err != nil {
		return nil, fmt.Errorf("error getting block children from source: %w", err)
	}

	// Allow the source to process blocks before caching (e.g., download images)
	for i := range rawBlocks {
		if err := c.source.ProcessBlockForStorage(rawBlocks, i); err != nil {
			log.Error("error processing block for storage: %v", err)
		}
	}

	// Cache the processed blocks
	if err := c.cacheData(blockID, rawBlocks); err != nil {
		return nil, fmt.Errorf("error caching block children: %w", err)
	}

	return rawBlocks, nil
}

// fetchAndCachePostEntries fetches post entries from source and caches them
func (c *cache) fetchAndCachePostEntries(ctx context.Context, collectionID, filter string) ([]content.PostEntry, error) {
	entries, err := c.source.GetPostEntries(ctx, collectionID, filter)
	if err != nil {
		return nil, fmt.Errorf("error getting post entries from source: %w", err)
	}

	cacheKey := buildCacheKey(collectionID, filter)
	if err := c.cacheData(cacheKey, entries); err != nil {
		return nil, fmt.Errorf("error caching post entries: %w", err)
	}

	return entries, nil
}

// fetchAndCacheReadingEntries fetches reading entries from source and caches them
func (c *cache) fetchAndCacheReadingEntries(ctx context.Context, collectionID, filter string) ([]content.ReadingEntry, error) {
	entries, err := c.source.GetReadingEntries(ctx, collectionID, filter)
	if err != nil {
		return nil, fmt.Errorf("error getting reading entries from source: %w", err)
	}

	cacheKey := buildCacheKey(collectionID, filter)
	if err := c.cacheData(cacheKey, entries); err != nil {
		return nil, fmt.Errorf("error caching reading entries: %w", err)
	}

	return entries, nil
}

// cacheData marshals and stores data in the JSON cache
func (c *cache) cacheData(key string, data any) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("error marshalling data: %w", err)
	}

	entry := &CacheEntry{
		Data:      json.RawMessage(jsonData),
		Timestamp: CurrentTime(),
	}

	if err := c.jsonClient.Set(key, entry); err != nil {
		return fmt.Errorf("error writing to cache: %w", err)
	}

	return nil
}

// refreshIfStale checks if cache is stale and refreshes it asynchronously
func (c *cache) refreshIfStale(key string, refreshFn func(ctx context.Context) error) {
	go func() {
		entry, err := c.jsonClient.Get(key)
		if err != nil {
			return
		}

		if time.Since(entry.Timestamp) > CacheTTL {
			log.Info("cache expired for %s, refreshing", key)
			if err := refreshFn(context.Background()); err != nil {
				log.Error("error refreshing cache for %s: %v", key, err)
			}
		}
	}()
}

// buildCacheKey creates a composite cache key from collection ID and filter
func buildCacheKey(collectionID, filter string) string {
	return fmt.Sprintf("%s-%s", collectionID, filter)
}

// ============================================================================
// JSON File Client Implementation
// ============================================================================

type jsonFileClient struct {
	cacheDir string
}

// NewJSONFileClient creates a new JSON file-based cache client
func NewJSONFileClient(cacheDir string) JSONClient {
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		os.MkdirAll(cacheDir, os.ModePerm)
	}
	return &jsonFileClient{cacheDir: cacheDir}
}

// Get retrieves a cache entry from a JSON file
func (jc *jsonFileClient) Get(key string) (*CacheEntry, error) {
	filePath := fmt.Sprintf("%s/%s.json", jc.cacheDir, key)
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrCacheMiss
		}
		return nil, fmt.Errorf("error reading cache file: %w", err)
	}

	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		// Invalid cache format (likely old format), treat as cache miss
		log.Info("invalid cache format for %s, treating as cache miss", key)
		return nil, ErrCacheMiss
	}

	// Also treat as cache miss if the entry has no timestamp (old format that happened to unmarshal)
	if entry.Timestamp.IsZero() {
		log.Info("cache entry missing timestamp for %s, treating as cache miss", key)
		return nil, ErrCacheMiss
	}

	return &entry, nil
}

// Set stores a cache entry to a JSON file
func (jc *jsonFileClient) Set(key string, entry *CacheEntry) error {
	filePath := fmt.Sprintf("%s/%s.json", jc.cacheDir, key)
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("error marshalling cache entry: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("error writing cache file: %w", err)
	}

	return nil
}
