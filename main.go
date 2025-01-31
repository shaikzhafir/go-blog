package main

import (
	log "htmx-blog/logging"
	"htmx-blog/services/notion"
	"htmx-blog/services/strava"
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
	mux.Handle("/static/", http.StripPrefix("/static/", staticFs))
	_, exists := os.LookupEnv("PROD")
	if !exists {
		mux.Handle("/images/", http.StripPrefix("/images/", http.FileServer(http.Dir("./images"))))
	}
	handler := markdownHandler.NewHandler()
	notionClient := notion.NewNotionClient()
	stravaClient := strava.NewStravaService()

	notionHandler := notionHandler.NewHandler(notionClient, stravaClient, rdb)
	mux.HandleFunc("GET /reviews", handler.GetReviewsList())
	mux.HandleFunc("GET /reviews/", handler.GetReviewByTitle())
	mux.HandleFunc("GET /blogposts", handler.GetBlogList())
	mux.HandleFunc("GET /notion/allposts/{filter}", notionHandler.GetAllPosts())
	mux.HandleFunc("GET /notion/posts/", notionHandler.RenderPostHTML())
	mux.HandleFunc("GET /notion/content/", notionHandler.GetSinglePost())
	mux.HandleFunc("GET /readingNow", notionHandler.GetReadingNowHandler())
	mux.HandleFunc("GET /strava", notionHandler.GetStravaHandler())

	mux.Handle("/", notionHandler.Index())

	internalMux := http.NewServeMux()
	internalMux.HandleFunc("GET /cron/refreshStrava", notionHandler.RefreshAccessToken())
	go runInternalServer(internalMux)
	localAddress := "localhost:3000"
	if os.Getenv("PROD") == "true" {
		localAddress = os.Getenv("PROD_ADDRESS")
	}
	log.Info("server started on %s", localAddress)
	log.Fatal("server died", http.ListenAndServe(localAddress, mux))
}

func runInternalServer(internalMux *http.ServeMux) {
	log.Info("Starting internal API server on 127.0.0.1:8081")
	if err := http.ListenAndServe("127.0.0.1:8081", internalMux); err != nil {
		log.Error(err.Error())
	}
}
