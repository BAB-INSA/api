package services

import (
	"core/models"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"gorm.io/gorm"
)

type TournamentService struct {
	db *gorm.DB
}

func NewTournamentService(db *gorm.DB) *TournamentService {
	return &TournamentService{
		db: db,
	}
}

func (s *TournamentService) CreateTournament(req models.CreateTournamentRequest) (*models.Tournament, error) {
	slug := s.generateUniqueSlug(req.Name)

	tournament := &models.Tournament{
		Name:        req.Name,
		Slug:        slug,
		Type:        req.Type,
		Status:      "opened",
		Description: req.Description,
	}

	if err := s.db.Create(tournament).Error; err != nil {
		return nil, err
	}

	return tournament, nil
}

func (s *TournamentService) GetTournamentByID(id uint) (*models.TournamentListItem, error) {
	var tournament models.TournamentListItem

	result := s.db.First(&tournament, id)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("tournament not found")
		}
		return nil, result.Error
	}

	return &tournament, nil
}

func (s *TournamentService) GetTournamentBySlug(slug string) (*models.TournamentListItem, error) {
	var tournament models.TournamentListItem

	result := s.db.Where("slug = ?", slug).First(&tournament)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("tournament not found")
		}
		return nil, result.Error
	}

	return &tournament, nil
}

func (s *TournamentService) GetAllTournaments(page, pageSize int, status *string, tournamentType *string) (*models.PaginatedTournamentsResponse, error) {
	var tournaments []models.TournamentListItem
	var total int64

	query := s.db.Model(&models.Tournament{})

	if status != nil {
		query = query.Where("status = ?", *status)
	}

	if tournamentType != nil {
		query = query.Where("type = ?", *tournamentType)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	offset := (page - 1) * pageSize

	if err := query.
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&tournaments).Error; err != nil {
		return nil, err
	}

	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	return &models.PaginatedTournamentsResponse{
		Data:       tournaments,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

func (s *TournamentService) UpdateTournament(id uint, req models.UpdateTournamentRequest) (*models.TournamentListItem, error) {
	tournament, err := s.GetTournamentByID(id)
	if err != nil {
		return nil, err
	}

	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Status != nil {
		validTransitions := map[string]string{
			"opened":  "ongoing",
			"ongoing": "finished",
		}
		expected, ok := validTransitions[tournament.Status]
		if !ok || *req.Status != expected {
			return nil, fmt.Errorf("cannot change status from %s to %s", tournament.Status, *req.Status)
		}
		updates["status"] = *req.Status
	}

	if len(updates) > 0 {
		if err := s.db.Model(&models.Tournament{}).Where("id = ?", id).Updates(updates).Error; err != nil {
			return nil, err
		}
	}

	return s.GetTournamentByID(id)
}

func (s *TournamentService) JoinTournament(tournamentID, teamID, userID uint) (*models.TournamentTeam, error) {
	// 1. Tournament must exist and be opened
	var tournament models.Tournament
	if err := s.db.First(&tournament, tournamentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("tournament not found")
		}
		return nil, err
	}

	if tournament.Status != "opened" {
		return nil, errors.New("tournament is not open for registration")
	}

	// 2. Team must exist
	var team models.Team
	if err := s.db.First(&team, teamID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("team not found")
		}
		return nil, err
	}

	// 3. User must be a member of the team
	if team.Player1ID != userID && team.Player2ID != userID {
		return nil, errors.New("you must be a member of the team to join a tournament")
	}

	// 4. Team must not already be registered
	var existingEntry models.TournamentTeam
	if err := s.db.Where("tournament_id = ? AND team_id = ?", tournamentID, teamID).First(&existingEntry).Error; err == nil {
		return nil, errors.New("team is already registered in this tournament")
	}

	// 5. No player from this team should already be in the tournament via another team
	var existingTeam models.Team
	err := s.db.Model(&models.Team{}).
		Joins("JOIN tournament_teams ON tournament_teams.team_id = teams.id").
		Where("tournament_teams.tournament_id = ? AND tournament_teams.deleted_at IS NULL AND (teams.player1_id IN (?, ?) OR teams.player2_id IN (?, ?))",
			tournamentID, team.Player1ID, team.Player2ID, team.Player1ID, team.Player2ID).
		First(&existingTeam).Error

	if err == nil {
		return nil, fmt.Errorf("already registered in this tournament with team %s", existingTeam.Name)
	}

	tournamentTeam := &models.TournamentTeam{
		TournamentID: tournamentID,
		TeamID:       teamID,
	}

	if err := s.db.Create(tournamentTeam).Error; err != nil {
		return nil, err
	}

	// Increment nb_participants
	s.db.Model(&models.Tournament{}).Where("id = ?", tournamentID).
		Update("nb_participants", gorm.Expr("nb_participants + 1"))

	// Load relationships
	if err := s.db.
		Preload("Team").
		Preload("Team.Player1").
		Preload("Team.Player2").
		First(tournamentTeam, tournamentTeam.ID).Error; err != nil {
		return nil, err
	}

	return tournamentTeam, nil
}

