package main

import (
	"fmt"
	"log"

	"go-entegra/internal/config"
	"go-entegra/internal/database"
	"go-entegra/internal/middleware"
	"go-entegra/internal/routes"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	if err := database.Migrate(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	middleware.SetJWTSecret(cfg.JWT.Secret)
	middleware.SetDB(db)

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	routes.Setup(r, db, cfg.JWT.ExpireHours)

	addr := fmt.Sprintf(":%s", cfg.App.Port)
	log.Printf("Server starting on %s (env: %s)", addr, cfg.App.Env)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
