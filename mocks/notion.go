package mocks

import (
	"encoding/json"
	"fmt"
	"htmx-blog/models"
	"htmx-blog/services/notion"
	"io"
)

// just trying out handrolling own mocks lol
// trying to see if just using mockery is better
func NewMockNotionClient() notion.NotionClient {
	return &mockNotionClient{}
}

type mockNotionClient struct {
}

// GetReadingNowEntries implements notion.NotionClient.
func (m *mockNotionClient) GetReadingNowEntries(datasourceID string, filter string) ([]notion.ReadingNow, error) {
	panic("unimplemented")
}

// GetAllPosts implements notion.NotionClient.
func (*mockNotionClient) GetAllPosts(databaseID string, filter string) (map[string]string, error) {
	panic("unimplemented")
}

// GetBlock implements notion.NotionClient.
func (*mockNotionClient) GetBlock(blockID string) (models.Block, error) {
	panic("unimplemented")
}

// GetBlockChildren implements notion.NotionClient.
func (*mockNotionClient) GetBlockChildren(blockID string) ([]json.RawMessage, error) {
	testRawJSON := `[{"test":"test"}]`
	var response []json.RawMessage
	err := json.Unmarshal([]byte(testRawJSON), &response)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling test raw json: %v", err)
	}
	return response, nil
}

// GetDatabaseID implements notion.NotionClient.
func (*mockNotionClient) GetDatabaseID() string {
	panic("unimplemented")
}

// GetPage implements notion.NotionClient.
func (*mockNotionClient) GetPage(pageID string) (models.Page, error) {
	panic("unimplemented")
}

// GetSlugEntries implements notion.NotionClient.
func (*mockNotionClient) GetSlugEntries(databaseID string, filter string) ([]notion.SlugEntry, error) {
	return []notion.SlugEntry{
		{
			Slug:        "test",
			ID:          "test",
			CreatedTime: "test",
			Title:       "test",
		},
	}, nil
}

// ParseAndWriteNotionBlock implements notion.NotionClient.
func (*mockNotionClient) ParseAndWriteNotionBlock(writer io.Writer, rawBlock []byte) error {
	panic("unimplemented")
}
