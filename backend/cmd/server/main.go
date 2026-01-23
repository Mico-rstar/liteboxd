package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/fslongjin/liteboxd/internal/handler"
	"github.com/fslongjin/liteboxd/internal/k8s"
	"github.com/fslongjin/liteboxd/internal/service"
)

func main() {
	kubeconfigPath := os.Getenv("KUBECONFIG")
	if kubeconfigPath == "" {
		home := os.Getenv("HOME")
		kubeconfigPath = home + "/.kube/config"
	}

	k8sClient, err := k8s.NewClient(kubeconfigPath)
	if err != nil {
		log.Fatalf("Failed to create k8s client: %v", err)
	}

	ctx := context.Background()
	if err := k8sClient.EnsureNamespace(ctx); err != nil {
		log.Fatalf("Failed to ensure namespace: %v", err)
	}
	fmt.Println("Namespace 'liteboxd' ensured")

	sandboxSvc := service.NewSandboxService(k8sClient)
	sandboxSvc.StartTTLCleaner(30 * time.Second)
	fmt.Println("TTL cleaner started (interval: 30s)")

	sandboxHandler := handler.NewSandboxHandler(sandboxSvc)

	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	api := r.Group("/api/v1")
	sandboxHandler.RegisterRoutes(api)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Server starting on port %s\n", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
