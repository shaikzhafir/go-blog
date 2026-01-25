package notion

import (
	"context"
	"encoding/json"
	"io"

	log "htmx-blog/logging"
	"htmx-blog/models"
	"htmx-blog/services/content"
)

// notionSource adapts NotionClient to implement content.Source
type notionSource struct {
	client NotionClient
}

// NewSource creates a content.Source backed by Notion
func NewSource() content.Source {
	return &notionSource{
		client: NewNotionClient(),
	}
}

// NewSourceWithClient creates a content.Source with a provided NotionClient (useful for testing)
func NewSourceWithClient(client NotionClient) content.Source {
	return &notionSource{
		client: client,
	}
}

// GetBlockChildren implements content.Source
func (ns *notionSource) GetBlockChildren(ctx context.Context, blockID string) ([]json.RawMessage, error) {
	return ns.client.GetBlockChildren(blockID)
}

// GetPostEntries implements content.Source
func (ns *notionSource) GetPostEntries(ctx context.Context, collectionID, filter string) ([]content.PostEntry, error) {
	slugEntries, err := ns.client.GetSlugEntries(collectionID, filter)
	if err != nil {
		return nil, err
	}

	// Convert Notion-specific SlugEntry to generic PostEntry
	entries := make([]content.PostEntry, len(slugEntries))
	for i, se := range slugEntries {
		entries[i] = content.PostEntry{
			ID:          se.ID,
			Title:       se.Title,
			CreatedTime: se.CreatedTime,
			Slug:        se.Slug,
		}
	}
	return entries, nil
}

// GetReadingEntries implements content.Source
func (ns *notionSource) GetReadingEntries(ctx context.Context, collectionID, filter string) ([]content.ReadingEntry, error) {
	readingNowEntries, err := ns.client.GetReadingNowEntries(collectionID, filter)
	if err != nil {
		return nil, err
	}

	// Convert Notion-specific ReadingNow to generic ReadingEntry
	entries := make([]content.ReadingEntry, len(readingNowEntries))
	for i, rn := range readingNowEntries {
		entries[i] = content.ReadingEntry{
			ID:          rn.ID,
			Title:       rn.Title,
			CreatedTime: rn.CreatedTime,
			Image:       rn.Image,
			Comment:     rn.Comment,
			Progress:    rn.Progress,
			Author:      rn.Author,
		}
	}
	return entries, nil
}

// GetDefaultCollectionID implements content.Source
func (ns *notionSource) GetDefaultCollectionID() string {
	return ns.client.GetDatabaseID()
}

// ProcessBlockForStorage implements content.Source
// For Notion, this handles downloading and storing images locally
func (ns *notionSource) ProcessBlockForStorage(blocks []json.RawMessage, index int) error {
	var b models.Block
	if err := json.Unmarshal(blocks[index], &b); err != nil {
		return err
	}

	// Only process image blocks
	if b.Type == "image" {
		if err := StoreNotionImage(blocks, index); err != nil {
			log.Error("error storing notion image: %v", err)
			return err
		}
	}

	return nil
}

// GetClient returns the underlying NotionClient for direct access when needed
// (e.g., for block rendering which is Notion-specific)
func (ns *notionSource) GetClient() NotionClient {
	return ns.client
}

// notionBlockRenderer implements content.BlockRenderer for Notion blocks
type notionBlockRenderer struct {
	client NotionClient
}

// NewBlockRenderer creates a content.BlockRenderer for Notion blocks
func NewBlockRenderer() content.BlockRenderer {
	return &notionBlockRenderer{
		client: NewNotionClient(),
	}
}

// NewBlockRendererWithClient creates a content.BlockRenderer with a provided NotionClient
func NewBlockRendererWithClient(client NotionClient) content.BlockRenderer {
	return &notionBlockRenderer{
		client: client,
	}
}

// RenderBlock implements content.BlockRenderer
func (r *notionBlockRenderer) RenderBlock(writer io.Writer, rawBlock []byte) error {
	return r.client.ParseAndWriteNotionBlock(writer, rawBlock)
}
