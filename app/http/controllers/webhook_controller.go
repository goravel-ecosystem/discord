package controllers

import (
	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/facades"

	"goravel/services"
)

type WebhookController struct {
	github services.Github
}

func NewWebhookController() *WebhookController {
	github, err := services.NewGithub()
	if err != nil {
		facades.Log().Error(err.Error())
	}

	return &WebhookController{
		github: github,
	}
}

func (r *WebhookController) Github(ctx http.Context) http.Response {
	if err := r.github.ProcessWebhook(ctx.Request().Origin()); err != nil {
		facades.Log().Error(err.Error())

		return ctx.Response().Json(http.StatusBadRequest, http.Json{
			"error": err.Error(),
		})
	}

	return ctx.Response().Json(http.StatusOK, http.Json{
		"message": "Webhook processed successfully",
	})
}
