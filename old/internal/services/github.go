package services

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/bwmarrin/discordgo"
	"github.com/go-playground/webhooks/v6/github"
	"github.com/rs/zerolog/log"

	"github.com/goravel-ecosystem/discord/internal/contracts"
	"github.com/goravel-ecosystem/discord/pkg/config"
	"github.com/goravel-ecosystem/discord/pkg/discord"
)

type GithubService struct {
	hook *github.Webhook
}

// NewGithubService creates a new instance of contracts.Github.
func NewGithubService() (contracts.Github, error) {
	hook, err := github.New(github.Options.Secret(config.GetInstance().GetString("github.secret")))
	if err != nil {
		return nil, err
	}

	return &GithubService{
		hook: hook,
	}, nil
}

// ProcessWebhook parses and consumes only supported events, ignoring others.
func (g *GithubService) ProcessWebhook(r *http.Request) error {
	event, err := g.hook.Parse(r, github.ReleaseEvent, github.PullRequestEvent)
	if err != nil {
		if errors.Is(err, github.ErrEventNotFound) {
			return nil
		}
		return err
	}

	switch e := event.(type) {
	case github.ReleasePayload:
		g.handleReleaseEvent(e)
	case github.PullRequestPayload:
		g.handlePullRequestEvent(e)
	}

	return nil
}

// handleReleaseEvent processes a github.ReleasePayload.
func (g *GithubService) handleReleaseEvent(payload github.ReleasePayload) {
}

// handlePullRequestEvent processes a github.PullRequestPayload.
func (g *GithubService) handlePullRequestEvent(payload github.PullRequestPayload) {
	if payload.Action != "opened" {
		return
	}

	thread, err := discord.GetInstance().ThreadStartComplex(config.GetInstance().GetString("discord.pull_requests.channel_id"), &discordgo.ThreadStart{
		Name: payload.PullRequest.Title,
		Type: discordgo.ChannelTypeGuildPublicThread,
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to create thread for pull request")
		return
	}

	messageContent := fmt.Sprintf("### 🛠️ New Pull Request Opened\n\n"+
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
		config.GetInstance().GetString("discord.roles.core"))

	message := &discordgo.MessageSend{
		Content: messageContent,
		Flags:   discordgo.MessageFlagsSuppressEmbeds,
	}

	_, err = discord.GetInstance().ChannelMessageSendComplex(thread.ID, message)
	if err != nil {
		log.Error().Err(err).Msg("Failed to send message in thread")
	}
}
