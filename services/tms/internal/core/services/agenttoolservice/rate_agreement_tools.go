package agenttoolservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/domain/rategeo"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolschema"
	"github.com/emoss08/trenova/internal/core/services/rateagreementservice"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const (
	paramRateAgreementID      = "rateAgreementId"
	paramPartyType            = "partyType"
	paramAgreementCode        = "code"
	paramAgreementName        = "name"
	paramDescription          = "description"
	paramAgreementType        = "agreementType"
	paramContractRef          = "contractRef"
	paramPriority             = "priority"
	paramAutoRenew            = "autoRenew"
	paramRenewalNoticeDays    = "renewalNoticeDays"
	paramAgreementCurrency    = "currency"
	paramDefaultMinCharge     = "defaultMinCharge"
	paramDefaultMaxCharge     = "defaultMaxCharge"
	paramLanes                = "lanes"
	paramRuleID               = "ruleId"
	paramLaneLabel            = "label"
	paramOriginScopeType      = "originScopeType"
	paramOriginValue          = "originValue"
	paramOriginCity           = "originCity"
	paramDestinationType      = "destinationScopeType"
	paramDestinationValue     = "destinationValue"
	paramDestinationCity      = "destinationCity"
	paramDirection            = "direction"
	paramFormulaTemplateID    = "formulaTemplateId"
	paramRateMatrixID         = "rateMatrixId"
	paramLaneRate             = "rate"
	paramLaneMinCharge        = "minCharge"
	paramLaneMaxCharge        = "maxCharge"
	paramHazmatOnly           = "hazmatOnly"
	paramTempControlOnly      = "tempControlOnly"
	paramSupersededRuleIDs    = "supersededRuleIds"
	paramReviewComment        = "comment"
	paramAgreementIDs         = "agreementIds"
	paramPercentChange        = "percentChange"
	paramFlatChange           = "flatChange"
	paramRateImportID         = "rateImportId"
	paramSimulationName       = "name"
	paramSampleFrom           = "sampleFrom"
	paramSampleTo             = "sampleTo"
	paramSampleLimit          = "sampleLimit"
	rateAgreementRecordEntity = "rate_agreement"
	rateAgreementKind         = "rate agreement"

	maxAgreementCode      = 50
	maxAgreementName      = 150
	maxAgreementText      = 2000
	maxContractRef        = 100
	maxAgreementPriority  = 1000
	maxRenewalNotice      = 365
	maxAgreementLanes     = 200
	maxLaneLabel          = 150
	maxLaneValue          = 120
	maxLaneCity           = 100
	maxReviewComment      = 1000
	maxIncreaseScope      = 50
	maxSampleShipments    = 50000
	maxPreviewLanes       = 5
	rateAgreementSupplier = "The rate agreement, from list_rate_agreements or " +
		"get_rate_agreement. Never guess one."
	rateImportSupplier = "The rate import, from list_rate_imports. Never guess one."
	rateDraftTaintHold = "An agreement drafted from a rate sheet or a message someone " +
		"outside sent is proposed, since its lanes are what the organization will charge."
)

var (
	partyTypes     = agenttoolschema.Source("rateAgreement.partyType", rateagreement.PartyTypeValues())
	agreementTypes = agenttoolschema.Source(
		"rateAgreement.agreementType",
		rateagreement.AgreementTypeValues(),
	)
	laneDirections = agenttoolschema.Source(
		"rateAgreement.direction",
		rateagreement.DirectionValues(),
	)
	laneScopeTypes = agenttoolschema.Source("rateAgreement.laneScopeType", []rategeo.ScopeType{
		rategeo.ScopeTypeAny,
		rategeo.ScopeTypeCountry,
		rategeo.ScopeTypeState,
		rategeo.ScopeTypeZone,
		rategeo.ScopeTypeCityState,
		rategeo.ScopeTypeZip3,
		rategeo.ScopeTypeZip5,
		rategeo.ScopeTypeLocation,
	})
	rateAgreementDates = map[string]assistantartifact.DisplayType{
		paramEffectiveFrom: assistantartifact.DisplayDate,
		paramEffectiveTo:   assistantartifact.DisplayDate,
	}
)

type rateAgreementKeeper interface {
	GetByID(
		ctx context.Context,
		req *repositories.GetRateAgreementByIDRequest,
	) (*rateagreement.RateAgreement, error)
	PlanCreate(
		ctx context.Context,
		entity *rateagreement.RateAgreement,
	) (*rateagreement.RateAgreement, error)
	Create(
		ctx context.Context,
		entity *rateagreement.RateAgreement,
		userID pulid.ID,
	) (*rateagreement.RateAgreement, error)
	PlanUpdate(
		ctx context.Context,
		entity *rateagreement.RateAgreement,
	) (*rateagreementservice.UpdatePlan, error)
	Update(
		ctx context.Context,
		entity *rateagreement.RateAgreement,
		userID pulid.ID,
	) (*rateagreement.RateAgreement, error)
	PlanDuplicate(
		ctx context.Context,
		req *rateagreementservice.DuplicateRateAgreementRequest,
	) (*rateagreementservice.DuplicatePlan, error)
	Duplicate(
		ctx context.Context,
		req *rateagreementservice.DuplicateRateAgreementRequest,
		userID pulid.ID,
	) (*rateagreement.RateAgreement, error)
	PlanReview(
		ctx context.Context,
		review rateagreementservice.Review,
		req *rateagreementservice.ApprovalActionRequest,
	) (*rateagreementservice.AgreementChange, error)
	Review(
		ctx context.Context,
		review rateagreementservice.Review,
		req *rateagreementservice.ApprovalActionRequest,
	) (*rateagreement.RateAgreement, error)
	PlanAmendRules(
		ctx context.Context,
		req *repositories.AmendRateAgreementRulesRequest,
	) (*rateagreementservice.AmendmentPlan, error)
	AmendRules(
		ctx context.Context,
		req *repositories.AmendRateAgreementRulesRequest,
		userID pulid.ID,
	) (*rateagreement.RateAgreement, error)
	PlanApplyRateIncrease(
		ctx context.Context,
		req *rateagreementservice.RateIncreaseRequest,
	) (*rateagreementservice.RateIncreasePlan, error)
	ApplyRateIncrease(
		ctx context.Context,
		req *rateagreementservice.RateIncreaseRequest,
		userID pulid.ID,
	) (*rateagreementservice.RateIncreasePlan, error)
}

type stateFinder interface {
	GetByAbbreviation(ctx context.Context, abbreviation string) (*usstate.UsState, error)
}

var _ rateAgreementKeeper = (*rateagreementservice.Service)(nil)

func targetRateAgreement(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, paramRateAgreementID, permission.ResourceRateAgreement)
}

func rateAgreementIDProperty() map[string]any {
	return stringProperty(rateAgreementSupplier, 0)
}

