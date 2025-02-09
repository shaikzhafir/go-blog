package handlers

import (
	"net/http"

	log "htmx-blog/logging"
	"htmx-blog/services/strava"
	"htmx-blog/utils"
)

type StravaHandler struct {
	stravaClient strava.StravaService
}

func NewStravaHandler(stravaClient strava.StravaService) *StravaHandler {
	return &StravaHandler{
		stravaClient: stravaClient,
	}
}

func (h *StravaHandler) GetStravaHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		activities, err := h.stravaClient.GetStravaData()
		if err != nil {
			log.Error("error getting strava data: %v", err)
			w.Write([]byte("error getting strava data"))
			return
		}

		utils.Render(w, map[string]interface{}{"Activities": activities}, "./templates/strava.page.html")
	}
}

func (h *StravaHandler) RefreshAccessToken() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := h.stravaClient.RefreshAccessToken()
		if err != nil {
			log.Error("error refreshing access token: %v", err)
			w.Write([]byte("error refreshing access token"))
			return
		}
		w.Write([]byte("access token refreshed"))
	}
}
