package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/goravel-ecosystem/discord/internal/contracts"
	"github.com/goravel-ecosystem/discord/internal/services"
)

type WebhookController struct {
	githubService contracts.Github
}

func NewWebhookController() (*WebhookController, error) {
	githubService, err := services.NewGithubService()
	if err != nil {
		return nil, err
	}

	return &WebhookController{
		githubService: githubService,
	}, nil
}

func (r *WebhookController) Github(c *gin.Context) {
	if err := r.githubService.ProcessWebhook(c.Request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Webhook processed successfully"})
}
