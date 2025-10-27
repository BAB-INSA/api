package models

import (
	"time"

	"gorm.io/gorm"
)

type RefreshToken struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	UserID    uint           `json:"user_id" gorm:"not null;index"`
	Token     string         `json:"token" gorm:"uniqueIndex;not null"`
	ExpiresAt time.Time      `json:"expires_at" gorm:"not null"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
	User      User           `json:"-" gorm:"foreignKey:UserID"`
}

// TableName spécifie le nom de la table
func (RefreshToken) TableName() string {
	return "refresh_tokens"
}

// IsExpired vérifie si le token est expiré
func (rt *RefreshToken) IsExpired() bool {
	return time.Now().After(rt.ExpiresAt)
}

// RefreshTokenRequest représente une requête de refresh
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// TokenResponse représente la réponse avec access et refresh tokens
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"` // secondes
	TokenType    string `json:"token_type"`
}
