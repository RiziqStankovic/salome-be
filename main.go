package main

import (
	"fmt"
	"log"

	"salome-be/internal/config"
	"salome-be/internal/database"
	"salome-be/internal/handlers"
	"salome-be/internal/middleware"
	"salome-be/internal/routes"

	"github.com/gin-gonic/gin"
)

func main() {
	// Load YAML configuration
	if err := config.LoadConfig(); err != nil {
		log.Printf("Warning: Failed to load YAML config: %v", err)
		log.Println("Using default configuration...")
	}

	// Initialize database
	db, err := database.InitDB()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Run migrations
	if err := database.RunMigrations(db); err != nil {
		log.Fatal("Failed to run migrations:", err)
	}

	// Initialize Gin router
	r := gin.Default()

	// CORS middleware
	r.Use(middleware.CORS())

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(db)
	groupHandler := handlers.NewGroupHandler(db)
	subscriptionHandler := handlers.NewSubscriptionHandler(db)
	paymentHandler := handlers.NewPaymentHandler(db)
	appHandler := handlers.NewAppHandler(db)
	messageHandler := handlers.NewMessageHandler(db)
	transactionHandler := handlers.NewTransactionHandler(db)
	otpHandler := handlers.NewOTPHandler(db)
	accountCredentialsHandler := handlers.NewAccountCredentialsHandler(db)
	emailSubmissionHandler := handlers.NewEmailSubmissionHandler(db)
	adminHandler := handlers.NewAdminHandler(db)

	// Setup routes
	routes.SetupRoutes(r, authHandler, groupHandler, subscriptionHandler, paymentHandler, appHandler, messageHandler, transactionHandler, otpHandler, accountCredentialsHandler, emailSubmissionHandler, adminHandler, db)

	// Start server
	appConfig := config.GetConfig()
	port := fmt.Sprintf("%d", appConfig.Server.Port)
	host := appConfig.Server.Host

	log.Printf("Server starting on %s:%s", host, port)
	if err := r.Run(host + ":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
