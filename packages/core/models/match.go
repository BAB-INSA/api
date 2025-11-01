package models

import (
	"time"

	"gorm.io/gorm"
)

type Match struct {
	ID          uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	Player1ID   uint           `gorm:"not null;constraint:OnDelete:CASCADE" json:"player1_id"`
	Player2ID   uint           `gorm:"not null;constraint:OnDelete:CASCADE" json:"player2_id"`
	WinnerID    uint           `gorm:"not null;constraint:OnDelete:CASCADE" json:"winner_id"`
	Status      string         `gorm:"size:20;default:pending" json:"status"` // pending, confirmed, rejected, cancelled
	CreatedAt   time.Time      `json:"created_at"`
	ConfirmedAt *time.Time     `json:"confirmed_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Player1 Player `gorm:"foreignKey:Player1ID;references:ID" json:"player1,omitempty"`
	Player2 Player `gorm:"foreignKey:Player2ID;references:ID" json:"player2,omitempty"`
	Winner  Player `gorm:"foreignKey:WinnerID;references:ID" json:"winner,omitempty"`
}

func (Match) TableName() string {
	return "matches"
}

type PaginatedMatchResponse struct {
	Data       []Match `json:"data"`
	Total      int64   `json:"total"`
	Page       int     `json:"page"`
	PageSize   int     `json:"pageSize"`
	TotalPages int     `json:"totalPages"`
}

type CreateMatchRequest struct {
	Player1ID uint `json:"player1_id" binding:"required"`
	Player2ID uint `json:"player2_id" binding:"required"`
	WinnerID  uint `json:"winner_id" binding:"required"`
}

type UpdateMatchStatusRequest struct {
	Status   *string `json:"status,omitempty" binding:"omitempty,oneof=confirmed rejected cancelled"`
	WinnerID *uint   `json:"winner_id,omitempty"`
}