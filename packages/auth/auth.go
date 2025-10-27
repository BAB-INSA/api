package auth

import (
	"auth/handlers"
	"auth/middleware"
	"core/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Module struct {
	Handler *handlers.AuthHandler
}

func NewModule(db *gorm.DB) *Module {
	playerService := services.NewPlayerService(db)
	return &Module{
		Handler: handlers.NewAuthHandler(db, playerService),
	}
}

func (m *Module) SetupRoutes(r *gin.Engine) {
	auth := r.Group("/auth")
	{
		auth.POST("/register", m.Handler.Register)
		auth.POST("/login", m.Handler.Login)
		auth.POST("/refresh", m.Handler.RefreshToken)
		auth.POST("/logout", m.Handler.Logout)
		auth.POST("/logout-all", middleware.JWTMiddleware(), m.Handler.LogoutAll)
		auth.POST("/reset-password/send-link", m.Handler.SendPasswordResetLink)
		auth.POST("/reset-password/confirm", m.Handler.ConfirmPasswordReset)
		auth.POST("/change-password", middleware.JWTMiddleware(), m.Handler.ChangePassword)
	}
}

func JWTMiddleware() gin.HandlerFunc {
	return middleware.JWTMiddleware()
}

func GetUserID(c *gin.Context) (uint, bool) {
	return middleware.GetUserID(c)
}

func GetUserEmail(c *gin.Context) (string, bool) {
	return middleware.GetUserEmail(c)
}

func RequireRole(db *gorm.DB, role string) gin.HandlerFunc {
	return middleware.RequireRole(db, role)
}

func RequireAnyRole(db *gorm.DB, roles ...string) gin.HandlerFunc {
	return middleware.RequireAnyRole(db, roles...)
}
