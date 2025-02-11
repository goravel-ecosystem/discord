package services

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/go-playground/webhooks/v6/github"
	"github.com/goravel/framework/facades"
)

type Github interface {
	ProcessWebhook(request *http.Request) error
}

type GithubImpl struct {
	discord Discord
	webhook *github.Webhook
}

func NewGithub() (*GithubImpl, error) {
	webhook, err := github.New(github.Options.Secret(facades.Config().GetString("github.secret")))
	if err != nil {
		return nil, err
	}

	discord, err := NewDiscord()
	if err != nil {
		return nil, err
	}

	return &GithubImpl{
		discord: discord,
		webhook: webhook,
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
		return r.handlePullRequestEvent(e)
	}

	return nil
}

// handlePullRequestEvent processes a github.PullRequestPayload.
func (r *GithubImpl) handlePullRequestEvent(payload github.PullRequestPayload) error {
	if payload.Action != "opened" {
		return nil
	}

	channelID := facades.Config().GetString("discord.pull_requests.channel_id")
	content := fmt.Sprintf("### 🛠️ New Pull Request Opened\n\n"+
		"**Pull Request**: [#%d - %s](%s)\n"+
		"**Repository**: [%s](%s)\n"+
		"**Author**: [%s](%s)\n\n"+
		"**State**: %s\n\n"+
		"CC: <@&%s>",
		payload.Number,
		payload.PullRequest.Title,
		payload.PullRequest.HTMLURL,
		payload.Repository.FullName,
		payload.Repository.HTMLURL,
		payload.PullRequest.User.Login,
		payload.PullRequest.User.HTMLURL,
		payload.PullRequest.State,
		facades.Config().GetString("discord.roles.core"))

	threadID, err := r.discord.CreateThread(channelID, Thread{
		Title:   payload.PullRequest.Title,
		Content: content,
	})
	if err != nil {
		return err
	}

	// TODO: Save thread ID to database
	fmt.Println(threadID)

	return nil
}
