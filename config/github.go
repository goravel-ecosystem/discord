package config

import (
	"github.com/goravel/framework/facades"
)

func init() {
	config := facades.Config()
	config.Add("github", map[string]any{
		"webhook_secret": config.Env("GITHUB_WEBHOOK_SECRET", ""),
	})
}
