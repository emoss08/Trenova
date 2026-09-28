package rateagreementservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	Repo         repositories.RateAgreementRepository
	Validator    *Validator
	AuditService services.AuditService
}

type Service struct {
	l            *zap.Logger
	repo         repositories.RateAgreementRepository
	validator    *Validator
	auditService services.AuditService
}

func New(p Params) *Service {
	return &Service{
		l:            p.Logger.Named("service.rate-agreement"),
		repo:         p.Repo,
		validator:    p.Validator,
		auditService: p.AuditService,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListRateAgreementRequest,
) (*pagination.ListResult[*rateagreement.RateAgreement], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) ListConnection(
	ctx context.Context,
	req *repositories.ListRateAgreementConnectionRequest,
) (*pagination.CursorListResult[*rateagreement.RateAgreement], error) {
	return s.repo.ListConnection(ctx, req)
}

func (s *Service) SelectOptions(
	ctx context.Context,
	req *pagination.SelectQueryRequest,
) (*pagination.ListResult[*rateagreement.RateAgreement], error) {
	return s.repo.SelectOptions(ctx, req)
}

func (s *Service) GetByID(
	ctx context.Context,
	req *repositories.GetRateAgreementByIDRequest,
) (*rateagreement.RateAgreement, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) Create(
	ctx context.Context,
	entity *rateagreement.RateAgreement,
	userID pulid.ID,
) (*rateagreement.RateAgreement, error) {
	log := s.l.With(
		zap.String("operation", "Create"),
		zap.String("userId", userID.String()),
	)

	// A new agreement starts in Draft whatever the payload says. Creating one
	// already active would route around the review the organization asked for.
	planned, err := s.PlanCreate(ctx, entity)
	if err != nil {
		return nil, err
	}

	created, err := s.repo.Create(ctx, planned)
	if err != nil {
		log.Error("failed to create rate agreement", zap.Error(err))
		return nil, err
	}

	s.recordVersion(
		ctx, log, created,
		created.EffectiveFrom, userID,
		"Initial terms", nil,
	)

	s.audit(log, created, nil, permission.OpCreate, userID, "Rate agreement created")

	return created, nil
}

func (s *Service) Update(
	ctx context.Context,
	entity *rateagreement.RateAgreement,
	userID pulid.ID,
) (*rateagreement.RateAgreement, error) {
	log := s.l.With(
		zap.String("operation", "Update"),
		zap.String("userId", userID.String()),
	)

	plan, err := s.PlanUpdate(ctx, entity)
	if err != nil {
		return nil, err
	}

	if _, err = s.repo.Update(ctx, plan.After); err != nil {
		log.Error("failed to update rate agreement", zap.Error(err))
		return nil, err
	}

	// A save's lane edits become an amendment: changed lanes are superseded by
	// successors effective from this moment, dropped lanes are closed out, and
	// restated lanes are left untouched.
	if len(plan.SupersededIDs) > 0 || len(plan.Inserts) > 0 {
		if err = s.repo.AmendRules(ctx, &repositories.AmendRateAgreementRulesRequest{
			TenantInfo:      tenantOf(plan.After),
			RateAgreementID: plan.After.ID,
			EffectiveFrom:   plan.AmendAt,
			SupersededIDs:   plan.SupersededIDs,
			Rules:           plan.Inserts,
		}); err != nil {
			log.Error("failed to amend rate agreement rules on save", zap.Error(err))
			return nil, err
		}
	}

	if plan.TermsChanged {
		s.recordVersion(ctx, log, plan.After, plan.AmendAt, userID, "", plan.ChangeSummary)
	}

	// The caller gets what the database now holds rather than an echo of what
	// it sent: rules with their minted identities, and none of the rows the
	// amendment closed out.
	updated, err := s.repo.GetByID(ctx, &repositories.GetRateAgreementByIDRequest{
		RateAgreementID: entity.ID,
		TenantInfo:      tenantOf(entity),
		IncludeChildren: true,
	})
	if err != nil {
		log.Error("failed to reload rate agreement after save", zap.Error(err))
		return nil, err
	}

	s.audit(log, updated, plan.Before, permission.OpUpdate, userID, "Rate agreement updated")

	return updated, nil
}

// AmendRules closes out the rules a change replaces and inserts their
// successors, which is how every rule change happens — a person editing a lane,
// a general rate increase, an imported rate sheet, or a simulation made real.
//
// Nothing is edited in place. The superseded rows keep their history and simply
// stop being effective at the moment their replacements begin, which is what
// makes "what did this lane cost on March 4th" answerable a year later.
func (s *Service) AmendRules(
	ctx context.Context,
	req *repositories.AmendRateAgreementRulesRequest,
	userID pulid.ID,
) (*rateagreement.RateAgreement, error) {
	log := s.l.With(
		zap.String("operation", "AmendRules"),
		zap.String("agreementId", req.RateAgreementID.String()),
	)

	plan, err := s.PlanAmendRules(ctx, req)
	if err != nil {
		return nil, err
	}
	agreement := plan.Agreement

	if err = s.repo.AmendRules(ctx, req); err != nil {
		log.Error("failed to amend rate agreement rules", zap.Error(err))
		return nil, err
	}

	amended, err := s.repo.GetByID(ctx, &repositories.GetRateAgreementByIDRequest{
		RateAgreementID: req.RateAgreementID,
		TenantInfo:      req.TenantInfo,
		IncludeChildren: true,
	})
	if err != nil {
		log.Error("failed to reload amended rate agreement", zap.Error(err))
		return nil, err
	}

	s.audit(log, amended, agreement, permission.OpUpdate, userID, "Rate agreement rules amended")

	return amended, nil
}

func (s *Service) ListRules(
	ctx context.Context,
	req *repositories.ListRateAgreementRulesRequest,
) ([]*rateagreement.RateAgreementRule, error) {
	return s.repo.ListRules(ctx, req)
}

func (s *Service) GetRuleByID(
	ctx context.Context,
	req *repositories.GetRateAgreementRuleByIDRequest,
) (*rateagreement.RateAgreementRule, error) {
	return s.repo.GetRuleByID(ctx, req)
}

func (s *Service) ListVersions(
	ctx context.Context,
	req *repositories.ListRateAgreementVersionsRequest,
) (*pagination.ListResult[*rateagreement.RateAgreementVersion], error) {
	return s.repo.ListVersions(ctx, req)
}

func (s *Service) audit(
	log *zap.Logger,
	entity *rateagreement.RateAgreement,
	original *rateagreement.RateAgreement,
	operation permission.Operation,
	userID pulid.ID,
	comment string,
) {
	params := &services.LogActionParams{
		Resource:       permission.ResourceRateAgreement,
		ResourceID:     entity.GetID().String(),
		Operation:      operation,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(entity),
		OrganizationID: entity.OrganizationID,
		BusinessUnitID: entity.BusinessUnitID,
	}

	opts := []services.LogOption{auditservice.WithComment(comment)}

	if original != nil {
		params.PreviousState = jsonutils.MustToJSON(original)
		opts = append(opts, auditservice.WithDiff(original, entity))
	}

	if err := s.auditService.LogAction(params, opts...); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}
}

func tenantOf(entity *rateagreement.RateAgreement) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID: entity.GetOrganizationID(),
		BuID:  entity.GetBusinessUnitID(),
	}
}
