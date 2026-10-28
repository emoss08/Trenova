package fixtures

import (
	"context"
	"fmt"

	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/tenantbootstrap"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/seeds/reference"
	"github.com/emoss08/trenova/internal/seeds/seedenv"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	adminPassword = "admin123!"
	adminEmail    = "admin@trenova.app"
	devTimezone   = "America/Los_Angeles"
)

var AdminAccount = Seeds.Add(seed.Seed{
	Name: "admin_account",
	Kind: seed.Fixture,
	Description: "The DEFAULT business unit, the Trenova Logistics and Trenova Transportation " +
		"organizations with their controls, sequences, reference data and administrator role, " +
		"and the admin user (admin / admin123!) who belongs to both",
	After: []*seed.Seed{reference.USStates},
	Run:   runAdminAccount,
})

var SystemAccount = Seeds.Add(seed.Seed{
	Name:        "system_account",
	Kind:        seed.Fixture,
	Description: "The system user background work runs as, with the configured password",
	After:       []*seed.Seed{AdminAccount},
	Run:         runSystemAccount,
})

type devOrganization struct {
	name       string
	scac       string
	dotNumber  string
	bucketName string
}

var devOrganizations = []devOrganization{
	{name: defaultOrgName, scac: "TRNV", dotNumber: "1234567", bucketName: "trenova-logistics"},
	{
		name:       "Trenova Transportation",
		scac:       secondOrgScac,
		dotNumber:  "0000000",
		bucketName: "trenova-transportation",
	},
}

func runAdminAccount(ctx context.Context, r *seed.Run) error {
	if err := requireDevelopment(r); err != nil {
		return err
	}

	state, err := stateByAbbreviation(ctx, r.DB, "CA")
	if err != nil {
		return err
	}

	now := timeutils.NowUnix()
	bu := &tenant.BusinessUnit{Name: "Default Business Unit", Code: defaultBUCode}
	if err = tenantdb.BusinessUnits.Create(ctx, r.DB, bu); err != nil {
		return fmt.Errorf("create business unit: %w", err)
	}

	orgs := make([]*tenant.Organization, 0, len(devOrganizations))
	for _, def := range devOrganizations {
		org := &tenant.Organization{
			BusinessUnitID:         bu.ID,
			Name:                   def.name,
			ScacCode:               def.scac,
			AddressLine1:           "1 Market Street",
			City:                   "Los Angeles",
			StateID:                state.ID,
			PostalCode:             "90001",
			Timezone:               devTimezone,
			TaxID:                  "12-3456789",
			DOTNumber:              def.dotNumber,
			BucketName:             def.bucketName,
			BrokerageEnabled:       true,
			AssetOperationsEnabled: true,
			Locale:                 "en",
		}
		if err = tenantdb.Organizations.Create(ctx, r.DB, org); err != nil {
			return fmt.Errorf("create organization %s: %w", def.name, err)
		}

		scope := tenantbootstrap.Scope{OrganizationID: org.ID, BusinessUnitID: bu.ID, Now: now}
		if err = tenantbootstrap.CreateControls(ctx, r.DB, scope); err != nil {
			return fmt.Errorf("create control files for %s: %w", def.name, err)
		}
		if err = tenantbootstrap.CreateSequences(ctx, r.DB, scope); err != nil {
			return fmt.Errorf("create sequences for %s: %w", def.name, err)
		}
		if err = tenantbootstrap.CreateReferenceData(ctx, r.DB, scope); err != nil {
			return fmt.Errorf("create reference data for %s: %w", def.name, err)
		}
		orgs = append(orgs, org)
	}
	seed.Put(r, defaultOrgKey, orgs[0])

	admin := &tenant.User{
		CurrentOrganizationID: orgs[0].ID,
		BusinessUnitID:        bu.ID,
		Name:                  "System Administrator",
		Username:              adminUsername,
		EmailAddress:          adminEmail,
		Status:                domaintypes.StatusActive,
		Timezone:              devTimezone,
		TimeFormat:            domaintypes.TimeFormat12Hour,
		Locale:                "en",
		AvatarStyle:           tenant.AvatarStyleAuto,
		AvailabilityStatus:    tenant.AvailabilityAvailable,
	}
	if admin.Password, err = admin.GeneratePassword(adminPassword); err != nil {
		return fmt.Errorf("hash admin password: %w", err)
	}
	if err = tenantdb.Users.Create(ctx, r.DB, admin); err != nil {
		return fmt.Errorf("create admin user: %w", err)
	}

	registry := permissioncatalog.NewRegistry()
	for i, org := range orgs {
		membership := &tenant.OrganizationMembership{
			BusinessUnitID: bu.ID,
			UserID:         admin.ID,
			OrganizationID: org.ID,
			JoinedAt:       now,
			GrantedByID:    admin.ID,
			IsDefault:      i == 0,
		}
		if err = tenantdb.OrganizationMemberships.Create(ctx, r.DB, membership); err != nil {
			return fmt.Errorf("add admin to %s: %w", org.Name, err)
		}

		scope := tenantbootstrap.Scope{OrganizationID: org.ID, BusinessUnitID: bu.ID, Now: now}
		role, roleErr := tenantbootstrap.CreateAdminRole(ctx, r.DB, tenantbootstrap.AdminRoleParams{
			Scope:     scope,
			CreatedBy: admin.ID,
			Registry:  registry,
		})
		if roleErr != nil {
			return fmt.Errorf("create admin role for %s: %w", org.Name, roleErr)
		}
		if err = tenantbootstrap.AssignRole(ctx, r.DB, tenantbootstrap.RoleAssignmentParams{
			UserID:         admin.ID,
			OrganizationID: org.ID,
			RoleID:         role.ID,
			AssignedBy:     admin.ID,
			AssignedAt:     now,
		}); err != nil {
			return fmt.Errorf("assign admin role in %s: %w", org.Name, err)
		}
	}
	r.Count(int64(len(orgs)))

	return nil
}

func runSystemAccount(ctx context.Context, r *seed.Run) error {
	org, err := defaultOrganization(ctx, r)
	if err != nil {
		return err
	}

	_, created, err := tenantbootstrap.EnsureSystemUser(ctx, r.DB, tenantbootstrap.SystemUserParams{
		Scope:    tenantbootstrap.Scope{OrganizationID: org.ID, BusinessUnitID: org.BusinessUnitID},
		Password: seedenv.Config(ctx).System.SystemUserPassword,
	})
	if created {
		r.Count(1)
	}

	return err
}
