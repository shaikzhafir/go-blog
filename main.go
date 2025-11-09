package main

import (
	"htmx-blog/handlers"
	"htmx-blog/handlers/mangaHandler"
	"htmx-blog/handlers/markdownHandler"
	log "htmx-blog/logging"
	"htmx-blog/services/cache"
	"htmx-blog/services/manga"
	"htmx-blog/services/notion"
	"htmx-blog/services/strava"
	"net/http"
	"os"

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
	mangaService := manga.NewMangaService()
	cacheService := cache.NewCache(rdb, notionClient)

	blogPostHandler := handlers.NewBlogPostHandler(notionClient, cacheService)
	readingNowHandler := handlers.NewReadingNowHandler(cacheService, notionClient)
	stravaHandler := handlers.NewStravaHandler(stravaClient)

	mux.HandleFunc("GET /reviews", handler.GetReviewsList())
	mux.HandleFunc("GET /reviews/", handler.GetReviewByTitle())
	mux.HandleFunc("GET /blogposts", handler.GetBlogList())
	mux.HandleFunc("GET /notion/allposts/{filter}", blogPostHandler.GetAllPosts())
	mux.HandleFunc("GET /notion/posts/", blogPostHandler.RenderPostHTML())
	mux.HandleFunc("GET /notion/content/", blogPostHandler.GetSinglePost())
	mux.HandleFunc("GET /readingNow", readingNowHandler.GetReadingNow())
	mux.HandleFunc("GET /strava", stravaHandler.GetStravaHandler())

	// Initialize manga handler
	mangaHandler := mangaHandler.NewHandler()
	mux.HandleFunc("GET /manga", mangaHandler.GetMangaPage())
	mux.HandleFunc("GET /api/proxy/covers/", mangaHandler.HandleCoverProxy())

	mux.Handle("/", readingNowHandler.GetReadingNow())

	internalMux := http.NewServeMux()
	internalMux.HandleFunc("GET /cron/refreshStrava", stravaHandler.RefreshAccessToken())
	internalMux.HandleFunc("GET /cron/refreshManga", mangaHandler.UpdateMangaData())
	// refresh strava token on init always in prod
	err := mangaService.UpdateMangaData()
	if err != nil {
		log.Error("error updating manga data: %v", err)
	}
	if os.Getenv("PROD") == "true" {
		err := stravaClient.RefreshAccessToken()
		if err != nil {
			log.Error("error refreshing strava token: %v", err)
		}
	}
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
