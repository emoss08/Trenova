package main

import (
	"log"
	"os"
	"trenova-go-backend/app/server"
	"trenova-go-backend/config/database"

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

	defer cancel()
	// Setup server
	server.SetupAndRun(db)

}
