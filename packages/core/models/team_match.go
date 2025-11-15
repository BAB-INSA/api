package models

import (
	"time"

	"gorm.io/gorm"
)

type TeamMatch struct {
	ID           uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	Team1ID      uint           `gorm:"not null;constraint:OnDelete:CASCADE" json:"team1_id"`
	Team2ID      uint           `gorm:"not null;constraint:OnDelete:CASCADE" json:"team2_id"`
	WinnerTeamID uint           `gorm:"not null;constraint:OnDelete:CASCADE" json:"winner_team_id"`
	Status       string         `gorm:"size:20;default:pending" json:"status"` // pending, confirmed, rejected, cancelled
	CreatedAt    time.Time      `json:"created_at"`
	ConfirmedAt  *time.Time     `json:"confirmed_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Team1      Team `gorm:"foreignKey:Team1ID;references:ID" json:"team1,omitempty"`
	Team2      Team `gorm:"foreignKey:Team2ID;references:ID" json:"team2,omitempty"`
	WinnerTeam Team `gorm:"foreignKey:WinnerTeamID;references:ID" json:"winner_team,omitempty"`
}

func (TeamMatch) TableName() string {
	return "team_matches"
}

type PaginatedTeamMatchResponse struct {
	Data       []TeamMatch `json:"data"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"pageSize"`
	TotalPages int         `json:"totalPages"`
}

type CreateTeamMatchRequest struct {
	Team1ID      uint `json:"team1_id" binding:"required"`
	Team2ID      uint `json:"team2_id" binding:"required"`
	WinnerTeamID uint `json:"winner_team_id" binding:"required"`
}

type UpdateTeamMatchStatusRequest struct {
	Status       *string `json:"status,omitempty" binding:"omitempty,oneof=confirmed rejected cancelled"`
	WinnerTeamID *uint   `json:"winner_team_id,omitempty"`
}
