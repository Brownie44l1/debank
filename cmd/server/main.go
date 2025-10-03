package main

import (
	"context"
	"log"
    "net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/Brownie44l1/debank/internal/config"
	"github.com/Brownie44l1/debank/internal/db"
	"github.com/Brownie44l1/debank/internal/handlers"
	"github.com/Brownie44l1/debank/internal/repository"
	"github.com/Brownie44l1/debank/internal/services"
)

func main() {
	// 1. Load configuration
	cfg := config.LoadConfig()
	log.Println("✓ Configuration loaded")

	// 2. Initialize database connection
	ctx := context.Background()
	pool, err := db.NewPool(ctx, cfg.DBUrl)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer pool.Close()

	// 3. Initialize layers
	walletRepo := repository.NewWalletRepository(pool)
	walletService := services.NewWalletService(walletRepo)
	walletHandler := handlers.NewWalletHandler(walletService)

	// 4. Setup Gin router
	router := gin.Default()

	// Health check endpoint (you already have this)
	//router.GET("/health", handlers.HealthCheck)

	// Register wallet routes
	walletHandler.RegisterRoutes(router)

	// 5. Start server with graceful shutdown
	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		log.Println("🚀 Server starting on :8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Failed to start server:", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("🛑 Shutting down server...")

	// Graceful shutdown with 5 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("✓ Server exited")
}