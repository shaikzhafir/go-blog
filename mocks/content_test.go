package mocks

import (
	"bytes"
	"context"
	"htmx-blog/services/content"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_MockContentSource_GetBlockChildren(t *testing.T) {
	source := NewMockContentSource()

	blocks, err := source.GetBlockChildren(context.Background(), "any-id")

	assert.NoError(t, err)
	assert.Len(t, blocks, 1)
}

func Test_MockContentSource_GetPostEntries(t *testing.T) {
	source := NewMockContentSource()

	entries, err := source.GetPostEntries(context.Background(), "db-id", "filter")

	assert.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "test", entries[0].ID)
	assert.Equal(t, "test", entries[0].Title)
	assert.Equal(t, "test", entries[0].Slug)
}

func Test_MockContentSource_GetReadingEntries(t *testing.T) {
	source := NewMockContentSource()

	entries, err := source.GetReadingEntries(context.Background(), "db-id", "filter")

	assert.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "test", entries[0].ID)
	assert.Equal(t, "test author", entries[0].Author)
}

func Test_MockContentSource_GetDefaultCollectionID(t *testing.T) {
	source := NewMockContentSource()

	id := source.GetDefaultCollectionID()

	assert.Equal(t, "test-collection", id)
}

func Test_MockContentSource_ProcessBlockForStorage(t *testing.T) {
	source := NewMockContentSource().(*MockContentSource)

	// Should be a no-op
	err := source.ProcessBlockForStorage(nil, 0)
	assert.NoError(t, err)
}

func Test_MockContentSource_ImplementsSource(t *testing.T) {
	source := NewMockContentSource()

	// Verify interface implementation
	var _ content.Source = source
}

func Test_MockBlockRenderer_RenderBlock(t *testing.T) {
	renderer := NewMockBlockRenderer()

	var buf bytes.Buffer
	err := renderer.RenderBlock(&buf, []byte(`{"test":"data"}`))

	assert.NoError(t, err)
	assert.Equal(t, `<div>{"test":"data"}</div>`, buf.String())
}

func Test_MockBlockRenderer_ImplementsBlockRenderer(t *testing.T) {
	renderer := NewMockBlockRenderer()

	// Verify interface implementation
	var _ content.BlockRenderer = renderer
}

func Test_MockContentSource_CustomData(t *testing.T) {
	// Test that you can customize mock data
	source := &MockContentSource{
		PostEntries: []content.PostEntry{
			{ID: "custom-1", Title: "Custom Post"},
			{ID: "custom-2", Title: "Another Custom Post"},
		},
		CollectionID: "custom-collection",
	}

	entries, err := source.GetPostEntries(context.Background(), "any", "any")

	assert.NoError(t, err)
	assert.Len(t, entries, 2)
	assert.Equal(t, "custom-1", entries[0].ID)
	assert.Equal(t, "custom-collection", source.GetDefaultCollectionID())
}
