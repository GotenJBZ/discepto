package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"gitlab.com/ranfdev/discepto/internal/db"
	"gitlab.com/ranfdev/discepto/internal/routes"
)


func main() {
	err := db.Connect()
	if err != nil {
		log.Fatal(err)
	}
	err = db.Migrate()
	if err != nil {
		log.Fatal(err)
	}

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

	r.Get("/users", routes.GetUsers)

	r.Get("/register", routes.GetRegister)
	r.Post("/register", routes.PostRegister)

	r.Route("/essay", routes.EssayRouter)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Println(fmt.Sprintf("Starting server at http://localhost:%s", port))
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s",port), r))
}
