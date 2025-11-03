package services

import (
	"core/models"
	"core/utils"
	"errors"
	"time"

	"gorm.io/gorm"
)

type MatchService struct {
	db *gorm.DB
}

func NewMatchService(db *gorm.DB) *MatchService {
	return &MatchService{
		db: db,
	}
}

func (s *MatchService) GetRecentMatches(limit int) ([]models.Match, error) {
	var matches []models.Match

	result := s.db.Order("created_at DESC").
		Limit(limit).
		Preload("Player1").
		Preload("Player2").
		Preload("Winner").
		Find(&matches)

	if result.Error != nil {
		return nil, result.Error
	}

	return matches, nil
}

type MatchFilters struct {
	PlayerID *uint      `json:"player_id,omitempty"`
	Status   *string    `json:"status,omitempty"`
	DateFrom *time.Time `json:"date_from,omitempty"`
	DateTo   *time.Time `json:"date_to,omitempty"`
	Page     int        `json:"page"`
	PerPage  int        `json:"per_page"`
}

func (s *MatchService) GetMatches(filters MatchFilters) (*models.PaginatedMatchResponse, error) {
	var matches []models.Match
	var total int64

	// Build query
	query := s.db.Model(&models.Match{})

	// Apply filters
	if filters.PlayerID != nil {
		query = query.Where("player1_id = ? OR player2_id = ?", *filters.PlayerID, *filters.PlayerID)
	}

	if filters.Status != nil {
		query = query.Where("status = ?", *filters.Status)
	}

	if filters.DateFrom != nil {
		query = query.Where("created_at >= ?", *filters.DateFrom)
	}

	if filters.DateTo != nil {
		// Add 24 hours to include the entire day
		dateTo := filters.DateTo.Add(24 * time.Hour)
		query = query.Where("created_at < ?", dateTo)
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// Calculate offset
	offset := (filters.Page - 1) * filters.PerPage

	// Get paginated results
	result := query.
		Offset(offset).
		Limit(filters.PerPage).
		Order("created_at DESC").
		Preload("Player1").
		Preload("Player2").
		Preload("Winner").
		Find(&matches)

	if result.Error != nil {
		return nil, result.Error
	}

	// Calculate total pages
	totalPages := int((total + int64(filters.PerPage) - 1) / int64(filters.PerPage))

	return &models.PaginatedMatchResponse{
		Data:       matches,
		Total:      total,
		Page:       filters.Page,
		PageSize:   filters.PerPage,
		TotalPages: totalPages,
	}, nil
}

func (s *MatchService) CreateMatch(req models.CreateMatchRequest) (*models.Match, error) {
	// Validate that players exist
	var player1, player2 models.Player
	if err := s.db.First(&player1, req.Player1ID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("player1 not found")
		}
		return nil, err
	}

	if err := s.db.First(&player2, req.Player2ID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("player2 not found")
		}
		return nil, err
	}

	// Validate that players are different
	if req.Player1ID == req.Player2ID {
		return nil, errors.New("player1 and player2 must be different")
	}

	// Validate that winner is one of the players
	if req.WinnerID != req.Player1ID && req.WinnerID != req.Player2ID {
		return nil, errors.New("winner must be either player1 or player2")
	}

	// Start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Create the match in pending status
	now := time.Now()
	match := models.Match{
		Player1ID: req.Player1ID,
		Player2ID: req.Player2ID,
		WinnerID:  req.WinnerID,
		Status:    "pending",
		CreatedAt: now,
		// ConfirmedAt will be set when confirmed
	}

	if err := tx.Create(&match).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// No ELO calculations or stats updates for pending matches
	// These will be done when the match is confirmed

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	// Load the created match with relationships
	if err := s.db.Preload("Player1").Preload("Player2").Preload("Winner").First(&match, match.ID).Error; err != nil {
		return nil, err
	}

	return &match, nil
}

