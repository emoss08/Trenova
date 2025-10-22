package shipment

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/internal/core/temporaljobs/searchjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/shipmentjobs"
	"github.com/emoss08/trenova/internal/infrastructure/meilisearch/providers"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/meilisearchtype"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/pkg/utils/jsonutils"
	"github.com/gin-gonic/gin"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ServiceParams struct {
	fx.In

	Logger              *zap.Logger
	Repo                repositories.ShipmentRepository
	TemporalClient      client.Client
	AuditService        services.AuditService
	UserRepo            repositories.UserRepository
	PermissionEngine    ports.PermissionEngine
	NotificationService services.NotificationService
	StreamingService    services.StreamingService
	ShipmentControlRepo repositories.ShipmentControlRepository
	SearchHelper        *providers.SearchHelper
}

type Service struct {
	l              *zap.Logger
	repo           repositories.ShipmentRepository
	temporalClient client.Client
	as             services.AuditService
	ns             services.NotificationService
	ur             repositories.UserRepository
	ss             services.StreamingService
	pe             ports.PermissionEngine
	scRepo         repositories.ShipmentControlRepository
	searchHelper   *providers.SearchHelper
}

//nolint:gocritic // service constructor
func NewService(p ServiceParams) *Service {
	return &Service{
		l:              p.Logger.Named("service.shipment"),
		repo:           p.Repo,
		temporalClient: p.TemporalClient,
		ns:             p.NotificationService,
		as:             p.AuditService,
		ur:             p.UserRepo,
		pe:             p.PermissionEngine,
		ss:             p.StreamingService,
		scRepo:         p.ShipmentControlRepo,
		searchHelper:   p.SearchHelper,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListShipmentRequest,
) (*pagination.ListResult[*shipment.Shipment], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) Get(
	ctx context.Context,
	req *repositories.GetShipmentByIDRequest,
) (*shipment.Shipment, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) GetPreviousRates(
	ctx context.Context,
	req *repositories.GetPreviousRatesRequest,
) (*pagination.ListResult[*shipment.Shipment], error) {
	return s.repo.GetPreviousRates(ctx, req)
}

func (s *Service) Create(
	ctx context.Context,
	entity *shipment.Shipment,
	userID pulid.ID,
) (*shipment.Shipment, error) {
	log := s.l.With(
		zap.String("operation", "Create"),
		zap.String("orgID", entity.OrganizationID.String()),
		zap.String("buID", entity.BusinessUnitID.String()),
		zap.String("userID", userID.String()),
	)

	createdEntity, err := s.repo.Create(ctx, entity, userID)
	if err != nil {
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceShipment,
			ResourceID:     createdEntity.GetID(),
			Operation:      permission.OpCreate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(createdEntity),
			OrganizationID: createdEntity.OrganizationID,
			BusinessUnitID: createdEntity.BusinessUnitID,
		},
		audit.WithComment("Shipment created"),
	)
	if err != nil {
		log.Error("failed to log shipment creation", zap.Error(err))
	}

	if err = s.IndexInSearch(ctx, createdEntity.ID, createdEntity.OrganizationID, createdEntity.BusinessUnitID); err != nil {
		log.Warn("failed to index created shipment in search", zap.Error(err))
	}

	return createdEntity, nil
}

func (s *Service) Update(
	ctx context.Context,
	entity *shipment.Shipment,
	userID pulid.ID,
) (*shipment.Shipment, error) {
	log := s.l.With(
		zap.String("operation", "Update"),
		zap.String("orgID", entity.OrganizationID.String()),
		zap.String("buID", entity.BusinessUnitID.String()),
		zap.String("userID", userID.String()),
	)

	original, err := s.repo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID:    entity.ID,
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
		ShipmentOptions: repositories.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		return nil, err
	}

	updatedEntity, err := s.repo.Update(ctx, entity, userID)
	if err != nil {
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceShipment,
			ResourceID:     updatedEntity.GetID(),
			Operation:      permission.OpUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(updatedEntity),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: updatedEntity.OrganizationID,
			BusinessUnitID: updatedEntity.BusinessUnitID,
		},
		audit.WithComment("Shipment updated"),
		audit.WithDiff(original, updatedEntity),
	)
	if err != nil {
		log.Error("failed to log shipment update", zap.Error(err))
	}

	if err = s.IndexInSearch(ctx, updatedEntity.ID, updatedEntity.OrganizationID, updatedEntity.BusinessUnitID); err != nil {
		log.Warn("failed to index ownership transferred shipment in search", zap.Error(err))
	}

	return updatedEntity, nil
}

