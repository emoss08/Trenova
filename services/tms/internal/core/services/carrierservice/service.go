package carrierservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/uptrace/bun"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/jsonutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	Repo         repositories.CarrierRepository
	Validator    *Validator
	AuditService services.AuditService
	Realtime     services.RealtimeService
	DB           ports.DBConnection              `optional:"true"`
	Sync         services.AccountingSyncEnqueuer `optional:"true"`
}

type Service struct {
	l            *zap.Logger
	repo         repositories.CarrierRepository
	validator    *Validator
	auditService services.AuditService
	realtime     services.RealtimeService
	observer     services.CarrierLifecycleObserver
	db           ports.DBConnection
	sync         services.AccountingSyncEnqueuer
}

func New(p Params) *Service {
	return &Service{
		l:            p.Logger.Named("service.carrier"),
		repo:         p.Repo,
		validator:    p.Validator,
		auditService: p.AuditService,
		realtime:     p.Realtime,
		db:           p.DB,
		sync:         p.Sync,
	}
}

func (s *Service) updateAndQueueSync(
	ctx context.Context,
	entity *carrier.Carrier,
	original *carrier.Carrier,
) (*carrier.Carrier, error) {
	if s.db == nil || s.sync == nil {
		return s.repo.Update(ctx, entity)
	}

	var updated *carrier.Carrier
	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		var txErr error
		if updated, txErr = s.repo.Update(txCtx, entity); txErr != nil {
			return txErr
		}
		if updated.SamePartyDetails(original) {
			return nil
		}
		return services.EnqueueAccountingSync(txCtx, s.sync, services.VendorSyncRequest(
			pagination.TenantInfo{OrgID: updated.OrganizationID, BuID: updated.BusinessUnitID},
			accountingsync.SyncObjectCarrierVendor,
			updated.ID,
			updated.Name,
			updated.Version,
		))
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListCarrierRequest,
) (*pagination.ListResult[*carrier.Carrier], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) ListConnection(
	ctx context.Context,
	req *repositories.ListCarrierConnectionRequest,
) (*pagination.CursorListResult[*carrier.Carrier], error) {
	return s.repo.ListConnection(ctx, req)
}

func (s *Service) Get(
	ctx context.Context,
	req repositories.GetCarrierByIDRequest,
) (*carrier.Carrier, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) SelectOptions(
	ctx context.Context,
	req *repositories.CarrierSelectOptionsRequest,
) (*pagination.ListResult[*carrier.Carrier], error) {
	return s.repo.SelectOptions(ctx, req)
}

func (s *Service) BulkUpdateStatus(
	ctx context.Context,
	req *repositories.BulkUpdateCarrierStatusRequest,
) ([]*carrier.Carrier, error) {
	log := s.l.With(
		zap.String("operation", "BulkUpdateStatus"),
		zap.Any("request", req),
	)

	// Note: customerservice.BulkUpdateStatus has the same missing-enum-check gap;
	// it is left untouched on this branch.
	if !req.Status.IsValid() {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalid,
			"Carrier status is invalid",
		)
	}

	originalEntities, err := s.repo.GetByIDs(ctx, repositories.GetCarriersByIDsRequest{
		TenantInfo: req.TenantInfo,
		CarrierIDs: req.CarrierIDs,
	})
	if err != nil {
		log.Error("failed to get original carriers", zap.Error(err))
		return nil, err
	}

	entities, err := s.repo.BulkUpdateStatus(ctx, req)
	if err != nil {
		log.Error("failed to bulk update carrier status", zap.Error(err))
		return nil, err
	}

	entries := auditservice.BuildBulkLogEntries(
		&auditservice.BulkLogEntriesParams[*carrier.Carrier]{
			Resource:  permission.ResourceCarrier,
			Operation: permission.OpUpdate,
			UserID:    req.TenantInfo.UserID,
			Updated:   entities,
			Originals: originalEntities,
		},
		auditservice.WithComment("Carrier status updated"),
	)

	if err = s.auditService.LogActions(entries); err != nil {
		log.Error("failed to log audit actions", zap.Error(err))
	}

	if err = realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		ActorUserID:    req.TenantInfo.UserID,
		Resource:       "carriers",
		Action:         "bulk_updated",
	}); err != nil {
		log.Warn("failed to publish carrier invalidation", zap.Error(err))
	}

	previous := make(map[string]carrier.Status, len(originalEntities))
	for _, original := range originalEntities {
		previous[original.ID.String()] = original.Status
	}
	for _, entity := range entities {
		s.notifyObserver(ctx, &services.CarrierLifecycleEvent{
			TenantInfo:     req.TenantInfo,
			Carrier:        entity,
			PreviousStatus: previous[entity.ID.String()],
		})
	}

	return entities, nil
}

