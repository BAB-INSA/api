package models

import (
	"time"

	"gorm.io/gorm"
)

type Team struct {
	ID           uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	Player1ID    uint           `gorm:"not null;constraint:OnDelete:CASCADE" json:"player1_id"`
	Player2ID    uint           `gorm:"not null;constraint:OnDelete:CASCADE" json:"player2_id"`
	Name         string         `gorm:"size:255" json:"name"`
	Slug         string         `gorm:"size:255;unique;not null" json:"slug"`
	EloRating    float64        `gorm:"default:1200" json:"elo_rating"`
	TotalMatches int            `gorm:"default:0" json:"total_matches"`
	Wins         int            `gorm:"default:0" json:"wins"`
	Losses       int            `gorm:"default:0" json:"losses"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Player1      Player      `gorm:"foreignKey:Player1ID;references:ID" json:"player1,omitempty"`
	Player2      Player      `gorm:"foreignKey:Player2ID;references:ID" json:"player2,omitempty"`
	Team1Matches []TeamMatch `gorm:"foreignKey:Team1ID" json:"team1_matches,omitempty"`
	Team2Matches []TeamMatch `gorm:"foreignKey:Team2ID" json:"team2_matches,omitempty"`
	WonMatches   []TeamMatch `gorm:"foreignKey:WinnerTeamID" json:"won_matches,omitempty"`
}

func (Team) TableName() string {
	return "teams"
}

type PaginatedTeamsResponse struct {
	Data       []Team `json:"data"`
	Total      int64  `json:"total"`
	Page       int    `json:"page"`
	PageSize   int    `json:"pageSize"`
	TotalPages int    `json:"totalPages"`
}

type CreateTeamRequest struct {
	Player1ID uint   `json:"player1_id" binding:"required"`
	Player2ID uint   `json:"player2_id" binding:"required"`
	Name      string `json:"name,omitempty"`
}

type UpdateTeamRequest struct {
	Name *string `json:"name,omitempty"`
}
