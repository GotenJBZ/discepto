package main

import (
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"gitlab.com/ranfdev/discepto/internal/routes"
)


func main() {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(5 * time.Second))

	// Serve static files
	staticFileServer := http.FileServer(http.Dir("web/static"))
	r.Get("/static/*", func(w http.ResponseWriter, r *http.Request) {
		fs := http.StripPrefix("/static", staticFileServer)
		fs.ServeHTTP(w, r)
	})

	// Serve dynamic routes
	r.Get("/", routes.GetHome)

	r.Route("/essay", routes.EssayRouter)

	log.Println("Starting server at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
