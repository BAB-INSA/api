package services

import (
	"core/models"
	"errors"

	"gorm.io/gorm"
)

type PlayerService struct {
	db *gorm.DB
}

func NewPlayerService(db *gorm.DB) *PlayerService {
	return &PlayerService{
		db: db,
	}
}

func (s *PlayerService) GetPlayerByID(id uint) (*models.Player, error) {
	var player models.Player
	
	result := s.db.First(&player, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("player not found")
		}
		return nil, result.Error
	}

	return &player, nil
}

func (s *PlayerService) CreatePlayer(userID uint, username string) (*models.Player, error) {
	player := &models.Player{
		ID:           userID,
		Username:     username,
		EloRating:    1200,
		TotalMatches: 0,
		Wins:         0,
		Losses:       0,
	}

	result := s.db.Create(player)
	if result.Error != nil {
		return nil, result.Error
	}

	return player, nil
}

func (s *PlayerService) GetEloHistoryByPlayerID(playerID uint) ([]models.EloHistory, error) {
	var eloHistory []models.EloHistory
	
	result := s.db.Where("player_id = ?", playerID).
		Order("id ASC").
		Preload("Match").
		Preload("Opponent").
		Find(&eloHistory)
	
	if result.Error != nil {
		return nil, result.Error
	}

	return eloHistory, nil
}

func (s *PlayerService) GetTopPlayersByElo(limit int) ([]models.Player, error) {
	var players []models.Player
	
	result := s.db.Order("elo_rating DESC").
		Limit(limit).
		Find(&players)
	
	if result.Error != nil {
		return nil, result.Error
	}

	return players, nil
}

func (s *PlayerService) UpdateWinStreak(playerID uint, isWin bool) error {
	var player models.Player
	
	result := s.db.First(&player, playerID)
	if result.Error != nil {
		return result.Error
	}

	if isWin {
		// Victoire : incrémenter la série actuelle
		player.CurrentWinStreak++
		// Mettre à jour la meilleure série si nécessaire
		if player.CurrentWinStreak > player.BestWinStreak {
			player.BestWinStreak = player.CurrentWinStreak
		}
	} else {
		// Défaite : remettre la série actuelle à zéro
		player.CurrentWinStreak = 0
	}

	return s.db.Save(&player).Error
}

func (s *PlayerService) GetTopPlayersByCurrentStreak(limit int) ([]models.Player, error) {
	var players []models.Player
	
	result := s.db.Order("current_win_streak DESC").
		Where("current_win_streak > 0").
		Limit(limit).
		Find(&players)
	
	if result.Error != nil {
		return nil, result.Error
	}

	return players, nil
}

func (s *PlayerService) GetTopPlayersByBestStreak(limit int) ([]models.Player, error) {
	var players []models.Player
	
	result := s.db.Order("best_win_streak DESC").
		Limit(limit).
		Find(&players)
	
	if result.Error != nil {
		return nil, result.Error
	}

	return players, nil
}

func (s *PlayerService) GetPlayerMatches(playerID uint, filter string, page int, pageSize int) (*models.PaginatedMatchResponse, error) {
	var matches []models.Match
	var total int64
	
	baseQuery := s.db.Model(&models.Match{}).Where("player1_id = ? OR player2_id = ?", playerID, playerID)
	
	switch filter {
	case "wins":
		baseQuery = baseQuery.Where("winner_id = ?", playerID)
	case "losses":
		baseQuery = baseQuery.Where("winner_id != ? AND (player1_id = ? OR player2_id = ?)", playerID, playerID, playerID)
	}
	
	// Count total records
	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, err
	}
	
	// Calculate offset
	offset := (page - 1) * pageSize
	
	// Get paginated matches
	query := baseQuery.Order("created_at DESC").
		Preload("Player1").
		Preload("Player2").
		Preload("Winner").
		Offset(offset).
		Limit(pageSize)
	
	if err := query.Find(&matches).Error; err != nil {
		return nil, err
	}
	
	// Calculate total pages
	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))
	
	return &models.PaginatedMatchResponse{
		Data:       matches,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

func (s *PlayerService) GetAllPlayers(orderBy string, direction string, page int, pageSize int) (*models.PaginatedPlayersResponse, error) {
	var players []models.Player
	var total int64
	
	// Validate order by field
	allowedOrderBy := map[string]bool{
		"created_at":  true,
		"elo_rating":  true,
		"username":    true,
	}
	
	if !allowedOrderBy[orderBy] {
		orderBy = "created_at"
	}
	
	// Validate direction
	if direction != "ASC" && direction != "DESC" {
		direction = "DESC"
	}
	
	// Count total records
	if err := s.db.Model(&models.Player{}).Count(&total).Error; err != nil {
		return nil, err
	}
	
	// Calculate offset
	offset := (page - 1) * pageSize
	
	// Build order clause
	orderClause := orderBy + " " + direction
	
	// Get paginated players
	if err := s.db.Order(orderClause).
		Offset(offset).
		Limit(pageSize).
		Find(&players).Error; err != nil {
		return nil, err
	}
	
	// Calculate total pages
	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))
	
	return &models.PaginatedPlayersResponse{
		Data:       players,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}


