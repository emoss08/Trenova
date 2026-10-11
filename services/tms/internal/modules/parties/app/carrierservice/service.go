package carrierservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierintelports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	Repo         carrierports.CarrierRepository
	Validator    *Validator
	AuditService auditports.AuditService
	Realtime     agentports.RealtimeService
	DB           kernelports.DBConnection                   `optional:"true"`
	Sync         accountingsyncports.AccountingSyncEnqueuer `optional:"true"`
}

type Service struct {
	l            *zap.Logger
	repo         carrierports.CarrierRepository
	validator    *Validator
	auditService auditports.AuditService
	realtime     agentports.RealtimeService
	observer     carrierintelports.CarrierLifecycleObserver
	db           kernelports.DBConnection
	sync         accountingsyncports.AccountingSyncEnqueuer
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
	err := s.db.WithTx(ctx, kernelports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		var txErr error
		if updated, txErr = s.repo.Update(txCtx, entity); txErr != nil {
			return txErr
		}
		if updated.SamePartyDetails(original) {
			return nil
		}
		return accountingsyncports.EnqueueAccountingSync(
			txCtx,
			s.sync,
			accountingsyncports.VendorSyncRequest(
				pagination.TenantInfo{OrgID: updated.OrganizationID, BuID: updated.BusinessUnitID},
				accountingsync.SyncObjectCarrierVendor,
				updated.ID,
				updated.Name,
				updated.Version,
			),
		)
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *Service) List(
	ctx context.Context,
	req *carrierports.ListCarrierRequest,
) (*pagination.ListResult[*carrier.Carrier], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) ListConnection(
	ctx context.Context,
	req *carrierports.ListCarrierConnectionRequest,
) (*pagination.CursorListResult[*carrier.Carrier], error) {
	return s.repo.ListConnection(ctx, req)
}

func (s *Service) Get(
	ctx context.Context,
	req carrierports.GetCarrierByIDRequest,
) (*carrier.Carrier, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) SelectOptions(
	ctx context.Context,
	req *carrierports.CarrierSelectOptionsRequest,
) (*pagination.ListResult[*carrier.Carrier], error) {
	return s.repo.SelectOptions(ctx, req)
}

func (s *Service) BulkUpdateStatus(
	ctx context.Context,
	req *carrierports.BulkUpdateCarrierStatusRequest,
) ([]*carrier.Carrier, error) {
	log := s.l.With(
		zap.String("operation", "BulkUpdateStatus"),
		zap.Any("request", req),
	)

	changes, err := s.PlanBulkUpdateStatus(ctx, req)
	if err != nil {
		return nil, err
	}
	originalEntities := kernelports.Befores(changes)

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
		s.notifyObserver(ctx, &carrierintelports.CarrierLifecycleEvent{
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
	actor *permissionports.RequestActor,
) (*carrier.Carrier, error) {
	auditActor := actor.AuditActor()
	log := s.l.With(
		zap.String("operation", "Create"),
		zap.String("principalType", string(auditActor.PrincipalType)),
		zap.String("principalID", auditActor.PrincipalID.String()),
	)

	planned, err := s.PlanCreate(ctx, entity)
	if err != nil {
		return nil, err
	}

	createdEntity, err := s.repo.Create(ctx, planned)
	if err != nil {
		log.Error("failed to create carrier", zap.Error(err))
		return nil, err
	}

	if err = s.auditService.LogAction(&auditports.LogActionParams{
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

	s.notifyObserver(ctx, &carrierintelports.CarrierLifecycleEvent{
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
	actor *permissionports.RequestActor,
) (*carrier.Carrier, error) {
	auditActor := actor.AuditActor()
	log := s.l.With(
		zap.String("operation", "Update"),
		zap.String("principalType", string(auditActor.PrincipalType)),
		zap.String("principalID", auditActor.PrincipalID.String()),
	)

	change, err := s.PlanUpdate(ctx, entity)
	if err != nil {
		return nil, err
	}
	original := change.Before

	updatedEntity, err := s.updateAndQueueSync(ctx, entity, original)
	if err != nil {
		log.Error("failed to update carrier", zap.Error(err))
		return nil, err
	}

	if err = s.auditService.LogAction(&auditports.LogActionParams{
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

	if auditActor.PrincipalType != permissionports.PrincipalTypeSystem {
		s.notifyObserver(ctx, &carrierintelports.CarrierLifecycleEvent{
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

func (s *Service) SetLifecycleObserver(observer carrierintelports.CarrierLifecycleObserver) {
	s.observer = observer
}

func (s *Service) notifyObserver(
	ctx context.Context,
	event *carrierintelports.CarrierLifecycleEvent,
) {
	if s.observer == nil {
		return
	}
	s.observer.CarrierSaved(ctx, event)
}
