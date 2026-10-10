package facades

import (
	"github.com/goravel/framework/contracts/log"
)

// Log returns the logging facade.
func Log() log.Log {
	return App().MakeLog()
}
