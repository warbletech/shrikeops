package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/warbletech/kestrelops/scanner/internal/engine"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8090"
	}

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"service": "kestrelops-scanner", "status": "healthy", "version": "0.1.0"})
	})

	r.POST("/scan", engine.HandleScan)

	log.Printf("[KestrelOps] Scanner listening on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start: %v", err)
	}
}
