package services

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-playground/webhooks/v6/github"
	"github.com/goravel/framework/facades"

	"goravel/app/models"
)

type Github interface {
	ProcessWebhook(request *http.Request) error
}

type GithubImpl struct {
	coreRoleID           string
	discord              Discord
	pullRequestChannelID string
	webhook              *github.Webhook
}

func NewGithub() (*GithubImpl, error) {
	webhook, err := github.New(github.Options.Secret(facades.Config().GetString("github.webhook_secret")))
	if err != nil {
		return nil, err
	}

	discord, err := NewDiscord()
	if err != nil {
		return nil, err
	}

	return &GithubImpl{
		coreRoleID:           facades.Config().GetString("discord.roles.core"),
		discord:              discord,
		pullRequestChannelID: facades.Config().GetString("discord.pull_requests.channel_id"),
		webhook:              webhook,
	}, nil
}

func (r *GithubImpl) ProcessWebhook(request *http.Request) error {
	event, err := r.webhook.Parse(request, github.ReleaseEvent, github.PullRequestEvent)
	if err != nil {
		if errors.Is(err, github.ErrEventNotFound) {
			return nil
		}
		return err
	}

	switch e := event.(type) {
	case github.PullRequestPayload:
		if e.Action == "opened" || e.Action == "reopened" {
			return r.handlePullRequestOpenedEvent(e)
		}

		pullRequest, err := r.getPullRequest(e.PullRequest.ID)
		if err != nil {
			return err
		}

		if pullRequest.ID == 0 {
			return nil
		}

		switch e.Action {
		case "ready_for_review":
			return r.handlePullRequestReadyForReviewEvent(pullRequest)
		case "labeled":
			return r.handlePullRequestLabeledEvent(e, pullRequest)
		case "closed":
			return r.handlePullRequestClosedEvent(pullRequest)
		}
	}

	return nil
}

// handlePullRequestOpenedEvent processes a github.PullRequestPayload.
func (r *GithubImpl) handlePullRequestOpenedEvent(payload github.PullRequestPayload) error {
	state := payload.PullRequest.State
	if payload.PullRequest.Draft {
		state = "Draft"
	}

	content := fmt.Sprintf("### 🛠️ New Pull Request Opened\n\n"+
		"Pull Request: [#%d - %s](%s)\n"+
		"Repository: [%s](%s)\n"+
		"Author: [%s](%s)\n"+
		"State: %s\n"+
		"Cc: <@&%s>",
		payload.Number,
		payload.PullRequest.Title,
		payload.PullRequest.HTMLURL,
		payload.Repository.FullName,
		payload.Repository.HTMLURL,
		payload.PullRequest.User.Login,
		payload.PullRequest.User.HTMLURL,
		state,
		r.coreRoleID)

	threadID, err := r.discord.CreateThread(r.pullRequestChannelID, Thread{
		Title:   payload.PullRequest.Title,
		Content: content,
	})
	if err != nil {
		return err
	}

	var pullRequest models.PullRequest
	if err := facades.Orm().Query().Where("github_id", payload.PullRequest.ID).FirstOrCreate(&pullRequest, &models.PullRequest{
		DiscordThreadID: threadID,
		GithubID:        payload.PullRequest.ID,
		Title:           payload.PullRequest.Title,
		Url:             payload.PullRequest.HTMLURL,
	}); err != nil {
		return err
	}

	return nil
}

func (r *GithubImpl) handlePullRequestReadyForReviewEvent(pullRequest *models.PullRequest) error {
	return r.discord.SendMessage(pullRequest.DiscordThreadID, fmt.Sprintf("Opend cc: <@&%s>", r.coreRoleID))
}

func (r *GithubImpl) handlePullRequestLabeledEvent(payload github.PullRequestPayload, pullRequest *models.PullRequest) error {
	for _, label := range payload.PullRequest.Labels {
		if strings.Contains(label.Name, "Review Ready") {
			return r.discord.SendMessage(pullRequest.DiscordThreadID, fmt.Sprintf("Review Ready cc: <@&%s>", r.coreRoleID))
		}
	}

	return nil
}

func (r *GithubImpl) handlePullRequestClosedEvent(pullRequest *models.PullRequest) error {
	if err := r.discord.DeleteThread(pullRequest.DiscordThreadID); err != nil {
		return err
	}

	if _, err := facades.Orm().Query().Delete(&pullRequest); err != nil {
		return err
	}

	return nil
}

func (r *GithubImpl) getPullRequest(id int64) (*models.PullRequest, error) {
	var pullRequest models.PullRequest
	if err := facades.Orm().Query().Where("github_id", id).First(&pullRequest); err != nil {
		return nil, err
	}

	return &pullRequest, nil
}
