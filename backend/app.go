package main

import (
	"github.com/google/uuid"
	"log"
	"os"
	"trenova/app/models"
	"trenova/app/server"
	"trenova/config/database"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()

	if err != nil {
		panic("Error loading .env file")
	}

	dbConfig := database.DBConfig{
		DSN:             os.Getenv("DB_DSN"),
		MaxIdleConns:    10,
		MaxOpenConns:    100,
		ConnMaxLifetime: 0,
		ConnMaxIdleTime: 0,
	}

	// Connect to the database.
	db, cancel, err := database.ConnectDb(dbConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	migrationsPath := "config/database/migrations"

	log.Println("Running types migration...")

	if err := database.MigrateTypes(db, migrationsPath); err != nil {
		log.Fatal("Failed to run types migration. \n", err)
	}

	if err := db.Create(&models.AccountingControl{
		BusinessUnitID:     uuid.MustParse("2aa25333-7032-4295-9d83-9882e6631fe7"),
		OrganizationID:     uuid.MustParse("f1d60024-7d0d-49e9-84a5-f8add9373fd7"),
		RecThreshold:       50,
		RecThresholdAction: "HALT",
	}).Error; err != nil {
		log.Fatalf("Failed to create accounting control record: %v", err)
	}

	defer cancel()
	// Setup server
	server.SetupAndRun(db)

}
