package fixtures

import (
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	authModels "auth/models"
	authUtils "auth/utils"
	"core/models"
	coreUtils "core/utils"

	"gorm.io/gorm"
)

type Fixtures struct {
	db *gorm.DB
}

func NewFixtures(db *gorm.DB) *Fixtures {
	return &Fixtures{db: db}
}

// GenerateTestData creates 10 users/players, 50 matches and ELO history
func (f *Fixtures) GenerateTestData() error {
	log.Println("Starting fixtures generation...")

	// Generate users and players
	users, err := f.generateUsers()
	if err != nil {
		return fmt.Errorf("failed to generate users: %w", err)
	}

	// Generate matches
	matches, err := f.generateMatches(users)
	if err != nil {
		return fmt.Errorf("failed to generate matches: %w", err)
	}

	// Generate ELO history and update win streaks
	err = f.generateEloHistoryAndStreaks(users, matches)
	if err != nil {
		return fmt.Errorf("failed to generate ELO history and streaks: %w", err)
	}

	// Generate teams
	teams, err := f.generateTeams(users)
	if err != nil {
		return fmt.Errorf("failed to generate teams: %w", err)
	}

	// Generate team matches
	teamMatches, err := f.generateTeamMatches(teams)
	if err != nil {
		return fmt.Errorf("failed to generate team matches: %w", err)
	}

	// Generate team ELO history
	err = f.generateTeamEloHistory(teams, teamMatches)
	if err != nil {
		return fmt.Errorf("failed to generate team ELO history: %w", err)
	}

	// Recalculate all ranks after ELO updates
	err = f.recalculateAllRanks()
	if err != nil {
		return fmt.Errorf("failed to recalculate ranks: %w", err)
	}

	// Generate tournaments
	err = f.generateTournaments(teams)
	if err != nil {
		return fmt.Errorf("failed to generate tournaments: %w", err)
	}

	log.Println("Fixtures generated successfully!")
	log.Printf("Created %d users, %d matches, %d teams, %d team matches and ELO history", len(users), len(matches), len(teams), len(teamMatches))
	return nil
}

func (f *Fixtures) generateUsers() ([]authModels.User, error) {
	usernames := []string{
		"alexandre", "marie", "julien", "sophie", "thomas",
		"camille", "nicolas", "laura", "antoine", "emma",
	}

	var users []authModels.User

	for i, username := range usernames {
		hashedPassword, err := authUtils.HashPassword("password123")
		if err != nil {
			return nil, err
		}

		email := fmt.Sprintf("%s@bab-insa.fr", username)
		slug := strings.ToLower(username)
		if i < 0 || i >= len(usernames) {
			return nil, fmt.Errorf("invalid user index: %d", i)
		}
		userID := uint(i + 1) // #nosec G115 -- Force IDs 1, 2, 3, ...

		user := authModels.User{
			ID:          userID,
			Email:       email,
			Username:    username,
			Slug:        slug,
			Password:    hashedPassword,
			Enabled:     true,
			NbConnexion: rand.Intn(50) + 1, // #nosec G404
			Roles:       authModels.GetDefaultRoles(),
		}

		if err := f.db.Create(&user).Error; err != nil {
			return nil, err
		}

		// Create corresponding player with fixed ELO based on index
		baseElos := []float64{1200, 1250, 1180, 1320, 1150, 1280, 1220, 1350, 1100, 1300}
		teamBaseElos := []float64{1180, 1280, 1200, 1300, 1160, 1260, 1240, 1380, 1120, 1320}
		player := models.Player{
			ID:               userID, // Same ID as user
			Username:         user.Username,
			EloRating:        baseElos[i],
			TeamEloRating:    teamBaseElos[i],
			TotalMatches:     0, // Will be updated when matches are created
			Wins:             0,
			Losses:           0,
			TeamTotalMatches: 0,
			TeamWins:         0,
			TeamLosses:       0,
		}

		if err := f.db.Create(&player).Error; err != nil {
			return nil, err
		}

		users = append(users, user)
		log.Printf("Created user: %s (ID: %d) -> Player (ID: %d, ELO: %.0f)", username, userID, userID, player.EloRating)
	}

	return users, nil
}

