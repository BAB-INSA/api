package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"
)

type Roles []string

// Implémente l'interface driver.Valuer pour GORM
func (r Roles) Value() (driver.Value, error) {
	if len(r) == 0 {
		return json.Marshal([]string{"user"})
	}
	return json.Marshal(r)
}

// Implémente l'interface sql.Scanner pour GORM
func (r *Roles) Scan(value interface{}) error {
	if value == nil {
		*r = Roles{"user"}
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(bytes, &r)
}

type User struct {
	ID                  uint           `json:"id" gorm:"primaryKey"`
	Email               string         `json:"email" gorm:"uniqueIndex;not null"`
	Username            string         `json:"username" gorm:"uniqueIndex"`
	Password            string         `json:"-" gorm:"not null"`
	Slug                string         `json:"slug" gorm:"uniqueIndex"`
	Enabled             bool           `json:"enabled" gorm:"default:true"`
	Roles               Roles          `json:"roles" gorm:"type:jsonb;default:'[\"user\"]'::jsonb"`
	LastLogin           *time.Time     `json:"last_login"`
	NbConnexion         int            `json:"nb_connexion" gorm:"default:0"`
	ConfirmationToken   *string        `json:"-"`
	PasswordRequestedAt *time.Time     `json:"-"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
	DeletedAt           gorm.DeletedAt `json:"-" gorm:"index"`
}

// TableName spécifie le nom de la table au pluriel
func (User) TableName() string {
	return "users"
}

// HasRole vérifie si l'utilisateur a un rôle spécifique
func (u *User) HasRole(role string) bool {
	for _, r := range u.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// AddRole ajoute un rôle à l'utilisateur
func (u *User) AddRole(role string) {
	if !u.HasRole(role) {
		u.Roles = append(u.Roles, role)
	}
}

// RemoveRole supprime un rôle de l'utilisateur
func (u *User) RemoveRole(role string) {
	for i, r := range u.Roles {
		if r == role {
			u.Roles = append(u.Roles[:i], u.Roles[i+1:]...)
			return
		}
	}
}

// IsPasswordRequestExpired vérifie si la demande de reset de mot de passe a expiré
func (u *User) IsPasswordRequestExpired(ttlSeconds int) bool {
	if u.PasswordRequestedAt == nil {
		return true
	}
	return time.Since(*u.PasswordRequestedAt).Seconds() > float64(ttlSeconds)
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required,min=6"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type PasswordResetRequest struct {
	Email       string `json:"email" binding:"required,email"`
	CallBackUrl string `json:"callBackUrl" binding:"required"`
}

type PasswordResetResponse struct {
	Success bool `json:"success"`
}

type PasswordResetConfirmRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=6"`
}

type PasswordResetConfirmResponse struct {
	Success bool `json:"success"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword" binding:"required"`
	NewPassword     string `json:"newPassword" binding:"required,min=6"`
}

type ChangePasswordResponse struct {
	Success bool `json:"success"`
}

type UpdateUserRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Username string `json:"username" binding:"required"`
}

type PatchUserRequest struct {
	Email   *string `json:"email,omitempty" binding:"omitempty,email"`
	Roles   *Roles  `json:"roles,omitempty"`
	Enabled *bool   `json:"enabled,omitempty"`
}

type UpdateUserResponse struct {
	Success bool `json:"success"`
	User    User `json:"user"`
}
