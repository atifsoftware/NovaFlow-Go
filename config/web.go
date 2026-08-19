package config

import (
	"novaflow/app/controllers"
	"novaflow/app/middleware"
	"novaflow/core"
)

// registerWebRoutes registers session/cookie-based routes.
func registerWebRoutes(app *core.App, kernel *middleware.Kernel) {
	r := app.Router

	home := controllers.NewHomeController()
	r.Get("/", home.Index)
	r.Get("/health", home.Health)
	r.Static("/public", "public")


	// API Interactive Swagger Documentation routes
	docs := controllers.NewDocsController()
	r.Get("/docs", docs.ShowDocs)
	r.Get("/openapi.json", docs.OpenAPISpec)

	// Web (session-based) routes
	if app.Auth != nil {
		authCtrl := controllers.NewAuthController(app.Auth)
		r.Get("/login", authCtrl.ShowLogin)
		r.Post("/login", authCtrl.Login)
		r.Post("/logout", authCtrl.Logout)

		if authMw := kernel.Resolve("auth"); authMw != nil {
			r.Group(core.GroupOptions{Middleware: []core.Middleware{authMw}}, func(r *core.Router) {
				r.Get("/dashboard", authCtrl.Dashboard)
			})
		}
	}
}
