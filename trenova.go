package main

import (
	"context"
	"log"

	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/migrate"
	_ "github.com/emoss08/trenova/ent/runtime"
	_ "github.com/lib/pq"
)

func main() {
	client, err := ent.Open("postgres", "host=localhost port=5432 user=postgres dbname=trenova_go_db password=postgres sslmode=disable")
	if err != nil {
		log.Fatalf("failed opening connection to postgres: %v", err)
	}
	defer client.Close()
	ctx := context.Background()
	// Run migration.
	err = client.Debug().Schema.Create(
		ctx,
		migrate.WithDropIndex(true),
		migrate.WithDropColumn(true),
	)
	if err != nil {
		log.Fatalf("failed creating schema resources: %v", err)
	}

	//err := godotenv.Load()
	//if err != nil {
	//	panic("Error loading .env file")
	//}
	//
	//maxIdleConns, _ := strconv.Atoi(os.Getenv("DB_MAX_IDLE_CONNS"))
	//maxOpenConns, _ := strconv.Atoi(os.Getenv("DB_MAX_OPEN_CONNS"))
	//
	//dbConfig := database.DBConfig{
	//	DSN:             os.Getenv("DB_DSN"),
	//	MaxIdleConns:    maxIdleConns,
	//	MaxOpenConns:    maxOpenConns,
	//	ConnMaxLifetime: 0,
	//	ConnMaxIdleTime: 0,
	//}
	//
	//// Connect to the database.
	//db, cancel, err := database.ConnectDB(dbConfig)
	//if err != nil {
	//	log.Fatalf("Failed to connect to database: %v", err)
	//}
	//
	//migrationsPath := "config/database/migrations"
	//
	//log.Println("Running types migration...")
	//
	//if typeErr := database.MigrateTypes(db, migrationsPath); typeErr != nil {
	//	log.Fatal("Failed to run types migration. \n", typeErr)
	//}
	//
	//defer cancel()
	//
	//// Setup server
	//server.SetupAndRun(db)
}
