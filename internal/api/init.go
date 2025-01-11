package api

import (
	"github.com/rs/zerolog/log"

	"github.com/goravel-ecosystem/discord/internal/api/controller"
	"github.com/goravel-ecosystem/discord/pkg/httpframework"
)

var (
	HealthCheckPath   = "/health"
	GithubWebhookPath = "/webhooks/github"
)

func Init() {
	webhookController, err := controller.NewWebhookController()
	if err != nil {
		log.Fatal().Msgf("Failed to initialize webhook controller: %v\n", err.Error())
	}

	httpframework.GetInstance().GET(HealthCheckPath, healthProvider)
	httpframework.GetInstance().POST(GithubWebhookPath, webhookController.Github)
}
