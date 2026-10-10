package facades

import (
	"github.com/goravel/framework/contracts/config"
)

// Config returns the configuration facade.
func Config() config.Config {
	return App().MakeConfig()
}
