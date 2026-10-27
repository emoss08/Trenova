package fixtures

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/reference/domain/holdreason/holdreasondb"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/modules/reference/domain/holdreason"
)

type HoldReasonSeed struct {
}

var HoldReason = Seeds.Add(seed.Seed{
	Name:        "hold_reason",
	Kind:        seed.Fixture,
	Description: "Seeds the hold reasons dispatch, compliance, customer service and billing place holds with",
	After:       []*seed.Seed{AdminAccount},
	Run:         (&HoldReasonSeed{}).Run,
})

func (s *HoldReasonSeed) Run(ctx context.Context, sr *seed.Run) error {
	db := sr.DB
	org, err := defaultOrganization(ctx, sr)
	if err != nil {
		return err
	}

	for idx, reason := range seededHoldReasons() {
		reason.OrganizationID = org.ID
		reason.BusinessUnitID = org.BusinessUnitID
		reason.Active = true
		reason.SortOrder = int32((idx + 1) * 10)
		if err = s.ensure(ctx, db, reason); err != nil {
			return fmt.Errorf("ensure hold reason %q: %w", reason.Code, err)
		}
	}

	return nil
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

			SortOrder: 100,
		},
		{
			Type:                  holdreason.HoldTypeCompliance,
			Code:                  "DRIVER_QUAL",
			Label:                 "Driver not qualified",
			Description:           "The assigned driver's credentials do not cover this load.",
			DefaultSeverity:       holdreason.HoldSeverityBlocking,
			DefaultBlocksDispatch: true,

			SortOrder: 100,
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

			SortOrder: 100,
		},
		{
			Type:            holdreason.HoldTypeOperational,
			Code:            "APPT_PENDING",
			Label:           "Appointment not confirmed",
			Description:     "The receiver has not confirmed a delivery appointment.",
			DefaultSeverity: holdreason.HoldSeverityAdvisory,

			SortOrder: 100,
		},
		{
			Type:                  holdreason.HoldTypeOperational,
			Code:                  "MISSED_PICKUP",
			Label:                 "Missed pickup",
			Description:           "The pickup window passed with nobody at the shipper; it needs a new appointment.",
			DefaultSeverity:       holdreason.HoldSeverityBlocking,
			DefaultBlocksDispatch: true,

			SortOrder: 100,
		},
		{
			Type:                  holdreason.HoldTypeOperational,
			Code:                  "EQUIP_ISSUE",
			Label:                 "Equipment problem",
			Description:           "The tractor or trailer for the load is out of service or unsuitable.",
			DefaultSeverity:       holdreason.HoldSeverityBlocking,
			DefaultBlocksDispatch: true,

			SortOrder: 100,
		},
		{
			Type:                 holdreason.HoldTypeFinance,
			Code:                 "CREDIT_HOLD",
			Label:                "Customer over credit limit",
			Description:          "Billing will not release the load until the account is current.",
			DefaultSeverity:      holdreason.HoldSeverityBlocking,
			DefaultBlocksBilling: true,

			SortOrder: 100,
		},
		{
			Type:                 holdreason.HoldTypeFinance,
			Code:                 "RATE_DISPUTE",
			Label:                "Rate in dispute",
			Description:          "The customer disputes the rate; billing waits for it to be settled.",
			DefaultSeverity:      holdreason.HoldSeverityAdvisory,
			DefaultBlocksBilling: true,

			SortOrder: 100,
		},
	}
}

func (s *HoldReasonSeed) ensure(ctx context.Context, db *keel.DB, reason *holdreason.HoldReason) error {
	return holdreasondb.HoldReasons.Create(ctx, db, reason)
}
