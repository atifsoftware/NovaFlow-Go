package core

import (
	"fmt"
	"net/http"
	"strings"
)

// HandlerFunc is the signature every route handler and controller action
// must satisfy.
type HandlerFunc func(*Context)

// Middleware wraps a HandlerFunc to run logic before/after it (auth,
// logging, rate limiting, CORS, etc).
type Middleware func(HandlerFunc) HandlerFunc

type route struct {
	method     string
	segments   []string
	handler    HandlerFunc
	middleware []Middleware
}

// Router is a small, dependency-free HTTP router supporting static and
// dynamic (":param") path segments, wildcard-free groups with shared
// prefix + middleware, and per-route middleware.
type Router struct {
	app         *App
	routes      []*route
	groupPrefix string
	groupMw     []Middleware
	NotFound    HandlerFunc
	MethodNotAllowed HandlerFunc
}

func NewRouter(app *App) *Router {
	return &Router{
		app: app,
		// BUG-08: Smart error handlers — JSON for API clients, HTML for browsers
		NotFound: func(c *Context) {
			if isJSONRequest(c.Request) {
				c.JSONError(http.StatusNotFound, "route not found: "+c.Request.URL.Path)
			} else {
				c.HTML(http.StatusNotFound, notFoundHTML(c.Request.URL.Path))
			}
		},
		MethodNotAllowed: func(c *Context) {
			if isJSONRequest(c.Request) {
				c.JSONError(http.StatusMethodNotAllowed, "method not allowed")
			} else {
				c.HTML(http.StatusMethodNotAllowed, "<h1>405 — Method Not Allowed</h1>")
			}
		},
	}
}

func (r *Router) add(method, path string, handler HandlerFunc, mw ...[]Middleware) {
	full := r.groupPrefix + path
	segs := splitPath(full)
	all := append([]Middleware{}, r.groupMw...)
	if len(mw) > 0 {
		all = append(all, mw[0]...)
	}
	r.routes = append(r.routes, &route{method: method, segments: segs, handler: handler, middleware: all})
}

func (r *Router) Get(path string, h HandlerFunc)    { r.add(http.MethodGet, path, h) }
func (r *Router) Post(path string, h HandlerFunc)   { r.add(http.MethodPost, path, h) }
func (r *Router) Put(path string, h HandlerFunc)    { r.add(http.MethodPut, path, h) }
func (r *Router) Patch(path string, h HandlerFunc)  { r.add(http.MethodPatch, path, h) }
func (r *Router) Delete(path string, h HandlerFunc) { r.add(http.MethodDelete, path, h) }

// Resource registers the conventional REST 7 (index/show/store/update/destroy)
// routes for a controller in one call, e.g. r.Resource("/products", ctrl).
type ResourceController interface {
	Index(*Context)
	Show(*Context)
	Store(*Context)
	Update(*Context)
	Destroy(*Context)
}

// RouteInfo is a read-only summary of a registered route, used by the CLI's
// `--routes` command.
type RouteInfo struct {
	Method string
	Path   string
}

// Routes lists every registered route (method + path), for `cli.php --routes`-
// style introspection.
func (r *Router) Routes() []RouteInfo {
	out := make([]RouteInfo, 0, len(r.routes))
	for _, rt := range r.routes {
		out = append(out, RouteInfo{Method: rt.method, Path: "/" + strings.Join(rt.segments, "/")})
	}
	return out
}

func (r *Router) Resource(path string, ctrl ResourceController) {
	r.Get(path, ctrl.Index)
	r.Get(path+"/:id", ctrl.Show)
	r.Post(path, ctrl.Store)
	r.Put(path+"/:id", ctrl.Update)
	r.Delete(path+"/:id", ctrl.Destroy)
}

// GroupOptions configures a route group.
type GroupOptions struct {
	Prefix     string
	Middleware []Middleware
}

// Group registers a set of routes sharing a prefix and middleware stack,
// mirroring $router->group(['prefix'=>..,'middleware'=>..], function(){}).
func (r *Router) Group(opts GroupOptions, fn func(r *Router)) {
	savedPrefix, savedMw := r.groupPrefix, r.groupMw
	r.groupPrefix = savedPrefix + opts.Prefix
	r.groupMw = append(append([]Middleware{}, savedMw...), opts.Middleware...)
	fn(r)
	r.groupPrefix, r.groupMw = savedPrefix, savedMw
}

func splitPath(p string) []string {
	p = strings.Trim(p, "/")
	if p == "" {
		return []string{}
	}
	return strings.Split(p, "/")
}

func (r *Router) match(method, path string) (*route, map[string]string, bool, bool) {
	reqSegs := splitPath(path)
	methodExistsForPath := false
	for _, rt := range r.routes {
		if len(rt.segments) != len(reqSegs) {
			continue
		}
		params := map[string]string{}
		matched := true
		for i, seg := range rt.segments {
			if strings.HasPrefix(seg, ":") {
				params[seg[1:]] = reqSegs[i]
				continue
			}
			if seg != reqSegs[i] {
				matched = false
				break
			}
		}
		if !matched {
			continue
		}
		methodExistsForPath = true
		if rt.method == method {
			return rt, params, true, true
		}
	}
	return nil, nil, false, methodExistsForPath
}

// ServeHTTP implements http.Handler, so *Router can be passed straight to
// http.ListenAndServe or wrapped by additional global middleware in App.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	ctx := newContext(w, req, r.app)
	rt, params, found, pathExists := r.match(req.Method, req.URL.Path)
	if !found {
		if pathExists {
			r.MethodNotAllowed(ctx)
		} else {
			r.NotFound(ctx)
		}
		return
	}
	ctx.params = params

	final := rt.handler
	for i := len(rt.middleware) - 1; i >= 0; i-- {
		final = rt.middleware[i](final)
	}
	final(ctx)
}

// isJSONRequest returns true if the client prefers a JSON response
// (API calls set Accept: application/json or use /api/ prefix).
func isJSONRequest(r *http.Request) bool {
	return strings.Contains(r.Header.Get("Accept"), "application/json") ||
		strings.HasPrefix(r.URL.Path, "/api/")
}

// notFoundHTML returns a minimal styled 404 HTML page for browser visitors.
func notFoundHTML(path string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>404 — Page Not Found</title>
<style>
  body{background:#0b0f19;color:#f3f4f6;font-family:sans-serif;display:flex;align-items:center;justify-content:center;height:100vh;margin:0}
  .box{text-align:center;padding:40px}
  h1{font-size:80px;margin:0;color:#ef4444}
  p{color:#9ca3af;font-size:18px}
  a{color:#60a5fa;text-decoration:none}
</style>
</head>
<body>
  <div class="box">
    <h1>404</h1>
    <p>The page <code>%s</code> could not be found.</p>
    <a href="/">← Back to Home</a>
  </div>
</body>
</html>`, path)
}
