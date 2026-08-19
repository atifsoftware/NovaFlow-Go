package config

import (
	"time"

	"novaflow/app/controllers"
	appmw "novaflow/app/middleware"
	"novaflow/core"
)

// registerApiRoutes registers JWT bearer token-based REST API routes.
func registerApiRoutes(app *core.App) {
	r := app.Router

	if app.DB != nil {
		productCtrl := controllers.NewProductController(app.DB)
		limiter := core.NewRateLimiter(60, time.Minute)

		r.Group(core.GroupOptions{
			Prefix:     "/api/v1",
			Middleware: []core.Middleware{core.CORS(), limiter.Middleware(), appmw.RequestID()},
		}, func(r *core.Router) {
			if app.Auth != nil {
				apiAuth := controllers.NewAuthApiController(app.Auth)
				r.Post("/login", apiAuth.Login)
				r.Post("/register", apiAuth.Register)

				r.Group(core.GroupOptions{Middleware: []core.Middleware{core.AuthAPI(app.Auth)}}, func(r *core.Router) {
					r.Get("/me", apiAuth.Me)
					r.Resource("/products", productCtrl)
				})
			} else {
				r.Resource("/products", productCtrl)
			}
		})
	}
}
