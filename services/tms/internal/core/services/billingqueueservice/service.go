package billingqueueservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/core/services/shipmentcommercial"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/pkg/seqgen"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	DB           ports.DBConnection
	Repo         repositories.BillingQueueRepository
	ShipmentRepo repositories.ShipmentRepository
	ControlRepo  repositories.ShipmentControlRepository
	CommentRepo  repositories.ShipmentCommentRepository
	CustomerRepo repositories.CustomerRepository
	UserRepo     repositories.UserRepository
	InvoiceSvc   services.InvoiceService
	Commercial   *shipmentcommercial.Calculator
	Generator    seqgen.Generator
	AuditService services.AuditService
	Realtime     services.RealtimeService
	Validator    *Validator
}

type service struct {
	l            *zap.Logger
	db           ports.DBConnection
	repo         repositories.BillingQueueRepository
	shipmentRepo repositories.ShipmentRepository
	controlRepo  repositories.ShipmentControlRepository
	commentRepo  repositories.ShipmentCommentRepository
	customerRepo repositories.CustomerRepository
	userRepo     repositories.UserRepository
	invoiceSvc   services.InvoiceService
	commercial   *shipmentcommercial.Calculator
	generator    seqgen.Generator
	auditService services.AuditService
	realtime     services.RealtimeService
	validator    *Validator
}

//nolint:gocritic // dependency injection
func New(p Params) services.BillingQueueService {
	return &service{
		l:            p.Logger.Named("service.billing-queue"),
		db:           p.DB,
		repo:         p.Repo,
		shipmentRepo: p.ShipmentRepo,
		controlRepo:  p.ControlRepo,
		commentRepo:  p.CommentRepo,
		customerRepo: p.CustomerRepo,
		userRepo:     p.UserRepo,
		invoiceSvc:   p.InvoiceSvc,
		commercial:   p.Commercial,
		generator:    p.Generator,
		auditService: p.AuditService,
		realtime:     p.Realtime,
		validator:    p.Validator,
	}
}

func (s *service) List(
	ctx context.Context,
	req *repositories.ListBillingQueueItemsRequest,
) (*pagination.ListResult[*billingqueue.BillingQueueItem], error) {
	return s.repo.List(ctx, req)
}

func (s *service) GetByID(
	ctx context.Context,
	req *repositories.GetBillingQueueItemByIDRequest,
) (*billingqueue.BillingQueueItem, error) {
	item, err := s.repo.GetByID(ctx, req)
	if err != nil {
		return nil, err
	}

	if !req.ExpandShipmentDetails {
		return item, nil
	}

	if err = s.expandShipmentDetails(ctx, item, req.TenantInfo); err != nil {
		s.l.Warn("failed to expand shipment details for billing queue item",
			zap.String("billingQueueItemId", item.ID.String()),
			zap.String("shipmentId", item.ShipmentID.String()),
			zap.Error(err),
		)
	}

	return item, nil
}

func (s *service) expandShipmentDetails(
	ctx context.Context,
	item *billingqueue.BillingQueueItem,
	tenantInfo pagination.TenantInfo,
) error {
	fullShipment, err := s.shipmentRepo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID:         item.ShipmentID,
		TenantInfo: tenantInfo,
		ShipmentOptions: repositories.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		return err
	}

	item.Shipment = fullShipment
	if fullShipment.CustomerID.IsNil() {
		return nil
	}

	cust, err := s.customerRepo.GetByID(ctx, repositories.GetCustomerByIDRequest{
		ID:         fullShipment.CustomerID,
		TenantInfo: tenantInfo,
		CustomerFilterOptions: repositories.CustomerFilterOptions{
			IncludeBillingProfile: true,
		},
	})
	if err == nil {
		item.Shipment.Customer = cust
	}

	return nil
}

func (s *service) GetStats(
	ctx context.Context,
	req *repositories.GetBillingQueueStatsRequest,
) (*services.BillingQueueStats, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Stats request is required",
		)
	}

	counts, err := s.repo.GetStatusCounts(ctx, req)
	if err != nil {
		return nil, err
	}

	total := 0
	for _, c := range counts {
		total += c
	}

	return &services.BillingQueueStats{
		ReadyForReview: counts[billingqueue.StatusReadyForReview],
		InReview:       counts[billingqueue.StatusInReview],
		Approved:       counts[billingqueue.StatusApproved],
		Posted:         counts[billingqueue.StatusPosted],
		OnHold:         counts[billingqueue.StatusOnHold],
		Exception:      counts[billingqueue.StatusException],
		SentBackToOps:  counts[billingqueue.StatusSentBackToOps],
		Canceled:       counts[billingqueue.StatusCanceled],
		Total:          total,
	}, nil
}

