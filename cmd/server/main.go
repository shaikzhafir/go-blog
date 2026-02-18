package main

import (
	"htmx-blog/handlers"
	"htmx-blog/handlers/mangaHandler"
	"htmx-blog/handlers/markdownHandler"
	log "htmx-blog/logging"
	"htmx-blog/services/cache"
	"htmx-blog/services/content"
	"htmx-blog/services/manga"
	"htmx-blog/services/notion"
	"htmx-blog/services/strava"
	"net/http"
	"os"
)

func main() {
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
	stravaClient := strava.NewStravaService()
	mangaService := manga.NewMangaService()

	// Create content source and cache (decoupled from specific implementation)
	contentSource := notion.NewSource()
	cacheService := cache.NewCache(contentSource)
	blockRenderer := notion.NewBlockRenderer()
	pageRenderer := content.NewPageRenderer(cacheService, blockRenderer)

	blogPostHandler := handlers.NewBlogPostHandler(cacheService, pageRenderer)
	readingNowHandler := handlers.NewReadingNowHandler(cacheService)
	stravaHandler := handlers.NewStravaHandler(stravaClient)

	// Page routes (no trailing slashes; resource params in path)
	mux.HandleFunc("GET /reviews", handler.GetReviewsList())
	mux.HandleFunc("GET /reviews/{slug}", handler.GetReviewByTitle())
	mux.HandleFunc("GET /blogposts", handler.GetBlogList())
	mux.HandleFunc("GET /notion/{filter}", blogPostHandler.ListPosts())
	mux.HandleFunc("GET /notion/posts/{slug}", blogPostHandler.GetPostPage())
	mux.HandleFunc("GET /notion/content/{slug}", blogPostHandler.GetPostContent())
	mangaH := mangaHandler.NewHandler()
	mux.HandleFunc("GET /strava", stravaHandler.GetStravaHandler())
	mux.HandleFunc("GET /manga", mangaH.GetMangaPage())
	mux.HandleFunc("GET /api/proxy/covers/{id}/{filename}", mangaH.HandleCoverProxy())

	mux.Handle("/", readingNowHandler.GetReadingNow())

	internalMux := http.NewServeMux()
	internalMux.HandleFunc("GET /cron/refresh-strava", stravaHandler.RefreshAccessToken())
	internalMux.HandleFunc("GET /cron/refresh-manga", mangaH.UpdateMangaData())
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
