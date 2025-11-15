package services

import (
	"core/models"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"gorm.io/gorm"
)

type TeamService struct {
	db *gorm.DB
}

func NewTeamService(db *gorm.DB) *TeamService {
	return &TeamService{
		db: db,
	}
}

func (s *TeamService) CreateTeam(player1ID, player2ID uint, name string) (*models.Team, error) {
	if player1ID == player2ID {
		return nil, errors.New("a team cannot have the same player twice")
	}

	// Check if team already exists
	_, err := s.GetTeamByPlayers(player1ID, player2ID)
	if err == nil {
		return nil, errors.New("team already exists")
	}

	// Check if both players exist
	var player1, player2 models.Player
	if err := s.db.First(&player1, player1ID).Error; err != nil {
		return nil, errors.New("player1 not found")
	}
	if err := s.db.First(&player2, player2ID).Error; err != nil {
		return nil, errors.New("player2 not found")
	}

	// Generate default name if not provided
	if name == "" {
		name = fmt.Sprintf("%s & %s", player1.Username, player2.Username)
	}

	// Generate unique slug
	slug := s.generateUniqueSlug(name)

	team := &models.Team{
		Player1ID:    player1ID,
		Player2ID:    player2ID,
		Name:         name,
		Slug:         slug,
		EloRating:    1200,
		TotalMatches: 0,
		Wins:         0,
		Losses:       0,
	}

	result := s.db.Create(team)
	if result.Error != nil {
		return nil, result.Error
	}

	// Load relationships
	if err := s.db.Preload("Player1").Preload("Player2").First(team, team.ID).Error; err != nil {
		return nil, err
	}

	return team, nil
}

func (s *TeamService) GetTeamByID(id uint) (*models.Team, error) {
	var team models.Team

	result := s.db.Preload("Player1").Preload("Player2").First(&team, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("team not found")
		}
		return nil, result.Error
	}

	return &team, nil
}

func (s *TeamService) GetTeamByPlayers(player1ID, player2ID uint) (*models.Team, error) {
	var team models.Team

	// Try both combinations (player1, player2) and (player2, player1)
	result := s.db.Where("(player1_id = ? AND player2_id = ?) OR (player1_id = ? AND player2_id = ?)",
		player1ID, player2ID, player2ID, player1ID).
		Preload("Player1").Preload("Player2").
		First(&team)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("team not found")
		}
		return nil, result.Error
	}

	return &team, nil
}

func (s *TeamService) UpdateTeam(id uint, name *string) (*models.Team, error) {
	team, err := s.GetTeamByID(id)
	if err != nil {
		return nil, err
	}

	updates := make(map[string]interface{})
	if name != nil {
		updates["name"] = *name
	}

	if len(updates) > 0 {
		if err := s.db.Model(team).Updates(updates).Error; err != nil {
			return nil, err
		}
	}

	return s.GetTeamByID(id)
}

func (s *TeamService) DeleteTeam(id uint) error {
	result := s.db.Delete(&models.Team{}, id)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("team not found")
	}

	return nil
}

func (s *TeamService) GetAllTeams(page int, pageSize int) (*models.PaginatedTeamsResponse, error) {
	var teams []models.Team
	var total int64

	// Count total records
	if err := s.db.Model(&models.Team{}).Count(&total).Error; err != nil {
		return nil, err
	}

	// Calculate offset
	offset := (page - 1) * pageSize

	// Get paginated teams
	if err := s.db.Preload("Player1").Preload("Player2").
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&teams).Error; err != nil {
		return nil, err
	}

	// Calculate total pages
	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	return &models.PaginatedTeamsResponse{
		Data:       teams,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

func (s *TeamService) GetTeamsByPlayer(playerID uint, page int, pageSize int) (*models.PaginatedTeamsResponse, error) {
	var teams []models.Team
	var total int64

	baseQuery := s.db.Model(&models.Team{}).Where("player1_id = ? OR player2_id = ?", playerID, playerID)

	// Count total records
	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, err
	}

	// Calculate offset
	offset := (page - 1) * pageSize

	// Get paginated teams
	if err := baseQuery.Preload("Player1").Preload("Player2").
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&teams).Error; err != nil {
		return nil, err
	}

	// Calculate total pages
	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	return &models.PaginatedTeamsResponse{
		Data:       teams,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

func (s *TeamService) GetAllTeamsByPlayer(playerID uint) ([]models.Team, error) {
	var teams []models.Team

	result := s.db.Where("player1_id = ? OR player2_id = ?", playerID, playerID).
		Preload("Player1").Preload("Player2").
		Order("created_at DESC").
		Find(&teams)

	if result.Error != nil {
		return nil, result.Error
	}

	return teams, nil
}

func (s *TeamService) GetTeamAverageElo(teamID uint) (float64, error) {
	team, err := s.GetTeamByID(teamID)
	if err != nil {
		return 0, err
	}

	return (team.Player1.TeamEloRating + team.Player2.TeamEloRating) / 2.0, nil
}

func (s *TeamService) UpdateTeamStats(teamID uint, won bool, eloChange float64) error {
	team, err := s.GetTeamByID(teamID)
	if err != nil {
		return err
	}

	updates := map[string]interface{}{
		"total_matches": team.TotalMatches + 1,
		"elo_rating":    team.EloRating + eloChange,
	}

	if won {
		updates["wins"] = team.Wins + 1
	} else {
		updates["losses"] = team.Losses + 1
	}

	return s.db.Model(team).Updates(updates).Error
}

func (s *TeamService) GetTopTeamsByElo(limit int) ([]models.Team, error) {
	var teams []models.Team

	result := s.db.Preload("Player1").Preload("Player2").
		Order("elo_rating DESC").
		Limit(limit).
		Find(&teams)

	if result.Error != nil {
		return nil, result.Error
	}

	return teams, nil
}

func (s *TeamService) generateSlug(name string) string {
	// Convert to lowercase
	slug := strings.ToLower(name)

	// Replace spaces and special characters with hyphens
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	slug = reg.ReplaceAllString(slug, "-")

	// Remove leading and trailing hyphens
	slug = strings.Trim(slug, "-")

	return slug
}

func (s *TeamService) generateUniqueSlug(name string) string {
	baseSlug := s.generateSlug(name)
	slug := baseSlug
	counter := 1

	for {
		var existingTeam models.Team
		result := s.db.Where("slug = ?", slug).First(&existingTeam)

		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			break
		}

		slug = fmt.Sprintf("%s-%d", baseSlug, counter)
		counter++
	}

	return slug
}

func (s *TeamService) GetTeamBySlug(slug string) (*models.Team, error) {
	var team models.Team

	result := s.db.Where("slug = ?", slug).
		Preload("Player1").Preload("Player2").
		First(&team)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("team not found")
		}
		return nil, result.Error
	}

	return &team, nil
}
