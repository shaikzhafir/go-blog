package mangaHandler

import (
	"htmx-blog/services/manga"
	"htmx-blog/utils"
	"net/http"

	log "htmx-blog/logging"
)

type mangaHandler struct {
	// We can add dependencies here later, like a manga service
	svc manga.MangaService
}

type MangaHandler interface {
	GetMangaPage() http.HandlerFunc
	UpdateMangaData() http.HandlerFunc
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
