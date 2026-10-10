package facades

import (
	contractsfoundation "github.com/goravel/framework/contracts/foundation"
	"github.com/goravel/framework/errors"
	"github.com/goravel/framework/foundation"
)

// App returns the current application instance.
func App() contractsfoundation.Application {
	if foundation.App == nil {
		panic(errors.ApplicationNotSet.SetModule(errors.ModuleFacade))
	}

	return foundation.App
}
