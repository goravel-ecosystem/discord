package routes

import (
	"goravel/app/facades"
	"goravel/app/http/controllers"
)

func Api() {
	webhookController := controllers.NewWebhookController()
	facades.Route().Post("/webhooks/github", webhookController.Github)
}
