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
	"htmx-blog/services/notion/imageenc"
	"htmx-blog/services/strava"
	"htmx-blog/services/visitors"
	"net/http"
	"os"
)

// immutableImageCache wraps a handler and sets a long-lived, immutable
// Cache-Control for responses under /images/. Safe because IDs are
// content-addressed (Notion block ID) and never reused.
func immutableImageCache(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		h.ServeHTTP(w, r)
	})
}

func main() {
	mux := http.NewServeMux()
	log.Info("Starting server, dev  %s", os.Getenv("DEV"))
	// for js and css files
	staticFs := http.FileServer(http.Dir("./static"))
	mux.Handle("/static/", http.StripPrefix("/static/", staticFs))
	if !imageenc.Available() {
		log.Error("cwebp binary not on PATH; new Notion images will fall back to their original format until it's installed (apt install webp)")
	}

	_, exists := os.LookupEnv("PROD")
	if !exists {
		imagesFs := http.StripPrefix("/images/", http.FileServer(http.Dir("./images")))
		mux.Handle("/images/", immutableImageCache(imagesFs))
	}
	handler := markdownHandler.NewHandler()
	stravaClient := strava.NewStravaService()
	mangaService := manga.NewMangaService()

	// Create content source and cache (decoupled from specific implementation)
	contentSource := notion.NewSource()
	cacheService := cache.NewCache(contentSource)
	blockRenderer := notion.NewBlockRenderer()
	pageRenderer := content.NewPageRenderer(cacheService, blockRenderer)
	visitorTracker := visitors.NewTracker("")

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
	internalMux.HandleFunc("POST /cron/backfill-images", handlers.ImageBackfillHandler())
	internalMux.HandleFunc("GET /stats/visitors", visitorTracker.StatsHandler())
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
	if err := http.ListenAndServe(localAddress, visitorTracker.Middleware(mux)); err != nil {
		log.Fatal("server died: %v", err)
	}
}

func runInternalServer(internalMux *http.ServeMux) {
	log.Info("Starting internal API server on 127.0.0.1:8081")
	if err := http.ListenAndServe("127.0.0.1:8081", internalMux); err != nil {
		log.Error(err.Error())
	}
}
