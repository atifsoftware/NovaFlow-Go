package config

import (
	"novaflow/app/middleware"
	"novaflow/core"
)

// RegisterRoutes wires every route, mirroring NovaFlow PHP's config/routes.php.
// It dispatches route registration to modular route files. Called once from main.go.
func RegisterRoutes(app *core.App, kernel *middleware.Kernel) {
	registerWebRoutes(app, kernel)
	registerApiRoutes(app, kernel)
	registerAuthRoutes(app, kernel)
	registerAdminRoutes(app, kernel)
}
