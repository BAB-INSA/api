package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"bab-insa-api/config"
	"bab-insa-api/migrations"

	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	config.ConnectDatabase()
	migrator := migrations.NewMigrator(config.DB)

	// Ajouter toutes les migrations
	for _, migration := range migrations.GetAuthMigrations() {
		migrator.AddMigration(migration)
	}
	for _, migration := range migrations.GetAllMigrations() {
		migrator.AddMigration(migration)
	}

	if len(os.Args) < 2 {
		printUsage()
		return
	}

	command := os.Args[1]

	switch command {
	case "migrate":
		if err := migrator.Migrate(); err != nil {
			log.Fatal("Migration failed:", err)
		}
	case "rollback":
		steps := 1
		if len(os.Args) > 2 {
			if s, err := strconv.Atoi(os.Args[2]); err == nil {
				steps = s
			}
		}
		if err := migrator.Rollback(steps); err != nil {
			log.Fatal("Rollback failed:", err)
		}
	case "status":
		showStatus(config.DB)
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
	}
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  go run cmd/migrate.go migrate          - Run pending migrations")
	fmt.Println("  go run cmd/migrate.go rollback [steps] - Rollback migrations (default: 1)")
	fmt.Println("  go run cmd/migrate.go status           - Show migration status")
}

func showStatus(db *gorm.DB) {
	var migrations []migrations.Migration
	db.Order("batch ASC, id ASC").Find(&migrations)

	if len(migrations) == 0 {
		fmt.Println("No migrations have been run yet.")
		return
	}

	fmt.Println("Migration Status:")
	fmt.Println("Batch | Name")
	fmt.Println("------|-----")

	for _, migration := range migrations {
		fmt.Printf("%-5d | %s\n", migration.Batch, migration.Name)
	}
}
