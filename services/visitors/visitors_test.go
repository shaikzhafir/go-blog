package visitors

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestMiddlewareCreatesCookieAndReusesExistingVisitor(t *testing.T) {
	dir := t.TempDir()
	restoreTime := setCurrentTime(t, time.Date(2026, 4, 24, 10, 0, 0, 0, time.Local))
	defer restoreTime()

	tracker := NewTracker(dir).(*tracker)
	handler := tracker.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	cookies := res.Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}

	cookie := cookies[0]
	if cookie.Name != defaultCookieName {
		t.Fatalf("unexpected cookie name %q", cookie.Name)
	}
	if !cookie.HttpOnly {
		t.Fatal("expected cookie to be HttpOnly")
	}
	if cookie.Path != "/" {
		t.Fatalf("unexpected cookie path %q", cookie.Path)
	}
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("unexpected SameSite value %v", cookie.SameSite)
	}
	if cookie.Secure {
		t.Fatal("expected cookie to be insecure outside PROD")
	}

	stats := readCurrentStats(t, dir)
	if stats.UniqueCount != 1 {
		t.Fatalf("expected 1 unique visitor, got %d", stats.UniqueCount)
	}
	record := readRecordStats(t, dir)
	if record.UniqueCount != 1 || record.Date != "2026-04-24" {
		t.Fatalf("unexpected record %+v", record)
	}

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if len(rec.Result().Cookies()) != 0 {
		t.Fatal("expected no new cookie when visitor cookie already exists")
	}

	stats = readCurrentStats(t, dir)
	if stats.UniqueCount != 1 {
		t.Fatalf("expected visitor to be deduped, got %d", stats.UniqueCount)
	}
}

func TestMiddlewareResetsCurrentDayButKeepsRecord(t *testing.T) {
	dir := t.TempDir()
	restoreTime := setCurrentTime(t, time.Date(2026, 4, 24, 10, 0, 0, 0, time.Local))
	defer restoreTime()

	tracker := NewTracker(dir).(*tracker)
	handler := tracker.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	firstReq := httptest.NewRequest(http.MethodGet, "/", nil)
	firstRec := httptest.NewRecorder()
	handler.ServeHTTP(firstRec, firstReq)
	cookie := firstRec.Result().Cookies()[0]

	currentTime = func() time.Time {
		return time.Date(2026, 4, 25, 9, 0, 0, 0, time.Local)
	}

	secondReq := httptest.NewRequest(http.MethodGet, "/", nil)
	secondReq.AddCookie(cookie)
	secondRec := httptest.NewRecorder()
	handler.ServeHTTP(secondRec, secondReq)

	stats := readCurrentStats(t, dir)
	if stats.Date != "2026-04-25" {
		t.Fatalf("expected current stats to reset to 2026-04-25, got %s", stats.Date)
	}
	if stats.UniqueCount != 1 {
		t.Fatalf("expected current day count 1, got %d", stats.UniqueCount)
	}

	record := readRecordStats(t, dir)
	if record.Date != "2026-04-24" || record.UniqueCount != 1 {
		t.Fatalf("expected record to stay on first highest day, got %+v", record)
	}
}

func TestMiddlewareSkipsExcludedRequests(t *testing.T) {
	dir := t.TempDir()
	restoreTime := setCurrentTime(t, time.Date(2026, 4, 24, 10, 0, 0, 0, time.Local))
	defer restoreTime()

	tracker := NewTracker(dir).(*tracker)
	handler := tracker.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	tests := []struct {
		name    string
		target  string
		headers map[string]string
	}{
		{
			name:   "htmx request",
			target: "/notion/content/test",
			headers: map[string]string{
				"HX-Request": "true",
			},
		},
		{
			name:   "asset request",
			target: "/static/output.css",
		},
		{
			name:   "api request",
			target: "/api/proxy/covers/1/a.png",
		},
		{
			name:   "bot request",
			target: "/",
			headers: map[string]string{
				"User-Agent": "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)",
			},
		},
		{
			name:   "dnt request",
			target: "/",
			headers: map[string]string{
				"DNT": "1",
			},
		},
		{
			name:   "gpc request",
			target: "/",
			headers: map[string]string{
				"Sec-GPC": "1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.target, nil)
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if len(rec.Result().Cookies()) != 0 {
				t.Fatal("expected excluded request to skip visitor cookie")
			}
		})
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read stats dir: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected no stats files for excluded requests, got %d", len(entries))
	}
}

