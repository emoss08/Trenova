package base

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/tenantbootstrap"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

// AdminAccountSeed Creates AdminAccount data
type AdminAccountSeed struct {
	seedhelpers.BaseSeed
}

const (
	coreAdminUsername           = "admin"
	logisticsAdminUsername      = "admin-logistics"
	transportationAdminUsername = "admin-transport"
)

// NewAdminAccountSeed creates a new AdminAccount seed
func NewAdminAccountSeed() *AdminAccountSeed {
	seed := &AdminAccountSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"AdminAccount",
		"1.0.0",
		"Creates AdminAccount data",
		[]common.Environment{
			common.EnvDevelopment, common.EnvTest,
		},
	)

	seed.SetDependencies(seedhelpers.SeedUSStates, seedhelpers.SeedTestOrganizations)
	return seed
}

func (s *AdminAccountSeed) Run(ctx context.Context, tx bun.Tx) error {
	return seedhelpers.RunInTransaction(
		ctx,
		tx,
		s.Name(),
		nil,
		func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
			if err := requireDevelopmentFixtures(sc); err != nil {
				return err
			}

			// An existing core admin means this seed already owns the tenant
			// below it. Re-running would duplicate the second organization, its
			// users, memberships, sequences, and control files.
			seeded, err := tx.NewSelect().
				Model((*tenant.User)(nil)).
				Where("username = ?", coreAdminUsername).
				Exists(ctx)
			if err != nil {
				return fmt.Errorf("check existing admin account: %w", err)
			}
			if seeded {
				return nil
			}

			bu, err := sc.GetDefaultBusinessUnit(ctx)
			if err != nil {
				bu, err = sc.CreateBusinessUnit(ctx, tx, &seedhelpers.BusinessUnitOptions{
					Name: "Default Business Unit",
					Code: "DEFAULT",
				}, s.Name())
				if err != nil {
					return err
				}
				if err := sc.TrackCreated(ctx, "business_units", bu.ID, s.Name()); err != nil {
					return err
				}
			}
			if err := sc.Set("default_bu", bu); err != nil {
				return err
			}

			state, err := sc.GetState(ctx, "CA")
			if err != nil {
				return err
			}

			now := timeutils.NowUnix()

			org, err := sc.GetDefaultOrganization(ctx)
			if err != nil {
				org, err = sc.CreateOrganization(ctx, tx, &seedhelpers.OrganizationOptions{
					BusinessUnitID: bu.ID,
					Name:           "Trenova Logistics",
					ScacCode:       "TRNV",
					AddressLine1:   "1 Market Street",
					City:           "Los Angeles",
					StateID:        state.ID,
					PostalCode:     "90001",
					Timezone:       "America/Los_Angeles",
					TaxID:          "12-3456789",
					DOTNumber:      "1234567",
					BucketName:     "trenova-logistics",
				}, s.Name())
				if err != nil {
					return err
				}

				if err := sc.TrackCreated(ctx, "organizations", org.ID, s.Name()); err != nil {
					return err
				}
			}
			if err := sc.Set("default_org", org); err != nil {
				return err
			}

			org2, err := sc.CreateOrganization(ctx, tx, &seedhelpers.OrganizationOptions{
				BusinessUnitID: bu.ID,
				Name:           "Trenova Transportation",
				ScacCode:       "TTNV",
				AddressLine1:   "1 Market Street",
				City:           "Los Angeles",
				StateID:        state.ID,
				PostalCode:     "90001",
				Timezone:       "America/Los_Angeles",
				TaxID:          "12-3456789",
				DOTNumber:      "0000000",
				BucketName:     "trenova-transportation",
			}, s.Name())
			if err != nil {
				return err
			}

			if err := sc.TrackCreated(ctx, "organizations", org2.ID, s.Name()); err != nil {
				return err
			}

			orgs := []*tenant.Organization{org, org2}
			for _, seedOrg := range orgs {
				if err = tenantbootstrap.CreateControls(ctx, tx, s.scopeFor(sc, seedOrg, now)); err != nil {
					return fmt.Errorf("create control files for org %s: %w", seedOrg.Name, err)
				}
			}

			adminUser, err := sc.CreateUser(ctx, tx, &seedhelpers.UserOptions{
				OrganizationID:     org.ID,
				BusinessUnitID:     bu.ID,
				Name:               "System Administrator",
				Username:           coreAdminUsername,
				Email:              "admin@trenova.app",
				Password:           "admin123!",
				Status:             domaintypes.StatusActive,
				Timezone:           "America/Los_Angeles",
				MustChangePassword: false,
			}, s.Name())
			if err != nil {
				return err
			}

			membership := &tenant.OrganizationMembership{
				BusinessUnitID: org.BusinessUnitID,
				UserID:         adminUser.ID,
				JoinedAt:       timeutils.NowUnix(),
				OrganizationID: org.ID,
				GrantedByID:    adminUser.ID,
				IsDefault:      true,
			}
			if _, err = tx.NewInsert().Model(membership).Exec(ctx); err != nil {
				return err
			}
			if err := sc.TrackCreated(ctx, "organization_memberships", membership.ID, s.Name()); err != nil {
				return err
			}

			membership2 := &tenant.OrganizationMembership{
				BusinessUnitID: org2.BusinessUnitID,
				UserID:         adminUser.ID,
				JoinedAt:       timeutils.NowUnix(),
				OrganizationID: org2.ID,
				GrantedByID:    adminUser.ID,
				IsDefault:      false,
			}
			if _, err = tx.NewInsert().Model(membership2).Exec(ctx); err != nil {
				return err
			}
			if err := sc.TrackCreated(ctx, "organization_memberships", membership2.ID, s.Name()); err != nil {
				return err
			}

			orgAdminUsers := []organizationAdminUserSeedParams{
				{
					org:         org,
					grantedByID: adminUser.ID,
					name:        "Trenova Logistics Administrator",
					username:    logisticsAdminUsername,
					email:       "admin.logistics@trenova.app",
				},
				{
					org:         org2,
					grantedByID: adminUser.ID,
					name:        "Trenova Transportation Administrator",
					username:    transportationAdminUsername,
					email:       "admin.transport@trenova.app",
				},
			}
			for _, params := range orgAdminUsers {
				if err := s.createOrganizationAdminUser(ctx, tx, sc, params); err != nil {
					return err
				}
			}

			for _, seedOrg := range orgs {
				if err = tenantbootstrap.CreateSequences(ctx, tx, s.scopeFor(sc, seedOrg, now)); err != nil {
					return fmt.Errorf("create sequences for org %s: %w", seedOrg.Name, err)
				}
			}

			return nil
		},
	)
}

