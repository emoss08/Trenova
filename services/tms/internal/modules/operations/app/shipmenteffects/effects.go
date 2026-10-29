package shipmenteffects

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/equipment/ports/equipmenttypeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/shipmentcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmenteventservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentvalidator"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/orderports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmenteventports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity"
	"github.com/emoss08/trenova/internal/modules/reference/ports/commodityports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

// RefuseCreditHold stops new business for a customer whose credit is held:
// a new or duplicated shipment, or an existing one moved onto that payer.
// A shipment already billed to a held payer can still be worked, since the
// freight is booked and has to move and be billed regardless.
func (s *Effects) RefuseCreditHold(
	ctx context.Context,
	entity *shipment.Shipment,
	original *shipment.Shipment,
) error {
	if s.customerRepo == nil || entity == nil {
		return nil
	}
	payerID := entity.PayerID()
	if payerID.IsNil() || (original != nil && original.PayerID() == payerID) {
		return nil
	}

	payer, err := s.customerRepo.GetByID(ctx, customerports.GetCustomerByIDRequest{
		ID: payerID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		CustomerFilterOptions: customerports.CustomerFilterOptions{IncludeBillingProfile: true},
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil
		}
		return err
	}
	if payer == nil || payer.BillingProfile == nil || !payer.BillingProfile.IsOnCreditHold() {
		return nil
	}

	message := payer.Name + " is on credit hold"
	if reason := payer.BillingProfile.CreditHoldReason; reason != "" {
		message += ": " + reason
	}
	message += ". Accounts receivable has to lift the hold before new shipments are booked for them."

	field := "customerId"
	if entity.HasExplicitBillTo() {
		field = "billToCustomerId"
	}
	return errortypes.NewValidationError(field, errortypes.ErrInvalid, message)
}

func (s *Effects) deckHeightFeetForShipment(
	ctx context.Context,
	entity *shipment.Shipment,
) float64 {
	if entity.TrailerTypeID.IsNil() || s.equipmentTypeRepo == nil {
		return 0
	}

	equipmentType, err := s.equipmentTypeRepo.GetByID(
		ctx,
		equipmenttypeports.GetEquipmentTypeByIDRequest{
			ID: entity.TrailerTypeID,
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		},
	)
	if err != nil {
		s.l.Warn("failed to load trailer type for envelope derivation", zap.Error(err))
		return 0
	}

	return equipmentType.DeckHeightFeet()
}

func (s *Effects) ApplyShipmentEnvelope(
	ctx context.Context,
	entity *shipment.Shipment,
) {
	if entity == nil {
		return
	}

	deckHeightFeet := s.deckHeightFeetForShipment(ctx, entity)

	entity.ApplyEnvelope(shipment.ComputeEnvelope(entity.Commodities, deckHeightFeet))
}

type Effects struct {
	l                   *zap.Logger
	repo                shipmentports.ShipmentRepository
	controlRepo         tenantports.ShipmentControlRepository
	commodityRepo       commodityports.CommodityRepository
	equipmentTypeRepo   equipmenttypeports.EquipmentTypeRepository
	customerRepo        customerports.CustomerRepository
	notificationService *notificationservice.Service
	auditService        auditports.AuditService
	eventService        shipmenteventports.ShipmentEventService
	orderDerivation     orderports.OrderDerivationService
}

type Deps struct {
	Logger              *zap.Logger
	Repo                shipmentports.ShipmentRepository
	ControlRepo         tenantports.ShipmentControlRepository
	CommodityRepo       commodityports.CommodityRepository
	EquipmentTypeRepo   equipmenttypeports.EquipmentTypeRepository
	CustomerRepo        customerports.CustomerRepository
	NotificationService *notificationservice.Service
	AuditService        auditports.AuditService
	EventService        shipmenteventports.ShipmentEventService
	OrderDerivation     orderports.OrderDerivationService
}

func New(d Deps) *Effects {
	return &Effects{
		l:                   d.Logger,
		repo:                d.Repo,
		controlRepo:         d.ControlRepo,
		commodityRepo:       d.CommodityRepo,
		equipmentTypeRepo:   d.EquipmentTypeRepo,
		customerRepo:        d.CustomerRepo,
		notificationService: d.NotificationService,
		auditService:        d.AuditService,
		eventService:        d.EventService,
		orderDerivation:     d.OrderDerivation,
	}
}

func (s *Effects) CreateNotification(
	ctx context.Context,
	entity *notification.Notification,
) {
	if s.notificationService == nil || entity == nil {
		return
	}

	if _, err := s.notificationService.Create(ctx, entity); err != nil {
		s.l.Warn("failed to create notification", zap.Error(err))
	}
}