func laneProperty(forRevision bool) map[string]any {
	properties := map[string]any{
		paramLaneLabel: stringProperty("A short name for the lane, such as Dallas to Atlanta.",
			maxLaneLabel),
		paramOriginScopeType: agenttoolschema.Enum("How the origin is named. Defaults to "+
			"Any.", laneScopeTypes),
		paramOriginValue: stringProperty("The origin for its scope type: a three-letter "+
			"country code such as USA, a two-letter state code such as TX (for State and "+
			"CityState), a postal code (Zip3, Zip5), a location from list_locations, or a "+
			"rate zone a lane of get_rate_agreement names. Leave it out for Any.", maxLaneValue),
		paramOriginCity: stringProperty("The origin city, for a CityState origin.",
			maxLaneCity),
		paramDestinationType: agenttoolschema.Enum("How the destination is named. "+
			"Defaults to Any.", laneScopeTypes),
		paramDestinationValue: stringProperty("The destination, read as originValue is.",
			maxLaneValue),
		paramDestinationCity: stringProperty("The destination city, for a CityState "+
			"destination.", maxLaneCity),
		paramDirection: agenttoolschema.Enum("Whether the lane also prices the reverse "+
			"haul. Defaults to Directional.", laneDirections),
		paramFormulaTemplateID: stringProperty("The formula template that prices the lane, "+
			"from list_formula_templates, with the lane's rate as its base rate. Give this or "+
			"rateMatrixId.", 0),
		paramRateMatrixID: stringProperty("The rate matrix that prices the lane, as a lane "+
			"of get_rate_agreement names it. Give this or formulaTemplateId.", 0),
		paramLaneRate: stringProperty("The lane's rate as a decimal such as 2.35, in what "+
			"its formula template charges by.", 0),
		paramLaneMinCharge: stringProperty("The least the lane charges, as a decimal.", 0),
		paramLaneMaxCharge: stringProperty("The most the lane charges, as a decimal.", 0),
		paramPriority: integerProperty("Which lane wins when two match equally; higher "+
			"wins.", 0, maxAgreementPriority),
		paramHazmatOnly:      booleanProperty("Only hazardous freight rates on this lane."),
		paramTempControlOnly: booleanProperty("Only temperature-controlled freight rates on this lane."),
	}
	description := "The lanes, each priced through a formula template or a rate matrix."
	if forRevision {
		properties[paramRuleID] = stringProperty("A lane already on the agreement, from "+
			"get_rate_agreement, to keep and change; leave it out for a new lane.", 0)
		description = "Every lane of the revised agreement, replacing its lanes: a lane " +
			"with ruleId keeps that lane and changes only what it gives, a lane without one " +
			"is new, and a lane left out is dropped. Leave it out to keep the lanes."
	}

	return map[string]any{
		toolschema.KeyType:        toolschema.TypeArray,
		toolschema.KeyDescription: description,
		toolschema.KeyMaxItems:    maxAgreementLanes,
		toolschema.KeyItems: map[string]any{
			toolschema.KeyType:                 toolschema.TypeObject,
			toolschema.KeyProperties:           properties,
			toolschema.KeyAdditionalProperties: false,
		},
	}
}

type laneReader struct {
	states stateFinder
}

func (r laneReader) scopeValue(
	ctx context.Context,
	scope rategeo.ScopeType,
	raw string,
) (string, error) {
	value := strings.TrimSpace(raw)
	switch scope {
	case rategeo.ScopeTypeAny:
		return "", nil
	case rategeo.ScopeTypeState, rategeo.ScopeTypeCityState:
		state, err := r.states.GetByAbbreviation(ctx, strings.ToUpper(value))
		if err != nil || state == nil {
			return "", fmt.Errorf("%q is not a state or province code", raw)
		}

		return state.ID.String(), nil
	case rategeo.ScopeTypeCountry:
		return strings.ToUpper(value), nil
	case rategeo.ScopeTypeZone,
		rategeo.ScopeTypeRadius,
		rategeo.ScopeTypeZip3,
		rategeo.ScopeTypeZip5,
		rategeo.ScopeTypeLocation:
		return value, nil
	default:
		return value, nil
	}
}

func (r laneReader) end(
	ctx context.Context,
	fields map[string]any,
	keys [3]string,
	current rategeo.Scope,
) (rategeo.Scope, error) {
	scope := current
	if _, given := fields[keys[0]]; given {
		kind, err := requireEnum(fields, keys[0], laneScopeTypes.Values)
		if err != nil {
			return scope, err
		}
		scope.Type = kind
		scope.Value = ""
		scope.City = ""
	}
	if scope.Type == "" {
		scope.Type = rategeo.ScopeTypeAny
	}
	if _, given := fields[keys[1]]; given {
		value, err := r.scopeValue(ctx, scope.Type, optionalString(fields, keys[1]))
		if err != nil {
			return scope, fmt.Errorf("%s: %w", keys[1], err)
		}
		scope.Value = value
	}
	if _, given := fields[keys[2]]; given {
		scope.City = strings.TrimSpace(optionalString(fields, keys[2]))
	}
	if scope.Type != rategeo.ScopeTypeAny && scope.Value == "" {
		return scope, fmt.Errorf("%s needs %s for a %s scope", keys[0], keys[1], scope.Type)
	}
	if scope.Type == rategeo.ScopeTypeCityState && scope.City == "" {
		return scope, fmt.Errorf("a CityState scope needs %s", keys[2])
	}

	return scope, nil
}

func optionalLaneMoney(fields map[string]any, key string) (decimal.NullDecimal, bool, error) {
	if _, given := fields[key]; !given {
		return decimal.NullDecimal{}, false, nil
	}
	value, present, err := optionalDecimal(fields, key)
	if err != nil {
		return decimal.NullDecimal{}, true, err
	}
	if present && value.IsNegative() {
		return decimal.NullDecimal{}, true, fmt.Errorf("%s cannot be negative", key)
	}

	return decimal.NullDecimal{Decimal: value, Valid: present}, true, nil
}

func (r laneReader) applyPricing(rule *rateagreement.RateAgreementRule, fields map[string]any) error {
	for key, target := range map[string]**pulid.ID{
		paramFormulaTemplateID: &rule.FormulaTemplateID,
		paramRateMatrixID:      &rule.RateMatrixID,
	} {
		id, err := optionalPulidParam(fields, key)
		if err != nil {
			return err
		}
		if id != nil {
			*target = id
		}
	}
	for key, target := range map[string]*decimal.NullDecimal{
		paramLaneRate:      &rule.Rate,
		paramLaneMinCharge: &rule.MinCharge,
		paramLaneMaxCharge: &rule.MaxCharge,
	} {
		value, given, err := optionalLaneMoney(fields, key)
		if err != nil {
			return err
		}
		if given {
			*target = value
		}
	}

	return nil
}

func (r laneReader) applyTerms(rule *rateagreement.RateAgreementRule, fields map[string]any) error {
	label, err := optionalBoundedText(fields, paramLaneLabel, maxLaneLabel)
	if err != nil {
		return err
	}
	if label != nil {
		rule.Label = *label
	}
	if _, given := fields[paramDirection]; given {
		if rule.Direction, err = requireEnum(fields, paramDirection, laneDirections.Values); err != nil {
			return err
		}
	}
	if _, given := fields[paramPriority]; given {
		priority, rangeErr := requireIntInRange(fields, paramPriority, 0, maxAgreementPriority)
		if rangeErr != nil {
			return rangeErr
		}
		rule.Priority = int16(priority) //nolint:gosec // bounded to maxAgreementPriority above
	}
	for key, target := range map[string]*bool{
		paramHazmatOnly:      &rule.HazmatOnly,
		paramTempControlOnly: &rule.TempControlOnly,
	} {
		flag, flagErr := optionalBoolPointer(fields, key)
		if flagErr != nil {
			return flagErr
		}
		if flag != nil {
			*target = *flag
		}
	}

	return nil
}

