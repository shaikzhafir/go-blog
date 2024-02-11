package main

import (
	log "htmx-blog/logging"
	"htmx-blog/services/notion"
	"net/http"
	"os"

	"htmx-blog/handlers/markdownHandler"
	"htmx-blog/handlers/notionHandler"

	"github.com/redis/go-redis/v9"
)

func main() {
	// connect to redis server
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})

	mux := http.NewServeMux()
	log.Info("Starting server, dev  %s", os.Getenv("DEV"))
	// for js and css files
	staticFs := http.FileServer(http.Dir("./static"))
	indexFs := http.FileServer(http.Dir("./"))
	mux.Handle("/static/", http.StripPrefix("/static/", staticFs))

	handler := markdownHandler.NewHandler()
	notionClient := notion.NewNotionClient()
	notionHandler := notionHandler.NewHandler(notionClient, rdb)
	mux.HandleFunc("/test", handler.ServeHTTP)
	mux.HandleFunc("/reviews", handler.GetReviewsList())
	mux.HandleFunc("/reviews/", handler.GetReviewByTitle())
	mux.HandleFunc("/blogposts", handler.GetBlogList())
	mux.HandleFunc("/notion/posts", notionHandler.GetAllPosts())
	mux.HandleFunc("/notion/posts/", notionHandler.RenderPostHTML())
	mux.HandleFunc("/notion/content/", notionHandler.GetSinglePost())
	mux.Handle("/", indexFs)

	log.Fatal("server died", http.ListenAndServe("localhost:3000", mux))
}
