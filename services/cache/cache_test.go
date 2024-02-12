package cache

import (
	"context"
	"htmx-blog/mocks"
	"testing"

	"github.com/go-redis/redismock/v9"
	"github.com/stretchr/testify/assert"
)

func Test_GetPostByID_Cache_Hit(t *testing.T) {
	t.Log("test")
	db, mock := redismock.NewClientMock()
	nc := mocks.NewMockNotionClient()
	cache := NewCache(db, nc)
	ctx := context.Background()

	mock.ExpectGet("test").SetVal(string([]byte(`[{"test": "test"}]`)))
	// test getting post, this will call the notion client
	rawBlocks, err := cache.GetPostByID(ctx, "test")
	assert.Nil(t, err)
	// check that the raw block is same as the one in the mock
	for _, rawBlock := range rawBlocks {
		assert.Equal(t, `{"test": "test"}`, string(rawBlock))
	}
}

func Test_GetPostByID_Cache_Miss(t *testing.T) {
	jsonRawArray := `[{"test":"test"}]`
	jsonRawItem := `{"test":"test"}`
	db, mock := redismock.NewClientMock()
	nc := mocks.NewMockNotionClient()
	cache := NewCache(db, nc)
	ctx := context.Background()
	mock.ExpectGet("test").RedisNil()
	// start with the rawjson string
	// unmarshal into the json rawmessage
	// marshal back into bytes
	mock.ExpectSet("test", []byte(jsonRawArray), 0).SetVal("OK")
	// test getting post, this will call the notion client
	rawBlocks, err := cache.GetPostByID(ctx, "test")
	assert.Nil(t, err)
	// check that the raw block is same as the one in the mock
	for _, rawBlock := range rawBlocks {
		assert.Equal(t, jsonRawItem, string(rawBlock))
	}
}
