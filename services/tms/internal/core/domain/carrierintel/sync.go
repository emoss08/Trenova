package carrierintel

import (
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/shopspring/decimal"
)

type SyncField string

const (
	SyncFieldSafetyRating = SyncField("safetyRating")
	SyncFieldMCNumber     = SyncField("mcNumber")
	SyncFieldName         = SyncField("name")
	SyncFieldDBAName      = SyncField("dbaName")
	SyncFieldAddressLine1 = SyncField("addressLine1")
	SyncFieldCity         = SyncField("city")
	SyncFieldPostalCode   = SyncField("postalCode")
	SyncFieldPhone        = SyncField("phone")
	SyncFieldEmail        = SyncField("email")
)

func (f SyncField) String() string { return string(f) }

func (f SyncField) IsValid() bool {
	return slices.Contains(AllSyncFields(), f)
}

func AllSyncFields() []SyncField {
	return []SyncField{
		SyncFieldSafetyRating,
		SyncFieldMCNumber,
		SyncFieldName,
		SyncFieldDBAName,
		SyncFieldAddressLine1,
		SyncFieldCity,
		SyncFieldPostalCode,
		SyncFieldPhone,
		SyncFieldEmail,
	}
}

func FillableSyncFields() []SyncField {
	return []SyncField{
		SyncFieldDBAName,
		SyncFieldAddressLine1,
		SyncFieldCity,
		SyncFieldPostalCode,
		SyncFieldPhone,
		SyncFieldEmail,
	}
}

type SyncSettings struct {
	AutoApplySafetyRating bool
	AutoFillFields        []SyncField
}

type FieldUpdate struct {
	Field    SyncField `json:"field"`
	Current  string    `json:"current"`
	Proposed string    `json:"proposed"`
	Reason   string    `json:"reason"`
}

type InsuranceChangeKind string

const (
	InsuranceChangeShortenExpiration = InsuranceChangeKind("ShortenExpiration")
	InsuranceChangeCoverageChanged   = InsuranceChangeKind("CoverageChanged")
	InsuranceChangeNewFiling         = InsuranceChangeKind("NewFiling")
)

type InsuranceChange struct {
	Kind                   InsuranceChangeKind         `json:"kind"`
	PolicyID               pulid.ID                    `json:"policyId,omitempty"`
	PolicyType             carrier.InsurancePolicyType `json:"policyType"`
	PolicyNumber           string                      `json:"policyNumber"`
	ProviderName           string                      `json:"providerName"`
	CurrentCoverage        *decimal.Decimal            `json:"currentCoverage,omitempty"`
	ProposedCoverage       *decimal.Decimal            `json:"proposedCoverage,omitempty"`
	CurrentExpirationDate  *int64                      `json:"currentExpirationDate,omitempty"`
	ProposedExpirationDate *int64                      `json:"proposedExpirationDate,omitempty"`
	EffectiveDate          *int64                      `json:"effectiveDate,omitempty"`
	Reason                 string                      `json:"reason"`
}

type SyncPlan struct {
	AutoApply            []FieldUpdate     `json:"autoApply"`
	Suggestions          []FieldUpdate     `json:"suggestions"`
	InsuranceAutoApply   []InsuranceChange `json:"insuranceAutoApply"`
	InsuranceSuggestions []InsuranceChange `json:"insuranceSuggestions"`
}

func (p *SyncPlan) HasAutoApply() bool {
	return len(p.AutoApply) > 0 || len(p.InsuranceAutoApply) > 0
}

func (p *SyncPlan) Suggestion(field SyncField) (FieldUpdate, bool) {
	for _, s := range p.Suggestions {
		if s.Field == field {
			return s, true
		}
	}
	return FieldUpdate{}, false
}

func mapSafetyRating(rating SafetyRating) carrier.SafetyRating {
	switch rating {
	case SafetyRatingSatisfactory:
		return carrier.SafetyRatingSatisfactory
	case SafetyRatingConditional:
		return carrier.SafetyRatingConditional
	case SafetyRatingUnsatisfactory:
		return carrier.SafetyRatingUnsatisfactory
	default:
		return carrier.SafetyRatingNotRated
	}
}

func mapFilingType(t InsuranceFilingType) (carrier.InsurancePolicyType, bool) {
	switch t {
	case InsuranceFilingTypeBIPD:
		return carrier.InsurancePolicyTypeAutoLiability, true
	case InsuranceFilingTypeCargo:
		return carrier.InsurancePolicyTypeCargoLiability, true
	default:
		return "", false
	}
}

