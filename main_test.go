package main_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"htmx-blog/handlers/markdownHandler"
)

func TestGetReviewsList(t *testing.T) {
	handler := markdownHandler.NewHandler()
	req, err := http.NewRequest("GET", "/reviews", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	expected := "List of reviews"
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}
}

func TestGetReviewByTitle(t *testing.T) {
	handler := markdownHandler.NewHandler()
	req, err := http.NewRequest("GET", "/reviews/test-review", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	expected := "Review: test-review"
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}
}

func TestGetBlogList(t *testing.T) {
	handler := markdownHandler.NewHandler()
	req, err := http.NewRequest("GET", "/blogposts", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	expected := "List of blog posts"
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}
}

func TestMain(m *testing.M) {
	// set up any test dependencies here
	os.Setenv("DEV", "true")

	// run tests
	code := m.Run()

	// clean up any test dependencies here
	os.Unsetenv("DEV")

	// exit with the test code
	os.Exit(code)
}
