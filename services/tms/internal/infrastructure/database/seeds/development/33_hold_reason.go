package development

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/holdreason"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/uptrace/bun"
)

type HoldReasonSeed struct {
	seedhelpers.BaseSeed
}

func NewHoldReasonSeed() *HoldReasonSeed {
	seed := &HoldReasonSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"HoldReason",
		"1.0.0",
		"Seeds the hold reasons dispatch, compliance, customer service and billing place holds with",
		[]common.Environment{common.EnvDevelopment},
	)
	seed.SetDependencies(seedhelpers.SeedTestOrganizations)

	return seed
}

func (s *HoldReasonSeed) Run(ctx context.Context, tx bun.Tx) error {
	return seedhelpers.RunInTransaction(
		ctx,
		tx,
		s.Name(),
		nil,
		func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
			org, err := sc.GetDefaultOrganization(ctx)
			if err != nil {
				return err
			}

			for idx, reason := range seededHoldReasons() {
				reason.OrganizationID = org.ID
				reason.BusinessUnitID = org.BusinessUnitID
				reason.Active = true
				reason.SortOrder = int32((idx + 1) * 10)
				if err = s.ensure(ctx, tx, reason); err != nil {
					return fmt.Errorf("ensure hold reason %q: %w", reason.Code, err)
				}
			}

			return nil
		},
	)
}

func seededHoldReasons() []*holdreason.HoldReason {
	return []*holdreason.HoldReason{
		{
			Type:                  holdreason.HoldTypeCompliance,
			Code:                  "HAZMAT_DOCS",
			Label:                 "Hazmat paperwork missing",
			Description:           "Shipping papers or placards are missing or do not match the load.",
			DefaultSeverity:       holdreason.HoldSeverityBlocking,
			DefaultBlocksDispatch: true,
			DefaultBlocksDelivery: true,
		},
		{
			Type:                  holdreason.HoldTypeCompliance,
			Code:                  "DRIVER_QUAL",
			Label:                 "Driver not qualified",
			Description:           "The assigned driver's credentials do not cover this load.",
			DefaultSeverity:       holdreason.HoldSeverityBlocking,
			DefaultBlocksDispatch: true,
		},
		{
			Type:                     holdreason.HoldTypeCustomer,
			Code:                     "CUST_REQUEST",
			Label:                    "Customer asked to hold",
			Description:              "The customer asked us not to move or deliver the load yet.",
			DefaultSeverity:          holdreason.HoldSeverityBlocking,
			DefaultBlocksDispatch:    true,
			DefaultBlocksDelivery:    true,
			DefaultVisibleToCustomer: true,
		},
		{
			Type:            holdreason.HoldTypeOperational,
			Code:            "APPT_PENDING",
			Label:           "Appointment not confirmed",
			Description:     "The receiver has not confirmed a delivery appointment.",
			DefaultSeverity: holdreason.HoldSeverityAdvisory,
		},
		{
			Type:                  holdreason.HoldTypeOperational,
			Code:                  "EQUIP_ISSUE",
			Label:                 "Equipment problem",
			Description:           "The tractor or trailer for the load is out of service or unsuitable.",
			DefaultSeverity:       holdreason.HoldSeverityBlocking,
			DefaultBlocksDispatch: true,
		},
		{
			Type:                 holdreason.HoldTypeFinance,
			Code:                 "CREDIT_HOLD",
			Label:                "Customer over credit limit",
			Description:          "Billing will not release the load until the account is current.",
			DefaultSeverity:      holdreason.HoldSeverityBlocking,
			DefaultBlocksBilling: true,
		},
		{
			Type:                 holdreason.HoldTypeFinance,
			Code:                 "RATE_DISPUTE",
			Label:                "Rate in dispute",
			Description:          "The customer disputes the rate; billing waits for it to be settled.",
			DefaultSeverity:      holdreason.HoldSeverityAdvisory,
			DefaultBlocksBilling: true,
		},
	}
}

func (s *HoldReasonSeed) ensure(ctx context.Context, tx bun.Tx, reason *holdreason.HoldReason) error {
	cols := buncolgen.HoldReasonColumns
	exists, err := tx.NewSelect().
		Model((*holdreason.HoldReason)(nil)).
		Where(cols.OrganizationID.Eq(), reason.OrganizationID).
		Where(cols.BusinessUnitID.Eq(), reason.BusinessUnitID).
		Where(cols.Code.Eq(), reason.Code).
		Exists(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if exists {
		return nil
	}

	_, err = tx.NewInsert().Model(reason).Exec(ctx)

	return err
}
