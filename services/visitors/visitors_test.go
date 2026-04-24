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

	stats := readTestDay(t, dir, "2026-04-24")
	if stats.UniqueCount != 1 {
		t.Fatalf("expected 1 unique visitor, got %d", stats.UniqueCount)
	}

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if len(rec.Result().Cookies()) != 0 {
		t.Fatal("expected no new cookie when visitor cookie already exists")
	}

	stats = readTestDay(t, dir, "2026-04-24")
	if stats.UniqueCount != 1 {
		t.Fatalf("expected visitor to be deduped, got %d", stats.UniqueCount)
	}
}

func TestMiddlewareCountsSameVisitorOnNextDay(t *testing.T) {
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

	if got := readTestDay(t, dir, "2026-04-24").UniqueCount; got != 1 {
		t.Fatalf("expected 1 visitor on day one, got %d", got)
	}
	if got := readTestDay(t, dir, "2026-04-25").UniqueCount; got != 1 {
		t.Fatalf("expected 1 visitor on day two, got %d", got)
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

	stats := readTestDay(t, dir, "2026-04-24")
	if stats.UniqueCount != 25 {
		t.Fatalf("expected 25 unique visitors, got %d", stats.UniqueCount)
	}
	if len(stats.VisitorHashes) != 25 {
		t.Fatalf("expected 25 visitor hashes, got %d", len(stats.VisitorHashes))
	}
}

func TestStatsHandlerDefaultWindowAndCap(t *testing.T) {
	dir := t.TempDir()
	restoreTime := setCurrentTime(t, time.Date(2026, 4, 24, 10, 0, 0, 0, time.Local))
	defer restoreTime()

	tracker := NewTracker(dir).(*tracker)
	writeTestDay(t, tracker, &dailyStats{
		Date:          "2026-04-24",
		VisitorHashes: []string{"a", "b"},
		UniqueCount:   2,
	})
	writeTestDay(t, tracker, &dailyStats{
		Date:          "2026-04-23",
		VisitorHashes: []string{"c"},
		UniqueCount:   1,
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
	if resp.Days != defaultStatsDays {
		t.Fatalf("expected default %d days, got %d", defaultStatsDays, resp.Days)
	}
	if len(resp.Daily) != defaultStatsDays {
		t.Fatalf("expected %d daily entries, got %d", defaultStatsDays, len(resp.Daily))
	}
	if resp.TotalUniqueVisitors != 3 {
		t.Fatalf("expected total 3, got %d", resp.TotalUniqueVisitors)
	}
	if resp.Daily[0].Date != "2026-04-24" || resp.Daily[0].UniqueCount != 2 {
		t.Fatalf("unexpected first day %+v", resp.Daily[0])
	}
	if resp.Daily[1].Date != "2026-04-23" || resp.Daily[1].UniqueCount != 1 {
		t.Fatalf("unexpected second day %+v", resp.Daily[1])
	}

	capRec := httptest.NewRecorder()
	capReq := httptest.NewRequest(http.MethodGet, "/stats/visitors?days=400", nil)
	tracker.StatsHandler().ServeHTTP(capRec, capReq)

	if capRec.Code != http.StatusOK {
		t.Fatalf("expected capped request to succeed, got %d", capRec.Code)
	}

	resp = statsResponse{}
	if err := json.NewDecoder(capRec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode capped response: %v", err)
	}
	if resp.Days != maxStatsDays {
		t.Fatalf("expected capped days %d, got %d", maxStatsDays, resp.Days)
	}
	if len(resp.Daily) != maxStatsDays {
		t.Fatalf("expected %d daily entries after cap, got %d", maxStatsDays, len(resp.Daily))
	}
}

func TestStatsHandlerRejectsInvalidDays(t *testing.T) {
	dir := t.TempDir()
	tracker := NewTracker(dir).(*tracker)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/stats/visitors?days=abc", nil)
	tracker.StatsHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
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

func readTestDay(t *testing.T, dir, day string) dailyStats {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(dir, day+".json"))
	if err != nil {
		t.Fatalf("failed to read test stats file: %v", err)
	}

	var stats dailyStats
	if err := json.Unmarshal(data, &stats); err != nil {
		t.Fatalf("failed to decode test stats file: %v", err)
	}

	return stats
}

func writeTestDay(t *testing.T, tracker *tracker, stats *dailyStats) {
	t.Helper()
	if err := tracker.writeDay(stats); err != nil {
		t.Fatalf("failed to write test day: %v", err)
	}
}
