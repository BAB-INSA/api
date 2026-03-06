package services

import (
	"core/models"

	"gorm.io/gorm"
)

type EloHistoryService struct {
	db *gorm.DB
}

func NewEloHistoryService(db *gorm.DB) *EloHistoryService {
	return &EloHistoryService{
		db: db,
	}
}

func (s *EloHistoryService) GetRecentEloChanges(limit int) ([]models.EloHistory, error) {
	var eloHistory []models.EloHistory

	result := s.db.Order("created_at DESC").
		Limit(limit).
		Preload("Player").
		Preload("Match").
		Preload("Opponent").
		Find(&eloHistory)

	if result.Error != nil {
		return nil, result.Error
	}

	return eloHistory, nil
}

func (s *EloHistoryService) GetRecentTeamEloChanges(limit int) ([]models.TeamEloHistory, error) {
	var eloHistory []models.TeamEloHistory

	result := s.db.Order("created_at DESC").
		Limit(limit).
		Preload("Player").
		Preload("TeamMatch").
		Preload("OpponentTeam").
		Find(&eloHistory)

	if result.Error != nil {
		return nil, result.Error
	}

	return eloHistory, nil
}
