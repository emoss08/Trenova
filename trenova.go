package main

import (
	"context"
	"log"
	"os"

	"github.com/emoss08/trenova/database"
	_ "github.com/emoss08/trenova/ent/runtime"
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

	ctx := context.Background()

	// // Create a business unit
	// businessUnit := client.BusinessUnit.
	// 	Create().
	// 	SetName("Trenova Transportation").
	// 	SetEntityKey("TREN").
	// 	SetPhoneNumber("123-456-7890").
	// 	SetCity("San Francisco").
	// 	SetState("CA").
	// 	SaveX(ctx)

	// // Create an organization
	// client.Organization.Create().
	// 	SetBusinessUnitID(uuid.MustParse("c4d959bc-0f75-4069-a130-ddd73e51c643")).
	// 	SetName("Trenova Transporation").
	// 	SetScacCode("TREX").
	// 	SetDotNumber("123456").
	// 	SetOrgType(organization.OrgTypeA).
	// 	SaveX(ctx)

	// Create an accounting control
	// client.BillingControl.Create().
	// 	SetOrganizationID(uuid.MustParse("c2ae34ad-cd7f-4dd9-be7e-da27ff0c0308")).
	// 	SetBusinessUnitID(uuid.MustParse("c4d959bc-0f75-4069-a130-ddd73e51c643")).
	// 	SaveX(ctx)

	defer client.Close()

	// Set the client to variabled defined in the database package
	// This will enable the client instance to be accessed anywhere through the accessor
	// named GetClient
	database.SetClient(client)

	// Setup server
	// server.SetupAndRun()
}
