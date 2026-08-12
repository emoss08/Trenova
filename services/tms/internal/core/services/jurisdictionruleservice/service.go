package jurisdictionruleservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/jurisdictionrule"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ServiceParams struct {
	fx.In

	Repo         repositories.JurisdictionRuleRepository
	AuditService services.AuditService
	Logger       *zap.Logger
}

type service struct {
	repo         repositories.JurisdictionRuleRepository
	auditService services.AuditService
	l            *zap.Logger
}

func NewService(p ServiceParams) services.JurisdictionRuleService {
	return &service{
		repo:         p.Repo,
		auditService: p.AuditService,
		l:            p.Logger.Named("service.jurisdiction-rule"),
	}
}

func (s *service) List(
	ctx context.Context,
	req *services.ListJurisdictionRulesRequest,
) (*pagination.ListResult[*jurisdictionrule.JurisdictionRule], error) {
	return s.repo.List(ctx, &repositories.ListJurisdictionRulesRequest{
		Filter:            req.Filter,
		VerificationState: req.VerificationState,
		Status:            req.Status,
	})
}

func (s *service) GetByID(
	ctx context.Context,
	ruleID pulid.ID,
) (*jurisdictionrule.JurisdictionRule, error) {
	return s.repo.GetByID(ctx, &repositories.GetJurisdictionRuleByIDRequest{
		RuleID:           ruleID,
		ExpandThresholds: true,
	})
}

