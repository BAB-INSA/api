package models

import (
	"time"

	"gorm.io/gorm"
)

type TeamEloHistory struct {
	ID             uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	PlayerID       uint           `gorm:"not null;constraint:OnDelete:CASCADE" json:"player_id"`
	TeamMatchID    uint           `gorm:"not null;constraint:OnDelete:CASCADE" json:"team_match_id"`
	EloBefore      float64        `gorm:"not null" json:"elo_before"`
	EloAfter       float64        `gorm:"not null" json:"elo_after"`
	EloChange      float64        `gorm:"not null" json:"elo_change"`
	OpponentTeamID *uint          `json:"opponent_team_id"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Player       Player    `gorm:"foreignKey:PlayerID;references:ID" json:"player,omitempty"`
	TeamMatch    TeamMatch `gorm:"foreignKey:TeamMatchID;references:ID" json:"team_match,omitempty"`
	OpponentTeam *Team     `gorm:"foreignKey:OpponentTeamID;references:ID" json:"opponent_team,omitempty"`
}

func (TeamEloHistory) TableName() string {
	return "team_elo_history"
}
