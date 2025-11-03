package models

type Stats struct {
	TotalPlayers         int64 `json:"total_players"`
	TotalMatches         int64 `json:"total_matches"`
	MatchesLast7Days     int64 `json:"matches_last_7_days"`
	MatchesPrevious7Days int64 `json:"matches_previous_7_days"`
}
