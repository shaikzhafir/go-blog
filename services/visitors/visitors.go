package visitors

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	log "htmx-blog/logging"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
)

const (
	defaultStatsDir   = "./stats/visitors"
	defaultCookieName = "szhafir_vid"
	cookieLifetime    = 365 * 24 * time.Hour
	dateLayout        = "2006-01-02"
	currentStatsFile  = "current.json"
	recordStatsFile   = "record.json"
)

var currentTime = func() time.Time {
	return time.Now()
}

// Tracker records anonymous unique visitors and exposes aggregated stats.
type Tracker interface {
	// Middleware wraps the public HTTP handler and records qualifying visits.
	Middleware(next http.Handler) http.Handler
	// StatsHandler returns an internal-only handler that reports current stats.
	StatsHandler() http.HandlerFunc
}

type tracker struct {
	dir        string
	cookieName string
	secure     bool
	mu         sync.Mutex
}

type currentStats struct {
	Date          string   `json:"date"`
	UniqueCount   int      `json:"unique_count"`
	VisitorHashes []string `json:"visitor_hashes"`
}

type recordStats struct {
	Date        string `json:"date"`
	UniqueCount int    `json:"unique_count"`
}

type statsResponse struct {
	TodayDate           string `json:"today_date"`
	TodayUniqueVisitors int    `json:"today_unique_visitors"`
	HighestVisitorDate  string `json:"highest_visitor_date"`
	HighestVisitorCount int    `json:"highest_visitor_count"`
}

// NewTracker creates a file-backed visitor tracker. If dir is empty it uses
// VISITOR_STATS_DIR when set, otherwise ./stats/visitors.
func NewTracker(dir string) Tracker {
	if dir == "" {
		dir = os.Getenv("VISITOR_STATS_DIR")
	}
	if dir == "" {
		dir = defaultStatsDir
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Error("error creating visitor stats directory %s: %v", dir, err)
	}

	return &tracker{
		dir:        dir,
		cookieName: defaultCookieName,
		secure:     os.Getenv("PROD") == "true",
	}
}

// Middleware wraps the public site handler and records qualifying top-level
// HTML visits without interrupting the main request flow on tracking failures.
func (t *tracker) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if t.shouldTrack(r) {
			if err := t.trackVisit(w, r); err != nil {
				log.Error("error tracking visitor for %s: %v", r.URL.Path, err)
			}
		}
		next.ServeHTTP(w, r)
	})
}

// StatsHandler serves the current day's count and the all-time highest day.
func (t *tracker) StatsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := t.readStats()
		if err != nil {
			log.Error("error reading visitor stats: %v", err)
			http.Error(w, "error reading visitor stats", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Error("error encoding visitor stats response: %v", err)
		}
	}
}

func (t *tracker) shouldTrack(r *http.Request) bool {
	if r.Method != http.MethodGet {
		return false
	}
	if strings.EqualFold(r.Header.Get("HX-Request"), "true") {
		return false
	}
	if r.Header.Get("DNT") == "1" || r.Header.Get("Sec-GPC") == "1" {
		return false
	}

	path := r.URL.Path
	if path == "/api" || strings.HasPrefix(path, "/api/") {
		return false
	}
	if path == "/static" || strings.HasPrefix(path, "/static/") {
		return false
	}
	if path == "/images" || strings.HasPrefix(path, "/images/") {
		return false
	}

	return !isBot(r.UserAgent())
}

