package services

import (
	"core/models"
	"log"
	"time"

	"gorm.io/gorm"
)

type AutoValidationService struct {
	db           *gorm.DB
	matchService *MatchService
}

func NewAutoValidationService(db *gorm.DB, matchService *MatchService) *AutoValidationService {
	return &AutoValidationService{
		db:           db,
		matchService: matchService,
	}
}

// ValidateExpiredMatches finds and confirms all pending matches that are older than 24 hours
func (s *AutoValidationService) ValidateExpiredMatches() error {
	// Calculate the cutoff time (24 hours ago)
	cutoffTime := time.Now().Add(-24 * time.Hour)

	// Find all pending matches older than 24 hours
	var expiredMatches []models.Match
	result := s.db.Where("status = ? AND created_at < ?", "pending", cutoffTime).Find(&expiredMatches)
	
	if result.Error != nil {
		log.Printf("Error finding expired matches: %v", result.Error)
		return result.Error
	}

	if len(expiredMatches) == 0 {
		log.Println("No expired matches found")
		return nil
	}

	log.Printf("Found %d expired matches to validate", len(expiredMatches))

	// Confirm each expired match
	for _, match := range expiredMatches {
		log.Printf("Auto-confirming match ID %d (created at %v)", match.ID, match.CreatedAt)
		
		_, err := s.matchService.ConfirmMatch(match.ID)
		if err != nil {
			log.Printf("Error auto-confirming match ID %d: %v", match.ID, err)
			// Continue with other matches even if one fails
			continue
		}
		
		log.Printf("Successfully auto-confirmed match ID %d", match.ID)
	}

	return nil
}

// GetPendingMatchesCount returns the number of pending matches
func (s *AutoValidationService) GetPendingMatchesCount() (int64, error) {
	var count int64
	result := s.db.Model(&models.Match{}).Where("status = ?", "pending").Count(&count)
	
	if result.Error != nil {
		return 0, result.Error
	}
	
	return count, nil
}

// GetExpiredMatchesCount returns the number of pending matches older than 24 hours
func (s *AutoValidationService) GetExpiredMatchesCount() (int64, error) {
	cutoffTime := time.Now().Add(-24 * time.Hour)
	
	var count int64
	result := s.db.Model(&models.Match{}).Where("status = ? AND created_at < ?", "pending", cutoffTime).Count(&count)
	
	if result.Error != nil {
		return 0, result.Error
	}
	
	return count, nil
}