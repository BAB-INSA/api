package models

import (
	"time"

	"gorm.io/gorm"
)

type Tournament struct {
	ID             uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	Name           string         `gorm:"size:255;not null" json:"name"`
	Slug           string         `gorm:"size:255;unique;not null" json:"slug"`
	Type           string         `gorm:"size:20;not null;default:team" json:"type"`     // solo, team
	Status         string         `gorm:"size:20;not null;default:opened" json:"status"` // opened, ongoing, finished
	Description    string         `gorm:"type:text" json:"description"`
	NbParticipants int            `gorm:"default:0" json:"nb_participants"`
	NbMatches      int            `gorm:"default:0" json:"nb_matches"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	TournamentTeams []TournamentTeam `gorm:"foreignKey:TournamentID" json:"tournament_teams,omitempty"`
	TeamMatches     []TeamMatch      `gorm:"foreignKey:TournamentID" json:"team_matches,omitempty"`
	Matches         []Match          `gorm:"foreignKey:TournamentID" json:"matches,omitempty"`
}

func (Tournament) TableName() string {
	return "tournaments"
}

type TournamentTeam struct {
	ID           uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	TournamentID uint           `gorm:"not null;constraint:OnDelete:CASCADE" json:"tournament_id"`
	TeamID       uint           `gorm:"not null;constraint:OnDelete:CASCADE" json:"team_id"`
	Wins         int            `gorm:"default:0" json:"wins"`
	Losses       int            `gorm:"default:0" json:"losses"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Tournament Tournament `gorm:"foreignKey:TournamentID;references:ID" json:"tournament,omitempty"`
	Team       Team       `gorm:"foreignKey:TeamID;references:ID" json:"team,omitempty"`
}

func (TournamentTeam) TableName() string {
	return "tournament_teams"
}

// DTOs

type CreateTournamentRequest struct {
	Name        string `json:"name" binding:"required"`
	Type        string `json:"type" binding:"required,oneof=solo team"`
	Description string `json:"description,omitempty"`
}

type UpdateTournamentRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Status      *string `json:"status,omitempty" binding:"omitempty,oneof=opened ongoing finished"`
}

type JoinTournamentRequest struct {
	TeamID uint `json:"team_id" binding:"required"`
}

// Responses

type TournamentListItem struct {
	ID             uint      `json:"id"`
	Name           string    `json:"name"`
	Slug           string    `json:"slug"`
	Type           string    `json:"type"`
	Status         string    `json:"status"`
	Description    string    `json:"description"`
	NbParticipants int       `json:"nb_participants"`
	NbMatches      int       `json:"nb_matches"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (TournamentListItem) TableName() string {
	return "tournaments"
}

type PaginatedTournamentsResponse struct {
	Data       []TournamentListItem `json:"data"`
	Total      int64                `json:"total"`
	Page       int                  `json:"page"`
	PageSize   int                  `json:"pageSize"`
	TotalPages int                  `json:"totalPages"`
}

type TournamentTeamItem struct {
	ID     uint `json:"id"`
	TeamID uint `json:"team_id"`
	Wins   int  `json:"wins"`
	Losses int  `json:"losses"`
	Team   Team `json:"team"`
}

type PaginatedTournamentTeamsResponse struct {
	Data       []TournamentTeamItem `json:"data"`
	Total      int64                `json:"total"`
	Page       int                  `json:"page"`
	PageSize   int                  `json:"pageSize"`
	TotalPages int                  `json:"totalPages"`
}
