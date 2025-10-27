package migrations

import "gorm.io/gorm"

func GetAuthMigrations() []MigrationDefinition {
	return []MigrationDefinition{
		{
			Name: "2024_01_01_000000_create_user_table",
			Up: func(db *gorm.DB) error {
				return db.Exec(`
					CREATE TABLE IF NOT EXISTS users (
						id SERIAL PRIMARY KEY,
						email VARCHAR(255) UNIQUE NOT NULL,
						password VARCHAR(255) NOT NULL,
						username VARCHAR(255) UNIQUE,
						slug VARCHAR(255) UNIQUE,
						enabled BOOLEAN DEFAULT true,
						last_login TIMESTAMP NULL,
						nb_connexion INTEGER DEFAULT 0,
						confirmation_token VARCHAR(255) NULL,
						password_requested_at TIMESTAMP NULL,
						roles JSONB DEFAULT '["user"]'::jsonb,
						created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
						updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
						deleted_at TIMESTAMP NULL
					);
					CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
					CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at);
					CREATE INDEX IF NOT EXISTS idx_users_roles ON users USING GIN (roles);
					CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username ON users(username);
					CREATE UNIQUE INDEX IF NOT EXISTS idx_users_slug ON users(slug);
				`).Error
			},
			Down: func(db *gorm.DB) error {
				return db.Exec("DROP TABLE IF EXISTS users CASCADE").Error
			},
		},
		{
			Name: "2024_01_02_000000_create_refresh_tokens_table",
			Up: func(db *gorm.DB) error {
				return db.Exec(`
					CREATE TABLE IF NOT EXISTS refresh_tokens (
						id SERIAL PRIMARY KEY,
						user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
						token VARCHAR(255) UNIQUE NOT NULL,
						expires_at TIMESTAMP NOT NULL,
						created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
						updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
						deleted_at TIMESTAMP NULL
					);
					CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id);
					CREATE INDEX IF NOT EXISTS idx_refresh_tokens_token ON refresh_tokens(token);
					CREATE INDEX IF NOT EXISTS idx_refresh_tokens_deleted_at ON refresh_tokens(deleted_at);
					CREATE INDEX IF NOT EXISTS idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);
				`).Error
			},
			Down: func(db *gorm.DB) error {
				return db.Exec("DROP TABLE IF EXISTS refresh_tokens CASCADE").Error
			},
		},
	}
}
