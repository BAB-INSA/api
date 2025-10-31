package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"auth/models"
	"auth/services"
	"auth/utils"
	coreServices "core/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AuthHandler struct {
	DB            *gorm.DB
	EmailService  services.EmailService
	PlayerService *coreServices.PlayerService
}

func NewAuthHandler(db *gorm.DB, playerService *coreServices.PlayerService) *AuthHandler {
	return &AuthHandler{
		DB:            db,
		EmailService:  services.NewEmailService(), // Service email automatique (SMTP si configuré, sinon log)
		PlayerService: playerService,
	}
}

// @Summary User Registration
// @Description Register a new user and get JWT tokens
// @Tags auth
// @Accept json
// @Produce json
// @Param user body models.RegisterRequest true "User registration data"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var existingUser models.User
	if err := h.DB.Where("email = ? OR username = ?", req.Email, req.Username).First(&existingUser).Error; err == nil {
		if existingUser.Email == req.Email {
			c.JSON(http.StatusConflict, gin.H{"error": "Email already exists"})
		} else {
			c.JSON(http.StatusConflict, gin.H{"error": "Username already exists"})
		}
		return
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	now := time.Now()
	slug := strings.ToLower(strings.ReplaceAll(req.Username, " ", "-"))

	user := models.User{
		Email:       req.Email,
		Username:    req.Username,
		Slug:        slug,
		Password:    hashedPassword,
		Enabled:     true,
		LastLogin:   &now,
		NbConnexion: 1, // Auto-login compte comme première connexion
		Roles:       models.GetDefaultRoles(),
	}

	if err := h.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	// Create corresponding player
	_, err = h.PlayerService.CreatePlayer(user.ID, user.Username)
	if err != nil {
		// If player creation fails, we should rollback the user creation
		h.DB.Delete(&user)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create player profile"})
		return
	}

	tokenPair, err := utils.GenerateTokenPair(h.DB, user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate tokens"})
		return
	}

	response := gin.H{
		"access_token":  tokenPair.AccessToken,
		"refresh_token": tokenPair.RefreshToken,
		"expires_in":    tokenPair.ExpiresIn,
		"token_type":    tokenPair.TokenType,
		"user":          user,
	}

	c.JSON(http.StatusCreated, response)
}

// @Summary User Login
// @Description Login with email and password to get JWT tokens
// @Tags auth
// @Accept json
// @Produce json
// @Param credentials body models.LoginRequest true "User login credentials"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := h.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	if !utils.CheckPassword(req.Password, user.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Mettre à jour lastLogin et nbConnexion (seulement si différent jour)
	now := time.Now()
	shouldIncrementConnexion := true

	if user.LastLogin != nil {
		lastLoginDate := user.LastLogin.Format("2006-01-02")
		todayDate := now.Format("2006-01-02")
		shouldIncrementConnexion = lastLoginDate != todayDate
	}

	if shouldIncrementConnexion {
		user.NbConnexion++
	}
	user.LastLogin = &now

	if err := h.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user login info"})
		return
	}

	tokenPair, err := utils.GenerateTokenPair(h.DB, user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate tokens"})
		return
	}

	response := gin.H{
		"access_token":  tokenPair.AccessToken,
		"refresh_token": tokenPair.RefreshToken,
		"expires_in":    tokenPair.ExpiresIn,
		"token_type":    tokenPair.TokenType,
		"user":          user,
	}

	c.JSON(http.StatusOK, response)
}

// @Summary Get User Profile
// @Description Get current user profile information
// @Tags user
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]models.User
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /users/me [get]
func (h *AuthHandler) Profile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var user models.User
	if err := h.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// @Summary Refresh Access Token
// @Description Get a new access token using refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Param refresh body models.RefreshTokenRequest true "Refresh token"
// @Success 200 {object} models.TokenResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req models.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Récupérer l'utilisateur du refresh token pour mettre à jour lastLogin
	var refreshToken models.RefreshToken
	if err := h.DB.Preload("User").Where("token = ?", req.RefreshToken).First(&refreshToken).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		return
	}

	// Mettre à jour lastLogin et nbConnexion (seulement si différent jour)
	now := time.Now()
	user := refreshToken.User
	shouldIncrementConnexion := true

	if user.LastLogin != nil {
		lastLoginDate := user.LastLogin.Format("2006-01-02")
		todayDate := now.Format("2006-01-02")
		shouldIncrementConnexion = lastLoginDate != todayDate
	}

	if shouldIncrementConnexion {
		user.NbConnexion++
	}
	user.LastLogin = &now

	if err := h.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user login info"})
		return
	}

	tokenPair, err := utils.RefreshAccessToken(h.DB, req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		return
	}

	c.JSON(http.StatusOK, tokenPair)
}

