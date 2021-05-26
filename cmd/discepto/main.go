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
		ds := DisceptoServer{EnvConfig: envConfig}
		ds.Setup()
		ds.Run()
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

func (ds *DisceptoServer) setupLogger() {
	var writer io.Writer
	if ds.Debug {
		writer = zerolog.ConsoleWriter{Out: os.Stdout}
	} else {
		writer = os.Stdout
	}
	log := zerolog.New(writer).With().Timestamp().Logger()
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	ds.logger = log
}
func (ds *DisceptoServer) setupTemplates() {
	ds.templates = render.GetTemplates(&ds.EnvConfig)
	ds.templates.SetFS(web.FS)
}
func (ds *DisceptoServer) setupRouter() {
	ds.router = routes.NewRouter(&ds.EnvConfig, &ds.database, ds.logger, &ds.templates)
}
func (ds *DisceptoServer) setupDB() {
	err := db.MigrateUp(ds.DatabaseURL)
	if err != nil {
		ds.logger.Fatal().Err(err).Send()
	}
	db, err := db.Connect(&ds.EnvConfig)
	if err != nil {
		ds.logger.Fatal().AnErr("Connecting to db", err).Send()
	}
	ds.database = db
}
func (ds *DisceptoServer) setupHttpServer() {
	ds.addr = fmt.Sprintf("localhost:%s", ds.EnvConfig.Port)
	ds.httpServer = &http.Server{
		Addr:         ds.addr,
		Handler:      ds.router,
		ReadTimeout:  1 * time.Minute,
		WriteTimeout: 1 * time.Minute,
	}
}
func (ds *DisceptoServer) Setup() {
	ds.setupLogger()
	ds.setupTemplates()
	ds.setupRouter()
	ds.setupDB()
	ds.setupHttpServer()
}
func (ds *DisceptoServer) Shutdown() {
	if err := ds.httpServer.Shutdown(context.Background()); err != nil {
		ds.logger.Error().
			Err(err).
			Msg("Error shutting down")
	}
}
func (ds *DisceptoServer) Run() {
	go func() {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		defer stop()
		<-ctx.Done()
		stop() // Stop listening for signals
		ds.logger.Info().Msg("Shutting down gracefully")
		ds.Shutdown()
	}()
	ds.logger.Info().Str("server_address", ds.addr).Msg("Server listening")
	if err := ds.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		ds.logger.Fatal().
			Err(err).
			Msg("Error starting server")
	}

}
