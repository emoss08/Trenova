package ratetools

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/internal/modules/places/domain/rategeo"
	"github.com/emoss08/trenova/internal/modules/rating/app/rateagreementservice"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateagreementports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/maputils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const paramRateAgreementID = "rateAgreementId"

const paramPartyType = "partyType"

const paramAgreementCode = "code"

const paramAgreementName = "name"

const paramDescription = "description"

const paramAgreementType = "agreementType"

const paramContractRef = "contractRef"

const paramPriority = "priority"

const paramAutoRenew = "autoRenew"

const paramRenewalNoticeDays = "renewalNoticeDays"

const paramAgreementCurrency = "currency"

const paramDefaultMinCharge = "defaultMinCharge"

const paramDefaultMaxCharge = "defaultMaxCharge"

const paramLanes = "lanes"

const paramRuleID = "ruleId"

const paramLaneLabel = "label"

const paramOriginScopeType = "originScopeType"

const paramOriginValue = "originValue"

const paramOriginCity = "originCity"

const paramDestinationType = "destinationScopeType"

const paramDestinationValue = "destinationValue"

const paramDestinationCity = "destinationCity"

const paramDirection = "direction"

const paramFormulaTemplateID = "formulaTemplateId"

const paramRateMatrixID = "rateMatrixId"

const paramLaneRate = "rate"

const paramLaneMinCharge = "minCharge"

const paramLaneMaxCharge = "maxCharge"

const paramHazmatOnly = "hazmatOnly"

const paramTempControlOnly = "tempControlOnly"

const paramSupersededRuleIDs = "supersededRuleIds"

const paramReviewComment = "comment"

const paramAgreementIDs = "agreementIds"

const paramPercentChange = "percentChange"

const paramFlatChange = "flatChange"

const paramRateImportID = "rateImportId"

const paramSimulationName = "name"

const paramSampleFrom = "sampleFrom"

const paramSampleTo = "sampleTo"

const paramSampleLimit = "sampleLimit"

const rateAgreementRecordEntity = "rate_agreement"

const rateAgreementKind = "rate agreement"

const maxAgreementCode = 50

const maxAgreementName = 150

const maxAgreementText = 2000

const maxContractRef = 100

const maxAgreementPriority = 1000

const maxRenewalNotice = 365

const maxAgreementLanes = 200

const maxLaneLabel = 150

const maxLaneValue = 120

const maxLaneCity = 100

const maxReviewComment = 1000

const maxIncreaseScope = 50

const maxSampleShipments = 50000

const maxPreviewLanes = 5

const rateImportSupplier = "The rate import, from rate_list_imports. Never guess one."

const rateDraftTaintHold = "An agreement drafted from a rate sheet or a message someone " +
	"outside sent is proposed, since its lanes are what the organization will charge."

var partyTypes = agenttoolschema.Source(
	"rateAgreement.partyType",
	rateagreement.PartyTypeValues(),
)

var agreementTypes = agenttoolschema.Source(
	"rateAgreement.agreementType",
	rateagreement.AgreementTypeValues(),
)

var laneDirections = agenttoolschema.Source(
	"rateAgreement.direction",
	rateagreement.DirectionValues(),
)

var laneScopeTypes = agenttoolschema.Source("rateAgreement.laneScopeType", []rategeo.ScopeType{
	rategeo.ScopeTypeAny,
	rategeo.ScopeTypeCountry,
	rategeo.ScopeTypeState,
	rategeo.ScopeTypeZone,
	rategeo.ScopeTypeCityState,
	rategeo.ScopeTypeZip3,
	rategeo.ScopeTypeZip5,
	rategeo.ScopeTypeLocation,
})

var rateAgreementDates = map[string]assistantartifact.DisplayType{
	common.ParamEffectiveFrom: assistantartifact.DisplayDate,
	common.ParamEffectiveTo:   assistantartifact.DisplayDate,
}