func (s *MatchService) UpdateMatchStatus(matchID uint, req models.UpdateMatchStatusRequest) (*models.Match, error) {
	// Start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Get the match
	var match models.Match
	if err := tx.First(&match, matchID).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("match not found")
		}
		return nil, err
	}

	// Check if match is still pending
	if match.Status != "pending" {
		tx.Rollback()
		return nil, errors.New("match is not pending")
	}

	// Update winner_id if provided
	if req.WinnerID != nil {
		// Validate that winner is one of the players
		if *req.WinnerID != match.Player1ID && *req.WinnerID != match.Player2ID {
			tx.Rollback()
			return nil, errors.New("winner must be either player1 or player2")
		}
		match.WinnerID = *req.WinnerID
	}

	// Update status if provided
	now := time.Now()
	if req.Status != nil {
		match.Status = *req.Status
		if *req.Status == "confirmed" {
			match.ConfirmedAt = &now
		}
	}

	if err := tx.Save(&match).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// If confirmed, calculate ELO and update stats
	if match.Status == "confirmed" {
		// Get current player ELO ratings
		var player1, player2 models.Player
		if err := tx.First(&player1, match.Player1ID).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
		if err := tx.First(&player2, match.Player2ID).Error; err != nil {
			tx.Rollback()
			return nil, err
		}

		// Calculate ELO changes
		player1Change, player2Change := utils.CalculateEloChange(
			player1.EloRating,
			player2.EloRating,
			match.WinnerID,
			match.Player1ID,
		)

		// Create ELO history entries
		eloHistory1 := models.EloHistory{
			PlayerID:   match.Player1ID,
			MatchID:    match.ID,
			EloBefore:  player1.EloRating,
			EloAfter:   player1.EloRating + player1Change,
			EloChange:  player1Change,
			OpponentID: &match.Player2ID,
			CreatedAt:  now,
		}

		eloHistory2 := models.EloHistory{
			PlayerID:   match.Player2ID,
			MatchID:    match.ID,
			EloBefore:  player2.EloRating,
			EloAfter:   player2.EloRating + player2Change,
			EloChange:  player2Change,
			OpponentID: &match.Player1ID,
			CreatedAt:  now,
		}

		if err := tx.Create(&eloHistory1).Error; err != nil {
			tx.Rollback()
			return nil, err
		}

		if err := tx.Create(&eloHistory2).Error; err != nil {
			tx.Rollback()
			return nil, err
		}

		// Update player stats and ELO ratings
		if err := s.updatePlayerStatsInTransaction(tx, match.Player1ID, match.Player2ID, match.WinnerID, player1Change, player2Change); err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	// Load the updated match with relationships
	if err := s.db.Preload("Player1").Preload("Player2").Preload("Winner").First(&match, match.ID).Error; err != nil {
		return nil, err
	}

	return &match, nil
}

func (s *MatchService) updatePlayerStatsInTransaction(tx *gorm.DB, player1ID, player2ID, winnerID uint, player1Change, player2Change float64) error {
	// Update ELO ratings
	if err := tx.Model(&models.Player{}).Where("id = ?", player1ID).Update("elo_rating", gorm.Expr("elo_rating + ?", player1Change)).Error; err != nil {
		return err
	}

	if err := tx.Model(&models.Player{}).Where("id = ?", player2ID).Update("elo_rating", gorm.Expr("elo_rating + ?", player2Change)).Error; err != nil {
		return err
	}

	// Update total matches for both players
	if err := tx.Model(&models.Player{}).Where("id IN ?", []uint{player1ID, player2ID}).Update("total_matches", gorm.Expr("total_matches + 1")).Error; err != nil {
		return err
	}

	// Update wins and losses
	if err := tx.Model(&models.Player{}).Where("id = ?", winnerID).Update("wins", gorm.Expr("wins + 1")).Error; err != nil {
		return err
	}

	loserID := player1ID
	if winnerID == player1ID {
		loserID = player2ID
	}
	if err := tx.Model(&models.Player{}).Where("id = ?", loserID).Update("losses", gorm.Expr("losses + 1")).Error; err != nil {
		return err
	}

	return nil
}

func (s *MatchService) reversePlayerStatsInTransaction(tx *gorm.DB, player1ID, player2ID, winnerID uint) error {
	// Reverse total matches for both players
	if err := tx.Model(&models.Player{}).Where("id IN ?", []uint{player1ID, player2ID}).Update("total_matches", gorm.Expr("total_matches - 1")).Error; err != nil {
		return err
	}

	// Reverse wins and losses
	if err := tx.Model(&models.Player{}).Where("id = ?", winnerID).Update("wins", gorm.Expr("wins - 1")).Error; err != nil {
		return err
	}

	loserID := player1ID
	if winnerID == player1ID {
		loserID = player2ID
	}
	if err := tx.Model(&models.Player{}).Where("id = ?", loserID).Update("losses", gorm.Expr("losses - 1")).Error; err != nil {
		return err
	}

	return nil
}