func (f *Fixtures) generateMatches(users []authModels.User) ([]models.Match, error) {
	var matches []models.Match

	// Generate 50 matches over the last 30 days
	now := time.Now()

	for i := 0; i < 50; i++ {
		// Random date in the last 30 days
		daysAgo := rand.Intn(30)                                                               // #nosec G404
		matchDate := now.AddDate(0, 0, -daysAgo).Add(time.Duration(rand.Intn(24)) * time.Hour) // #nosec G404

		// Pick two random different players
		player1 := users[rand.Intn(len(users))] // #nosec G404
		var player2 authModels.User
		for {
			player2 = users[rand.Intn(len(users))] // #nosec G404
			if player2.ID != player1.ID {
				break
			}
		}

		// Random winner
		var winner uint
		winRoll := rand.Float32() // #nosec G404
		if winRoll < 0.5 {
			winner = player1.ID
		} else {
			winner = player2.ID
		}

		// Random status (most confirmed)
		status := "confirmed"
		confirmedAt := &matchDate
		statusRoll := rand.Float32() // #nosec G404
		if statusRoll < 0.1 {
			status = "pending"
			confirmedAt = nil
		}

		match := models.Match{
			Player1ID:   player1.ID,
			Player2ID:   player2.ID,
			WinnerID:    winner,
			Status:      status,
			CreatedAt:   matchDate,
			ConfirmedAt: confirmedAt,
		}

		if err := f.db.Create(&match).Error; err != nil {
			return nil, err
		}

		matches = append(matches, match)

		// Don't update player stats here - will be done chronologically later
	}

	log.Printf("Created %d matches", len(matches))
	return matches, nil
}

func (f *Fixtures) generateEloHistoryAndStreaks(users []authModels.User, matches []models.Match) error {
	// Track current ELO and win streaks for each player
	playerElos := make(map[uint]float64)
	playerCurrentStreaks := make(map[uint]int)
	playerBestStreaks := make(map[uint]int)
	playerTotalMatches := make(map[uint]int)
	playerWins := make(map[uint]int)
	playerLosses := make(map[uint]int)

	// Initialize with player's base ELO
	var players []models.Player
	f.db.Find(&players)
	for _, player := range players {
		playerElos[player.ID] = player.EloRating
		playerCurrentStreaks[player.ID] = 0
		playerBestStreaks[player.ID] = 0
		playerTotalMatches[player.ID] = 0
		playerWins[player.ID] = 0
		playerLosses[player.ID] = 0
	}

	// Sort matches by creation date to process chronologically
	var sortedMatches []models.Match
	f.db.Order("created_at ASC").Find(&sortedMatches)

	for _, match := range sortedMatches {
		if match.Status != "confirmed" {
			continue
		}

		// Get current ELOs
		player1Elo := playerElos[match.Player1ID]
		player2Elo := playerElos[match.Player2ID]

		// Calculate ELO changes
		player1Change, player2Change := coreUtils.CalculateEloChange(player1Elo, player2Elo, match.WinnerID, match.Player1ID)

		// Create ELO history entries
		eloHistory1 := models.EloHistory{
			PlayerID:   match.Player1ID,
			MatchID:    match.ID,
			EloBefore:  player1Elo,
			EloAfter:   player1Elo + player1Change,
			EloChange:  player1Change,
			OpponentID: &match.Player2ID,
			CreatedAt:  match.CreatedAt,
		}

		eloHistory2 := models.EloHistory{
			PlayerID:   match.Player2ID,
			MatchID:    match.ID,
			EloBefore:  player2Elo,
			EloAfter:   player2Elo + player2Change,
			EloChange:  player2Change,
			OpponentID: &match.Player1ID,
			CreatedAt:  match.CreatedAt,
		}

		// Save to database
		if err := f.db.Create(&eloHistory1).Error; err != nil {
			return err
		}
		if err := f.db.Create(&eloHistory2).Error; err != nil {
			return err
		}

		// Update player ELOs for next calculation
		playerElos[match.Player1ID] += player1Change
		playerElos[match.Player2ID] += player2Change

		// Update match stats
		playerTotalMatches[match.Player1ID]++
		playerTotalMatches[match.Player2ID]++

		// Update win streaks
		if match.WinnerID == match.Player1ID {
			// Player1 wins
			playerWins[match.Player1ID]++
			playerLosses[match.Player2ID]++

			// Update streaks
			playerCurrentStreaks[match.Player1ID]++
			playerCurrentStreaks[match.Player2ID] = 0

			// Update best streak if necessary
			if playerCurrentStreaks[match.Player1ID] > playerBestStreaks[match.Player1ID] {
				playerBestStreaks[match.Player1ID] = playerCurrentStreaks[match.Player1ID]
			}
		} else {
			// Player2 wins
			playerWins[match.Player2ID]++
			playerLosses[match.Player1ID]++

			// Update streaks
			playerCurrentStreaks[match.Player2ID]++
			playerCurrentStreaks[match.Player1ID] = 0

			// Update best streak if necessary
			if playerCurrentStreaks[match.Player2ID] > playerBestStreaks[match.Player2ID] {
				playerBestStreaks[match.Player2ID] = playerCurrentStreaks[match.Player2ID]
			}
		}
	}

	// Update final stats in players table
	for _, player := range players {
		playerID := player.ID
		f.db.Model(&models.Player{}).Where("id = ?", playerID).Updates(map[string]interface{}{
			"elo_rating":    playerElos[playerID],
			"total_matches": playerTotalMatches[playerID],
			"wins":          playerWins[playerID],
			"losses":        playerLosses[playerID],
		})
	}

	log.Println("Generated ELO history and win streaks for all matches")
	return nil
}

