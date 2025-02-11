package commands

import (
	"fmt"
	"goravel/services"

	"github.com/go-resty/resty/v2"
	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"
	"github.com/goravel/framework/facades"
	"github.com/rs/zerolog/log"
)

type Heartbeat struct {
	discord services.Discord
}

func NewHeartbeat(discord services.Discord) *Heartbeat {
	return &Heartbeat{discord: discord}
}

// Signature The name and signature of the console command.
func (receiver *Heartbeat) Signature() string {
	return "heartbeat"
}

// Description The console command description.
func (receiver *Heartbeat) Description() string {
	return "Heartbeat"
}

// Extend The console command extend.
func (receiver *Heartbeat) Extend() command.Extend {
	return command.Extend{}
}

// Handle Execute the console command.
func (receiver *Heartbeat) Handle(ctx console.Context) error {
	client := resty.New()

	urls, ok := facades.Config().Get("discord.heartbeat.url").([]string)
	if !ok {
		facades.Log().Error("invalid url")
		return nil
	}

	for _, url := range urls {
		resp, err := client.R().Get(url)

		var message string
		if err != nil || (resp.StatusCode() < 200 || resp.StatusCode() >= 300) {
			message = fmt.Sprintf("Website %s is down", url)
		}

		if message != "" {
			if err := receiver.discord.SendMessage(facades.Config().GetString("discord.heartbeat.channel_id"), message); err != nil {
				log.Error().Msgf("Failed to send message to Discord: %v", err)
			}
		}
	}

	return nil
}
