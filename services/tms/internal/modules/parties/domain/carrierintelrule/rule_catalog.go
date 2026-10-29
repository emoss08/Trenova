package carrierintelrule

import (
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/fleetsafety"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
)

var (
	catalogOnce sync.Once
	catalog     []RuleDefinition
	catalogMap  map[RuleCode]*RuleDefinition
)

var (
	carrierSubjects = []carrierintel.SubjectType{
		carrierintel.SubjectTypeCarrier,
		carrierintel.SubjectTypeProspect,
	}
	operatingSubjects = []carrierintel.SubjectType{
		carrierintel.SubjectTypeCarrier,
		carrierintel.SubjectTypeProspect,
		carrierintel.SubjectTypeOrganization,
	}
	authoritySubjects = []carrierintel.SubjectType{
		carrierintel.SubjectTypeCarrier,
		carrierintel.SubjectTypeProspect,
		carrierintel.SubjectTypeOrganization,
		carrierintel.SubjectTypeCustomer,
	}
	brokerBondSubjects = []carrierintel.SubjectType{
		carrierintel.SubjectTypeCarrier,
		carrierintel.SubjectTypeProspect,
		carrierintel.SubjectTypeCustomer,
	}
	allNetworkKindNames = []string{
		carrierintel.NetworkKindAddress.String(),
		carrierintel.NetworkKindPhone.String(),
		carrierintel.NetworkKindEmail.String(),
		carrierintel.NetworkKindEIN.String(),
		carrierintel.NetworkKindEquipment.String(),
	}
)

func allBasicNames() []string {
	basics := fleetsafety.AllCSABasics()
	names := make([]string, 0, len(basics))
	for _, b := range basics {
		names = append(names, b.String())
	}
	return names
}

func Catalog() []RuleDefinition {
	catalogOnce.Do(buildCatalog)
	return catalog
}

func RuleByCode(code RuleCode) (*RuleDefinition, bool) {
	catalogOnce.Do(buildCatalog)
	def, ok := catalogMap[code]
	return def, ok
}