func (s *service) TransferToBilling(
	ctx context.Context,
	req *services.TransferToBillingRequest,
	actor *services.RequestActor,
) (*billingqueue.BillingQueueItem, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Transfer to billing request is required",
		)
	}

	shp, err := s.shipmentRepo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID:         req.ShipmentID,
		TenantInfo: req.TenantInfo,
		ShipmentOptions: repositories.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		return nil, err
	}

	if shp.Status != shipment.StatusReadyToInvoice {
		return nil, errortypes.NewValidationError(
			"shipmentId",
			errortypes.ErrInvalidOperation,
			"Shipment must be in ReadyToInvoice status to transfer to billing",
		)
	}

	exists, err := s.repo.ExistsByShipmentAndType(ctx, req.TenantInfo, req.ShipmentID, req.BillType)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errortypes.NewConflictError(
			"A billing queue item already exists for this shipment and bill type",
		)
	}

	number, err := s.generateBillingNumber(
		ctx,
		req.BillType,
		req.TenantInfo.OrgID,
		req.TenantInfo.BuID,
	)
	if err != nil {
		return nil, err
	}

	entity := &billingqueue.BillingQueueItem{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		ShipmentID:     req.ShipmentID,
		Status:         billingqueue.StatusReadyForReview,
		BillType:       req.BillType,
		Number:         number,
	}

	if multiErr := s.validator.ValidateCreate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	created, err := s.repo.Create(ctx, entity)
	if err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	shp.BillingTransferStatus = shipment.BillingTransferReadyForReview
	shp.TransferredToBillingAt = &now
	if _, err = s.shipmentRepo.UpdateDerivedState(ctx, shp); err != nil {
		s.l.Warn("failed to update shipment billing tracking fields",
			zap.String("shipmentId", shp.ID.String()),
			zap.Error(err),
		)
	}

	s.autoAssignDefaultBiller(ctx, created, shp.CustomerID, req.TenantInfo, actor)

	auditActor := actor.AuditActor()
	s.logAction(
		created,
		auditActor,
		permission.OpCreate,
		nil,
		created,
		"Shipment transferred to billing queue",
	)
	s.publishInvalidation(ctx, created, auditActor, "created", created)

	return created, nil
}

func (s *service) generateBillingNumber(
	ctx context.Context,
	billType billingqueue.BillType,
	orgID, buID pulid.ID,
) (string, error) {
	switch billType {
	case billingqueue.BillTypeInvoice:
		return s.generator.GenerateInvoiceNumber(ctx, orgID, buID, "", "")
	case billingqueue.BillTypeCreditMemo:
		return s.generator.GenerateCreditMemoNumber(ctx, orgID, buID, "", "")
	case billingqueue.BillTypeDebitMemo:
		return s.generator.GenerateDebitMemoNumber(ctx, orgID, buID, "", "")
	default:
		return "", errortypes.NewValidationError(
			"billType",
			errortypes.ErrInvalid,
			"Unsupported bill type for number generation",
		)
	}
}

func (s *service) autoAssignDefaultBiller(
	ctx context.Context,
	item *billingqueue.BillingQueueItem,
	customerID pulid.ID,
	tenantInfo pagination.TenantInfo,
	actor *services.RequestActor,
) {
	cust, err := s.customerRepo.GetByID(ctx, repositories.GetCustomerByIDRequest{
		ID:         customerID,
		TenantInfo: tenantInfo,
		CustomerFilterOptions: repositories.CustomerFilterOptions{
			IncludeBillingProfile: true,
		},
	})
	if err != nil || cust.BillingProfile == nil || cust.BillingProfile.DefaultBillerID == nil ||
		cust.BillingProfile.DefaultBillerID.IsNil() {
		return
	}

	s.l.Info("auto-assigning default biller from customer billing profile",
		zap.String("billingQueueItemId", item.ID.String()),
		zap.String("customerId", customerID.String()),
		zap.String("defaultBillerId", cust.BillingProfile.DefaultBillerID.String()),
	)

	if _, err = s.AssignBiller(ctx, &services.AssignBillerRequest{
		ItemID:     item.ID,
		BillerID:   *cust.BillingProfile.DefaultBillerID,
		TenantInfo: tenantInfo,
	}, actor); err != nil {
		s.l.Warn("failed to auto-assign default biller from customer billing profile",
			zap.String("billingQueueItemId", item.ID.String()),
			zap.String("customerId", customerID.String()),
			zap.Error(err),
		)
	}
}

