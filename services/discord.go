package services

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/goravel/framework/facades"
)

type Discord interface {
	CreateThread(channelID string, thread Thread) (string, error)
	DeleteThread(threadID string) error
	SendMessage(channelID string, message string) error
}

type Thread struct {
	Title   string
	Content string
}

type DiscordImpl struct {
	session *discordgo.Session
}

func NewDiscord() (*DiscordImpl, error) {
	session, err := discordgo.New("Bot " + facades.Config().GetString("discord.bot.token"))
	if err != nil {
		return nil, err
	}

	return &DiscordImpl{
		session: session,
	}, nil
}

func (r *DiscordImpl) CreateThread(channelID string, thread Thread) (string, error) {
	createdThread, err := r.session.ThreadStartComplex(channelID, &discordgo.ThreadStart{
		Name: thread.Title,
		Type: discordgo.ChannelTypeGuildPublicThread,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create thread for pull request: %w", err)
	}

	if err := r.SendMessage(createdThread.ID, thread.Content); err != nil {
		return "", fmt.Errorf("failed to send message in thread: %w", err)
	}

	return createdThread.ID, nil
}

func (r *DiscordImpl) DeleteThread(threadID string) error {
	_, err := r.session.ChannelDelete(threadID)
	if err != nil {
		return fmt.Errorf("failed to delete thread: %w", err)
	}

	return nil
}

func (r *DiscordImpl) SendMessage(channelID string, message string) error {
	_, err := r.session.ChannelMessageSend(channelID, message)
	if err != nil {
		return fmt.Errorf("failed to send message to Discord: %w", err)
	}

	return nil
}
