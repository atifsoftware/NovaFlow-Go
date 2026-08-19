package core

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSecurityHeadersMiddleware(t *testing.T) {
	app := &App{Container: NewContainer()}
	router := NewRouter(app)
	router.Get("/sec-test", func(c *Context) {
		c.String(http.StatusOK, "ok")
	})

	handler := withGlobalMiddleware(router, SecurityHeaders())

	req := httptest.NewRequest("GET", "/sec-test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Errorf("expected X-Content-Type-Options: nosniff, got %q", w.Header().Get("X-Content-Type-Options"))
	}
	if w.Header().Get("X-Frame-Options") != "SAMEORIGIN" {
		t.Errorf("expected X-Frame-Options: SAMEORIGIN, got %q", w.Header().Get("X-Frame-Options"))
	}
	if w.Header().Get("X-XSS-Protection") != "1; mode=block" {
		t.Errorf("expected X-XSS-Protection: 1; mode=block, got %q", w.Header().Get("X-XSS-Protection"))
	}
}

func TestCSRFMiddlewareTokenIssuanceAndVerification(t *testing.T) {
	app := &App{Container: NewContainer()}
	router := NewRouter(app)
	router.Get("/form", func(c *Context) {
		c.String(http.StatusOK, c.CSRFToken())
	})
	router.Post("/form", func(c *Context) {
		c.String(http.StatusOK, "submitted")
	})

	csrfMw := CSRF()

	// 1. GET /form -> should issue CSRF cookie
	reqGet := httptest.NewRequest("GET", "/form", nil)
	wGet := httptest.NewRecorder()
	csrfMw(func(c *Context) {
		router.ServeHTTP(c.Writer, c.Request)
	})(newContext(wGet, reqGet, app))

	cookie := wGet.Result().Cookies()
	var csrfCookie *http.Cookie
	for _, ck := range cookie {
		if ck.Name == "csrf_token" {
			csrfCookie = ck
			break
		}
	}
	if csrfCookie == nil || csrfCookie.Value == "" {
		t.Fatal("expected CSRF cookie to be issued on GET request")
	}

	// 2. POST /form without token -> should fail with 403 Forbidden
	reqPostFail := httptest.NewRequest("POST", "/form", nil)
	reqPostFail.AddCookie(csrfCookie)
	wPostFail := httptest.NewRecorder()
	csrfMw(func(c *Context) {
		router.ServeHTTP(c.Writer, c.Request)
	})(newContext(wPostFail, reqPostFail, app))

	if wPostFail.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for missing CSRF token on POST, got %d", wPostFail.Code)
	}

	// 3. POST /form with valid header token -> should pass 200 OK
	reqPostSuccess := httptest.NewRequest("POST", "/form", strings.NewReader("_csrf_token="+csrfCookie.Value))
	reqPostSuccess.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqPostSuccess.AddCookie(csrfCookie)
	wPostSuccess := httptest.NewRecorder()
	csrfMw(func(c *Context) {
		router.ServeHTTP(c.Writer, c.Request)
	})(newContext(wPostSuccess, reqPostSuccess, app))

	if wPostSuccess.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for valid CSRF token on POST, got %d", wPostSuccess.Code)
	}
}
