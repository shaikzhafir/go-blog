package main

import (
	log "htmx-blog/logging"
	"net/http"
	"os"

	"htmx-blog/markdownHandler"
)

func main() {
	mux := http.NewServeMux()
	log.Info("Starting server %s", os.Getenv("DEV"))
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

	//log.Fatal(http.ListenAndServe("127.0.0.1:8081", mux))

	isDev := os.Getenv("DEV")
	if isDev == "true" {
		log.Fatal("server died", http.ListenAndServe(":8080", mux))
	} else {
		certFile := os.Getenv("CERT_FILE")
		if certFile == "" {
			log.Fatal("CERT_FILE env variable not set")
		}
		keyFile := os.Getenv("KEY_FILE")
		if keyFile == "" {
			log.Fatal("KEY_FILE env variable not set")
		}
		log.Fatal("server died", http.ListenAndServeTLS(":443", certFile, keyFile, mux))
	}
}
