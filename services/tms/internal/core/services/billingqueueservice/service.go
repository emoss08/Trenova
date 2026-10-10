package billingqueueservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/watchtower"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/core/services/drivernotificationservice"
	"github.com/emoss08/trenova/internal/core/services/shipmentcommercial"
	"github.com/emoss08/trenova/internal/core/services/watchtowersources"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceadjustmentports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/orderports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateagreementports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/pkg/seqgen"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger               *zap.Logger
	DB                   kernelports.DBConnection
	Repo                 billingqueueports.BillingQueueRepository
	ShipmentRepo         shipmentports.ShipmentRepository
	ControlRepo          tenantports.ShipmentControlRepository
	CommentRepo          shipmentports.ShipmentCommentRepository
	CustomerRepo         customerports.CustomerRepository
	UserRepo             tenantports.UserRepository
	AdjustmentRepo       invoiceadjustmentports.InvoiceAdjustmentRepository
	InvoiceRepo          invoiceports.InvoiceRepository
	InvoiceSvc           invoiceports.InvoiceService
	ChargeAllocationRepo shipmentports.ChargeAllocationRepository
	Commercial           *shipmentcommercial.Calculator
	Generator            seqgen.Generator
	AuditService         auditports.AuditService
	Realtime             agentports.RealtimeService
	Validator            *Validator
	OrderDerivation      orderports.OrderDerivationService
	DetentionBilling     detentionports.DetentionBillingService
	AgentEvents          agentports.AgentEventPublisher `optional:"true"`
	// Watchtower puts an item that cannot be invoiced on the feed.
	Watchtower watchtowerports.WatchtowerProjector `optional:"true"`
	// The review reads: issues and activity, the shipment's paperwork and
	// detention evidence, the contract's accessorials, and the driver to ask
	// for a missing signature.
	ReviewRepo     billingqueueports.BillingQueueReviewRepository `optional:"true"`
	DocumentRepo   documentports.DocumentRepository               `optional:"true"`
	OccurrenceRepo detentionports.DetentionOccurrenceRepository   `optional:"true"`
	AgreementRepo  rateagreementports.RateAgreementRepository     `optional:"true"`
	DriverNotifier *drivernotificationservice.Service             `optional:"true"`
}

type service struct {
	l                    *zap.Logger
	db                   kernelports.DBConnection
	repo                 billingqueueports.BillingQueueRepository
	shipmentRepo         shipmentports.ShipmentRepository
	controlRepo          tenantports.ShipmentControlRepository
	commentRepo          shipmentports.ShipmentCommentRepository
	customerRepo         customerports.CustomerRepository
	userRepo             tenantports.UserRepository
	adjustmentRepo       invoiceadjustmentports.InvoiceAdjustmentRepository
	invoiceRepo          invoiceports.InvoiceRepository
	invoiceSvc           invoiceports.InvoiceService
	chargeAllocationRepo shipmentports.ChargeAllocationRepository
	commercial           *shipmentcommercial.Calculator
	generator            seqgen.Generator
	auditService         auditports.AuditService
	realtime             agentports.RealtimeService
	validator            *Validator
	orderDerivation      orderports.OrderDerivationService
	detentionBilling     detentionports.DetentionBillingService
	agentEvents          agentports.AgentEventPublisher
	watchtower           watchtowerports.WatchtowerProjector
	reviewRepo           billingqueueports.BillingQueueReviewRepository
	documentRepo         documentports.DocumentRepository
	occurrenceRepo       detentionports.DetentionOccurrenceRepository
	agreementRepo        rateagreementports.RateAgreementRepository
	drivers              driverNotifier
}

