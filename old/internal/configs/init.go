package configs

import (
	"github.com/goravel-ecosystem/discord/pkg/config"
	"github.com/goravel-ecosystem/discord/support"
)

func Init() {
	config.InitEnv(support.EnvPath)

	LoadAppConfig()
	LoadDiscordConfig()
	LoadGithubConfig()
}
