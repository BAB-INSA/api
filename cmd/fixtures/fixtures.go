package main

import (
	"fmt"
	"log"
	"os"

	"bab-insa-api/config"
	"bab-insa-api/fixtures"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	config.ConnectDatabase()
	fixtureManager := fixtures.NewFixtures(config.DB)

	if len(os.Args) < 2 {
		printUsage()
		return
	}

	command := os.Args[1]

	switch command {
	case "generate":
		if err := fixtureManager.GenerateTestData(); err != nil {
			log.Fatal("Failed to generate fixtures:", err)
		}
		fmt.Println("✅ Fixtures generated successfully!")
	case "clear":
		if err := fixtureManager.ClearAllData(); err != nil {
			log.Fatal("Failed to clear fixtures:", err)
		}
		fmt.Println("✅ All fixture data cleared!")
	case "regenerate":
		fmt.Println("Clearing existing data...")
		if err := fixtureManager.ClearAllData(); err != nil {
			log.Fatal("Failed to clear fixtures:", err)
		}
		fmt.Println("Generating new fixtures...")
		if err := fixtureManager.GenerateTestData(); err != nil {
			log.Fatal("Failed to generate fixtures:", err)
		}
		fmt.Println("✅ Fixtures regenerated successfully!")
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
	}
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  go run cmd/fixtures.go generate    - Generate test data (10 users, 50 matches, ELO history)")
	fmt.Println("  go run cmd/fixtures.go clear       - Clear all fixture data")
	fmt.Println("  go run cmd/fixtures.go regenerate  - Clear and regenerate all data")
}