func (s *Service) Duplicate(req *repositories.DuplicateShipmentRequest) error {
	log := s.l.With(
		zap.String("operation", "Duplicate"),
		zap.Any("request", req),
	)
	if err := req.Validate(); err != nil {
		return err
	}

	payload := &shipmentjobs.DuplicateShipmentPayload{
		BasePayload: temporaltype.BasePayload{
			OrganizationID: req.OrgID,
			BusinessUnitID: req.BuID,
			UserID:         req.UserID,
		},
		ShipmentID:               req.ShipmentID,
		Count:                    req.Count,
		OverrideDates:            req.OverrideDates,
		IncludeCommodities:       req.IncludeCommodities,
		IncludeAdditionalCharges: req.IncludeAdditionalCharges,
	}

	workflowID := fmt.Sprintf(
		"shipment-duplicate-%s-%d",
		req.ShipmentID.String(),
		time.Now().UnixNano(),
	)

	_, err := s.temporalClient.ExecuteWorkflow(
		context.Background(),
		client.StartWorkflowOptions{
			ID:        workflowID,
			TaskQueue: shipmentjobs.ShipmentTaskQueue,
		},
		shipmentjobs.SendBulkDuplicateShipmentWorkflow,
		payload,
	)
	if err != nil {
		log.Error("failed to execute workflow", zap.Error(err))
	}

	return nil
}

func (s *Service) TransferOwnership(
	ctx context.Context,
	req *repositories.TransferOwnershipRequest,
) (*shipment.Shipment, error) {
	log := s.l.With(
		zap.String("operation", "TransferOwnership"),
		zap.String("shipmentID", req.ShipmentID.String()),
		zap.String("ownerID", req.OwnerID.String()),
	)

	original, err := s.repo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID:    req.ShipmentID,
		OrgID: req.OrgID,
		BuID:  req.BuID,
	})
	if err != nil {
		return nil, err
	}
	isAdmin, err := s.pe.HasAdminRole(ctx, req.UserID, req.OrgID)
	if err != nil {
		return nil, err
	}

	isOwner := original.OwnerID != nil && *original.OwnerID == req.UserID

	if !isOwner && !isAdmin {
		return nil, errortypes.NewValidationError(
			"ownerId",
			errortypes.ErrInvalid,
			"You do not have permission to transfer ownership of this shipment",
		)
	}

	updatedEntity, err := s.repo.TransferOwnership(ctx, req)
	if err != nil {
		log.Error("failed to transfer ownership of shipment", zap.Error(err))
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceShipment,
			ResourceID:     updatedEntity.GetID(),
			Operation:      permission.OpUpdate,
			UserID:         req.UserID,
			CurrentState:   jsonutils.MustToJSON(updatedEntity),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: updatedEntity.OrganizationID,
			BusinessUnitID: updatedEntity.BusinessUnitID,
		},
		audit.WithComment("Shipment ownership transferred"),
		audit.WithDiff(original, updatedEntity),
	)
	if err != nil {
		log.Error("failed to log shipment ownership transfer", zap.Error(err))
	}

	ownerName, err := s.ur.GetNameByID(ctx, req.UserID)
	if err != nil {
		log.Error("failed to get original owner", zap.Error(err))
		return nil, err
	}

	err = s.ns.SendOwnershipTransferNotification(
		ctx,
		&services.OwnershipTransferNotificationRequest{
			OrgID:        req.OrgID,
			BuID:         req.BuID,
			OwnerName:    ownerName,
			ProNumber:    original.ProNumber,
			TargetUserID: pulid.ConvertFromPtr(updatedEntity.OwnerID),
		},
	)
	if err != nil {
		log.Error("failed to send ownership transfer notification", zap.Error(err))
	}

	if err = s.IndexInSearch(ctx, updatedEntity.ID, updatedEntity.OrganizationID, updatedEntity.BusinessUnitID); err != nil {
		log.Warn("failed to index ownership transferred shipment in search", zap.Error(err))
	}

	return updatedEntity, nil
}

func (s *Service) CheckForDuplicateBOLs(ctx context.Context, entity *shipment.Shipment) error {
	log := s.l.With(
		zap.String("operation", "CheckForDuplicateBOLs"),
		zap.String("bol", entity.BOL),
	)

	sc, err := s.scRepo.GetByOrgID(ctx, entity.OrganizationID)
	if err != nil {
		log.Error("failed to get shipment control", zap.Error(err))
		return err
	}

	if !sc.CheckForDuplicateBOLs {
		return nil
	}

	var excludeID *pulid.ID
	if !entity.ID.IsNil() {
		excludeID = &entity.ID
		log.Debug(
			"excluding current shipment from duplicate check",
			zap.String("excludeID", entity.ID.String()),
		)
	}

	me := errortypes.NewMultiError()

	duplicates, err := s.repo.CheckForDuplicateBOLs(ctx, &repositories.DuplicateBolsRequest{
		CurrentBOL: entity.BOL,
		OrgID:      entity.OrganizationID,
		BuID:       entity.BusinessUnitID,
		ExcludeID:  excludeID,
	})
	if err != nil {
		log.Error("failed to check for duplicate BOLs", zap.Error(err))
		return err
	}

	if len(duplicates) > 0 {
		log.Debug("duplicate BOLs found", zap.Int("duplicateCount", len(duplicates)))

		proNumbers := make([]string, 0, len(duplicates))
		for _, dup := range duplicates {
			proNumbers = append(proNumbers, dup.ProNumber)
		}

		me.Add(
			"bol",
			errortypes.ErrInvalid,
			fmt.Sprintf(
				"BOL is already in use by shipment(s) with Pro Number(s): %s",
				strings.Join(proNumbers, ", "),
			),
		)
	}

	if me.HasErrors() {
		return me
	}

	return nil
}

