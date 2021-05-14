package main

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/go-chi/chi"
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
		server.Start()
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
	logger    zerolog.Logger
	router    chi.Router
	database  db.SharedDB
	templates render.Templates
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
func (server *DisceptoServer) listen() {
	addr := fmt.Sprintf("http://localhost:%s", server.EnvConfig.Port)
	server.logger.Info().Str("server_address", addr).Msg("Server is starting")
	server.logger.Error().
		Err(http.ListenAndServe(fmt.Sprintf(":%s", server.EnvConfig.Port), server.router)).
		Msg("Server startup failed")
}
func (server *DisceptoServer) Start() {
	server.setupLogger()
	server.setupTemplates()
	server.setupRouter()
	server.setupDB()
	server.listen()
}