func buildCatalog() {
	catalog = []RuleDefinition{
		{
			Code:              RuleUSDOTInactive,
			Label:             "USDOT number inactive",
			Description:       "The USDOT registration is not active with FMCSA.",
			Category:          carrierintel.SectionAuthority,
			DefaultAction:     carrierintel.RuleActionBlock,
			RecommendedAction: carrierintel.RuleActionBlock,
			RequiredSections:  []carrierintel.Section{carrierintel.SectionIdentity},
			Subjects:          authoritySubjects,
			GateRelevant:      true,
			evaluate:          evalUSDOTInactive,
		},
		{
			Code:              RuleAuthorityInactive,
			Label:             "Operating authority inactive",
			Description:       "The operating authority the party needs (common or contract for carriers, broker for brokers) is not active.",
			Category:          carrierintel.SectionAuthority,
			DefaultAction:     carrierintel.RuleActionBlock,
			RecommendedAction: carrierintel.RuleActionBlock,
			RequiredSections:  []carrierintel.Section{carrierintel.SectionAuthority},
			Subjects:          authoritySubjects,
			GateRelevant:      true,
			evaluate:          evalAuthorityInactive,
		},
		{
			Code:              RuleAuthorityRevoked,
			Label:             "Operating authority revoked",
			Description:       "FMCSA has revoked the operating authority.",
			Category:          carrierintel.SectionAuthority,
			DefaultAction:     carrierintel.RuleActionBlock,
			RecommendedAction: carrierintel.RuleActionBlock,
			RequiredSections:  []carrierintel.Section{carrierintel.SectionAuthority},
			Subjects:          authoritySubjects,
			GateRelevant:      true,
			evaluate:          evalAuthorityRevoked,
		},
		{
			Code:              RuleAuthorityPendingRevoke,
			Label:             "Authority revocation pending",
			Description:       "FMCSA has served a pending revocation against the operating authority.",
			Category:          carrierintel.SectionAuthority,
			DefaultAction:     carrierintel.RuleActionBlock,
			RecommendedAction: carrierintel.RuleActionBlock,
			RequiredSections:  []carrierintel.Section{carrierintel.SectionAuthority},
			Subjects:          authoritySubjects,
			GateRelevant:      true,
			evaluate:          evalAuthorityPendingRevocation,
		},
		{
			Code:              RuleInsuranceNoneOnFile,
			Label:             "No liability insurance on file",
			Description:       "No BIPD liability filing is on record with FMCSA.",
			Category:          carrierintel.SectionInsurance,
			DefaultAction:     carrierintel.RuleActionBlock,
			RecommendedAction: carrierintel.RuleActionBlock,
			RequiredSections:  []carrierintel.Section{carrierintel.SectionInsurance},
			Subjects:          operatingSubjects,
			GateRelevant:      true,
			evaluate:          evalInsuranceNoneOnFile,
		},
		{
			Code:              RuleInsuranceBIPDBelow,
			Label:             "Liability coverage below requirement",
			Description:       "BIPD liability on file is below the FMCSA requirement or the organization's minimum.",
			Category:          carrierintel.SectionInsurance,
			DefaultAction:     carrierintel.RuleActionBlock,
			RecommendedAction: carrierintel.RuleActionBlock,
			RequiredSections:  []carrierintel.Section{carrierintel.SectionInsurance},
			Subjects:          operatingSubjects,
			GateRelevant:      true,
			Params: []RuleParamSpec{
				{
					Key:      "minimum",
					Label:    "Organization minimum (USD)",
					Type:     RuleParamTypeDecimal,
					Min:      new(float64(0)),
					HelpText: "Leave blank to use the FMCSA requirement only.",
				},
			},
			evaluate: evalBIPDBelow,
		},
		{
			Code:              RuleInsuranceCargoBelow,
			Label:             "Cargo coverage below requirement",
			Description:       "Cargo insurance on file is below the FMCSA requirement or the organization's minimum.",
			Category:          carrierintel.SectionInsurance,
			DefaultAction:     carrierintel.RuleActionBlock,
			RecommendedAction: carrierintel.RuleActionBlock,
			RequiredSections:  []carrierintel.Section{carrierintel.SectionInsurance},
			Subjects:          carrierSubjects,
			GateRelevant:      true,
			Params: []RuleParamSpec{
				{
					Key:      "minimum",
					Label:    "Organization minimum (USD)",
					Type:     RuleParamTypeDecimal,
					Min:      new(float64(0)),
					HelpText: "Leave blank to use the FMCSA requirement only.",
				},
			},
			evaluate: evalCargoBelow,
		},
		{
			Code:              RuleInsuranceBondBelow,
			Label:             "Broker bond below requirement",
			Description:       "The broker surety bond or trust fund on file is below the requirement.",
			Category:          carrierintel.SectionInsurance,
			DefaultAction:     carrierintel.RuleActionBlock,
			RecommendedAction: carrierintel.RuleActionBlock,
			RequiredSections:  []carrierintel.Section{carrierintel.SectionInsurance},
			Subjects:          brokerBondSubjects,
			GateRelevant:      true,
			Params: []RuleParamSpec{
				{
					Key:     "minimum",
					Label:   "Minimum bond (USD)",
					Type:    RuleParamTypeDecimal,
					Default: "75000",
					Min:     new(float64(0)),
				},
			},
			evaluate: evalBondBelow,
		},
		{
			Code:              RuleInsurancePendingCancel,
			Label:             "Insurance cancellation pending",
			Description:       "An insurer has filed a cancellation that takes effect soon.",
			Category:          carrierintel.SectionInsurance,
			DefaultAction:     carrierintel.RuleActionBlock,
			RecommendedAction: carrierintel.RuleActionBlock,
			RequiredSections:  []carrierintel.Section{carrierintel.SectionInsurance},
			Subjects:          operatingSubjects,
			GateRelevant:      true,
			Params: []RuleParamSpec{
				{
					Key:     "withinDays",
					Label:   "Flag cancellations effective within (days)",
					Type:    RuleParamTypeInteger,
					Default: "30",
					Min:     new(float64(1)),
					Max:     new(float64(365)),
				},
			},
			evaluate: evalPendingCancellation,
		},
		{
			Code:              RuleSafetyOOSOrder,
			Label:             "Out-of-service order",
			Description:       "FMCSA has placed the operation under an out-of-service order.",
			Category:          carrierintel.SectionSafety,
			DefaultAction:     carrierintel.RuleActionBlock,
			RecommendedAction: carrierintel.RuleActionBlock,
			RequiredSections:  []carrierintel.Section{carrierintel.SectionSafety},
			Subjects:          operatingSubjects,
			GateRelevant:      true,
			evaluate:          evalOOSOrder,
		},
		{
			Code:              RuleSafetyUnsatisfactory,
			Label:             "Unsatisfactory safety rating",
			Description:       "The most recent compliance review produced an Unsatisfactory rating.",
			Category:          carrierintel.SectionSafety,
			DefaultAction:     carrierintel.RuleActionWarn,
			RecommendedAction: carrierintel.RuleActionBlock,
			RequiredSections:  []carrierintel.Section{carrierintel.SectionSafety},
			Subjects:          operatingSubjects,
			GateRelevant:      true,
			evaluate:          evalRatingUnsatisfactory,
		},
		{
			Code:              RuleSafetyConditional,
			Label:             "Conditional safety rating",
			Description:       "The most recent compliance review produced a Conditional rating.",
			Category:          carrierintel.SectionSafety,
			DefaultAction:     carrierintel.RuleActionWarn,
			RecommendedAction: carrierintel.RuleActionWarn,
			RequiredSections:  []carrierintel.Section{carrierintel.SectionSafety},
			Subjects:          operatingSubjects,
			GateRelevant:      true,
			evaluate:          evalRatingConditional,
		},
		{
			Code:              RuleBasicsAlert,
			Label:             "BASIC over intervention threshold",
			Description:       "One or more CSA BASICs is in alert or above the configured percentile.",
			Category:          carrierintel.SectionBasics,
			DefaultAction:     carrierintel.RuleActionWarn,
			RecommendedAction: carrierintel.RuleActionWarn,
			RequiredSections:  []carrierintel.Section{carrierintel.SectionBasics},
			Subjects:          operatingSubjects,
			GateRelevant:      true,
			Params: []RuleParamSpec{
				{
					Key:     "basics",
					Label:   "BASICs to check",
					Type:    RuleParamTypeMultiSelect,
					Default: strings.Join(allBasicNames(), ","),
					Options: allBasicNames(),
				},
				{
					Key:      "percentile",
					Label:    "Percentile override",
					Type:     RuleParamTypeNumber,
					Min:      new(float64(1)),
					Max:      new(float64(100)),
					HelpText: "Flag any selected BASIC at or above this percentile even without an FMCSA alert. Leave blank to use FMCSA alerts only.",
				},
			},
			evaluate: evalBasicsAlert,
		},
		{
			Code:              RuleSafetyISSHigh,
			Label:             "High inspection selection score",
			Description:       "The Inspection Selection System score marks the carrier for roadside inspection.",
			Category:          carrierintel.SectionSafety,
			DefaultAction:     carrierintel.RuleActionWarn,
			RecommendedAction: carrierintel.RuleActionWarn,
			RequiredSections:  []carrierintel.Section{carrierintel.SectionSafety},
			Subjects:          operatingSubjects,
			GateRelevant:      true,
			Params: []RuleParamSpec{
				{
					Key:     "minimum",
					Label:   "Minimum ISS score",
					Type:    RuleParamTypeInteger,
					Default: "75",
					Min:     new(float64(1)),
					Max:     new(float64(100)),
				},
			},
			evaluate: evalISSHigh,
		},
		{
			Code:              RuleSafetyRiskScoreHigh,
			Label:             "High provider risk score",
			Description:       "The provider's composite risk model rates the carrier at or above the configured level.",
			Category:          carrierintel.SectionSafety,
			DefaultAction:     carrierintel.RuleActionWarn,
			RecommendedAction: carrierintel.RuleActionWarn,
			RequiredSections:  []carrierintel.Section{carrierintel.SectionSafety},
			Subjects:          carrierSubjects,
			GateRelevant:      true,
			Params: []RuleParamSpec{
				{
					Key:     "minimumLevel",
					Label:   "Minimum risk level",
					Type:    RuleParamTypeSelect,
					Default: carrierintel.RiskLevelHigh.String(),
					Options: []string{
						carrierintel.RiskLevelModerate.String(),
						carrierintel.RiskLevelElevated.String(),
						carrierintel.RiskLevelHigh.String(),
						carrierintel.RiskLevelVeryHigh.String(),
					},
				},
			},
			evaluate: evalRiskScoreHigh,
		},
		{
			Code:              RuleInspectionsOOSHigh,
			Label:             "Out-of-service rate above national average",
			Description:       "Vehicle or driver out-of-service rates exceed the national average by the configured multiple.",
			Category:          carrierintel.SectionInspections,
			DefaultAction:     carrierintel.RuleActionWarn,
			RecommendedAction: carrierintel.RuleActionWarn,
			RequiredSections:  []carrierintel.Section{carrierintel.SectionInspections},
			Subjects:          operatingSubjects,
			GateRelevant:      true,
			Params: []RuleParamSpec{
				{
					Key:     "multiplier",
					Label:   "Multiple of national average",
					Type:    RuleParamTypeNumber,
					Default: "1.5",
					Min:     new(float64(1)),
					Max:     new(float64(10)),
				},
				{
					Key:     "minimumInspections",
					Label:   "Minimum inspections",
					Type:    RuleParamTypeInteger,
					Default: "5",
					Min:     new(float64(1)),
					Max:     new(float64(1000)),
				},
			},
			evaluate: evalOOSRateHigh,
		},
		{
			Code:              RuleIdentityNewEntrant,
			Label:             "New entrant",
			Description:       "The operating authority is younger than the configured age.",
			Category:          carrierintel.SectionAuthority,
			DefaultAction:     carrierintel.RuleActionWarn,
			RecommendedAction: carrierintel.RuleActionWarn,
			RequiredSections:  []carrierintel.Section{carrierintel.SectionAuthority},
			Subjects:          carrierSubjects,
			GateRelevant:      true,
			Params: []RuleParamSpec{
				{
					Key:     "minimumAgeDays",
					Label:   "Minimum authority age (days)",
					Type:    RuleParamTypeInteger,
					Default: "180",
					Min:     new(float64(1)),
					Max:     new(float64(3650)),
				},
			},
			evaluate: evalNewEntrant,
		},
		{
			Code:              RuleOperationsMCS150Stale,
			Label:             "MCS-150 update overdue",
			Description:       "The biennial MCS-150 update has not been filed within the configured window.",
			Category:          carrierintel.SectionOperations,
			DefaultAction:     carrierintel.RuleActionNotify,
			RecommendedAction: carrierintel.RuleActionNotify,
			RequiredSections:  []carrierintel.Section{carrierintel.SectionOperations},
			Subjects:          operatingSubjects,
			Params: []RuleParamSpec{
				{
					Key:     "maximumAgeDays",
					Label:   "Maximum filing age (days)",
					Type:    RuleParamTypeInteger,
					Default: "730",
					Min:     new(float64(30)),
					Max:     new(float64(3650)),
				},
			},
			evaluate: evalMCS150Stale,
		},
		{
			Code:              RuleFraudNetworkSharing,
			Label:             "Shares identity with other carriers",
			Description:       "The carrier shares addresses, phones, emails, EINs or equipment with other USDOT numbers, a common sign of chameleon or double-brokering operations.",
			Category:          carrierintel.SectionNetwork,
			DefaultAction:     carrierintel.RuleActionWarn,
			RecommendedAction: carrierintel.RuleActionWarn,
			RequiredSections:  []carrierintel.Section{carrierintel.SectionNetwork},
			Subjects:          carrierSubjects,
			GateRelevant:      true,
			Params: []RuleParamSpec{
				{
					Key:     "kinds",
					Label:   "Shared identifiers to check",
					Type:    RuleParamTypeMultiSelect,
					Default: strings.Join(allNetworkKindNames, ","),
					Options: allNetworkKindNames,
				},
				{
					Key:     "minimumShared",
					Label:   "Minimum other carriers sharing",
					Type:    RuleParamTypeInteger,
					Default: "1",
					Min:     new(float64(1)),
					Max:     new(float64(1000)),
				},
			},
			evaluate: evalNetworkSharing,
		},
		{
			Code:              RuleFraudContactChurn,
			Label:             "Recent contact changes",
			Description:       "Name, phone, email or address changed repeatedly within the window, a common sign of an account takeover.",
			Category:          carrierintel.SectionChangeHistory,
			DefaultAction:     carrierintel.RuleActionWarn,
			RecommendedAction: carrierintel.RuleActionWarn,
			RequiredSections:  []carrierintel.Section{carrierintel.SectionChangeHistory},
			Subjects:          carrierSubjects,
			GateRelevant:      true,
			Params: []RuleParamSpec{
				{
					Key:     "windowDays",
					Label:   "Window (days)",
					Type:    RuleParamTypeInteger,
					Default: "90",
					Min:     new(float64(1)),
					Max:     new(float64(730)),
				},
				{
					Key:     "minimumChanges",
					Label:   "Minimum changes",
					Type:    RuleParamTypeInteger,
					Default: "2",
					Min:     new(float64(1)),
					Max:     new(float64(100)),
				},
			},
			evaluate: evalContactChurn,
		},
		{
			Code:              RuleBenchmarksAnomaly,
			Label:             "Operating profile anomaly",
			Description:       "Reported mileage, inspections and power units fall outside industry benchmarks.",
			Category:          carrierintel.SectionBenchmarks,
			DefaultAction:     carrierintel.RuleActionNotify,
			RecommendedAction: carrierintel.RuleActionWarn,
			RequiredSections:  []carrierintel.Section{carrierintel.SectionBenchmarks},
			Subjects:          carrierSubjects,
			evaluate:          evalBenchmarksAnomaly,
		},
		{
			Code:              RuleIdentityNotFound,
			Label:             "Not found with provider",
			Description:       "The provider has no record for the USDOT or docket number.",
			Category:          carrierintel.SectionIdentity,
			DefaultAction:     carrierintel.RuleActionWarn,
			RecommendedAction: carrierintel.RuleActionBlock,
			Subjects:          authoritySubjects,
			GateRelevant:      true,
			evaluate:          evalNotFound,
		},
	}

	catalogMap = make(map[RuleCode]*RuleDefinition, len(catalog))
	for idx := range catalog {
		catalogMap[catalog[idx].Code] = &catalog[idx]
	}
}

