package controllers

import (
	"net/http"

	"novaflow/core"
)

// AuthController handles browser-based (session cookie) login/logout.
type AuthController struct {
	auth *core.AuthService
}

func NewAuthController(auth *core.AuthService) *AuthController {
	return &AuthController{auth: auth}
}

// GET /login
func (a *AuthController) ShowLogin(c *core.Context) {
	_ = c.View("auth/login", map[string]interface{}{"Title": "Login"})
}

// POST /login
func (a *AuthController) Login(c *core.Context) {
	_, err := a.auth.Login(c.Writer, c.Input("email"), c.Input("password"))
	if err != nil {
		_ = c.View("auth/login", map[string]interface{}{
			"Title": "Login",
			"Error": "Invalid email or password",
		})
		return
	}
	c.Redirect("/dashboard", http.StatusFound)
}

// POST /logout
func (a *AuthController) Logout(c *core.Context) {
	a.auth.Logout(c.Writer)
	c.Redirect("/login", http.StatusFound)
}

// GET /dashboard  (protected by core.Auth middleware)
func (a *AuthController) Dashboard(c *core.Context) {
	_ = c.View("auth/dashboard", map[string]interface{}{
		"Title":  "Dashboard",
		"UserID": c.Get("user_id"),
	})
}
