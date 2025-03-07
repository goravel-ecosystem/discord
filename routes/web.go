package routes

import (
	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/facades"
)

func Web() {
	facades.Route().Get("/", func(ctx http.Context) http.Response {
		facades.Log().Info("Running")

		return ctx.Response().String(http.StatusOK, "Running")
	})
}
