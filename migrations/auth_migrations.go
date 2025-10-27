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
						roles JSONB DEFAULT '["user"]'::jsonb,
						created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
						updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
						deleted_at TIMESTAMP NULL
					);
					CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
					CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at);
					CREATE INDEX IF NOT EXISTS idx_users_roles ON users USING GIN (roles);
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
		{
			Name: "2024_01_03_000000_add_user_extended_fields",
			Up: func(db *gorm.DB) error {
				return db.Exec(`
					ALTER TABLE users 
					ADD COLUMN IF NOT EXISTS username VARCHAR(255) UNIQUE,
					ADD COLUMN IF NOT EXISTS slug VARCHAR(255) UNIQUE,
					ADD COLUMN IF NOT EXISTS enabled BOOLEAN DEFAULT true,
					ADD COLUMN IF NOT EXISTS last_login TIMESTAMP NULL,
					ADD COLUMN IF NOT EXISTS nb_connexion INTEGER DEFAULT 0,
					ADD COLUMN IF NOT EXISTS confirmation_token VARCHAR(255) NULL,
					ADD COLUMN IF NOT EXISTS password_requested_at TIMESTAMP NULL;
					
					CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username ON users(username);
					CREATE UNIQUE INDEX IF NOT EXISTS idx_users_slug ON users(slug);
				`).Error
			},
			Down: func(db *gorm.DB) error {
				return db.Exec(`
					DROP INDEX IF EXISTS idx_users_username;
					DROP INDEX IF EXISTS idx_users_slug;
					ALTER TABLE users 
					DROP COLUMN IF EXISTS username,
					DROP COLUMN IF EXISTS slug,
					DROP COLUMN IF EXISTS enabled,
					DROP COLUMN IF EXISTS last_login,
					DROP COLUMN IF EXISTS nb_connexion,
					DROP COLUMN IF EXISTS confirmation_token,
					DROP COLUMN IF EXISTS password_requested_at;
				`).Error
			},
		},
	}
}
