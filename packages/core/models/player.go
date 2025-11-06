package models

import (
	"time"

	"gorm.io/gorm"
)

type Player struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	Username     string         `gorm:"size:255;not null" json:"username"`
	EloRating    float64        `gorm:"default:1200" json:"elo_rating"`
	Rank         int            `gorm:"default:1" json:"rank"`
	TotalMatches int            `gorm:"default:0" json:"total_matches"`
	Wins         int            `gorm:"default:0" json:"wins"`
	Losses       int            `gorm:"default:0" json:"losses"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Player1Matches []Match      `gorm:"foreignKey:Player1ID" json:"player1_matches,omitempty"`
	Player2Matches []Match      `gorm:"foreignKey:Player2ID" json:"player2_matches,omitempty"`
	WonMatches     []Match      `gorm:"foreignKey:WinnerID" json:"won_matches,omitempty"`
	EloHistory     []EloHistory `gorm:"foreignKey:PlayerID" json:"elo_history,omitempty"`
}

func (Player) TableName() string {
	return "players"
}

type PaginatedPlayersResponse struct {
	Data       []Player `json:"data"`
	Total      int64    `json:"total"`
	Page       int      `json:"page"`
	PageSize   int      `json:"pageSize"`
	TotalPages int      `json:"totalPages"`
}
