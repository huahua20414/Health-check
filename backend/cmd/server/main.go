package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"health-checkup/backend/internal/cache"
	"health-checkup/backend/internal/config"
	"health-checkup/backend/internal/database"
	"health-checkup/backend/internal/handlers"
	"health-checkup/backend/internal/seed"
)

func main() {
	cfg := config.Load()
	db, err := database.Open(cfg.DBDSN)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	if err := database.Migrate(db); err != nil {
		log.Fatalf("migrate database: %v", err)
	}

	command := "serve"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	switch command {
	case "serve":
		redisClient := cache.Open(cfg.RedisAddr)
		if err := cache.Ping(context.Background(), redisClient); err != nil {
			log.Fatalf("connect redis: %v", err)
		}
		router := handlers.NewRouter(db, redisClient, cfg)
		if err := router.Run(cfg.Addr); err != nil {
			log.Fatalf("start server: %v", err)
		}
	case "seed":
		if err := seed.Run(db); err != nil {
			log.Fatalf("seed database: %v", err)
		}
		fmt.Println("seed data inserted")
	default:
		log.Fatalf("unknown command %q", command)
	}
}
