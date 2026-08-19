package tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"novaflow/app/middleware"
	"novaflow/config"
	"novaflow/core"
)

func TestRouterBasicGet(t *testing.T) {
	app := &core.App{Container: core.NewContainer()}
	app.Router = core.NewRouter(app)
	app.Router.Get("/ping", func(c *core.Context) {
		c.JSON(http.StatusOK, map[string]string{"pong": "true"})
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()
	app.Router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestRouterDynamicParam(t *testing.T) {
	app := &core.App{Container: core.NewContainer()}
	app.Router = core.NewRouter(app)
	var captured string
	app.Router.Get("/users/:id", func(c *core.Context) {
		captured = c.Param("id")
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/users/42", nil)
	rec := httptest.NewRecorder()
	app.Router.ServeHTTP(rec, req)

	if captured != "42" {
		t.Fatalf("expected param 42, got %q", captured)
	}
}

func TestRouterNotFound(t *testing.T) {
	app := &core.App{Container: core.NewContainer()}
	app.Router = core.NewRouter(app)
	app.Router.Get("/known", func(c *core.Context) { c.String(http.StatusOK, "ok") })

	req := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	rec := httptest.NewRecorder()
	app.Router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestValidatorRequired(t *testing.T) {
	v := core.NewValidator(map[string]string{"email": ""})
	v.Required("email").Email("email")
	if v.Passes() {
		t.Fatal("expected validation to fail for empty required field")
	}
}

func TestPasswordHashRoundTrip(t *testing.T) {
	hash, err := core.HashPassword("secret123")
	if err != nil {
		t.Fatal(err)
	}
	if !core.VerifyPassword("secret123", hash) {
		t.Fatal("expected password to verify")
	}
	if core.VerifyPassword("wrong", hash) {
		t.Fatal("expected wrong password to fail verification")
	}
}

func TestJWTRoundTrip(t *testing.T) {
	token, err := core.GenerateJWT(core.Claims{"sub": 1}, "test-secret", 60_000_000_000)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := core.ParseJWT(token, "test-secret")
	if err != nil {
		t.Fatal(err)
	}
	if claims["sub"].(float64) != 1 {
		t.Fatalf("expected sub=1, got %v", claims["sub"])
	}
}

func TestRegisterRoutesWithKernel(t *testing.T) {
	app := core.NewApp("../.env", "")
	kernel := middleware.NewKernel(app)
	config.RegisterRoutes(app, kernel)

	routes := app.Router.Routes()
	if len(routes) == 0 {
		t.Fatal("expected routes to be registered by the kernel")
	}
}

func TestAppGracefulShutdown(t *testing.T) {
	app := core.NewApp("../.env", "")
	
	// Dispatch a job into the queue
	var jobDone bool
	app.Queue.Dispatch(func() {
		jobDone = true
	})

	// Set a cache item
	app.Cache.Set("k", "v", time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := app.GracefulShutdown(nil, ctx); err != nil {
		t.Fatalf("expected clean graceful shutdown, got: %v", err)
	}

	if !jobDone {
		t.Error("expected background queue jobs to drain during graceful shutdown")
	}
}


