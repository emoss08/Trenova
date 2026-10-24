package reference

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate/usstatedb"
	"github.com/emoss08/trenova/internal/modules/places/domain/jurisdictionrule"
	"github.com/emoss08/trenova/internal/modules/places/domain/jurisdictionrule/jurisdictionruledb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/dothazmatreference"
	"github.com/emoss08/trenova/internal/modules/reference/domain/dothazmatreference/dothazmatreferencedb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/ifta/iftadb"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	usStatesFile      = "data/us_states.yaml"
	hazmatFile        = "data/dot_hazmat_references.yaml"
	iftaFile          = "data/ifta_jurisdictions.yaml"
	hazmatUpsertBatch = 1000

	baselinePermitLeadTimeDays = 1
	baselinePermitValidityDays = 5
)

const baselineSourceNote = "Federal baseline only: 102in width (23 CFR 658.15) and " +
	"80,000lb Interstate gross (23 U.S.C. 127). Height and length are the conservative " +
	"lower bound of common state limits, not this state's statute. Permit lead time, " +
	"validity, fees, superload thresholds, escort requirements and travel curfews are " +
	"NOT populated. Confirm against the state permit office and mark this row Verified " +
	"before relying on it."

const baselineSourceURL = "https://www.fhwa.dot.gov/policy/otps/truck/"

var USStates = Seeds.Add(seed.Seed{
	Name:        "us_states",
	Kind:        seed.Reference,
	Description: "US states, keyed by abbreviation",
	Inputs:      seed.Files(data, usStatesFile),
	Run:         runUSStates,
})

var DotHazmatReferences = Seeds.Add(seed.Seed{
	Name:        "dot_hazmat_references",
	Kind:        seed.Reference,
	Description: "DOT hazardous materials table (49 CFR 172.101), keyed by id",
	Inputs:      seed.Files(data, hazmatFile),
	Run:         runDotHazmatReferences,
})

var IFTAJurisdictions = Seeds.Add(seed.Seed{
	Name:        "ifta_jurisdictions",
	Kind:        seed.Reference,
	Description: "IFTA jurisdictions for US states, DC and Canadian provinces",
	After:       []*seed.Seed{USStates},
	Inputs:      seed.Files(data, iftaFile),
	Run:         runIFTAJurisdictions,
})

var JurisdictionRulesBaseline = Seeds.Add(seed.Seed{
	Name:        "jurisdiction_rules_baseline",
	Kind:        seed.Reference,
	Description: "An unverified federal-baseline oversize rule for every state without a rule",
	After:       []*seed.Seed{USStates},
	Version:     "1",
	Run:         runJurisdictionRulesBaseline,
})

type usStateRow struct {
	Name         string `yaml:"name"`
	Abbreviation string `yaml:"abbreviation"`
	CountryName  string `yaml:"country_name"`
	CountryIso3  string `yaml:"country_iso3"`
}

func runUSStates(ctx context.Context, r *seed.Run) error {
	file, err := seed.YAML[struct {
		States []usStateRow `yaml:"states"`
	}](r, usStatesFile)
	if err != nil {
		return err
	}

	states := make([]*usstate.UsState, 0, len(file.States))
	for _, row := range file.States {
		states = append(states, &usstate.UsState{
			Name:         row.Name,
			Abbreviation: row.Abbreviation,
			CountryName:  row.CountryName,
			CountryIso3:  row.CountryIso3,
		})
	}

	s := usstatedb.UsStates
	if _, err = s.Upsert(ctx, r.DB, states,
		keel.OnConflict(s.Abbreviation).Update(s.Name, s.CountryName, s.CountryIso3),
	); err != nil {
		return fmt.Errorf("upsert us states: %w", err)
	}
	r.Count(int64(len(states)))

	return nil
}

type hazmatRow struct {
	ID                  string `yaml:"id"`
	UnNumber            string `yaml:"un_number"`
	ProperShippingName  string `yaml:"proper_shipping_name"`
	HazardClass         string `yaml:"hazard_class"`
	SubsidiaryHazard    string `yaml:"subsidiary_hazard"`
	PackingGroup        string `yaml:"packing_group"`
	SpecialProvisions   string `yaml:"special_provisions"`
	PackagingExceptions string `yaml:"packaging_exceptions"`
	PackagingNonBulk    string `yaml:"packaging_non_bulk"`
	PackagingBulk       string `yaml:"packaging_bulk"`
	QuantityPassenger   string `yaml:"quantity_passenger"`
	QuantityCargo       string `yaml:"quantity_cargo"`
	VesselStowage       string `yaml:"vessel_stowage"`
	ErgGuide            string `yaml:"erg_guide"`
	Symbols             string `yaml:"symbols"`
}

