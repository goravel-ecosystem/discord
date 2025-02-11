package config

import (
	"github.com/goravel/framework/facades"
)

func init() {
	config := facades.Config()
	config.Add("discord", map[string]any{
		"bot": map[string]any{
			"token": "user",
		},
		"heartbeat": map[string]any{
			"interval":   10,
			"url":        []string{"https://www.google.com", "https://www.youtube.com"},
			"channel_id": "1234567890",
		},
	})
}
