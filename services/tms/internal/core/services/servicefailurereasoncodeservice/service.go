package servicefailurereasoncodeservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/servicefailure"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	Repo         repositories.ServiceFailureReasonCodeRepository
	AuditService services.AuditService
}

type service struct {
	l            *zap.Logger
	repo         repositories.ServiceFailureReasonCodeRepository
	auditService services.AuditService
}

func New(p Params) services.ServiceFailureReasonCodeService {
	return &service{
		l:            p.Logger.Named("service.service-failure-reason-code"),
		repo:         p.Repo,
		auditService: p.AuditService,
	}
}

func (s *service) List(
	ctx context.Context,
	req *repositories.ListServiceFailureReasonCodesRequest,
) (*pagination.ListResult[*servicefailure.ReasonCode], error) {
	if req == nil || req.Filter == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Service failure reason code list request is required",
		)
	}
	return s.repo.List(ctx, req)
}

func (s *service) Get(
	ctx context.Context,
	req repositories.GetServiceFailureReasonCodeByIDRequest,
) (*servicefailure.ReasonCode, error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}
	return s.repo.GetByID(ctx, req)
}

func (s *service) SelectOptions(
	ctx context.Context,
	req *repositories.ServiceFailureReasonCodeSelectOptionsRequest,
) (*pagination.ListResult[*servicefailure.ReasonCode], error) {
	if req == nil || req.SelectQueryRequest == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Reason code select options request is required",
		)
	}
	return s.repo.SelectOptions(ctx, req)
}

func (s *service) Create(
	ctx context.Context,
	entity *servicefailure.ReasonCode,
	actor *services.RequestActor,
) (*servicefailure.ReasonCode, error) {
	if entity == nil {
		return nil, errortypes.NewValidationError(
			"reasonCode",
			errortypes.ErrRequired,
			"Service failure reason code is required",
		)
	}
	if multiErr := validateReasonCode(entity); multiErr != nil {
		return nil, multiErr
	}

	created, err := s.repo.Create(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.logReasonCodeAction(created, actor, permission.OpCreate, nil, created, "Service failure reason code created")
	return created, nil
}

func (s *service) Update(
	ctx context.Context,
	entity *servicefailure.ReasonCode,
	actor *services.RequestActor,
) (*servicefailure.ReasonCode, error) {
	if entity == nil {
		return nil, errortypes.NewValidationError(
			"reasonCode",
			errortypes.ErrRequired,
			"Service failure reason code is required",
		)
	}
	if multiErr := validateReasonCode(entity); multiErr != nil {
		return nil, multiErr
	}

	original, err := s.repo.GetByID(ctx, repositories.GetServiceFailureReasonCodeByIDRequest{
		ID: entity.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	})
	if err != nil {
		return nil, err
	}

	updated, err := s.repo.Update(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.logReasonCodeAction(updated, actor, permission.OpUpdate, original, updated, "Service failure reason code updated")
	return updated, nil
}

func (s *service) Archive(
	ctx context.Context,
	id pulid.ID,
	tenantInfo pagination.TenantInfo,
	actor *services.RequestActor,
) (*servicefailure.ReasonCode, error) {
	original, err := s.repo.GetByID(ctx, repositories.GetServiceFailureReasonCodeByIDRequest{
		ID:         id,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if !original.Active {
		return original, nil
	}

	archived, err := s.repo.Archive(ctx, id, tenantInfo, actor.UserIDOrNil())
	if err != nil {
		return nil, err
	}

	s.logReasonCodeAction(archived, actor, permission.OpArchive, original, archived, "Service failure reason code archived")
	return archived, nil
}

func (s *service) Activate(
	ctx context.Context,
	id pulid.ID,
	tenantInfo pagination.TenantInfo,
	actor *services.RequestActor,
) (*servicefailure.ReasonCode, error) {
	original, err := s.repo.GetByID(ctx, repositories.GetServiceFailureReasonCodeByIDRequest{
		ID:         id,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if original.Active {
		return original, nil
	}

	activated, err := s.repo.Activate(ctx, id, tenantInfo, actor.UserIDOrNil())
	if err != nil {
		return nil, err
	}

	s.logReasonCodeAction(activated, actor, permission.OpUpdate, original, activated, "Service failure reason code activated")
	return activated, nil
}

func (s *service) Reorder(
	ctx context.Context,
	req *repositories.ReorderServiceFailureReasonCodesRequest,
	actor *services.RequestActor,
) ([]*servicefailure.ReasonCode, error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	reordered, err := s.repo.Reorder(ctx, req)
	if err != nil {
		return nil, err
	}

	auditActor := actor.AuditActorOrSystem()
	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceServiceFailureReasonCode,
		ResourceID:     req.TenantInfo.BuID.String(),
		Operation:      permission.OpUpdate,
		UserID:         auditActor.UserID,
		APIKeyID:       auditActor.APIKeyID,
		PrincipalType:  auditActor.PrincipalType,
		PrincipalID:    auditActor.PrincipalID,
		CurrentState:   jsonutils.MustToJSON(reordered),
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
	}, auditservice.WithComment("Service failure reason codes reordered")); err != nil {
		s.l.Warn("failed to log service failure reason code reorder audit", zap.Error(err))
	}

	return reordered, nil
}

func validateReasonCode(entity *servicefailure.ReasonCode) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func (s *service) logReasonCodeAction(
	entity *servicefailure.ReasonCode,
	actor *services.RequestActor,
	op permission.Operation,
	previous any,
	current any,
	comment string,
) {
	auditActor := actor.AuditActorOrSystem()
	params := &services.LogActionParams{
		Resource:       permission.ResourceServiceFailureReasonCode,
		ResourceID:     entity.ID.String(),
		Operation:      op,
		UserID:         auditActor.UserID,
		APIKeyID:       auditActor.APIKeyID,
		PrincipalType:  auditActor.PrincipalType,
		PrincipalID:    auditActor.PrincipalID,
		OrganizationID: entity.OrganizationID,
		BusinessUnitID: entity.BusinessUnitID,
	}
	if current != nil {
		params.CurrentState = jsonutils.MustToJSON(current)
	}
	if previous != nil {
		params.PreviousState = jsonutils.MustToJSON(previous)
	}

	opts := []services.LogOption{auditservice.WithComment(comment)}
	if previous != nil && current != nil {
		opts = append(opts, auditservice.WithDiff(previous, current))
	}
	if err := s.auditService.LogAction(params, opts...); err != nil {
		s.l.Warn("failed to log service failure reason code audit", zap.Error(err))
	}
}
