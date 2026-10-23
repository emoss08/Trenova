package rateagreementservice

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateagreementports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// A general rate increase is the amendment primitive applied in bulk: every
// affected rule is closed out and succeeded by a copy at the new rate,
// effective from the announced date. Nothing is edited in place, so "what did
// this lane cost before the GRI" stays answerable forever — which is the whole
// reason carriers announce GRIs with effective dates instead of quietly
// repricing.

const ratePrecision = 6

type RateAdjustment struct {
	PercentChange decimal.NullDecimal `json:"percentChange"`
}

func (ra RateAdjustment) Validate() *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	switch {
	case !ra.PercentChange.Valid:
		multiErr.Add(
			"adjustment.percentChange",
			errortypes.ErrRequired,
			"A percent change is required",
		)
	case ra.PercentChange.Decimal.LessThanOrEqual(decimal.NewFromInt(-100)):
		multiErr.Add(
			"adjustment.percentChange",
			errortypes.ErrInvalid,
			"A decrease of 100% or more would invert every rate it touches",
		)
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (ra RateAdjustment) factor() decimal.Decimal {
	return decimal.NewFromInt(1).Add(ra.PercentChange.Decimal.Div(decimal.NewFromInt(100)))
}

func roundRate(value decimal.Decimal) decimal.Decimal {
	return value.Round(ratePrecision)
}

func (ra RateAdjustment) applyTo(inputs tarifftypes.Inputs) (tarifftypes.Inputs, error) {
	return inputs.ScaleMoney(ra.factor(), roundRate)
}

// RateIncreaseRequest names the agreements a GRI touches and how far it moves
// them. Scope is the widest filter that matches: explicit agreements win, then
// a single customer or carrier, then every active agreement of the party type.
type RateIncreaseRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`

	AgreementIDs []pulid.ID              `json:"agreementIds"`
	CustomerID   *pulid.ID               `json:"customerId"`
	CarrierID    *pulid.ID               `json:"carrierId"`
	PartyType    rateagreement.PartyType `json:"partyType"`

	Adjustment    RateAdjustment `json:"adjustment"`
	EffectiveFrom int64          `json:"effectiveFrom"`
}

func (req *RateIncreaseRequest) Validate() *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	if req == nil {
		multiErr.Add("request", errortypes.ErrRequired, "A rate increase request is required")
		return multiErr
	}

	if adjErr := req.Adjustment.Validate(); adjErr != nil {
		return adjErr
	}

	if req.EffectiveFrom == 0 {
		multiErr.Add(
			"effectiveFrom",
			errortypes.ErrRequired,
			"The day the new rates take effect is required",
		)
	}

	if req.CustomerID != nil && req.CarrierID != nil {
		multiErr.Add(
			"customerId",
			errortypes.ErrInvalid,
			"A rate increase targets a customer or a carrier, not both",
		)
	}

	if len(req.AgreementIDs) == 0 && req.CustomerID == nil && req.CarrierID == nil &&
		!req.PartyType.IsValid() {
		multiErr.Add(
			"partyType",
			errortypes.ErrRequired,
			"Say which agreements this touches: a selection, a customer, a carrier, or a whole party type",
		)
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

// RateIncreaseLine is one lane's before and after, which is what somebody
// reviews before applying a GRI.
type RateIncreaseLine struct {
	RateAgreementID pulid.ID           `json:"rateAgreementId"`
	AgreementCode   string             `json:"agreementCode"`
	AgreementName   string             `json:"agreementName"`
	RuleID          pulid.ID           `json:"ruleId"`
	LaneKey         string             `json:"laneKey"`
	Label           string             `json:"label"`
	Before          tarifftypes.Inputs `json:"before"`
	After           tarifftypes.Inputs `json:"after"`
	Changed         []string           `json:"changed"`
}

// RateIncreasePlan is everything a GRI would do, stated before it does it.
type RateIncreasePlan struct {
	EffectiveFrom  int64               `json:"effectiveFrom"`
	AgreementCount int                 `json:"agreementCount"`
	Lines          []*RateIncreaseLine `json:"lines"`

	SkippedNoMoney int `json:"skippedNoMoney"`
}

// agreementIncreasePlan is one agreement's slice of the GRI: what to show, what
// to close out, and the successors to insert.
type agreementIncreasePlan struct {
	Lines         []*RateIncreaseLine
	Clones        []*rateagreement.RateAgreementRule
	SupersededIDs  []pulid.ID
	SkippedNoMoney int
	AmendAt        int64
}

func planAgreementIncrease(
	agreement *rateagreement.RateAgreement,
	rules []*rateagreement.RateAgreementRule,
	adjustment RateAdjustment,
	amendAt int64,
) (agreementIncreasePlan, error) {
	plan := agreementIncreasePlan{
		Lines:         make([]*RateIncreaseLine, 0, len(rules)),
		Clones:        make([]*rateagreement.RateAgreementRule, 0, len(rules)),
		SupersededIDs: make([]pulid.ID, 0, len(rules)),
		AmendAt:       amendAt,
	}

	for _, rule := range rules {
		if rule == nil {
			continue
		}

		after, err := adjustment.applyTo(rule.Inputs)
		if err != nil {
			return plan, err
		}
		changed := changedInputs(rule.Inputs, after)
		if len(changed) == 0 {
			plan.SkippedNoMoney++
			continue
		}

		plan.Lines = append(plan.Lines, &RateIncreaseLine{
			RateAgreementID: agreement.ID,
			AgreementCode:   agreement.Code,
			AgreementName:   agreement.Name,
			RuleID:          rule.ID,
			LaneKey:         rule.LaneKey,
			Label:           rule.Label,
			Before:          rule.Inputs,
			After:           after,
			Changed:         changed,
		})

		plan.Clones = append(plan.Clones, cloneRuleWithInputs(rule, after, amendAt))
		plan.SupersededIDs = append(plan.SupersededIDs, rule.ID)
	}

	return plan, nil
}

func changedInputs(before, after tarifftypes.Inputs) []string {
	changed := make([]string, 0, len(after))
	for name, value := range after {
		if !(tarifftypes.Inputs{name: value}).Equal(tarifftypes.Inputs{name: before[name]}) {
			changed = append(changed, name)
		}
	}
	slices.Sort(changed)
	return changed
}

func cloneRuleWithInputs(
	rule *rateagreement.RateAgreementRule,
	inputs tarifftypes.Inputs,
	amendAt int64,
) *rateagreement.RateAgreementRule {
	successor := *rule
	supersededID := rule.ID

	successor.ID = pulid.Nil
	successor.SupersedesRuleID = &supersededID
	successor.EffectiveFrom = amendAt
	successor.EffectiveTo = nil
	successor.Version = 0
	successor.CreatedAt = 0
	successor.UpdatedAt = 0
	successor.Inputs = inputs

	return &successor
}

// PlanRateIncrease answers what a GRI would do without doing any of it.
func (s *Service) PlanRateIncrease(
	ctx context.Context,
	req *RateIncreaseRequest,
) (*RateIncreasePlan, error) {
	plan, _, err := s.planRateIncrease(ctx, req)

	return plan, err
}

// ApplyRateIncrease re-plans server side — the preview somebody read may be
// minutes old — and applies each agreement's amendment through the same
// close-out-and-insert primitive every other rule change uses.
func (s *Service) ApplyRateIncrease(
	ctx context.Context,
	req *RateIncreaseRequest,
	userID pulid.ID,
) (*RateIncreasePlan, error) {
	log := s.l.With(
		zap.String("operation", "ApplyRateIncrease"),
		zap.String("userId", userID.String()),
	)

	plan, agreementPlans, err := s.planRateIncrease(ctx, req)
	if err != nil {
		return nil, err
	}

	if err = rateIncreaseApplicable(plan); err != nil {
		return nil, err
	}

	for _, slice := range agreementPlans {
		if len(slice.plan.Clones) == 0 {
			continue
		}

		if err = s.repo.AmendRules(ctx, &rateagreementports.AmendRateAgreementRulesRequest{
			TenantInfo:      req.TenantInfo,
			RateAgreementID: slice.agreement.ID,
			EffectiveFrom:   slice.plan.AmendAt,
			SupersededIDs:   slice.plan.SupersededIDs,
			Rules:           slice.plan.Clones,
		}); err != nil {
			log.Error("failed to apply rate increase to an agreement",
				zap.String("agreementId", slice.agreement.ID.String()),
				zap.Error(err),
			)
			return nil, err
		}

		s.audit(
			log, slice.agreement, nil, permission.OpUpdate, userID,
			"General rate increase applied to "+slice.agreement.Code,
		)
	}

	return plan, nil
}

// agreementSlice pairs an agreement with its share of the plan, so apply can
// amend each one without re-deriving anything.
type agreementSlice struct {
	agreement *rateagreement.RateAgreement
	plan      agreementIncreasePlan
}

func (s *Service) planRateIncrease(
	ctx context.Context,
	req *RateIncreaseRequest,
) (*RateIncreasePlan, []agreementSlice, error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, nil, multiErr
	}

	agreements, err := s.scopedAgreements(ctx, req)
	if err != nil {
		return nil, nil, err
	}

	plan := &RateIncreasePlan{
		EffectiveFrom: req.EffectiveFrom,
		Lines:         make([]*RateIncreaseLine, 0),
	}
	agreementPlans := make([]agreementSlice, 0, len(agreements))

	now := timeutils.NowUnix()

	for _, agreement := range agreements {
		// The amendment moment can be neither in the past — those shipments
		// have already been rated — nor before the contract itself began.
		amendAt := max(req.EffectiveFrom, agreement.EffectiveFrom, now)

		rules, listErr := s.repo.ListRules(ctx, &rateagreementports.ListRateAgreementRulesRequest{
			TenantInfo:      req.TenantInfo,
			RateAgreementID: agreement.ID,
			AsOf:            amendAt,
		})
		if listErr != nil {
			return nil, nil, listErr
		}

		slice, planErr := planAgreementIncrease(agreement, rules, req.Adjustment, amendAt)
		if planErr != nil {
			return nil, nil, planErr
		}

		if len(slice.Lines) == 0 && slice.SkippedNoMoney == 0 {
			continue
		}

		plan.AgreementCount++
		plan.Lines = append(plan.Lines, slice.Lines...)
		plan.SkippedNoMoney += slice.SkippedNoMoney

		agreementPlans = append(agreementPlans, agreementSlice{agreement: agreement, plan: slice})
	}

	return plan, agreementPlans, nil
}

// scopedAgreements resolves the request's scope to the active agreements it
// names. Only active agreements take a GRI: a draft is edited directly, and a
// suspended contract has been deliberately taken out of pricing.
func (s *Service) scopedAgreements(
	ctx context.Context,
	req *RateIncreaseRequest,
) ([]*rateagreement.RateAgreement, error) {
	if len(req.AgreementIDs) > 0 {
		agreements := make([]*rateagreement.RateAgreement, 0, len(req.AgreementIDs))

		for _, agreementID := range req.AgreementIDs {
			agreement, err := s.repo.GetByID(ctx, &rateagreementports.GetRateAgreementByIDRequest{
				RateAgreementID: agreementID,
				TenantInfo:      req.TenantInfo,
			})
			if err != nil {
				return nil, err
			}

			if agreement.Status == rateagreement.StatusActive {
				agreements = append(agreements, agreement)
			}
		}

		return agreements, nil
	}

	return s.listActiveAgreements(ctx, req)
}

func (s *Service) listActiveAgreements(
	ctx context.Context,
	req *RateIncreaseRequest,
) ([]*rateagreement.RateAgreement, error) {
	const pageSize = 100

	agreements := make([]*rateagreement.RateAgreement, 0)

	for offset := 0; ; offset += pageSize {
		page, err := s.repo.List(ctx, &rateagreementports.ListRateAgreementRequest{
			Filter: &pagination.QueryOptions{
				TenantInfo: req.TenantInfo,
				Pagination: pagination.Info{Limit: pageSize, Offset: offset},
			},
			PartyType:  req.PartyType,
			Status:     rateagreement.StatusActive,
			CustomerID: req.CustomerID,
			CarrierID:  req.CarrierID,
		})
		if err != nil {
			return nil, err
		}

		agreements = append(agreements, page.Items...)

		if len(page.Items) < pageSize {
			break
		}
	}

	return agreements, nil
}
