package rateagreementservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

type Review string

const (
	ReviewSubmit  = Review("Submit")
	ReviewApprove = Review("Approve")
	ReviewReject  = Review("Reject")
	ReviewSuspend = Review("Suspend")
	ReviewResume  = Review("Resume")
	ReviewArchive = Review("Archive")
)

type AgreementChange struct {
	Before *rateagreement.RateAgreement
	After  *rateagreement.RateAgreement
}

type UpdatePlan struct {
	Before        *rateagreement.RateAgreement
	After         *rateagreement.RateAgreement
	AmendAt       int64
	SupersededIDs []pulid.ID
	Inserts       []*rateagreement.RateAgreementRule
	TermsChanged  bool
	ChangeSummary map[string]jsonutils.FieldChange
}

type DuplicatePlan struct {
	Original *rateagreement.RateAgreement
	Copy     *rateagreement.RateAgreement
}

type AmendmentPlan struct {
	Agreement     *rateagreement.RateAgreement
	EffectiveFrom int64
	Superseded    []*rateagreement.RateAgreementRule
	Rules         []*rateagreement.RateAgreementRule
}

func (s *Service) transitionFor(
	ctx context.Context,
	review Review,
	req *ApprovalActionRequest,
) (agreementTransition, error) {
	switch review {
	case ReviewSubmit:
		return submitTransition(), nil
	case ReviewApprove:
		return approveTransition(), nil
	case ReviewReject:
		return rejectTransition(req)
	case ReviewSuspend:
		return suspendTransition(req)
	case ReviewResume:
		return resumeTransition(), nil
	case ReviewArchive:
		return s.archiveTransition(ctx, req)
	default:
		return agreementTransition{}, errortypes.NewValidationError(
			"review",
			errortypes.ErrInvalid,
			"{0} is not a step in a rate agreement's review", string(review),
		)
	}
}

// PlanReview works out one review step without saving it.
func (s *Service) PlanReview(
	ctx context.Context,
	review Review,
	req *ApprovalActionRequest,
) (*AgreementChange, error) {
	transition, err := s.transitionFor(ctx, review, req)
	if err != nil {
		return nil, err
	}

	change, err := s.approvals().Plan(ctx, req, transition)
	if err != nil {
		return nil, err
	}

	return &AgreementChange{Before: change.Before, After: change.After}, nil
}

// Review runs one review step, the one PlanReview works out.
func (s *Service) Review(
	ctx context.Context,
	review Review,
	req *ApprovalActionRequest,
) (*rateagreement.RateAgreement, error) {
	transition, err := s.transitionFor(ctx, review, req)
	if err != nil {
		return nil, err
	}

	return s.approvals().Apply(ctx, req, transition)
}

// PlanCreate is Create without the save: the agreement as it would be stored,
// in Draft whatever the payload says, checked by the same validator.
func (s *Service) PlanCreate(
	ctx context.Context,
	entity *rateagreement.RateAgreement,
) (*rateagreement.RateAgreement, error) {
	planned := *entity
	planned.Status = rateagreement.StatusDraft
	planned.CurrentVersionNumber = 1
	planned.StampRules()

	if multiErr := s.validator.ValidateCreate(ctx, &planned); multiErr != nil {
		return nil, multiErr
	}

	return &planned, nil
}

