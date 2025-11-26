package utils

import "math"

// CalculateEloChange calculates ELO rating changes using the standard ELO formula
// Returns (player1Change, player2Change)
// Ensures that no player can go below 1200 ELO
func CalculateEloChange(player1Elo, player2Elo float64, winnerID, player1ID uint) (float64, float64) {
	const K = 32.0        // ELO K-factor
	const MinElo = 1200.0 // Minimum ELO rating

	// Expected scores
	expectedScore1 := 1.0 / (1.0 + math.Pow(10, (player2Elo-player1Elo)/400))
	expectedScore2 := 1.0 - expectedScore1

	// Actual scores
	var actualScore1, actualScore2 float64
	if winnerID == player1ID {
		actualScore1 = 1.0
		actualScore2 = 0.0
	} else {
		actualScore1 = 0.0
		actualScore2 = 1.0
	}

	// Calculate changes
	change1 := K * (actualScore1 - expectedScore1)
	change2 := K * (actualScore2 - expectedScore2)

	// Apply minimum ELO constraint
	if player1Elo+change1 < MinElo {
		change1 = MinElo - player1Elo
	}
	if player2Elo+change2 < MinElo {
		change2 = MinElo - player2Elo
	}

	return math.Round(change1), math.Round(change2)
}

// CalculateTeamEloChange calculates ELO rating changes for team matches
// Each player's ELO is calculated individually against the average ELO of the opposing team
// Ensures that no player can go below 1200 ELO
func CalculateTeamEloChange(playerElo, opponentTeamAvgElo float64, isWinner bool) float64 {
	const K = 32.0        // ELO K-factor
	const MinElo = 1200.0 // Minimum ELO rating

	// Expected score for this player against the opposing team's average
	expectedScore := 1.0 / (1.0 + math.Pow(10, (opponentTeamAvgElo-playerElo)/400))

	// Actual score
	var actualScore float64
	if isWinner {
		actualScore = 1.0
	} else {
		actualScore = 0.0
	}

	// Calculate change
	change := K * (actualScore - expectedScore)

	// Apply minimum ELO constraint
	if playerElo+change < MinElo {
		change = MinElo - playerElo
	}

	return math.Round(change)
}

// CalculateTeamAverageElo calculates the average ELO of a team
func CalculateTeamAverageElo(player1Elo, player2Elo float64) float64 {
	return (player1Elo + player2Elo) / 2.0
}
