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

func (s *PlayerService) CreatePlayerWithTx(tx *gorm.DB, userID uint, username string) (*models.Player, error) {
	player := &models.Player{
		ID:           userID,
		Username:     username,
		EloRating:    1200,
		TotalMatches: 0,
		Wins:         0,
		Losses:       0,
	}

	if err := tx.Create(player).Error; err != nil {
		return nil, err
	}

	return player, nil
}

func (s *PlayerService) GetEloHistoryByPlayerID(playerID uint, matchType string) ([]models.EloHistory, error) {
	var eloHistory []models.EloHistory

	query := s.db.Where("player_id = ?", playerID)

	// Filter by match type if specified
	if matchType != "" {
		query = query.Where("match_type = ?", matchType)
	}

	result := query.Order("id ASC").
		Preload("Match").
		Preload("Opponent").
		Preload("OpponentTeam").
		Find(&eloHistory)

	if result.Error != nil {
		return nil, result.Error
	}

	return eloHistory, nil
}

func (s *PlayerService) GetTopPlayersByElo(limit int, currentUserID *uint) ([]models.Player, error) {
	var players []models.Player

	result := s.db.Order("elo_rating DESC").
		Limit(limit).
		Find(&players)

	if result.Error != nil {
		return nil, result.Error
	}

	// Si currentUserID est fourni et que l'utilisateur n'est pas dans le top, l'ajouter
	if currentUserID != nil {
		// Vérifier si l'utilisateur connecté est déjà dans la liste
		userInTop := false
		for _, player := range players {
			if player.ID == *currentUserID {
				userInTop = true
				break
			}
		}

		// Si l'utilisateur n'est pas dans le top, le récupérer et l'ajouter
		if !userInTop {
			var currentUser models.Player
			if err := s.db.First(&currentUser, *currentUserID).Error; err == nil {
				players = append(players, currentUser)
			}
		}
	}

	return players, nil
}

func (s *PlayerService) GetTopPlayersByTeamElo(limit int, currentUserID *uint) ([]models.Player, error) {
	var players []models.Player

	result := s.db.Order("team_elo_rating DESC").
		Limit(limit).
		Find(&players)

	if result.Error != nil {
		return nil, result.Error
	}

	// Si currentUserID est fourni et que l'utilisateur n'est pas dans le top, l'ajouter
	if currentUserID != nil {
		// Vérifier si l'utilisateur connecté est déjà dans la liste
		userInTop := false
		for _, player := range players {
			if player.ID == *currentUserID {
				userInTop = true
				break
			}
		}

		// Si l'utilisateur n'est pas dans le top, le récupérer et l'ajouter
		if !userInTop {
			var currentUser models.Player
			if err := s.db.First(&currentUser, *currentUserID).Error; err == nil {
				players = append(players, currentUser)
			}
		}
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
		"created_at": true,
		"elo_rating": true,
		"username":   true,
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

// RecalculateAllRanks recalcule les rangs de tous les joueurs basés sur leur ELO
// Gère les égalités : joueurs avec même ELO ont le même rang
func (s *PlayerService) RecalculateAllRanks() error {
	// Récupérer tous les joueurs triés par ELO décroissant
	var players []models.Player
	if err := s.db.Order("elo_rating DESC, id ASC").Find(&players).Error; err != nil {
		return err
	}

	// Calculer les rangs avec gestion des égalités
	currentRank := 1
	var previousElo float64

	for i, player := range players {
		// Si ce n'est pas le premier joueur et que l'ELO est différent du précédent
		if i > 0 && player.EloRating != previousElo {
			currentRank = i + 1
		}

		// Mettre à jour le rang du joueur
		if err := s.db.Model(&player).Update("rank", currentRank).Error; err != nil {
			return err
		}

		previousElo = player.EloRating
	}

	return nil
}
