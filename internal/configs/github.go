package configs

import "github.com/goravel-ecosystem/discord/pkg/config"

func LoadGithubConfig() {
	instance := config.GetInstance()
	instance.Add("github", map[string]any{
		"secret": instance.Env("GITHUB_WEBHOOK_SECRET", ""),
	})
}