func relevantGrants(ctx *RuleContext) []*carrierintel.AuthorityGrant {
	auth := ctx.Profile.Authority
	if auth == nil {
		return nil
	}
	if ctx.BrokerAuthority {
		return []*carrierintel.AuthorityGrant{auth.Broker}
	}
	return []*carrierintel.AuthorityGrant{auth.Common, auth.Contract}
}

func authorityLabel(ctx *RuleContext) string {
	if ctx.BrokerAuthority {
		return "Broker authority"
	}
	return "Carrier operating authority"
}

func evalUSDOTInactive(ctx *RuleContext) (bool, string) {
	active, known := ctx.Profile.Identity.USDOTActive()
	if !known || active {
		return false, ""
	}
	return true, fmt.Sprintf("USDOT %s status is %s",
		ctx.Profile.Identity.DOTNumber, ctx.Profile.Identity.USDOTStatus)
}

func evalAuthorityInactive(ctx *RuleContext) (bool, string) {
	if ctx.AuthorityExempt && !ctx.BrokerAuthority {
		return false, ""
	}
	grants := relevantGrants(ctx)
	known := false
	for _, grant := range grants {
		if grant.IsActive() {
			return false, ""
		}
		if grant.IsKnown() {
			known = true
		}
	}
	if !known {
		return false, ""
	}
	return true, authorityLabel(ctx) + " is not active"
}

