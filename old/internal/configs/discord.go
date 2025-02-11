package configs

import "github.com/goravel-ecosystem/discord/pkg/config"

func LoadDiscordConfig() {
	instance := config.GetInstance()
	instance.Add("discord", map[string]any{
		"bot_token": instance.Env("DISCORD_BOT_TOKEN"),
		"uptime": map[string]any{
			"channel_id":  instance.Env("UPTIME_CHANNEL_ID"),
			"website_url": instance.Env("UPTIME_WEBSITE_URL"),
			"interval":    instance.Env("UPTIME_CHECK_INTERVAL", 60), // minutes
			"message": instance.Env("UPTIME_ALERT_MESSAGE",
				"🚨 The documentation website is experiencing issues!\n"),
		},
		"pull_requests": map[string]any{
			"channel_id": instance.Env("DISCORD_PULL_REQUEST_CHANNEL_ID"),
		},
		"roles": map[string]any{
			"core": instance.Env("DISCORD_CORE_ROLE_ID"),
		},
	})
}