// PlanUpdate is Update without the save: the header as it would be stored and
// the lanes the save would supersede and insert.
func (s *Service) PlanUpdate(
	ctx context.Context,
	entity *rateagreement.RateAgreement,
) (*UpdatePlan, error) {
	original, err := s.repo.GetByID(ctx, &repositories.GetRateAgreementByIDRequest{
		RateAgreementID: entity.ID,
		TenantInfo:      tenantOf(entity),
		IncludeChildren: true,
	})
	if err != nil {
		return nil, err
	}

	planned := *entity
	planned.Status = original.Status
	planned.StampRules()

	changeSummary, termsChanged := headerTermsChanged(original, &planned)
	planned.CurrentVersionNumber = original.CurrentVersionNumber
	if termsChanged {
		planned.CurrentVersionNumber = original.CurrentVersionNumber + 1
	}

	if multiErr := s.validator.ValidateUpdate(ctx, &planned); multiErr != nil {
		return nil, multiErr
	}

	amendAt := max(timeutils.NowUnix(), original.EffectiveFrom)
	amendment, planErr := planRuleAmendment(original.Rules, planned.Rules, amendAt)
	if planErr != nil {
		return nil, planErr
	}

	plan := &UpdatePlan{
		Before:        original,
		After:         &planned,
		AmendAt:       amendAt,
		TermsChanged:  termsChanged,
		ChangeSummary: changeSummary,
	}
	if amendment != nil {
		plan.SupersededIDs = amendment.SupersededIDs
		plan.Inserts = amendment.Inserts
	}

	return plan, nil
}

// PlanDuplicate is Duplicate without the save: the copy as Create would store
// it, checked by the same validator.
func (s *Service) PlanDuplicate(
	ctx context.Context,
	req *DuplicateRateAgreementRequest,
) (*DuplicatePlan, error) {
	original, err := s.repo.GetByID(ctx, &repositories.GetRateAgreementByIDRequest{
		RateAgreementID: req.RateAgreementID,
		TenantInfo:      req.TenantInfo,
		IncludeChildren: true,
	})
	if err != nil {
		return nil, err
	}

	planned, err := s.PlanCreate(ctx, copyAgreement(original, req))
	if err != nil {
		return nil, err
	}

	return &DuplicatePlan{Original: original, Copy: planned}, nil
}

// PlanAmendRules checks a rule amendment as AmendRules does, and names the
// lanes it would close out, without writing anything.
func (s *Service) PlanAmendRules(
	ctx context.Context,
	req *repositories.AmendRateAgreementRulesRequest,
) (*AmendmentPlan, error) {
	agreement, err := s.repo.GetByID(ctx, &repositories.GetRateAgreementByIDRequest{
		RateAgreementID: req.RateAgreementID,
		TenantInfo:      req.TenantInfo,
		IncludeChildren: true,
	})
	if err != nil {
		return nil, err
	}

	if multiErr := s.validator.ValidateAmendment(ctx, agreement, req); multiErr != nil {
		return nil, multiErr
	}

	superseded := make(map[pulid.ID]struct{}, len(req.SupersededIDs))
	for _, id := range req.SupersededIDs {
		superseded[id] = struct{}{}
	}
	plan := &AmendmentPlan{
		Agreement:     agreement,
		EffectiveFrom: req.EffectiveFrom,
		Superseded:    make([]*rateagreement.RateAgreementRule, 0, len(req.SupersededIDs)),
		Rules:         req.Rules,
	}
	for _, rule := range agreement.Rules {
		if rule == nil {
			continue
		}
		if _, named := superseded[rule.ID]; named {
			plan.Superseded = append(plan.Superseded, rule)
		}
	}

	return plan, nil
}

// PlanApplyRateIncrease is ApplyRateIncrease without the writes: the plan it
// would apply, refused for the reasons it would refuse it.
func (s *Service) PlanApplyRateIncrease(
	ctx context.Context,
	req *RateIncreaseRequest,
) (*RateIncreasePlan, error) {
	plan, err := s.PlanRateIncrease(ctx, req)
	if err != nil {
		return nil, err
	}
	if err = rateIncreaseApplicable(plan); err != nil {
		return nil, err
	}

	return plan, nil
}

func rateIncreaseApplicable(plan *RateIncreasePlan) error {
	if plan.NegativeCount > 0 {
		return errortypes.NewBusinessError(
			"This decrease would push some lanes below zero, and a negative rate is not a discount. Narrow the scope or soften the change.",
		)
	}

	if len(plan.Lines) == 0 {
		return errortypes.NewBusinessError(
			"No lane in scope carries a rate this change could move",
		)
	}

	return nil
}
