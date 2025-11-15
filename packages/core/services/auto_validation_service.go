package services

import (
	"core/models"
	"log"
	"time"

	"gorm.io/gorm"
)

type AutoValidationService struct {
	db               *gorm.DB
	matchService     *MatchService
	teamMatchService *TeamMatchService
}

func NewAutoValidationService(db *gorm.DB, matchService *MatchService, teamMatchService *TeamMatchService) *AutoValidationService {
	return &AutoValidationService{
		db:               db,
		matchService:     matchService,
		teamMatchService: teamMatchService,
	}
}

// ValidateExpiredMatches finds and confirms all pending matches that are older than 24 hours
func (s *AutoValidationService) ValidateExpiredMatches() error {
	// Calculate the cutoff time (24 hours ago)
	cutoffTime := time.Now().Add(-24 * time.Hour)

	// Find all pending solo matches older than 24 hours
	var expiredMatches []models.Match
	result := s.db.Where("status = ? AND created_at < ?", "pending", cutoffTime).Find(&expiredMatches)

	if result.Error != nil {
		log.Printf("Error finding expired matches: %v", result.Error)
		return result.Error
	}

	// Find all pending team matches older than 24 hours
	var expiredTeamMatches []models.TeamMatch
	teamResult := s.db.Where("status = ? AND created_at < ?", "pending", cutoffTime).Find(&expiredTeamMatches)

	if teamResult.Error != nil {
		log.Printf("Error finding expired team matches: %v", teamResult.Error)
		return teamResult.Error
	}

	totalExpired := len(expiredMatches) + len(expiredTeamMatches)
	if totalExpired == 0 {
		log.Println("No expired matches found")
		return nil
	}

	log.Printf("Found %d expired matches to validate (%d solo, %d team)", totalExpired, len(expiredMatches), len(expiredTeamMatches))

	// Confirm each expired solo match
	for _, match := range expiredMatches {
		log.Printf("Auto-confirming solo match ID %d (created at %v)", match.ID, match.CreatedAt)

		_, err := s.matchService.ConfirmMatch(match.ID)
		if err != nil {
			log.Printf("Error auto-confirming solo match ID %d: %v", match.ID, err)
			// Continue with other matches even if one fails
			continue
		}

		log.Printf("Successfully auto-confirmed solo match ID %d", match.ID)
	}

	// Confirm each expired team match
	for _, teamMatch := range expiredTeamMatches {
		log.Printf("Auto-confirming team match ID %d (created at %v)", teamMatch.ID, teamMatch.CreatedAt)

		_, err := s.teamMatchService.ConfirmTeamMatch(teamMatch.ID)
		if err != nil {
			log.Printf("Error auto-confirming team match ID %d: %v", teamMatch.ID, err)
			// Continue with other matches even if one fails
			continue
		}

		log.Printf("Successfully auto-confirmed team match ID %d", teamMatch.ID)
	}

	return nil
}

// GetPendingMatchesCount returns the number of pending matches (solo + team)
func (s *AutoValidationService) GetPendingMatchesCount() (int64, error) {
	var soloCount int64
	result := s.db.Model(&models.Match{}).Where("status = ?", "pending").Count(&soloCount)

	if result.Error != nil {
		return 0, result.Error
	}

	var teamCount int64
	teamResult := s.db.Model(&models.TeamMatch{}).Where("status = ?", "pending").Count(&teamCount)

	if teamResult.Error != nil {
		return 0, teamResult.Error
	}

	return soloCount + teamCount, nil
}

// GetExpiredMatchesCount returns the number of pending matches older than 24 hours (solo + team)
func (s *AutoValidationService) GetExpiredMatchesCount() (int64, error) {
	cutoffTime := time.Now().Add(-24 * time.Hour)

	var soloCount int64
	result := s.db.Model(&models.Match{}).Where("status = ? AND created_at < ?", "pending", cutoffTime).Count(&soloCount)

	if result.Error != nil {
		return 0, result.Error
	}

	var teamCount int64
	teamResult := s.db.Model(&models.TeamMatch{}).Where("status = ? AND created_at < ?", "pending", cutoffTime).Count(&teamCount)

	if teamResult.Error != nil {
		return 0, teamResult.Error
	}

	return soloCount + teamCount, nil
}