func (r laneReader) lane(
	ctx context.Context,
	fields map[string]any,
	base *rateagreement.RateAgreementRule,
) (*rateagreement.RateAgreementRule, error) {
	rule := &rateagreement.RateAgreementRule{Status: rateagreement.RuleStatusActive}
	if base != nil {
		copied := *base
		rule = &copied
	}

	origin, err := r.end(ctx, fields,
		[3]string{paramOriginScopeType, paramOriginValue, paramOriginCity},
		rule.OriginScope())
	if err != nil {
		return nil, err
	}
	destination, err := r.end(ctx, fields,
		[3]string{paramDestinationType, paramDestinationValue, paramDestinationCity},
		rule.DestinationScope())
	if err != nil {
		return nil, err
	}
	rule.OriginScopeType, rule.OriginScopeValue, rule.OriginCity =
		origin.Type, origin.Value, origin.City
	rule.DestinationScopeType, rule.DestinationScopeValue, rule.DestinationCity =
		destination.Type, destination.Value, destination.City

	if err = r.applyPricing(rule, fields); err != nil {
		return nil, err
	}
	if err = r.applyTerms(rule, fields); err != nil {
		return nil, err
	}
	rule.ApplyLaneKey()

	return rule, nil
}

func laneObjects(params map[string]any) ([]map[string]any, error) {
	raw, ok := params[paramLanes].([]any)
	if !ok {
		return nil, fmt.Errorf("parameter %q must be a list of lanes", paramLanes)
	}
	if len(raw) > maxAgreementLanes {
		return nil, fmt.Errorf("parameter %q holds %d lanes; an agreement takes at most %d "+
			"in one call", paramLanes, len(raw), maxAgreementLanes)
	}

	lanes := make([]map[string]any, 0, len(raw))
	for idx, item := range raw {
		fields, isObject := item.(map[string]any)
		if !isObject {
			return nil, fmt.Errorf("%s[%d] must be an object", paramLanes, idx)
		}
		lanes = append(lanes, fields)
	}

	return lanes, nil
}

func (r laneReader) newLanes(
	ctx context.Context,
	params map[string]any,
) ([]*rateagreement.RateAgreementRule, error) {
	if _, given := params[paramLanes]; !given {
		return nil, nil
	}
	objects, err := laneObjects(params)
	if err != nil {
		return nil, err
	}

	rules := make([]*rateagreement.RateAgreementRule, 0, len(objects))
	for idx, fields := range objects {
		rule, laneErr := r.lane(ctx, fields, nil)
		if laneErr != nil {
			return nil, fmt.Errorf("%s[%d]: %w", paramLanes, idx, laneErr)
		}
		rules = append(rules, rule)
	}

	return rules, nil
}

func (r laneReader) revisedLanes(
	ctx context.Context,
	params map[string]any,
	current []*rateagreement.RateAgreementRule,
) ([]*rateagreement.RateAgreementRule, error) {
	objects, err := laneObjects(params)
	if err != nil {
		return nil, err
	}
	byID := make(map[pulid.ID]*rateagreement.RateAgreementRule, len(current))
	for _, rule := range current {
		if rule != nil {
			byID[rule.ID] = rule
		}
	}

	rules := make([]*rateagreement.RateAgreementRule, 0, len(objects))
	for idx, fields := range objects {
		var base *rateagreement.RateAgreementRule
		ruleID, idErr := optionalPulidParam(fields, paramRuleID)
		if idErr != nil {
			return nil, fmt.Errorf("%s[%d]: %w", paramLanes, idx, idErr)
		}
		if ruleID != nil {
			known, found := byID[*ruleID]
			if !found {
				return nil, fmt.Errorf("%s[%d]: %s is not a lane of this agreement; read "+
					"its lanes with get_rate_agreement", paramLanes, idx, ruleID)
			}
			base = known
		}
		rule, laneErr := r.lane(ctx, fields, base)
		if laneErr != nil {
			return nil, fmt.Errorf("%s[%d]: %w", paramLanes, idx, laneErr)
		}
		rules = append(rules, rule)
	}

	return rules, nil
}

func agreementHeaderProperties() map[string]any {
	return map[string]any{
		paramAgreementCode: stringProperty("A short code for the agreement, unique in the "+
			"organization, such as ACME-2027.", maxAgreementCode),
		paramAgreementName: stringProperty("The agreement's name.", maxAgreementName),
		paramDescription:   stringProperty("What the agreement covers.", maxAgreementText),
		paramAgreementType: agenttoolschema.Enum("The commercial arrangement. Defaults to "+
			"Contract.", agreementTypes),
		paramContractRef: stringProperty("The signed contract's own reference.",
			maxContractRef),
		paramPriority: integerProperty("Which agreement wins when two price a shipment; "+
			"higher wins.", 0, maxAgreementPriority),
		paramEffectiveFrom: dateProperty("The first day the agreement prices shipments."),
		paramEffectiveTo:   dateProperty("The last day it prices shipments. Leave it out for an open-ended agreement."),
		paramAutoRenew:     booleanProperty("Whether it renews itself at its end."),
		paramRenewalNoticeDays: integerProperty("How many days before its end to warn about "+
			"renewal.", 0, maxRenewalNotice),
		paramAgreementCurrency: stringProperty("The ISO currency code it prices in. Defaults "+
			"to USD.", currencyCodeLength),
		paramDefaultMinCharge: stringProperty("The least any lane charges, as a decimal.", 0),
		paramDefaultMaxCharge: stringProperty("The most any lane charges, as a decimal.", 0),
	}
}

type agreementHeader struct {
	values map[string]any
}

func (h agreementHeader) given(key string) bool {
	_, ok := h.values[key]

	return ok
}

func (h agreementHeader) applyTexts(entity *rateagreement.RateAgreement) error {
	for key, field := range map[string]struct {
		target *string
		limit  int
	}{
		paramAgreementCode: {&entity.Code, maxAgreementCode},
		paramAgreementName: {&entity.Name, maxAgreementName},
		paramDescription:   {&entity.Description, maxAgreementText},
		paramContractRef:   {&entity.ContractRef, maxContractRef},
	} {
		text, err := optionalBoundedText(h.values, key, field.limit)
		if err != nil {
			return err
		}
		if text != nil {
			*field.target = *text
		}
	}
	if h.given(paramAgreementCurrency) {
		code := strings.ToUpper(strings.TrimSpace(optionalString(h.values, paramAgreementCurrency)))
		if len(code) != currencyCodeLength {
			return fmt.Errorf("%s must be a three-letter code such as USD", paramAgreementCurrency)
		}
		entity.Currency = code
	}

	return nil
}