func (s *Effects) RecomputeOrdersForShipments(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipments []*shipment.Shipment,
) error {
	if s.orderDerivation == nil {
		return nil
	}

	orderIDs := make(map[pulid.ID]struct{}, len(shipments))
	for _, entity := range shipments {
		if entity == nil || entity.OrderID.IsNil() {
			continue
		}
		orderIDs[entity.OrderID] = struct{}{}
	}

	for orderID := range orderIDs {
		if err := s.orderDerivation.RecomputeOrder(ctx, tenantInfo, orderID); err != nil {
			return err
		}
	}

	return nil
}

func (s *Effects) GetShipmentControl(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*shipmentcontrol.ShipmentControl, error) {
	return s.controlRepo.Get(ctx, tenantports.GetShipmentControlRequest{
		TenantInfo: tenantInfo,
	})
}

func (s *Effects) CheckDuplicateBOLsWithControl(
	ctx context.Context,
	control *shipmentcontrol.ShipmentControl,
	req *shipmentports.DuplicateBOLCheckRequest,
) error {
	if !control.CheckForDuplicateBOLs || strings.TrimSpace(req.BOL) == "" {
		return nil
	}

	duplicates, err := s.repo.CheckForDuplicateBOLs(ctx, req)
	if err != nil {
		return err
	}

	if len(duplicates) == 0 {
		return nil
	}

	proNumbers := make([]string, 0, len(duplicates))
	for _, duplicate := range duplicates {
		proNumbers = append(proNumbers, duplicate.ProNumber)
	}

	me := errortypes.NewMultiError()
	me.Add(
		"bol",
		errortypes.ErrInvalid,
		"BOL is already in use by shipment(s) with Pro Number(s): {0}",
		strings.Join(proNumbers, ", "),
	)

	if me.HasErrors() {
		return me
	}

	return nil
}

func (s *Effects) LogShipmentAction(
	entity shipmentTenantResource,
	actor kernelports.AuditActor,
	operation permission.Operation,
	previous any,
	current any,
	opts ...auditports.LogOption,
) error {
	params := &auditports.LogActionParams{
		Resource:       permission.ResourceShipment,
		ResourceID:     entity.GetID().String(),
		Operation:      operation,
		UserID:         actor.UserID,
		PrincipalType:  actor.PrincipalType,
		PrincipalID:    actor.PrincipalID,
		APIKeyID:       actor.APIKeyID,
		OrganizationID: entity.GetOrganizationID(),
		BusinessUnitID: entity.GetBusinessUnitID(),
	}
	if current != nil {
		params.CurrentState = jsonutils.MustToJSON(current)
	}
	if previous != nil {
		params.PreviousState = jsonutils.MustToJSON(previous)
	}

	return s.auditService.LogAction(params, opts...)
}

func (s *Effects) RecordShipmentEvent(
	ctx context.Context,
	params *shipmenteventports.RecordShipmentEventParams,
) {
	if params == nil {
		return
	}
	if err := s.eventService.Record(ctx, params); err != nil {
		s.l.Warn("failed to record shipment event", zap.Error(err))
	}
}

func (s *Effects) EmitStatusChangeEvent(
	ctx context.Context,
	original, updated *shipment.Shipment,
	actor kernelports.AuditActor,
) {
	if original == nil || updated == nil || original.Status == updated.Status {
		return
	}
	s.RecordShipmentEvent(ctx, shipmenteventservice.BuildStatusChanged(
		TenantRefForShipment(updated),
		updated,
		original.Status,
		actor,
	))
}

func (s *Effects) HydrateShipmentCommodityDetails(
	ctx context.Context,
	entity *shipment.Shipment,
) error {
	if entity == nil || len(entity.Commodities) == 0 {
		return nil
	}

	commodityIDs := shipmentvalidator.UniqueShipmentCommodityIDs(entity.Commodities)
	if len(commodityIDs) == 0 || s.commodityRepo == nil {
		return nil
	}

	commodities, err := s.commodityRepo.GetByIDs(ctx, commodityports.GetCommoditiesByIDsRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		CommodityIDs: commodityIDs,
	})
	if err != nil {
		return err
	}

	commodityMap := make(map[pulid.ID]*commodity.Commodity, len(commodities))
	for _, item := range commodities {
		if item == nil || item.ID.IsNil() {
			continue
		}
		commodityMap[item.ID] = item
	}

	for _, shipmentCommodity := range entity.Commodities {
		if shipmentCommodity == nil || shipmentCommodity.CommodityID.IsNil() {
			continue
		}
		if loaded, ok := commodityMap[shipmentCommodity.CommodityID]; ok {
			shipmentCommodity.Commodity = mergeCommodityDetails(shipmentCommodity.Commodity, loaded)
		}
	}

	return nil
}
