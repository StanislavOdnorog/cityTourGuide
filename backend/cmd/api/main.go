package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/saas/city-stories-guide/backend/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("Config loaded: %s", cfg.LogSafe())

	gin.SetMode(cfg.Server.Mode)
	r := gin.Default()

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	log.Printf("Starting server on :%s", cfg.Server.Port)
	if err := r.Run(":" + cfg.Server.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
