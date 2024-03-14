package main

import (
	"log"
	"os"

	"github.com/emoss08/trenova/database"
	_ "github.com/emoss08/trenova/ent/runtime"
	"github.com/emoss08/trenova/server"
	"github.com/emoss08/trenova/tools/redis"
	"github.com/emoss08/trenova/tools/session"
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

	defer client.Close()

	// Set the client to variable defined in the database package
	// This will enable the client instance to be accessed anywhere through the accessor
	// named GetClient
	database.SetClient(client)

	// Setup Session Store
	sessionStore := session.New(client, []byte(os.Getenv("SESSION_KEY")))

	// If the session store is not set, panic
	if sessionStore == nil {
		panic("Failed to create session store. Exiting...")
	}

	session.SetStore(sessionStore)

	// Initialize the redis client
	redisClient := redis.NewRedisClient(os.Getenv("REDIS_ADDR"))

	// Set the redis client to variable defined in the redis package
	// This will enable the client instance to be accessed anywhere through the accessor
	// named GetClient
	redis.SetClient(redisClient)

	// Setup server
	server.SetupAndRun()
}
