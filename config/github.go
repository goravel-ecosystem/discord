package config

import (
	"github.com/goravel/framework/facades"
)

func init() {
	config := facades.Config()
	config.Add("github", map[string]any{
		"secret": config.Env("GITHUB_SECRET", ""),
	})
}