//nolint:gocritic // dependency injection
func New(p Params) billingqueueports.BillingQueueService {
	return &service{
		l:                    p.Logger.Named("service.billing-queue"),
		db:                   p.DB,
		repo:                 p.Repo,
		shipmentRepo:         p.ShipmentRepo,
		controlRepo:          p.ControlRepo,
		commentRepo:          p.CommentRepo,
		customerRepo:         p.CustomerRepo,
		userRepo:             p.UserRepo,
		adjustmentRepo:       p.AdjustmentRepo,
		invoiceRepo:          p.InvoiceRepo,
		invoiceSvc:           p.InvoiceSvc,
		chargeAllocationRepo: p.ChargeAllocationRepo,
		commercial:           p.Commercial,
		generator:            p.Generator,
		auditService:         p.AuditService,
		realtime:             p.Realtime,
		validator:            p.Validator,
		orderDerivation:      p.OrderDerivation,
		detentionBilling:     p.DetentionBilling,
		agentEvents:          p.AgentEvents,
		watchtower:           p.Watchtower,
		reviewRepo:           p.ReviewRepo,
		documentRepo:         p.DocumentRepo,
		occurrenceRepo:       p.OccurrenceRepo,
		agreementRepo:        p.AgreementRepo,
		drivers:              notifierOrNil(p.DriverNotifier),
	}
}

// NewReview is the same service seen as the review a biller does from one
// item; it shares every dependency with the queue it reviews.
func NewReview(svc billingqueueports.BillingQueueService) billingqueueports.BillingQueueReviewService {
	return svc.(*service) //nolint:errcheck,forcetypeassert // New always returns *service
}

func (s *service) List(
	ctx context.Context,
	req *billingqueueports.ListBillingQueueItemsRequest,
) (*pagination.ListResult[*billingqueue.BillingQueueItem], error) {
	return s.repo.List(ctx, req)
}

func (s *service) GetByID(
	ctx context.Context,
	req *billingqueueports.GetBillingQueueItemByIDRequest,
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

	holds, err := s.detentionHolds(ctx, item, req.TenantInfo)
	if err != nil {
		s.l.Warn("failed to read detention billing holds for billing queue item",
			zap.String("billingQueueItemId", item.ID.String()),
			zap.String("shipmentId", item.ShipmentID.String()),
			zap.Error(err),
		)
	}
	item.DetentionHolds = billingqueue.NewDetentionHolds(holds)
	s.attachReview(ctx, item, req.TenantInfo)

	return item, nil
}

// detentionHolds reads the detention charges holding the item's shipment off an
// invoice. A credit memo gives money back and bills no charge, and a posted or
// canceled item is past billing, so nothing holds either.
func (s *service) detentionHolds(
	ctx context.Context,
	item *billingqueue.BillingQueueItem,
	tenantInfo pagination.TenantInfo,
) ([]*detention.DetentionOccurrence, error) {
	if s.detentionBilling == nil || item == nil || item.ShipmentID.IsNil() ||
		item.BillType == billingqueue.BillTypeCreditMemo ||
		billingqueue.IsTerminalStatus(item.Status) {
		return nil, nil
	}

	return s.detentionBilling.HoldsForShipments(ctx, &detentionports.DetentionBillingHoldsRequest{
		TenantInfo:  tenantInfo,
		ShipmentIDs: []pulid.ID{item.ShipmentID},
	})
}

func (s *service) guardDetentionHolds(
	ctx context.Context,
	item *billingqueue.BillingQueueItem,
	tenantInfo pagination.TenantInfo,
) error {
	holds, err := s.detentionHolds(ctx, item, tenantInfo)
	if err != nil {
		return err
	}

	return detention.BillingHoldError(holds)
}

