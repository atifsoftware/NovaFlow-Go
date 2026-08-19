package middleware

import (
	"time"

	"novaflow/core"
)

// Kernel acts as the central hub for defining and registering middlewares,
// mirroring PHP NovaFlow's Kernel.php.
type Kernel struct {
	app *core.App
}

// NewKernel creates a new Kernel instance linked to the main App.
func NewKernel(app *core.App) *Kernel {
	return &Kernel{app: app}
}

// Global returns middleware that runs on EVERY HTTP request.
func (k *Kernel) Global() []core.Middleware {
	return []core.Middleware{
		core.Recover(),
		core.Logger(),
		core.SecurityHeaders(),
	}
}

// Groups defines middleware groups (like web, api).
func (k *Kernel) Groups(group string) []core.Middleware {
	switch group {
	case "web":
		return []core.Middleware{
			core.CSRF(),
		}
	case "api":
		limiter := core.NewRateLimiter(60, time.Minute)
		return []core.Middleware{
			core.CORS(),
			limiter.Middleware(),
			RequestID(),
		}
	default:
		return nil
	}
}

// Resolve translates an alias string (e.g., "auth", "api_auth", "csrf") into its middleware handler.
func (k *Kernel) Resolve(alias string) core.Middleware {
	switch alias {
	case "auth":
		if k.app.Auth != nil {
			return core.Auth(k.app.Auth)
		}
	case "api_auth":
		if k.app.Auth != nil {
			return core.AuthAPI(k.app.Auth)
		}
	case "csrf":
		return core.CSRF()
	case "security":
		return core.SecurityHeaders()
	}
	return nil
}

