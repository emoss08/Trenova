package cmd

import (
	"log"
	"os"
	"strconv"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/server"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		panic("Error loading .env file")
	}

	maxIdleConns, _ := strconv.Atoi(os.Getenv("DB_MAX_IDLE_CONNS"))
	maxOpenConns, _ := strconv.Atoi(os.Getenv("DB_MAX_OPEN_CONNS"))

	dbConfig := database.DBConfig{
		DSN:             os.Getenv("DB_DSN"),
		MaxIdleConns:    maxIdleConns,
		MaxOpenConns:    maxOpenConns,
		ConnMaxLifetime: 0,
		ConnMaxIdleTime: 0,
	}

	// Connect to the database.
	db, cancel, err := database.ConnectDB(dbConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	migrationsPath := "config/database/migrations"

	log.Println("Running types migration...")

	if typeErr := database.MigrateTypes(db, migrationsPath); typeErr != nil {
		log.Fatal("Failed to run types migration. \n", typeErr)
	}

	defer cancel()

	// Setup server
	server.SetupAndRun(db)
}
