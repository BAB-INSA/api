package services

import (
	"core/models"
	"core/utils"
	"errors"
	"time"

	"gorm.io/gorm"
)

type TeamMatchService struct {
	db            *gorm.DB
	teamService   *TeamService
	playerService *PlayerService
}

func NewTeamMatchService(db *gorm.DB) *TeamMatchService {
	return &TeamMatchService{
		db:            db,
		teamService:   NewTeamService(db),
		playerService: NewPlayerService(db),
	}
}

func (s *TeamMatchService) GetRecentTeamMatches(limit int) ([]models.TeamMatch, error) {
	var matches []models.TeamMatch

	result := s.db.Order("created_at DESC").
		Limit(limit).
		Preload("Team1").
		Preload("Team1.Player1").
		Preload("Team1.Player2").
		Preload("Team2").
		Preload("Team2.Player1").
		Preload("Team2.Player2").
		Preload("WinnerTeam").
		Preload("WinnerTeam.Player1").
		Preload("WinnerTeam.Player2").
		Find(&matches)

	if result.Error != nil {
		return nil, result.Error
	}

	return matches, nil
}

type TeamMatchFilters struct {
	TeamID   *uint      `json:"team_id,omitempty"`
	PlayerID *uint      `json:"player_id,omitempty"`
	Status   *string    `json:"status,omitempty"`
	DateFrom *time.Time `json:"date_from,omitempty"`
	DateTo   *time.Time `json:"date_to,omitempty"`
	Page     int        `json:"page"`
	PerPage  int        `json:"per_page"`
}

