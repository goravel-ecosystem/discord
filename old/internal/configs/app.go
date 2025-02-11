package configs

import "github.com/goravel-ecosystem/discord/pkg/config"

func LoadAppConfig() {
	instance := config.GetInstance()
	instance.Add("app", map[string]any{
		"name":      instance.Env("APP_NAME", "Discord bot"),
		"log_level": instance.Env("APP_LOG_LEVEL", "INFO"),
	})
}
