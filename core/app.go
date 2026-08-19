package core

import (
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// App is the central object every NovaFlow project builds once in main.go.
// It wires together config, the database connection, the DI container,
// parsed view templates, the auth service, and the router.
type App struct {
	Config    *Config
	DB        *DB
	Container *Container
	Views     map[string]*template.Template
	Auth      *AuthService
	Router    *Router
}

// NewApp loads .env, connects to the database (if DB_HOST is set),
// compiles views, and builds an empty router ready for routes to be
// registered against.
func NewApp(envPath, viewsDir string) *App {
	cfg, err := LoadEnv(envPath)
	InitLogger(cfg.Get("APP_ENV", "local"))

	if err != nil {
		slog.Warn("could not load env file", "path", envPath, "error", err)
	}

	app := &App{
		Config:    cfg,
		Container: NewContainer(),
	}

	if host := cfg.Get("DB_HOST", ""); host != "" {
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4",
			cfg.Get("DB_USER", "root"),
			cfg.Get("DB_PASS", ""),
			host,
			cfg.Get("DB_PORT", "3306"),
			cfg.Get("DB_NAME", "novaflow_db"),
		)
		db, err := OpenDB("mysql", dsn)
		if err != nil {
			slog.Error("database connection failed", "error", err)
		} else {
			app.DB = db
			app.Container.Bind("db", db)
		}
	}

	if app.DB != nil {
		app.Auth = NewAuthService(app.DB, cfg.Get("JWT_SECRET", "change-me-in-.env"))
		app.Container.Bind("auth", app.Auth)
	}

	if viewsDir != "" {
		views, err := LoadViews(viewsDir)
		if err != nil {
			slog.Error("view compilation failed", "error", err)
		}
		app.Views = views
	}

	app.Router = NewRouter(app)
	return app
}

// Run starts the HTTP server, wrapping the router with Recover + Logger
// global middleware so a panicking handler never takes the process down.
// Run starts the HTTP server, wrapping the router with the provided
// global middleware chain so requests are logged, recovered, etc.
func (a *App) Run(addr string, globalMw ...Middleware) error {
	handler := withGlobalMiddleware(a.Router, globalMw...)
	srv := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}
	slog.Info("NovaFlow server listening", "addr", addr)
	return srv.ListenAndServe()
}

// withGlobalMiddleware adapts the *Router (an http.Handler) so that global
// middleware wraps every request regardless of which route matched.
func withGlobalMiddleware(router *Router, mw ...Middleware) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		final := func(c *Context) { router.ServeHTTP(c.Writer, c.Request) }
		for i := len(mw) - 1; i >= 0; i-- {
			final = mw[i](final)
		}
		final(newContext(w, r, router.app))
	})
}
