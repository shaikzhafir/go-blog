package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	log "htmx-blog/logging"
	"htmx-blog/models"
	"htmx-blog/services/cache"
	"htmx-blog/utils"
)

type ReadingNowHandler struct {
	cache cache.Cache
}

func NewReadingNowHandler(cache cache.Cache) *ReadingNowHandler {
	return &ReadingNowHandler{
		cache: cache,
	}
}

func (h *ReadingNowHandler) GetReadingNowHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		blockID := os.Getenv("READING_NOW_BLOCK_ID")
		readingNowBlocks, err := h.GetReadingNow(r.Context(), blockID)
		if err != nil {
			log.Error("error getting reading now blocks: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		utils.Render(w, map[string]interface{}{
			"Books": readingNowBlocks,
		}, "./templates/readingNow.html")
	}
}

func (h *ReadingNowHandler) GetReadingNow(ctx context.Context, blockID string) ([]models.ReadingNowBlock, error) {
	rawBlocks, err := h.cache.GetReadingNowPage(ctx, blockID)
	if err != nil {
		return nil, fmt.Errorf("error getting block children: %v", err)
	}

	readingNowBlocks := []models.ReadingNowBlock{}
	var currentBook models.ReadingNowBlock
	for i := range rawBlocks {
		var b models.Block
		err := json.Unmarshal(rawBlocks[i], &b)
		if err != nil {
			log.Error("error unmarshalling rawblock: %v", err)
			continue
		}

		switch b.Type {
		case "divider":
			if i != 0 {
				readingNowBlocks = append(readingNowBlocks, currentBook)
			}
			currentBook = models.ReadingNowBlock{}
		case "heading_1":
			var heading1Block models.Heading1
			err := json.Unmarshal(rawBlocks[i], &heading1Block)
			if err != nil {
				log.Error("error unmarshalling heading1 block: %v", err)
				continue
			}
			currentBook.Title = heading1Block.Heading1.Text[0].Text.Content
		case "heading_2":
			var heading2Block models.Heading2
			err := json.Unmarshal(rawBlocks[i], &heading2Block)
			if err != nil {
				log.Error("error unmarshalling heading2 block: %v", err)
				continue
			}
			currentBook.Author = heading2Block.Heading2.Text[0].Text.Content
		case "heading_3":
			var heading3Block models.Heading3
			err := json.Unmarshal(rawBlocks[i], &heading3Block)
			if err != nil {
				log.Error("error unmarshalling heading3 block: %v", err)
				continue
			}
			currentBook.Progress = heading3Block.Heading3.Text[0].Text.Content
		case "image":
			var block models.Image
			err := json.Unmarshal(rawBlocks[i], &block)
			if err != nil {
				log.Error("error unmarshalling image block: %v", err)
				continue
			}
			currentBook.ImageURL = block.Image.File.URL
		case "paragraph":
			var block models.Paragraph
			err := json.Unmarshal(rawBlocks[i], &block)
			if err != nil {
				log.Error("error unmarshalling paragraph block: %v", err)
				continue
			}
			currentBook.Comments = block.Paragraph.RichText[0].PlainText
		}
	}
	return readingNowBlocks, nil
}

func StoreNotionImage(rawBlocks []json.RawMessage, i int) (string, error) {
	var imageBlock models.Image
	err := json.Unmarshal(rawBlocks[i], &imageBlock)
	if err != nil {
		log.Error("error unmarshalling imageblock: %v", err)
	}
	awsImageURL := imageBlock.Image.File.URL
	resp, err := http.Get(awsImageURL)
	if err != nil {
		return "", fmt.Errorf("error downloading image from s3: %v", err)
	}
	defer resp.Body.Close()

	imageBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading image bytes: %v", err)
	}
	absPath, err := filepath.Abs("./images")
	if err != nil {
		return "", fmt.Errorf("error getting absolute path: %v", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		os.Mkdir(absPath, os.ModePerm)
	}

	filePath := filepath.Join(absPath, "/", imageBlock.ID+".png")
	err = os.WriteFile(filePath, imageBytes, 0755)
	if err != nil {
		return "", fmt.Errorf("error writing image to file: %v", err)
	}

	imageBlock.Image.File.URL = "https://cloud.shaikzhafir.com/images/" + imageBlock.ID + ".png"
	if os.Getenv("DEV") == "true" {
		imageBlock.Image.File.URL = "/images/" + imageBlock.ID + ".png"
	}

	rawBlocks[i], err = json.Marshal(imageBlock)
	if err != nil {
		return "", fmt.Errorf("error marshalling imageblock: %v", err)
	}
	return imageBlock.Image.File.URL, nil
}
