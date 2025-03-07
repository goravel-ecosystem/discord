package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/goravel/framework/facades"

	"goravel/bootstrap"
	"goravel/services"
)

func main() {
	// This bootstraps the framework and gets it ready for use.
	bootstrap.Boot()

	// Create a channel to listen for OS signals
	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start http server by facades.Route().
	go func() {
		if err := facades.Route().Run(); err != nil {
			facades.Log().Errorf("Route Run error: %v", err)
		}
	}()

	go facades.Schedule().Run()

	// Listen for the OS signal
	go func() {
		<-quit
		if err := facades.Route().Shutdown(); err != nil {
			facades.Log().Errorf("Route Shutdown error: %v", err)
		}

		discord, err := services.NewDiscord()
		if err == nil {
			discord.Close()
		}

		os.Exit(0)
	}()

	select {}
}
