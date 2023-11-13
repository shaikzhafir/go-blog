package main

import (
	log "htmx-blog/logging"
	"net/http"
	"os"

	"htmx-blog/markdownHandler"
)

func main() {
	mux := http.NewServeMux()
	log.Info("Starting server, dev  %s", os.Getenv("DEV"))
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

	log.Fatal("server died", http.ListenAndServe(":3000", mux))
}