func (h agreementHeader) applyTerms(entity *rateagreement.RateAgreement) error {
	var err error
	if h.given(paramAgreementType) {
		if entity.AgreementType, err = requireEnum(
			h.values, paramAgreementType, agreementTypes.Values,
		); err != nil {
			return err
		}
	}
	if h.given(paramPriority) {
		priority, rangeErr := requireIntInRange(h.values, paramPriority, 0, maxAgreementPriority)
		if rangeErr != nil {
			return rangeErr
		}
		entity.Priority = int16(priority) //nolint:gosec // bounded to maxAgreementPriority above
	}
	if h.given(paramRenewalNoticeDays) {
		days, rangeErr := requireIntInRange(h.values, paramRenewalNoticeDays, 0, maxRenewalNotice)
		if rangeErr != nil {
			return rangeErr
		}
		entity.RenewalNoticeDays = int16(days) //nolint:gosec // bounded to maxRenewalNotice above
	}
	if autoRenew, flagErr := optionalBoolPointer(h.values, paramAutoRenew); flagErr != nil {
		return flagErr
	} else if autoRenew != nil {
		entity.AutoRenew = *autoRenew
	}
	for key, target := range map[string]*decimal.NullDecimal{
		paramDefaultMinCharge: &entity.DefaultMinCharge,
		paramDefaultMaxCharge: &entity.DefaultMaxCharge,
	} {
		value, given, moneyErr := optionalLaneMoney(h.values, key)
		if moneyErr != nil {
			return moneyErr
		}
		if given {
			*target = value
		}
	}

	return nil
}

func (h agreementHeader) applyWindow(entity *rateagreement.RateAgreement) error {
	if h.given(paramEffectiveFrom) {
		from, err := requireDay(h.values, paramEffectiveFrom)
		if err != nil {
			return err
		}
		entity.EffectiveFrom = from
	}
	if h.given(paramEffectiveTo) {
		to, given, err := optionalDay(h.values, paramEffectiveTo)
		if err != nil {
			return err
		}
		if given {
			entity.EffectiveTo = &to
		} else {
			entity.EffectiveTo = nil
		}
	}

	return nil
}

func (h agreementHeader) apply(entity *rateagreement.RateAgreement) error {
	for _, step := range []func(*rateagreement.RateAgreement) error{
		h.applyTexts,
		h.applyTerms,
		h.applyWindow,
	} {
		if err := step(entity); err != nil {
			return err
		}
	}

	return nil
}

func stampLaneWindows(entity *rateagreement.RateAgreement, rules []*rateagreement.RateAgreementRule) {
	for _, rule := range rules {
		if rule.ID.IsNil() || rule.EffectiveFrom < entity.EffectiveFrom {
			rule.EffectiveFrom = entity.EffectiveFrom
		}
		if rule.ID.IsNil() {
			rule.EffectiveTo = entity.EffectiveTo
		}
	}
}

type agreementView struct {
	Code          string `json:"code"`
	Name          string `json:"name"`
	PartyType     string `json:"partyType"`
	CustomerID    string `json:"customerId,omitempty"`
	CarrierID     string `json:"carrierId,omitempty"`
	AgreementType string `json:"agreementType"`
	Status        string `json:"status"`
	EffectiveFrom int64  `json:"effectiveFrom"`
	EffectiveTo   int64  `json:"effectiveTo,omitempty"`
	Currency      string `json:"currency"`
	Priority      int16  `json:"priority"`
	ContractRef   string `json:"contractRef,omitempty"`
	MinCharge     string `json:"defaultMinCharge,omitempty"`
	MaxCharge     string `json:"defaultMaxCharge,omitempty"`
	Lanes         int    `json:"lanes"`
	ReviewComment string `json:"reviewComment,omitempty"`
}

func agreementViewOf(entity *rateagreement.RateAgreement) *agreementView {
	view := &agreementView{
		Code:          entity.Code,
		Name:          entity.Name,
		PartyType:     string(entity.PartyType),
		AgreementType: string(entity.AgreementType),
		Status:        string(entity.Status),
		EffectiveFrom: entity.EffectiveFrom,
		Currency:      entity.Currency,
		Priority:      entity.Priority,
		ContractRef:   entity.ContractRef,
		Lanes:         len(entity.Rules),
		ReviewComment: entity.ReviewComment,
	}
	if entity.CustomerID != nil {
		view.CustomerID = entity.CustomerID.String()
	}
	if entity.CarrierID != nil {
		view.CarrierID = entity.CarrierID.String()
	}
	if entity.EffectiveTo != nil {
		view.EffectiveTo = *entity.EffectiveTo
	}
	if entity.DefaultMinCharge.Valid {
		view.MinCharge = entity.DefaultMinCharge.Decimal.String()
	}
	if entity.DefaultMaxCharge.Valid {
		view.MaxCharge = entity.DefaultMaxCharge.Decimal.String()
	}

	return view
}

func agreementOptions() []toolpreview.Option {
	return []toolpreview.Option{
		toolpreview.WithRefs(map[string]permission.Resource{
			paramCustomerID: permission.ResourceCustomer,
			paramCarrierID:  permission.ResourceCarrier,
		}),
		toolpreview.Types(rateAgreementDates),
	}
}

func agreementRecord(entity *rateagreement.RateAgreement) toolpreview.Record {
	return toolpreview.Record{
		Resource: permission.ResourceRateAgreement,
		ID:       entity.ID,
		Label:    entity.Code + " " + entity.Name,
		Version:  pinnedVersion(entity.Version),
	}
}

func agreementResult(action string, entity *rateagreement.RateAgreement) *agent.ToolExecutionResult {
	return &agent.ToolExecutionResult{
		Action: action,
		Kind:   rateAgreementKind,
		Name:   entity.Code,
		IDs:    map[string]string{paramRateAgreementID: entity.ID.String()},
		Record: recordOf(rateAgreementRecordEntity, entity.ID),
	}
}

func rateDraftSpec(spec *receivableSpec) *receivableSpec {
	spec.resource = permission.ResourceRateAgreement
	spec.artifact = rateAgreementRecordEntity
	spec.taintHold = rateDraftTaintHold

	return fuelInternalSpec(spec)
}

func rateMoneySpec(spec *receivableSpec) *receivableSpec {
	spec.resource = permission.ResourceRateAgreement

	return ledgerMoneySpec(spec)
}

func loadAgreement(
	ctx context.Context,
	agreements rateAgreementKeeper,
	tenant pagination.TenantInfo,
	id pulid.ID,
) (*rateagreement.RateAgreement, error) {
	return agreements.GetByID(ctx, &repositories.GetRateAgreementByIDRequest{
		RateAgreementID: id,
		TenantInfo:      tenant,
		IncludeChildren: true,
	})
}

type agreementDraft struct {
	params *serviceports.ToolExecuteParams
}

