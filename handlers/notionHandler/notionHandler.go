package notionHandler

import (
	"htmx-blog/services/cache"
	"htmx-blog/services/notion"
)

type notionHandler struct {
	cache        cache.Cache
	notionClient notion.NotionClient
}

func NewHandler(notionClient notion.NotionClient, cache cache.Cache) *notionHandler {
	return &notionHandler{
		notionClient: notionClient,
		cache: cache,
	}
}