// ClearAllData removes all fixture data
func (f *Fixtures) ClearAllData() error {
	log.Println("Clearing all fixture data...")

	// Delete in correct order due to foreign key constraints
	tables := []interface{}{
		&models.TournamentTeam{},
		&models.Tournament{},
		&models.EloHistory{},
		&models.TeamMatch{},
		&models.Team{},
		&models.Match{},
		&models.Player{},
		&authModels.RefreshToken{},
		&authModels.User{},
	}

	for _, table := range tables {
		if err := f.db.Unscoped().Where("1 = 1").Delete(table).Error; err != nil {
			return fmt.Errorf("failed to clear table %T: %w", table, err)
		}
	}

	// Reset auto-increment sequences to start from 1
	sequences := []string{
		"ALTER SEQUENCE users_id_seq RESTART WITH 1",
		"ALTER SEQUENCE matches_id_seq RESTART WITH 1",
		"ALTER SEQUENCE teams_id_seq RESTART WITH 1",
		"ALTER SEQUENCE team_matches_id_seq RESTART WITH 1",
		"ALTER SEQUENCE elo_history_id_seq RESTART WITH 1",
		"ALTER SEQUENCE refresh_tokens_id_seq RESTART WITH 1",
		"ALTER SEQUENCE tournaments_id_seq RESTART WITH 1",
		"ALTER SEQUENCE tournament_teams_id_seq RESTART WITH 1",
	}

	for _, seq := range sequences {
		f.db.Exec(seq)
	}

	log.Println("All fixture data cleared!")
	return nil
}