func (d agreementDraft) build(
	ctx context.Context,
	lanes laneReader,
) (*rateagreement.RateAgreement, error) {
	values := d.params.Params
	party, err := requireEnum(values, paramPartyType, partyTypes.Values)
	if err != nil {
		return nil, err
	}
	entity := &rateagreement.RateAgreement{
		OrganizationID: d.params.OrganizationID,
		BusinessUnitID: d.params.BusinessUnitID,
		PartyType:      party,
		AgreementType:  rateagreement.AgreementTypeContract,
		Currency:       "USD",
	}
	partyKey := paramCustomerID
	if party == rateagreement.PartyTypeCarrier {
		partyKey = paramCarrierID
	}
	partyID, err := requirePulid(values, partyKey)
	if err != nil {
		return nil, fmt.Errorf("a %s agreement names its %s: %w", party, partyKey, err)
	}
	if party == rateagreement.PartyTypeCarrier {
		entity.CarrierID = &partyID
	} else {
		entity.CustomerID = &partyID
	}
	if err = (agreementHeader{values: values}).apply(entity); err != nil {
		return nil, err
	}
	if entity.Rules, err = lanes.newLanes(ctx, values); err != nil {
		return nil, err
	}
	stampLaneWindows(entity, entity.Rules)

	return entity, nil
}

func newDraftRateAgreementTool(
	agreements rateAgreementKeeper,
	states stateFinder,
) serviceports.AgentTool {
	properties := agreementHeaderProperties()
	properties[paramPartyType] = agenttoolschema.Enum("Whether it prices what a customer is "+
		"charged or what a carrier is paid.", partyTypes)
	properties[paramCustomerID] = stringProperty("The customer, from list_customers, for a "+
		"Customer agreement.", 0)
	properties[paramCarrierID] = stringProperty("The carrier, from list_carriers, for a "+
		"Carrier agreement.", 0)
	properties[paramLanes] = laneProperty(false)
	lanes := laneReader{states: states}

	return newReportingReceivableTool(rateDraftSpec(&receivableSpec{
		name: "draft_rate_agreement",
		description: "Draft a rate agreement with a customer or a carrier from a rate sheet " +
			"or a quote the person agreed: its code, name, dates, currency and lanes. It is " +
			"saved as a draft and prices nothing until submit_rate_agreement sends it for " +
			"review and a person approves it.",
		operation:  permission.OpCreate,
		reversible: true,
		rationale: "Saves a draft agreement inside Trenova that prices nothing until a person " +
			"approves it; archive_rate_agreement retires it.",
		properties: properties,
		required: []string{
			paramPartyType, paramAgreementCode, paramAgreementName, paramEffectiveFrom,
		},
		searchTerms: []string{"contract", "tariff", "rates", "lanes", "pricing", "rate sheet"},
	}), receivablePlan[agreementDraft, *rateagreement.RateAgreement]{
		request: func(params *serviceports.ToolExecuteParams) (agreementDraft, error) {
			return agreementDraft{params: params}, nil
		},
		plan: func(
			ctx context.Context,
			draft agreementDraft,
			_ *serviceports.ToolExecuteParams,
		) (*rateagreement.RateAgreement, error) {
			entity, err := draft.build(ctx, lanes)
			if err != nil {
				return nil, err
			}

			return agreements.PlanCreate(ctx, entity)
		},
		refused: func(agreementDraft) string {
			return "Would draft a rate agreement."
		},
		render: func(_ agreementDraft, planned *rateagreement.RateAgreement) (*agent.ToolPreview, error) {
			record := agreementRecord(planned)
			record.Version = nil
			change, err := toolpreview.Create(record, agreementViewOf(planned), agreementOptions()...)
			if err != nil {
				return nil, err
			}

			return toolpreview.Build(fmt.Sprintf(
				"Would draft the %s rate agreement %s with %s from %s; it prices nothing until "+
					"it is approved.",
				strings.ToLower(string(planned.PartyType)), planned.Code,
				countOf(len(planned.Rules), "lane"), dayLabel(planned.EffectiveFrom),
			), change), nil
		},
		run: func(
			ctx context.Context,
			draft agreementDraft,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			entity, err := draft.build(ctx, lanes)
			if err != nil {
				return nil, err
			}
			created, err := agreements.Create(ctx, entity, params.Actor.UserID)
			if err != nil {
				return nil, err
			}

			return agreementResult("drafted", created), nil
		},
	})
}

type agreementRevision struct {
	id     pulid.ID
	values map[string]any
}

var errOnlyDraftsRevise = fmt.Errorf(
	"only a draft agreement is revised here; an agreement in review is rejected back to " +
		"draft first, and an active one's lanes change with amend_rate_agreement_rules",
)

func (r agreementRevision) build(
	ctx context.Context,
	agreements rateAgreementKeeper,
	lanes laneReader,
	params *serviceports.ToolExecuteParams,
) (*rateagreement.RateAgreement, error) {
	stored, err := loadAgreement(ctx, agreements, tenantFrom(*params), r.id)
	if err != nil {
		return nil, err
	}
	if stored.Status != rateagreement.StatusDraft {
		return nil, errOnlyDraftsRevise
	}

	revised := *stored
	if err = (agreementHeader{values: r.values}).apply(&revised); err != nil {
		return nil, err
	}
	if _, given := r.values[paramLanes]; given {
		if revised.Rules, err = lanes.revisedLanes(ctx, r.values, stored.Rules); err != nil {
			return nil, err
		}
	}
	stampLaneWindows(&revised, revised.Rules)

	return &revised, nil
}

func newReviseRateAgreementDraftTool(
	agreements rateAgreementKeeper,
	states stateFinder,
) serviceports.AgentTool {
	properties := agreementHeaderProperties()
	properties[paramRateAgreementID] = rateAgreementIDProperty()
	properties[paramLanes] = laneProperty(true)
	lanes := laneReader{states: states}

	return newReportingReceivableTool(rateDraftSpec(&receivableSpec{
		name: "revise_rate_agreement_draft",
		description: "Rewrite a draft rate agreement before it is submitted: its header " +
			"terms or its lanes. Fields left out keep their value. Only a draft is revised " +
			"here; an active agreement's lanes change with amend_rate_agreement_rules. Read it " +
			"with get_rate_agreement first for its lanes' ruleIds.",
		operation:  permission.OpUpdate,
		reversible: true,
		rationale: "Changes a draft agreement inside Trenova that prices nothing yet; a later " +
			"revision changes it back.",
		properties: properties,
		required:   []string{paramRateAgreementID},
		target:     targetRateAgreement,
	}), receivablePlan[agreementRevision, *rateagreementservice.UpdatePlan]{
		request: func(params *serviceports.ToolExecuteParams) (agreementRevision, error) {
			id, err := requirePulid(params.Params, paramRateAgreementID)
			if err != nil {
				return agreementRevision{}, err
			}
			values := make(map[string]any, len(params.Params))
			for key, value := range params.Params {
				if key != paramRateAgreementID {
					values[key] = value
				}
			}
			if len(values) == 0 {
				return agreementRevision{}, fmt.Errorf("name at least one term to revise")
			}

			return agreementRevision{id: id, values: values}, nil
		},
		plan: func(
			ctx context.Context,
			revision agreementRevision,
			params *serviceports.ToolExecuteParams,
		) (*rateagreementservice.UpdatePlan, error) {
			revised, err := revision.build(ctx, agreements, lanes, params)
			if err != nil {
				return nil, err
			}

			return agreements.PlanUpdate(ctx, revised)
		},
		refused: func(agreementRevision) string {
			return "Would revise a draft rate agreement."
		},
		render: func(
			_ agreementRevision,
			plan *rateagreementservice.UpdatePlan,
		) (*agent.ToolPreview, error) {
			change, err := toolpreview.Changed(
				agreementRecord(plan.Before),
				agreementViewOf(plan.Before),
				agreementViewOf(plan.After),
				agreementOptions()...,
			)
			if err != nil {
				return nil, err
			}

			return toolpreview.Build(fmt.Sprintf(
				"Would revise the draft rate agreement %s: %s replaced or dropped and %s "+
					"added.",
				plan.Before.Code,
				countOf(len(plan.SupersededIDs), "lane"),
				countOf(len(plan.Inserts), "lane"),
			), change), nil
		},
		run: func(
			ctx context.Context,
			revision agreementRevision,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			revised, err := revision.build(ctx, agreements, lanes, params)
			if err != nil {
				return nil, err
			}
			updated, err := agreements.Update(ctx, revised, params.Actor.UserID)
			if err != nil {
				return nil, err
			}

			return agreementResult("revised", updated), nil
		},
	})
}

