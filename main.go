package main

import (
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"auth"
	"bab-insa-api/config"
	_ "bab-insa-api/docs" // Swagger docs
	"core"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title           BAB-INSA API
// @version         1.0
// @description     API pour l'association de baby foot BAB-INSA avec JWT
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  MIT
// @license.url   http://opensource.org/licenses/MIT

// @host      localhost:8080
// @BasePath  /

// @securityDefinitions.apikey  BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	config.ConnectDatabase()

	r := gin.Default()

	// Get CORS allowed origins from environment
	corsOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
	var allowedOrigins []string
	if corsOrigins != "" {
		allowedOrigins = strings.Split(corsOrigins, ",")
		// Trim whitespace from each origin
		for i, origin := range allowedOrigins {
			allowedOrigins[i] = strings.TrimSpace(origin)
		}
	} else {
		// Default origins for development
		allowedOrigins = []string{"http://127.0.0.1:5173", "http://localhost:5173"}
	}

	// CORS middleware
	r.Use(cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	// Setup auth module (includes all refresh token routes)
	authModule := auth.NewModule(config.DB)
	authModule.SetupRoutes(r)

	// Setup core module (players, matches, etc.)
	coreModule := core.NewModule(config.DB)
	coreModule.SetupRoutes(r)

	// Start the scheduler for auto-validation
	if err := coreModule.StartScheduler(); err != nil {
		log.Fatalf("Failed to start scheduler: %v", err)
	}

	// Setup graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	
	go func() {
		<-c
		log.Println("Shutting down gracefully...")
		coreModule.StopScheduler()
		os.Exit(0)
	}()

	// Users routes (protected)
	users := r.Group("/users")
	users.Use(auth.JWTMiddleware())
	{
		users.GET("/me", authModule.Handler.Profile)
		users.PUT("/:id", authModule.Handler.UpdateUser)
	}

	// Swagger endpoint
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	r.GET("/health", healthHandler)

	protected := r.Group("/protected")
	protected.Use(auth.JWTMiddleware())
	{
		protected.GET("/test", protectedTestHandler)
	}

	// Add admin endpoint to manually trigger auto-validation (for testing)
	admin := r.Group("/admin")
	admin.Use(auth.JWTMiddleware())
	{
		admin.POST("/auto-validate", func(c *gin.Context) {
			coreModule.RunAutoValidationNow()
			c.JSON(200, gin.H{"message": "Auto-validation triggered manually"})
		})
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	r.Run(":" + port)
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Message  string `json:"message" example:"Server is running"`
	Database string `json:"database" example:"connected"`
}

// @Summary Health Check
// @Description Check if the server is running and database is connected
// @Tags health
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /health [get]
func healthHandler(c *gin.Context) {
	c.JSON(200, HealthResponse{
		Message:  "Server is running",
		Database: "connected",
	})
}

// ProtectedResponse represents the protected endpoint response
type ProtectedResponse struct {
	Message string `json:"message" example:"Protected route accessed"`
	UserID  uint   `json:"user_id" example:"1"`
	Email   string `json:"email" example:"user@example.com"`
}

// @Summary Protected Test Endpoint
// @Description Test endpoint that requires JWT authentication
// @Tags protected
// @Security BearerAuth
// @Produce json
// @Success 200 {object} ProtectedResponse
// @Failure 401 {object} map[string]string
// @Router /protected/test [get]
func protectedTestHandler(c *gin.Context) {
	userID, _ := auth.GetUserID(c)
	email, _ := auth.GetUserEmail(c)
	c.JSON(200, ProtectedResponse{
		Message: "Protected route accessed",
		UserID:  userID,
		Email:   email,
	})
}