func (s *TeamMatchService) GetTeamMatches(filters TeamMatchFilters) (*models.PaginatedTeamMatchResponse, error) {
	var matches []models.TeamMatch
	var total int64

	// Build query
	query := s.db.Model(&models.TeamMatch{})

	// Apply filters
	if filters.TeamID != nil {
		query = query.Where("team1_id = ? OR team2_id = ?", *filters.TeamID, *filters.TeamID)
	}

	if filters.PlayerID != nil {
		// Find teams that include this player
		var playerTeams []models.Team
		if err := s.db.Where("player1_id = ? OR player2_id = ?", *filters.PlayerID, *filters.PlayerID).Find(&playerTeams).Error; err != nil {
			return nil, err
		}

		if len(playerTeams) > 0 {
			var teamIDs []uint
			for _, team := range playerTeams {
				teamIDs = append(teamIDs, team.ID)
			}
			query = query.Where("team1_id IN ? OR team2_id IN ?", teamIDs, teamIDs)
		} else {
			// No teams found for this player, return empty result
			return &models.PaginatedTeamMatchResponse{
				Data:       []models.TeamMatch{},
				Total:      0,
				Page:       filters.Page,
				PageSize:   filters.PerPage,
				TotalPages: 0,
			}, nil
		}
	}

	if filters.Status != nil {
		query = query.Where("status = ?", *filters.Status)
	}

	if filters.DateFrom != nil {
		query = query.Where("created_at >= ?", *filters.DateFrom)
	}

	if filters.DateTo != nil {
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
		Preload("Team1").
		Preload("Team1.Player1").
		Preload("Team1.Player2").
		Preload("Team2").
		Preload("Team2.Player1").
		Preload("Team2.Player2").
		Preload("WinnerTeam").
		Preload("WinnerTeam.Player1").
		Preload("WinnerTeam.Player2").
		Find(&matches)

	if result.Error != nil {
		return nil, result.Error
	}

	// Calculate total pages
	totalPages := int((total + int64(filters.PerPage) - 1) / int64(filters.PerPage))

	return &models.PaginatedTeamMatchResponse{
		Data:       matches,
		Total:      total,
		Page:       filters.Page,
		PageSize:   filters.PerPage,
		TotalPages: totalPages,
	}, nil
}

func (s *TeamMatchService) CreateTeamMatch(req models.CreateTeamMatchRequest) (*models.TeamMatch, error) {
	// Validate that teams exist
	team1, err := s.teamService.GetTeamByID(req.Team1ID)
	if err != nil {
		return nil, errors.New("team1 not found")
	}

	team2, err := s.teamService.GetTeamByID(req.Team2ID)
	if err != nil {
		return nil, errors.New("team2 not found")
	}

	// Validate that teams are different
	if req.Team1ID == req.Team2ID {
		return nil, errors.New("team1 and team2 must be different")
	}

	// Validate that winner is one of the teams
	if req.WinnerTeamID != req.Team1ID && req.WinnerTeamID != req.Team2ID {
		return nil, errors.New("winner must be either team1 or team2")
	}

	// Check for overlapping players
	if team1.Player1ID == team2.Player1ID || team1.Player1ID == team2.Player2ID ||
		team1.Player2ID == team2.Player1ID || team1.Player2ID == team2.Player2ID {
		return nil, errors.New("teams cannot share players")
	}

	// Start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Create the team match in pending status
	now := time.Now()
	match := models.TeamMatch{
		Team1ID:      req.Team1ID,
		Team2ID:      req.Team2ID,
		WinnerTeamID: req.WinnerTeamID,
		Status:       "pending",
		CreatedAt:    now,
	}

	if err := tx.Create(&match).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	// Load the created match with relationships
	if err := s.db.Preload("Team1").Preload("Team1.Player1").Preload("Team1.Player2").
		Preload("Team2").Preload("Team2.Player1").Preload("Team2.Player2").
		Preload("WinnerTeam").Preload("WinnerTeam.Player1").Preload("WinnerTeam.Player2").
		First(&match, match.ID).Error; err != nil {
		return nil, err
	}

	return &match, nil
}

func (s *TeamMatchService) UpdateTeamMatchStatus(matchID uint, req models.UpdateTeamMatchStatusRequest) (*models.TeamMatch, error) {
	// Start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Get the match
	var match models.TeamMatch
	if err := tx.Preload("Team1").Preload("Team1.Player1").Preload("Team1.Player2").
		Preload("Team2").Preload("Team2.Player1").Preload("Team2.Player2").
		First(&match, matchID).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("team match not found")
		}
		return nil, err
	}

	// Check if match is still pending
	if match.Status != "pending" {
		tx.Rollback()
		return nil, errors.New("team match is not pending")
	}

	// Update winner_team_id if provided
	if req.WinnerTeamID != nil {
		if *req.WinnerTeamID != match.Team1ID && *req.WinnerTeamID != match.Team2ID {
			tx.Rollback()
			return nil, errors.New("winner must be either team1 or team2")
		}
		match.WinnerTeamID = *req.WinnerTeamID
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

	// If confirmed, calculate team ELO and update stats
	if match.Status == "confirmed" {
		if err := s.updateTeamEloAndStats(tx, &match, now); err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	// If match was confirmed, recalculate team ranks
	if match.Status == "confirmed" {
		if err := s.recalculateTeamRanks(); err != nil {
			// Log error but don't fail the request
		}
	}

	// Load the updated match with relationships
	if err := s.db.Preload("Team1").Preload("Team1.Player1").Preload("Team1.Player2").
		Preload("Team2").Preload("Team2.Player1").Preload("Team2.Player2").
		Preload("WinnerTeam").Preload("WinnerTeam.Player1").Preload("WinnerTeam.Player2").
		First(&match, match.ID).Error; err != nil {
		return nil, err
	}

	return &match, nil
}

func (s *TeamMatchService) updateTeamEloAndStats(tx *gorm.DB, match *models.TeamMatch, now time.Time) error {
	// Calculate team averages
	team1AvgElo := utils.CalculateTeamAverageElo(match.Team1.Player1.TeamEloRating, match.Team1.Player2.TeamEloRating)
	team2AvgElo := utils.CalculateTeamAverageElo(match.Team2.Player1.TeamEloRating, match.Team2.Player2.TeamEloRating)

	isTeam1Winner := match.WinnerTeamID == match.Team1ID

	// Calculate ELO changes for each player
	team1Player1Change := utils.CalculateTeamEloChange(match.Team1.Player1.TeamEloRating, team2AvgElo, isTeam1Winner)
	team1Player2Change := utils.CalculateTeamEloChange(match.Team1.Player2.TeamEloRating, team2AvgElo, isTeam1Winner)
	team2Player1Change := utils.CalculateTeamEloChange(match.Team2.Player1.TeamEloRating, team1AvgElo, !isTeam1Winner)
	team2Player2Change := utils.CalculateTeamEloChange(match.Team2.Player2.TeamEloRating, team1AvgElo, !isTeam1Winner)

	// Calculate team ELO changes (average of the two players' changes)
	team1EloChange := (team1Player1Change + team1Player2Change) / 2.0
	team2EloChange := (team2Player1Change + team2Player2Change) / 2.0

	// Create ELO history entries
	eloHistories := []models.EloHistory{
		{
			PlayerID:       match.Team1.Player1.ID,
			MatchID:        match.ID,
			EloBefore:      match.Team1.Player1.TeamEloRating,
			EloAfter:       match.Team1.Player1.TeamEloRating + team1Player1Change,
			EloChange:      team1Player1Change,
			OpponentTeamID: &match.Team2ID,
			MatchType:      "team",
			CreatedAt:      now,
		},
		{
			PlayerID:       match.Team1.Player2.ID,
			MatchID:        match.ID,
			EloBefore:      match.Team1.Player2.TeamEloRating,
			EloAfter:       match.Team1.Player2.TeamEloRating + team1Player2Change,
			EloChange:      team1Player2Change,
			OpponentTeamID: &match.Team2ID,
			MatchType:      "team",
			CreatedAt:      now,
		},
		{
			PlayerID:       match.Team2.Player1.ID,
			MatchID:        match.ID,
			EloBefore:      match.Team2.Player1.TeamEloRating,
			EloAfter:       match.Team2.Player1.TeamEloRating + team2Player1Change,
			EloChange:      team2Player1Change,
			OpponentTeamID: &match.Team1ID,
			MatchType:      "team",
			CreatedAt:      now,
		},
		{
			PlayerID:       match.Team2.Player2.ID,
			MatchID:        match.ID,
			EloBefore:      match.Team2.Player2.TeamEloRating,
			EloAfter:       match.Team2.Player2.TeamEloRating + team2Player2Change,
			EloChange:      team2Player2Change,
			OpponentTeamID: &match.Team1ID,
			MatchType:      "team",
			CreatedAt:      now,
		},
	}

	for _, eloHistory := range eloHistories {
		if err := tx.Create(&eloHistory).Error; err != nil {
			return err
		}
	}

	// Update player team ELO ratings and stats
	playerUpdates := map[uint]map[string]interface{}{
		match.Team1.Player1.ID: {
			"team_elo_rating":    match.Team1.Player1.TeamEloRating + team1Player1Change,
			"team_total_matches": match.Team1.Player1.TeamTotalMatches + 1,
		},
		match.Team1.Player2.ID: {
			"team_elo_rating":    match.Team1.Player2.TeamEloRating + team1Player2Change,
			"team_total_matches": match.Team1.Player2.TeamTotalMatches + 1,
		},
		match.Team2.Player1.ID: {
			"team_elo_rating":    match.Team2.Player1.TeamEloRating + team2Player1Change,
			"team_total_matches": match.Team2.Player1.TeamTotalMatches + 1,
		},
		match.Team2.Player2.ID: {
			"team_elo_rating":    match.Team2.Player2.TeamEloRating + team2Player2Change,
			"team_total_matches": match.Team2.Player2.TeamTotalMatches + 1,
		},
	}

	// Update wins for winners
	if isTeam1Winner {
		playerUpdates[match.Team1.Player1.ID]["team_wins"] = match.Team1.Player1.TeamWins + 1
		playerUpdates[match.Team1.Player2.ID]["team_wins"] = match.Team1.Player2.TeamWins + 1
		playerUpdates[match.Team2.Player1.ID]["team_losses"] = match.Team2.Player1.TeamLosses + 1
		playerUpdates[match.Team2.Player2.ID]["team_losses"] = match.Team2.Player2.TeamLosses + 1
	} else {
		playerUpdates[match.Team2.Player1.ID]["team_wins"] = match.Team2.Player1.TeamWins + 1
		playerUpdates[match.Team2.Player2.ID]["team_wins"] = match.Team2.Player2.TeamWins + 1
		playerUpdates[match.Team1.Player1.ID]["team_losses"] = match.Team1.Player1.TeamLosses + 1
		playerUpdates[match.Team1.Player2.ID]["team_losses"] = match.Team1.Player2.TeamLosses + 1
	}

	// Apply all updates
	for playerID, updates := range playerUpdates {
		if err := tx.Model(&models.Player{}).Where("id = ?", playerID).Updates(updates).Error; err != nil {
			return err
		}
	}

	// Update team statistics
	if err := s.teamService.UpdateTeamStats(match.Team1ID, isTeam1Winner, team1EloChange); err != nil {
		return err
	}

	if err := s.teamService.UpdateTeamStats(match.Team2ID, !isTeam1Winner, team2EloChange); err != nil {
		return err
	}

	return nil
}

func (s *TeamMatchService) recalculateTeamRanks() error {
	// Get all players sorted by team ELO rating descending
	var players []models.Player
	if err := s.db.Order("team_elo_rating DESC, id ASC").Find(&players).Error; err != nil {
		return err
	}

	// Calculate ranks with tie handling
	currentRank := 1
	var previousElo float64

	for i, player := range players {
		if i > 0 && player.TeamEloRating != previousElo {
			currentRank = i + 1
		}

		if err := s.db.Model(&player).Update("team_rank", currentRank).Error; err != nil {
			return err
		}

		previousElo = player.TeamEloRating
	}

	return nil
}

func (s *TeamMatchService) ConfirmTeamMatch(matchID uint) (*models.TeamMatch, error) {
	status := "confirmed"
	confirmRequest := models.UpdateTeamMatchStatusRequest{
		Status: &status,
	}
	return s.UpdateTeamMatchStatus(matchID, confirmRequest)
}

func (s *TeamMatchService) RejectTeamMatch(matchID uint) (*models.TeamMatch, error) {
	status := "rejected"
	rejectRequest := models.UpdateTeamMatchStatusRequest{
		Status: &status,
	}
	return s.UpdateTeamMatchStatus(matchID, rejectRequest)
}

func (s *TeamMatchService) CancelTeamMatch(matchID uint) (*models.TeamMatch, error) {
	var match models.TeamMatch
	if err := s.db.First(&match, matchID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("team match not found")
		}
		return nil, err
	}

	match.Status = "cancelled"
	if err := s.db.Save(&match).Error; err != nil {
		return nil, err
	}

	// Load with relationships
	if err := s.db.Preload("Team1").Preload("Team1.Player1").Preload("Team1.Player2").
		Preload("Team2").Preload("Team2.Player1").Preload("Team2.Player2").
		Preload("WinnerTeam").Preload("WinnerTeam.Player1").Preload("WinnerTeam.Player2").
		First(&match, match.ID).Error; err != nil {
		return nil, err
	}

	return &match, nil
}