// generateTeams creates teams from existing users
func (f *Fixtures) generateTeams(users []authModels.User) ([]models.Team, error) {
	var teams []models.Team

	// Create 8 different teams with various combinations
	teamCombinations := []struct {
		player1Index int
		player2Index int
		name         string
	}{
		{0, 1, "Alex & Marie"},
		{2, 3, "Jul & Sophie"},
		{4, 5, "Tom & Cam"},
		{6, 7, "Nico & Laura"},
		{8, 9, "Antoine & Emma"},
		{0, 4, "Alex & Tom"},
		{1, 6, "Marie & Nico"},
		{3, 7, "Sophie & Laura"},
	}

	for _, combo := range teamCombinations {
		if combo.player1Index >= len(users) || combo.player2Index >= len(users) {
			continue
		}

		// Generate slug from name
		slug := strings.ToLower(strings.ReplaceAll(combo.name, " & ", "-"))
		slug = strings.ReplaceAll(slug, " ", "-")

		team := models.Team{
			Player1ID:    users[combo.player1Index].ID,
			Player2ID:    users[combo.player2Index].ID,
			Name:         combo.name,
			Slug:         slug,
			EloRating:    1200,
			TotalMatches: 0,
			Wins:         0,
			Losses:       0,
		}

		if err := f.db.Create(&team).Error; err != nil {
			return nil, err
		}

		teams = append(teams, team)
		log.Printf("Created team: %s (ID: %d)", combo.name, team.ID)
	}

	log.Printf("Created %d teams", len(teams))
	return teams, nil
}

// generateTeamMatches creates team matches
func (f *Fixtures) generateTeamMatches(teams []models.Team) ([]models.TeamMatch, error) {
	var teamMatches []models.TeamMatch

	// Generate 20 team matches over the last 30 days
	now := time.Now()

	for i := 0; i < 20; i++ {
		// Random date in the last 30 days
		daysAgo := rand.Intn(30)                                                               // #nosec G404
		matchDate := now.AddDate(0, 0, -daysAgo).Add(time.Duration(rand.Intn(24)) * time.Hour) // #nosec G404

		// Pick two random different teams
		team1 := teams[rand.Intn(len(teams))] // #nosec G404
		var team2 models.Team
		for {
			team2 = teams[rand.Intn(len(teams))] // #nosec G404
			if team2.ID != team1.ID {
				break
			}
		}

		// Check for overlapping players
		if team1.Player1ID == team2.Player1ID || team1.Player1ID == team2.Player2ID ||
			team1.Player2ID == team2.Player1ID || team1.Player2ID == team2.Player2ID {
			// Skip if teams share players
			continue
		}

		// Random winner
		var winnerTeamID uint
		winRoll := rand.Float32() // #nosec G404
		if winRoll < 0.5 {
			winnerTeamID = team1.ID
		} else {
			winnerTeamID = team2.ID
		}

		// Random status (most confirmed)
		status := "confirmed"
		confirmedAt := &matchDate
		statusRoll := rand.Float32() // #nosec G404
		if statusRoll < 0.1 {
			status = "pending"
			confirmedAt = nil
		}

		teamMatch := models.TeamMatch{
			Team1ID:      team1.ID,
			Team2ID:      team2.ID,
			WinnerTeamID: winnerTeamID,
			Status:       status,
			CreatedAt:    matchDate,
			ConfirmedAt:  confirmedAt,
		}

		if err := f.db.Create(&teamMatch).Error; err != nil {
			return nil, err
		}

		teamMatches = append(teamMatches, teamMatch)
	}

	log.Printf("Created %d team matches", len(teamMatches))
	return teamMatches, nil
}

