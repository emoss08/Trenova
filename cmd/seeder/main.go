//go:build ignore
// +build ignore

package main

import (
	"context"
	"log"
	"os"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent/organization"
	"github.com/joho/godotenv"
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

	ctx := context.Background()

	// Create a business unit
	businessUnit := client.BusinessUnit.
		Create().
		SetName("Trenova Transportation").
		SetEntityKey("TREN").
		SetPhoneNumber("123-456-7890").
		SetCity("San Francisco").
		SetState("CA").
		SaveX(ctx)

	// Create an organization
	organization := client.Organization.Create().
		SetBusinessUnit(businessUnit).
		SetName("Trenova Transporation").
		SetScacCode("TREN").
		SetDotNumber("123456").
		SetOrgType(organization.OrgTypeA).
		SaveX(ctx)

	// Create an accounting control
	_, err = client.AccountingControl.Create().
		SetOrganization(organization).
		SetBusinessUnit(businessUnit).
		Save(ctx)
}