func PlanCarrierSync(c *carrier.Carrier, p *Profile, settings SyncSettings, now int64) SyncPlan {
	plan := SyncPlan{
		AutoApply:            make([]FieldUpdate, 0, 2),
		Suggestions:          make([]FieldUpdate, 0, 4),
		InsuranceAutoApply:   make([]InsuranceChange, 0),
		InsuranceSuggestions: make([]InsuranceChange, 0),
	}
	if c == nil || p == nil {
		return plan
	}

	if p.Safety != nil && p.Safety.Rating.IsValid() {
		proposed := mapSafetyRating(p.Safety.Rating)
		if proposed != c.SafetyRating {
			update := FieldUpdate{
				Field:    SyncFieldSafetyRating,
				Current:  c.SafetyRating.String(),
				Proposed: proposed.String(),
				Reason:   "FMCSA safety rating differs from the carrier record",
			}
			if settings.AutoApplySafetyRating {
				plan.AutoApply = append(plan.AutoApply, update)
			} else {
				plan.Suggestions = append(plan.Suggestions, update)
			}
		}
	}

	if p.Identity != nil {
		planIdentity(&plan, c, p.Identity, settings)
	}
	if p.Contacts != nil {
		planScalar(
			&plan,
			settings,
			SyncFieldPhone,
			c.Phone,
			stringutils.DigitsOnly(p.Contacts.Phone),
			"Phone on file with FMCSA",
			stringutils.DigitsOnly,
		)
		planScalar(&plan, settings, SyncFieldEmail, c.Email, strings.ToLower(p.Contacts.Email),
			"Email on file with FMCSA", strings.ToLower)
	}
	if p.Insurance != nil {
		planInsurance(&plan, c, p.Insurance, now)
	}

	return plan
}

func planIdentity(plan *SyncPlan, c *carrier.Carrier, identity *Identity, settings SyncSettings) {
	docket := stringutils.DigitsOnly(identity.DocketNumber)
	prefix := strings.ToUpper(identity.DocketPrefix)
	if docket != "" && (prefix == "" || prefix == "MC") {
		switch {
		case c.MCNumber == "":
			plan.AutoApply = append(plan.AutoApply, FieldUpdate{
				Field:    SyncFieldMCNumber,
				Proposed: docket,
				Reason:   "MC number filled from FMCSA registration",
			})
		case stringutils.DigitsOnly(c.MCNumber) != docket:
			plan.Suggestions = append(plan.Suggestions, FieldUpdate{
				Field:    SyncFieldMCNumber,
				Current:  c.MCNumber,
				Proposed: docket,
				Reason:   "MC number differs from FMCSA registration",
			})
		}
	}

	if legal := strings.TrimSpace(identity.LegalName); legal != "" &&
		stringutils.CollapseUpper(legal) != stringutils.CollapseUpper(c.Name) {
		plan.Suggestions = append(plan.Suggestions, FieldUpdate{
			Field:    SyncFieldName,
			Current:  c.Name,
			Proposed: stringutils.TruncateRunes(legal, 255),
			Reason:   "Legal name differs from FMCSA registration",
		})
	}

	planScalar(plan, settings, SyncFieldDBAName, c.DBAName,
		stringutils.TruncateRunes(strings.TrimSpace(identity.DBAName), 255),
		"DBA name on file with FMCSA", stringutils.CollapseUpper)

	if addr := identity.PhysicalAddress; !addr.IsZero() {
		planScalar(plan, settings, SyncFieldAddressLine1, c.AddressLine1,
			stringutils.TruncateRunes(strings.TrimSpace(addr.Line1), 150),
			"Physical address on file with FMCSA", stringutils.CollapseUpper)
		planScalar(plan, settings, SyncFieldCity, c.City,
			stringutils.TruncateRunes(strings.TrimSpace(addr.City), 100),
			"Physical address on file with FMCSA", stringutils.CollapseUpper)
		planScalar(plan, settings, SyncFieldPostalCode, c.PostalCode,
			stringutils.TruncateRunes(strings.TrimSpace(addr.PostalCode), 10),
			"Physical address on file with FMCSA", stringutils.CollapseUpper)
	}
}

func planScalar(
	plan *SyncPlan,
	settings SyncSettings,
	field SyncField,
	current, proposed, reason string,
	normalize func(string) string,
) {
	if proposed == "" {
		return
	}
	if normalize(current) == normalize(proposed) {
		return
	}
	update := FieldUpdate{Field: field, Current: current, Proposed: proposed, Reason: reason}
	if current == "" && slices.Contains(settings.AutoFillFields, field) {
		plan.AutoApply = append(plan.AutoApply, update)
		return
	}
	plan.Suggestions = append(plan.Suggestions, update)
}