// generateTeamEloHistory creates team ELO history and updates team stats
func (f *Fixtures) generateTeamEloHistory(teams []models.Team, teamMatches []models.TeamMatch) error {
	// Track current team ELO for each player
	playerTeamElos := make(map[uint]float64)
	playerTeamTotalMatches := make(map[uint]int)
	playerTeamWins := make(map[uint]int)
	playerTeamLosses := make(map[uint]int)

	// Initialize with player's base team ELO
	var players []models.Player
	f.db.Find(&players)
	for _, player := range players {
		playerTeamElos[player.ID] = player.TeamEloRating
		playerTeamTotalMatches[player.ID] = 0
		playerTeamWins[player.ID] = 0
		playerTeamLosses[player.ID] = 0
	}

	// Sort team matches by creation date to process chronologically
	var sortedTeamMatches []models.TeamMatch
	f.db.Preload("Team1").Preload("Team2").Order("created_at ASC").Find(&sortedTeamMatches)

	for _, teamMatch := range sortedTeamMatches {
		if teamMatch.Status != "confirmed" {
			continue
		}

		// Get team players
		var team1, team2 models.Team
		f.db.First(&team1, teamMatch.Team1ID)
		f.db.First(&team2, teamMatch.Team2ID)

		// Calculate team averages
		team1AvgElo := coreUtils.CalculateTeamAverageElo(
			playerTeamElos[team1.Player1ID],
			playerTeamElos[team1.Player2ID],
		)
		team2AvgElo := coreUtils.CalculateTeamAverageElo(
			playerTeamElos[team2.Player1ID],
			playerTeamElos[team2.Player2ID],
		)

		isTeam1Winner := teamMatch.WinnerTeamID == teamMatch.Team1ID

		// Calculate ELO changes for each player
		team1Player1Change := coreUtils.CalculateTeamEloChange(playerTeamElos[team1.Player1ID], team2AvgElo, isTeam1Winner)
		team1Player2Change := coreUtils.CalculateTeamEloChange(playerTeamElos[team1.Player2ID], team2AvgElo, isTeam1Winner)
		team2Player1Change := coreUtils.CalculateTeamEloChange(playerTeamElos[team2.Player1ID], team1AvgElo, !isTeam1Winner)
		team2Player2Change := coreUtils.CalculateTeamEloChange(playerTeamElos[team2.Player2ID], team1AvgElo, !isTeam1Winner)

		// Create ELO history entries for team match
		eloHistories := []models.EloHistory{
			{
				PlayerID:       team1.Player1ID,
				MatchID:        teamMatch.ID,
				EloBefore:      playerTeamElos[team1.Player1ID],
				EloAfter:       playerTeamElos[team1.Player1ID] + team1Player1Change,
				EloChange:      team1Player1Change,
				OpponentTeamID: &teamMatch.Team2ID,
				MatchType:      "team",
				CreatedAt:      teamMatch.CreatedAt,
			},
			{
				PlayerID:       team1.Player2ID,
				MatchID:        teamMatch.ID,
				EloBefore:      playerTeamElos[team1.Player2ID],
				EloAfter:       playerTeamElos[team1.Player2ID] + team1Player2Change,
				EloChange:      team1Player2Change,
				OpponentTeamID: &teamMatch.Team2ID,
				MatchType:      "team",
				CreatedAt:      teamMatch.CreatedAt,
			},
			{
				PlayerID:       team2.Player1ID,
				MatchID:        teamMatch.ID,
				EloBefore:      playerTeamElos[team2.Player1ID],
				EloAfter:       playerTeamElos[team2.Player1ID] + team2Player1Change,
				EloChange:      team2Player1Change,
				OpponentTeamID: &teamMatch.Team1ID,
				MatchType:      "team",
				CreatedAt:      teamMatch.CreatedAt,
			},
			{
				PlayerID:       team2.Player2ID,
				MatchID:        teamMatch.ID,
				EloBefore:      playerTeamElos[team2.Player2ID],
				EloAfter:       playerTeamElos[team2.Player2ID] + team2Player2Change,
				EloChange:      team2Player2Change,
				OpponentTeamID: &teamMatch.Team1ID,
				MatchType:      "team",
				CreatedAt:      teamMatch.CreatedAt,
			},
		}

		// Save all ELO history entries
		for _, eloHistory := range eloHistories {
			if err := f.db.Create(&eloHistory).Error; err != nil {
				return err
			}
		}

		// Update player team ELOs for next calculation
		playerTeamElos[team1.Player1ID] += team1Player1Change
		playerTeamElos[team1.Player2ID] += team1Player2Change
		playerTeamElos[team2.Player1ID] += team2Player1Change
		playerTeamElos[team2.Player2ID] += team2Player2Change

		// Update team match stats
		playerTeamTotalMatches[team1.Player1ID]++
		playerTeamTotalMatches[team1.Player2ID]++
		playerTeamTotalMatches[team2.Player1ID]++
		playerTeamTotalMatches[team2.Player2ID]++

		// Update wins/losses
		if isTeam1Winner {
			playerTeamWins[team1.Player1ID]++
			playerTeamWins[team1.Player2ID]++
			playerTeamLosses[team2.Player1ID]++
			playerTeamLosses[team2.Player2ID]++
		} else {
			playerTeamWins[team2.Player1ID]++
			playerTeamWins[team2.Player2ID]++
			playerTeamLosses[team1.Player1ID]++
			playerTeamLosses[team1.Player2ID]++
		}
	}

	// Update team statistics as well
	teamStats := make(map[uint]map[string]interface{})
	for _, teamMatch := range sortedTeamMatches {
		if teamMatch.Status != "confirmed" {
			continue
		}

		// Initialize team stats if not exists
		if _, exists := teamStats[teamMatch.Team1ID]; !exists {
			teamStats[teamMatch.Team1ID] = map[string]interface{}{
				"total_matches": 0,
				"wins":          0,
				"losses":        0,
				"elo_rating":    1200.0,
			}
		}
		if _, exists := teamStats[teamMatch.Team2ID]; !exists {
			teamStats[teamMatch.Team2ID] = map[string]interface{}{
				"total_matches": 0,
				"wins":          0,
				"losses":        0,
				"elo_rating":    1200.0,
			}
		}

		// Update match counts
		teamStats[teamMatch.Team1ID]["total_matches"] = teamStats[teamMatch.Team1ID]["total_matches"].(int) + 1
		teamStats[teamMatch.Team2ID]["total_matches"] = teamStats[teamMatch.Team2ID]["total_matches"].(int) + 1

		// Update wins/losses
		if teamMatch.WinnerTeamID == teamMatch.Team1ID {
			teamStats[teamMatch.Team1ID]["wins"] = teamStats[teamMatch.Team1ID]["wins"].(int) + 1
			teamStats[teamMatch.Team2ID]["losses"] = teamStats[teamMatch.Team2ID]["losses"].(int) + 1
		} else {
			teamStats[teamMatch.Team2ID]["wins"] = teamStats[teamMatch.Team2ID]["wins"].(int) + 1
			teamStats[teamMatch.Team1ID]["losses"] = teamStats[teamMatch.Team1ID]["losses"].(int) + 1
		}

		// Calculate team ELO change (average of players' changes)
		var team1, team2 models.Team
		f.db.First(&team1, teamMatch.Team1ID)
		f.db.First(&team2, teamMatch.Team2ID)

		team1AvgElo := coreUtils.CalculateTeamAverageElo(
			playerTeamElos[team1.Player1ID],
			playerTeamElos[team1.Player2ID],
		)
		team2AvgElo := coreUtils.CalculateTeamAverageElo(
			playerTeamElos[team2.Player1ID],
			playerTeamElos[team2.Player2ID],
		)

		isTeam1Winner := teamMatch.WinnerTeamID == teamMatch.Team1ID

		team1Player1Change := coreUtils.CalculateTeamEloChange(playerTeamElos[team1.Player1ID], team2AvgElo, isTeam1Winner)
		team1Player2Change := coreUtils.CalculateTeamEloChange(playerTeamElos[team1.Player2ID], team2AvgElo, isTeam1Winner)
		team2Player1Change := coreUtils.CalculateTeamEloChange(playerTeamElos[team2.Player1ID], team1AvgElo, !isTeam1Winner)
		team2Player2Change := coreUtils.CalculateTeamEloChange(playerTeamElos[team2.Player2ID], team1AvgElo, !isTeam1Winner)

		team1EloChange := (team1Player1Change + team1Player2Change) / 2.0
		team2EloChange := (team2Player1Change + team2Player2Change) / 2.0

		teamStats[teamMatch.Team1ID]["elo_rating"] = teamStats[teamMatch.Team1ID]["elo_rating"].(float64) + team1EloChange
		teamStats[teamMatch.Team2ID]["elo_rating"] = teamStats[teamMatch.Team2ID]["elo_rating"].(float64) + team2EloChange
	}

	// Update final team stats in players table
	for _, player := range players {
		playerID := player.ID
		f.db.Model(&models.Player{}).Where("id = ?", playerID).Updates(map[string]interface{}{
			"team_elo_rating":    playerTeamElos[playerID],
			"team_total_matches": playerTeamTotalMatches[playerID],
			"team_wins":          playerTeamWins[playerID],
			"team_losses":        playerTeamLosses[playerID],
		})
	}

	// Update final team statistics
	for teamID, stats := range teamStats {
		f.db.Model(&models.Team{}).Where("id = ?", teamID).Updates(stats)
	}

	log.Println("Generated team ELO history and stats for all team matches")
	return nil
}

