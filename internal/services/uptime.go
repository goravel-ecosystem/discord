package services

import (
	"context"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog/log"

	"github.com/goravel-ecosystem/discord/internal/contracts"
	"github.com/goravel-ecosystem/discord/pkg/config"
	"github.com/goravel-ecosystem/discord/pkg/discord"
)

type UptimeService struct{}

func NewUptimeService() contracts.Uptime {
	return &UptimeService{}
}

func (r *UptimeService) Monitor(ctx context.Context) {
	client := resty.New()
	configInstance := config.GetInstance()

	interval := configInstance.GetInt("discord.uptime.interval")
	ticker := time.NewTicker(time.Duration(interval) * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("Stopping uptime monitoring...")
			return
		case <-ticker.C:
			resp, err := client.R().Get(configInstance.GetString("discord.uptime.website_url"))

			var message string
			if err != nil || resp.StatusCode() >= 100 {
				message = configInstance.GetString("discord.uptime.message")
			}

			if message != "" {
				_, err = discord.GetInstance().ChannelMessageSend(configInstance.GetString("discord.uptime.channel_id"), message)
				if err != nil {
					log.Error().Msgf("Error sending message to Discord: %v", err)
				}
			}
		}
	}
}
