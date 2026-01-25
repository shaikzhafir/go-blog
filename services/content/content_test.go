package content

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_PostEntry_JSONSerialization(t *testing.T) {
	entry := PostEntry{
		ID:          "123",
		Title:       "Test Post",
		CreatedTime: "2024-01-01T00:00:00Z",
		Slug:        "test-post",
	}

	// Test serialization
	data, err := json.Marshal(entry)
	assert.NoError(t, err)

	// Test deserialization
	var decoded PostEntry
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)

	assert.Equal(t, entry, decoded)
}

func Test_ReadingEntry_JSONSerialization(t *testing.T) {
	entry := ReadingEntry{
		ID:          "456",
		Title:       "Test Book",
		CreatedTime: "2024-01-01T00:00:00Z",
		Image:       "https://example.com/image.jpg",
		Comment:     "Great book!",
		Progress:    "75",
		Author:      "Test Author",
	}

	// Test serialization
	data, err := json.Marshal(entry)
	assert.NoError(t, err)

	// Test deserialization
	var decoded ReadingEntry
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)

	assert.Equal(t, entry, decoded)
}

func Test_ReadingEntry_OmitsEmptyFields(t *testing.T) {
	entry := ReadingEntry{
		ID:          "789",
		Title:       "Minimal Entry",
		CreatedTime: "2024-01-01",
	}

	data, err := json.Marshal(entry)
	assert.NoError(t, err)

	// Verify empty fields are omitted
	jsonStr := string(data)
	assert.NotContains(t, jsonStr, "image")
	assert.NotContains(t, jsonStr, "comment")
	assert.NotContains(t, jsonStr, "progress")
	assert.NotContains(t, jsonStr, "author")
}

func Test_PostEntry_EmptySlug(t *testing.T) {
	entry := PostEntry{
		ID:          "123",
		Title:       "No Slug Post",
		CreatedTime: "2024-01-01",
		Slug:        "",
	}

	data, err := json.Marshal(entry)
	assert.NoError(t, err)

	var decoded PostEntry
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)

	assert.Empty(t, decoded.Slug)
}
