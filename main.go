package main

import (
	"log"

	"novaflow/config"
	"novaflow/core"
)

func main() {
	app := core.NewApp(".env", "app/views")
	config.RegisterRoutes(app)

	addr := ":" + app.Config.Get("APP_PORT", "8080")
	if err := app.Run(addr); err != nil {
		log.Fatal(err)
	}
}
