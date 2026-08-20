package main

import (
	"log"

	"ehd-api/config"
	"ehd-api/internal/app"
)

func main() {
	cfg, err := config.New()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	if err := app.Run(cfg); err != nil {
		log.Fatalf("app: %v", err)
	}
}
