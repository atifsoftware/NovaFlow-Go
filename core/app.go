package core

import (
	"context"
	"html/template"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"golang.org/x/exp/slog"
)


// App is the central object every NovaFlow project builds once in main.go.
// It wires together config, the database connection, the DI container,
// parsed view templates, the auth service, cache, async queue, and the router.
type App struct {
	Config    *Config
	DB        *DB
	Container *Container
	Views     map[string]*template.Template
	Auth      *AuthService
	Router    *Router
	AI        *AIService
	Cache     *Cache
	Queue     *Queue
	WS        *WebSocketHub
	Events    *EventDispatcher
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

	queue := NewQueue(4, 100)
	app := &App{
		Config:    cfg,
		Container: NewContainer(),
		AI:        NewAIService(cfg),
		Cache:     NewCache(time.Minute),
		Queue:     queue,
		WS:        NewWebSocketHub(),
		Events:    NewEventDispatcher(queue),
	}
	app.Container.Bind("ai", app.AI)
	app.Container.Bind("cache", app.Cache)
	app.Container.Bind("queue", app.Queue)
	app.Container.Bind("ws", app.WS)
	app.Container.Bind("events", app.Events)


	driver := cfg.Get("DB_CONNECTION", "")
	host := cfg.Get("DB_HOST", "")
	sqliteFile := cfg.Get("DB_DATABASE", "")

	if driver != "" || host != "" || sqliteFile != "" {
		if driver == "" {
			driver = "mysql"
		}
		dialect := GetDialect(driver)
		dsn := dialect.BuildDSN(cfg)

		shouldConnect := false
		if dialect.DriverName() == "sqlite" && (sqliteFile != "" || strings.EqualFold(driver, "sqlite") || strings.EqualFold(driver, "sqlite3")) {
			shouldConnect = true
		} else if host != "" {
			shouldConnect = true
		}

		if shouldConnect {
			db, err := OpenDBWithDialect(dialect, dsn)
			if err != nil {
				slog.Error("database connection failed", "driver", dialect.DriverName(), "error", err)
			} else {
				app.DB = db
				app.Container.Bind("db", db)
			}
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

// Run starts the HTTP server with graceful shutdown support.
// On receiving SIGINT or SIGTERM, the server stops accepting new connections
// and waits up to 10 seconds for in-flight requests, queue workers, cache workers,
// and database connections to clean up gracefully before exiting.
func (a *App) Run(addr string, globalMw ...Middleware) error {
	handler := withGlobalMiddleware(a.Router, globalMw...)
	srv := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	// Channel to receive OS shutdown signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	errCh := make(chan error, 1)
	go func() {
		slog.Info("NovaFlow server listening", "addr", addr)

		// Auto-open browser if AUTO_OPEN_BROWSER=true in .env (default: false for servers, true for desktop standalone)
		if a.Config.GetBool("AUTO_OPEN_BROWSER", false) {
			go func() {
				time.Sleep(350 * time.Millisecond)
				targetPath := a.Config.Get("AUTO_OPEN_PATH", "/")
				if !strings.HasPrefix(targetPath, "/") {
					targetPath = "/" + targetPath
				}

				host := "http://localhost"
				if strings.HasPrefix(addr, ":") {
					host += addr
				} else {
					host = "http://" + addr
				}
				targetURL := host + targetPath
				slog.Info("opening default browser", "url", targetURL)
				_ = OpenBrowser(targetURL)
			}()
		}

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// Block until we receive a signal or a startup error
	select {
	case sig := <-quit:
		slog.Info("shutdown signal received, initiating graceful shutdown...", "signal", sig.String())
	case err := <-errCh:
		return err
	}

	// Create a deadline context for the shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return a.GracefulShutdown(srv, ctx)
}

// OpenBrowser opens the specified URL in the system's default web browser.
func OpenBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}


// GracefulShutdown coordinates a robust, ordered shutdown of all framework subsystems:
// 1. Stops HTTP listener and finishes in-flight requests (releasing pooled contexts).
// 2. Drains background Queue worker pool of all remaining dispatched jobs.
// 3. Closes background Cache cleanup ticker.
// 4. Closes database connection pool after all handlers and background tasks finish.
func (a *App) GracefulShutdown(srv *http.Server, ctx context.Context) error {
	slog.Info("graceful shutdown: stopping HTTP server and completing in-flight requests...")
	if srv != nil {
		if err := srv.Shutdown(ctx); err != nil {
			slog.Error("forced HTTP server shutdown", "error", err)
			return err
		}
		slog.Info("graceful shutdown: HTTP server stopped")
	}

	// 2. Drain background job queue workers
	if a.Queue != nil {
		slog.Info("graceful shutdown: draining background job queue...")
		a.Queue.Shutdown(3 * time.Second)
	}

	// 3. Stop cache cleanup worker
	if a.Cache != nil {
		slog.Info("graceful shutdown: stopping cache cleanup worker...")
		a.Cache.Close()
	}

	// 4. Close database connection pool after all handlers & queue tasks complete
	if a.DB != nil && a.DB.Conn != nil {
		slog.Info("graceful shutdown: closing database connection pool...")
		if err := a.DB.Conn.Close(); err != nil {
			slog.Error("error closing database connection pool", "error", err)
		}
	}

	// 5. Close WebSocket Hub connections
	if a.WS != nil {
		slog.Info("graceful shutdown: closing active websocket connections...")
		a.WS.Close()
	}

	slog.Info("NovaFlow server stopped gracefully with all resources cleaned up")
	return nil

}


// withGlobalMiddleware adapts the *Router (an http.Handler) so that global
// middleware wraps every request regardless of which route matched.
func withGlobalMiddleware(router *Router, mw ...Middleware) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := newContext(w, r, router.app)
		defer releaseContext(ctx)
		final := func(c *Context) { router.handleContext(c) }
		for i := len(mw) - 1; i >= 0; i-- {
			final = mw[i](final)
		}
		final(ctx)
	})
}