type RateAgreementKeeper interface {
	GetByID(
		ctx context.Context,
		req *rateagreementports.GetRateAgreementByIDRequest,
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
		actor *permissionports.RequestActor,
	) (*rateagreementservice.UpdatePlan, error)
	Update(
		ctx context.Context,
		entity *rateagreement.RateAgreement,
		actor *permissionports.RequestActor,
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
		req *rateagreementports.AmendRateAgreementRulesRequest,
	) (*rateagreementservice.AmendmentPlan, error)
	AmendRules(
		ctx context.Context,
		req *rateagreementports.AmendRateAgreementRulesRequest,
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

func targetRateAgreement(params map[string]any) (agentports.ToolTarget, bool) {
	return common.TargetOf(params, paramRateAgreementID, permission.ResourceRateAgreement)
}

func rateAgreementIDProperty() map[string]any {
	return agenttoolschema.RecordID(permission.ResourceRateAgreement, "The rate agreement",
		"rate_list_agreements or rate_get_agreement")
}

func laneProperty(forRevision bool) map[string]any {
	properties := map[string]any{
		paramLaneLabel: common.StringProperty(
			"A short name for the lane, such as Dallas to Atlanta.",
			maxLaneLabel,
		),
		paramOriginScopeType: agenttoolschema.Enum("How the origin is named. Defaults to "+
			"Any.", laneScopeTypes),
		paramOriginValue: common.StringProperty("The origin for its scope type: a three-letter "+
			"country code such as USA, a two-letter state code such as TX (for State and "+
			"CityState), a postal code (Zip3, Zip5), a location from location_list, or a "+
			"rate zone a lane of rate_get_agreement names. Leave it out for Any.", maxLaneValue),
		paramOriginCity: common.StringProperty("The origin city, for a CityState origin.",
			maxLaneCity),
		paramDestinationType: agenttoolschema.Enum("How the destination is named. "+
			"Defaults to Any.", laneScopeTypes),
		paramDestinationValue: common.StringProperty("The destination, read as originValue is.",
			maxLaneValue),
		paramDestinationCity: common.StringProperty("The destination city, for a CityState "+
			"destination.", maxLaneCity),
		paramDirection: agenttoolschema.Enum("Whether the lane also prices the reverse "+
			"haul. Defaults to Directional.", laneDirections),
		paramFormulaTemplateID: agenttoolschema.RecordIDText(permission.ResourceFormulaTemplate,
			"The formula template that prices the lane, from formula_list_templates, with the "+
				"lane's rate as its base rate. Give this or rateMatrixId."),
		paramRateMatrixID: agenttoolschema.RecordIDText(permission.ResourceRateMatrix,
			"The rate matrix that prices the lane, as a lane of rate_get_agreement names it. "+
				"Give this or formulaTemplateId."),
		paramLaneRate: common.StringProperty("The lane's rate as a decimal such as 2.35, in what "+
			"its formula template charges by.", 0),
		paramLaneMinCharge: common.StringProperty("The least the lane charges, as a decimal.", 0),
		paramLaneMaxCharge: common.StringProperty("The most the lane charges, as a decimal.", 0),
		paramPriority: common.IntegerProperty("Which lane wins when two match equally; higher "+
			"wins.", 0, maxAgreementPriority),
		paramHazmatOnly: common.BooleanProperty("Only hazardous freight rates on this lane."),
		paramTempControlOnly: common.BooleanProperty(
			"Only temperature-controlled freight rates on this lane.",
		),
	}
	description := "The lanes, each priced through a formula template or a rate matrix."
	if forRevision {
		properties[paramRuleID] = agenttoolschema.KindID("A lane already on the agreement, from "+
			"rate_get_agreement, to keep and change; leave it out for a new lane.",
			permission.KindRateAgreementLane)
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
		kind, err := common.RequireEnum(fields, keys[0], laneScopeTypes.Values)
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
		value, err := r.scopeValue(ctx, scope.Type, common.OptionalString(fields, keys[1]))
		if err != nil {
			return scope, fmt.Errorf("%s: %w", keys[1], err)
		}
		scope.Value = value
	}
	if _, given := fields[keys[2]]; given {
		scope.City = strings.TrimSpace(common.OptionalString(fields, keys[2]))
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
	value, present, err := common.OptionalDecimal(fields, key)
	if err != nil {
		return decimal.NullDecimal{}, true, err
	}
	if present && value.IsNegative() {
		return decimal.NullDecimal{}, true, fmt.Errorf("%s cannot be negative", key)
	}

	return decimal.NullDecimal{Decimal: value, Valid: present}, true, nil
}

func (r laneReader) applyPricing(
	rule *rateagreement.RateAgreementRule,
	fields map[string]any,
) error {
	for key, target := range map[string]**pulid.ID{
		paramFormulaTemplateID: &rule.FormulaTemplateID,
		paramRateMatrixID:      &rule.RateMatrixID,
	} {
		id, err := common.OptionalPulidParam(fields, key)
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
	label, err := common.OptionalBoundedText(fields, paramLaneLabel, maxLaneLabel)
	if err != nil {
		return err
	}
	if label != nil {
		rule.Label = *label
	}
	if _, given := fields[paramDirection]; given {
		direction, enumErr := common.RequireEnum(fields, paramDirection, laneDirections.Values)
		if enumErr != nil {
			return enumErr
		}
		rule.Direction = direction
	}
	if _, given := fields[paramPriority]; given {
		priority, rangeErr := common.RequireIntInRange(
			fields,
			paramPriority,
			0,
			maxAgreementPriority,
		)
		if rangeErr != nil {
			return rangeErr
		}
		rule.Priority = int16(priority) //nolint:gosec // bounded to maxAgreementPriority above
	}
	for key, target := range map[string]*bool{
		paramHazmatOnly:      &rule.HazmatOnly,
		paramTempControlOnly: &rule.TempControlOnly,
	} {
		flag, flagErr := common.OptionalBoolPointer(fields, key)
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
		ruleID, idErr := common.OptionalPulidParam(fields, paramRuleID)
		if idErr != nil {
			return nil, fmt.Errorf("%s[%d]: %w", paramLanes, idx, idErr)
		}
		if ruleID != nil {
			known, found := byID[*ruleID]
			if !found {
				return nil, fmt.Errorf("%s[%d]: %s is not a lane of this agreement; read "+
					"its lanes with rate_get_agreement", paramLanes, idx, ruleID)
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
		paramAgreementCode: common.StringProperty("A short code for the agreement, unique in the "+
			"organization, such as ACME-2027.", maxAgreementCode),
		paramAgreementName: common.StringProperty("The agreement's name.", maxAgreementName),
		paramDescription:   common.StringProperty("What the agreement covers.", maxAgreementText),
		paramAgreementType: agenttoolschema.Enum("The commercial arrangement. Defaults to "+
			"Contract.", agreementTypes),
		paramContractRef: common.StringProperty("The signed contract's own reference.",
			maxContractRef),
		paramPriority: common.IntegerProperty("Which agreement wins when two price a shipment; "+
			"higher wins.", 0, maxAgreementPriority),
		common.ParamEffectiveFrom: agenttoolschema.Date(
			"The first day the agreement prices shipments.",
		),
		common.ParamEffectiveTo: agenttoolschema.Date(
			"The last day it prices shipments. Leave it out for an open-ended agreement.",
		),
		paramAutoRenew: common.BooleanProperty("Whether it renews itself at its end."),
		paramRenewalNoticeDays: common.IntegerProperty(
			"How many days before its end to warn about "+
				"renewal.",
			0,
			maxRenewalNotice,
		),
		paramAgreementCurrency: common.StringProperty(
			"The ISO currency code it prices in. Defaults "+
				"to USD.",
			common.CurrencyCodeLength,
		),
		paramDefaultMinCharge: common.StringProperty(
			"The least any lane charges, as a decimal.",
			0,
		),
		paramDefaultMaxCharge: common.StringProperty("The most any lane charges, as a decimal.", 0),
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
		text, err := common.OptionalBoundedText(h.values, key, field.limit)
		if err != nil {
			return err
		}
		if text != nil {
			*field.target = *text
		}
	}
	if h.given(paramAgreementCurrency) {
		code := strings.ToUpper(
			strings.TrimSpace(common.OptionalString(h.values, paramAgreementCurrency)),
		)
		if len(code) != common.CurrencyCodeLength {
			return fmt.Errorf("%s must be a three-letter code such as USD", paramAgreementCurrency)
		}
		entity.Currency = code
	}

	return nil
}

func (h agreementHeader) applyTerms(entity *rateagreement.RateAgreement) error {
	var err error
	if h.given(paramAgreementType) {
		if entity.AgreementType, err = common.RequireEnum(
			h.values, paramAgreementType, agreementTypes.Values,
		); err != nil {
			return err
		}
	}
	if h.given(paramPriority) {
		priority, rangeErr := common.RequireIntInRange(
			h.values,
			paramPriority,
			0,
			maxAgreementPriority,
		)
		if rangeErr != nil {
			return rangeErr
		}
		entity.Priority = int16(priority) //nolint:gosec // bounded to maxAgreementPriority above
	}
	if h.given(paramRenewalNoticeDays) {
		days, rangeErr := common.RequireIntInRange(
			h.values,
			paramRenewalNoticeDays,
			0,
			maxRenewalNotice,
		)
		if rangeErr != nil {
			return rangeErr
		}
		entity.RenewalNoticeDays = int16(days) //nolint:gosec // bounded to maxRenewalNotice above
	}
	if autoRenew, flagErr := common.OptionalBoolPointer(h.values, paramAutoRenew); flagErr != nil {
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
	if h.given(common.ParamEffectiveFrom) {
		from, err := common.RequireDay(h.values, common.ParamEffectiveFrom)
		if err != nil {
			return err
		}
		entity.EffectiveFrom = from
	}
	if h.given(common.ParamEffectiveTo) {
		to, given, err := common.OptionalDay(h.values, common.ParamEffectiveTo)
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

func stampLaneWindows(
	entity *rateagreement.RateAgreement,
	rules []*rateagreement.RateAgreementRule,
) {
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
			common.ParamCustomerID: permission.ResourceCustomer,
			common.ParamCarrierID:  permission.ResourceCarrier,
		}),
		toolpreview.Types(rateAgreementDates),
	}
}

func agreementRecord(entity *rateagreement.RateAgreement) toolpreview.Record {
	return toolpreview.Record{
		Resource: permission.ResourceRateAgreement,
		ID:       entity.ID,
		Label:    entity.Code + " " + entity.Name,
		Version:  common.PinnedVersion(entity.Version),
	}
}

func agreementResult(
	action string,
	entity *rateagreement.RateAgreement,
) *agent.ToolExecutionResult {
	return &agent.ToolExecutionResult{
		Action: action,
		Kind:   rateAgreementKind,
		Name:   entity.Code,
		IDs:    map[string]string{paramRateAgreementID: entity.ID.String()},
		Record: common.RecordOf(rateAgreementRecordEntity, entity.ID),
	}
}

func rateDraftSpec(spec *common.ReceivableSpec) *common.ReceivableSpec {
	spec.Resource = permission.ResourceRateAgreement
	spec.Artifact = rateAgreementRecordEntity
	spec.TaintHold = rateDraftTaintHold

	return common.FuelInternalSpec(spec)
}

func rateMoneySpec(spec *common.ReceivableSpec) *common.ReceivableSpec {
	spec.Resource = permission.ResourceRateAgreement

	return common.LedgerMoneySpec(spec)
}

func loadAgreement(
	ctx context.Context,
	agreements RateAgreementKeeper,
	tenant pagination.TenantInfo,
	id pulid.ID,
) (*rateagreement.RateAgreement, error) {
	return agreements.GetByID(ctx, &rateagreementports.GetRateAgreementByIDRequest{
		RateAgreementID: id,
		TenantInfo:      tenant,
		IncludeChildren: true,
	})
}

type agreementDraft struct {
	params *agentports.ToolExecuteParams
}

func (d agreementDraft) Build(
	ctx context.Context,
	lanes laneReader,
) (*rateagreement.RateAgreement, error) {
	values := d.params.Params
	party, err := common.RequireEnum(values, paramPartyType, partyTypes.Values)
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
	partyKey := common.ParamCustomerID
	if party == rateagreement.PartyTypeCarrier {
		partyKey = common.ParamCarrierID
	}
	partyID, err := common.RequirePulid(values, partyKey)
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

func NewDraftRateAgreementTool(
	agreements RateAgreementKeeper,
	states stateFinder,
) agentports.AgentTool {
	properties := agreementHeaderProperties()
	properties[paramPartyType] = agenttoolschema.Enum("Whether it prices what a customer is "+
		"charged or what a carrier is paid.", partyTypes)
	properties[common.ParamCustomerID] = agenttoolschema.RecordIDText(permission.ResourceCustomer,
		"The customer, from customer_list, for a Customer agreement.")
	properties[common.ParamCarrierID] = agenttoolschema.RecordIDText(permission.ResourceCarrier,
		"The carrier, from carrier_list, for a Carrier agreement.")
	properties[paramLanes] = laneProperty(false)
	lanes := laneReader{states: states}

	return common.NewReportingReceivableTool(rateDraftSpec(&common.ReceivableSpec{
		Name: "rate_create_agreement",
		Description: "Draft a rate agreement with a customer or a carrier from a rate sheet " +
			"or a quote the person agreed: its code, name, dates, currency and lanes. It is " +
			"saved as a draft and prices nothing until rate_submit_agreement sends it for " +
			"review and a person approves it.",
		Operation:  permission.OpCreate,
		Reversible: true,
		Rationale: "Saves a draft agreement inside Trenova that prices nothing until a person " +
			"approves it; rate_archive_agreement retires it.",
		Properties: properties,
		Required: []string{
			paramPartyType, paramAgreementCode, paramAgreementName, common.ParamEffectiveFrom,
		},
		SearchTerms: []string{"tariff", "rate sheet", "quote", "pricing"},
	}), common.ReceivablePlan[agreementDraft, *rateagreement.RateAgreement]{
		Request: func(params *agentports.ToolExecuteParams) (agreementDraft, error) {
			return agreementDraft{params: params}, nil
		},
		Plan: func(
			ctx context.Context,
			draft agreementDraft,
			_ *agentports.ToolExecuteParams,
		) (*rateagreement.RateAgreement, error) {
			entity, err := draft.Build(ctx, lanes)
			if err != nil {
				return nil, err
			}

			return agreements.PlanCreate(ctx, entity)
		},
		Refused: func(agreementDraft) string {
			return "Would draft a rate agreement."
		},
		Render: func(_ agreementDraft, planned *rateagreement.RateAgreement) (*agent.ToolPreview, error) {
			record := agreementRecord(planned)
			record.Version = nil
			change, err := toolpreview.Create(
				record,
				agreementViewOf(planned),
				agreementOptions()...)
			if err != nil {
				return nil, err
			}

			return toolpreview.Build(fmt.Sprintf(
				"Would draft the %s rate agreement %s with %s from %s; it prices nothing until "+
					"it is approved.",
				strings.ToLower(string(planned.PartyType)), planned.Code,
				common.CountOf(len(planned.Rules), "lane"), common.DayLabel(planned.EffectiveFrom),
			), change), nil
		},
		Run: func(
			ctx context.Context,
			draft agreementDraft,
			params *agentports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			entity, err := draft.Build(ctx, lanes)
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

var errOnlyDraftsRevise = errors.New(
	"only a draft agreement is revised here; an agreement in review is rejected back to " +
		"draft first, and an active one's lanes change with rate_amend_agreement_rules",
)

func (r agreementRevision) Build(
	ctx context.Context,
	agreements RateAgreementKeeper,
	lanes laneReader,
	params *agentports.ToolExecuteParams,
) (*rateagreement.RateAgreement, error) {
	stored, err := loadAgreement(ctx, agreements, common.TenantFrom(*params), r.id)
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

func NewReviseRateAgreementDraftTool(
	agreements RateAgreementKeeper,
	states stateFinder,
) agentports.AgentTool {
	properties := agreementHeaderProperties()
	properties[paramRateAgreementID] = rateAgreementIDProperty()
	properties[paramLanes] = laneProperty(true)
	lanes := laneReader{states: states}

	return common.NewReportingReceivableTool(rateDraftSpec(&common.ReceivableSpec{
		Name: "rate_update_agreement_draft",
		Recipe: []string{
			"rate_list_agreements",
			"rate_get_agreement",
			"rate_update_agreement_draft",
		},
		Description: "Rewrite a draft rate agreement before it is submitted: its header " +
			"terms or its lanes. Fields left out keep their value. Only a draft is revised " +
			"here; an active agreement's lanes change with rate_amend_agreement_rules. Read it " +
			"with rate_get_agreement first for its lanes' ruleIds.",
		Operation:  permission.OpUpdate,
		Reversible: true,
		Rationale: "Changes a draft agreement inside Trenova that prices nothing yet; a later " +
			"revision changes it back.",
		Properties: properties,
		Required:   []string{paramRateAgreementID},
		Target:     targetRateAgreement,
	}), common.ReceivablePlan[agreementRevision, *rateagreementservice.UpdatePlan]{
		Request: func(params *agentports.ToolExecuteParams) (agreementRevision, error) {
			id, err := common.RequirePulid(params.Params, paramRateAgreementID)
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
				return agreementRevision{}, errors.New("name at least one term to revise")
			}

			return agreementRevision{id: id, values: values}, nil
		},
		Plan: func(
			ctx context.Context,
			revision agreementRevision,
			params *agentports.ToolExecuteParams,
		) (*rateagreementservice.UpdatePlan, error) {
			revised, err := revision.Build(ctx, agreements, lanes, params)
			if err != nil {
				return nil, err
			}

			return agreements.PlanUpdate(ctx, revised, params.Actor)
		},
		Refused: func(agreementRevision) string {
			return "Would revise a draft rate agreement."
		},
		Render: func(
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
				common.CountOf(len(plan.SupersededIDs), "lane"),
				common.CountOf(len(plan.Inserts), "lane"),
			), change), nil
		},
		Run: func(
			ctx context.Context,
			revision agreementRevision,
			params *agentports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			revised, err := revision.Build(ctx, agreements, lanes, params)
			if err != nil {
				return nil, err
			}
			updated, err := agreements.Update(ctx, revised, params.Actor)
			if err != nil {
				return nil, err
			}

			return agreementResult("revised", updated), nil
		},
	})
}

func NewDuplicateRateAgreementTool(agreements RateAgreementKeeper) agentports.AgentTool {
	return common.NewReportingReceivableTool(rateDraftSpec(&common.ReceivableSpec{
		Name: "rate_duplicate_agreement",
		Description: "Copy a rate agreement with its lanes, weight breaks, accessorials and " +
			"fuel terms into a new draft, to start a renewal. The copy has none of the " +
			"original's history or approvals; rate_update_agreement_draft then changes it.",
		Operation:  permission.OpDuplicate,
		Reversible: true,
		Rationale: "Saves a draft copy of an agreement inside Trenova that prices nothing " +
			"until a person approves it.",
		Properties: map[string]any{
			paramRateAgreementID: rateAgreementIDProperty(),
			paramAgreementCode: common.StringProperty(
				"The copy's code. Defaults to the original's "+
					"with -COPY.",
				maxAgreementCode,
			),
			paramAgreementName: common.StringProperty(
				"The copy's name. Defaults to the original's "+
					"with (Copy).",
				maxAgreementName,
			),
		},
		Required: []string{paramRateAgreementID},
		Target:   targetRateAgreement,
	}), common.ReceivablePlan[*rateagreementservice.DuplicateRateAgreementRequest, *rateagreementservice.DuplicatePlan]{
		Request: func(
			params *agentports.ToolExecuteParams,
		) (*rateagreementservice.DuplicateRateAgreementRequest, error) {
			id, err := common.RequirePulid(params.Params, paramRateAgreementID)
			if err != nil {
				return nil, err
			}
			code, err := common.BoundedText(params.Params, paramAgreementCode, maxAgreementCode)
			if err != nil {
				return nil, err
			}
			name, err := common.BoundedText(params.Params, paramAgreementName, maxAgreementName)
			if err != nil {
				return nil, err
			}

			return &rateagreementservice.DuplicateRateAgreementRequest{
				TenantInfo:      common.TenantFrom(*params),
				RateAgreementID: id,
				Code:            code,
				Name:            name,
			}, nil
		},
		Plan: func(
			ctx context.Context,
			req *rateagreementservice.DuplicateRateAgreementRequest,
			_ *agentports.ToolExecuteParams,
		) (*rateagreementservice.DuplicatePlan, error) {
			return agreements.PlanDuplicate(ctx, req)
		},
		Refused: func(*rateagreementservice.DuplicateRateAgreementRequest) string {
			return "Would copy a rate agreement into a new draft."
		},
		Render: func(
			_ *rateagreementservice.DuplicateRateAgreementRequest,
			plan *rateagreementservice.DuplicatePlan,
		) (*agent.ToolPreview, error) {
			record := agreementRecord(plan.Copy)
			record.Version = nil
			change, err := toolpreview.Create(
				record,
				agreementViewOf(plan.Copy),
				agreementOptions()...)
			if err != nil {
				return nil, err
			}

			return toolpreview.Build(fmt.Sprintf(
				"Would copy the rate agreement %s into the draft %s with %s.",
				plan.Original.Code, plan.Copy.Code, common.CountOf(len(plan.Copy.Rules), "lane"),
			), change), nil
		},
		Run: func(
			ctx context.Context,
			req *rateagreementservice.DuplicateRateAgreementRequest,
			params *agentports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			created, err := agreements.Duplicate(ctx, req, params.Actor.UserID)
			if err != nil {
				return nil, err
			}

			return agreementResult("duplicated", created), nil
		},
	})
}

type AgreementReviewStep struct {
	name        string
	recipe      []string
	review      rateagreementservice.Review
	operation   permission.Operation
	money       bool
	needsReason bool
	description string
	searchTerms []string
	rationale   string
	outcome     string
}

func (s *AgreementReviewStep) spec() *common.ReceivableSpec {
	properties := map[string]any{paramRateAgreementID: rateAgreementIDProperty()}
	required := []string{paramRateAgreementID}
	comment := "A note on the review for whoever reads the agreement next."
	if s.needsReason {
		comment = "Why, in a sentence the agreement's owner can act on."
		required = append(required, paramReviewComment)
	}
	properties[paramReviewComment] = common.StringProperty(comment, maxReviewComment)

	spec := &common.ReceivableSpec{
		Name:        s.name,
		Recipe:      s.recipe,
		Description: s.description,
		SearchTerms: s.searchTerms,
		Operation:   s.operation,
		Rationale:   s.rationale,
		Properties:  properties,
		Required:    required,
		Target:      targetRateAgreement,
	}
	if s.money {
		return rateMoneySpec(spec)
	}
	spec.Resource = permission.ResourceRateAgreement
	spec.MaxTier = agent.TierPropose

	return common.FuelInternalSpec(spec)
}

func (s *AgreementReviewStep) request(
	params *agentports.ToolExecuteParams,
) (*rateagreementservice.ApprovalActionRequest, error) {
	id, err := common.RequirePulid(params.Params, paramRateAgreementID)
	if err != nil {
		return nil, err
	}
	read := common.BoundedText
	if s.needsReason {
		read = common.RequireBoundedText
	}
	comment, err := read(params.Params, paramReviewComment, maxReviewComment)
	if err != nil {
		return nil, err
	}

	return &rateagreementservice.ApprovalActionRequest{
		TenantInfo: common.TenantFrom(*params),
		EntityID:   id,
		Comment:    comment,
	}, nil
}

func NewAgreementReviewTool(
	agreements RateAgreementKeeper,
	step *AgreementReviewStep,
) agentports.AgentTool {
	return common.NewReceivableTool(
		step.spec(),
		common.ReceivablePlan[*rateagreementservice.ApprovalActionRequest, *rateagreementservice.AgreementChange]{
			Request: step.request,
			Plan: func(
				ctx context.Context,
				req *rateagreementservice.ApprovalActionRequest,
				_ *agentports.ToolExecuteParams,
			) (*rateagreementservice.AgreementChange, error) {
				return agreements.PlanReview(ctx, step.review, req)
			},
			Refused: func(*rateagreementservice.ApprovalActionRequest) string {
				return fmt.Sprintf(
					"Would %s a rate agreement.",
					strings.ToLower(string(step.review)),
				)
			},
			Render: func(
				_ *rateagreementservice.ApprovalActionRequest,
				plan *rateagreementservice.AgreementChange,
			) (*agent.ToolPreview, error) {
				change, err := toolpreview.Changed(
					agreementRecord(plan.Before),
					agreementViewOf(plan.Before),
					agreementViewOf(plan.After),
					toolpreview.Only(common.FieldStatus, "reviewComment"),
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
			Run: func(
				ctx context.Context,
				req *rateagreementservice.ApprovalActionRequest,
				_ *agentports.ToolExecuteParams,
			) (*agent.ToolExecutionResult, error) {
				_, err := agreements.Review(ctx, step.review, req)

				return nil, err
			},
		},
	)
}

func AgreementReviewSteps() []*AgreementReviewStep {
	return []*AgreementReviewStep{
		{
			name: "rate_submit_agreement",
			recipe: []string{
				"rate_list_agreements",
				"rate_get_agreement",
				"rate_submit_agreement",
			},
			searchTerms: []string{"submit for approval", "send for approval"},
			review:      rateagreementservice.ReviewSubmit,
			operation:   permission.OpSubmit,
			description: "Submit a draft rate agreement for review. It prices nothing until a " +
				"person approves it. Read it with rate_get_agreement first.",
			rationale: "Moves a draft agreement into review inside Trenova; it prices nothing " +
				"until a person approves it.",
			outcome: "a reviewer approves or rejects it and it prices nothing until then",
		},
		{
			name:      "rate_approve_agreement",
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
			name:        "rate_reject_agreement",
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
			name:        "rate_suspend_agreement",
			review:      rateagreementservice.ReviewSuspend,
			operation:   permission.OpUpdate,
			money:       true,
			needsReason: true,
			description: "Propose suspending an active rate agreement, such as for a customer " +
				"on credit hold, with the reason. Shipments stop rating against it at once; " +
				"rate_resume_agreement puts it back. A person always decides.",
			rationale: "Stops an agreement pricing shipments, which changes what customers " +
				"are charged; only a person suspends one.",
			outcome: "shipments stop rating against it until it is resumed",
		},
		{
			name:      "rate_resume_agreement",
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
			name:      "rate_archive_agreement",
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
	agreements RateAgreementKeeper,
	lanes laneReader,
	params *agentports.ToolExecuteParams,
) (*rateagreementports.AmendRateAgreementRulesRequest, error) {
	agreement, err := loadAgreement(ctx, agreements, common.TenantFrom(*params), a.id)
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

	return &rateagreementports.AmendRateAgreementRulesRequest{
		TenantInfo:      common.TenantFrom(*params),
		RateAgreementID: agreement.ID,
		EffectiveFrom:   a.from,
		SupersededIDs:   a.superseded,
		Rules:           rules,
	}, nil
}

func NewAmendRateAgreementRulesTool(
	agreements RateAgreementKeeper,
	states stateFinder,
) agentports.AgentTool {
	lanes := laneReader{states: states}

	return common.NewReceivableTool(rateMoneySpec(&common.ReceivableSpec{
		Name: "rate_amend_agreement_rules",
		Description: "Propose changing an active rate agreement's lanes from a day: close out " +
			"the lanes it replaces and add their successors, keeping the old rates in history. " +
			"Take the lanes' ruleIds from rate_get_agreement. It changes what shipments are " +
			"charged from that day, so a person always decides.",
		Operation: permission.OpUpdate,
		Rationale: "Changes the rates an agreement prices shipments at; only a person amends " +
			"a live contract.",
		Properties: map[string]any{
			paramRateAgreementID: rateAgreementIDProperty(),
			common.ParamEffectiveFrom: agenttoolschema.Date(
				"The first day the new lanes price shipments.",
			),
			paramSupersededRuleIDs: agenttoolschema.KindIDs("The lanes this closes out, by ruleId "+
				"from rate_get_agreement.", maxAgreementLanes, permission.KindRateAgreementLane),
			paramLanes: laneProperty(false),
		},
		Required: []string{paramRateAgreementID, common.ParamEffectiveFrom},
		Target:   targetRateAgreement,
	}), common.ReceivablePlan[*ruleAmendment, *rateagreementservice.AmendmentPlan]{
		Request: func(params *agentports.ToolExecuteParams) (*ruleAmendment, error) {
			id, err := common.RequirePulid(params.Params, paramRateAgreementID)
			if err != nil {
				return nil, err
			}
			from, err := common.RequireDay(params.Params, common.ParamEffectiveFrom)
			if err != nil {
				return nil, err
			}
			amendment := &ruleAmendment{id: id, from: from, values: params.Params}
			if _, given := params.Params[paramSupersededRuleIDs]; given {
				if amendment.superseded, err = common.RequirePulidSlice(
					params.Params, paramSupersededRuleIDs, maxAgreementLanes,
				); err != nil {
					return nil, err
				}
			}

			return amendment, nil
		},
		Plan: func(
			ctx context.Context,
			amendment *ruleAmendment,
			params *agentports.ToolExecuteParams,
		) (*rateagreementservice.AmendmentPlan, error) {
			req, err := amendment.request(ctx, agreements, lanes, params)
			if err != nil {
				return nil, err
			}

			return agreements.PlanAmendRules(ctx, req)
		},
		Refused: func(*ruleAmendment) string {
			return "Would amend a rate agreement's lanes."
		},
		Render: renderRuleAmendment,
		Run: func(
			ctx context.Context,
			amendment *ruleAmendment,
			params *agentports.ToolExecuteParams,
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
		plan.Agreement.Code, common.DayLabel(plan.EffectiveFrom),
		common.CountOf(len(plan.Superseded), "lane"), common.CountOf(len(plan.Rules), "lane"),
	), change), nil
}

func rateIncreaseRequestFrom(
	params *agentports.ToolExecuteParams,
) (*rateagreementservice.RateIncreaseRequest, error) {
	req := &rateagreementservice.RateIncreaseRequest{TenantInfo: common.TenantFrom(*params)}
	var err error
	if req.EffectiveFrom, err = common.RequireDay(
		params.Params,
		common.ParamEffectiveFrom,
	); err != nil {
		return nil, err
	}
	if _, given := params.Params[paramAgreementIDs]; given {
		if req.AgreementIDs, err = common.RequirePulidSlice(
			params.Params, paramAgreementIDs, maxIncreaseScope,
		); err != nil {
			return nil, err
		}
	}
	if len(req.AgreementIDs) == 0 {
		if err = readRateIncreaseScope(params.Params, req); err != nil {
			return nil, err
		}
	}
	for key, target := range map[string]*decimal.NullDecimal{
		paramPercentChange: &req.Adjustment.PercentChange,
		paramFlatChange:    &req.Adjustment.FlatChange,
	} {
		value, present, decimalErr := common.OptionalDecimal(params.Params, key)
		if decimalErr != nil {
			return nil, decimalErr
		}
		*target = decimal.NullDecimal{Decimal: value, Valid: present}
	}

	return req, nil
}

// rateIncreaseScopes choose the agreements a rate increase moves when none
// are named by id.
var rateIncreaseScopes = []string{common.ParamCustomerID, common.ParamCarrierID, paramPartyType}

// readRateIncreaseScope reads the customer, carrier and party type that choose
// the agreements when no agreementIds are named. With agreementIds they are
// never read: a model that listed one customer's agreements sent the
// customerId back beside their ids, and a carrierId as well, and was refused
// for targeting a customer and a carrier at once, though the ids it named
// were all the call could move.
func readRateIncreaseScope(
	params map[string]any,
	req *rateagreementservice.RateIncreaseRequest,
) error {
	var err error
	if req.CustomerID, err = common.OptionalPulidParam(params, common.ParamCustomerID); err != nil {
		return err
	}
	if req.CarrierID, err = common.OptionalPulidParam(params, common.ParamCarrierID); err != nil {
		return err
	}
	party, given, err := common.OptionalEnum(params, paramPartyType, partyTypes.Values)
	if err != nil {
		return err
	}
	if given {
		req.PartyType = party
	}

	return nil
}

// rateIncreaseTool is rate_apply_increase, which files a call naming its
// agreements without the scopes it set aside, so the person approving reads
// only the agreements that move.
type RateIncreaseTool struct {
	*common.ReceivableTool[*rateagreementservice.RateIncreaseRequest, *rateagreementservice.RateIncreasePlan]
}

func (t RateIncreaseTool) ResolveSelection(
	_ context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolSelectionResolver interface passes params by value
) (map[string]any, error) {
	if _, named := params.Params[paramAgreementIDs]; !named {
		return params.Params, nil
	}

	return maputils.WithoutKeys(params.Params, rateIncreaseScopes...), nil
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
		samples = append(samples, common.CountOf(more, "more lane"))
	}
	summary := fmt.Sprintf(
		"Would move %s on %s by %s from %s: %s.",
		common.CountOf(len(plan.Lines), "lane rate"),
		common.CountOf(plan.AgreementCount, rateAgreementKind),
		rateAdjustmentLabel(req.Adjustment),
		common.DayLabel(plan.EffectiveFrom),
		strings.Join(samples, "; "),
	)
	if plan.SkippedNoRate > 0 {
		summary += fmt.Sprintf(" %s priced by a rate matrix are left as they are.",
			common.CountOf(plan.SkippedNoRate, "lane"))
	}

	return toolpreview.Build(summary), nil
}

func NewApplyRateIncreaseTool(agreements RateAgreementKeeper) agentports.AgentTool {
	return RateIncreaseTool{common.NewReceivableTool(rateMoneySpec(&common.ReceivableSpec{
		Name: "rate_apply_increase",
		Description: "Propose a general rate increase or decrease on agreements' lane rates, " +
			"by a percent or a flat amount, from a day. It moves every lane rate on the named " +
			"agreements, one customer's or carrier's, or every active agreement of a party " +
			"type. The old rates stay in history. A person always decides.",
		Operation: permission.OpUpdate,
		Rationale: "Changes the rates many agreements charge customers or pay carriers; " +
			"only a person applies a rate increase.",
		Properties: map[string]any{
			common.ParamEffectiveFrom: agenttoolschema.Date(
				"The first day the new rates price shipments.",
			),
			paramAgreementIDs: agenttoolschema.RecordIDs(
				permission.ResourceRateAgreement,
				"The agreements to move, from "+
					"rate_list_agreements. When given, customerId, carrierId and partyType "+
					"are ignored.",
				maxIncreaseScope,
			),
			common.ParamCustomerID: agenttoolschema.RecordIDText(permission.ResourceCustomer,
				"Every active agreement with this customer, from customer_list. Ignored "+
					"when agreementIds are named."),
			common.ParamCarrierID: agenttoolschema.RecordIDText(permission.ResourceCarrier,
				"Every active agreement with this carrier, from carrier_list. Ignored "+
					"when agreementIds are named."),
			paramPartyType: agenttoolschema.Enum("Every active agreement of this party type, "+
				"when no agreements, customer or carrier is named.", partyTypes),
			paramPercentChange: common.StringProperty("The change in percent, such as 3.5 or -2. "+
				"Give this or flatChange.", 0),
			paramFlatChange: common.StringProperty(
				"The change per rating unit, such as 0.10. Give "+
					"this or percentChange.",
				0,
			),
		},
		Required:    []string{common.ParamEffectiveFrom},
		SearchTerms: []string{"gri", "general rate increase", "raise rates", "price increase"},
	}), common.ReceivablePlan[*rateagreementservice.RateIncreaseRequest, *rateagreementservice.RateIncreasePlan]{
		Request: rateIncreaseRequestFrom,
		Plan: func(
			ctx context.Context,
			req *rateagreementservice.RateIncreaseRequest,
			_ *agentports.ToolExecuteParams,
		) (*rateagreementservice.RateIncreasePlan, error) {
			return agreements.PlanApplyRateIncrease(ctx, req)
		},
		Refused: func(*rateagreementservice.RateIncreaseRequest) string {
			return "Would apply a general rate increase."
		},
		Render: renderRateIncrease,
		Run: func(
			ctx context.Context,
			req *rateagreementservice.RateIncreaseRequest,
			params *agentports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			_, err := agreements.ApplyRateIncrease(ctx, req, params.Actor.UserID)

			return nil, err
		},
	})}
}
