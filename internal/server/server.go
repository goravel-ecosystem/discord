package server

import (
	"net/http"
	"os"

	"github.com/rs/zerolog/log"

	"github.com/goravel-ecosystem/discord/pkg/httpframework"
)

func Init() {
	host := os.Getenv("APP_HOST")
	if host == "" {
		host = "localhost"
	}

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	addr := host + ":" + port

	err := http.ListenAndServe(addr, httpframework.GetInstance())
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to start server")
	}
}
