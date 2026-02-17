package content

import (
	"context"
	"io"
)

// blockPageRenderer implements PageRenderer by fetching blocks from a
// BlockFetcher and rendering each with a BlockRenderer. Used for Notion
// today; other backends can provide their own BlockFetcher + BlockRenderer
// or implement PageRenderer directly.
type blockPageRenderer struct {
	fetcher  BlockFetcher
	renderer BlockRenderer
}

// NewPageRenderer returns a PageRenderer that fetches blocks from fetcher
// and renders them with renderer. Handlers use this so the data source
// (Notion, Markdown, CMS) can be swapped without changing handler code.
func NewPageRenderer(fetcher BlockFetcher, renderer BlockRenderer) PageRenderer {
	return &blockPageRenderer{
		fetcher:  fetcher,
		renderer: renderer,
	}
}

// RenderPage fetches block children for the page and writes their HTML to w.
func (p *blockPageRenderer) RenderPage(ctx context.Context, w io.Writer, pageIDOrSlug string, opts RenderOptions) error {
	blocks, err := p.fetcher.GetBlockChildren(ctx, pageIDOrSlug)
	if err != nil {
		return err
	}
	for _, raw := range blocks {
		if err := p.renderer.RenderBlock(w, raw, opts.PostType); err != nil {
			return err
		}
	}
	return nil
}