func (t *tracker) trackVisit(w http.ResponseWriter, r *http.Request) error {
	visitorID, err := t.visitorID(w, r)
	if err != nil {
		return err
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	today := currentDay()
	hash := hashVisitorID(visitorID)

	stats, err := t.readCurrent()
	if err != nil {
		return err
	}
	if stats.Date != today {
		stats = &currentStats{
			Date:          today,
			UniqueCount:   0,
			VisitorHashes: []string{},
		}
	}
	if slices.Contains(stats.VisitorHashes, hash) {
		return nil
	}

	stats.VisitorHashes = append(stats.VisitorHashes, hash)
	slices.Sort(stats.VisitorHashes)
	stats.UniqueCount = len(stats.VisitorHashes)
	if err := t.writeJSON(currentStatsFile, stats); err != nil {
		return err
	}

	record, err := t.readRecord()
	if err != nil {
		return err
	}
	if stats.UniqueCount > record.UniqueCount {
		record.Date = stats.Date
		record.UniqueCount = stats.UniqueCount
		if err := t.writeJSON(recordStatsFile, record); err != nil {
			return err
		}
	}

	return nil
}

func (t *tracker) visitorID(w http.ResponseWriter, r *http.Request) (string, error) {
	cookie, err := r.Cookie(t.cookieName)
	if err == nil && cookie.Value != "" {
		return cookie.Value, nil
	}

	id, err := newVisitorID()
	if err != nil {
		return "", fmt.Errorf("generate visitor id: %w", err)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     t.cookieName,
		Value:    id,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   t.secure,
		MaxAge:   int(cookieLifetime / time.Second),
		Expires:  currentTime().Add(cookieLifetime),
	})

	return id, nil
}

func (t *tracker) readStats() (*statsResponse, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	today := currentDay()
	stats, err := t.readCurrent()
	if err != nil {
		return nil, err
	}
	if stats.Date != today {
		stats = &currentStats{
			Date:          today,
			UniqueCount:   0,
			VisitorHashes: []string{},
		}
	}

	record, err := t.readRecord()
	if err != nil {
		return nil, err
	}

	return &statsResponse{
		TodayDate:           stats.Date,
		TodayUniqueVisitors: stats.UniqueCount,
		HighestVisitorDate:  record.Date,
		HighestVisitorCount: record.UniqueCount,
	}, nil
}

func (t *tracker) readCurrent() (*currentStats, error) {
	var stats currentStats
	if err := t.readJSON(currentStatsFile, &stats); err != nil {
		return nil, err
	}
	if stats.Date == "" {
		stats.Date = currentDay()
	}
	if stats.VisitorHashes == nil {
		stats.VisitorHashes = []string{}
	}
	stats.UniqueCount = len(stats.VisitorHashes)
	return &stats, nil
}

func (t *tracker) readRecord() (*recordStats, error) {
	var record recordStats
	if err := t.readJSON(recordStatsFile, &record); err != nil {
		return nil, err
	}
	return &record, nil
}

func (t *tracker) readJSON(name string, target any) error {
	path := filepath.Join(t.dir, name)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read stats file %s: %w", path, err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("decode stats file %s: %w", path, err)
	}
	return nil
}

func (t *tracker) writeJSON(name string, value any) error {
	if err := os.MkdirAll(t.dir, 0o755); err != nil {
		return fmt.Errorf("create visitor stats dir: %w", err)
	}

	payload, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal stats file %s: %w", name, err)
	}

	tmpFile, err := os.CreateTemp(t.dir, name+".*.tmp")
	if err != nil {
		return fmt.Errorf("create temp stats file %s: %w", name, err)
	}

	tmpName := tmpFile.Name()
	if _, err := tmpFile.Write(payload); err != nil {
		tmpFile.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("write temp stats file %s: %w", name, err)
	}
	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("close temp stats file %s: %w", name, err)
	}
	if err := os.Rename(tmpName, filepath.Join(t.dir, name)); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("rename temp stats file %s: %w", name, err)
	}

	return nil
}

func currentDay() string {
	return currentTime().In(time.Local).Format(dateLayout)
}

func newVisitorID() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func hashVisitorID(id string) string {
	sum := sha256.Sum256([]byte(id))
	return hex.EncodeToString(sum[:])
}

func isBot(userAgent string) bool {
	ua := strings.ToLower(userAgent)
	for _, needle := range []string{
		"bot",
		"crawler",
		"spider",
		"slurp",
		"bingpreview",
		"facebookexternalhit",
		"discordbot",
		"googleother",
	} {
		if strings.Contains(ua, needle) {
			return true
		}
	}
	return false
}