// @Summary Logout
// @Description Logout and revoke refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Param refresh body models.RefreshTokenRequest true "Refresh token to revoke"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Router /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	var req models.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := utils.RevokeRefreshToken(h.DB, req.RefreshToken); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to revoke token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// @Summary Logout from All Devices
// @Description Revoke all refresh tokens for the current user
// @Tags auth
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /auth/logout-all [post]
func (h *AuthHandler) LogoutAll(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if err := utils.RevokeAllUserTokens(h.DB, userID.(uint)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke tokens"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Logged out from all devices"})
}

// generateConfirmationToken génère un token de confirmation sécurisé
func generateConfirmationToken() (string, error) {
	bytes := make([]byte, 32) // 256 bits
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// @Summary Send Password Reset Link
// @Description Send password reset link to user email
// @Tags auth
// @Accept json
// @Produce json
// @Param request body models.PasswordResetRequest true "Password reset request"
// @Success 200 {object} models.PasswordResetResponse
// @Failure 400 {object} map[string]string
// @Router /auth/reset-password/send-link [post]
func (h *AuthHandler) SendPasswordResetLink(c *gin.Context) {
	var req models.PasswordResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	const retryTTL = 7200 // 2 heures en secondes

	var user models.User
	if err := h.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		// Toujours renvoyer success pour éviter l'énumération des emails
		c.JSON(http.StatusOK, models.PasswordResetResponse{Success: true})
		return
	}

	// Vérifier si une demande récente existe et n'est pas expirée
	if user.ConfirmationToken != nil && !user.IsPasswordRequestExpired(retryTTL) {
		// Une demande récente existe déjà, ne pas en créer une nouvelle
		c.JSON(http.StatusOK, models.PasswordResetResponse{Success: true})
		return
	}

	// Générer un nouveau token de confirmation
	token, err := generateConfirmationToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate confirmation token"})
		return
	}

	// Mettre à jour l'utilisateur avec le nouveau token
	now := time.Now()
	user.ConfirmationToken = &token
	user.PasswordRequestedAt = &now

	if err := h.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save password reset request"})
		return
	}

	// Construire l'URL de reset avec le token
	origin := c.GetHeader("Origin")
	if origin == "" {
		origin = "http://localhost:3030" // URL par défaut pour le développement
	}
	resetURL := fmt.Sprintf("%s%s", origin, strings.ReplaceAll(req.CallBackUrl, "[token]", token))

	// Envoyer l'email de reset
	if err := h.EmailService.SendPasswordResetEmail(user.Email, resetURL); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send password reset email"})
		return
	}

	c.JSON(http.StatusOK, models.PasswordResetResponse{Success: true})
}

// @Summary Confirm Password Reset
// @Description Confirm password reset with token and new password
// @Tags auth
// @Accept json
// @Produce json
// @Param request body models.PasswordResetConfirmRequest true "Password reset confirmation"
// @Success 200 {object} models.PasswordResetConfirmResponse
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /auth/reset-password/confirm [post]
func (h *AuthHandler) ConfirmPasswordReset(c *gin.Context) {
	var req models.PasswordResetConfirmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	const resetTTL = 7200 // 2 heures en secondes

	var user models.User
	if err := h.DB.Where("confirmation_token = ?", req.Token).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Invalid or expired token"})
		return
	}

	// Vérifier si le token n'a pas expiré
	if user.IsPasswordRequestExpired(resetTTL) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token has expired"})
		return
	}

	// Hasher le nouveau mot de passe
	hashedPassword, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// Mettre à jour le mot de passe et supprimer le token
	user.Password = hashedPassword
	user.ConfirmationToken = nil
	user.PasswordRequestedAt = nil

	if err := h.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
		return
	}

	// Révoquer tous les refresh tokens existants pour forcer une nouvelle connexion
	if err := utils.RevokeAllUserTokens(h.DB, user.ID); err != nil {
		// Ne pas faire échouer la réponse si la révocation échoue
		log.Printf("Warning: Failed to revoke user tokens after password reset: %v", err)
	}

	c.JSON(http.StatusOK, models.PasswordResetConfirmResponse{Success: true})
}

// @Summary Change Password
// @Description Change password for authenticated user
// @Tags auth
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body models.ChangePasswordRequest true "Password change request"
// @Success 200 {object} models.ChangePasswordResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /auth/change-password [post]
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req models.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var user models.User
	if err := h.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	// Verify current password
	if !utils.CheckPassword(req.CurrentPassword, user.Password) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Current password is invalid"})
		return
	}

	// Hash the new password
	hashedPassword, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// Update the password
	user.Password = hashedPassword
	if err := h.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
		return
	}

	c.JSON(http.StatusOK, models.ChangePasswordResponse{Success: true})
}

