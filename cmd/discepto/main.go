package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
	"gitlab.com/ranfdev/discepto/internal/db"
	"gitlab.com/ranfdev/discepto/internal/models"
	"gitlab.com/ranfdev/discepto/internal/render"
	"gitlab.com/ranfdev/discepto/internal/routes"
	"gitlab.com/ranfdev/discepto/web"
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
	envConfig := models.ReadEnvConfig()
	switch os.Args[1] {
	case "start":
		server := DisceptoServer{EnvConfig: envConfig}
		server.Setup()
		server.Run()
	case "migrate":
		var err error
		switch os.Args[2] {
		case "up":
			err = db.MigrateUp(envConfig.DatabaseURL)
		case "down":
			err = db.MigrateDown(envConfig.DatabaseURL)
		case "drop":
			err = db.Drop(envConfig.DatabaseURL)
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

type DisceptoServer struct {
	models.EnvConfig
	addr              string
	logger            zerolog.Logger
	router            chi.Router
	httpServer        *http.Server
	database          db.SharedDB
	templates         render.Templates
	cancelBaseContext context.Context
}

func (server *DisceptoServer) setupLogger() {
	var writer io.Writer
	if server.Debug {
		writer = zerolog.ConsoleWriter{Out: os.Stdout}
	} else {
		writer = os.Stdout
	}
	log := zerolog.New(writer).With().Timestamp().Logger()
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	server.logger = log
}
func (server *DisceptoServer) setupTemplates() {
	server.templates = render.GetTemplates(&server.EnvConfig)
	server.templates.SetFS(web.FS)
}
func (server *DisceptoServer) setupRouter() {
	server.router = routes.NewRouter(&server.EnvConfig, &server.database, server.logger, &server.templates)
}
func (server *DisceptoServer) setupDB() {
	err := db.MigrateUp(server.DatabaseURL)
	if err != nil {
		server.logger.Fatal().Err(err).Send()
	}
	db, err := db.Connect(&server.EnvConfig)
	if err != nil {
		server.logger.Fatal().AnErr("Connecting to db", err).Send()
	}
	server.database = db
}
func (server *DisceptoServer) setupHttpServer() {
	server.addr = fmt.Sprintf("http://localhost:%s", server.EnvConfig.Port)
	server.httpServer = &http.Server{
		Addr:         server.addr,
		Handler:      server.router,
		ReadTimeout:  1 * time.Minute,
		WriteTimeout: 1 * time.Minute,
	}
}
func (server *DisceptoServer) Setup() {
	server.setupLogger()
	server.setupTemplates()
	server.setupRouter()
	server.setupDB()
	server.setupHttpServer()
}
func (server *DisceptoServer) Shutdown() {
	if err := server.httpServer.Shutdown(context.Background()); err != nil {
		server.logger.Error().
			Err(err).
			Msg("Error shutting down")
	}
}
func (server *DisceptoServer) Run() {
	server.logger.Info().Str("server_address", server.addr).Msg("Server is starting")
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	go server.httpServer.ListenAndServe()
	server.logger.Info().Msg("Ready")

	<-ctx.Done()
	stop() // Stop listening for signals
	server.logger.Info().Msg("Shutting down gracefully")
	server.Shutdown()
}