// recalculateAllRanks calculates both solo and team ranks for all players
func (f *Fixtures) recalculateAllRanks() error {
	log.Println("Recalculating all player ranks...")

	// Recalculate solo ranks (based on elo_rating)
	var playersForSoloRank []models.Player
	if err := f.db.Order("elo_rating DESC, id ASC").Find(&playersForSoloRank).Error; err != nil {
		return err
	}

	currentRank := 1
	var previousElo float64

	for i, player := range playersForSoloRank {
		// Si ce n'est pas le premier joueur et que l'ELO est différent du précédent
		if i > 0 && player.EloRating != previousElo {
			currentRank = i + 1
		}

		// Mettre à jour le rang solo du joueur
		if err := f.db.Model(&player).Update("rank", currentRank).Error; err != nil {
			return err
		}

		previousElo = player.EloRating
	}

	// Recalculate team ranks (based on team_elo_rating)
	var playersForTeamRank []models.Player
	if err := f.db.Order("team_elo_rating DESC, id ASC").Find(&playersForTeamRank).Error; err != nil {
		return err
	}

	currentTeamRank := 1
	var previousTeamElo float64

	for i, player := range playersForTeamRank {
		// Si ce n'est pas le premier joueur et que l'ELO équipe est différent du précédent
		if i > 0 && player.TeamEloRating != previousTeamElo {
			currentTeamRank = i + 1
		}

		// Mettre à jour le rang équipe du joueur
		if err := f.db.Model(&player).Update("team_rank", currentTeamRank).Error; err != nil {
			return err
		}

		previousTeamElo = player.TeamEloRating
	}

	log.Println("Successfully recalculated solo and team ranks for all players")
	return nil
}