func (s *service) AssignBiller(
	ctx context.Context,
	req *services.AssignBillerRequest,
	actor *services.RequestActor,
) (*billingqueue.BillingQueueItem, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Assign biller request is required",
		)
	}

	entity, err := s.repo.GetByID(ctx, &repositories.GetBillingQueueItemByIDRequest{
		ItemID:     req.ItemID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	if billingqueue.IsTerminalStatus(entity.Status) {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"Cannot assign a biller to a billing queue item in "+string(entity.Status)+" status",
		)
	}

	if _, err = s.userRepo.GetByID(ctx, repositories.GetUserByIDRequest{
		LookupUserID: req.BillerID,
		TenantInfo: pagination.TenantInfo{
			OrgID: req.TenantInfo.OrgID,
			BuID:  req.TenantInfo.BuID,
		},
	}); err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, errortypes.NewValidationError(
				"billerId",
				errortypes.ErrInvalid,
				"Biller user not found in the current tenant",
			)
		}

		return nil, err
	}

	previous := *entity
	entity.AssignedBillerID = &req.BillerID

	if entity.Status == billingqueue.StatusReadyForReview {
		entity.Status = billingqueue.StatusInReview
		now := timeutils.NowUnix()
		entity.ReviewStartedAt = &now
	}

	updated, err := s.repo.Update(ctx, entity)
	if err != nil {
		return nil, err
	}

	auditActor := actor.AuditActor()
	s.logAction(
		updated,
		auditActor,
		permission.OpAssign,
		&previous,
		updated,
		"Biller assigned to billing queue item",
	)
	s.publishInvalidation(ctx, updated, auditActor, "updated", updated)

	return updated, nil
}

