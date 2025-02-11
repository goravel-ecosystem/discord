package routes

import (
	"github.com/goravel/framework/facades"

	"goravel/app/http/controllers"
)

func Api() {
	webhookController := controllers.NewWebhookController()
	facades.Route().Post("/webhooks/github", webhookController.Github)
}
