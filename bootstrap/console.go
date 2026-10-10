package bootstrap

import (
	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/schedule"

	"goravel/app/console/commands"
	"goravel/app/facades"
)

// Commands returns the console commands registered with the application.
func Commands() []console.Command {
	return []console.Command{
		commands.NewHeartbeatCommand(),
	}
}

// Schedule returns the scheduled events registered with the application.
func Schedule() []schedule.Event {
	return []schedule.Event{
		facades.Schedule().Command("heartbeat").EveryMinute(),
	}
}
