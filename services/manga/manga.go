package manga

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	log "htmx-blog/logging"
)

type MangaService struct {
	baseURL     string
	accessToken string
}

func NewMangaService() *MangaService {
	return &MangaService{
		baseURL: "https://api.mangadex.org",
	}
}

type MangaInfoResponse struct {
	Result     string `json:"result"`
	Response   string `json:"response"`
	ReadStatus string `json:"readStatus"`
	Data       struct {
		ID         string `json:"id"`
		Type       string `json:"type"`
		Author     string `json:"author"`
		ImageURL   string `json:"imageUrl"`
		Attributes struct {
			Title struct {
				En string `json:"en"`
			} `json:"title"`
			AltTitles              []map[string]string `json:"altTitles"`
			Description            map[string]string   `json:"description"`
			IsLocked               bool                `json:"isLocked"`
			Links                  map[string]string   `json:"links"`
			OriginalLanguage       string              `json:"originalLanguage"`
			LastVolume             string              `json:"lastVolume"`
			LastChapter            string              `json:"lastChapter"`
			PublicationDemographic string              `json:"publicationDemographic"`
			Status                 string              `json:"status"`
			Year                   int                 `json:"year"`
			ContentRating          string              `json:"contentRating"`
			Tags                   []struct {
				ID         string `json:"id"`
				Type       string `json:"type"`
				Attributes struct {
					Name struct {
						En string `json:"en"`
					} `json:"name"`
					Description map[string]string `json:"description"`
					Group       string            `json:"group"`
					Version     int               `json:"version"`
				} `json:"attributes"`
				Relationships []interface{} `json:"relationships"`
			} `json:"tags"`
			State                          string   `json:"state"`
			ChapterNumbersResetOnNewVolume bool     `json:"chapterNumbersResetOnNewVolume"`
			CreatedAt                      string   `json:"createdAt"`
			UpdatedAt                      string   `json:"updatedAt"`
			Version                        int      `json:"version"`
			AvailableTranslatedLanguages   []string `json:"availableTranslatedLanguages"`
			LatestUploadedChapter          string   `json:"latestUploadedChapter"`
		} `json:"attributes"`
		Relationships []struct {
			ID         string `json:"id"`
			Type       string `json:"type"`
			Attributes struct {
				Name        string            `json:"name,omitempty"`
				ImageUrl    interface{}       `json:"imageUrl,omitempty"`
				Biography   map[string]string `json:"biography,omitempty"`
				Twitter     interface{}       `json:"twitter,omitempty"`
				Pixiv       interface{}       `json:"pixiv,omitempty"`
				MelonBook   interface{}       `json:"melonBook,omitempty"`
				FanBox      interface{}       `json:"fanBox,omitempty"`
				Booth       interface{}       `json:"booth,omitempty"`
				Namicomi    interface{}       `json:"namicomi,omitempty"`
				NicoVideo   interface{}       `json:"nicoVideo,omitempty"`
				Skeb        interface{}       `json:"skeb,omitempty"`
				Fantia      interface{}       `json:"fantia,omitempty"`
				Tumblr      interface{}       `json:"tumblr,omitempty"`
				Youtube     interface{}       `json:"youtube,omitempty"`
				Weibo       interface{}       `json:"weibo,omitempty"`
				Naver       interface{}       `json:"naver,omitempty"`
				Website     interface{}       `json:"website,omitempty"`
				CreatedAt   string            `json:"createdAt,omitempty"`
				UpdatedAt   string            `json:"updatedAt,omitempty"`
				Version     int               `json:"version,omitempty"`
				Description string            `json:"description,omitempty"`
				Volume      string            `json:"volume,omitempty"`
				FileName    string            `json:"fileName,omitempty"`
				Locale      string            `json:"locale,omitempty"`
			} `json:"attributes,omitempty"`
		} `json:"relationships"`
	} `json:"data"`
}

type MangaStatusResponse struct {
	Result   string            `json:"result"`
	Statuses map[string]string `json:"statuses"`
}

type TokenResponse struct {
	AccessToken      string `json:"access_token"`
	ExpiresIn        int    `json:"expires_in"`
	RefreshExpiresIn int    `json:"refresh_expires_in"`
	RefreshToken     string `json:"refresh_token"`
}

