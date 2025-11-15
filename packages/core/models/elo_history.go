package models

import (
	"time"

	"gorm.io/gorm"
)

type EloHistory struct {
	ID             uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	PlayerID       uint           `gorm:"not null;constraint:OnDelete:CASCADE" json:"player_id"`
	MatchID        uint           `gorm:"not null;constraint:OnDelete:CASCADE" json:"match_id"`
	EloBefore      float64        `gorm:"not null" json:"elo_before"`
	EloAfter       float64        `gorm:"not null" json:"elo_after"`
	EloChange      float64        `gorm:"not null" json:"elo_change"`
	OpponentID     *uint          `json:"opponent_id"`
	OpponentTeamID *uint          `json:"opponent_team_id"`
	MatchType      string         `gorm:"size:20;default:solo" json:"match_type"` // solo or team
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Player       Player  `gorm:"foreignKey:PlayerID;references:ID" json:"player,omitempty"`
	Match        Match   `gorm:"foreignKey:MatchID;references:ID" json:"match,omitempty"`
	Opponent     *Player `gorm:"foreignKey:OpponentID;references:ID" json:"opponent,omitempty"`
	OpponentTeam *Team   `gorm:"foreignKey:OpponentTeamID;references:ID" json:"opponent_team,omitempty"`
}

func (EloHistory) TableName() string {
	return "elo_history"
}
