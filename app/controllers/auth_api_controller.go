package controllers

import (
	"net/http"
	"strings"

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

type authLoginPayload struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// POST /api/v1/login
func (a *AuthApiController) Login(c *core.Context) {
	var payload authLoginPayload

	if strings.Contains(c.Request.Header.Get("Content-Type"), "application/json") {
		if err := c.BindJSON(&payload); err != nil {
			c.JSONError(http.StatusBadRequest, "invalid JSON payload")
			return
		}
	} else {
		payload.Email = c.Input("email")
		payload.Password = c.Input("password")
	}

	v := core.NewValidator(map[string]string{
		"email":    payload.Email,
		"password": payload.Password,
	})
	v.Required("email").Email("email").Required("password")
	if !v.Passes() {
		c.JSONError(http.StatusUnprocessableEntity, v.FirstError())
		return
	}

	result, err := a.auth.LoginAPI(payload.Email, payload.Password)
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

type authRegisterPayload struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// POST /api/v1/register
func (a *AuthApiController) Register(c *core.Context) {
	var payload authRegisterPayload

	if strings.Contains(c.Request.Header.Get("Content-Type"), "application/json") {
		if err := c.BindJSON(&payload); err != nil {
			c.JSONError(http.StatusBadRequest, "invalid JSON payload")
			return
		}
	} else {
		payload.Name = c.Input("name")
		payload.Email = c.Input("email")
		payload.Password = c.Input("password")
	}

	v := core.NewValidator(map[string]string{
		"name":     payload.Name,
		"email":    payload.Email,
		"password": payload.Password,
	})
	v.Required("name").Required("email").Email("email").Required("password").MinLen("password", 6)
	if !v.Passes() {
		c.JSONError(http.StatusUnprocessableEntity, v.FirstError())
		return
	}

	id, err := a.auth.Register(payload.Name, payload.Email, payload.Password)
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