func (s *MatchService) recalculateSubsequentMatchesInTransaction(tx *gorm.DB, player1ID, player2ID uint, deletedMatchTime *time.Time) error {
	if deletedMatchTime == nil {
		return nil // No need to recalculate if match wasn't confirmed
	}

	// Get ALL confirmed matches after the deleted match (not just for these 2 players)
	var subsequentMatches []models.Match
	if err := tx.Where("status = 'confirmed' AND confirmed_at > ?", *deletedMatchTime).
		Order("confirmed_at ASC").Find(&subsequentMatches).Error; err != nil {
		return err
	}

	// For each subsequent match, recalculate ELO
	for _, subsequentMatch := range subsequentMatches {
		// Delete existing ELO history for this match
		if err := tx.Where("match_id = ?", subsequentMatch.ID).Delete(&models.EloHistory{}).Error; err != nil {
			return err
		}

		// Get current ELO ratings for the players
		var player1, player2 models.Player
		if err := tx.First(&player1, subsequentMatch.Player1ID).Error; err != nil {
			return err
		}
		if err := tx.First(&player2, subsequentMatch.Player2ID).Error; err != nil {
			return err
		}

		// Calculate new ELO changes based on current ratings
		player1Change, player2Change := utils.CalculateEloChange(
			player1.EloRating,
			player2.EloRating,
			subsequentMatch.WinnerID,
			subsequentMatch.Player1ID,
		)

		// Create new ELO history entries
		eloHistory1 := models.EloHistory{
			PlayerID:   subsequentMatch.Player1ID,
			MatchID:    subsequentMatch.ID,
			EloBefore:  player1.EloRating,
			EloAfter:   player1.EloRating + player1Change,
			EloChange:  player1Change,
			OpponentID: &subsequentMatch.Player2ID,
			CreatedAt:  *subsequentMatch.ConfirmedAt,
		}

		eloHistory2 := models.EloHistory{
			PlayerID:   subsequentMatch.Player2ID,
			MatchID:    subsequentMatch.ID,
			EloBefore:  player2.EloRating,
			EloAfter:   player2.EloRating + player2Change,
			EloChange:  player2Change,
			OpponentID: &subsequentMatch.Player1ID,
			CreatedAt:  *subsequentMatch.ConfirmedAt,
		}

		if err := tx.Create(&eloHistory1).Error; err != nil {
			return err
		}
		if err := tx.Create(&eloHistory2).Error; err != nil {
			return err
		}

		// Update player ELO ratings
		if err := tx.Model(&models.Player{}).Where("id = ?", subsequentMatch.Player1ID).
			Update("elo_rating", player1.EloRating+player1Change).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.Player{}).Where("id = ?", subsequentMatch.Player2ID).
			Update("elo_rating", player2.EloRating+player2Change).Error; err != nil {
			return err
		}
	}

	return nil
}

func (s *MatchService) RejectMatch(matchID uint) (*models.Match, error) {
	status := "rejected"
	rejectRequest := models.UpdateMatchStatusRequest{
		Status: &status,
	}
	return s.UpdateMatchStatus(matchID, rejectRequest)
}

func (s *MatchService) ConfirmMatch(matchID uint) (*models.Match, error) {
	status := "confirmed"
	confirmRequest := models.UpdateMatchStatusRequest{
		Status: &status,
	}
	return s.UpdateMatchStatus(matchID, confirmRequest)
}

func (s *MatchService) CancelMatch(matchID uint) (*models.Match, error) {
	// Get the match
	var match models.Match
	if err := s.db.First(&match, matchID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("match not found")
		}
		return nil, err
	}

	// Update status to cancelled
	match.Status = "cancelled"
	if err := s.db.Save(&match).Error; err != nil {
		return nil, err
	}

	// Load the updated match with relationships
	if err := s.db.Preload("Player1").Preload("Player2").Preload("Winner").First(&match, match.ID).Error; err != nil {
		return nil, err
	}

	return &match, nil
}

func (s *MatchService) DeleteMatch(matchID uint) (*models.Match, error) {
	// Start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Get the match
	var match models.Match
	if err := tx.First(&match, matchID).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("match not found")
		}
		return nil, err
	}

	// If match was confirmed, reverse the stats and ELO changes
	if match.Status == "confirmed" {
		// Get ELO history entries for this match to reverse changes
		var eloHistories []models.EloHistory
		if err := tx.Where("match_id = ?", match.ID).Find(&eloHistories).Error; err != nil {
			tx.Rollback()
			return nil, err
		}

		// Reverse ELO changes for both players
		for _, eloHistory := range eloHistories {
			// Reverse ELO rating
			if err := tx.Model(&models.Player{}).Where("id = ?", eloHistory.PlayerID).
				Update("elo_rating", gorm.Expr("elo_rating - ?", eloHistory.EloChange)).Error; err != nil {
				tx.Rollback()
				return nil, err
			}
		}

		// Reverse player stats
		if err := s.reversePlayerStatsInTransaction(tx, match.Player1ID, match.Player2ID, match.WinnerID); err != nil {
			tx.Rollback()
			return nil, err
		}

		// Delete ELO history entries for this match
		if err := tx.Where("match_id = ?", match.ID).Delete(&models.EloHistory{}).Error; err != nil {
			tx.Rollback()
			return nil, err
		}

		// Recalculate ELO for subsequent matches involving these players
		if err := s.recalculateSubsequentMatchesInTransaction(tx, match.Player1ID, match.Player2ID, match.ConfirmedAt); err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	// Soft delete the match (sets deleted_at)
	if err := tx.Delete(&match).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	// Load the deleted match with relationships using Unscoped
	if err := s.db.Unscoped().Preload("Player1").Preload("Player2").Preload("Winner").First(&match, match.ID).Error; err != nil {
		return nil, err
	}

	return &match, nil
}