func (s *service) expandShipmentDetails(
	ctx context.Context,
	item *billingqueue.BillingQueueItem,
	tenantInfo pagination.TenantInfo,
) error {
	fullShipment, err := s.shipmentRepo.GetByID(ctx, &shipmentports.GetShipmentByIDRequest{
		ID:         item.ShipmentID,
		TenantInfo: tenantInfo,
		ShipmentOptions: shipmentports.ShipmentOptions{
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

	defer s.attachPayerShare(ctx, item, tenantInfo)

	cust, err := s.customerRepo.GetByID(ctx, customerports.GetCustomerByIDRequest{
		ID:         fullShipment.CustomerID,
		TenantInfo: tenantInfo,
		CustomerFilterOptions: customerports.CustomerFilterOptions{
			IncludeBillingProfile: true,
			IncludeEmailProfile:   true,
		},
	})
	if err == nil {
		item.Shipment.Customer = cust
	}

	// The payer's billing profile is what the queue reads for review rules and
	// auto-approval, and on a split shipment it is not the shipment customer's.
	switch {
	case item.BillToCustomerID.IsNil() || item.BillToCustomerID == fullShipment.CustomerID:
		item.BillToCustomer = item.Shipment.Customer
	default:
		payer, payerErr := s.customerRepo.GetByID(ctx, customerports.GetCustomerByIDRequest{
			ID:         item.BillToCustomerID,
			TenantInfo: tenantInfo,
			CustomerFilterOptions: customerports.CustomerFilterOptions{
				IncludeBillingProfile: true,
				IncludeEmailProfile:   true,
			},
		})
		if payerErr == nil {
			item.BillToCustomer = payer
		}
	}

	return nil
}

func (s *service) GetStats(
	ctx context.Context,
	req *billingqueueports.GetBillingQueueStatsRequest,
) (*billingqueueports.BillingQueueStats, error) {
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

	return &billingqueueports.BillingQueueStats{
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

// TransferToBilling queues a shipment for billing and returns the primary
// payer's item. A split-billed shipment gets one item per payer; callers that
// need every item use TransferToBillingItems.
func (s *service) TransferToBilling(
	ctx context.Context,
	req *billingqueueports.TransferToBillingRequest,
	actor *permissionports.RequestActor,
) (*billingqueue.BillingQueueItem, error) {
	result, err := s.TransferToBillingItems(ctx, req, actor)
	if err != nil {
		return nil, err
	}

	return result.Primary, nil
}

// TransferToBillingItems creates one billing queue item per payer of the
// shipment, all in one transaction so a split shipment is either fully queued or
// not queued at all. Each item carries its payer and that payer's share of the
// charges, which is what lets the queue, the statements and the invoices work
// per payer from here on.
func (s *service) TransferToBillingItems(
	ctx context.Context,
	req *billingqueueports.TransferToBillingRequest,
	actor *permissionports.RequestActor,
) (*billingqueueports.TransferToBillingResult, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Transfer to billing request is required",
		)
	}

	shp, err := s.shipmentForTransfer(ctx, req)
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

	resolution, err := shipment.ResolveShares(shp, shp.ChargeAllocations)
	if err != nil {
		return nil, err
	}

	for _, share := range resolution.Shares {
		if len(share.Charges) == 0 {
			continue
		}
		exists, existsErr := s.repo.ExistsByShipmentPayerAndType(
			ctx,
			req.TenantInfo,
			req.ShipmentID,
			share.PayerID,
			req.BillType,
		)
		if existsErr != nil {
			return nil, existsErr
		}
		if exists {
			return nil, errortypes.NewConflictError(
				"A billing queue item already exists for this shipment, payer and bill type",
			)
		}
	}

	entities := make([]*billingqueue.BillingQueueItem, 0, len(resolution.Shares))
	for _, share := range resolution.Shares {
		if len(share.Charges) == 0 {
			continue
		}
		number, numberErr := s.generateBillingNumber(
			ctx,
			req.BillType,
			req.TenantInfo.OrgID,
			req.TenantInfo.BuID,
		)
		if numberErr != nil {
			return nil, numberErr
		}

		entity := &billingqueue.BillingQueueItem{
			OrganizationID:       req.TenantInfo.OrgID,
			BusinessUnitID:       req.TenantInfo.BuID,
			ShipmentID:           req.ShipmentID,
			OrderID:              shp.OrderID,
			BillToCustomerID:     share.PayerID,
			AllocatedTotalAmount: share.TotalAmount,
			Status:               billingqueue.StatusReadyForReview,
			BillType:             req.BillType,
			Number:               number,
		}
		entity.SyncAllocatedMinor()
		if multiErr := s.validator.ValidateCreate(ctx, entity); multiErr != nil {
			return nil, multiErr
		}
		entities = append(entities, entity)
	}

	created := make([]*billingqueue.BillingQueueItem, 0, len(entities))
	err = s.db.WithTx(ctx, kernelports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		for _, entity := range entities {
			item, txErr := s.repo.Create(txCtx, entity)
			if txErr != nil {
				return txErr
			}
			created = append(created, item)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	result := &billingqueueports.TransferToBillingResult{
		Items: make([]*billingqueue.BillingQueueItem, 0, len(created)),
	}
	auditActor := actor.AuditActor()
	for _, item := range created {
		s.autoAssignDefaultBiller(ctx, item, item.BillToCustomerID, req.TenantInfo, actor)

		if s.shouldAutoApprove(req, item.BillToCustomerID) {
			item = s.autoApprove(ctx, item, req.TenantInfo, actor)
		}

		s.recordEvent(systemEvents(ctx), item, billingqueue.EventTransferred,
			"Queued when "+shipmentRef(shp)+" was transferred to billing", actor, nil)
		s.syncReview(ctx, item.ID, req.TenantInfo)

		s.logAction(
			item,
			auditActor,
			permission.OpCreate,
			nil,
			item,
			"Shipment transferred to billing queue",
		)
		s.publishInvalidation(ctx, item, auditActor, "created", item)

		result.Items = append(result.Items, item)
		if item.BillToCustomerID == shp.PayerID() {
			result.Primary = item
		}
	}
	if result.Primary == nil && len(result.Items) > 0 {
		result.Primary = result.Items[0]
	}

	return result, nil
}

func (s *service) shipmentForTransfer(
	ctx context.Context,
	req *billingqueueports.TransferToBillingRequest,
) (*shipment.Shipment, error) {
	if shp := req.DetailedShipment; shp != nil && shp.ID == req.ShipmentID &&
		shp.OrganizationID == req.TenantInfo.OrgID && shp.BusinessUnitID == req.TenantInfo.BuID {
		return shp, nil
	}

	return s.shipmentRepo.GetByID(ctx, &shipmentports.GetShipmentByIDRequest{
		ID:         req.ShipmentID,
		TenantInfo: req.TenantInfo,
		ShipmentOptions: shipmentports.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
}

// shouldAutoApprove reports whether one payer's item may skip review: either the
// whole transfer was cleared, or this payer is among those who opted in.
func (s *service) shouldAutoApprove(req *billingqueueports.TransferToBillingRequest, payerID pulid.ID) bool {
	if req.AutoApprove {
		return true
	}
	for _, id := range req.AutoApprovePayerIDs {
		if id == payerID {
			return true
		}
	}

	return false
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

// autoApprove clears a clean item straight through the queue.
//
// Whether the freight is clean was decided by the billing readiness evaluation
// before transfer — this only carries the decision out, so the requirement and
// rate gates are enforced in exactly one place.
//
// A failure here is logged and swallowed: the item is already in the queue, and
// leaving it at ReadyForReview for a biller is the right outcome when something
// unexpected stops the approval. Failing the transfer instead would strand the
// shipment outside billing altogether.
func (s *service) autoApprove(
	ctx context.Context,
	item *billingqueue.BillingQueueItem,
	tenantInfo pagination.TenantInfo,
	actor *permissionports.RequestActor,
) *billingqueue.BillingQueueItem {
	holds, err := s.detentionHolds(ctx, item, tenantInfo)
	if err != nil {
		s.l.Warn("failed to read detention billing holds; left for review",
			zap.String("billingQueueItemId", item.ID.String()),
			zap.Error(err),
		)
		return item
	}
	if len(holds) > 0 {
		s.l.Info("detention charges still need approval; billing queue item left for review",
			zap.String("billingQueueItemId", item.ID.String()),
			zap.String("shipmentId", item.ShipmentID.String()),
			zap.Int("heldCharges", len(holds)),
		)
		return item
	}

	approved, err := s.UpdateStatus(ctx, &billingqueueports.UpdateBillingQueueStatusRequest{
		ItemID:     item.ID,
		NewStatus:  billingqueue.StatusApproved,
		TenantInfo: tenantInfo,
	}, actor)
	if err != nil {
		s.l.Warn("failed to auto-approve billing queue item; left for review",
			zap.String("billingQueueItemId", item.ID.String()),
			zap.Error(err),
		)
		return item
	}

	s.l.Info("auto-approved billing queue item",
		zap.String("billingQueueItemId", item.ID.String()),
		zap.String("shipmentId", item.ShipmentID.String()),
	)

	return approved
}

func (s *service) autoAssignDefaultBiller(
	ctx context.Context,
	item *billingqueue.BillingQueueItem,
	customerID pulid.ID,
	tenantInfo pagination.TenantInfo,
	actor *permissionports.RequestActor,
) {
	cust, err := s.customerRepo.GetByID(ctx, customerports.GetCustomerByIDRequest{
		ID:         customerID,
		TenantInfo: tenantInfo,
		CustomerFilterOptions: customerports.CustomerFilterOptions{
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

	if _, err = s.AssignBiller(systemEvents(ctx), &billingqueueports.AssignBillerRequest{
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

// CheckBiller says whether a user can be assigned as a biller in the tenant,
// as AssignBiller will decide it, so a caller refuses a biller before
// proposing the assignment rather than after it is approved.
func (s *service) CheckBiller(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	billerID pulid.ID,
) error {
	_, err := s.biller(ctx, tenantInfo, billerID)

	return err
}

func (s *service) biller(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	billerID pulid.ID,
) (*tenant.User, error) {
	biller, err := s.userRepo.GetByID(ctx, tenantports.GetUserByIDRequest{
		LookupUserID: billerID,
		TenantInfo: pagination.TenantInfo{
			OrgID: tenantInfo.OrgID,
			BuID:  tenantInfo.BuID,
		},
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, errortypes.NewValidationError(
				"billerId",
				errortypes.ErrInvalid,
				"Biller user not found in the current tenant",
			)
		}

		return nil, err
	}

	return biller, nil
}

func (s *service) AssignBiller(
	ctx context.Context,
	req *billingqueueports.AssignBillerRequest,
	actor *permissionports.RequestActor,
) (*billingqueue.BillingQueueItem, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Assign biller request is required",
		)
	}

	entity, err := s.repo.GetByID(ctx, &billingqueueports.GetBillingQueueItemByIDRequest{
		ItemID:     req.ItemID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	previous := *entity
	if err = PlanAssignBiller(entity, req.BillerID, timeutils.NowUnix()); err != nil {
		return nil, err
	}

	biller, err := s.biller(ctx, req.TenantInfo, req.BillerID)
	if err != nil {
		return nil, err
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
	s.recordEvent(ctx, updated, billingqueue.EventAssigned,
		"Assigned "+userName(biller)+" as biller", actor,
		map[string]any{"billerId": req.BillerID.String()})
	s.publishInvalidation(ctx, updated, auditActor, "updated", updated)

	return updated, nil
}

func (s *service) UpdateStatus(
	ctx context.Context,
	req *billingqueueports.UpdateBillingQueueStatusRequest,
	actor *permissionports.RequestActor,
) (*billingqueue.BillingQueueItem, error) {
	if err := guardAgentTransition(actor, req.NewStatus); err != nil {
		return nil, err
	}

	var (
		updated      *billingqueue.BillingQueueItem
		previous     *billingqueue.BillingQueueItem
		createResult *invoiceports.CreateInvoiceFromBillingQueueResult
	)

	err := s.db.WithTx(ctx, kernelports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		entity, getErr := s.repo.GetByID(txCtx, &billingqueueports.GetBillingQueueItemByIDRequest{
			ItemID:     req.ItemID,
			TenantInfo: req.TenantInfo,
		})
		if getErr != nil {
			return getErr
		}

		prev := *entity
		previous = &prev

		startReviewForApproval(entity, req, timeutils.NowUnix())
		if transitionErr := checkStatusTransition(entity, req); transitionErr != nil {
			return transitionErr
		}

		if req.NewStatus == billingqueue.StatusApproved {
			if holdErr := s.guardDetentionHolds(txCtx, entity, req.TenantInfo); holdErr != nil {
				return holdErr
			}
			if issueErr := s.guardOpenIssues(txCtx, entity, req.TenantInfo); issueErr != nil {
				return issueErr
			}
		}
		if planErr := PlanStatusChange(entity, req, actor, timeutils.NowUnix()); planErr != nil {
			return planErr
		}

		if multiErr := s.validator.ValidateUpdate(txCtx, entity); multiErr != nil {
			return multiErr
		}

		updatedEntity, updateErr := s.repo.Update(txCtx, entity)
		if updateErr != nil {
			return updateErr
		}
		updated = updatedEntity

		if req.NewStatus == billingqueue.StatusApproved && s.invoiceSvc != nil {
			// Approving means the freight is verified, not that it bills today. A
			// statement customer's item is approved onto their statement and waits
			// for their cycle; everyone else is invoiced here as before.
			createResult, updateErr = s.invoiceSvc.CreateFromApprovedBillingQueueItem(
				txCtx,
				&invoiceports.CreateInvoiceFromBillingQueueRequest{
					BillingQueueItemID: updated.ID,
					TenantInfo:         req.TenantInfo,
					DeferToStatement:   true,
				},
				actor,
			)
			if updateErr != nil {
				return updateErr
			}
		}

		if updateErr = s.completeReplacementReview(txCtx, updated, req); updateErr != nil {
			return updateErr
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

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
	s.recordStatusEvent(ctx, previous, updated, createResult, actor)
	s.publishInvalidation(ctx, updated, auditActor, "updated", updated)
	s.publishStatusAgentEvent(ctx, updated)
	s.projectToWatchtower(ctx, updated)

	return updated, nil
}

func (s *service) publishStatusAgentEvent(
	ctx context.Context,
	item *billingqueue.BillingQueueItem,
) {
	var kind agent.EventKind
	switch item.Status {
	case billingqueue.StatusException:
		kind = agent.EventBillingQueueItemException
	case billingqueue.StatusOnHold:
		kind = agent.EventBillingQueueItemOnHold
	default:
		return
	}

	agentports.PublishAgentEvent(ctx, s.agentEvents, agentports.AgentEvent{
		Kind:      kind,
		SubjectID: item.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: item.OrganizationID,
			BuID:  item.BusinessUnitID,
		},
	})
}

// projectToWatchtower puts an item in exception on the feed and takes it
// off once it has moved on, whatever it moved to.
func (s *service) projectToWatchtower(
	ctx context.Context,
	item *billingqueue.BillingQueueItem,
) {
	if s.watchtower == nil || item == nil {
		return
	}

	tenant := pagination.TenantInfo{OrgID: item.OrganizationID, BuID: item.BusinessUnitID}
	if item.Status == billingqueue.StatusException {
		s.watchtower.Upsert(ctx, watchtowersources.DescribeBillingException(item))

		return
	}

	s.watchtower.Resolve(ctx, tenant, watchtower.SourceBillingException, item.ID.String())
}

func (s *service) completeReplacementReview(
	ctx context.Context,
	item *billingqueue.BillingQueueItem,
	req *billingqueueports.UpdateBillingQueueStatusRequest,
) error {
	if req.NewStatus != billingqueue.StatusApproved && req.NewStatus != billingqueue.StatusPosted {
		return nil
	}
	if !item.RequiresReplacementReview || item.SourceInvoiceAdjustmentID == nil ||
		item.SourceInvoiceAdjustmentID.IsNil() {
		return nil
	}

	adjustment, err := s.adjustmentRepo.GetByID(ctx, invoiceadjustmentports.GetInvoiceAdjustmentRequest{
		ID:         *item.SourceInvoiceAdjustmentID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return err
	}
	if adjustment.ReplacementReviewStatus != invoiceadjustment.ReplacementReviewStatusRequired {
		return nil
	}

	adjustment.ReplacementReviewStatus = invoiceadjustment.ReplacementReviewStatusCompleted
	_, err = s.adjustmentRepo.UpdateAdjustment(ctx, adjustment)
	return err
}

func (s *service) createOpsComment(
	ctx context.Context,
	entity *billingqueue.BillingQueueItem,
	actor *permissionports.RequestActor,
) {
	_, err := s.commentRepo.Create(ctx, SendBackComment(entity, actor.UserIDOrNil()))
	if err != nil {
		s.l.Warn("failed to create ops comment for billing exception",
			zap.String("shipmentId", entity.ShipmentID.String()),
			zap.Error(err),
		)
	}
}

// recomputeParentOrder rolls an edited leg's charge totals up to its commercial
// order (status + total) — billing-queue charge edits change total_charge_amount
// outside the shipment service, so the derivation port must be invoked directly.
func (s *service) recomputeParentOrder(ctx context.Context, shp *shipment.Shipment) error {
	if s.orderDerivation == nil || shp == nil || shp.OrderID.IsNil() {
		return nil
	}

	return s.orderDerivation.RecomputeOrder(ctx, pagination.TenantInfo{
		OrgID: shp.OrganizationID,
		BuID:  shp.BusinessUnitID,
	}, shp.OrderID)
}

func (s *service) UpdateCharges(
	ctx context.Context,
	req *billingqueueports.UpdateChargesRequest,
	actor *permissionports.RequestActor,
) (*billingqueue.BillingQueueItem, error) {
	return s.updateCharges(ctx, req, actor, false)
}

func (s *service) updateCharges(
	ctx context.Context,
	req *billingqueueports.UpdateChargesRequest,
	actor *permissionports.RequestActor,
	allowReady bool,
) (*billingqueue.BillingQueueItem, error) {
	plan, err := s.planChargeUpdateFrom(ctx, req, actor, true, allowReady)
	if err != nil {
		return nil, err
	}
	item, shp := plan.Item, plan.After

	if _, err = s.shipmentRepo.UpdateDerivedState(ctx, shp); err != nil {
		return nil, err
	}

	if err = s.recomputeParentOrder(ctx, shp); err != nil {
		return nil, err
	}

	auditActor := actor.AuditActor()
	s.logAction(
		item,
		auditActor,
		permission.OpUpdate,
		nil,
		nil,
		chargesUpdatedComment(plan.ConvertedSplits),
	)
	s.publishInvalidation(ctx, item, auditActor, "updated", item)
	// An edit made by settling a check is re-checked by the settlement once
	// it has recorded itself; checking here would clear the very issue it is
	// settling.
	if !allowReady {
		s.syncReview(ctx, req.ItemID, req.TenantInfo)
	}

	return s.repo.GetByID(ctx, &billingqueueports.GetBillingQueueItemByIDRequest{
		ItemID:     req.ItemID,
		TenantInfo: req.TenantInfo,
	})
}

// guardSiblingPayersUnposted refuses a charge edit once any other payer's share
// of the same shipment has been invoiced: the charges are shared, so changing
// them now would silently disagree with an invoice already in a customer's hands.
func (s *service) guardSiblingPayersUnposted(
	ctx context.Context,
	item *billingqueue.BillingQueueItem,
	tenantInfo pagination.TenantInfo,
) error {
	if item.ShipmentID.IsNil() {
		return nil
	}

	invoices, err := s.invoiceRepo.ListByShipmentIDs(
		ctx,
		invoiceports.ListInvoicesByShipmentIDsRequest{
			TenantInfo:  tenantInfo,
			ShipmentIDs: []pulid.ID{item.ShipmentID},
		},
	)
	if err != nil {
		return err
	}
	for _, inv := range invoices[item.ShipmentID] {
		if inv == nil || inv.CustomerID == item.BillToCustomerID || inv.IsAdjustmentArtifact {
			continue
		}
		return errortypes.NewValidationError(
			"additionalCharges",
			errortypes.ErrInvalidOperation,
			"Charges are locked: another payer's share of this shipment is already on invoice {0}",
			inv.Number,
		)
	}

	return nil
}

func (s *service) logAction(
	entity *billingqueue.BillingQueueItem,
	actor kernelports.AuditActor,
	op permission.Operation,
	previous any,
	current any,
	comment string,
) {
	params := &auditports.LogActionParams{
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

	opts := []auditports.LogOption{
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
	actor kernelports.AuditActor,
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
