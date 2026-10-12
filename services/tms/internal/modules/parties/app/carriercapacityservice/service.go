package carriercapacityservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carriercapacity"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carriercapacityports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const invalidationAction = "carrier_capacity_changed"

type Params struct {
	fx.In

	Logger       *zap.Logger
	Repo         carriercapacityports.CarrierCapacityPostingRepository
	Validator    *Validator
	AuditService auditports.AuditService
	Invalidator  shipmentports.ShipmentInvalidator
}

type Service struct {
	l            *zap.Logger
	repo         carriercapacityports.CarrierCapacityPostingRepository
	validator    *Validator
	auditService auditports.AuditService
	invalidator  shipmentports.ShipmentInvalidator
}

func New(p Params) *Service {
	return &Service{
		l:            p.Logger.Named("service.carrier-capacity"),
		repo:         p.Repo,
		validator:    p.Validator,
		auditService: p.AuditService,
		invalidator:  p.Invalidator,
	}
}

func (s *Service) ListConnection(
	ctx context.Context,
	req *carriercapacityports.ListCarrierCapacityPostingsRequest,
) (*pagination.CursorListResult[*carriercapacity.Posting], error) {
	return s.repo.ListConnection(ctx, req)
}

func (s *Service) Get(
	ctx context.Context,
	req *carriercapacityports.GetCarrierCapacityPostingRequest,
) (*carriercapacity.Posting, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) ListOpen(
	ctx context.Context,
	req *carriercapacityports.ListOpenCarrierCapacityRequest,
) ([]*carriercapacity.Posting, error) {
	return s.repo.ListOpen(ctx, req)
}

func (s *Service) PlanCreate(
	ctx context.Context,
	entity *carriercapacity.Posting,
) (*carriercapacity.Posting, error) {
	normalize(entity)
	if multiErr := s.validator.ValidateCreate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}
	return entity, nil
}

func (s *Service) PlanUpdate(
	ctx context.Context,
	entity *carriercapacity.Posting,
) (*kernelports.RecordChange[carriercapacity.Posting], error) {
	normalize(entity)
	if multiErr := s.validator.ValidateUpdate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	original, err := s.repo.GetByID(ctx, &carriercapacityports.GetCarrierCapacityPostingRequest{
		ID: entity.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	})
	if err != nil {
		return nil, err
	}

	entity.CreatedAt = original.CreatedAt
	return &kernelports.RecordChange[carriercapacity.Posting]{Before: original, After: entity}, nil
}

func (s *Service) PlanDelete(
	ctx context.Context,
	req *carriercapacityports.DeleteCarrierCapacityPostingRequest,
) (*carriercapacity.Posting, error) {
	return s.repo.GetByID(ctx, &carriercapacityports.GetCarrierCapacityPostingRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
	})
}

func (s *Service) Create(
	ctx context.Context,
	entity *carriercapacity.Posting,
	actor *permissionports.RequestActor,
) (*carriercapacity.Posting, error) {
	planned, err := s.PlanCreate(ctx, entity)
	if err != nil {
		return nil, err
	}

	created, err := s.repo.Create(ctx, planned)
	if err != nil {
		return nil, err
	}

	s.audit(actor, created, nil, permission.OpCreate, "Carrier capacity posted")
	s.invalidate(ctx, created, actor)

	return created, nil
}

func (s *Service) Update(
	ctx context.Context,
	entity *carriercapacity.Posting,
	actor *permissionports.RequestActor,
) (*carriercapacity.Posting, error) {
	change, err := s.PlanUpdate(ctx, entity)
	if err != nil {
		return nil, err
	}

	updated, err := s.repo.Update(ctx, change.After)
	if err != nil {
		return nil, err
	}

	s.audit(actor, updated, change.Before, permission.OpUpdate, "Carrier capacity posting updated")
	s.invalidate(ctx, updated, actor)

	return updated, nil
}

func (s *Service) Delete(
	ctx context.Context,
	req *carriercapacityports.DeleteCarrierCapacityPostingRequest,
	actor *permissionports.RequestActor,
) error {
	original, err := s.PlanDelete(ctx, req)
	if err != nil {
		return err
	}

	if err = s.repo.Delete(ctx, req); err != nil {
		return err
	}

	s.audit(actor, nil, original, permission.OpDelete, "Carrier capacity posting removed")
	s.invalidate(ctx, original, actor)

	return nil
}

func (s *Service) audit(
	actor *permissionports.RequestActor,
	current, previous *carriercapacity.Posting,
	operation permission.Operation,
	comment string,
) {
	auditActor := actor.AuditActorOrSystem()
	subject := current
	if subject == nil {
		subject = previous
	}

	params := &auditports.LogActionParams{
		Resource:       permission.ResourceCarrierCapacityPosting,
		ResourceID:     subject.ID.String(),
		Operation:      operation,
		UserID:         auditActor.UserID,
		PrincipalType:  auditActor.PrincipalType,
		PrincipalID:    auditActor.PrincipalID,
		APIKeyID:       auditActor.APIKeyID,
		OrganizationID: subject.OrganizationID,
		BusinessUnitID: subject.BusinessUnitID,
	}
	options := []auditports.LogOption{auditservice.WithComment(comment)}
	if current != nil {
		params.CurrentState = jsonutils.MustToJSON(current)
	}
	if previous != nil {
		params.PreviousState = jsonutils.MustToJSON(previous)
	}
	if current != nil && previous != nil {
		options = append(options, auditservice.WithDiff(previous, current))
	}

	if err := s.auditService.LogAction(params, options...); err != nil {
		s.l.Error("failed to log carrier capacity audit action", zap.Error(err))
	}
}

func (s *Service) invalidate(
	ctx context.Context,
	entity *carriercapacity.Posting,
	actor *permissionports.RequestActor,
) {
	shipmentports.InvalidateShipments(
		ctx,
		s.invalidator,
		shipmentports.ShipmentInvalidationByActor(
			pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID},
			actor.AuditActorOrSystem(),
			pulid.Nil,
			invalidationAction,
		),
	)
}

func normalize(entity *carriercapacity.Posting) {
	if entity.Source == "" {
		entity.Source = carriercapacity.SourceManual
	}
	if entity.RateMethod == "" {
		entity.RateMethod = carriercapacity.RateMethodPerMile
	}
	if entity.TruckCount == 0 {
		entity.TruckCount = 1
	}
	entity.Carrier = nil
	entity.OriginLocation = nil
	entity.OriginState = nil
	entity.DestinationState = nil
	entity.EquipmentType = nil
	entity.BusinessUnit = nil
	entity.Organization = nil
}
