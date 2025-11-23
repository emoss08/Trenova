package development

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/uptrace/bun"
)

// CustomerSeed Creates customer data
type CustomerSeed struct {
	seedhelpers.BaseSeed
}

// NewCustomerSeed creates a new customer seed
func NewCustomerSeed() *CustomerSeed {
	seed := &CustomerSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"Customer",
		"1.0.0",
		"Creates customer data",
		[]common.Environment{
			common.EnvDevelopment,
		},
	)

	// Development seeds typically depend on base seeds
	seed.SetDependencies("USStates", "AdminAccount", "Permissions", "HazmatExpiration")

	return seed
}

// Run executes the seed
func (s *CustomerSeed) Run(ctx context.Context, db *bun.DB) error {
	return seedhelpers.RunInTransaction(
		ctx,
		db,
		s.Name(),
		func(ctx context.Context, tx bun.Tx, seedCtx *seedhelpers.SeedContext) error {
			// Get default organization and business unit
			defaultOrg, err := seedCtx.GetDefaultOrganization()
			if err != nil {
				return fmt.Errorf("get default organization: %w", err)
			}

			defaultBU, err := seedCtx.GetDefaultBusinessUnit()
			if err != nil {
				return fmt.Errorf("get default business unit: %w", err)
			}

			// Get a state for reference (example: California)
			caState, err := seedCtx.GetState("CA")
			if err != nil {
				return fmt.Errorf("get California state: %w", err)
			}

			if err := s.createFacebookCustomer(ctx, tx, defaultOrg.ID, defaultBU.ID, caState.ID); err != nil {
				return fmt.Errorf("create Facebook customer: %w", err)
			}

			seedhelpers.LogSuccess("Created customer fixtures",
				"- 1 Facebook customer created",
			)

			return nil
		},
	)
}

func (s *CustomerSeed) createFacebookCustomer(
	ctx context.Context,
	tx bun.Tx,
	orgID,
	buID,
	stateID pulid.ID,
) error {
	fcbk := &customer.Customer{
		ID:             pulid.MustNew("cus"),
		BusinessUnitID: buID,
		OrganizationID: orgID,
		StateID:        stateID,
		Code:           "FCBK",
		Name:           "Facebook Inc.",
		AddressLine1:   "1 Hacker Way",
		City:           "Menlo Park",
		PostalCode:     "94025",
		Status:         domain.StatusActive,
		Latitude:       utils.Float64ToPointer(37.485023),
		Longitude:      utils.Float64ToPointer(-122.148369),
		IsGeocoded:     true,
		PlaceID:        "ChIJN1t_t3R1M4RAFUEcmHZ92EQ",
	}

	if _, err := tx.NewInsert().Model(fcbk).Exec(ctx); err != nil {
		return fmt.Errorf("create Facebook customer: %w", err)
	}

	fcbkBillingProfile := &customer.CustomerBillingProfile{
		ID:             pulid.MustNew("cbr"),
		BusinessUnitID: buID,
		OrganizationID: orgID,
		CustomerID:     fcbk.ID,
	}
	if _, err := tx.NewInsert().Model(fcbkBillingProfile).Exec(ctx); err != nil {
		return fmt.Errorf("create Facebook billing profile: %w", err)
	}

	fcbkEmailProfile := &customer.CustomerEmailProfile{
		ID:             pulid.MustNew("cem"),
		BusinessUnitID: buID,
		OrganizationID: orgID,
		CustomerID:     fcbk.ID,
	}
	if _, err := tx.NewInsert().Model(fcbkEmailProfile).Exec(ctx); err != nil {
		return fmt.Errorf("create Facebook email profile: %w", err)
	}

	return nil
}