func (m *MangaService) UpdateMangaData() error {
	// regnerate access token
	// get user manga statuses
	// loop through manga statuses and get manga info
	// store all manga info in json
	// Get refresh token from env
	refreshToken := os.Getenv("MANGADEX_REFRESH_TOKEN")
	if refreshToken == "" {
		return fmt.Errorf("refresh token not found")
	}

	// Regenerate access token
	err := m.RegenerateAccessToken(refreshToken)
	if err != nil {
		return fmt.Errorf("failed to regenerate access token: %w", err)
	}

	// Get user manga statuses
	statusResp, err := m.GetUserMangaStatuses()
	if err != nil {
		return fmt.Errorf("failed to get user manga statuses: %w", err)
	}

	// Store manga info for each status
	var mangaInfos []MangaInfoResponse
	for mangaID := range statusResp.Statuses {
		info, err := m.GetMangaByID(mangaID)
		if err != nil {
			log.Error("Failed to get manga info for %s: %v", mangaID, err)
			continue
		}
		info.ReadStatus = statusResp.Statuses[mangaID]
		mangaInfos = append(mangaInfos, *info)
	}

	// Convert to JSON
	mangaJson, err := json.Marshal(mangaInfos)
	if err != nil {
		return fmt.Errorf("failed to marshal manga info: %w", err)
	}

	// Save to file based on environment
	filePath := "./manga.json"
	if _, exists := os.LookupEnv("PROD"); exists {
		filePath = "/opt/blog/manga/manga.json"
	}

	if err := os.WriteFile(filePath, mangaJson, 0644); err != nil {
		return fmt.Errorf("failed to write manga info to file: %w", err)
	}
	return nil
}

func (h *MangaService) GetMangaData() ([]MangaInfoResponse, error) {
	// read from file
	// if prod, read from /opt/blog/manga/manga.json
	// if dev, read from ./manga.json
	if _, exists := os.LookupEnv("PROD"); exists {
		mangaJson, err := os.ReadFile("/opt/blog/manga/manga.json")
		if err != nil {
			log.Error("error reading manga from file: %v", err)
			return nil, err
		}
		var mangas []MangaInfoResponse
		if err := json.Unmarshal(mangaJson, &mangas); err != nil {
			log.Error("error unmarshalling manga: %v", err)
			return nil, err
		}
		return mangas, nil
	}
	mangaJson, err := os.ReadFile("./manga.json")
	if err != nil {
		log.Error("error reading manga from file: %v", err)
		return nil, err
	}
	var mangas []MangaInfoResponse
	if err := json.Unmarshal(mangaJson, &mangas); err != nil {
		log.Error("error unmarshalling manga: %v", err)
		return nil, err
	}
	return mangas, nil
}

func (m *MangaService) RegenerateAccessToken(refreshToken string) error {
	authUrl := "https://auth.mangadex.org/realms/mangadex/protocol/openid-connect/token"

	values := url.Values{}
	values.Add("grant_type", "refresh_token")
	values.Add("refresh_token", refreshToken)
	values.Add("client_id", os.Getenv("MANGADEX_CLIENT_ID"))
	values.Add("client_secret", os.Getenv("MANGADEX_CLIENT_SECRET"))

	data := values.Encode()

	req, err := http.NewRequest("POST", authUrl, strings.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received non-200 response: %d", resp.StatusCode)
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Set new access token
	m.accessToken = tokenResp.AccessToken

	return nil
}

func (m *MangaService) GetUserMangaStatuses() (*MangaStatusResponse, error) {
	url := fmt.Sprintf("%s/manga/status", m.baseURL)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", m.accessToken))
	req.Header.Add("accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-200 response: %d", resp.StatusCode)
	}

	var statusResp MangaStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&statusResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &statusResp, nil
}

func (m *MangaService) GetMangaByID(id string) (*MangaInfoResponse, error) {
	// Add query parameters for including cover art and author
	url := fmt.Sprintf("%s/manga/%s?includes[]=cover_art&includes[]=author", m.baseURL, id)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-200 response: %d", resp.StatusCode)
	}

	var manga MangaInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&manga); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	// loop through relationships to get author and cover art and set them in the main data struct
	for _, rel := range manga.Data.Relationships {
		if rel.Type == "author" {
			manga.Data.Author = rel.Attributes.Name
		}
		if rel.Type == "cover_art" {
			manga.Data.ImageURL = fmt.Sprintf("https://uploads.mangadex.org/covers/%s/%s", manga.Data.ID, rel.Attributes.FileName)
		}
	}
	return &manga, nil
}