func newDuplicateRateAgreementTool(agreements rateAgreementKeeper) serviceports.AgentTool {
	return newReportingReceivableTool(rateDraftSpec(&receivableSpec{
		name: "duplicate_rate_agreement",
		description: "Copy a rate agreement with its lanes, weight breaks, accessorials and " +
			"fuel terms into a new draft with none of the original's history or approvals, to " +
			"start a renewal. revise_rate_agreement_draft then changes the copy.",
		operation:  permission.OpDuplicate,
		reversible: true,
		rationale: "Saves a draft copy of an agreement inside Trenova that prices nothing " +
			"until a person approves it.",
		properties: map[string]any{
			paramRateAgreementID: rateAgreementIDProperty(),
			paramAgreementCode: stringProperty("The copy's code. Defaults to the original's "+
				"with -COPY.", maxAgreementCode),
			paramAgreementName: stringProperty("The copy's name. Defaults to the original's "+
				"with (Copy).", maxAgreementName),
		},
		required: []string{paramRateAgreementID},
		target:   targetRateAgreement,
	}), receivablePlan[*rateagreementservice.DuplicateRateAgreementRequest, *rateagreementservice.DuplicatePlan]{
		request: func(
			params *serviceports.ToolExecuteParams,
		) (*rateagreementservice.DuplicateRateAgreementRequest, error) {
			id, err := requirePulid(params.Params, paramRateAgreementID)
			if err != nil {
				return nil, err
			}
			code, err := boundedText(params.Params, paramAgreementCode, maxAgreementCode)
			if err != nil {
				return nil, err
			}
			name, err := boundedText(params.Params, paramAgreementName, maxAgreementName)
			if err != nil {
				return nil, err
			}

			return &rateagreementservice.DuplicateRateAgreementRequest{
				TenantInfo:      tenantFrom(*params),
				RateAgreementID: id,
				Code:            code,
				Name:            name,
			}, nil
		},
		plan: func(
			ctx context.Context,
			req *rateagreementservice.DuplicateRateAgreementRequest,
			_ *serviceports.ToolExecuteParams,
		) (*rateagreementservice.DuplicatePlan, error) {
			return agreements.PlanDuplicate(ctx, req)
		},
		refused: func(*rateagreementservice.DuplicateRateAgreementRequest) string {
			return "Would copy a rate agreement into a new draft."
		},
		render: func(
			_ *rateagreementservice.DuplicateRateAgreementRequest,
			plan *rateagreementservice.DuplicatePlan,
		) (*agent.ToolPreview, error) {
			record := agreementRecord(plan.Copy)
			record.Version = nil
			change, err := toolpreview.Create(record, agreementViewOf(plan.Copy), agreementOptions()...)
			if err != nil {
				return nil, err
			}

			return toolpreview.Build(fmt.Sprintf(
				"Would copy the rate agreement %s into the draft %s with %s.",
				plan.Original.Code, plan.Copy.Code, countOf(len(plan.Copy.Rules), "lane"),
			), change), nil
		},
		run: func(
			ctx context.Context,
			req *rateagreementservice.DuplicateRateAgreementRequest,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			created, err := agreements.Duplicate(ctx, req, params.Actor.UserID)
			if err != nil {
				return nil, err
			}

			return agreementResult("duplicated", created), nil
		},
	})
}

type agreementReviewStep struct {
	name        string
	review      rateagreementservice.Review
	operation   permission.Operation
	money       bool
	needsReason bool
	description string
	rationale   string
	outcome     string
}

func (s *agreementReviewStep) spec() *receivableSpec {
	properties := map[string]any{paramRateAgreementID: rateAgreementIDProperty()}
	required := []string{paramRateAgreementID}
	comment := "A note on the review for whoever reads the agreement next."
	if s.needsReason {
		comment = "Why, in a sentence the agreement's owner can act on."
		required = append(required, paramReviewComment)
	}
	properties[paramReviewComment] = stringProperty(comment, maxReviewComment)

	spec := &receivableSpec{
		name:        s.name,
		description: s.description,
		operation:   s.operation,
		rationale:   s.rationale,
		properties:  properties,
		required:    required,
		target:      targetRateAgreement,
	}
	if s.money {
		return rateMoneySpec(spec)
	}
	spec.resource = permission.ResourceRateAgreement
	spec.maxTier = agent.TierPropose

	return fuelInternalSpec(spec)
}

func (s *agreementReviewStep) request(
	params *serviceports.ToolExecuteParams,
) (*rateagreementservice.ApprovalActionRequest, error) {
	id, err := requirePulid(params.Params, paramRateAgreementID)
	if err != nil {
		return nil, err
	}
	read := boundedText
	if s.needsReason {
		read = requireBoundedText
	}
	comment, err := read(params.Params, paramReviewComment, maxReviewComment)
	if err != nil {
		return nil, err
	}

	return &rateagreementservice.ApprovalActionRequest{
		TenantInfo: tenantFrom(*params),
		EntityID:   id,
		Comment:    comment,
	}, nil
}

func newAgreementReviewTool(
	agreements rateAgreementKeeper,
	step *agreementReviewStep,
) serviceports.AgentTool {
	return newReceivableTool(step.spec(), receivablePlan[*rateagreementservice.ApprovalActionRequest, *rateagreementservice.AgreementChange]{
		request: step.request,
		plan: func(
			ctx context.Context,
			req *rateagreementservice.ApprovalActionRequest,
			_ *serviceports.ToolExecuteParams,
		) (*rateagreementservice.AgreementChange, error) {
			return agreements.PlanReview(ctx, step.review, req)
		},
		refused: func(*rateagreementservice.ApprovalActionRequest) string {
			return fmt.Sprintf("Would %s a rate agreement.", strings.ToLower(string(step.review)))
		},
		render: func(
			_ *rateagreementservice.ApprovalActionRequest,
			plan *rateagreementservice.AgreementChange,
		) (*agent.ToolPreview, error) {
			change, err := toolpreview.Changed(
				agreementRecord(plan.Before),
				agreementViewOf(plan.Before),
				agreementViewOf(plan.After),
				toolpreview.Only(fieldStatus, "reviewComment"),
			)
			if err != nil {
				return nil, err
			}

			return toolpreview.Build(fmt.Sprintf(
				"Would %s the rate agreement %s (now %s); %s.",
				strings.ToLower(string(step.review)), plan.Before.Code, plan.Before.Status,
				step.outcome,
			), change), nil
		},
		run: func(
			ctx context.Context,
			req *rateagreementservice.ApprovalActionRequest,
			_ *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			_, err := agreements.Review(ctx, step.review, req)

			return nil, err
		},
	})
}