func evalAuthorityRevoked(ctx *RuleContext) (bool, string) {
	if ctx.AuthorityExempt && !ctx.BrokerAuthority {
		return false, ""
	}
	grants := relevantGrants(ctx)
	revoked := false
	for _, grant := range grants {
		if grant.IsActive() {
			return false, ""
		}
		if grant != nil && grant.Status == carrierintel.AuthorityStatusRevoked {
			revoked = true
		}
	}
	if !revoked {
		return false, ""
	}
	return true, authorityLabel(ctx) + " has been revoked"
}

func evalAuthorityPendingRevocation(ctx *RuleContext) (bool, string) {
	for _, grant := range relevantGrants(ctx) {
		if grant != nil && grant.RevocationPending {
			return true, authorityLabel(ctx) + " has a pending revocation"
		}
	}
	return false, ""
}

func decimalOrZero(v *decimal.Decimal) decimal.Decimal {
	if v == nil {
		return decimal.Zero
	}
	return *v
}

func evalInsuranceNoneOnFile(ctx *RuleContext) (bool, string) {
	if ctx.BrokerAuthority {
		return false, ""
	}
	ins := ctx.Profile.Insurance
	if ins.BIPDOnFile == nil || ins.BIPDOnFile.IsPositive() {
		return false, ""
	}
	if ins.BIPDRequired != nil && !ins.BIPDRequired.IsPositive() {
		return false, ""
	}
	return true, "No BIPD liability insurance is on file with FMCSA"
}

