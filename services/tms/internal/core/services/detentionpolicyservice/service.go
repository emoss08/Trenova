package detentionpolicyservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	Repo         detentionports.DetentionPolicyRepository
	Validator    *Validator
	AuditService auditports.AuditService
}

type Service struct {
	l            *zap.Logger
	repo         detentionports.DetentionPolicyRepository
	validator    *Validator
	auditService auditports.AuditService
}

func New(p Params) *Service {
	return &Service{
		l:            p.Logger.Named("service.detention-policy"),
		repo:         p.Repo,
		validator:    p.Validator,
		auditService: p.AuditService,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *detentionports.ListDetentionPoliciesRequest,
) (*pagination.ListResult[*detention.DetentionPolicy], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) ListConnection(
	ctx context.Context,
	req *detentionports.ListDetentionPolicyConnectionRequest,
) (*pagination.CursorListResult[*detention.DetentionPolicy], error) {
	return s.repo.ListConnection(ctx, req)
}

func (s *Service) GetByID(
	ctx context.Context,
	req *detentionports.GetDetentionPolicyByIDRequest,
) (*detention.DetentionPolicy, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) SelectOptions(
	ctx context.Context,
	req *detentionports.DetentionPolicySelectOptionsRequest,
) (*pagination.ListResult[*detention.DetentionPolicy], error) {
	return s.repo.SelectOptions(ctx, req)
}

func (s *Service) Create(
	ctx context.Context,
	entity *detention.DetentionPolicy,
	userID pulid.ID,
) (*detention.DetentionPolicy, error) {
	log := s.l.With(
		zap.String("operation", "Create"),
		zap.String("userID", userID.String()),
	)

	if multiErr := s.validator.ValidateCreate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	createdEntity, err := s.repo.Create(ctx, entity)
	if err != nil {
		log.Error("failed to create detention policy", zap.Error(err))
		return nil, err
	}

	if err = s.auditService.LogAction(&auditports.LogActionParams{
		Resource:       permission.ResourceDetentionPolicy,
		ResourceID:     createdEntity.GetID().String(),
		Operation:      permission.OpCreate,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(createdEntity),
		OrganizationID: createdEntity.OrganizationID,
		BusinessUnitID: createdEntity.BusinessUnitID,
	},
		auditservice.WithComment("Detention policy created"),
	); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	return createdEntity, nil
}

func (s *Service) Update(
	ctx context.Context,
	entity *detention.DetentionPolicy,
	userID pulid.ID,
) (*detention.DetentionPolicy, error) {
	log := s.l.With(
		zap.String("operation", "Update"),
		zap.String("userID", userID.String()),
	)

	if multiErr := s.validator.ValidateUpdate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	original, err := s.repo.GetByID(ctx, &detentionports.GetDetentionPolicyByIDRequest{
		DetentionPolicyID: entity.GetID(),
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.GetOrganizationID(),
			BuID:  entity.GetBusinessUnitID(),
		},
		IncludeTiers: true,
	})
	if err != nil {
		log.Error("failed to get original detention policy", zap.Error(err))
		return nil, err
	}

	updatedEntity, err := s.repo.Update(ctx, entity)
	if err != nil {
		log.Error("failed to update detention policy", zap.Error(err))
		return nil, err
	}

	if err = s.auditService.LogAction(&auditports.LogActionParams{
		Resource:       permission.ResourceDetentionPolicy,
		ResourceID:     updatedEntity.GetID().String(),
		Operation:      permission.OpUpdate,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(updatedEntity),
		PreviousState:  jsonutils.MustToJSON(original),
		OrganizationID: updatedEntity.OrganizationID,
		BusinessUnitID: updatedEntity.BusinessUnitID,
	},
		auditservice.WithComment("Detention policy updated"),
		auditservice.WithDiff(original, updatedEntity),
	); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	return updatedEntity, nil
}

func (s *Service) Delete(
	ctx context.Context,
	req *detentionports.GetDetentionPolicyByIDRequest,
	userID pulid.ID,
) error {
	log := s.l.With(
		zap.String("operation", "Delete"),
		zap.String("id", req.DetentionPolicyID.String()),
		zap.String("userID", userID.String()),
	)

	existing, err := s.repo.GetByID(ctx, req)
	if err != nil {
		log.Error("failed to get detention policy for delete", zap.Error(err))
		return err
	}

	if err = s.repo.Delete(ctx, req); err != nil {
		log.Error("failed to delete detention policy", zap.Error(err))
		return err
	}

	if err = s.auditService.LogAction(&auditports.LogActionParams{
		Resource:       permission.ResourceDetentionPolicy,
		ResourceID:     existing.GetID().String(),
		Operation:      permission.OpDelete,
		UserID:         userID,
		PreviousState:  jsonutils.MustToJSON(existing),
		OrganizationID: existing.OrganizationID,
		BusinessUnitID: existing.BusinessUnitID,
	},
		auditservice.WithComment("Detention policy deleted"),
	); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	return nil
}
