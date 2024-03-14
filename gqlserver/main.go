package main

import (
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/graph"
	"github.com/joho/godotenv"
	"log"
	"net/http"
	"os"

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

	// Configure the server and must start on :8001
	srv := handler.NewDefaultServer(graph.NewSchema(client))
	http.Handle("/",
		playground.Handler("GraphQL playground", "/query"),
	)
	http.Handle("/query", srv)
	log.Println("server started at http://localhost:8001")
	if err := http.ListenAndServe(":8001", nil); err != nil {
		log.Fatal("http server terminated with error: ", err)
	}

	// Setup server
	//server.SetupAndRun()
}