func planInsurance(plan *SyncPlan, c *carrier.Carrier, ins *Insurance, now int64) {
	for idx := range ins.Filings {
		filing := &ins.Filings[idx]
		policyType, ok := mapFilingType(filing.Type)
		if !ok || strings.TrimSpace(filing.PolicyNumber) == "" {
			continue
		}

		existing := findPolicy(c, policyType, filing.PolicyNumber)
		if existing != nil {
			if filing.CancelEffectiveAt != nil &&
				*filing.CancelEffectiveAt < existing.ExpirationDate {
				proposed := max(*filing.CancelEffectiveAt, existing.EffectiveDate+1)
				currentExpiry := existing.ExpirationDate
				plan.InsuranceAutoApply = append(plan.InsuranceAutoApply, InsuranceChange{
					Kind:                   InsuranceChangeShortenExpiration,
					PolicyID:               existing.ID,
					PolicyType:             policyType,
					PolicyNumber:           existing.PolicyNumber,
					ProviderName:           existing.ProviderName,
					CurrentExpirationDate:  &currentExpiry,
					ProposedExpirationDate: &proposed,
					Reason:                 "Insurer filed a cancellation with FMCSA",
				})
				continue
			}
			if filing.Coverage != nil && !filing.Coverage.Equal(existing.CoverageAmount) {
				currentCoverage := existing.CoverageAmount
				plan.InsuranceSuggestions = append(plan.InsuranceSuggestions, InsuranceChange{
					Kind:             InsuranceChangeCoverageChanged,
					PolicyID:         existing.ID,
					PolicyType:       policyType,
					PolicyNumber:     existing.PolicyNumber,
					ProviderName:     existing.ProviderName,
					CurrentCoverage:  &currentCoverage,
					ProposedCoverage: filing.Coverage,
					Reason:           "Coverage on the FMCSA filing differs from the policy record",
				})
			}
			continue
		}

		if filing.CancelEffectiveAt != nil && *filing.CancelEffectiveAt <= now {
			continue
		}
		plan.InsuranceSuggestions = append(plan.InsuranceSuggestions, InsuranceChange{
			Kind:       InsuranceChangeNewFiling,
			PolicyType: policyType,
			PolicyNumber: stringutils.TruncateRunes(
				strings.TrimSpace(filing.PolicyNumber),
				100,
			),
			ProviderName:     stringutils.TruncateRunes(strings.TrimSpace(filing.InsurerName), 255),
			ProposedCoverage: filing.Coverage,
			EffectiveDate:    filing.EffectiveAt,
			Reason:           "Active FMCSA filing is not recorded as a policy",
		})
	}
}

func findPolicy(
	c *carrier.Carrier,
	policyType carrier.InsurancePolicyType,
	policyNumber string,
) *carrier.CarrierInsurancePolicy {
	target := stringutils.CollapseUpper(policyNumber)
	for _, policy := range c.InsurancePolicies {
		if policy.PolicyType == policyType &&
			stringutils.CollapseUpper(policy.PolicyNumber) == target {
			return policy
		}
	}
	return nil
}

func ApplyFieldUpdates(c *carrier.Carrier, updates []FieldUpdate) []SyncField {
	applied := make([]SyncField, 0, len(updates))
	for _, update := range updates {
		switch update.Field {
		case SyncFieldSafetyRating:
			rating := carrier.SafetyRating(update.Proposed)
			if !rating.IsValid() {
				continue
			}
			c.SafetyRating = rating
		case SyncFieldMCNumber:
			c.MCNumber = update.Proposed
		case SyncFieldName:
			c.Name = update.Proposed
		case SyncFieldDBAName:
			c.DBAName = update.Proposed
		case SyncFieldAddressLine1:
			c.AddressLine1 = update.Proposed
		case SyncFieldCity:
			c.City = update.Proposed
		case SyncFieldPostalCode:
			c.PostalCode = update.Proposed
		case SyncFieldPhone:
			c.Phone = update.Proposed
		case SyncFieldEmail:
			c.Email = update.Proposed
		default:
			continue
		}
		applied = append(applied, update.Field)
	}
	return applied
}

func ApplyInsuranceChanges(c *carrier.Carrier, changes []InsuranceChange) int {
	applied := 0
	for _, change := range changes {
		if change.PolicyID.IsNil() {
			continue
		}
		for _, policy := range c.InsurancePolicies {
			if policy.ID != change.PolicyID {
				continue
			}
			switch change.Kind {
			case InsuranceChangeShortenExpiration:
				if change.ProposedExpirationDate != nil &&
					*change.ProposedExpirationDate < policy.ExpirationDate {
					policy.ExpirationDate = *change.ProposedExpirationDate
					policy.IsVerified = false
					applied++
				}
			case InsuranceChangeCoverageChanged:
				if change.ProposedCoverage != nil {
					policy.CoverageAmount = *change.ProposedCoverage
					policy.IsVerified = false
					applied++
				}
			}
		}
	}
	return applied
}
