package handlers

import (
	"net/http"

	"htmx-blog/utils"
)

type HomeHandler struct {}

func NewHomeHandler() *HomeHandler {
	return &HomeHandler{}
}

func (h *HomeHandler) Index() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		utils.Render(w, nil)
	}
}