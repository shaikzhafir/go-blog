package main

import (
	"log"
	"net/http"

	"htmx-blog/markdownHandler"
)

func main() {
	mux := http.NewServeMux()

	// for js and css files
	staticFs := http.FileServer(http.Dir("./static"))
	indexFs := http.FileServer(http.Dir("./"))
	mux.Handle("/static/", http.StripPrefix("/static/", staticFs))

	handler := markdownHandler.NewHandler()
	mux.HandleFunc("/test", handler.ServeHTTP)
	mux.HandleFunc("/reviews", handler.GetReviewsList())
	mux.HandleFunc("/reviews/", handler.GetReviewByTitle())
	mux.HandleFunc("/blogposts", handler.GetBlogList())
	mux.Handle("/", indexFs)

	err := http.ListenAndServe(":3000", mux)
	if err != nil {
		log.Fatalf("server exited with err %v", err)
	}
}
