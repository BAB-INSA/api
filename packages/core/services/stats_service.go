package services

import (
	"core/models"
	"time"

	"gorm.io/gorm"
)

type StatsService struct {
	db *gorm.DB
}

func NewStatsService(db *gorm.DB) *StatsService {
	return &StatsService{
		db: db,
	}
}

func (s *StatsService) GetStats() (*models.Stats, error) {
	var totalPlayers int64
	var totalMatches int64
	var matchesLast7Days int64
	var matchesPrevious7Days int64
	var totalTeams int64
	var totalTeamMatches int64
	var teamMatchesLast7Days int64
	var teamMatchesPrevious7Days int64

	// Count total players
	if err := s.db.Model(&models.Player{}).Count(&totalPlayers).Error; err != nil {
		return nil, err
	}

	// Count total matches (solo)
	if err := s.db.Model(&models.Match{}).Count(&totalMatches).Error; err != nil {
		return nil, err
	}

	// Count total teams
	if err := s.db.Model(&models.Team{}).Count(&totalTeams).Error; err != nil {
		return nil, err
	}

	// Count total team matches
	if err := s.db.Model(&models.TeamMatch{}).Count(&totalTeamMatches).Error; err != nil {
		return nil, err
	}

	// Calculate date ranges
	now := time.Now()
	last7DaysStart := now.AddDate(0, 0, -7)
	previous7DaysStart := now.AddDate(0, 0, -14)
	previous7DaysEnd := last7DaysStart

	// Count solo matches in the last 7 days
	if err := s.db.Model(&models.Match{}).
		Where("created_at >= ?", last7DaysStart).
		Count(&matchesLast7Days).Error; err != nil {
		return nil, err
	}

	// Count solo matches in the previous 7 days (7-14 days ago)
	if err := s.db.Model(&models.Match{}).
		Where("created_at >= ? AND created_at < ?", previous7DaysStart, previous7DaysEnd).
		Count(&matchesPrevious7Days).Error; err != nil {
		return nil, err
	}

	// Count team matches in the last 7 days
	if err := s.db.Model(&models.TeamMatch{}).
		Where("created_at >= ?", last7DaysStart).
		Count(&teamMatchesLast7Days).Error; err != nil {
		return nil, err
	}

	// Count team matches in the previous 7 days (7-14 days ago)
	if err := s.db.Model(&models.TeamMatch{}).
		Where("created_at >= ? AND created_at < ?", previous7DaysStart, previous7DaysEnd).
		Count(&teamMatchesPrevious7Days).Error; err != nil {
		return nil, err
	}

	stats := &models.Stats{
		TotalPlayers:             totalPlayers,
		TotalMatches:             totalMatches,
		MatchesLast7Days:         matchesLast7Days,
		MatchesPrevious7Days:     matchesPrevious7Days,
		TotalTeams:               totalTeams,
		TotalTeamMatches:         totalTeamMatches,
		TeamMatchesLast7Days:     teamMatchesLast7Days,
		TeamMatchesPrevious7Days: teamMatchesPrevious7Days,
	}

	return stats, nil
}
