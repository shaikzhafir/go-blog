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

const (
	localStravaDataPath = "./activities.json"
	prodStravaDataPath  = "/opt/blog/strava/activities.json"
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
	// read from file
	// if prod, read from /opt/blog/strava/activities.json
	// if dev, read from ./strava/activities.json
	if _, exists := os.LookupEnv("PROD"); exists {
		activitiesJson, err := os.ReadFile("/opt/blog/strava/activities.json")
		if err != nil {
			log.Error("error reading activities from file: %v", err)
			return nil, err
		}
		var activities []Activity
		if err := json.Unmarshal(activitiesJson, &activities); err != nil {
			log.Error("error unmarshalling activities: %v", err)
			return nil, err
		}
		return activities, nil
	}
	activitiesJson, err := os.ReadFile(localStravaDataPath)
	if err != nil {
		log.Error("error reading activities from file: %v", err)
		return nil, err
	}
	var activities []Activity
	if err := json.Unmarshal(activitiesJson, &activities); err != nil {
		log.Error("error unmarshalling activities: %v", err)
		return nil, err
	}
	return activities, nil
}

func (s *stravaService) fetchStravaData() ([]Activity, error) {
	client := &http.Client{}
	// only get activities after 1st Jan 2025
	// this is to limit the number of activities fetched
	activities, err := getAllActivities(client, 1735660800)
	if err != nil {
		return nil, fmt.Errorf("error getting all activities: %w", err)
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
	log.Info("Successfully refreshed Strava access token, updating strava data")
	s.updateStravaData()
	return nil
}

// fetch activities will only be used when access token is refreshed
// this is to limit the number of requests to strava
func (s *stravaService) updateStravaData() {
	// get strava data
	activities, err := s.fetchStravaData()
	if err != nil {
		log.Error("error getting strava data: %v", err)
	}
	// store in json lolol
	activitiesJson, err := json.Marshal(activities)
	if err != nil {
		log.Error("error marshalling activities: %v", err)
	}
	// store in raw json file in filesystem
	// if prod, store in /opt/blog/strava/activities.json
	// if dev, store in ./strava/activities.json
	if _, exists := os.LookupEnv("PROD"); exists {
		err = os.WriteFile("/opt/blog/strava/activities.json", activitiesJson, 0755)
		if err != nil {
			log.Error("error writing activities to file: %v", err)
		}
		return
	}
	err = os.WriteFile(localStravaDataPath, activitiesJson, 0755)
	if err != nil {
		log.Error("error writing activities to file: %v", err)
	}
}

func getAllActivities(client *http.Client, after int64) ([]Activity, error) {
	var allActivities []Activity
	page := 1
	perPage := 100

	for {
		// Construct URL with pagination parameters
		url := fmt.Sprintf("https://www.strava.com/api/v3/athlete/activities?after=%d&page=%d&per_page=%d",
			after, page, perPage)

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("error creating request: %w", err)
		}

		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", os.Getenv("BEARER_TOKEN")))

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("error making request: %w", err)
		}

		bodyBytes, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			return nil, fmt.Errorf("error reading response body: %w", err)
		}

		var pageActivities []Activity
		if err := json.Unmarshal(bodyBytes, &pageActivities); err != nil {
			return nil, fmt.Errorf("error decoding response: %w", err)
		}

		// If no activities returned, we've reached the end
		if len(pageActivities) == 0 {
			break
		}

		// Append this page's activities to our total
		allActivities = append(allActivities, pageActivities...)

		// If we got fewer activities than perPage, we've reached the last page
		if len(pageActivities) < perPage {
			break
		}

		// Move to next page
		page++
	}

	return allActivities, nil
}
