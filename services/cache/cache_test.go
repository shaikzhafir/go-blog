package cache

import (
	"context"
	"encoding/json"
	"htmx-blog/mocks"
	"htmx-blog/services/content"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_GetBlockChildren_Cache_Hit(t *testing.T) {
	// Create temp cache directory for testing
	tempDir := "./test_cache"
	os.Setenv("CACHE_DIR", tempDir)
	defer os.RemoveAll(tempDir)
	defer os.Unsetenv("CACHE_DIR")

	source := mocks.NewMockContentSource()
	cache := NewCache(source)
	ctx := context.Background()

	// Pre-populate cache with test data using CacheEntry
	testData := []byte(`[{"test": "test"}]`)
	entry := &CacheEntry{
		Data:      json.RawMessage(testData),
		Timestamp: time.Now(),
	}
	jsonClient := NewJSONFileClient(tempDir)
	jsonClient.Set("test", entry)

	// test getting post, this should hit the cache
	rawBlocks, err := cache.GetBlockChildren(ctx, "test")
	assert.Nil(t, err)
	// check that the raw block is same as the one in the cache (JSON compact format)
	for _, rawBlock := range rawBlocks {
		assert.Equal(t, `{"test":"test"}`, string(rawBlock))
	}
}

func Test_GetBlockChildren_Cache_Miss(t *testing.T) {
	jsonRawItem := `{"test":"test"}`

	// Create temp cache directory for testing
	tempDir := "./test_cache_miss"
	os.Setenv("CACHE_DIR", tempDir)
	defer os.RemoveAll(tempDir)
	defer os.Unsetenv("CACHE_DIR")

	source := mocks.NewMockContentSource()
	cache := NewCache(source)

	CurrentTime = func() time.Time {
		return time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	}

	ctx := context.Background()

	// test getting post, this will call the source (cache miss)
	rawBlocks, err := cache.GetBlockChildren(ctx, "test")
	assert.Nil(t, err)
	// check that the raw block is same as the one from source
	for _, rawBlock := range rawBlocks {
		assert.Equal(t, jsonRawItem, string(rawBlock))
	}
}

func Test_GetPostEntries_Cache_Hit(t *testing.T) {
	// Create temp cache directory for testing
	tempDir := "./test_cache_slug"
	os.Setenv("CACHE_DIR", tempDir)
	defer os.RemoveAll(tempDir)
	defer os.Unsetenv("CACHE_DIR")

	source := mocks.NewMockContentSource()
	cache := NewCache(source)
	ctx := context.Background()

	mockPostEntry := []content.PostEntry{
		{
			Slug:        "test",
			ID:          "test",
			Title:       "test",
			CreatedTime: "test",
		},
	}
	jsonPostEntry, _ := json.Marshal(mockPostEntry)

	// Pre-populate cache with test data using CacheEntry
	entry := &CacheEntry{
		Data:      json.RawMessage(jsonPostEntry),
		Timestamp: time.Now(),
	}
	jsonClient := NewJSONFileClient(tempDir)
	jsonClient.Set("test-", entry)

	// test getting post, this should hit the cache
	postEntries, err := cache.GetPostEntries(ctx, "test", "")
	assert.Nil(t, err)
	// check that the entry is same as the one in the cache
	assert.Equal(t, mockPostEntry, postEntries)
}

func Test_GetPostEntries_Cache_Miss(t *testing.T) {
	// Create temp cache directory for testing
	tempDir := "./test_cache_slug_miss"
	os.Setenv("CACHE_DIR", tempDir)
	defer os.RemoveAll(tempDir)
	defer os.Unsetenv("CACHE_DIR")

	source := mocks.NewMockContentSource()
	cache := NewCache(source)

	CurrentTime = func() time.Time {
		return time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	}

	ctx := context.Background()
	mockPostEntry := []content.PostEntry{
		{
			Slug:        "test",
			ID:          "test",
			Title:       "test",
			CreatedTime: "test",
		},
	}

	// test getting post, this will call the source (cache miss)
	postEntries, err := cache.GetPostEntries(ctx, "test", "")
	assert.Nil(t, err)
	// check that the entry is same as the one from source
	assert.Equal(t, mockPostEntry, postEntries)
}

func Test_GetReadingEntries_Cache_Miss(t *testing.T) {
	// Create temp cache directory for testing
	tempDir := "./test_cache_reading_miss"
	os.Setenv("CACHE_DIR", tempDir)
	defer os.RemoveAll(tempDir)
	defer os.Unsetenv("CACHE_DIR")

	source := mocks.NewMockContentSource()
	cache := NewCache(source)

	CurrentTime = func() time.Time {
		return time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	}

	ctx := context.Background()
	mockReadingEntry := []content.ReadingEntry{
		{
			ID:          "test",
			Title:       "test",
			CreatedTime: "test",
			Author:      "test author",
		},
	}

	// test getting reading entries, this will call the source (cache miss)
	readingEntries, err := cache.GetReadingEntries(ctx, "test", "")
	assert.Nil(t, err)
	assert.Equal(t, mockReadingEntry, readingEntries)
}

func Test_ErrCacheMiss(t *testing.T) {
	// Create temp cache directory for testing
	tempDir := "./test_cache_miss_error"
	defer os.RemoveAll(tempDir)

	jsonClient := NewJSONFileClient(tempDir)

	// Try to get a key that doesn't exist
	_, err := jsonClient.Get("nonexistent")
	assert.ErrorIs(t, err, ErrCacheMiss)
}

func Test_CacheExpiry(t *testing.T) {
	// Create temp cache directory for testing
	tempDir := "./test_cache_expiry"
	defer os.RemoveAll(tempDir)

	jsonClient := NewJSONFileClient(tempDir)

	// Create an old cache entry
	oldTime := time.Now().Add(-2 * CacheTTL)
	entry := &CacheEntry{
		Data:      json.RawMessage(`{"test": "data"}`),
		Timestamp: oldTime,
	}
	jsonClient.Set("test-key", entry)

	// Read it back and verify timestamp
	retrieved, err := jsonClient.Get("test-key")
	assert.Nil(t, err)
	assert.True(t, time.Since(retrieved.Timestamp) > CacheTTL)
}

func Test_GetSource(t *testing.T) {
	tempDir := "./test_cache_source"
	os.Setenv("CACHE_DIR", tempDir)
	defer os.RemoveAll(tempDir)
	defer os.Unsetenv("CACHE_DIR")

	source := mocks.NewMockContentSource()
	cache := NewCache(source)

	// Verify GetSource returns the underlying source
	assert.Equal(t, source, cache.GetSource())
	assert.Equal(t, "test-collection", cache.GetSource().GetDefaultCollectionID())
}
