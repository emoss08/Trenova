package base

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/dataentrycontrol"
	"github.com/emoss08/trenova/internal/core/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
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

// NewAdminAccountSeed creates a new AdminAccount seed
func NewAdminAccountSeed() *AdminAccountSeed {
	seed := &AdminAccountSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"AdminAccount",
		"1.0.0",
		"Creates AdminAccount data",
		[]common.Environment{
			common.EnvProduction, common.EnvStaging, common.EnvDevelopment, common.EnvTest,
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

			accountingControl := &tenant.AccountingControl{
				ID:             pulid.MustNew("ac_"),
				OrganizationID: org.ID,
				BusinessUnitID: bu.ID,
				CreatedAt:      now,
				UpdatedAt:      now,
			}

			if _, err := tx.NewInsert().Model(accountingControl).Exec(ctx); err != nil {
				return fmt.Errorf("create accounting control: %w", err)
			}

			billingControl := &tenant.BillingControl{
				ID:             pulid.MustNew("bc_"),
				OrganizationID: org.ID,
				BusinessUnitID: bu.ID,
				CreatedAt:      now,
				UpdatedAt:      now,
			}
			if _, err := tx.NewInsert().Model(billingControl).Exec(ctx); err != nil {
				return fmt.Errorf("create billing control: %w", err)
			}
			if err := sc.TrackCreated(ctx, "billing_controls", billingControl.ID, s.Name()); err != nil {
				return err
			}

			dispatchControl := &dispatchcontrol.DispatchControl{
				ID:             pulid.MustNew("dc_"),
				OrganizationID: org.ID,
				BusinessUnitID: bu.ID,
				CreatedAt:      now,
				UpdatedAt:      now,
			}
			if _, err := tx.NewInsert().Model(dispatchControl).Exec(ctx); err != nil {
				return fmt.Errorf("create dispatch control: %w", err)
			}

			if err := sc.TrackCreated(ctx, "dispatch_controls", dispatchControl.ID, s.Name()); err != nil {
				return err
			}

			shipmentControl := &tenant.ShipmentControl{
				ID:             pulid.MustNew("sc_"),
				OrganizationID: org.ID,
				BusinessUnitID: bu.ID,
				CreatedAt:      now,
				UpdatedAt:      now,
			}
			if _, err := tx.NewInsert().Model(shipmentControl).Exec(ctx); err != nil {
				return fmt.Errorf("create shipment control: %w", err)
			}
			if err := sc.TrackCreated(ctx, "shipment_controls", shipmentControl.ID, s.Name()); err != nil {
				return err
			}

			documentControl := tenant.NewDefaultDocumentControl(org.ID, bu.ID)
			if _, err := tx.NewInsert().Model(documentControl).Exec(ctx); err != nil {
				return fmt.Errorf("create document control: %w", err)
			}
			if err := sc.TrackCreated(ctx, "document_controls", documentControl.ID, s.Name()); err != nil {
				return err
			}

			dataEntryControl := &dataentrycontrol.DataEntryControl{
				ID:             pulid.MustNew("dec_"),
				OrganizationID: org.ID,
				BusinessUnitID: bu.ID,
				CreatedAt:      now,
				UpdatedAt:      now,
				CodeCase:       dataentrycontrol.CaseFormatUpper,
				NameCase:       dataentrycontrol.CaseFormatTitleCase,
				EmailCase:      dataentrycontrol.CaseFormatLower,
				CityCase:       dataentrycontrol.CaseFormatTitleCase,
			}
			if _, err := tx.NewInsert().Model(dataEntryControl).Exec(ctx); err != nil {
				return fmt.Errorf("create data entry control: %w", err)
			}
			if err := sc.TrackCreated(ctx, "data_entry_controls", dataEntryControl.ID, s.Name()); err != nil {
				return err
			}

			adminUser, err := sc.CreateUser(ctx, tx, &seedhelpers.UserOptions{
				OrganizationID:     org.ID,
				BusinessUnitID:     bu.ID,
				Name:               "System Administrator",
				Username:           "admin",
				Email:              "admin@trenova.app",
				Password:           "admin123!",
				Status:             domaintypes.StatusActive,
				Timezone:           "America/Los_Angeles",
				IsPlatformAdmin:    true,
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

			year := int16(time.Unix(now, 0).Year())
			month := int16(time.Unix(now, 0).Month())

			sequences := []*tenant.Sequence{
				{
					ID:              pulid.MustNew("seq_"),
					SequenceType:    tenant.SequenceTypeProNumber,
					OrganizationID:  org.ID,
					BusinessUnitID:  bu.ID,
					Year:            year,
					Month:           month,
					CurrentSequence: 0,
					Version:         0,
					CreatedAt:       now,
					UpdatedAt:       now,
				},
				{
					ID:              pulid.MustNew("seq_"),
					SequenceType:    tenant.SequenceTypeConsolidation,
					OrganizationID:  org.ID,
					BusinessUnitID:  bu.ID,
					Year:            year,
					Month:           month,
					CurrentSequence: 0,
					Version:         0,
					CreatedAt:       now,
					UpdatedAt:       now,
				},
				{
					ID:              pulid.MustNew("seq_"),
					SequenceType:    tenant.SequenceTypeInvoice,
					OrganizationID:  org.ID,
					BusinessUnitID:  bu.ID,
					Year:            year,
					Month:           month,
					CurrentSequence: 0,
					Version:         0,
					CreatedAt:       now,
					UpdatedAt:       now,
				},
				{
					ID:              pulid.MustNew("seq_"),
					SequenceType:    tenant.SequenceTypeWorkOrder,
					OrganizationID:  org.ID,
					BusinessUnitID:  bu.ID,
					Year:            year,
					Month:           month,
					CurrentSequence: 0,
					Version:         0,
					CreatedAt:       now,
					UpdatedAt:       now,
				},
				{
					ID:              pulid.MustNew("seq_"),
					SequenceType:    tenant.SequenceTypeProNumber,
					OrganizationID:  org2.ID,
					BusinessUnitID:  bu.ID,
					Year:            year,
					Month:           month,
					CurrentSequence: 0,
					Version:         0,
					CreatedAt:       now,
					UpdatedAt:       now,
				},
				{
					ID:              pulid.MustNew("seq_"),
					SequenceType:    tenant.SequenceTypeConsolidation,
					OrganizationID:  org2.ID,
					BusinessUnitID:  bu.ID,
					Year:            year,
					Month:           month,
					CurrentSequence: 0,
					Version:         0,
					CreatedAt:       now,
					UpdatedAt:       now,
				},
				{
					ID:              pulid.MustNew("seq_"),
					SequenceType:    tenant.SequenceTypeInvoice,
					OrganizationID:  org2.ID,
					BusinessUnitID:  bu.ID,
					Year:            year,
					Month:           month,
					CurrentSequence: 0,
					Version:         0,
					CreatedAt:       now,
					UpdatedAt:       now,
				},
				{
					ID:              pulid.MustNew("seq_"),
					SequenceType:    tenant.SequenceTypeWorkOrder,
					OrganizationID:  org2.ID,
					BusinessUnitID:  bu.ID,
					Year:            year,
					Month:           month,
					CurrentSequence: 0,
					Version:         0,
					CreatedAt:       now,
					UpdatedAt:       now,
				},
			}

			for _, sequence := range sequences {
				result, insertErr := tx.NewInsert().
					Model(sequence).
					On("CONFLICT (sequence_type, organization_id, business_unit_id, year, month) DO NOTHING").
					Exec(ctx)
				if insertErr != nil {
					return fmt.Errorf("create sequence seed: %w", insertErr)
				}

				if rows, _ := result.RowsAffected(); rows > 0 {
					if err := sc.TrackCreated(ctx, "sequences", sequence.ID, s.Name()); err != nil {
						return err
					}
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