func runDotHazmatReferences(ctx context.Context, r *seed.Run) error {
	file, err := seed.YAML[struct {
		References []hazmatRow `yaml:"references"`
	}](r, hazmatFile)
	if err != nil {
		return err
	}

	refs := make([]*dothazmatreference.DotHazmatReference, 0, len(file.References))
	for i := range file.References {
		row := &file.References[i]
		if row.ID == "" {
			return fmt.Errorf("hazmat reference %s (%s) has no id", row.UnNumber, row.ProperShippingName)
		}
		refs = append(refs, &dothazmatreference.DotHazmatReference{
			ID:                  pulid.ID(row.ID),
			UnNumber:            row.UnNumber,
			ProperShippingName:  row.ProperShippingName,
			HazardClass:         row.HazardClass,
			SubsidiaryHazard:    row.SubsidiaryHazard,
			PackingGroup:        row.PackingGroup,
			SpecialProvisions:   row.SpecialProvisions,
			PackagingExceptions: row.PackagingExceptions,
			PackagingNonBulk:    row.PackagingNonBulk,
			PackagingBulk:       row.PackagingBulk,
			QuantityPassenger:   row.QuantityPassenger,
			QuantityCargo:       row.QuantityCargo,
			VesselStowage:       row.VesselStowage,
			ErgGuide:            row.ErgGuide,
			Symbols:             row.Symbols,
		})
	}

	h := dothazmatreferencedb.DotHazmatReferences
	for start := 0; start < len(refs); start += hazmatUpsertBatch {
		batch := refs[start:min(start+hazmatUpsertBatch, len(refs))]
		if _, err = h.Upsert(ctx, r.DB, batch, keel.OnConflict(h.ID)); err != nil {
			return fmt.Errorf("upsert hazmat references: %w", err)
		}
	}
	r.Count(int64(len(refs)))

	return nil
}

func runIFTAJurisdictions(ctx context.Context, r *seed.Run) error {
	rows, err := seed.YAML[iftaJurisdictionFile](r, iftaFile)
	if err != nil {
		return err
	}

	stateIDs, err := stateIDsByAbbreviation(ctx, r.DB)
	if err != nil {
		return err
	}

	jurisdictions, err := buildIFTAJurisdictions(rows.Jurisdictions, stateIDs)
	if err != nil {
		return err
	}

	j := iftadb.Jurisdictions
	if _, err = j.Upsert(ctx, r.DB, jurisdictions,
		keel.OnConflict(j.CountryCode, j.Code).Update(j.Name, j.SortOrder, j.HasSurcharge, j.UsStateID),
	); err != nil {
		return fmt.Errorf("upsert ifta jurisdictions: %w", err)
	}
	r.Count(int64(len(jurisdictions)))

	return nil
}

func stateIDsByAbbreviation(ctx context.Context, db *keel.DB) (map[string]pulid.ID, error) {
	s := usstatedb.UsStates
	states, err := s.Pick(s.ID, s.Abbreviation).All(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("load us states: %w", err)
	}

	ids := make(map[string]pulid.ID, len(states))
	for _, state := range states {
		ids[state.Abbreviation] = state.ID
	}

	return ids, nil
}

func runJurisdictionRulesBaseline(ctx context.Context, r *seed.Run) error {
	s := usstatedb.UsStates
	j := jurisdictionruledb.JurisdictionRules
	states, err := s.Pick(s.ID).
		Where(keel.NotExists(j.Where(j.StateID.EqCol(s.ID)))).
		All(ctx, r.DB)
	if err != nil {
		return fmt.Errorf("list states without a jurisdiction rule: %w", err)
	}

	rules := make([]*jurisdictionrule.JurisdictionRule, 0, len(states))
	for _, state := range states {
		rules = append(rules, baselineRule(state.ID))
	}

	if err = j.CreateMany(ctx, r.DB, rules); err != nil {
		return fmt.Errorf("create baseline jurisdiction rules: %w", err)
	}
	r.Count(int64(len(rules)))

	return nil
}

func baselineRule(stateID pulid.ID) *jurisdictionrule.JurisdictionRule {
	return &jurisdictionrule.JurisdictionRule{
		StateID:            stateID,
		Status:             jurisdictionrule.StatusActive,
		HolidayRestricted:  true,
		PermitLeadTimeDays: baselinePermitLeadTimeDays,
		PermitValidityDays: baselinePermitValidityDays,
		MaxWidthFeet:       jurisdictionrule.FederalMaxWidthFeet,
		MaxHeightFeet:      jurisdictionrule.FederalMaxHeightFeet,
		MaxLengthFeet:      jurisdictionrule.FederalMaxLengthFeet,
		MaxWeightPounds:    jurisdictionrule.FederalMaxWeightPounds,
		SourceNote:         baselineSourceNote,
		SourceURL:          baselineSourceURL,
		VerificationState:  jurisdictionrule.VerificationUnverified,
	}
}
