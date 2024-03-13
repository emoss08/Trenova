package migratedata

import (
	"context"

	"ariga.io/atlas/sql/migrate"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql/schema"
	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/organization"
)

func SeedBusinessUnit(dir *migrate.LocalDir) error {
	w := &schema.DirWriter{Dir: dir}
	client := ent.NewClient(ent.Driver(schema.NewWriteDriver(dialect.Postgres, w)))

	// The statement that generates the INSERT statement.
	err := client.BusinessUnit.Create().
		SetName("Trenova Transportation").
		SetEntityKey("TREN").
		SetPhoneNumber("123-456-7890").
		SetCity("San Francisco").
		SetState("CA").
		Exec(context.Background())
	if err != nil {
		return nil
	}

	return w.FlushChange(
		"seed_business_unit",
		"Add the initial business unit to the database",
	)
}

func SeedOrganization(dir *migrate.LocalDir) error {
	w := &schema.DirWriter{Dir: dir}
	client := ent.NewClient(ent.Driver(schema.NewWriteDriver(dialect.Postgres, w)))

	// Get the first business unit from the client
	businessunit, buErr := client.BusinessUnit.Query().First(context.Background())

	if buErr != nil {
		return buErr
	}

	// The Statement that generates the INSERT statement.
	err := client.Organization.Create().
		SetBusinessUnitID(businessunit.ID).
		SetName("Trenova Transportation").
		SetScacCode("TREX").
		SetDotNumber("123456").
		SetOrgType(organization.OrgTypeA).
		Exec(context.Background())
	if err != nil {
		return nil
	}

	return w.FlushChange(
		"seed_organization",
		"Add the initial organization to the database",
	)
}
