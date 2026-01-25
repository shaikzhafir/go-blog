package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestJSONFileClient_Get(t *testing.T) {
	// Create a temporary directory for test cache files
	tempDir, err := os.MkdirTemp("", "cache_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	client := NewJSONFileClient(tempDir)

	t.Run("cache miss when file does not exist", func(t *testing.T) {
		_, err := client.Get("nonexistent")
		if err != ErrCacheMiss {
			t.Errorf("expected ErrCacheMiss, got %v", err)
		}
	})

	t.Run("cache miss for old format (raw array)", func(t *testing.T) {
		// Write old format cache file (just a raw array, no CacheEntry wrapper)
		oldFormat := []map[string]string{
			{"id": "1", "title": "Test Post"},
			{"id": "2", "title": "Another Post"},
		}
		data, _ := json.Marshal(oldFormat)
		filePath := filepath.Join(tempDir, "old-format.json")
		if err := os.WriteFile(filePath, data, 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		_, err := client.Get("old-format")
		if err != ErrCacheMiss {
			t.Errorf("expected ErrCacheMiss for old format, got %v", err)
		}
	})

	t.Run("cache miss for invalid JSON", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "invalid-json.json")
		if err := os.WriteFile(filePath, []byte("not valid json"), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		_, err := client.Get("invalid-json")
		if err != ErrCacheMiss {
			t.Errorf("expected ErrCacheMiss for invalid JSON, got %v", err)
		}
	})

	t.Run("cache miss for entry with zero timestamp", func(t *testing.T) {
		// CacheEntry with zero timestamp (could happen if old format partially matches)
		entry := CacheEntry{
			Data:      json.RawMessage(`["test"]`),
			Timestamp: time.Time{}, // zero value
		}
		data, _ := json.Marshal(entry)
		filePath := filepath.Join(tempDir, "zero-timestamp.json")
		if err := os.WriteFile(filePath, data, 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		_, err := client.Get("zero-timestamp")
		if err != ErrCacheMiss {
			t.Errorf("expected ErrCacheMiss for zero timestamp, got %v", err)
		}
	})

	t.Run("valid cache entry", func(t *testing.T) {
		now := time.Now()
		entry := CacheEntry{
			Data:      json.RawMessage(`{"key":"value"}`),
			Timestamp: now,
		}
		data, _ := json.Marshal(entry)
		filePath := filepath.Join(tempDir, "valid-entry.json")
		if err := os.WriteFile(filePath, data, 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		result, err := client.Get("valid-entry")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if string(result.Data) != `{"key":"value"}` {
			t.Errorf("unexpected data: %s", result.Data)
		}
		if result.Timestamp.Unix() != now.Unix() {
			t.Errorf("unexpected timestamp: %v", result.Timestamp)
		}
	})
}

func TestJSONFileClient_Set(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cache_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	client := NewJSONFileClient(tempDir)

	t.Run("set and get cache entry", func(t *testing.T) {
		now := time.Now()
		entry := &CacheEntry{
			Data:      json.RawMessage(`["item1","item2"]`),
			Timestamp: now,
		}

		err := client.Set("test-key", entry)
		if err != nil {
			t.Fatalf("failed to set cache entry: %v", err)
		}

		// Verify file was created
		filePath := filepath.Join(tempDir, "test-key.json")
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Error("cache file was not created")
		}

		// Verify we can read it back
		result, err := client.Get("test-key")
		if err != nil {
			t.Errorf("failed to get cache entry: %v", err)
		}
		if string(result.Data) != `["item1","item2"]` {
			t.Errorf("unexpected data: %s", result.Data)
		}
	})
}

func TestBuildCacheKey(t *testing.T) {
	tests := []struct {
		collectionID string
		filter       string
		expected     string
	}{
		{"abc123", "engineering", "abc123-engineering"},
		{"def456", "", "def456-"},
		{"", "filter", "-filter"},
	}

	for _, tt := range tests {
		result := buildCacheKey(tt.collectionID, tt.filter)
		if result != tt.expected {
			t.Errorf("buildCacheKey(%q, %q) = %q, want %q",
				tt.collectionID, tt.filter, result, tt.expected)
		}
	}
}

func TestNewJSONFileClient_CreatesDirectory(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cache_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	newCacheDir := filepath.Join(tempDir, "new_cache_dir")

	// Verify directory doesn't exist
	if _, err := os.Stat(newCacheDir); !os.IsNotExist(err) {
		t.Fatal("directory should not exist before test")
	}

	// Create client - should create directory
	_ = NewJSONFileClient(newCacheDir)

	// Verify directory was created
	if _, err := os.Stat(newCacheDir); os.IsNotExist(err) {
		t.Error("NewJSONFileClient should create cache directory")
	}
}