func (s *Service) Create(
	ctx context.Context,
	entity *carrier.Carrier,
	actor *services.RequestActor,
) (*carrier.Carrier, error) {
	auditActor := actor.AuditActor()
	log := s.l.With(
		zap.String("operation", "Create"),
		zap.String("principalType", string(auditActor.PrincipalType)),
		zap.String("principalID", auditActor.PrincipalID.String()),
	)

	if multiErr := s.validator.ValidateCreate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	createdEntity, err := s.repo.Create(ctx, entity)
	if err != nil {
		log.Error("failed to create carrier", zap.Error(err))
		return nil, err
	}

	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceCarrier,
		ResourceID:     createdEntity.GetID().String(),
		Operation:      permission.OpCreate,
		UserID:         auditActor.UserID,
		PrincipalType:  auditActor.PrincipalType,
		PrincipalID:    auditActor.PrincipalID,
		APIKeyID:       auditActor.APIKeyID,
		CurrentState:   jsonutils.MustToJSON(createdEntity),
		OrganizationID: createdEntity.OrganizationID,
		BusinessUnitID: createdEntity.BusinessUnitID,
	},
		auditservice.WithComment("Carrier created"),
	); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	if err = realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: createdEntity.OrganizationID,
		BusinessUnitID: createdEntity.BusinessUnitID,
		ActorUserID:    auditActor.UserID,
		ActorType:      auditActor.PrincipalType,
		ActorID:        auditActor.PrincipalID,
		ActorAPIKeyID:  auditActor.APIKeyID,
		Resource:       "carriers",
		Action:         "created",
		RecordID:       createdEntity.GetID(),
		Entity:         createdEntity,
	}); err != nil {
		log.Warn("failed to publish carrier invalidation", zap.Error(err))
	}

	s.notifyObserver(ctx, &services.CarrierLifecycleEvent{
		TenantInfo: pagination.TenantInfo{
			OrgID:  createdEntity.OrganizationID,
			BuID:   createdEntity.BusinessUnitID,
			UserID: auditActor.UserID,
		},
		Carrier: createdEntity,
		Created: true,
	})

	return createdEntity, nil
}

func (s *Service) Update(
	ctx context.Context,
	entity *carrier.Carrier,
	actor *services.RequestActor,
) (*carrier.Carrier, error) {
	auditActor := actor.AuditActor()
	log := s.l.With(
		zap.String("operation", "Update"),
		zap.String("principalType", string(auditActor.PrincipalType)),
		zap.String("principalID", auditActor.PrincipalID.String()),
	)

	if multiErr := s.validator.ValidateUpdate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	original, err := s.repo.GetByID(ctx, repositories.GetCarrierByIDRequest{
		ID: entity.GetID(),
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.GetOrganizationID(),
			BuID:  entity.GetBusinessUnitID(),
		},
		CarrierFilterOptions: repositories.CarrierFilterOptions{
			IncludeContacts:          true,
			IncludeInsurancePolicies: true,
		},
	})
	if err != nil {
		log.Error("failed to get original carrier", zap.Error(err))
		return nil, err
	}

	updatedEntity, err := s.updateAndQueueSync(ctx, entity, original)
	if err != nil {
		log.Error("failed to update carrier", zap.Error(err))
		return nil, err
	}

	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceCarrier,
		ResourceID:     updatedEntity.GetID().String(),
		Operation:      permission.OpUpdate,
		UserID:         auditActor.UserID,
		PrincipalType:  auditActor.PrincipalType,
		PrincipalID:    auditActor.PrincipalID,
		APIKeyID:       auditActor.APIKeyID,
		CurrentState:   jsonutils.MustToJSON(updatedEntity),
		PreviousState:  jsonutils.MustToJSON(original),
		OrganizationID: updatedEntity.OrganizationID,
		BusinessUnitID: updatedEntity.BusinessUnitID,
	},
		auditservice.WithComment("Carrier updated"),
		auditservice.WithDiff(original, updatedEntity),
	); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	if err = realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: updatedEntity.OrganizationID,
		BusinessUnitID: updatedEntity.BusinessUnitID,
		ActorUserID:    auditActor.UserID,
		ActorType:      auditActor.PrincipalType,
		ActorID:        auditActor.PrincipalID,
		ActorAPIKeyID:  auditActor.APIKeyID,
		Resource:       "carriers",
		Action:         "updated",
		RecordID:       updatedEntity.GetID(),
		Entity:         updatedEntity,
	}); err != nil {
		log.Warn("failed to publish carrier invalidation", zap.Error(err))
	}

	if auditActor.PrincipalType != services.PrincipalTypeSystem {
		s.notifyObserver(ctx, &services.CarrierLifecycleEvent{
			TenantInfo: pagination.TenantInfo{
				OrgID:  updatedEntity.OrganizationID,
				BuID:   updatedEntity.BusinessUnitID,
				UserID: auditActor.UserID,
			},
			Carrier:        updatedEntity,
			PreviousStatus: original.Status,
		})
	}

	return updatedEntity, nil
}

func (s *Service) SetLifecycleObserver(observer services.CarrierLifecycleObserver) {
	s.observer = observer
}

func (s *Service) notifyObserver(ctx context.Context, event *services.CarrierLifecycleEvent) {
	if s.observer == nil {
		return
	}
	s.observer.CarrierSaved(ctx, event)
}
