package controllers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"novaflow/core"
)

// --- HomeController Tests ---

func TestHomeControllerHealth(t *testing.T) {
	app := core.NewApp("../../.env", "")
	router := core.NewRouter(app)

	ctrl := NewHomeController()
	router.Get("/health", ctrl.Health)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal("could not decode JSON response:", err)
	}

	if resp["status"] != "ok" {
		t.Errorf("expected status 'ok', got %q", resp["status"])
	}
	if resp["service"] != "novaflow" {
		t.Errorf("expected service 'novaflow', got %q", resp["service"])
	}
}

// --- DocsController Tests ---

func TestDocsControllerServeOpenAPI(t *testing.T) {
	app := core.NewApp("../../.env", "")
	router := core.NewRouter(app)

	ctrl := NewDocsController()
	router.Get("/openapi.json", ctrl.OpenAPISpec)

	req := httptest.NewRequest("GET", "/openapi.json", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got %q", contentType)
	}

	var spec map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&spec); err != nil {
		t.Fatal("could not decode OpenAPI JSON:", err)
	}

	if spec["openapi"] == nil {
		t.Error("expected 'openapi' key in spec")
	}
	if spec["info"] == nil {
		t.Error("expected 'info' key in spec")
	}
	if spec["paths"] == nil {
		t.Error("expected 'paths' key in spec")
	}
}

func TestDocsControllerServeSwaggerUI(t *testing.T) {
	app := core.NewApp("../../.env", "")
	router := core.NewRouter(app)

	ctrl := NewDocsController()
	router.Get("/docs", ctrl.ShowDocs)

	req := httptest.NewRequest("GET", "/docs", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if len(body) == 0 {
		t.Error("expected non-empty HTML response for Swagger UI")
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "text/html; charset=utf-8" {
		t.Errorf("expected Content-Type 'text/html; charset=utf-8', got %q", contentType)
	}
}

// --- AuthApiController Validation Tests ---

func TestAuthApiControllerLoginValidation(t *testing.T) {
	app := core.NewApp("../../.env", "")
	if app.Auth == nil {
		t.Skip("database not configured, skipping auth controller test")
	}

	router := core.NewRouter(app)
	ctrl := NewAuthApiController(app.Auth)
	router.Post("/api/v1/login", ctrl.Login)

	// Test with empty body — should fail validation
	req := httptest.NewRequest("POST", "/api/v1/login", nil)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal("could not decode JSON response:", err)
	}

	if resp["success"] != false {
		t.Error("expected success=false for missing credentials")
	}
}

func TestAuthApiControllerRegisterValidation(t *testing.T) {
	app := core.NewApp("../../.env", "")
	if app.Auth == nil {
		t.Skip("database not configured, skipping auth controller test")
	}

	router := core.NewRouter(app)
	ctrl := NewAuthApiController(app.Auth)
	router.Post("/api/v1/register", ctrl.Register)

	// Test with empty body — should fail validation
	req := httptest.NewRequest("POST", "/api/v1/register", nil)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal("could not decode JSON response:", err)
	}

	if resp["success"] != false {
		t.Error("expected success=false for missing registration data")
	}
}

func TestAuthApiControllerJSONBodyValidation(t *testing.T) {
	app := core.NewApp("../../.env", "")
	if app.Auth == nil {
		t.Skip("database not configured, skipping auth controller test")
	}

	router := core.NewRouter(app)
	ctrl := NewAuthApiController(app.Auth)
	router.Post("/api/v1/login", ctrl.Login)

	// Test with JSON body missing password
	req := httptest.NewRequest("POST", "/api/v1/login", strings.NewReader(`{"email":"test@example.com","password":""}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422 for missing password, got %d", w.Code)
	}
}

