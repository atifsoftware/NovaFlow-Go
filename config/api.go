package config

import (
	"novaflow/app/controllers"
	"novaflow/app/middleware"
	"novaflow/core"
)

// registerApiRoutes registers JWT bearer token-based REST API routes.
func registerApiRoutes(app *core.App, kernel *middleware.Kernel) {
	r := app.Router

	if app.DB != nil {
		productCtrl := controllers.NewProductController(app.DB)

		r.Group(core.GroupOptions{
			Prefix:     "/api/v1",
			Middleware: kernel.Groups("api"),
		}, func(r *core.Router) {
			if app.Auth != nil {
				apiAuth := controllers.NewAuthApiController(app.Auth)
				r.Post("/login", apiAuth.Login)
				r.Post("/register", apiAuth.Register)

				if apiAuthMw := kernel.Resolve("api_auth"); apiAuthMw != nil {
					r.Group(core.GroupOptions{Middleware: []core.Middleware{apiAuthMw}}, func(r *core.Router) {
						r.Get("/me", apiAuth.Me)
						r.Resource("/products", productCtrl)
					})
				}
			} else {
				r.Resource("/products", productCtrl)
			}
		})
	}
}