func coverageBelow(
	ctx *RuleContext,
	onFile, required *decimal.Decimal,
	label string,
) (bool, string) {
	if onFile == nil {
		return false, ""
	}
	threshold := decimalOrZero(required)
	if minimum, ok := ctx.paramDecimal("minimum"); ok && minimum.GreaterThan(threshold) {
		threshold = minimum
	}
	if !threshold.IsPositive() || onFile.GreaterThanOrEqual(threshold) {
		return false, ""
	}
	return true, fmt.Sprintf("%s on file (%s) is below the required %s",
		label, money.FormatWholeDollars(*onFile), money.FormatWholeDollars(threshold))
}

func evalBIPDBelow(ctx *RuleContext) (bool, string) {
	if ctx.BrokerAuthority {
		return false, ""
	}
	ins := ctx.Profile.Insurance
	return coverageBelow(ctx, ins.BIPDOnFile, ins.BIPDRequired, "BIPD liability coverage")
}

func evalCargoBelow(ctx *RuleContext) (bool, string) {
	if ctx.BrokerAuthority {
		return false, ""
	}
	ins := ctx.Profile.Insurance
	return coverageBelow(ctx, ins.CargoOnFile, ins.CargoRequired, "Cargo coverage")
}

func evalBondBelow(ctx *RuleContext) (bool, string) {
	if !ctx.BrokerAuthority {
		return false, ""
	}
	ins := ctx.Profile.Insurance
	onFile := ins.BondOnFile
	if onFile == nil {
		zero := decimal.Zero
		onFile = &zero
	}
	return coverageBelow(ctx, onFile, ins.BondRequired, "Broker bond")
}