func agreementReviewSteps() []*agreementReviewStep {
	return []*agreementReviewStep{
		{
			name:      "submit_rate_agreement",
			review:    rateagreementservice.ReviewSubmit,
			operation: permission.OpSubmit,
			description: "Submit a draft rate agreement for review. It prices nothing until a " +
				"person approves it. Read it with get_rate_agreement first.",
			rationale: "Moves a draft agreement into review inside Trenova; it prices nothing " +
				"until a person approves it.",
			outcome: "a reviewer approves or rejects it and it prices nothing until then",
		},
		{
			name:      "approve_rate_agreement",
			review:    rateagreementservice.ReviewApprove,
			operation: permission.OpApprove,
			money:     true,
			description: "Propose approving a rate agreement in review, which is the moment it " +
				"starts pricing shipments. A person always decides.",
			rationale: "Turns on an agreement that prices what customers are charged or carriers " +
				"are paid; only a person approves one.",
			outcome: "it prices shipments from its effective date",
		},
		{
			name:        "reject_rate_agreement",
			review:      rateagreementservice.ReviewReject,
			operation:   permission.OpReject,
			needsReason: true,
			description: "Propose sending a rate agreement in review back to draft, with what is " +
				"wrong with it. A person always decides.",
			rationale: "Returns an agreement to its author with the reviewer's reason; it " +
				"prices nothing either way, but the review is a person's decision.",
			outcome: "it goes back to its author as a draft",
		},
		{
			name:        "suspend_rate_agreement",
			review:      rateagreementservice.ReviewSuspend,
			operation:   permission.OpUpdate,
			money:       true,
			needsReason: true,
			description: "Propose suspending an active rate agreement, such as for a customer " +
				"on credit hold, with the reason. Shipments stop rating against it at once; " +
				"resume_rate_agreement puts it back. A person always decides.",
			rationale: "Stops an agreement pricing shipments, which changes what customers " +
				"are charged; only a person suspends one.",
			outcome: "shipments stop rating against it until it is resumed",
		},
		{
			name:      "resume_rate_agreement",
			review:    rateagreementservice.ReviewResume,
			operation: permission.OpUpdate,
			money:     true,
			description: "Propose putting a suspended rate agreement back in service without " +
				"another approval round. A person always decides.",
			rationale: "Turns an agreement back on so it prices shipments again; only a person " +
				"resumes one.",
			outcome: "it prices shipments again",
		},
		{
			name:      "archive_rate_agreement",
			review:    rateagreementservice.ReviewArchive,
			operation: permission.OpArchive,
			money:     true,
			description: "Propose retiring a rate agreement for good. It is kept because its " +
				"quotes point at it but never prices again. A person always decides.",
			rationale: "Ends an agreement for good, which changes what customers are charged " +
				"when it was active; only a person archives one.",
			outcome: "it never prices a shipment again",
		},
	}
}

type ruleAmendment struct {
	id         pulid.ID
	from       int64
	superseded []pulid.ID
	values     map[string]any
}

func (a *ruleAmendment) request(
	ctx context.Context,
	agreements rateAgreementKeeper,
	lanes laneReader,
	params *serviceports.ToolExecuteParams,
) (*repositories.AmendRateAgreementRulesRequest, error) {
	agreement, err := loadAgreement(ctx, agreements, tenantFrom(*params), a.id)
	if err != nil {
		return nil, err
	}
	rules, err := lanes.newLanes(ctx, a.values)
	if err != nil {
		return nil, err
	}
	for _, rule := range rules {
		rule.OrganizationID = agreement.OrganizationID
		rule.BusinessUnitID = agreement.BusinessUnitID
		rule.RateAgreementID = agreement.ID
		rule.PartyType = agreement.PartyType
		rule.PartyID = agreement.PartyID()
		rule.EffectiveFrom = a.from
		rule.EffectiveTo = agreement.EffectiveTo
	}

	return &repositories.AmendRateAgreementRulesRequest{
		TenantInfo:      tenantFrom(*params),
		RateAgreementID: agreement.ID,
		EffectiveFrom:   a.from,
		SupersededIDs:   a.superseded,
		Rules:           rules,
	}, nil
}

func newAmendRateAgreementRulesTool(
	agreements rateAgreementKeeper,
	states stateFinder,
) serviceports.AgentTool {
	lanes := laneReader{states: states}

	return newReceivableTool(rateMoneySpec(&receivableSpec{
		name: "amend_rate_agreement_rules",
		description: "Propose changing an active rate agreement's lanes from a day: close out " +
			"the lanes it replaces and add their successors, keeping the old rates in history. " +
			"Take the lanes' ruleIds from get_rate_agreement. It changes what shipments are " +
			"charged from that day, so a person always decides.",
		operation: permission.OpUpdate,
		rationale: "Changes the rates an agreement prices shipments at; only a person amends " +
			"a live contract.",
		properties: map[string]any{
			paramRateAgreementID: rateAgreementIDProperty(),
			paramEffectiveFrom:   dateProperty("The first day the new lanes price shipments."),
			paramSupersededRuleIDs: idListProperty("The lanes this closes out, by ruleId "+
				"from get_rate_agreement.", maxAgreementLanes),
			paramLanes: laneProperty(false),
		},
		required: []string{paramRateAgreementID, paramEffectiveFrom},
		target:   targetRateAgreement,
	}), receivablePlan[*ruleAmendment, *rateagreementservice.AmendmentPlan]{
		request: func(params *serviceports.ToolExecuteParams) (*ruleAmendment, error) {
			id, err := requirePulid(params.Params, paramRateAgreementID)
			if err != nil {
				return nil, err
			}
			from, err := requireDay(params.Params, paramEffectiveFrom)
			if err != nil {
				return nil, err
			}
			amendment := &ruleAmendment{id: id, from: from, values: params.Params}
			if _, given := params.Params[paramSupersededRuleIDs]; given {
				if amendment.superseded, err = requirePulidSlice(
					params.Params, paramSupersededRuleIDs, maxAgreementLanes,
				); err != nil {
					return nil, err
				}
			}

			return amendment, nil
		},
		plan: func(
			ctx context.Context,
			amendment *ruleAmendment,
			params *serviceports.ToolExecuteParams,
		) (*rateagreementservice.AmendmentPlan, error) {
			req, err := amendment.request(ctx, agreements, lanes, params)
			if err != nil {
				return nil, err
			}

			return agreements.PlanAmendRules(ctx, req)
		},
		refused: func(*ruleAmendment) string {
			return "Would amend a rate agreement's lanes."
		},
		render: renderRuleAmendment,
		run: func(
			ctx context.Context,
			amendment *ruleAmendment,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			req, err := amendment.request(ctx, agreements, lanes, params)
			if err != nil {
				return nil, err
			}
			_, err = agreements.AmendRules(ctx, req, params.Actor.UserID)

			return nil, err
		},
	})
}

