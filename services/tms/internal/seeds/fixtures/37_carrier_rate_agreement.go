package fixtures

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier/carrierdb"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate/formulatemplatedb"

	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement/rateagreementdb"

	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/places/domain/rategeo"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
)

type CarrierRateAgreementSeed struct {
}

var CarrierRateAgreement = Seeds.Add(seed.Seed{
	Name:        "carrier_rate_agreement",
	Kind:        seed.Fixture,
	Description: "Gives each active carrier an approved buy-side contract, so a carrier shop prices a lane",
	After:       []*seed.Seed{RateAgreement},
	Run:         (&CarrierRateAgreementSeed{}).Run,
})

var carrierBuyRates = []string{"1.65", "1.72", "1.80", "1.88", "1.95", "2.05", "2.15", "2.25"}

const carrierMinCharge = "450.00"

func (s *CarrierRateAgreementSeed) Run(ctx context.Context, sr *seed.Run) error {
	db := sr.DB
	org, err := defaultOrganization(ctx, sr)
	if err != nil {
		return err
	}
	admin, err := userByUsername(ctx, db, "admin")
	if err != nil {
		return err
	}

	templateID, err := s.perMileTemplate(ctx, db, org.ID, org.BusinessUnitID)
	if err != nil {
		return err
	}

	carriers, err := carrierdb.Carriers.AllTenants().Where(
		carrierdb.Carriers.OrganizationID.Eq(org.ID),
		carrierdb.Carriers.BusinessUnitID.Eq(org.BusinessUnitID),
		carrierdb.Carriers.Status.Eq(carrier.StatusActive),
	).Order(carrierdb.Carriers.Name.Asc()).All(ctx, db)
	if err != nil {
		return fmt.Errorf("load carriers: %w", err)
	}

	now := timeutils.NowUnix()
	for idx, entry := range carriers {
		rate := carrierBuyRates[idx%len(carrierBuyRates)]
		if err = s.ensure(ctx, db, sr, &carrierAgreementSpec{
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
	db *keel.DB,
	orgID, buID pulid.ID,
) (pulid.ID, error) {
	template, err := formulatemplatedb.FormulaTemplates.AllTenants().
		Pick(formulatemplatedb.FormulaTemplates.ID).
		Where(
			formulatemplatedb.FormulaTemplates.OrganizationID.Eq(orgID),
			formulatemplatedb.FormulaTemplates.BusinessUnitID.Eq(buID),
			formulatemplatedb.FormulaTemplates.Name.Eq(formulatemplate.StandardPerMile),
			formulatemplatedb.FormulaTemplates.Status.Eq(formulatemplate.StatusActive),
		).
		First(ctx, db)
	if err != nil {
		return pulid.Nil, fmt.Errorf(
			"load the %s template: %w",
			formulatemplate.StandardPerMile,
			err,
		)
	}

	return template.ID, nil
}

func (s *CarrierRateAgreementSeed) ensure(
	ctx context.Context,
	db *keel.DB,
	sr *seed.Run,
	spec *carrierAgreementSpec,
) error {
	carrierID := spec.carrier.ID

	approverID := spec.approverID
	stamped := spec.now
	agreement := &rateagreement.RateAgreement{
		ID:                   pulid.MustNew("rag_"),
		OrganizationID:       spec.carrier.OrganizationID,
		BusinessUnitID:       spec.carrier.BusinessUnitID,
		PartyType:            rateagreement.PartyTypeCarrier,
		CarrierID:            &carrierID,
		Code:                 "SEED-BUY-" + spec.carrier.Code,
		Name:                 spec.carrier.Name + " Linehaul Agreement",
		Description:          "Seeded buy-side contract so a carrier shop can price any lane",
		AgreementType:        rateagreement.AgreementTypeContract,
		Status:               rateagreement.StatusActive,
		Priority:             10,
		EffectiveFrom:        rateSeedEffectiveFrom,
		Currency:             "USD",
		Inputs:               tarifftypes.Inputs{},
		SubmittedByID:        &approverID,
		SubmittedAt:          &stamped,
		ApprovedByID:         &approverID,
		ApprovedAt:           &stamped,
		CreatedAt:            spec.now,
		UpdatedAt:            spec.now,
		RenewalNoticeDays:    30,
		CurrentVersionNumber: 1,
	}
	if err := rateagreementdb.RateAgreements.Create(ctx, db, agreement); err != nil {
		return fmt.Errorf("insert agreement: %w", err)
	}

	rule := &rateagreement.RateAgreementRule{
		ID:                   pulid.MustNew("ragr_"),
		OrganizationID:       agreement.OrganizationID,
		BusinessUnitID:       agreement.BusinessUnitID,
		RateAgreementID:      agreement.ID,
		PartyType:            agreement.PartyType,
		PartyID:              agreement.PartyID(),
		Label:                "Anywhere to anywhere, per mile",
		Status:               rateagreement.RuleStatusActive,
		OriginScopeType:      rategeo.ScopeTypeAny,
		DestinationScopeType: rategeo.ScopeTypeAny,
		Direction:            rateagreement.DirectionDirectional,
		FormulaTemplateID:    spec.templateID,
		Inputs: tarifftypes.Inputs{
			formulatemplate.StandardInputRate: rules.QuantityValue(
				decimal.RequireFromString(spec.rate),
				rules.UnitMoney+"/mi",
			),
			formulatemplate.StandardInputMinimumCharge: rules.MoneyValue(
				decimal.RequireFromString(carrierMinCharge),
			),
		},
		EffectiveFrom: rateSeedEffectiveFrom,
		CreatedAt:     spec.now,
		UpdatedAt:     spec.now,
	}
	if err := rateagreement.ApplyLanes(ctx, []*rateagreement.RateAgreementRule{rule}); err != nil {
		return fmt.Errorf("key the carrier lane: %w", err)
	}
	if err := rateagreementdb.RateAgreementRules.Create(ctx, db, rule); err != nil {
		return fmt.Errorf("insert rule: %w", err)
	}

	return nil
}
