package migrations

import "gorm.io/gorm"

func GetCoreMigrations() []MigrationDefinition {
	return []MigrationDefinition{
		{
			Name: "2024_01_04_000000_create_core_tables",
			Up: func(db *gorm.DB) error {
				// Create players table
				if err := db.Exec(`
					CREATE TABLE IF NOT EXISTS players (
						id BIGINT PRIMARY KEY,
						username VARCHAR(255) NOT NULL,
						elo_rating FLOAT DEFAULT 1200,
						total_matches INT DEFAULT 0,
						wins INT DEFAULT 0,
						losses INT DEFAULT 0,
						current_win_streak INT DEFAULT 0,
						best_win_streak INT DEFAULT 0,
						created_at TIMESTAMP DEFAULT NOW(),
						updated_at TIMESTAMP DEFAULT NOW(),
						deleted_at TIMESTAMP NULL,
						FOREIGN KEY (id) REFERENCES users(id) ON DELETE CASCADE
					);
					CREATE INDEX IF NOT EXISTS idx_players_deleted_at ON players(deleted_at);
					CREATE INDEX IF NOT EXISTS idx_players_elo_rating ON players(elo_rating);
					CREATE INDEX IF NOT EXISTS idx_players_current_win_streak ON players(current_win_streak);
					CREATE INDEX IF NOT EXISTS idx_players_best_win_streak ON players(best_win_streak);
				`).Error; err != nil {
					return err
				}

				// Create matches table
				if err := db.Exec(`
					CREATE TABLE IF NOT EXISTS matches (
						id BIGSERIAL PRIMARY KEY,
						player1_id BIGINT NOT NULL,
						player2_id BIGINT NOT NULL,
						winner_id BIGINT NOT NULL,
						status VARCHAR(20) DEFAULT 'pending',
						created_at TIMESTAMP DEFAULT NOW(),
						confirmed_at TIMESTAMP NULL,
						updated_at TIMESTAMP DEFAULT NOW(),
						deleted_at TIMESTAMP NULL,
						FOREIGN KEY (player1_id) REFERENCES players(id) ON DELETE CASCADE,
						FOREIGN KEY (player2_id) REFERENCES players(id) ON DELETE CASCADE,
						FOREIGN KEY (winner_id) REFERENCES players(id) ON DELETE CASCADE
					);
					CREATE INDEX IF NOT EXISTS idx_matches_deleted_at ON matches(deleted_at);
					CREATE INDEX IF NOT EXISTS idx_matches_status ON matches(status);
					CREATE INDEX IF NOT EXISTS idx_matches_player1_id ON matches(player1_id);
					CREATE INDEX IF NOT EXISTS idx_matches_player2_id ON matches(player2_id);
					CREATE INDEX IF NOT EXISTS idx_matches_winner_id ON matches(winner_id);
				`).Error; err != nil {
					return err
				}

				// Create elo_history table
				if err := db.Exec(`
					CREATE TABLE IF NOT EXISTS elo_history (
						id BIGSERIAL PRIMARY KEY,
						player_id BIGINT NOT NULL,
						match_id BIGINT NOT NULL,
						elo_before FLOAT NOT NULL,
						elo_after FLOAT NOT NULL,
						elo_change FLOAT NOT NULL,
						opponent_id BIGINT,
						created_at TIMESTAMP DEFAULT NOW(),
						updated_at TIMESTAMP DEFAULT NOW(),
						deleted_at TIMESTAMP NULL,
						FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE CASCADE,
						FOREIGN KEY (match_id) REFERENCES matches(id) ON DELETE CASCADE,
						FOREIGN KEY (opponent_id) REFERENCES players(id)
					);
					CREATE INDEX IF NOT EXISTS idx_elo_history_deleted_at ON elo_history(deleted_at);
					CREATE INDEX IF NOT EXISTS idx_elo_history_player_id ON elo_history(player_id);
					CREATE INDEX IF NOT EXISTS idx_elo_history_match_id ON elo_history(match_id);
					CREATE INDEX IF NOT EXISTS idx_elo_history_opponent_id ON elo_history(opponent_id);
				`).Error; err != nil {
					return err
				}

				return nil
			},
			Down: func(db *gorm.DB) error {
				// Drop tables in reverse order (because of foreign keys)
				if err := db.Exec("DROP TABLE IF EXISTS elo_history CASCADE").Error; err != nil {
					return err
				}
				if err := db.Exec("DROP TABLE IF EXISTS matches CASCADE").Error; err != nil {
					return err
				}
				if err := db.Exec("DROP TABLE IF EXISTS players CASCADE").Error; err != nil {
					return err
				}
				return nil
			},
		},
	}
}
