package strava

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	log "htmx-blog/logging"
)

type Activity struct {
	Id             int     `json:"id"`
	StartDateLocal string  `json:"start_date_local"`
	Distance       float64 `json:"distance"`
	MovingTime     int     `json:"moving_time"`
}

func NewStravaService() StravaService {
	return &stravaService{}
}

type StravaService interface {
	GetStravaData() ([]Activity, error)
	RefreshAccessToken() error
}

type stravaService struct {
}

func (s *stravaService) GetStravaData() ([]Activity, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://www.strava.com/api/v3/athlete/activities?after=1735660800", nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	log.Info("bearer token: %s", os.Getenv("BEARER_TOKEN"))
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", os.Getenv("BEARER_TOKEN")))

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	log.Info("response body: %+v", string(bodyBytes))

	var activities []Activity
	if err := json.Unmarshal(bodyBytes, &activities); err != nil {
		return nil, fmt.Errorf("error decoding response, proabbly issue with token: %w", err)
	}
	log.Info("activities: %+v", activities)
	return activities, nil
}

type TokenResponse struct {
	TokenType    string `json:"token_type"`
	AccessToken  string `json:"access_token"`
	ExpiresAt    int64  `json:"expires_at"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

func (s *stravaService) RefreshAccessToken() error {
	client := &http.Client{}

	// Create form data
	formData := url.Values{}
	formData.Set("client_id", os.Getenv("STRAVA_CLIENT_ID"))
	formData.Set("client_secret", os.Getenv("STRAVA_CLIENT_SECRET"))
	formData.Set("refresh_token", os.Getenv("STRAVA_REFRESH_TOKEN"))
	formData.Set("grant_type", "refresh_token")

	// Make POST request
	resp, err := client.PostForm("https://www.strava.com/oauth/token", formData)
	if err != nil {
		return fmt.Errorf("error making token refresh request: %w", err)
	}
	defer resp.Body.Close()

	// Read and parse response
	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("error decoding token response: %w", err)
	}

	log.Info("token response: %+v", tokenResp)

	// Set new bearer token in environment
	if err := os.Setenv("BEARER_TOKEN", tokenResp.AccessToken); err != nil {
		return fmt.Errorf("error setting bearer token: %w", err)
	}

	log.Info("Successfully refreshed Strava access token")
	return nil
}
