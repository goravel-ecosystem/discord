package commands

import (
	"fmt"

	"github.com/go-resty/resty/v2"
	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"
	"github.com/goravel/framework/facades"

	"goravel/services"
)

type Heartbeat struct {
	discord services.Discord
}

func NewHeartbeatCommand() *Heartbeat {
	discord, err := services.NewDiscord()
	if err != nil {
		facades.Log().Error(err.Error())
	}

	return &Heartbeat{discord: discord}
}

// Signature The name and signature of the console command.
func (r *Heartbeat) Signature() string {
	return "heartbeat"
}

// Description The console command description.
func (r *Heartbeat) Description() string {
	return "Heartbeat"
}

// Extend The console command extend.
func (r *Heartbeat) Extend() command.Extend {
	return command.Extend{}
}

// Handle Execute the console command.
func (r *Heartbeat) Handle(ctx console.Context) error {
	client := resty.New()

	channelID := facades.Config().GetString("discord.heartbeat.channel_id")
	urls, ok := facades.Config().Get("discord.heartbeat.url").([]string)
	if !ok {
		facades.Log().Error("invalid url")
		return nil
	}

	for _, url := range urls {
		resp, err := client.R().Get(url)
		if err != nil || (resp.StatusCode() < 200 || resp.StatusCode() >= 300) {
			if err := r.discord.SendMessage(channelID, fmt.Sprintf("Website %s is down", url)); err != nil {
				facades.Log().Error(fmt.Sprintf("Failed to send message to Discord: %v", err))
			}
		}
	}

	return nil
}
