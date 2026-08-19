package middleware

import (
	"fmt"
	"time"

	"novaflow/core"
)

// RequestID is an example of a project-specific middleware, stamping every
// response with a unique X-Request-Id header. Register your own custom
// middleware here alongside the framework's built-ins in core/middleware.go.
func RequestID() core.Middleware {
	return func(next core.HandlerFunc) core.HandlerFunc {
		return func(c *core.Context) {
			id := fmt.Sprintf("req_%d", time.Now().UnixNano())
			c.Writer.Header().Set("X-Request-Id", id)
			c.Set("request_id", id)
			next(c)
		}
	}
}
