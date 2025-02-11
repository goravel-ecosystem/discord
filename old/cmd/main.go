package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog/log"

	"github.com/goravel-ecosystem/discord/internal/api"
	"github.com/goravel-ecosystem/discord/internal/configs"
	"github.com/goravel-ecosystem/discord/internal/server"
	"github.com/goravel-ecosystem/discord/internal/services"
	"github.com/goravel-ecosystem/discord/pkg/config"
	"github.com/goravel-ecosystem/discord/pkg/discord"
	"github.com/goravel-ecosystem/discord/pkg/httpframework"
	"github.com/goravel-ecosystem/discord/pkg/logger"
)

func main() {
	configs.Init()
	logger.Init(config.GetInstance().GetString("app.log_level"))
	httpframework.Init()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	err := discord.Init(config.GetInstance().GetString("discord.bot_token"))
	if err != nil {
		log.Fatal().Msgf("Failed to initialize Discord service: %v", err)
	}

	err = discord.GetInstance().Open()
	if err != nil {
		log.Fatal().Msgf("Failed to start Discord service: %v", err)
	}

	uptimeService := services.NewUptimeService()
	go uptimeService.Monitor(ctx)

	// Listen for the OS signal
	go func() {
		<-ctx.Done()
		if err := discord.GetInstance().Close(); err != nil {
			log.Error().Msgf("Failed to close Discord service: %v", err)
		}

		log.Info().Msg("Shutting down Discord service")
		os.Exit(0)
	}()

	api.Init()
	server.Init()
	select {}
}
