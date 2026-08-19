package config

import (
	"time"

	"novaflow/app/controllers"
	appmw "novaflow/app/middleware"
	"novaflow/core"
)

// RegisterRoutes wires every route, mirroring NovaFlow PHP's
// config/routes.php ($router->get/post/group). Called once from main.go.
func RegisterRoutes(app *core.App) {
	r := app.Router

	home := controllers.NewHomeController()
	r.Get("/", home.Index)
	r.Get("/health", home.Health)

	// --- Web (session-based) routes ---------------------------------
	if app.Auth != nil {
		authCtrl := controllers.NewAuthController(app.Auth)
		r.Get("/login", authCtrl.ShowLogin)
		r.Post("/login", authCtrl.Login)
		r.Post("/logout", authCtrl.Logout)

		r.Group(core.GroupOptions{Middleware: []core.Middleware{core.Auth(app.Auth)}}, func(r *core.Router) {
			r.Get("/dashboard", authCtrl.Dashboard)
		})
	}

	// --- REST API (JWT-based) routes ---------------------------------
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