// @Summary Update User
// @Description Update user email and username (only authenticated user can update their own profile)
// @Tags user
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path uint true "User ID"
// @Param request body models.UpdateUserRequest true "User update request"
// @Success 200 {object} models.UpdateUserResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /users/{id} [put]
func (h *AuthHandler) UpdateUser(c *gin.Context) {
	var req models.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Get the ID from URL parameter
	paramID := c.Param("id")
	if paramID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID is required"})
		return
	}

	// Check if user is trying to update their own profile
	if paramID != fmt.Sprintf("%d", userID.(uint)) {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only update your own profile"})
		return
	}

	var user models.User
	if err := h.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Check if email is already taken by another user
	if req.Email != user.Email {
		var existingUser models.User
		if err := h.DB.Where("email = ? AND id != ?", req.Email, user.ID).First(&existingUser).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "Email already exists"})
			return
		}
	}

	// Check if username is already taken by another user
	if req.Username != user.Username {
		var existingUser models.User
		if err := h.DB.Where("username = ? AND id != ?", req.Username, user.ID).First(&existingUser).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "Username already exists"})
			return
		}
	}

	// Update the user email and username
	user.Email = req.Email
	user.Username = req.Username
	// Update slug based on new username
	user.Slug = strings.ToLower(strings.ReplaceAll(req.Username, " ", "-"))
	if err := h.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// @Summary Patch User Roles and Status
// @Description Update user email, roles and enabled status (admin only)
// @Tags user
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path uint true "User ID"
// @Param request body models.PatchUserRequest true "User patch request"
// @Success 200 {object} models.User
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /users/{id} [patch]
func (h *AuthHandler) PatchUser(c *gin.Context) {
	var req models.PatchUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	currentUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Get current user to check permissions
	var currentUser models.User
	if err := h.DB.First(&currentUser, currentUserID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Current user not found"})
		return
	}

	// Check if user is admin or superAdmin
	isAdmin := currentUser.HasRole(models.RoleAdmin) || currentUser.HasRole(models.RoleSuperAdmin)
	if !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only admins can modify user data"})
		return
	}

	// Get the ID from URL parameter
	paramID := c.Param("id")
	if paramID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID is required"})
		return
	}

	// Find target user to update
	var targetUser models.User
	if err := h.DB.First(&targetUser, paramID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Update email if provided
	if req.Email != nil {
		// Check if email is already taken by another user
		if *req.Email != targetUser.Email {
			var existingUser models.User
			if err := h.DB.Where("email = ? AND id != ?", *req.Email, targetUser.ID).First(&existingUser).Error; err == nil {
				c.JSON(http.StatusConflict, gin.H{"error": "Email already exists"})
				return
			}
		}
		targetUser.Email = *req.Email
	}

	// Validate roles if provided
	if req.Roles != nil {
		for _, role := range *req.Roles {
			if !models.IsValidRole(role) {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid role: %s", role)})
				return
			}
		}
		targetUser.Roles = *req.Roles
	}

	// Update enabled status if provided
	if req.Enabled != nil {
		targetUser.Enabled = *req.Enabled
	}

	if err := h.DB.Save(&targetUser).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	c.JSON(http.StatusOK, targetUser)
}

// UserListResponse represents the paginated user list response
type UserListResponse struct {
	Users      []models.User `json:"users"`
	Total      int64         `json:"total"`
	Page       int           `json:"page"`
	PerPage    int           `json:"per_page"`
	TotalPages int           `json:"total_pages"`
}

// @Summary Get Users List
// @Description Get paginated list of users with optional search
// @Tags user
// @Security BearerAuth
// @Produce json
// @Param page query int false "Page number (default: 1)" default(1)
// @Param per_page query int false "Items per page (default: 10, max: 100)" default(10)
// @Param search query string false "Search in username or email"
// @Success 200 {object} UserListResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /users [get]
func (h *AuthHandler) GetUsers(c *gin.Context) {
	// Check if user is authenticated
	_, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Parse pagination parameters
	pageStr := c.DefaultQuery("page", "1")
	perPageStr := c.DefaultQuery("per_page", "10")
	search := c.Query("search")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page parameter"})
		return
	}

	perPage, err := strconv.Atoi(perPageStr)
	if err != nil || perPage < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid per_page parameter"})
		return
	}

	// Limit per_page to maximum 100
	if perPage > 100 {
		perPage = 100
	}

	// Calculate offset
	offset := (page - 1) * perPage

	// Build query with search
	query := h.DB.Model(&models.User{})
	if search != "" {
		searchPattern := "%" + search + "%"
		query = query.Where("username ILIKE ? OR email ILIKE ?", searchPattern, searchPattern)
	}

	// Get total count with search filter
	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count users"})
		return
	}

	// Get paginated users with search filter
	var users []models.User
	if err := query.Offset(offset).Limit(perPage).Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve users"})
		return
	}

	// Calculate total pages
	totalPages := int((total + int64(perPage) - 1) / int64(perPage))

	response := UserListResponse{
		Users:      users,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
	}

	c.JSON(http.StatusOK, response)
}
