package core

import (
	"encoding/json"
	"errors"
	"html/template"
	"net/http"
	"strconv"
	"sync"
)

var contextPool = sync.Pool{
	New: func() interface{} {
		return &Context{
			params: make(map[string]string),
			store:  make(map[string]interface{}),
		}
	},
}

// Context is passed to every handler and carries the request, response
// writer, route parameters, and a per-request key/value bag that
// middleware can use to pass data down the chain (e.g. the authenticated
// user set by the Auth middleware).
type Context struct {
	Writer  http.ResponseWriter
	Request *http.Request
	params  map[string]string
	store   map[string]interface{}
	app     *App
}

func newContext(w http.ResponseWriter, r *http.Request, app *App) *Context {
	return acquireContext(w, r, app)
}

func acquireContext(w http.ResponseWriter, r *http.Request, app *App) *Context {
	c := contextPool.Get().(*Context)
	c.Writer = w
	c.Request = r
	c.app = app
	for k := range c.params {
		delete(c.params, k)
	}
	for k := range c.store {
		delete(c.store, k)
	}
	return c
}

func releaseContext(c *Context) {
	c.Writer = nil
	c.Request = nil
	c.app = nil
	contextPool.Put(c)
}

// Param returns a route parameter, e.g. Param("id") for a route
// registered as "/users/:id".
func (c *Context) Param(name string) string {
	return c.params[name]
}

func (c *Context) ParamInt(name string) int {
	i, _ := strconv.Atoi(c.params[name])
	return i
}

// Query returns a URL query string parameter.
func (c *Context) Query(name string) string {
	return c.Request.URL.Query().Get(name)
}

// Input returns a form/query value from the request (works for both
// application/x-www-form-urlencoded POSTs and query strings).
func (c *Context) Input(name string) string {
	return c.Request.FormValue(name)
}

// BindJSON decodes the incoming JSON request body into the destination struct or map.
func (c *Context) BindJSON(dest interface{}) error {
	if c.Request.Body == nil {
		return errors.New("context: empty request body")
	}
	return json.NewDecoder(c.Request.Body).Decode(dest)
}

// Set / Get let middleware and handlers share request-scoped data.
func (c *Context) Set(key string, val interface{}) { c.store[key] = val }
func (c *Context) Get(key string) interface{}      { return c.store[key] }

// CSRFToken returns the CSRF token set by CSRF middleware for the request.
func (c *Context) CSRFToken() string {
	if token, ok := c.store["_csrf_token"].(string); ok {
		return token
	}
	return ""
}

// SetCSRFToken stores the active CSRF token in the request store.
func (c *Context) SetCSRFToken(token string) {
	c.store["_csrf_token"] = token
}

// App exposes the framework's App container (DB, config, container) to
// handlers that need it.
func (c *Context) App() *App { return c.app }

// Cache provides convenient access to the app's in-memory Cache.
func (c *Context) Cache() *Cache {
	if c.app != nil {
		return c.app.Cache
	}
	return nil
}

// Queue provides convenient access to the app's background job Queue.
func (c *Context) Queue() *Queue {
	if c.app != nil {
		return c.app.Queue
	}
	return nil
}

// --- Response helpers -----------------------------------------------------

// JSON writes a JSON response with the given status code.
func (c *Context) JSON(status int, payload interface{}) {
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(status)
	_ = json.NewEncoder(c.Writer).Encode(payload)
}

// JSONSuccess is a convenience wrapper matching the {success, data} shape
// used throughout the API controllers.
func (c *Context) JSONSuccess(data interface{}) {
	c.JSON(http.StatusOK, map[string]interface{}{"success": true, "data": data})
}

// JSONError writes a {success:false, message} response.
func (c *Context) JSONError(status int, message string) {
	c.JSON(status, map[string]interface{}{"success": false, "message": message})
}

// String writes a plain text response.
func (c *Context) String(status int, body string) {
	c.Writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	c.Writer.WriteHeader(status)
	_, _ = c.Writer.Write([]byte(body))
}

// HTML writes a raw HTML string response.
func (c *Context) HTML(status int, body string) {
	c.Writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	c.Writer.WriteHeader(status)
	_, _ = c.Writer.Write([]byte(body))
}

// Redirect sends an HTTP redirect.
func (c *Context) Redirect(url string, status int) {
	http.Redirect(c.Writer, c.Request, url, status)
}

// View renders a template from app/views using the shared *template.Template
// set loaded at startup, with the given data.
func (c *Context) View(name string, data map[string]interface{}) error {
	tpl, ok := c.app.Views[name]
	if !ok {
		c.String(http.StatusInternalServerError, "view not found: "+name)
		return nil
	}
	c.Writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	return tpl.ExecuteTemplate(c.Writer, "layout", data)
}

// RenderStandalone renders a raw template string (used mostly by tests/CLI).
func RenderStandalone(w http.ResponseWriter, tplText string, data interface{}) error {
	t, err := template.New("t").Parse(tplText)
	if err != nil {
		return err
	}
	return t.Execute(w, data)
}