// generateTournaments creates 3 tournaments (2 finished, 1 ongoing) with registered teams and matches
func (f *Fixtures) generateTournaments(teams []models.Team) error {
	now := time.Now()

	tournamentDefs := []struct {
		name        string
		slug        string
		typ         string
		status      string
		description string
		createdAt   time.Time
	}{
		{
			name:        "Tournoi de Noël",
			slug:        "tournoi-de-noel",
			typ:         "team",
			status:      "finished",
			description: "Le grand tournoi de Noël de l'association BAB-INSA",
			createdAt:   now.AddDate(0, -2, 0),
		},
		{
			name:        "Coupe de Printemps",
			slug:        "coupe-de-printemps",
			typ:         "team",
			status:      "finished",
			description: "La coupe de printemps, édition 2025",
			createdAt:   now.AddDate(0, -1, 0),
		},
		{
			name:        "Tournoi d'Hiver",
			slug:        "tournoi-d-hiver",
			typ:         "team",
			status:      "opened",
			description: "Le tournoi d'hiver ouvert aux inscriptions",
			createdAt:   now.AddDate(0, 0, -7),
		},
	}

	// Teams to register per tournament (indices into teams slice)
	// Tournament 1 (finished): 4 teams, Tournament 2 (finished): 3 teams, Tournament 3 (opened): 4 teams
	teamRegistrations := [][]int{
		{0, 1, 2, 3},
		{0, 4, 5},
		{1, 2, 4, 6},
	}

	for i, def := range tournamentDefs {
		tournament := models.Tournament{
			Name:        def.name,
			Slug:        def.slug,
			Type:        def.typ,
			Status:      def.status,
			Description: def.description,
			CreatedAt:   def.createdAt,
		}

		if err := f.db.Create(&tournament).Error; err != nil {
			return err
		}

		// Register teams
		registeredTeams := []models.Team{}
		for _, teamIdx := range teamRegistrations[i] {
			if teamIdx >= len(teams) {
				continue
			}

			tt := models.TournamentTeam{
				TournamentID: tournament.ID,
				TeamID:       teams[teamIdx].ID,
			}

			if err := f.db.Create(&tt).Error; err != nil {
				return err
			}
			registeredTeams = append(registeredTeams, teams[teamIdx])
		}

		// Update nb_participants
		f.db.Model(&tournament).Update("nb_participants", len(registeredTeams))

		// Generate matches for finished tournaments
		matchCount := 0
		if def.status == "finished" && len(registeredTeams) >= 2 {
			matchCount = f.generateTournamentMatches(tournament, registeredTeams)
		}

		// Update nb_matches
		if matchCount > 0 {
			f.db.Model(&tournament).Update("nb_matches", matchCount)
		}

		log.Printf("Created tournament: %s (ID: %d, status: %s, %d teams, %d matches)", def.name, tournament.ID, def.status, len(registeredTeams), matchCount)
	}

	log.Println("Generated 3 tournaments (2 finished, 1 opened)")
	return nil
}

