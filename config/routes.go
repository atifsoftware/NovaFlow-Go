package config

import (
	"novaflow/core"
)

// RegisterRoutes wires every route, mirroring NovaFlow PHP's config/routes.php.
// It dispatches route registration to modular route files. Called once from main.go.
func RegisterRoutes(app *core.App) {
	registerWebRoutes(app)
	registerApiRoutes(app)
	registerAuthRoutes(app)
	registerAdminRoutes(app)
}
