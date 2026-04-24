package visitors

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	log "htmx-blog/logging"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultStatsDir   = "./stats/visitors"
	defaultCookieName = "szhafir_vid"
	maxStatsDays      = 365
	defaultStatsDays  = 30
	cookieLifetime    = 365 * 24 * time.Hour
	dateLayout        = "2006-01-02"
)

var (
	currentTime = func() time.Time {
		return time.Now()
	}

	errBadDays = errors.New("days must be a positive integer")
)

// Tracker records anonymous unique visitors and exposes aggregated stats.
type Tracker interface {
	// Middleware wraps the public HTTP handler and records qualifying visits.
	Middleware(next http.Handler) http.Handler
	// StatsHandler returns an internal-only handler that reports recent daily stats.
	StatsHandler() http.HandlerFunc
}

type tracker struct {
	dir        string
	cookieName string
	secure     bool
	mu         sync.Mutex
}

type dailyStats struct {
	Date          string   `json:"date"`
	UniqueCount   int      `json:"unique_count"`
	VisitorHashes []string `json:"visitor_hashes"`
}

type dayCount struct {
	Date        string `json:"date"`
	UniqueCount int    `json:"unique_count"`
}

type statsResponse struct {
	Days                int        `json:"days"`
	TotalUniqueVisitors int        `json:"total_unique_visitors"`
	Daily               []dayCount `json:"daily"`
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

// StatsHandler serves recent visitor counts for the internal localhost mux.
func (t *tracker) StatsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		days, err := parseDays(r.URL.Query().Get("days"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		resp, err := t.readRange(days)
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

	day := currentDay()
	hash := hashVisitorID(visitorID)

	t.mu.Lock()
	defer t.mu.Unlock()

	stats, err := t.readDay(day)
	if err != nil {
		return err
	}

	if slices.Contains(stats.VisitorHashes, hash) {
		return nil
	}

	stats.VisitorHashes = append(stats.VisitorHashes, hash)
	slices.Sort(stats.VisitorHashes)
	stats.UniqueCount = len(stats.VisitorHashes)

	return t.writeDay(stats)
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

func (t *tracker) readDay(day string) (*dailyStats, error) {
	path := t.dayPath(day)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &dailyStats{
				Date:          day,
				UniqueCount:   0,
				VisitorHashes: []string{},
			}, nil
		}
		return nil, fmt.Errorf("read stats file %s: %w", path, err)
	}

	var stats dailyStats
	if err := json.Unmarshal(data, &stats); err != nil {
		return nil, fmt.Errorf("decode stats file %s: %w", path, err)
	}

	if stats.Date == "" {
		stats.Date = day
	}
	if stats.VisitorHashes == nil {
		stats.VisitorHashes = []string{}
	}
	stats.UniqueCount = len(stats.VisitorHashes)

	return &stats, nil
}

func (t *tracker) writeDay(stats *dailyStats) error {
	if err := os.MkdirAll(t.dir, 0o755); err != nil {
		return fmt.Errorf("create visitor stats dir: %w", err)
	}

	payload, err := json.Marshal(stats)
	if err != nil {
		return fmt.Errorf("marshal stats for %s: %w", stats.Date, err)
	}

	tmpFile, err := os.CreateTemp(t.dir, stats.Date+".*.tmp")
	if err != nil {
		return fmt.Errorf("create temp stats file: %w", err)
	}

	tmpName := tmpFile.Name()
	if _, err := tmpFile.Write(payload); err != nil {
		tmpFile.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("write temp stats file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("close temp stats file: %w", err)
	}
	if err := os.Rename(tmpName, t.dayPath(stats.Date)); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("rename temp stats file: %w", err)
	}

	return nil
}

func (t *tracker) readRange(days int) (*statsResponse, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	daily := make([]dayCount, 0, days)
	total := 0
	now := currentTime().In(time.Local)

	for i := 0; i < days; i++ {
		day := now.AddDate(0, 0, -i).Format(dateLayout)
		stats, err := t.readDay(day)
		if err != nil {
			return nil, err
		}

		daily = append(daily, dayCount{
			Date:        day,
			UniqueCount: stats.UniqueCount,
		})
		total += stats.UniqueCount
	}

	return &statsResponse{
		Days:                days,
		TotalUniqueVisitors: total,
		Daily:               daily,
	}, nil
}

func (t *tracker) dayPath(day string) string {
	return filepath.Join(t.dir, day+".json")
}

func parseDays(raw string) (int, error) {
	if raw == "" {
		return defaultStatsDays, nil
	}

	days, err := strconv.Atoi(raw)
	if err != nil || days < 1 {
		return 0, errBadDays
	}
	if days > maxStatsDays {
		return maxStatsDays, nil
	}

	return days, nil
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