// generateTournamentMatches creates round-robin matches for a finished tournament
func (f *Fixtures) generateTournamentMatches(tournament models.Tournament, registeredTeams []models.Team) int {
	matchCount := 0
	tournamentID := tournament.ID

	// Track wins/losses per team for TournamentTeam stats
	teamWins := make(map[uint]int)
	teamLosses := make(map[uint]int)

	// Round-robin: each team plays every other team
	for a := 0; a < len(registeredTeams); a++ {
		for b := a + 1; b < len(registeredTeams); b++ {
			team1 := registeredTeams[a]
			team2 := registeredTeams[b]

			// Skip if teams share a player
			if team1.Player1ID == team2.Player1ID || team1.Player1ID == team2.Player2ID ||
				team1.Player2ID == team2.Player1ID || team1.Player2ID == team2.Player2ID {
				continue
			}

			// Match date: spread across the tournament period (starting a few days after creation)
			daysOffset := 3 + matchCount*2
			matchDate := tournament.CreatedAt.AddDate(0, 0, daysOffset).Add(time.Duration(10+rand.Intn(10)) * time.Hour) // #nosec G404

			// Random winner
			var winnerTeamID uint
			winRoll := rand.Float32() // #nosec G404
			if winRoll < 0.5 {
				winnerTeamID = team1.ID
			} else {
				winnerTeamID = team2.ID
			}

			confirmedAt := matchDate.Add(time.Duration(rand.Intn(60)) * time.Minute) // #nosec G404

			teamMatch := models.TeamMatch{
				Team1ID:      team1.ID,
				Team2ID:      team2.ID,
				WinnerTeamID: winnerTeamID,
				Status:       "confirmed",
				TournamentID: &tournamentID,
				CreatedAt:    matchDate,
				ConfirmedAt:  &confirmedAt,
			}

			if err := f.db.Create(&teamMatch).Error; err != nil {
				log.Printf("Error creating tournament match: %v", err)
				continue
			}

			// Track wins/losses
			if winnerTeamID == team1.ID {
				teamWins[team1.ID]++
				teamLosses[team2.ID]++
			} else {
				teamWins[team2.ID]++
				teamLosses[team1.ID]++
			}

			matchCount++
		}
	}

	// Update TournamentTeam wins/losses
	for _, team := range registeredTeams {
		f.db.Model(&models.TournamentTeam{}).
			Where("tournament_id = ? AND team_id = ?", tournamentID, team.ID).
			Updates(map[string]interface{}{
				"wins":   teamWins[team.ID],
				"losses": teamLosses[team.ID],
			})
	}

	return matchCount
}
