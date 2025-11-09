package handlers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	log "htmx-blog/logging"
	"htmx-blog/models"
	"htmx-blog/services/cache"
	"htmx-blog/services/notion"
	"htmx-blog/utils"
)

type ReadingNowHandler struct {
	cache        cache.Cache
	notionClient notion.NotionClient
}

func NewReadingNowHandler(cache cache.Cache, notionClient notion.NotionClient) *ReadingNowHandler {
	return &ReadingNowHandler{
		cache:        cache,
		notionClient: notionClient,
	}
}

func (h *ReadingNowHandler) GetReadingNow() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Info("getting reading now")
		databaseID := h.notionClient.GetDatabaseID()
		readingNowEntries, err := h.cache.GetReadingNowEntries(r.Context(), databaseID, "speaking")
		if err != nil {
			log.Error("error getting reading now entries: %v", err)
			w.Write([]byte("error getting reading now entries"))
			return
		}

		readingNowBlocks := []models.ReadingNowBlock{}
		for _, entry := range readingNowEntries {
			var readingNowBlock models.ReadingNowBlock
			readingNowBlock.Title = entry.Title
			readingNowBlock.Author = entry.Author
			readingNowBlock.Progress = entry.Progress
			readingNowBlock.ImageURL = entry.Image
			readingNowBlock.Comments = entry.Comment
			readingNowBlocks = append(readingNowBlocks, readingNowBlock)
		}

		log.Info("reading now blocks: %v", readingNowBlocks)
		utils.Render(w, map[string]interface{}{
			"Books": readingNowBlocks,
		}, "./templates/readingNow.html")
	}
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
