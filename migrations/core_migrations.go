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
		{
			Name: "2024_11_06_000000_add_rank_to_players",
			Up: func(db *gorm.DB) error {
				// Add rank column to players table
				if err := db.Exec(`
					ALTER TABLE players ADD COLUMN IF NOT EXISTS rank INT DEFAULT 1;
					CREATE INDEX IF NOT EXISTS idx_players_rank ON players(rank);
				`).Error; err != nil {
					return err
				}
				return nil
			},
			Down: func(db *gorm.DB) error {
				// Remove rank column from players table
				if err := db.Exec(`
					DROP INDEX IF EXISTS idx_players_rank;
					ALTER TABLE players DROP COLUMN IF EXISTS rank;
				`).Error; err != nil {
					return err
				}
				return nil
			},
		},
		{
			Name: "2024_11_14_000000_add_team_support",
			Up: func(db *gorm.DB) error {
				// Add team ELO fields to players table
				if err := db.Exec(`
					ALTER TABLE players 
					ADD COLUMN IF NOT EXISTS team_elo_rating FLOAT DEFAULT 1200,
					ADD COLUMN IF NOT EXISTS team_rank INT DEFAULT 1,
					ADD COLUMN IF NOT EXISTS team_total_matches INT DEFAULT 0,
					ADD COLUMN IF NOT EXISTS team_wins INT DEFAULT 0,
					ADD COLUMN IF NOT EXISTS team_losses INT DEFAULT 0;
					
					CREATE INDEX IF NOT EXISTS idx_players_team_elo_rating ON players(team_elo_rating);
					CREATE INDEX IF NOT EXISTS idx_players_team_rank ON players(team_rank);
				`).Error; err != nil {
					return err
				}

				// Create teams table
				if err := db.Exec(`
					CREATE TABLE IF NOT EXISTS teams (
						id BIGSERIAL PRIMARY KEY,
						player1_id BIGINT NOT NULL,
						player2_id BIGINT NOT NULL,
						name VARCHAR(255),
						slug VARCHAR(255) UNIQUE NOT NULL,
						elo_rating FLOAT DEFAULT 1200,
						total_matches INT DEFAULT 0,
						wins INT DEFAULT 0,
						losses INT DEFAULT 0,
						created_at TIMESTAMP DEFAULT NOW(),
						updated_at TIMESTAMP DEFAULT NOW(),
						deleted_at TIMESTAMP NULL,
						FOREIGN KEY (player1_id) REFERENCES players(id) ON DELETE CASCADE,
						FOREIGN KEY (player2_id) REFERENCES players(id) ON DELETE CASCADE
					);
					CREATE INDEX IF NOT EXISTS idx_teams_deleted_at ON teams(deleted_at);
					CREATE INDEX IF NOT EXISTS idx_teams_player1_id ON teams(player1_id);
					CREATE INDEX IF NOT EXISTS idx_teams_player2_id ON teams(player2_id);
					CREATE INDEX IF NOT EXISTS idx_teams_elo_rating ON teams(elo_rating);
					CREATE INDEX IF NOT EXISTS idx_teams_slug ON teams(slug);
				`).Error; err != nil {
					return err
				}

				// Create team_matches table
				if err := db.Exec(`
					CREATE TABLE IF NOT EXISTS team_matches (
						id BIGSERIAL PRIMARY KEY,
						team1_id BIGINT NOT NULL,
						team2_id BIGINT NOT NULL,
						winner_team_id BIGINT NOT NULL,
						status VARCHAR(20) DEFAULT 'pending',
						created_at TIMESTAMP DEFAULT NOW(),
						confirmed_at TIMESTAMP NULL,
						updated_at TIMESTAMP DEFAULT NOW(),
						deleted_at TIMESTAMP NULL,
						FOREIGN KEY (team1_id) REFERENCES teams(id) ON DELETE CASCADE,
						FOREIGN KEY (team2_id) REFERENCES teams(id) ON DELETE CASCADE,
						FOREIGN KEY (winner_team_id) REFERENCES teams(id) ON DELETE CASCADE
					);
					CREATE INDEX IF NOT EXISTS idx_team_matches_deleted_at ON team_matches(deleted_at);
					CREATE INDEX IF NOT EXISTS idx_team_matches_status ON team_matches(status);
					CREATE INDEX IF NOT EXISTS idx_team_matches_team1_id ON team_matches(team1_id);
					CREATE INDEX IF NOT EXISTS idx_team_matches_team2_id ON team_matches(team2_id);
					CREATE INDEX IF NOT EXISTS idx_team_matches_winner_team_id ON team_matches(winner_team_id);
				`).Error; err != nil {
					return err
				}

				// Add team support to elo_history table
				if err := db.Exec(`
					ALTER TABLE elo_history 
					ADD COLUMN IF NOT EXISTS opponent_team_id BIGINT,
					ADD COLUMN IF NOT EXISTS match_type VARCHAR(20) DEFAULT 'solo';
					
					ALTER TABLE elo_history 
					ADD CONSTRAINT fk_elo_history_opponent_team 
					FOREIGN KEY (opponent_team_id) REFERENCES teams(id);
					
					CREATE INDEX IF NOT EXISTS idx_elo_history_opponent_team_id ON elo_history(opponent_team_id);
					CREATE INDEX IF NOT EXISTS idx_elo_history_match_type ON elo_history(match_type);
				`).Error; err != nil {
					return err
				}

				return nil
			},
			Down: func(db *gorm.DB) error {
				// Remove team support from elo_history
				if err := db.Exec(`
					DROP INDEX IF EXISTS idx_elo_history_match_type;
					DROP INDEX IF EXISTS idx_elo_history_opponent_team_id;
					ALTER TABLE elo_history DROP CONSTRAINT IF EXISTS fk_elo_history_opponent_team;
					ALTER TABLE elo_history 
					DROP COLUMN IF EXISTS match_type,
					DROP COLUMN IF EXISTS opponent_team_id;
				`).Error; err != nil {
					return err
				}

				// Drop team_matches table
				if err := db.Exec("DROP TABLE IF EXISTS team_matches CASCADE").Error; err != nil {
					return err
				}

				// Drop teams table
				if err := db.Exec("DROP TABLE IF EXISTS teams CASCADE").Error; err != nil {
					return err
				}

				// Remove team fields from players table
				if err := db.Exec(`
					DROP INDEX IF EXISTS idx_players_team_rank;
					DROP INDEX IF EXISTS idx_players_team_elo_rating;
					ALTER TABLE players 
					DROP COLUMN IF EXISTS team_losses,
					DROP COLUMN IF EXISTS team_wins,
					DROP COLUMN IF EXISTS team_total_matches,
					DROP COLUMN IF EXISTS team_rank,
					DROP COLUMN IF EXISTS team_elo_rating;
				`).Error; err != nil {
					return err
				}

				return nil
			},
		},
	}
}
