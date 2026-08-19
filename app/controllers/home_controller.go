package controllers

import "novaflow/core"

type HomeController struct{}

func NewHomeController() *HomeController { return &HomeController{} }

// Index renders app/views/home/index.html inside the shared layout.
func (h *HomeController) Index(c *core.Context) {
	_ = c.View("home/index", map[string]interface{}{
		"Title": "NovaFlow",
	})
}

// Health is a tiny JSON endpoint useful for uptime checks / load balancers.
func (h *HomeController) Health(c *core.Context) {
	c.JSON(200, map[string]interface{}{
		"status":  "ok",
		"service": "novaflow",
	})
}