func evalPendingCancellation(ctx *RuleContext) (bool, string) {
	ins := ctx.Profile.Insurance
	window := int64(ctx.paramInt("withinDays")) * timeutils.SecondsPerDay
	from, until := ctx.Now-timeutils.SecondsPerDay, ctx.Now+window
	candidates := make([]int64, 0, len(ins.Filings)+1)
	if at := ins.PendingCancelAt; at != nil && *at >= from && *at <= until {
		candidates = append(candidates, *at)
	}
	for idx := range ins.Filings {
		if at := ins.Filings[idx].CancelEffectiveAt; at != nil && *at >= from && *at <= until {
			candidates = append(candidates, *at)
		}
	}
	if len(candidates) == 0 {
		return false, ""
	}
	earliest := slices.Min(candidates)
	days := timeutils.WholeDaysBetween(ctx.Now, earliest)
	if days <= 0 {
		return true, "An insurance cancellation takes effect today"
	}
	return true, fmt.Sprintf("An insurance cancellation takes effect in %d days", days)
}

func evalOOSOrder(ctx *RuleContext) (bool, string) {
	safety := ctx.Profile.Safety
	if safety.OutOfServiceOrder == nil || !*safety.OutOfServiceOrder {
		return false, ""
	}
	return true, "An FMCSA out-of-service order is in effect"
}

