package main

import (
	"log"
	"os"

	"github.com/emoss08/trenova/database"
	_ "github.com/emoss08/trenova/ent/runtime"
	"github.com/emoss08/trenova/server"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		panic("Error loading .env file")
	}

	// Initialize the database
	client, err := database.NewEntClient(os.Getenv("DB_DSN"))
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// ctx := context.Background()

	// client.BusinessUnit.
	// 	Create().
	// 	SetName("Trenova Transportation").
	// 	SetEntityKey("TREN").
	// 	SetPhoneNumber("123-456-7890").
	// 	SetCity("San Francisco").
	// 	SetState("CA").
	// 	SaveX(ctx)

	defer client.Close()

	// Set the client to variabled defined in the database package
	// This will enable the client instance to be accessed anywhere through the accessor
	// named GetClient
	database.SetClient(client)

	// Setup server
	server.SetupAndRun()
}
