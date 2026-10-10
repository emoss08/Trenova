package development

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/domain/rategeo"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

type CarrierRateAgreementSeed struct {
	seedhelpers.BaseSeed
}

func NewCarrierRateAgreementSeed() *CarrierRateAgreementSeed {
	seed := &CarrierRateAgreementSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"CarrierRateAgreement",
		"1.0.0",
		"Gives each active carrier an approved buy-side contract, so a carrier shop prices a lane",
		[]common.Environment{common.EnvDevelopment},
	)
	seed.SetDependencies(seedhelpers.SeedRateAgreement)

	return seed
}

var carrierBuyRates = []string{"1.65", "1.72", "1.80", "1.88", "1.95", "2.05", "2.15", "2.25"}

const carrierMinCharge = "450.00"

func (s *CarrierRateAgreementSeed) Run(ctx context.Context, tx bun.Tx) error {
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
			admin, err := sc.GetUserByUsername(ctx, "admin")
			if err != nil {
				return err
			}

			templateID, err := s.perMileTemplate(ctx, tx, org.ID, org.BusinessUnitID)
			if err != nil {
				return err
			}

			var carriers []*carrier.Carrier
			cols := buncolgen.CarrierColumns
			if err = tx.NewSelect().
				Model(&carriers).
				Where(cols.OrganizationID.Eq(), org.ID).
				Where(cols.BusinessUnitID.Eq(), org.BusinessUnitID).
				Where(cols.Status.Eq(), carrier.StatusActive).
				Order(cols.Name.OrderAsc()).
				Scan(ctx); err != nil {
				return fmt.Errorf("load carriers: %w", err)
			}

			now := timeutils.NowUnix()
			for idx, entry := range carriers {
				rate := carrierBuyRates[idx%len(carrierBuyRates)]
				if err = s.ensure(ctx, tx, sc, &carrierAgreementSpec{
					carrier:    entry,
					templateID: templateID,
					approverID: admin.ID,
					rate:       rate,
					now:        now,
				}); err != nil {
					return fmt.Errorf("carrier agreement for %s: %w", entry.Name, err)
				}
			}

			return nil
		},
	)
}

type carrierAgreementSpec struct {
	carrier    *carrier.Carrier
	templateID pulid.ID
	approverID pulid.ID
	rate       string
	now        int64
}

func (s *CarrierRateAgreementSeed) perMileTemplate(
	ctx context.Context,
	tx bun.Tx,
	orgID, buID pulid.ID,
) (pulid.ID, error) {
	template := new(formulatemplate.FormulaTemplate)
	if err := tx.NewSelect().
		Model(template).
		Column("id").
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Where("name = ?", formulatemplate.StandardPerMile).
		Where("status = ?", formulatemplate.StatusActive).
		Limit(1).
		Scan(ctx); err != nil {
		return pulid.Nil, fmt.Errorf("load the %s template: %w", formulatemplate.StandardPerMile, err)
	}

	return template.ID, nil
}

func (s *CarrierRateAgreementSeed) ensure(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
	spec *carrierAgreementSpec,
) error {
	cols := buncolgen.RateAgreementColumns
	carrierID := spec.carrier.ID
	exists, err := tx.NewSelect().
		Model((*rateagreement.RateAgreement)(nil)).
		Where(cols.OrganizationID.Eq(), spec.carrier.OrganizationID).
		Where(cols.BusinessUnitID.Eq(), spec.carrier.BusinessUnitID).
		Where(cols.CarrierID.Eq(), carrierID).
		Exists(ctx)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	approverID := spec.approverID
	stamped := spec.now
	agreement := &rateagreement.RateAgreement{
		ID:                pulid.MustNew("rag_"),
		OrganizationID:    spec.carrier.OrganizationID,
		BusinessUnitID:    spec.carrier.BusinessUnitID,
		PartyType:         rateagreement.PartyTypeCarrier,
		CarrierID:         &carrierID,
		Code:              "SEED-BUY-" + spec.carrier.Code,
		Name:              spec.carrier.Name + " Linehaul Agreement",
		Description:       "Seeded buy-side contract so a carrier shop can price any lane",
		AgreementType:     rateagreement.AgreementTypeContract,
		Status:            rateagreement.StatusActive,
		Priority:          10,
		EffectiveFrom:     rateSeedEffectiveFrom,
		Currency:          "USD",
		RoundingMode:      ratetypes.RoundingModeHalfUp,
		RoundingPrecision: 2,
		SubmittedByID:     &approverID,
		SubmittedAt:       &stamped,
		ApprovedByID:      &approverID,
		ApprovedAt:        &stamped,
		CreatedAt:         spec.now,
		UpdatedAt:         spec.now,
	}
	if _, err = tx.NewInsert().Model(agreement).Exec(ctx); err != nil {
		return fmt.Errorf("insert agreement: %w", err)
	}
	if err = sc.TrackCreated(ctx, "rate_agreements", agreement.ID, s.Name()); err != nil {
		return fmt.Errorf("track agreement: %w", err)
	}

	templateID := spec.templateID
	rule := &rateagreement.RateAgreementRule{
		ID:                   pulid.MustNew("ragr_"),
		OrganizationID:       agreement.OrganizationID,
		BusinessUnitID:       agreement.BusinessUnitID,
		RateAgreementID:      agreement.ID,
		PartyType:            agreement.PartyType,
		PartyID:              agreement.PartyID(),
		Label:                "Anywhere to anywhere, per mile",
		Status:               rateagreement.RuleStatusActive,
		AllowDeficitRating:   true,
		OriginScopeType:      rategeo.ScopeTypeAny,
		DestinationScopeType: rategeo.ScopeTypeAny,
		Direction:            rateagreement.DirectionDirectional,
		FormulaTemplateID:    &templateID,
		Rate:                 decimal.NewNullDecimal(decimal.RequireFromString(spec.rate)),
		MinCharge:            decimal.NewNullDecimal(decimal.RequireFromString(carrierMinCharge)),
		EffectiveFrom:        rateSeedEffectiveFrom,
		CreatedAt:            spec.now,
		UpdatedAt:            spec.now,
	}
	rule.ApplyLaneKey()
	rule.SpecificityScore = rule.ComputeSpecificity()
	if _, err = tx.NewInsert().Model(rule).Exec(ctx); err != nil {
		return fmt.Errorf("insert rule: %w", err)
	}

	return nil
}