func evalRatingUnsatisfactory(ctx *RuleContext) (bool, string) {
	if ctx.Profile.Safety.Rating != carrierintel.SafetyRatingUnsatisfactory {
		return false, ""
	}
	return true, "FMCSA safety rating is Unsatisfactory"
}

func evalRatingConditional(ctx *RuleContext) (bool, string) {
	if ctx.Profile.Safety.Rating != carrierintel.SafetyRatingConditional {
		return false, ""
	}
	return true, "FMCSA safety rating is Conditional"
}

func evalBasicsAlert(ctx *RuleContext) (bool, string) {
	selected := ctx.paramList("basics")
	percentileOverride := ctx.paramFloat("percentile")
	flagged := make([]string, 0, len(selected))
	for idx := range ctx.Profile.Basics {
		measure := &ctx.Profile.Basics[idx]
		if len(selected) > 0 && !slices.Contains(selected, measure.Basic.String()) {
			continue
		}
		over := measure.Alert
		if !over && percentileOverride > 0 && measure.Percentile != nil &&
			*measure.Percentile >= percentileOverride {
			over = true
		}
		if over {
			flagged = append(flagged, measure.Basic.String())
		}
	}
	if len(flagged) == 0 {
		return false, ""
	}
	return true, "BASICs over threshold: " + strings.Join(flagged, ", ")
}

func evalISSHigh(ctx *RuleContext) (bool, string) {
	iss := ctx.Profile.Safety.ISSValue
	minimum := ctx.paramInt("minimum")
	if iss == nil || *iss < minimum {
		return false, ""
	}
	return true, fmt.Sprintf("Inspection Selection System score is %d", *iss)
}

func evalRiskScoreHigh(ctx *RuleContext) (bool, string) {
	score := ctx.Profile.Safety.RiskScore
	if !score.IsValid() || score == carrierintel.RiskLevelUnknown {
		return false, ""
	}
	minimum := carrierintel.RiskLevel(ctx.paramString("minimumLevel"))
	if !minimum.IsValid() {
		minimum = carrierintel.RiskLevelHigh
	}
	if score.Rank() < minimum.Rank() {
		return false, ""
	}
	return true, "Provider risk score is " + score.String()
}

func evalOOSRateHigh(ctx *RuleContext) (bool, string) {
	insp := ctx.Profile.Inspections
	multiplier := ctx.paramFloat("multiplier")
	if multiplier <= 0 {
		multiplier = 1.5
	}
	minimum := ctx.paramInt("minimumInspections")

	check := func(count *int, rate, national *float64, label string) string {
		if count == nil || *count < minimum || rate == nil || national == nil || *national <= 0 {
			return ""
		}
		if *rate <= *national*multiplier {
			return ""
		}
		return fmt.Sprintf(
			"%s out-of-service rate %.1f%% vs national %.1f%%",
			label,
			*rate,
			*national,
		)
	}

	messages := make([]string, 0, 2)
	if msg := check(
		insp.Vehicle,
		insp.VehicleOOSRate,
		insp.NationalVehicleOOS,
		"Vehicle",
	); msg != "" {
		messages = append(messages, msg)
	}
	if msg := check(
		insp.Driver,
		insp.DriverOOSRate,
		insp.NationalDriverOOSRate,
		"Driver",
	); msg != "" {
		messages = append(messages, msg)
	}
	if len(messages) == 0 {
		return false, ""
	}
	return true, strings.Join(messages, "; ")
}

