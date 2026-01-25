package notion

import (
	"bytes"
	"context"
	"encoding/json"
	"htmx-blog/models"
	"htmx-blog/services/content"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

// mockNotionClientForSource is a mock for testing the source adapter
type mockNotionClientForSource struct {
	blockChildren     []json.RawMessage
	slugEntries       []SlugEntry
	readingNowEntries []ReadingNow
	databaseID        string
}

func newMockNotionClientForSource() *mockNotionClientForSource {
	return &mockNotionClientForSource{
		blockChildren: []json.RawMessage{
			json.RawMessage(`{"type":"paragraph","id":"123"}`),
		},
		slugEntries: []SlugEntry{
			{ID: "1", Title: "Test Post", Slug: "test-post", CreatedTime: "2024-01-01"},
			{ID: "2", Title: "Another Post", Slug: "another-post", CreatedTime: "2024-01-02"},
		},
		readingNowEntries: []ReadingNow{
			{ID: "1", Title: "Test Book", Author: "Test Author", Progress: "50"},
		},
		databaseID: "test-db-id",
	}
}

func (m *mockNotionClientForSource) GetBlockChildren(blockID string) ([]json.RawMessage, error) {
	return m.blockChildren, nil
}

func (m *mockNotionClientForSource) GetSlugEntries(databaseID string, filter string) ([]SlugEntry, error) {
	return m.slugEntries, nil
}

func (m *mockNotionClientForSource) GetReadingNowEntries(datasourceID string, filter string) ([]ReadingNow, error) {
	return m.readingNowEntries, nil
}

func (m *mockNotionClientForSource) GetDatabaseID() string {
	return m.databaseID
}

func (m *mockNotionClientForSource) GetBlock(blockID string) (models.Block, error) {
	return models.Block{}, nil
}

func (m *mockNotionClientForSource) GetPage(pageID string) (models.Page, error) {
	return models.Page{}, nil
}

func (m *mockNotionClientForSource) GetAllPosts(databaseID string, filter string) (map[string]string, error) {
	return nil, nil
}

func (m *mockNotionClientForSource) ParseAndWriteNotionBlock(writer io.Writer, rawBlock []byte) error {
	writer.Write([]byte("<div>rendered</div>"))
	return nil
}

func Test_NotionSource_GetBlockChildren(t *testing.T) {
	mockClient := newMockNotionClientForSource()
	source := NewSourceWithClient(mockClient)

	blocks, err := source.GetBlockChildren(context.Background(), "test-block-id")

	assert.NoError(t, err)
	assert.Len(t, blocks, 1)
	assert.Contains(t, string(blocks[0]), "paragraph")
}

func Test_NotionSource_GetPostEntries(t *testing.T) {
	mockClient := newMockNotionClientForSource()
	source := NewSourceWithClient(mockClient)

	entries, err := source.GetPostEntries(context.Background(), "db-id", "blog")

	assert.NoError(t, err)
	assert.Len(t, entries, 2)

	// Verify conversion from SlugEntry to PostEntry
	assert.Equal(t, "1", entries[0].ID)
	assert.Equal(t, "Test Post", entries[0].Title)
	assert.Equal(t, "test-post", entries[0].Slug)
	assert.Equal(t, "2024-01-01", entries[0].CreatedTime)

	assert.Equal(t, "2", entries[1].ID)
	assert.Equal(t, "Another Post", entries[1].Title)
}

func Test_NotionSource_GetReadingEntries(t *testing.T) {
	mockClient := newMockNotionClientForSource()
	source := NewSourceWithClient(mockClient)

	entries, err := source.GetReadingEntries(context.Background(), "db-id", "reading")

	assert.NoError(t, err)
	assert.Len(t, entries, 1)

	// Verify conversion from ReadingNow to ReadingEntry
	assert.Equal(t, "1", entries[0].ID)
	assert.Equal(t, "Test Book", entries[0].Title)
	assert.Equal(t, "Test Author", entries[0].Author)
	assert.Equal(t, "50", entries[0].Progress)
}

func Test_NotionSource_GetDefaultCollectionID(t *testing.T) {
	mockClient := newMockNotionClientForSource()
	source := NewSourceWithClient(mockClient)

	collectionID := source.GetDefaultCollectionID()

	assert.Equal(t, "test-db-id", collectionID)
}

func Test_NotionSource_ProcessBlockForStorage_NonImageBlock(t *testing.T) {
	mockClient := newMockNotionClientForSource()
	source := NewSourceWithClient(mockClient)

	blocks := []json.RawMessage{
		json.RawMessage(`{"type":"paragraph","id":"123"}`),
	}

	// Should not error for non-image blocks
	err := source.ProcessBlockForStorage(blocks, 0)
	assert.NoError(t, err)
}

func Test_NotionSource_ImplementsContentSource(t *testing.T) {
	mockClient := newMockNotionClientForSource()
	source := NewSourceWithClient(mockClient)

	// Verify that notionSource implements content.Source
	var _ content.Source = source
}

func Test_NotionBlockRenderer_RenderBlock(t *testing.T) {
	mockClient := newMockNotionClientForSource()
	renderer := NewBlockRendererWithClient(mockClient)

	var buf bytes.Buffer
	err := renderer.RenderBlock(&buf, []byte(`{"type":"paragraph"}`))

	assert.NoError(t, err)
	assert.Equal(t, "<div>rendered</div>", buf.String())
}

func Test_NotionBlockRenderer_ImplementsBlockRenderer(t *testing.T) {
	mockClient := newMockNotionClientForSource()
	renderer := NewBlockRendererWithClient(mockClient)

	// Verify that notionBlockRenderer implements content.BlockRenderer
	var _ content.BlockRenderer = renderer
}