func laneRateLabel(rule *rateagreement.RateAgreementRule) string {
	if rule.Label != "" {
		return rule.Label
	}

	return rule.LaneKey
}

func renderRuleAmendment(
	_ *ruleAmendment,
	plan *rateagreementservice.AmendmentPlan,
) (*agent.ToolPreview, error) {
	lines := make([]agent.MoneyLine, 0, len(plan.Superseded)+len(plan.Rules))
	for _, rule := range plan.Superseded {
		lines = append(lines, agent.MoneyLine{
			Label:  laneRateLabel(rule) + " (closed out)",
			Before: rule.Rate,
		})
	}
	for _, rule := range plan.Rules {
		lines = append(lines, agent.MoneyLine{
			Label: laneRateLabel(rule) + " (new)",
			After: rule.Rate,
		})
	}
	change := toolpreview.Money(
		agreementRecord(plan.Agreement),
		toolpreview.MoneyBlock(plan.Agreement.Currency, lines...),
	)

	return toolpreview.Build(fmt.Sprintf(
		"Would amend the rate agreement %s from %s: close out %s and add %s. Shipments from "+
			"that day are charged at the new rates.",
		plan.Agreement.Code, dayLabel(plan.EffectiveFrom),
		countOf(len(plan.Superseded), "lane"), countOf(len(plan.Rules), "lane"),
	), change), nil
}

func rateIncreaseRequestFrom(
	params *serviceports.ToolExecuteParams,
) (*rateagreementservice.RateIncreaseRequest, error) {
	req := &rateagreementservice.RateIncreaseRequest{TenantInfo: tenantFrom(*params)}
	var err error
	if req.EffectiveFrom, err = requireDay(params.Params, paramEffectiveFrom); err != nil {
		return nil, err
	}
	if _, given := params.Params[paramAgreementIDs]; given {
		if req.AgreementIDs, err = requirePulidSlice(
			params.Params, paramAgreementIDs, maxIncreaseScope,
		); err != nil {
			return nil, err
		}
	}
	if req.CustomerID, err = optionalPulidParam(params.Params, paramCustomerID); err != nil {
		return nil, err
	}
	if req.CarrierID, err = optionalPulidParam(params.Params, paramCarrierID); err != nil {
		return nil, err
	}
	if party, given, partyErr := optionalEnum(
		params.Params, paramPartyType, partyTypes.Values,
	); partyErr != nil {
		return nil, partyErr
	} else if given {
		req.PartyType = party
	}
	for key, target := range map[string]*decimal.NullDecimal{
		paramPercentChange: &req.Adjustment.PercentChange,
		paramFlatChange:    &req.Adjustment.FlatChange,
	} {
		value, present, decimalErr := optionalDecimal(params.Params, key)
		if decimalErr != nil {
			return nil, decimalErr
		}
		*target = decimal.NullDecimal{Decimal: value, Valid: present}
	}

	return req, nil
}

func rateAdjustmentLabel(adjustment rateagreementservice.RateAdjustment) string {
	if adjustment.PercentChange.Valid {
		return adjustment.PercentChange.Decimal.String() + "%"
	}
	if adjustment.FlatChange.Valid {
		return adjustment.FlatChange.Decimal.String() + " per rating unit"
	}

	return "nothing"
}

func renderRateIncrease(
	req *rateagreementservice.RateIncreaseRequest,
	plan *rateagreementservice.RateIncreasePlan,
) (*agent.ToolPreview, error) {
	samples := make([]string, 0, min(len(plan.Lines), maxPreviewLanes))
	for _, line := range plan.Lines[:min(len(plan.Lines), maxPreviewLanes)] {
		label := line.Label
		if label == "" {
			label = line.LaneKey
		}
		samples = append(samples, line.AgreementCode+" "+label)
	}
	if more := len(plan.Lines) - len(samples); more > 0 {
		samples = append(samples, countOf(more, "more lane"))
	}
	summary := fmt.Sprintf(
		"Would move %s on %s by %s from %s: %s.",
		countOf(len(plan.Lines), "lane rate"),
		countOf(plan.AgreementCount, rateAgreementKind),
		rateAdjustmentLabel(req.Adjustment),
		dayLabel(plan.EffectiveFrom),
		strings.Join(samples, "; "),
	)
	if plan.SkippedNoRate > 0 {
		summary += fmt.Sprintf(" %s priced by a rate matrix are left as they are.",
			countOf(plan.SkippedNoRate, "lane"))
	}

	return toolpreview.Build(summary), nil
}

func newApplyRateIncreaseTool(agreements rateAgreementKeeper) serviceports.AgentTool {
	return newReceivableTool(rateMoneySpec(&receivableSpec{
		name: "apply_rate_increase",
		description: "Propose a general rate increase or decrease: move every lane rate on " +
			"the named agreements, one customer's or carrier's, or every active agreement of " +
			"a party type, by a percent or a flat amount, from a day. The old rates stay in " +
			"history. A person always decides.",
		operation: permission.OpUpdate,
		rationale: "Changes the rates many agreements charge customers or pay carriers; " +
			"only a person applies a rate increase.",
		properties: map[string]any{
			paramEffectiveFrom: dateProperty("The first day the new rates price shipments."),
			paramAgreementIDs: idListProperty("The agreements to move, from "+
				"list_rate_agreements; wins over the other scopes.", maxIncreaseScope),
			paramCustomerID: stringProperty("Every active agreement with this customer, from "+
				"list_customers.", 0),
			paramCarrierID: stringProperty("Every active agreement with this carrier, from "+
				"list_carriers.", 0),
			paramPartyType: agenttoolschema.Enum("Every active agreement of this party type, "+
				"when no agreements, customer or carrier is named.", partyTypes),
			paramPercentChange: stringProperty("The change in percent, such as 3.5 or -2. "+
				"Give this or flatChange.", 0),
			paramFlatChange: stringProperty("The change per rating unit, such as 0.10. Give "+
				"this or percentChange.", 0),
		},
		required:    []string{paramEffectiveFrom},
		searchTerms: []string{"gri", "general rate increase", "raise rates", "price increase"},
	}), receivablePlan[*rateagreementservice.RateIncreaseRequest, *rateagreementservice.RateIncreasePlan]{
		request: rateIncreaseRequestFrom,
		plan: func(
			ctx context.Context,
			req *rateagreementservice.RateIncreaseRequest,
			_ *serviceports.ToolExecuteParams,
		) (*rateagreementservice.RateIncreasePlan, error) {
			return agreements.PlanApplyRateIncrease(ctx, req)
		},
		refused: func(*rateagreementservice.RateIncreaseRequest) string {
			return "Would apply a general rate increase."
		},
		render: renderRateIncrease,
		run: func(
			ctx context.Context,
			req *rateagreementservice.RateIncreaseRequest,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			_, err := agreements.ApplyRateIncrease(ctx, req, params.Actor.UserID)

			return nil, err
		},
	})
}