func TestMiddlewareWritesConcurrentUniqueVisitorsSafely(t *testing.T) {
	dir := t.TempDir()
	restoreTime := setCurrentTime(t, time.Date(2026, 4, 24, 10, 0, 0, 0, time.Local))
	defer restoreTime()

	tracker := NewTracker(dir).(*tracker)
	handler := tracker.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	var wg sync.WaitGroup
	for i := 0; i < 25; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.AddCookie(&http.Cookie{
				Name:  defaultCookieName,
				Value: "visitor-" + time.Date(2026, 4, 24, 10, 0, i, 0, time.Local).Format(time.RFC3339Nano),
			})
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
		}(i)
	}
	wg.Wait()

	stats := readCurrentStats(t, dir)
	if stats.UniqueCount != 25 {
		t.Fatalf("expected 25 unique visitors, got %d", stats.UniqueCount)
	}
	if len(stats.VisitorHashes) != 25 {
		t.Fatalf("expected 25 visitor hashes, got %d", len(stats.VisitorHashes))
	}

	record := readRecordStats(t, dir)
	if record.UniqueCount != 25 || record.Date != "2026-04-24" {
		t.Fatalf("unexpected record %+v", record)
	}
}

func TestStatsHandlerReturnsTodayAndHighestDay(t *testing.T) {
	dir := t.TempDir()
	restoreTime := setCurrentTime(t, time.Date(2026, 4, 24, 10, 0, 0, 0, time.Local))
	defer restoreTime()

	tracker := NewTracker(dir).(*tracker)
	writeJSONFile(t, dir, currentStatsFile, currentStats{
		Date:          "2026-04-24",
		UniqueCount:   2,
		VisitorHashes: []string{"a", "b"},
	})
	writeJSONFile(t, dir, recordStatsFile, recordStats{
		Date:        "2026-04-20",
		UniqueCount: 7,
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/stats/visitors", nil)
	tracker.StatsHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp statsResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.TodayDate != "2026-04-24" || resp.TodayUniqueVisitors != 2 {
		t.Fatalf("unexpected today stats %+v", resp)
	}
	if resp.HighestVisitorDate != "2026-04-20" || resp.HighestVisitorCount != 7 {
		t.Fatalf("unexpected record stats %+v", resp)
	}
}

func TestStatsHandlerResetsStaleCurrentDayInResponse(t *testing.T) {
	dir := t.TempDir()
	restoreTime := setCurrentTime(t, time.Date(2026, 4, 24, 10, 0, 0, 0, time.Local))
	defer restoreTime()

	tracker := NewTracker(dir).(*tracker)
	writeJSONFile(t, dir, currentStatsFile, currentStats{
		Date:          "2026-04-23",
		UniqueCount:   3,
		VisitorHashes: []string{"a", "b", "c"},
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/stats/visitors", nil)
	tracker.StatsHandler().ServeHTTP(rec, req)

	var resp statsResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.TodayDate != "2026-04-24" || resp.TodayUniqueVisitors != 0 {
		t.Fatalf("expected stale day to reset in response, got %+v", resp)
	}
}

func setCurrentTime(t *testing.T, now time.Time) func() {
	t.Helper()
	original := currentTime
	currentTime = func() time.Time {
		return now
	}
	return func() {
		currentTime = original
	}
}

func readCurrentStats(t *testing.T, dir string) currentStats {
	t.Helper()

	var stats currentStats
	readJSONFile(t, filepath.Join(dir, currentStatsFile), &stats)
	return stats
}

func readRecordStats(t *testing.T, dir string) recordStats {
	t.Helper()

	var stats recordStats
	readJSONFile(t, filepath.Join(dir, recordStatsFile), &stats)
	return stats
}

func readJSONFile(t *testing.T, path string, target any) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read test file %s: %v", path, err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatalf("failed to decode test file %s: %v", path, err)
	}
}

func writeJSONFile(t *testing.T, dir, name string, value any) {
	t.Helper()

	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("failed to marshal test file %s: %v", name, err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
		t.Fatalf("failed to write test file %s: %v", name, err)
	}
}
