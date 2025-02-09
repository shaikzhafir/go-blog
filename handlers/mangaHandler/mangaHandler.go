package mangaHandler

import (
	"htmx-blog/utils"
	"net/http"
)

type mangaHandler struct {
	// We can add dependencies here later, like a manga service
}

type MangaHandler interface {
	GetMangaPage() http.HandlerFunc
}

func NewHandler() MangaHandler {
	return &mangaHandler{}
}

func (h *mangaHandler) GetMangaPage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		utils.Render(w, nil, "./templates/manga.page.html")
	}
}