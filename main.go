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
	mux.HandleFunc("GET /reviews", handler.GetReviewsList())
	mux.HandleFunc("GET /reviews/", handler.GetReviewByTitle())
	mux.HandleFunc("GET /blogposts", handler.GetBlogList())
	mux.HandleFunc("GET /notion/allposts/{filter}", notionHandler.GetAllPosts())
	mux.HandleFunc("GET /notion/posts/", notionHandler.RenderPostHTML())
	mux.HandleFunc("GET /notion/content/", notionHandler.GetSinglePost())
	mux.Handle("/", indexFs)
	localAddress := "localhost:3000"
	if os.Getenv("PROD") == "true" {
		localAddress = os.Getenv("PROD_ADDRESS")
	}
	log.Info("server started on %s", localAddress)
	log.Fatal("server died", http.ListenAndServe(localAddress, mux))
}