func (s *service) UpdateStatus(
	ctx context.Context,
	req *services.UpdateBillingQueueStatusRequest,
	actor *services.RequestActor,
) (*billingqueue.BillingQueueItem, error) {
	var (
		updated      *billingqueue.BillingQueueItem
		previous     *billingqueue.BillingQueueItem
		createResult *services.CreateInvoiceFromBillingQueueResult
	)

	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		entity, getErr := s.repo.GetByID(txCtx, &repositories.GetBillingQueueItemByIDRequest{
			ItemID:     req.ItemID,
			TenantInfo: req.TenantInfo,
		})
		if getErr != nil {
			return getErr
		}

		if !billingqueue.IsAllowedTransition(entity.Status, req.NewStatus) {
			return errortypes.NewValidationError(
				"status",
				errortypes.ErrInvalidOperation,
				"Cannot transition from "+string(entity.Status)+" to "+string(req.NewStatus),
			)
		}

		prev := *entity
		previous = &prev
		entity.Status = req.NewStatus

		s.applyStatusFields(entity, req, actor)

		if multiErr := s.validator.ValidateUpdate(txCtx, entity); multiErr != nil {
			return multiErr
		}

		updatedEntity, updateErr := s.repo.Update(txCtx, entity)
		if updateErr != nil {
			return updateErr
		}
		updated = updatedEntity

		if req.NewStatus == billingqueue.StatusApproved && s.invoiceSvc != nil {
			createResult, updateErr = s.invoiceSvc.CreateFromApprovedBillingQueueItem(
				txCtx,
				&services.CreateInvoiceFromBillingQueueRequest{
					BillingQueueItemID: updated.ID,
					TenantInfo:         req.TenantInfo,
				},
				actor,
			)
			if updateErr != nil {
				return updateErr
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	s.syncShipmentBillingStatus(ctx, updated, req.NewStatus)

	if req.NewStatus == billingqueue.StatusSentBackToOps {
		s.createOpsComment(ctx, updated, actor)
	}

	if createResult != nil && createResult.AutoPost && createResult.Invoice != nil {
		if err = s.invoiceSvc.EnqueueAutoPost(ctx, createResult.Invoice, actor); err != nil {
			s.l.Warn("failed to enqueue invoice auto-post workflow",
				zap.String("billingQueueItemId", updated.ID.String()),
				zap.String("invoiceId", createResult.Invoice.ID.String()),
				zap.Error(err),
			)
		}
	}

	auditActor := actor.AuditActor()
	s.logAction(
		updated,
		auditActor,
		permission.OpUpdate,
		previous,
		updated,
		"Billing queue item status updated to "+string(req.NewStatus),
	)
	s.publishInvalidation(ctx, updated, auditActor, "updated", updated)

	return updated, nil
}

func (s *service) syncShipmentBillingStatus(
	ctx context.Context,
	entity *billingqueue.BillingQueueItem,
	newStatus billingqueue.Status,
) {
	shp, err := s.shipmentRepo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID: entity.ShipmentID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		ShipmentOptions: repositories.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		s.l.Warn("failed to fetch shipment for billing status sync",
			zap.String("shipmentId", entity.ShipmentID.String()),
			zap.Error(err),
		)
		return
	}

	shp.BillingTransferStatus = shipmentBillingTransferStatus(newStatus)

	if _, err = s.shipmentRepo.UpdateDerivedState(ctx, shp); err != nil {
		s.l.Warn("failed to sync shipment billing transfer status",
			zap.String("shipmentId", shp.ID.String()),
			zap.Error(err),
		)
	}
}

func shipmentBillingTransferStatus(status billingqueue.Status) shipment.BillingTransferStatus {
	switch status {
	case billingqueue.StatusReadyForReview:
		return shipment.BillingTransferReadyForReview
	case billingqueue.StatusInReview:
		return shipment.BillingTransferInReview
	case billingqueue.StatusApproved:
		return shipment.BillingTransferApproved
	case billingqueue.StatusPosted:
		return shipment.BillingTransferApproved
	case billingqueue.StatusOnHold:
		return shipment.BillingTransferOnHold
	case billingqueue.StatusException:
		return shipment.BillingTransferException
	case billingqueue.StatusSentBackToOps:
		return shipment.BillingTransferSentBackToOps
	case billingqueue.StatusCanceled:
		return shipment.BillingTransferCanceled
	default:
		return shipment.BillingTransferNone
	}
}

func (s *service) createOpsComment(
	ctx context.Context,
	entity *billingqueue.BillingQueueItem,
	actor *services.RequestActor,
) {
	reasonLabel := string(*entity.ExceptionReasonCode)
	comment := "Sent back from billing: " + reasonLabel
	if entity.ExceptionNotes != "" {
		comment += "\n\n" + entity.ExceptionNotes
	}

	_, err := s.commentRepo.Create(ctx, &shipment.ShipmentComment{
		OrganizationID: entity.OrganizationID,
		BusinessUnitID: entity.BusinessUnitID,
		ShipmentID:     entity.ShipmentID,
		UserID:         actor.UserID,
		Comment:        comment,
		Type:           shipment.CommentTypeBilling,
		Visibility:     shipment.CommentVisibilityOperations,
		Priority:       shipment.CommentPriorityHigh,
		Source:         shipment.CommentSourceSystem,
	})
	if err != nil {
		s.l.Warn("failed to create ops comment for billing exception",
			zap.String("shipmentId", entity.ShipmentID.String()),
			zap.Error(err),
		)
	}
}

func (s *service) UpdateCharges(
	ctx context.Context,
	req *services.UpdateChargesRequest,
	actor *services.RequestActor,
) (*billingqueue.BillingQueueItem, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Update charges request is required",
		)
	}

	item, err := s.repo.GetByID(ctx, &repositories.GetBillingQueueItemByIDRequest{
		ItemID:     req.ItemID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	if item.Status != billingqueue.StatusInReview {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"Charges can only be edited when the item is in InReview status",
		)
	}

	shp, err := s.shipmentRepo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID:         item.ShipmentID,
		TenantInfo: req.TenantInfo,
		ShipmentOptions: repositories.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		return nil, err
	}

	if req.FormulaTemplateID != nil && !req.FormulaTemplateID.IsNil() {
		shp.FormulaTemplateID = *req.FormulaTemplateID

		control, _ := s.controlRepo.Get(ctx, repositories.GetShipmentControlRequest{
			TenantInfo: req.TenantInfo,
		})
		if err = s.commercial.Recalculate(ctx, shp, control, actor.UserID); err != nil {
			return nil, errortypes.NewValidationError(
				"formulaTemplateId",
				errortypes.ErrInvalid,
				"Failed to recalculate charges with the selected formula template",
			)
		}
	} else if req.BaseRate != nil {
		shp.BaseRate = decimal.NewNullDecimal(*req.BaseRate)

		control, _ := s.controlRepo.Get(ctx, repositories.GetShipmentControlRequest{
			TenantInfo: req.TenantInfo,
		})
		if err = s.commercial.Recalculate(ctx, shp, control, actor.UserID); err != nil {
			return nil, errortypes.NewValidationError(
				"baseRate",
				errortypes.ErrInvalid,
				"Failed to recalculate charges with updated base rate",
			)
		}
	}

	if req.AdditionalCharges != nil {
		shp.AdditionalCharges = req.AdditionalCharges
	}

	freight := shp.FreightChargeAmount.Decimal
	otherTotal := shipmentcommercial.CalculateAdditionalCharges(shp.AdditionalCharges, freight)
	shp.OtherChargeAmount = decimal.NewNullDecimal(otherTotal)
	shp.TotalChargeAmount = decimal.NewNullDecimal(freight.Add(otherTotal))

	if _, err = s.shipmentRepo.UpdateDerivedState(ctx, shp); err != nil {
		return nil, err
	}

	auditActor := actor.AuditActor()
	s.logAction(
		item,
		auditActor,
		permission.OpUpdate,
		nil,
		nil,
		"Charges updated from billing queue",
	)
	s.publishInvalidation(ctx, item, auditActor, "updated", item)

	return s.repo.GetByID(ctx, &repositories.GetBillingQueueItemByIDRequest{
		ItemID:     req.ItemID,
		TenantInfo: req.TenantInfo,
	})
}

