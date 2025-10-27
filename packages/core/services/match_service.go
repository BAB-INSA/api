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

	// Update status
	now := time.Now()
	match.Status = req.Status
	if req.Status == "confirmed" {
		match.ConfirmedAt = &now
	}

	if err := tx.Save(&match).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// If confirmed, calculate ELO and update stats
	if req.Status == "confirmed" {
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

	// Update win streaks
	var winner, loser models.Player
	if err := tx.First(&winner, winnerID).Error; err != nil {
		return err
	}
	if err := tx.First(&loser, loserID).Error; err != nil {
		return err
	}

	// Winner: increment current streak, update best if necessary
	newWinnerStreak := winner.CurrentWinStreak + 1
	updates := map[string]interface{}{"current_win_streak": newWinnerStreak}
	if newWinnerStreak > winner.BestWinStreak {
		updates["best_win_streak"] = newWinnerStreak
	}
	if err := tx.Model(&models.Player{}).Where("id = ?", winnerID).Updates(updates).Error; err != nil {
		return err
	}

	// Loser: reset current streak to 0
	if err := tx.Model(&models.Player{}).Where("id = ?", loserID).Update("current_win_streak", 0).Error; err != nil {
		return err
	}

	return nil
}

func (s *MatchService) ConfirmMatch(matchID uint) (*models.Match, error) {
	confirmRequest := models.UpdateMatchStatusRequest{
		Status: "confirmed",
	}
	return s.UpdateMatchStatus(matchID, confirmRequest)
}