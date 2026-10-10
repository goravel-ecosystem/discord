package bootstrap

import (
	contractsfoundation "github.com/goravel/framework/contracts/foundation"
	"github.com/goravel/framework/foundation"

	"goravel/app/runners"
	"goravel/config"
	"goravel/routes"
)

// Boot builds the application: providers, commands, schedule, migrations,
// seeders, routes, config and the Discord runner are all wired through the
// foundation.Setup builder.
func Boot() contractsfoundation.Application {
	return foundation.Setup().
		WithMigrations(Migrations).
		WithSeeders(Seeders).
		WithProviders(Providers).
		WithCommands(Commands).
		WithSchedule(Schedule).
		WithRunners(func() []contractsfoundation.Runner {
			return []contractsfoundation.Runner{
				runners.NewDiscord(foundation.App),
			}
		}).
		WithRouting(func() {
			routes.Web()
			routes.Api()
			routes.Grpc()
		}).
		WithConfig(config.Boot).
		Create()
}
