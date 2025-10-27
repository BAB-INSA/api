package migrations

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type Migration struct {
	ID        uint      `gorm:"primaryKey"`
	Name      string    `gorm:"unique;not null"`
	Batch     int       `gorm:"not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

type MigrationFunc func(*gorm.DB) error

type MigrationDefinition struct {
	Name string
	Up   MigrationFunc
	Down MigrationFunc
}

type Migrator struct {
	db         *gorm.DB
	migrations []MigrationDefinition
}

func NewMigrator(db *gorm.DB) *Migrator {
	db.AutoMigrate(&Migration{})
	return &Migrator{
		db:         db,
		migrations: []MigrationDefinition{},
	}
}

func (m *Migrator) AddMigration(migration MigrationDefinition) {
	m.migrations = append(m.migrations, migration)
}

func (m *Migrator) Migrate() error {
	fmt.Println("Running database migrations...")

	batch := m.getNextBatch()

	for _, migration := range m.migrations {
		if m.hasRun(migration.Name) {
			continue
		}

		fmt.Printf("Migrating: %s\n", migration.Name)

		tx := m.db.Begin()

		if err := migration.Up(tx); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %s failed: %w", migration.Name, err)
		}

		migrationRecord := Migration{
			Name:  migration.Name,
			Batch: batch,
		}

		if err := tx.Create(&migrationRecord).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration %s: %w", migration.Name, err)
		}

		tx.Commit()
		fmt.Printf("Migrated: %s\n", migration.Name)
	}

	fmt.Println("Migration completed successfully")
	return nil
}

func (m *Migrator) Rollback(steps int) error {
	if steps <= 0 {
		steps = 1
	}

	fmt.Printf("Rolling back %d migration(s)...\n", steps)

	batch := m.getLatestBatch()

	for i := 0; i < steps && batch > 0; i++ {
		var migrationsToRollback []Migration
		m.db.Where("batch = ?", batch).Order("id DESC").Find(&migrationsToRollback)

		for _, migrationRecord := range migrationsToRollback {
			migration := m.findMigration(migrationRecord.Name)
			if migration == nil {
				return fmt.Errorf("migration definition not found: %s", migrationRecord.Name)
			}

			if migration.Down == nil {
				return fmt.Errorf("rollback not defined for migration: %s", migrationRecord.Name)
			}

			fmt.Printf("Rolling back: %s\n", migrationRecord.Name)

			tx := m.db.Begin()

			if err := migration.Down(tx); err != nil {
				tx.Rollback()
				return fmt.Errorf("rollback failed for %s: %w", migrationRecord.Name, err)
			}

			if err := tx.Delete(&migrationRecord).Error; err != nil {
				tx.Rollback()
				return fmt.Errorf("failed to remove migration record %s: %w", migrationRecord.Name, err)
			}

			tx.Commit()
			fmt.Printf("Rolled back: %s\n", migrationRecord.Name)
		}

		batch--
	}

	fmt.Println("Rollback completed successfully")
	return nil
}

func (m *Migrator) hasRun(name string) bool {
	var count int64
	m.db.Model(&Migration{}).Where("name = ?", name).Count(&count)
	return count > 0
}

func (m *Migrator) getNextBatch() int {
	var migration Migration
	m.db.Order("batch DESC").First(&migration)
	return migration.Batch + 1
}

func (m *Migrator) getLatestBatch() int {
	var migration Migration
	m.db.Order("batch DESC").First(&migration)
	return migration.Batch
}

func (m *Migrator) findMigration(name string) *MigrationDefinition {
	for _, migration := range m.migrations {
		if migration.Name == name {
			return &migration
		}
	}
	return nil
}
