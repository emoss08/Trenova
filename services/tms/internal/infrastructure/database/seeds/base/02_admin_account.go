package base

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/dedicatedlane"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/permissionbuilder"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/fatih/color"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

type AdminAccountSeed struct {
	seedhelpers.BaseSeed
}

func NewAdminAccountSeed() *AdminAccountSeed {
	seed := &AdminAccountSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"AdminAccount",
		"1.0.0",
		"Creates admin account data",
		[]common.Environment{
			common.EnvProduction, common.EnvStaging, common.EnvDevelopment, common.EnvTest,
		},
	)

	return seed
}

func (s *AdminAccountSeed) Run(
	ctx context.Context,
	db *bun.DB,
) error {
	return seedhelpers.RunInTransaction(
		ctx,
		db,
		s.Name(),
		func(ctx context.Context, tx bun.Tx, _ *seedhelpers.SeedContext) error {
			exists, err := db.NewSelect().
				Model((*tenant.User)(nil)).
				Where("email_address = ?", "admin@trenova.app").
				Exists(ctx)
			if err != nil {
				return fmt.Errorf("check admin exists: %w", err)
			}

			if exists {
				color.Yellow("→ Admin account already exists, skipping")
				return nil
			}

			var caState usstate.UsState
			err = db.NewSelect().
				Model(&caState).
				Where("abbreviation = ?", "CA").
				Scan(ctx)
			if err != nil {
				return fmt.Errorf("get California state: %w", err)
			}

			businessUnit := &tenant.BusinessUnit{
				Name:           "Default Business Unit",
				Code:           "DEFAULT",
				PrimaryContact: "System Administrator",
				PrimaryEmail:   "admin@trenova.app",
				PrimaryPhone:   "555-000-0000",
				AddressLine1:   "1 Market Street",
				City:           "Los Angeles",
				StateID:        caState.ID,
				PostalCode:     "90001",
				TaxID:          "00-0000000",
				Description:    "Default business unit created during system initialization",
				Timezone:       "America/Los_Angeles",
				Locale:         "en-US",
			}

			if _, err = db.NewInsert().Model(businessUnit).Exec(ctx); err != nil {
				return fmt.Errorf("create business unit: %w", err)
			}

			organization := &tenant.Organization{
				BusinessUnitID: businessUnit.ID,
				Name:           "Default Organization",
				AddressLine1:   "1 Market Street",
				City:           "Los Angeles",
				StateID:        caState.ID,
				PostalCode:     "90001",
				DOTNumber:      "0000000",
				ScacCode:       "DFLT",
				OrgType:        tenant.TypeCarrier,
				Timezone:       "America/Los_Angeles",
				BucketName:     "trenova-default",
				CreatedAt:      utils.NowUnix(),
				UpdatedAt:      utils.NowUnix(),
				State:          &caState,
			}

			if _, err = db.NewInsert().Model(organization).Exec(ctx); err != nil {
				return fmt.Errorf("create organization: %w", err)
			}

			adminUser := &tenant.User{
				CurrentOrganizationID: organization.ID,
				BusinessUnitID:        businessUnit.ID,
				Name:                  "System Administrator",
				Username:              "admin",
				EmailAddress:          "admin@trenova.app",
				Status:                domaintypes.StatusActive,
				Timezone:              "America/Los_Angeles",
				MustChangePassword:    false, // TODO(wolfred): Change this to true in production
				CreatedAt:             utils.NowUnix(),
				UpdatedAt:             utils.NowUnix(),
			}

			hashedPassword, err := adminUser.GeneratePassword("admin123!")
			if err != nil {
				return fmt.Errorf("generate password hash: %w", err)
			}
			adminUser.Password = hashedPassword

			if _, err = db.NewInsert().Model(adminUser).Exec(ctx); err != nil {
				return fmt.Errorf("create admin user: %w", err)
			}

			adminPolicy, adminRole, err := s.createAdminPermissions(
				ctx,
				db,
				businessUnit.ID,
				organization.ID,
			)
			if err != nil {
				return fmt.Errorf("create admin permissions: %w", err)
			}

			membership := &tenant.OrganizationMembership{
				BusinessUnitID: businessUnit.ID,
				UserID:         adminUser.ID,
				OrganizationID: organization.ID,
				RoleIDs:        []pulid.ID{adminRole.ID},
				DirectPolicies: []pulid.ID{},
				JoinedAt:       utils.NowUnix(),
				GrantedByID:    adminUser.ID,
				IsDefault:      true,
			}
			if _, err = db.NewInsert().Model(membership).Exec(ctx); err != nil {
				return fmt.Errorf("create organization membership: %w", err)
			}

			roleAssignment := &permission.RoleAssignment{
				UserID:         adminUser.ID,
				OrganizationID: organization.ID,
				RoleID:         adminRole.ID,
				AssignedBy:     adminUser.ID,
				AssignedAt:     utils.NowUnix(),
			}
			if _, err = db.NewInsert().Model(roleAssignment).Exec(ctx); err != nil {
				return fmt.Errorf("assign admin role: %w", err)
			}

			if err = s.createDefaultSettings(ctx, tx, organization.ID, businessUnit.ID); err != nil {
				return fmt.Errorf("create default settings: %w", err)
			}

			color.Green("✓ Created admin account:")
			fmt.Printf("  Email: admin@trenova.app\n")
			fmt.Printf("  Password: admin123!\n")
			fmt.Printf("  Policy: %s\n", adminPolicy.Name)
			fmt.Printf("  Role: %s\n", adminRole.Name)
			color.Yellow("  ⚠ IMPORTANT: Change the admin password after first login!")

			seedhelpers.LogSuccess("Created admin_account fixtures",
				"- Business Unit, Organization, Admin User",
				"- Admin Policy with full access",
				"- Administrator Role",
			)

			return nil
		},
	)
}

