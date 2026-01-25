package mocks

import (
	"context"
	"encoding/json"
	"fmt"
	"htmx-blog/services/content"
	"io"
)

// MockContentSource implements content.Source for testing
type MockContentSource struct {
	BlockChildren  []json.RawMessage
	PostEntries    []content.PostEntry
	ReadingEntries []content.ReadingEntry
	CollectionID   string
}

// NewMockContentSource creates a new mock content source with default test data
func NewMockContentSource() content.Source {
	testRawJSON := `[{"test":"test"}]`
	var blocks []json.RawMessage
	json.Unmarshal([]byte(testRawJSON), &blocks)

	return &MockContentSource{
		BlockChildren: blocks,
		PostEntries: []content.PostEntry{
			{
				Slug:        "test",
				ID:          "test",
				Title:       "test",
				CreatedTime: "test",
			},
		},
		ReadingEntries: []content.ReadingEntry{
			{
				ID:          "test",
				Title:       "test",
				CreatedTime: "test",
				Author:      "test author",
			},
		},
		CollectionID: "test-collection",
	}
}

// GetBlockChildren implements content.Source
func (m *MockContentSource) GetBlockChildren(ctx context.Context, blockID string) ([]json.RawMessage, error) {
	return m.BlockChildren, nil
}

// GetPostEntries implements content.Source
func (m *MockContentSource) GetPostEntries(ctx context.Context, collectionID, filter string) ([]content.PostEntry, error) {
	return m.PostEntries, nil
}

// GetReadingEntries implements content.Source
func (m *MockContentSource) GetReadingEntries(ctx context.Context, collectionID, filter string) ([]content.ReadingEntry, error) {
	return m.ReadingEntries, nil
}

// GetDefaultCollectionID implements content.Source
func (m *MockContentSource) GetDefaultCollectionID() string {
	return m.CollectionID
}

// ProcessBlockForStorage implements content.Source
func (m *MockContentSource) ProcessBlockForStorage(blocks []json.RawMessage, index int) error {
	// No-op for testing
	return nil
}

// MockBlockRenderer implements content.BlockRenderer for testing
type MockBlockRenderer struct{}

// NewMockBlockRenderer creates a new mock block renderer
func NewMockBlockRenderer() content.BlockRenderer {
	return &MockBlockRenderer{}
}

// RenderBlock implements content.BlockRenderer
func (m *MockBlockRenderer) RenderBlock(writer io.Writer, rawBlock []byte) error {
	_, err := writer.Write([]byte(fmt.Sprintf("<div>%s</div>", string(rawBlock))))
	return err
}
