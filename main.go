package main

import (
	"log"

	"novaflow/app/middleware"
	"novaflow/config"
	"novaflow/core"
)

func main() {
	app := core.NewApp(".env", "app/views")

	// Instantiate central middleware kernel
	kernel := middleware.NewKernel(app)

	// Register routes with kernel awareness
	config.RegisterRoutes(app, kernel)

	addr := ":" + app.Config.Get("APP_PORT", "8080")

	// Start server passing the global middleware chain defined in the kernel
	if err := app.Run(addr, kernel.Global()...); err != nil {
		log.Fatal(err)
	}
}