func (s *AdminAccountSeed) Down(ctx context.Context, tx bun.Tx) error {
	return seedhelpers.RunInTransaction(
		ctx,
		tx,
		s.Name(),
		nil,
		func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
			return seedhelpers.DeleteTrackedEntities(ctx, tx, s.Name(), sc)
		},
	)
}

func (s *AdminAccountSeed) CanRollback() bool {
	return true
}

type organizationAdminUserSeedParams struct {
	org         *tenant.Organization
	grantedByID pulid.ID
	name        string
	username    string
	email       string
}

func (s *AdminAccountSeed) scopeFor(
	sc *seedhelpers.SeedContext,
	org *tenant.Organization,
	now int64,
) tenantbootstrap.Scope {
	return tenantbootstrap.Scope{
		OrganizationID: org.ID,
		BusinessUnitID: org.BusinessUnitID,
		Now:            now,
		Record:         seedRecorder(sc, s.Name()),
	}
}

func (s *AdminAccountSeed) createOrganizationAdminUser(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
	params organizationAdminUserSeedParams,
) error {
	adminUser, err := sc.CreateUser(ctx, tx, &seedhelpers.UserOptions{
		OrganizationID:     params.org.ID,
		BusinessUnitID:     params.org.BusinessUnitID,
		Name:               params.name,
		Username:           params.username,
		Email:              params.email,
		Password:           "admin123!",
		Status:             domaintypes.StatusActive,
		Timezone:           "America/Los_Angeles",
		MustChangePassword: false,
	}, s.Name())
	if err != nil {
		return fmt.Errorf("create organization admin user %s: %w", params.username, err)
	}

	membership := &tenant.OrganizationMembership{
		BusinessUnitID: params.org.BusinessUnitID,
		UserID:         adminUser.ID,
		JoinedAt:       timeutils.NowUnix(),
		OrganizationID: params.org.ID,
		GrantedByID:    params.grantedByID,
		IsDefault:      true,
	}
	if _, err = tx.NewInsert().Model(membership).Exec(ctx); err != nil {
		return fmt.Errorf("create organization admin membership %s: %w", params.username, err)
	}
	if err := sc.TrackCreated(ctx, "organization_memberships", membership.ID, s.Name()); err != nil {
		return err
	}

	return nil
}