func (s *TournamentService) LeaveTournament(tournamentID, teamID, userID uint) error {
	// Tournament must exist and be opened
	var tournament models.Tournament
	if err := s.db.First(&tournament, tournamentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("tournament not found")
		}
		return err
	}

	if tournament.Status != "opened" {
		return errors.New("tournament is not open for registration")
	}

	// Team must exist
	var team models.Team
	if err := s.db.First(&team, teamID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("team not found")
		}
		return err
	}

	// User must be a member of the team
	if team.Player1ID != userID && team.Player2ID != userID {
		return errors.New("you must be a member of the team to leave a tournament")
	}

	// Find and delete the registration
	result := s.db.Where("tournament_id = ? AND team_id = ?", tournamentID, teamID).Delete(&models.TournamentTeam{})
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("team is not registered in this tournament")
	}

	// Decrement nb_participants
	s.db.Model(&models.Tournament{}).Where("id = ? AND nb_participants > 0", tournamentID).
		Update("nb_participants", gorm.Expr("nb_participants - 1"))

	return nil
}

func (s *TournamentService) GetTournamentTeams(tournamentID uint, page, pageSize int) (*models.PaginatedTournamentTeamsResponse, error) {
	var tournamentTeams []models.TournamentTeam
	var total int64

	query := s.db.Model(&models.TournamentTeam{}).Where("tournament_id = ?", tournamentID)

	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	offset := (page - 1) * pageSize

	if err := s.db.Where("tournament_id = ?", tournamentID).
		Preload("Team").
		Preload("Team.Player1").
		Preload("Team.Player2").
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&tournamentTeams).Error; err != nil {
		return nil, err
	}

	items := make([]models.TournamentTeamItem, len(tournamentTeams))
	for i, tt := range tournamentTeams {
		items[i] = models.TournamentTeamItem{
			ID:     tt.ID,
			TeamID: tt.TeamID,
			Wins:   tt.Wins,
			Losses: tt.Losses,
			Team:   tt.Team,
		}
	}

	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	return &models.PaginatedTournamentTeamsResponse{
		Data:       items,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// UpdateTournamentTeamStats updates wins/losses for a team in a tournament
func (s *TournamentService) UpdateTournamentTeamStats(tournamentID, teamID uint, won bool) error {
	updates := map[string]interface{}{}
	if won {
		updates["wins"] = gorm.Expr("wins + 1")
	} else {
		updates["losses"] = gorm.Expr("losses + 1")
	}

	result := s.db.Model(&models.TournamentTeam{}).
		Where("tournament_id = ? AND team_id = ?", tournamentID, teamID).
		Updates(updates)

	return result.Error
}

// IncrementTournamentNbMatches increments the match counter on a tournament
func (s *TournamentService) IncrementTournamentNbMatches(tournamentID uint) error {
	return s.db.Model(&models.Tournament{}).Where("id = ?", tournamentID).
		Update("nb_matches", gorm.Expr("nb_matches + 1")).Error
}

func (s *TournamentService) GetTournamentMatches(tournamentID uint, page, pageSize int) (*models.PaginatedTeamMatchResponse, error) {
	var matches []models.TeamMatch
	var total int64

	query := s.db.Model(&models.TeamMatch{}).Where("tournament_id = ?", tournamentID)

	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	offset := (page - 1) * pageSize

	if err := s.db.Where("tournament_id = ?", tournamentID).
		Preload("Team1").
		Preload("Team1.Player1").
		Preload("Team1.Player2").
		Preload("Team2").
		Preload("Team2.Player1").
		Preload("Team2.Player2").
		Preload("WinnerTeam").
		Preload("WinnerTeam.Player1").
		Preload("WinnerTeam.Player2").
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&matches).Error; err != nil {
		return nil, err
	}

	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	return &models.PaginatedTeamMatchResponse{
		Data:       matches,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

func (s *TournamentService) DeleteTournament(id uint) error {
	result := s.db.Delete(&models.Tournament{}, id)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("tournament not found")
	}

	return nil
}

func (s *TournamentService) generateSlug(name string) string {
	slug := strings.ToLower(name)

	reg := regexp.MustCompile(`[^a-z0-9]+`)
	slug = reg.ReplaceAllString(slug, "-")

	slug = strings.Trim(slug, "-")

	return slug
}

func (s *TournamentService) generateUniqueSlug(name string) string {
	baseSlug := s.generateSlug(name)
	slug := baseSlug
	counter := 1

	for {
		var existing models.Tournament
		result := s.db.Where("slug = ?", slug).First(&existing)

		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			break
		}

		slug = fmt.Sprintf("%s-%d", baseSlug, counter)
		counter++
	}

	return slug
}