func (s *Service) Cancel(
	ctx context.Context,
	req *repositories.CancelShipmentRequest,
) (*shipment.Shipment, error) {
	log := s.l.With(
		zap.String("operation", "Cancel"),
		zap.String("shipmentID", req.ShipmentID.String()),
	)

	original, err := s.repo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID:    req.ShipmentID,
		OrgID: req.OrgID,
		BuID:  req.BuID,
	})
	if err != nil {
		return nil, err
	}

	updatedEntity, err := s.repo.Cancel(ctx, req)
	if err != nil {
		log.Error("failed to cancel shipment", zap.Error(err))
		return nil, err
	}

	err = s.as.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceShipment,
		ResourceID:     updatedEntity.GetID(),
		Operation:      permission.OpUpdate,
		UserID:         req.CanceledByID,
		CurrentState:   jsonutils.MustToJSON(updatedEntity),
		PreviousState:  jsonutils.MustToJSON(original),
		OrganizationID: updatedEntity.OrganizationID,
		BusinessUnitID: updatedEntity.BusinessUnitID,
	},
		audit.WithComment("Shipment canceled"),
		audit.WithDiff(original, updatedEntity),
	)
	if err != nil {
		log.Error("failed to log shipment cancellation", zap.Error(err))
	}

	if err = s.IndexInSearch(ctx, updatedEntity.ID, updatedEntity.OrganizationID, updatedEntity.BusinessUnitID); err != nil {
		log.Warn("failed to index canceled shipment in search", zap.Error(err))
	}

	return updatedEntity, nil
}

func (s *Service) UnCancel(
	ctx context.Context,
	req *repositories.UnCancelShipmentRequest,
) (*shipment.Shipment, error) {
	log := s.l.With(
		zap.String("operation", "UnCancel"),
		zap.String("shipmentID", req.ShipmentID.String()),
	)

	original, err := s.repo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID:    req.ShipmentID,
		OrgID: req.OrgID,
		BuID:  req.BuID,
	})
	if err != nil {
		return nil, err
	}

	updatedEntity, err := s.repo.UnCancel(ctx, req)
	if err != nil {
		log.Error("failed to un-cancel shipment", zap.Error(err))
		return nil, err
	}

	err = s.as.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceShipment,
		ResourceID:     updatedEntity.GetID(),
		Operation:      permission.OpUpdate,
		UserID:         req.UserID,
		CurrentState:   jsonutils.MustToJSON(updatedEntity),
		PreviousState:  jsonutils.MustToJSON(original),
		OrganizationID: updatedEntity.OrganizationID,
		BusinessUnitID: updatedEntity.BusinessUnitID,
	},
		audit.WithComment("Shipment uncanceled"),
		audit.WithDiff(original, updatedEntity),
	)
	if err != nil {
		log.Error("failed to log shipment un-cancellation", zap.Error(err))
	}

	if err = s.IndexInSearch(ctx, updatedEntity.ID, updatedEntity.OrganizationID, updatedEntity.BusinessUnitID); err != nil {
		log.Warn("failed to index uncanceled shipment in search", zap.Error(err))
	}

	return updatedEntity, nil
}

func (s *Service) CalculateTotals(
	ctx context.Context,
	shp *shipment.Shipment,
	userID pulid.ID,
) (*repositories.ShipmentTotalsResponse, error) {
	return s.repo.CalculateTotals(ctx, shp, userID)
}

func (s *Service) Stream(c *gin.Context) {
	s.ss.StreamData(c, "shipments")
}

func (s *Service) IndexInSearch(ctx context.Context, shpID, orgID, buID pulid.ID) error {
	log := s.l.With(
		zap.String("operation", "IndexInSearch"),
		zap.String("shipmentID", shpID.String()),
		zap.String("orgID", orgID.String()),
		zap.String("buID", buID.String()),
	)
	payload := &searchjobs.IndexEntityPayload{
		BasePayload: temporaltype.BasePayload{
			OrganizationID: orgID,
			BusinessUnitID: buID,
		},
		EntityType: meilisearchtype.EntityTypeShipment,
		EntityID:   shpID,
	}

	workflowID := fmt.Sprintf("index-shipment-in-search-%s-%d", shpID.String(), time.Now().Unix())

	_, err := s.temporalClient.ExecuteWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID:        workflowID,
			TaskQueue: searchjobs.SearchTaskQueue,
		},
		searchjobs.IndexEntityWorkflow,
		payload,
	)
	if err != nil {
		log.Error("failed to execute workflow", zap.Error(err))
	}

	return nil
}
