package utils

import "math"

// CalculateEloChange calculates ELO rating changes using the standard ELO formula
// Returns (player1Change, player2Change)
func CalculateEloChange(player1Elo, player2Elo float64, winnerID, player1ID uint) (float64, float64) {
	const K = 32.0 // ELO K-factor

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

	return math.Round(change1), math.Round(change2)
}