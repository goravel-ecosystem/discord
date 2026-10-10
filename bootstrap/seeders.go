package bootstrap

import (
	"github.com/goravel/framework/contracts/database/seeder"

	"goravel/database/seeders"
)

// Seeders returns the database seeders registered with the application.
func Seeders() []seeder.Seeder {
	return []seeder.Seeder{
		&seeders.DatabaseSeeder{},
	}
}