func (s *AdminAccountSeed) createAdminPermissions(
	ctx context.Context,
	db *bun.DB,
	businessUnitID, organizationID pulid.ID,
) (*permission.Policy, *permission.Role, error) {
	registry := permissionbuilder.CreatePermissionRegistry()

	adminPolicy := permissionbuilder.CreateAdminPolicy(
		"System Admin Policy",
		businessUnitID,
		[]pulid.ID{organizationID},
		registry,
	)

	if _, err := db.NewInsert().Model(adminPolicy).Exec(ctx); err != nil {
		return nil, nil, fmt.Errorf("create admin policy: %w", err)
	}

	adminRole := permissionbuilder.CreateAdminRole(businessUnitID, adminPolicy.ID)

	if _, err := db.NewInsert().Model(adminRole).Exec(ctx); err != nil {
		return nil, nil, fmt.Errorf("create admin role: %w", err)
	}

	return adminPolicy, adminRole, nil
}

func (s *AdminAccountSeed) createDefaultSettings(
	ctx context.Context,
	tx bun.Tx,
	orgID, buID pulid.ID,
) error {
	shipmentControl := &tenant.ShipmentControl{
		ID:             pulid.MustNew("sc_"),
		OrganizationID: orgID,
		BusinessUnitID: buID,
		CreatedAt:      utils.NowUnix(),
		UpdatedAt:      utils.NowUnix(),
	}

	if _, err := tx.NewInsert().Model(shipmentControl).Exec(ctx); err != nil {
		return fmt.Errorf("create shipment control: %w", err)
	}

	dispatchControl := &tenant.DispatchControl{
		ID:             pulid.MustNew("dc_"),
		OrganizationID: orgID,
		BusinessUnitID: buID,
		CreatedAt:      utils.NowUnix(),
		UpdatedAt:      utils.NowUnix(),
	}

	if _, err := tx.NewInsert().Model(dispatchControl).Exec(ctx); err != nil {
		return fmt.Errorf("create dispatch control: %w", err)
	}

	billingControl := &tenant.BillingControl{
		ID:             pulid.MustNew("bc_"),
		OrganizationID: orgID,
		BusinessUnitID: buID,
		CreatedAt:      utils.NowUnix(),
		UpdatedAt:      utils.NowUnix(),
	}

	if _, err := tx.NewInsert().Model(billingControl).Exec(ctx); err != nil {
		return fmt.Errorf("create billing control: %w", err)
	}

	dataRetention := &tenant.DataRetention{
		ID:             pulid.MustNew("dr_"),
		OrganizationID: orgID,
		BusinessUnitID: buID,
		CreatedAt:      utils.NowUnix(),
		UpdatedAt:      utils.NowUnix(),
	}

	if _, err := tx.NewInsert().Model(dataRetention).Exec(ctx); err != nil {
		return fmt.Errorf("create data retention: %w", err)
	}

	patternConfig := &dedicatedlane.PatternConfig{
		ID:                    pulid.MustNew("pco_"),
		BusinessUnitID:        buID,
		OrganizationID:        orgID,
		Enabled:               true,
		RequireExactMatch:     false,
		WeightRecentShipments: true,
		MinConfidenceScore:    decimal.NewFromFloat(0.7),
		MinFrequency:          3,
		AnalysisWindowDays:    90,
		SuggestionTTLDays:     30,
	}

	if _, err := tx.NewInsert().Model(patternConfig).Exec(ctx); err != nil {
		return fmt.Errorf("create pattern config: %w", err)
	}

	return nil
}
