package core

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRecoverHtmlLocal(t *testing.T) {
	app := &App{
		Config:    Env(),
		Container: NewContainer(),
	}
	app.Config.Set("APP_ENV", "local")
	app.Router = NewRouter(app)

	app.Router.Get("/panic", func(c *Context) {
		panic("Testing local HTML error view")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		final := func(c *Context) { app.Router.ServeHTTP(c.Writer, c.Request) }
		Recover()(final)(newContext(w, r, app))
	})

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "NovaFlow Error Recovery") {
		t.Error("expected body to contain debug screen title")
	}
	if !strings.Contains(body, "Testing local HTML error view") {
		t.Error("expected body to contain panic message")
	}
}

func TestRecoverApiLocal(t *testing.T) {
	app := &App{
		Config:    Env(),
		Container: NewContainer(),
	}
	app.Config.Set("APP_ENV", "local")
	app.Router = NewRouter(app)

	app.Router.Get("/api/panic", func(c *Context) {
		panic("API Panic!")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/panic", nil)
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		final := func(c *Context) { app.Router.ServeHTTP(c.Writer, c.Request) }
		Recover()(final)(newContext(w, r, app))
	})

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}

	var resp map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	if err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}

	if resp["success"].(bool) != false {
		t.Error("expected success field to be false")
	}
	if resp["message"].(string) != "API Panic!" {
		t.Errorf("unexpected message: %s", resp["message"])
	}
	if resp["file"].(string) == "" || resp["line"].(float64) <= 0 {
		t.Errorf("missing or invalid debug trace in JSON response: %v", resp)
	}
}

func TestRecoverProd(t *testing.T) {
	app := &App{
		Config:    Env(),
		Container: NewContainer(),
	}
	app.Config.Set("APP_ENV", "production")
	app.Router = NewRouter(app)

	app.Router.Get("/panic", func(c *Context) {
		panic("Prod Panic!")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		final := func(c *Context) { app.Router.ServeHTTP(c.Writer, c.Request) }
		Recover()(final)(newContext(w, r, app))
	})

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}

	body := rec.Body.String()
	if body != "internal server error" {
		t.Errorf("expected generic message 'internal server error', got: %q", body)
	}
}
