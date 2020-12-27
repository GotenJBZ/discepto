package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/markbates/pkger"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"gitlab.com/ranfdev/discepto/internal/db"
	"gitlab.com/ranfdev/discepto/internal/routes"
)

const usage = `Usage:
	- start
	- migrate [up/down]
`

func main() {
	if len(os.Args) == 1 {
		fmt.Println(usage)
		return
	}
	switch os.Args[1] {
	case "start":
		Start()
	case "migrate":
		var err error
		switch os.Args[2] {
		case "up":
			err = db.MigrateUp()
		case "down":
			err = db.MigrateDown()
		case "drop":
			err = db.Drop()
		default:
			fmt.Println(usage)
			return
		}
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println("Done")
	default:
		fmt.Println(usage)
	}
}

func Start() {
	err := db.Connect()
	if err != nil {
		log.Fatal(err)
	}
	err = db.MigrateUp()
	if err != nil {
		log.Fatal(err)
	}

	r := chi.NewRouter()

	var writer io.Writer
	if os.Getenv("DEBUG") == "true" {
		writer = zerolog.ConsoleWriter{Out: os.Stdout}
	} else {
		writer = os.Stdout
	}
	log := zerolog.New(writer).With().Timestamp().Logger()
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	logger := hlog.AccessHandler(func(r *http.Request, status, size int, duration time.Duration) {
		hlog.
			FromRequest(r).
			Info().
			Str("request_id", middleware.GetReqID(r.Context())).
			Int("status", status).
			Str("url", r.URL.String()).
			Str("method", r.Method).Int("size", size).
			Dur("duration", duration).
			Str("ip", r.RemoteAddr).
			Str("user_agent", r.UserAgent()).
			Msg("")
	})

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(hlog.NewHandler(log))
	r.Use(logger)

	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(15 * time.Second))

	// Serve static files
	staticFileServer := http.FileServer(pkger.Dir("/web/static"))
	r.Get("/static/*", func(w http.ResponseWriter, r *http.Request) {
		fs := http.StripPrefix("/static", staticFileServer)
		fs.ServeHTTP(w, r)
	})

	// Serve dynamic routes
	r.Get("/", routes.GetHome)

	r.Get("/users", routes.AppHandler(routes.GetUsers))

	r.Get("/signup", routes.GetSignup)
	r.Post("/signup", routes.AppHandler(routes.PostSignup))

	r.Route("/essays", routes.EssaysRouter)
	r.Get("/newessay", routes.GetNewEssay)

	port := os.Getenv("PORT")
	if port == "" {
		port = "23495"
	}
	addr := fmt.Sprintf("http://localhost:%s", port)
	log.Info().Str("server_address", addr).Msg("Server is starting")
	log.Error().Err(http.ListenAndServe(fmt.Sprintf(":%s", port), r)).Msg("Server startup failed")
}
