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

	log.Println("Fixtures generated successfully!")
	log.Printf("Created %d users, %d matches, and ELO history", len(users), len(matches))
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
		userID := uint(i + 1) // Force IDs 1, 2, 3, ...

		user := authModels.User{
			ID:          userID,
			Email:       email,
			Username:    username,
			Slug:        slug,
			Password:    hashedPassword,
			Enabled:     true,
			NbConnexion: rand.Intn(50) + 1,
			Roles:       authModels.GetDefaultRoles(),
		}

		if err := f.db.Create(&user).Error; err != nil {
			return nil, err
		}

		// Create corresponding player with fixed ELO based on index
		baseElos := []float64{1200, 1250, 1180, 1320, 1150, 1280, 1220, 1350, 1100, 1300}
		player := models.Player{
			ID:               userID, // Same ID as user
			Username:         user.Username,
			EloRating:        baseElos[i],
			TotalMatches:     0, // Will be updated when matches are created
			Wins:             0,
			Losses:           0,
			CurrentWinStreak: 0,
			BestWinStreak:    0,
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
		daysAgo := rand.Intn(30)
		matchDate := now.AddDate(0, 0, -daysAgo).Add(time.Duration(rand.Intn(24)) * time.Hour)

		// Pick two random different players
		player1 := users[rand.Intn(len(users))]
		var player2 authModels.User
		for {
			player2 = users[rand.Intn(len(users))]
			if player2.ID != player1.ID {
				break
			}
		}

		// Random winner
		var winner uint
		if rand.Float32() < 0.5 {
			winner = player1.ID
		} else {
			winner = player2.ID
		}

		// Random status (most confirmed)
		status := "confirmed"
		confirmedAt := &matchDate
		if rand.Float32() < 0.1 { // 10% pending
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

func (f *Fixtures) updatePlayerStats(player1ID, player2ID, winnerID uint) {
	// Update total matches
	f.db.Model(&models.Player{}).Where("id IN ?", []uint{player1ID, player2ID}).
		Update("total_matches", gorm.Expr("total_matches + 1"))

	// Update wins/losses
	f.db.Model(&models.Player{}).Where("id = ?", winnerID).
		Update("wins", gorm.Expr("wins + 1"))

	loserID := player1ID
	if winnerID == player1ID {
		loserID = player2ID
	}
	f.db.Model(&models.Player{}).Where("id = ?", loserID).
		Update("losses", gorm.Expr("losses + 1"))
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
			"elo_rating":         playerElos[playerID],
			"total_matches":      playerTotalMatches[playerID],
			"wins":               playerWins[playerID],
			"losses":             playerLosses[playerID],
			"current_win_streak": playerCurrentStreaks[playerID],
			"best_win_streak":    playerBestStreaks[playerID],
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
		&models.EloHistory{},
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
		"ALTER SEQUENCE elo_history_id_seq RESTART WITH 1",
		"ALTER SEQUENCE refresh_tokens_id_seq RESTART WITH 1",
	}

	for _, seq := range sequences {
		f.db.Exec(seq)
	}

	log.Println("All fixture data cleared!")
	return nil
}
