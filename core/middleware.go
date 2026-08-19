package core

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/exp/slog"
)

// responseCapture wraps http.ResponseWriter to capture the status code written by a handler.
type responseCapture struct {
	http.ResponseWriter
	statusCode int
}

func (rc *responseCapture) WriteHeader(code int) {
	rc.statusCode = code
	rc.ResponseWriter.WriteHeader(code)
}

// Logger logs method, path, status code, and duration for every request using slog.
func Logger() Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) {
			start := time.Now()
			// BUG-07: wrap writer to capture the response status code
			rc := &responseCapture{ResponseWriter: c.Writer, statusCode: http.StatusOK}
			c.Writer = rc
			next(c)
			slog.Info("HTTP request",
				"method", c.Request.Method,
				"path", c.Request.URL.Path,
				"status", rc.statusCode,
				"duration", time.Since(start),
				"client_ip", clientIP(c.Request),
			)
		}
	}
}

// Recover turns a panic inside a handler into an Intelligent Error Page (dev) or generic error (prod).
func Recover() Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) {
			defer func() {
				if err := recover(); err != nil {
					detail := NewPanicDetail(err, c.Request)

					slog.Error("panic recovered",
						"error", detail.Message,
						"file", detail.File,
						"line", detail.Line,
						"function", detail.Function,
					)

					isAPI := strings.Contains(c.Request.URL.Path, "/api/") || strings.Contains(c.Request.Header.Get("Accept"), "application/json")
					isLocal := c.app.Config.Get("APP_ENV", "local") != "production"

					if isLocal {
						if isAPI {
							c.JSON(http.StatusInternalServerError, map[string]interface{}{
								"success": false,
								"message": detail.Message,
								"file":    filepath.Base(detail.File),
								"line":    detail.Line,
							})
						} else {
							RenderIntelligentError(c.Writer, detail)
						}
					} else {
						if isAPI {
							c.JSONError(http.StatusInternalServerError, "internal server error")
						} else {
							c.String(http.StatusInternalServerError, "internal server error")
						}
					}
				}
			}()
			next(c)
		}
	}
}

// CORS adds permissive CORS headers; pass explicit origins for production.
func CORS(allowedOrigins ...string) Middleware {
	origin := "*"
	if len(allowedOrigins) > 0 {
		origin = strings.Join(allowedOrigins, ", ")
	}
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			if c.Request.Method == http.MethodOptions {
				c.Writer.WriteHeader(http.StatusNoContent)
				return
			}
			next(c)
		}
	}
}

// Auth requires a valid session (cookie-based, for web routes). It reads
// the "session_user" cookie set by AuthService.Login and rejects the
// request if missing/invalid, storing the user id in the context on success.
func Auth(auth *AuthService) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) {
			userID, ok := auth.CurrentSessionUser(c.Request)
			if !ok {
				if strings.Contains(c.Request.Header.Get("Accept"), "application/json") {
					c.JSONError(http.StatusUnauthorized, "unauthenticated")
				} else {
					c.Redirect("/login", http.StatusFound)
				}
				return
			}
			c.Set("user_id", userID)
			next(c)
		}
	}
}

// AuthAPI requires a valid "Authorization: Bearer <jwt>" header — used for
// the api/v1 route group. On success the JWT claims are stored in context.
func AuthAPI(auth *AuthService) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) {
			header := c.Request.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				c.JSONError(http.StatusUnauthorized, "missing bearer token")
				return
			}
			token := strings.TrimPrefix(header, "Bearer ")
			claims, err := ParseJWT(token, auth.jwtSecret)
			if err != nil {
				c.JSONError(http.StatusUnauthorized, "invalid or expired token")
				return
			}
			c.Set("claims", claims)
			if sub, ok := claims["sub"]; ok {
				c.Set("user_id", sub)
			}
			next(c)
		}
	}
}

// --- Rate limiting (fixed-window, in-memory, per client IP) --------------

type rateBucket struct {
	count     int
	windowEnd time.Time
}

// RateLimiter implements simple brute-force protection: at most `max`
// requests per client IP within `window`.
type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*rateBucket
	max     int
	window  time.Duration
}

func NewRateLimiter(max int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{buckets: make(map[string]*rateBucket), max: max, window: window}
	// BUG-05: start a background goroutine to clean up expired buckets,
	// preventing unbounded memory growth for long-running servers.
	go func() {
		ticker := time.NewTicker(window)
		defer ticker.Stop()
		for range ticker.C {
			now := time.Now()
			rl.mu.Lock()
			for key, b := range rl.buckets {
				if now.After(b.windowEnd) {
					delete(rl.buckets, key)
				}
			}
			rl.mu.Unlock()
		}
	}()
	return rl
}

func (rl *RateLimiter) Middleware() Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) {
			ip := clientIP(c.Request)
			if !rl.allow(ip) {
				c.JSONError(http.StatusTooManyRequests, "too many requests, please slow down")
				return
			}
			next(c)
		}
	}
}

func (rl *RateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	b, ok := rl.buckets[key]
	if !ok || now.After(b.windowEnd) {
		rl.buckets[key] = &rateBucket{count: 1, windowEnd: now.Add(rl.window)}
		return true
	}
	if b.count >= rl.max {
		return false
	}
	b.count++
	return true
}

func clientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return strings.Split(fwd, ",")[0]
	}
	return r.RemoteAddr
}

// ensure fmt import is used if future edits remove other usages
var _ = fmt.Sprintf
