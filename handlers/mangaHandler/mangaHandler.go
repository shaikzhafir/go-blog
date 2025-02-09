package mangaHandler

import (
	"fmt"
	"htmx-blog/services/manga"
	"htmx-blog/utils"
	"io"
	"net/http"
	"time"

	log "htmx-blog/logging"
)

type mangaHandler struct {
	// We can add dependencies here later, like a manga service
	svc manga.MangaService
}

type MangaHandler interface {
	GetMangaPage() http.HandlerFunc
	UpdateMangaData() http.HandlerFunc
	HandleCoverProxy() http.HandlerFunc
}

func NewHandler() MangaHandler {
	return &mangaHandler{}
}

func (h *mangaHandler) GetMangaPage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// get manga data
		mangaData, err := h.svc.GetMangaData()
		if err != nil {
			log.Error("error getting manga data: %v", err)
			w.Write([]byte("error getting manga data"))
			return
		}

		utils.Render(w, map[string]interface{}{"Manga": mangaData}, "./templates/manga.page.html")
	}
}

func (h *mangaHandler) UpdateMangaData() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := h.svc.UpdateMangaData()
		if err != nil {
			w.Write([]byte("error updating manga data"))
			return
		}
		w.Write([]byte("manga data updated"))
	}
}

func (h *mangaHandler) HandleCoverProxy() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Create HTTP client with timeout
		client := &http.Client{
			Timeout: 10 * time.Second,
		}

		// Construct MangaDex URL
		mangadexURL := fmt.Sprintf("https://uploads.mangadex.org/covers/%s", r.URL.Path[len("/api/proxy/covers/"):])

		// Create request
		req, err := http.NewRequest("GET", mangadexURL, nil)
		if err != nil {
			http.Error(w, "Failed to create request", http.StatusInternalServerError)
			return
		}

		// Set headers
		req.Header.Set("User-Agent", "YourApp/1.0")
		req.Header.Set("Referer", "https://mangadex.org/")
		req.Header.Set("Accept", "image/*")

		// Make request
		resp, err := client.Do(req)
		if err != nil {
			http.Error(w, "Failed to fetch image", http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		// Check if response is successful
		if resp.StatusCode != http.StatusOK {
			http.Error(w, "Failed to fetch image", resp.StatusCode)
			return
		}

		// Copy headers
		for k, v := range resp.Header {
			w.Header()[k] = v
		}

		// Stream the response
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	}
}