func evalNewEntrant(ctx *RuleContext) (bool, string) {
	age := ctx.Profile.Authority.OldestActiveAgeDays()
	if age == nil && ctx.Profile.Identity != nil {
		age = ctx.Profile.Identity.DOTAgeDays
	}
	if age == nil {
		return false, ""
	}
	minimum := ctx.paramInt("minimumAgeDays")
	if *age >= minimum {
		return false, ""
	}
	return true, fmt.Sprintf("Operating authority is %d days old", *age)
}

func evalMCS150Stale(ctx *RuleContext) (bool, string) {
	filed := ctx.Profile.Operations.MCS150At
	if filed == nil {
		return false, ""
	}
	ageDays := timeutils.WholeDaysBetween(*filed, ctx.Now)
	if ageDays <= int64(ctx.paramInt("maximumAgeDays")) {
		return false, ""
	}
	return true, fmt.Sprintf("MCS-150 was last updated %d days ago", ageDays)
}

func evalNetworkSharing(ctx *RuleContext) (bool, string) {
	kinds := ctx.paramList("kinds")
	minimum := ctx.paramInt("minimumShared")
	if minimum < 1 {
		minimum = 1
	}
	flagged := make([]string, 0, len(kinds))
	for _, kind := range kinds {
		count := ctx.Profile.Network.SharedCount(carrierintel.NetworkKind(kind))
		if count >= minimum {
			flagged = append(flagged, fmt.Sprintf("%s (%d)", strings.ToLower(kind), count))
		}
	}
	if len(flagged) == 0 {
		return false, ""
	}
	return true, "Shares identifiers with other USDOT numbers: " + strings.Join(flagged, ", ")
}

func evalContactChurn(ctx *RuleContext) (bool, string) {
	history := ctx.Profile.ChangeHistory
	latest := history.LatestChangeAt()
	if latest == nil {
		return false, ""
	}
	window := int64(ctx.paramInt("windowDays")) * timeutils.SecondsPerDay
	if ctx.Now-*latest > window {
		return false, ""
	}
	total := history.TotalChanges()
	if total < ctx.paramInt("minimumChanges") {
		return false, ""
	}
	return true, fmt.Sprintf("%d identity changes on record, the latest within %d days",
		total, ctx.paramInt("windowDays"))
}

func evalBenchmarksAnomaly(ctx *RuleContext) (bool, string) {
	b := ctx.Profile.Benchmarks
	flagged := make([]string, 0, 3)
	if b.InspectionMileageAnomaly != nil && *b.InspectionMileageAnomaly {
		flagged = append(flagged, "inspections per mile")
	}
	if b.InspectedUnitsAnomaly != nil && *b.InspectedUnitsAnomaly {
		flagged = append(flagged, "inspected power units")
	}
	if b.PowerUnitMileageAnomaly != nil && *b.PowerUnitMileageAnomaly {
		flagged = append(flagged, "miles per power unit")
	}
	if len(flagged) == 0 {
		if b.AnyAnomaly != nil && *b.AnyAnomaly {
			return true, "Operating profile falls outside industry benchmarks"
		}
		return false, ""
	}
	return true, "Outside industry benchmarks: " + strings.Join(flagged, ", ")
}

func evalNotFound(ctx *RuleContext) (bool, string) {
	if !ctx.NotFound {
		return false, ""
	}
	return true, "The provider has no record for this USDOT or docket number"
}