func (s *service) applyStatusFields(
	entity *billingqueue.BillingQueueItem,
	req *services.UpdateBillingQueueStatusRequest,
	actor *services.RequestActor,
) {
	now := timeutils.NowUnix()

	switch req.NewStatus {
	case billingqueue.StatusInReview:
		if entity.ReviewStartedAt == nil {
			entity.ReviewStartedAt = &now
		}
		entity.ReviewCompletedAt = nil
	case billingqueue.StatusApproved:
		entity.ReviewCompletedAt = &now
		if req.ReviewNotes != "" {
			entity.ReviewNotes = req.ReviewNotes
		}
	case billingqueue.StatusPosted:
		entity.ReviewCompletedAt = &now
	case billingqueue.StatusSentBackToOps, billingqueue.StatusException:
		entity.ExceptionReasonCode = req.ExceptionReasonCode
		entity.ExceptionNotes = req.ExceptionNotes
	case billingqueue.StatusCanceled:
		userID := actor.UserID
		entity.CanceledByID = &userID
		entity.CanceledAt = &now
		entity.CancelReason = req.CancelReason
	case billingqueue.StatusReadyForReview:
		entity.ExceptionReasonCode = nil
		entity.ExceptionNotes = ""
	case billingqueue.StatusOnHold:
		if req.ReviewNotes != "" {
			entity.ReviewNotes = req.ReviewNotes
		}
	}
}

func (s *service) logAction(
	entity *billingqueue.BillingQueueItem,
	actor services.AuditActor,
	op permission.Operation,
	previous any,
	current any,
	comment string,
) {
	params := &services.LogActionParams{
		Resource:       permission.ResourceBillingQueue,
		ResourceID:     entity.ID.String(),
		Operation:      op,
		UserID:         actor.UserID,
		APIKeyID:       actor.APIKeyID,
		PrincipalType:  actor.PrincipalType,
		PrincipalID:    actor.PrincipalID,
		OrganizationID: entity.OrganizationID,
		BusinessUnitID: entity.BusinessUnitID,
	}
	if current != nil {
		params.CurrentState = jsonutils.MustToJSON(current)
	}
	if previous != nil {
		params.PreviousState = jsonutils.MustToJSON(previous)
	}

	opts := []services.LogOption{
		auditservice.WithComment(comment),
		auditservice.WithMetadata(map[string]any{
			"shipmentId":     entity.ShipmentID.String(),
			"billingQueueId": entity.ID.String(),
			"status":         string(entity.Status),
		}),
	}
	if previous != nil && current != nil {
		opts = append(opts, auditservice.WithDiff(previous, current))
	}

	if err := s.auditService.LogAction(params, opts...); err != nil {
		s.l.Error("failed to log billing queue action", zap.Error(err))
	}
}

func (s *service) publishInvalidation(
	ctx context.Context,
	entity *billingqueue.BillingQueueItem,
	actor services.AuditActor,
	action string,
	payload any,
) {
	err := realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: entity.OrganizationID,
		BusinessUnitID: entity.BusinessUnitID,
		ActorUserID:    actor.UserID,
		ActorType:      actor.PrincipalType,
		ActorID:        actor.PrincipalID,
		ActorAPIKeyID:  actor.APIKeyID,
		Resource:       permission.ResourceBillingQueue.String(),
		Action:         action,
		RecordID:       entity.ID,
		Entity:         payload,
	})
	if err != nil {
		s.l.Warn("failed to publish billing queue invalidation", zap.Error(err))
	}
}