func (s *service) Create(
	ctx context.Context,
	entity *jurisdictionrule.JurisdictionRule,
	actor *services.RequestActor,
) (*jurisdictionrule.JurisdictionRule, error) {
	// A new row has not been checked against anything yet, whatever the caller
	// claims. Verification is earned through Verify, which records who did it
	// and when.
	entity.VerificationState = jurisdictionrule.VerificationUnverified
	entity.VerifiedAt = nil

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	created, err := s.repo.Create(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.logAction(&auditParams{
		Rule:      created,
		Operation: permission.OpCreate,
		Actor:     actor,
		Current:   created,
		Comment:   "Jurisdiction rule created",
	})

	return created, nil
}

func (s *service) Update(
	ctx context.Context,
	entity *jurisdictionrule.JurisdictionRule,
	actor *services.RequestActor,
) (*jurisdictionrule.JurisdictionRule, error) {
	previous, err := s.repo.GetByID(ctx, &repositories.GetJurisdictionRuleByIDRequest{
		RuleID: entity.ID,
	})
	if err != nil {
		return nil, err
	}

	// Changing what the row claims is legal invalidates any prior confirmation.
	// Keeping the Verified badge across an edit would let someone confirm a
	// conservative row and then quietly raise the limits under that badge, which
	// is precisely the state an operator trusts when deciding not to permit.
	if limitsDiffer(previous, entity) {
		entity.VerificationState = jurisdictionrule.VerificationUnverified
		entity.VerifiedAt = nil
	} else {
		entity.VerificationState = previous.VerificationState
		entity.VerifiedAt = previous.VerifiedAt
	}

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	updated, err := s.repo.Update(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.logAction(&auditParams{
		Rule:      updated,
		Operation: permission.OpUpdate,
		Actor:     actor,
		Previous:  previous,
		Current:   updated,
		Comment:   "Jurisdiction rule updated",
	})

	return updated, nil
}

// limitsDiffer reports whether an edit changes anything a verification was an
// assertion about. Presentation-only fields — the source note and URL — are
// excluded, so correcting a citation does not force a re-check.
func limitsDiffer(previous, next *jurisdictionrule.JurisdictionRule) bool {
	if previous == nil || next == nil {
		return true
	}

	return previous.MaxWidthFeet != next.MaxWidthFeet ||
		previous.MaxHeightFeet != next.MaxHeightFeet ||
		previous.MaxLengthFeet != next.MaxLengthFeet ||
		previous.MaxWeightPounds != next.MaxWeightPounds ||
		previous.PermitLeadTimeDays != next.PermitLeadTimeDays ||
		previous.PermitValidityDays != next.PermitValidityDays ||
		previous.DaylightOnly != next.DaylightOnly ||
		previous.RushHourRestricted != next.RushHourRestricted ||
		previous.WeekendRestricted != next.WeekendRestricted ||
		previous.HolidayRestricted != next.HolidayRestricted ||
		nullableFloatDiffers(previous.SuperloadWidthFeet, next.SuperloadWidthFeet) ||
		nullableIntDiffers(previous.SuperloadWeightPounds, next.SuperloadWeightPounds)
}

func nullableFloatDiffers(a, b *float64) bool {
	if a == nil || b == nil {
		return (a == nil) != (b == nil)
	}
	return *a != *b
}

func nullableIntDiffers(a, b *int64) bool {
	if a == nil || b == nil {
		return (a == nil) != (b == nil)
	}
	return *a != *b
}

func (s *service) Verify(
	ctx context.Context,
	req *services.VerifyJurisdictionRuleRequest,
) (*jurisdictionrule.JurisdictionRule, error) {
	multiErr := errortypes.NewMultiError()

	switch req.State {
	case jurisdictionrule.VerificationVerified, jurisdictionrule.VerificationDisputed:
	case jurisdictionrule.VerificationUnverified:
		multiErr.Add("verificationState", errortypes.ErrInvalid,
			"Use verified or disputed; a row cannot be actively marked unverified")
	default:
		multiErr.Add("verificationState", errortypes.ErrInvalid,
			"Verification state is invalid")
	}

	// A confirmation nobody can trace is not a confirmation. The note is what a
	// later reader follows to see what this was checked against.
	if len(req.SourceNote) < jurisdictionrule.MinSourceNoteLength {
		multiErr.Add("sourceNote", errortypes.ErrInvalid,
			"Record what this was checked against, at least 10 characters")
	}

	if multiErr.HasErrors() {
		return nil, multiErr
	}

	verified, err := s.repo.Verify(ctx, &repositories.VerifyJurisdictionRuleRequest{
		RuleID:     req.RuleID,
		State:      req.State,
		SourceNote: req.SourceNote,
		SourceURL:  req.SourceURL,
		VerifiedAt: timeutils.NowUnix(),
	})
	if err != nil {
		return nil, err
	}

	s.logAction(&auditParams{
		Rule:      verified,
		Operation: permission.OpApprove,
		Actor:     req.Actor,
		Current:   verified,
		Comment:   "Jurisdiction rule marked " + string(req.State) + ": " + req.SourceNote,
	})

	return verified, nil
}

type auditParams struct {
	Rule      *jurisdictionrule.JurisdictionRule
	Operation permission.Operation
	Actor     *services.RequestActor
	Previous  any
	Current   any
	Comment   string
}

// logAction records a change to shared reference data.
//
// Critical because the blast radius is every organization on the platform: a
// row that silently gained four feet of legal width only surfaces when a load
// is stopped, and the audit trail is what makes it answerable.
//
// The organization and business unit are deliberately left zero — this table has
// no tenant columns, and stamping the acting user's tenant onto a global change
// would imply it was scoped to them.
func (s *service) logAction(params *auditParams) {
	if params.Rule == nil {
		return
	}

	auditservice.Record(s.auditService, s.l, &auditservice.RecordParams{
		Resource:   permission.ResourceJurisdictionRule,
		ResourceID: params.Rule.ID.String(),
		Operation:  params.Operation,
		Actor:      params.Actor.AuditActorOrSystem(),
		Critical:   true,
		Previous:   params.Previous,
		Current:    params.Current,
		Comment:    params.Comment,
		Metadata: map[string]any{
			"stateId": params.Rule.StateID.String(),
			"scope":   "global",
		},
	})
}
