package utils

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"auth/models"

	"gorm.io/gorm"
)

const (
	AccessTokenExpiry  = 15 * time.Minute   // Access token court
	RefreshTokenExpiry = 7 * 24 * time.Hour // Refresh token longue durée
)

// GenerateTokenPair génère un access token et un refresh token
func GenerateTokenPair(db *gorm.DB, user models.User) (*models.TokenResponse, error) {
	// Générer l'access token (courte durée)
	accessToken, err := GenerateToken(user)
	if err != nil {
		return nil, err
	}

	// Générer le refresh token (longue durée, sécurisé)
	refreshTokenString, err := generateSecureToken()
	if err != nil {
		return nil, err
	}

	// Révoquer les anciens refresh tokens de l'utilisateur
	db.Where("user_id = ?", user.ID).Delete(&models.RefreshToken{})

	// Créer le nouveau refresh token en base
	refreshToken := models.RefreshToken{
		UserID:    user.ID,
		Token:     refreshTokenString,
		ExpiresAt: time.Now().Add(RefreshTokenExpiry),
	}

	if err := db.Create(&refreshToken).Error; err != nil {
		return nil, err
	}

	return &models.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshTokenString,
		ExpiresIn:    int64(AccessTokenExpiry.Seconds()),
		TokenType:    "Bearer",
	}, nil
}

// RefreshAccessToken génère un nouvel access token à partir d'un refresh token
func RefreshAccessToken(db *gorm.DB, refreshTokenString string) (*models.TokenResponse, error) {
	var refreshToken models.RefreshToken

	// Trouver le refresh token avec l'utilisateur
	if err := db.Preload("User").Where("token = ?", refreshTokenString).First(&refreshToken).Error; err != nil {
		return nil, err
	}

	// Vérifier si le token est expiré
	if refreshToken.IsExpired() {
		// Supprimer le token expiré
		db.Delete(&refreshToken)
		return nil, gorm.ErrRecordNotFound
	}

	// Générer un nouvel access token
	accessToken, err := GenerateToken(refreshToken.User)
	if err != nil {
		return nil, err
	}

	// Optionnel : rotation du refresh token (recommandé pour la sécurité)
	newRefreshTokenString, err := generateSecureToken()
	if err != nil {
		return nil, err
	}

	// Mettre à jour le refresh token
	refreshToken.Token = newRefreshTokenString
	refreshToken.ExpiresAt = time.Now().Add(RefreshTokenExpiry)
	db.Save(&refreshToken)

	return &models.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshTokenString,
		ExpiresIn:    int64(AccessTokenExpiry.Seconds()),
		TokenType:    "Bearer",
	}, nil
}

// RevokeRefreshToken révoque un refresh token
func RevokeRefreshToken(db *gorm.DB, refreshTokenString string) error {
	return db.Where("token = ?", refreshTokenString).Delete(&models.RefreshToken{}).Error
}

// RevokeAllUserTokens révoque tous les refresh tokens d'un utilisateur
func RevokeAllUserTokens(db *gorm.DB, userID uint) error {
	return db.Where("user_id = ?", userID).Delete(&models.RefreshToken{}).Error
}

// CleanExpiredTokens supprime les tokens expirés (à appeler périodiquement)
func CleanExpiredTokens(db *gorm.DB) error {
	return db.Where("expires_at < ?", time.Now()).Delete(&models.RefreshToken{}).Error
}

// generateSecureToken génère un token sécurisé pour le refresh token
func generateSecureToken() (string, error) {
	bytes := make([]byte, 32) // 256 bits
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
