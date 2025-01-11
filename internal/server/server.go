package server

import (
	"net/http"
	"os"

	"github.com/goravel-ecosystem/discord/pkg/httpframework"
)

func Init() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	err := http.ListenAndServe(":"+port, httpframework.GetInstance())
	if err != nil {
		panic(err)
	}
}
