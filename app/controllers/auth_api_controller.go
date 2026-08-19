package controllers

import (
	"net/http"

	"novaflow/core"
)

// AuthApiController exposes JSON login/register endpoints that return a
// JWT for use as "Authorization: Bearer <token>" on protected api/v1 routes.
type AuthApiController struct {
	auth *core.AuthService
}

func NewAuthApiController(auth *core.AuthService) *AuthApiController {
	return &AuthApiController{auth: auth}
}

// POST /api/v1/login
func (a *AuthApiController) Login(c *core.Context) {
	v := core.NewValidator(map[string]string{
		"email":    c.Input("email"),
		"password": c.Input("password"),
	})
	v.Required("email").Email("email").Required("password")
	if !v.Passes() {
		c.JSONError(http.StatusUnprocessableEntity, v.FirstError())
		return
	}

	result, err := a.auth.LoginAPI(c.Input("email"), c.Input("password"))
	if err != nil {
		c.JSONError(http.StatusUnauthorized, "invalid email or password")
		return
	}
	c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"token":   result.Token,
		"user": map[string]interface{}{
			"id":    result.UserID,
			"name":  result.Name,
			"email": result.Email,
		},
	})
}

// POST /api/v1/register
func (a *AuthApiController) Register(c *core.Context) {
	v := core.NewValidator(map[string]string{
		"name":     c.Input("name"),
		"email":    c.Input("email"),
		"password": c.Input("password"),
	})
	v.Required("name").Required("email").Email("email").Required("password").MinLen("password", 6)
	if !v.Passes() {
		c.JSONError(http.StatusUnprocessableEntity, v.FirstError())
		return
	}

	id, err := a.auth.Register(c.Input("name"), c.Input("email"), c.Input("password"))
	if err != nil {
		c.JSONError(http.StatusInternalServerError, "could not create account: "+err.Error())
		return
	}
	c.JSON(http.StatusCreated, map[string]interface{}{"success": true, "user_id": id})
}

// GET /api/v1/me  (protected by core.AuthAPI middleware)
func (a *AuthApiController) Me(c *core.Context) {
	claims, _ := c.Get("claims").(core.Claims)
	c.JSONSuccess(claims)
}
