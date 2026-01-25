// Package content defines the abstract interfaces and types for content sources.
// This allows the cache layer to work with any data source (Notion, Markdown, CMS, etc.)
package content

import (
	"context"
	"encoding/json"
	"io"
)

// PostEntry represents a blog post entry from any content source
type PostEntry struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	CreatedTime string `json:"created_time"`
	Slug        string `json:"slug"`
}

// ReadingEntry represents a reading/book entry from any content source
type ReadingEntry struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	CreatedTime string `json:"created_time"`
	Image       string `json:"image,omitempty"`
	Comment     string `json:"comment,omitempty"`
	Progress    string `json:"progress,omitempty"`
	Author      string `json:"author,omitempty"`
}

// Source is the interface that any content data source must implement.
// This abstraction allows swapping between different backends (Notion, Markdown files, a database, etc.)
type Source interface {
	// GetBlockChildren returns raw block data for a given block/page ID.
	// The raw blocks are returned as JSON to allow source-specific rendering.
	GetBlockChildren(ctx context.Context, blockID string) ([]json.RawMessage, error)

	// GetPostEntries returns post entries from a collection with optional filter.
	GetPostEntries(ctx context.Context, collectionID, filter string) ([]PostEntry, error)

	// GetReadingEntries returns reading entries from a collection with filter.
	GetReadingEntries(ctx context.Context, collectionID, filter string) ([]ReadingEntry, error)

	// GetDefaultCollectionID returns the default collection/database ID for this source.
	GetDefaultCollectionID() string

	// ProcessBlockForStorage allows the source to transform blocks before caching.
	// For example, downloading and storing images locally.
	ProcessBlockForStorage(blocks []json.RawMessage, index int) error
}

// BlockRenderer handles rendering of raw blocks to HTML.
// This is separate from Source because rendering may be reused across sources.
type BlockRenderer interface {
	// RenderBlock writes the HTML representation of a raw block to the writer.
	RenderBlock(writer io.Writer, rawBlock []byte) error
}
