//nolint:gocognit // existing legacy workflow/API shape is intentionally kept stable
package invoiceservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/order"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	servicesports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/accountingcontrolpolicyservice"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/core/services/billingcontrolpolicyservice"
	"github.com/emoss08/trenova/internal/core/services/invoicelines"
	"github.com/emoss08/trenova/internal/core/services/notificationservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs/billingjobs"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/pkg/seqgen"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger               *zap.Logger
	DB                   ports.DBConnection
	Repo                 repositories.InvoiceRepository
	BillingQueueRepo     repositories.BillingQueueRepository
	ShipmentRepo         repositories.ShipmentRepository
	OrderRepo            repositories.OrderRepository
	CustomerRepo         repositories.CustomerRepository
	OrganizationRepo     repositories.OrganizationRepository
	DocumentTypeRepo     repositories.DocumentTypeRepository
	CustomerLedgerRepo   repositories.CustomerLedgerProjectionRepository
	BillingRepo          repositories.BillingControlRepository
	AccountingRepo       repositories.AccountingControlRepository
	JournalRepo          repositories.JournalPostingRepository
	AdjustmentRepo       repositories.InvoiceAdjustmentRepository
	ChargeAllocationRepo repositories.ChargeAllocationRepository
	AdjustmentService    servicesports.InvoiceAdjustmentService `optional:"true"`
	NotificationService  *notificationservice.Service
	AccessorialRepo      repositories.AccessorialChargeRepository
	EmailRepo            repositories.EmailRepository
	Validator            *Validator
	AuditService         servicesports.AuditService
	DocumentService      servicesports.InvoiceDocumentService `optional:"true"`
	EmailService         servicesports.EmailService
	Templates            servicesports.DocumentTemplateResolver
	ContextBuilder       *ContextBuilder
	Realtime             servicesports.RealtimeService
	WorkflowStarter      servicesports.WorkflowStarter
	SequenceGenerator    seqgen.Generator
	SequenceProvider     seqgen.FormatProvider
	OrderDerivation      servicesports.OrderDerivationService
	AccountingPolicy     *accountingcontrolpolicyservice.Service
	BillingPolicy        *billingcontrolpolicyservice.Service

	EDIPartnerRepo              repositories.EDIPartnerRepository                `optional:"true"`
	EDIDocumentProfileRepo      repositories.EDIPartnerDocumentProfileRepository `optional:"true"`
	EDICommunicationProfileRepo repositories.EDICommunicationProfileRepository   `optional:"true"`
	LateChargeRepo              repositories.LateChargeRepository                `optional:"true"`
}

type Service struct {
	l                    *zap.Logger
	db                   ports.DBConnection
	repo                 repositories.InvoiceRepository
	billingQueueRepo     repositories.BillingQueueRepository
	shipmentRepo         repositories.ShipmentRepository
	orderRepo            repositories.OrderRepository
	customerRepo         repositories.CustomerRepository
	organizationRepo     repositories.OrganizationRepository
	documentTypeRepo     repositories.DocumentTypeRepository
	customerLedgerRepo   repositories.CustomerLedgerProjectionRepository
	billingRepo          repositories.BillingControlRepository
	accountingRepo       repositories.AccountingControlRepository
	journalRepo          repositories.JournalPostingRepository
	adjustmentRepo       repositories.InvoiceAdjustmentRepository
	chargeAllocationRepo repositories.ChargeAllocationRepository
	adjustmentService    servicesports.InvoiceAdjustmentService
	notificationService  *notificationservice.Service
	accessorialRepo      repositories.AccessorialChargeRepository
	emailRepo            repositories.EmailRepository
	validator            *Validator
	auditService         servicesports.AuditService
	documentService      servicesports.InvoiceDocumentService
	emailService         servicesports.EmailService
	templates            servicesports.DocumentTemplateResolver
	contextBuilder       *ContextBuilder
	realtime             servicesports.RealtimeService
	workflowStarter      servicesports.WorkflowStarter
	sequenceGenerator    seqgen.Generator
	sequenceProvider     seqgen.FormatProvider
	orderDerivation      servicesports.OrderDerivationService
	accountingPolicy     *accountingcontrolpolicyservice.Service
	billingPolicy        *billingcontrolpolicyservice.Service

	ediPartnerRepo              repositories.EDIPartnerRepository
	ediDocumentProfileRepo      repositories.EDIPartnerDocumentProfileRepository
	ediCommunicationProfileRepo repositories.EDICommunicationProfileRepository
	lateChargeRepo              repositories.LateChargeRepository
}

type existingInvoiceLookupResult struct {
	Invoice *invoice.Invoice
	Found   bool
}

type invoiceDependencies struct {
	Shipment       *shipment.Shipment
	Order          *order.Order
	Customer       *customer.Customer
	BillingControl *tenant.BillingControl
	// Share is the payer's slice of the shipment's charges; Shipper is the
	// ordering customer when it is not the payer.
	Share       *shipment.PayerShare
	Shipper     *customer.Customer
	IsSplitBill bool
}

type postedBillingQueueResult struct {
	Previous *billingqueue.BillingQueueItem
	Updated  *billingqueue.BillingQueueItem
}

var _ servicesports.InvoiceService = (*Service)(nil)

func New(p Params) servicesports.InvoiceService { //nolint:gocritic // stable API shape
	return NewService(p)
}

// NewService returns the concrete service. The run service needs
// CreateConsolidated, which is not on the InvoiceService port, so the container
// provides this and derives the port from it rather than building twice.
func NewService(p Params) *Service { //nolint:gocritic // mirrors New
	return &Service{
		l:                           p.Logger.Named("service.invoice"),
		db:                          p.DB,
		repo:                        p.Repo,
		billingQueueRepo:            p.BillingQueueRepo,
		shipmentRepo:                p.ShipmentRepo,
		orderRepo:                   p.OrderRepo,
		customerRepo:                p.CustomerRepo,
		organizationRepo:            p.OrganizationRepo,
		documentTypeRepo:            p.DocumentTypeRepo,
		customerLedgerRepo:          p.CustomerLedgerRepo,
		billingRepo:                 p.BillingRepo,
		accountingRepo:              p.AccountingRepo,
		journalRepo:                 p.JournalRepo,
		adjustmentRepo:              p.AdjustmentRepo,
		chargeAllocationRepo:        p.ChargeAllocationRepo,
		adjustmentService:           p.AdjustmentService,
		notificationService:         p.NotificationService,
		accessorialRepo:             p.AccessorialRepo,
		emailRepo:                   p.EmailRepo,
		validator:                   p.Validator,
		auditService:                p.AuditService,
		documentService:             p.DocumentService,
		emailService:                p.EmailService,
		templates:                   p.Templates,
		contextBuilder:              p.ContextBuilder,
		realtime:                    p.Realtime,
		workflowStarter:             p.WorkflowStarter,
		sequenceGenerator:           p.SequenceGenerator,
		sequenceProvider:            p.SequenceProvider,
		orderDerivation:             p.OrderDerivation,
		accountingPolicy:            p.AccountingPolicy,
		billingPolicy:               p.BillingPolicy,
		ediPartnerRepo:              p.EDIPartnerRepo,
		ediDocumentProfileRepo:      p.EDIDocumentProfileRepo,
		ediCommunicationProfileRepo: p.EDICommunicationProfileRepo,
		lateChargeRepo:              p.LateChargeRepo,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListInvoicesRequest,
) (*pagination.ListResult[*invoice.Invoice], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) ListConnection(
	ctx context.Context,
	req *repositories.ListInvoiceConnectionRequest,
) (*pagination.CursorListResult[*invoice.Invoice], error) {
	return s.repo.ListConnection(ctx, req)
}

func (s *Service) GetByID(
	ctx context.Context,
	req repositories.GetInvoiceByIDRequest,
) (*invoice.Invoice, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) GetByIDs(
	ctx context.Context,
	req repositories.GetInvoicesByIDsRequest,
) ([]*invoice.Invoice, error) {
	return s.repo.GetByIDs(ctx, req)
}

func (s *Service) CreateFromApprovedBillingQueueItem(
	ctx context.Context,
	req *servicesports.CreateInvoiceFromBillingQueueRequest,
	actor *servicesports.RequestActor,
) (*servicesports.CreateInvoiceFromBillingQueueResult, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Request is required",
		)
	}

	existingLookup, err := s.getExistingInvoiceByBillingQueueItem(ctx, req)
	if err != nil {
		return nil, err
	}
	if existingLookup.Found {
		return &servicesports.CreateInvoiceFromBillingQueueResult{
			Invoice:  existingLookup.Invoice,
			AutoPost: s.resolveAutoPost(existingLookup.Invoice.Customer),
		}, nil
	}

	item, err := s.getApprovedBillingQueueItem(ctx, req)
	if err != nil {
		return nil, err
	}

	dependencies, err := s.getInvoiceDependencies(ctx, req, item)
	if err != nil {
		return nil, err
	}

	// Approving a statement customer's freight puts it on their statement; it does
	// not bill it. Checked before the cadence guard because this is not a
	// deviation to be justified — it is the schedule working.
	if req.DeferToStatement && isStatementBilled(dependencies.Customer) {
		return &servicesports.CreateInvoiceFromBillingQueueResult{
			DeferredToStatement: true,
		}, nil
	}

	// The last gate before an invoice exists, so it cannot be bypassed by any of
	// the paths that reach this one.
	if err = guardStatementCadence(dependencies.Customer, req.OffCycleReason); err != nil {
		return nil, err
	}

	entity := s.buildInvoiceEntity(&buildInvoiceParams{
		Anchor:         item,
		Scope:          invoice.ScopeShipment,
		Customer:       dependencies.Customer,
		Control:        dependencies.BillingControl,
		Legs:           legsFromShipment(dependencies.Shipment),
		Order:          dependencies.Order,
		OffCycleReason: offCycleReasonFor(dependencies.Customer, req.OffCycleReason),
		Shipper:        dependencies.Shipper,
		Shares:         sharesForLeg(dependencies.Shipment, dependencies.Share),
		IsSplitBill:    dependencies.IsSplitBill,
	})
	if entity == nil {
		return nil, errortypes.NewValidationError(
			"billingQueueItemId",
			errortypes.ErrInvalidOperation,
			"Billing queue item has no shipment or adjustment context to bill from",
		)
	}
	if multiErr := s.validator.ValidateCreate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	created, err := s.repo.Create(ctx, entity)
	if err != nil {
		return nil, err
	}

	if item.IsAdjustmentOrigin {
		s.syncAdjustmentLineage(ctx, created, item)
	}

	autoPost := s.shouldAutoPost(ctx, created.OrganizationID, dependencies.Customer)
	auditActor := actor.AuditActor()
	s.logAction(
		created,
		auditActor,
		permission.OpCreate,
		nil,
		created,
		"Invoice created from approved billing queue item",
	)
	s.publishInvalidation(ctx, created, auditActor, "created", created)

	return &servicesports.CreateInvoiceFromBillingQueueResult{
		Invoice:  created,
		AutoPost: autoPost,
	}, nil
}

func (s *Service) syncAdjustmentLineage(
	ctx context.Context,
	created *invoice.Invoice,
	item *billingqueue.BillingQueueItem,
) {
	if created == nil || item == nil {
		return
	}
	if item.SourceInvoiceID != nil && item.SourceInvoiceID.IsNotNil() {
		sourceInvoice, err := s.repo.GetByID(ctx, repositories.GetInvoiceByIDRequest{
			ID: *item.SourceInvoiceID,
			TenantInfo: pagination.TenantInfo{
				OrgID: item.OrganizationID,
				BuID:  item.BusinessUnitID,
			},
		})
		if err == nil && sourceInvoice != nil {
			sourceInvoice.SupersededByInvoiceID = created.ID
			sourceInvoice.CorrectionGroupID = created.CorrectionGroupID
			if _, err = s.repo.Update(ctx, sourceInvoice); err != nil {
				s.l.Warn("failed to update source invoice supersession linkage", zap.Error(err))
			}
		}
	}

	if s.adjustmentRepo != nil && item.CorrectionGroupID != nil &&
		item.CorrectionGroupID.IsNotNil() {
		group, err := s.adjustmentRepo.GetCorrectionGroup(
			ctx,
			repositories.GetCorrectionGroupRequest{
				ID: *item.CorrectionGroupID,
				TenantInfo: pagination.TenantInfo{
					OrgID: item.OrganizationID,
					BuID:  item.BusinessUnitID,
				},
			},
		)
		if err == nil && group != nil {
			group.CurrentInvoiceID = created.ID
			if _, err = s.adjustmentRepo.UpdateCorrectionGroup(ctx, group); err != nil {
				s.l.Warn("failed to update correction group current invoice", zap.Error(err))
			}
		}
	}
}

//nolint:nestif // existing validation flow mirrors business rule nesting
func (s *Service) Post( //nolint:funlen // legacy workflow
	ctx context.Context,
	req *servicesports.PostInvoiceRequest,
	actor *servicesports.RequestActor,
) (*invoice.Invoice, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Request is required",
		)
	}

	var posted *invoice.Invoice
	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		entity, getErr := s.repo.GetByID(txCtx, repositories.GetInvoiceByIDRequest{
			ID:         req.InvoiceID,
			TenantInfo: req.TenantInfo,
		})
		if getErr != nil {
			return getErr
		}

		if s.billingRepo != nil {
			control, controlErr := s.billingRepo.GetByOrgID(txCtx, entity.OrganizationID)
			if controlErr != nil {
				if req.TriggeredBy == billingcontrolpolicyservice.AutoPostInvoiceTrigger {
					return controlErr
				}
			} else {
				if policyErr := s.billingPolicyService().
					ValidateInvoicePosting(control, req.TriggeredBy); policyErr != nil {
					return policyErr
				}
			}
		}

		auditActor := actor.AuditActor()

		if entity.Status == invoice.StatusVoided {
			return errortypes.NewValidationError(
				"invoiceId",
				errortypes.ErrInvalidOperation,
				"Voided invoices cannot be posted",
			)
		}

		if entity.Status == invoice.StatusPosted {
			queueUpdate, err := s.markBillingQueueItemPosted(txCtx, entity, req.TenantInfo)
			if err != nil {
				return err
			}

			s.logBillingQueuePosted(queueUpdate, auditActor)
			s.publishBillingQueueInvalidation(txCtx, queueUpdate, auditActor)

			posted = entity
			return nil
		}

		previous := *entity
		now := timeutils.NowUnix()

		if multiErr := s.validator.ValidatePost(
			txCtx,
			entity,
			req.TenantInfo,
			now,
		); multiErr != nil {
			return multiErr
		}

		entity.Status = invoice.StatusPosted
		entity.PostedAt = &now

		if multiErr := s.validator.ValidateUpdate(txCtx, entity); multiErr != nil {
			return multiErr
		}

		updated, updateErr := s.repo.Update(txCtx, entity)
		if updateErr != nil {
			return updateErr
		}

		legs, shipErr := s.markInvoicedLegs(txCtx, entity, now, req.TenantInfo)
		if shipErr != nil {
			return shipErr
		}

		if derivErr := s.recomputeOrdersForLegs(
			txCtx,
			req.TenantInfo,
			entity.OrderID,
			legs,
		); derivErr != nil {
			return derivErr
		}

		queueUpdate, updateErr := s.markBillingQueueItemPosted(txCtx, updated, req.TenantInfo)
		if updateErr != nil {
			return updateErr
		}

		if postErr := s.createInvoiceJournalPosting(txCtx, updated, actor); postErr != nil {
			return postErr
		}

		posted = updated

		s.logAction(updated, auditActor, permission.OpUpdate, &previous, updated, "Invoice posted")
		s.publishInvalidation(txCtx, updated, auditActor, "updated", updated)
		s.logBillingQueuePosted(queueUpdate, auditActor)
		s.publishBillingQueueInvalidation(txCtx, queueUpdate, auditActor)
		s.notifyReconciliationWarning(txCtx, updated, legs)

		return nil
	})
	if err != nil {
		return nil, err
	}
	s.enqueueEDIAfterPost(ctx, posted, req.TenantInfo, actor)

	return posted, nil
}

func (s *Service) notifyReconciliationWarning(
	ctx context.Context,
	entity *invoice.Invoice,
	legs []*shipment.Shipment,
) {
	if s.accountingRepo == nil || s.notificationService == nil || entity == nil || len(legs) == 0 {
		return
	}

	control, err := s.accountingRepo.GetByOrgID(ctx, entity.OrganizationID)
	if err != nil || control == nil {
		return
	}

	if control.ReconciliationMode != tenant.ReconciliationModeWarnOnly ||
		!control.NotifyOnReconciliationException {
		return
	}

	expectedTotal := reconciliationExpectedTotal(entity, legs)
	discrepancy := entity.TotalAmount.Sub(expectedTotal).Abs()
	if !discrepancy.GreaterThan(control.ReconciliationToleranceAmount) {
		return
	}

	if _, err = s.notificationService.Create(ctx, &notification.Notification{
		OrganizationID: entity.OrganizationID,
		BusinessUnitID: &entity.BusinessUnitID,
		EventType:      "invoice_reconciliation_warning",
		Priority:       notification.PriorityMedium,
		Channel:        notification.ChannelGlobal,
		Title:          "Invoice posted with reconciliation warning",
		Message:        "A posted invoice exceeded the organization reconciliation tolerance and requires follow-up.",
		Data: map[string]any{
			"invoiceTotal":      entity.TotalAmount.String(),
			"expectedTotal":     expectedTotal.String(),
			"toleranceAmount":   control.ReconciliationToleranceAmount.String(),
			"discrepancyAmount": discrepancy.String(),
		},
		RelatedEntities: reconciliationRelatedEntities(entity),
		Source:          "invoiceservice.Post",
	}); err != nil {
		s.l.Warn("failed to create reconciliation warning notification", zap.Error(err))
	}
}

// recomputeOrdersForLegs re-derives every order touched by a posting: the invoice's
// own order (grouped path) plus each leg's parent order (single-shipment path). The
// dedupe matters because all grouped-invoice legs share one order.
func (s *Service) recomputeOrdersForLegs(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	invoiceOrderID pulid.ID,
	legs []*shipment.Shipment,
) error {
	if s.orderDerivation == nil {
		return nil
	}

	orderIDs := make(map[pulid.ID]struct{}, 1+len(legs))
	if !invoiceOrderID.IsNil() {
		orderIDs[invoiceOrderID] = struct{}{}
	}
	for _, leg := range legs {
		if leg == nil || leg.OrderID.IsNil() {
			continue
		}
		orderIDs[leg.OrderID] = struct{}{}
	}

	for orderID := range orderIDs {
		if err := s.orderDerivation.RecomputeOrder(ctx, tenantInfo, orderID); err != nil {
			return err
		}
	}

	return nil
}

// markInvoicedLegs flips every shipment billed by this invoice to Invoiced. For a
// single-shipment invoice that is one leg; for a grouped (order) invoice it is the set
// of legs carried on the invoice lines. It returns the legs for downstream
// reconciliation notification.
//
// A split-billed leg is invoiced only once every payer's share has posted: while
// another payer's item is still open the leg keeps its status and only records
// that it has been billed.
func (s *Service) markInvoicedLegs(
	ctx context.Context,
	entity *invoice.Invoice,
	now int64,
	tenantInfo pagination.TenantInfo,
) ([]*shipment.Shipment, error) {
	legs, err := loadInvoiceLegs(ctx, s.shipmentRepo, entity, tenantInfo, true)
	if err != nil {
		return nil, err
	}

	stillOpen, err := s.legsWithOpenSiblingItems(ctx, entity, legs, tenantInfo)
	if err != nil {
		return nil, err
	}

	for _, shp := range legs {
		shp.BilledAt = &now
		if _, open := stillOpen[shp.ID]; !open {
			shp.Status = shipment.StatusInvoiced
		}
		if _, err = s.shipmentRepo.UpdateDerivedState(ctx, shp); err != nil {
			return nil, err
		}
	}

	return legs, nil
}

// legsWithOpenSiblingItems reports which legs still have an invoice item that
// belongs to another payer and has not posted. Items this invoice bills are not
// siblings: they post in the same transaction, a moment after this runs.
func (s *Service) legsWithOpenSiblingItems(
	ctx context.Context,
	entity *invoice.Invoice,
	legs []*shipment.Shipment,
	tenantInfo pagination.TenantInfo,
) (map[pulid.ID]struct{}, error) {
	open := make(map[pulid.ID]struct{})
	if len(legs) == 0 {
		return open, nil
	}

	legIDs := make([]pulid.ID, 0, len(legs))
	for _, shp := range legs {
		legIDs = append(legIDs, shp.ID)
	}
	active, err := s.billingQueueRepo.ListActiveInvoiceItemsByShipmentIDs(
		ctx,
		&repositories.ListActiveInvoiceItemsRequest{
			TenantInfo:  tenantInfo,
			ShipmentIDs: legIDs,
		},
	)
	if err != nil {
		return nil, err
	}

	for shipmentID, items := range active {
		for _, item := range items {
			if item == nil || item.ID == entity.BillingQueueItemID ||
				item.InvoiceID == entity.ID {
				continue
			}
			open[shipmentID] = struct{}{}
			break
		}
	}

	return open, nil
}

func (s *Service) markBillingQueueItemPosted(
	ctx context.Context,
	entity *invoice.Invoice,
	tenantInfo pagination.TenantInfo,
) (*postedBillingQueueResult, error) {
	item, err := s.billingQueueRepo.GetByID(ctx, &repositories.GetBillingQueueItemByIDRequest{
		ItemID:     entity.BillingQueueItemID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	if item.Status == billingqueue.StatusPosted {
		return &postedBillingQueueResult{
			Updated: item,
		}, nil
	}

	previous := *item
	item.Status = billingqueue.StatusPosted

	updated, err := s.billingQueueRepo.Update(ctx, item)
	if err != nil {
		return nil, err
	}

	// Settle every sibling billing-queue item this invoice bills so none linger in
	// Approved, without touching items that belong to other still-draft invoices.
	// The order and shipment ids are only a fallback for rows written before the
	// invoice back-link existed.
	if _, sweepErr := s.billingQueueRepo.MarkPostedForInvoice(
		ctx,
		&repositories.MarkPostedForInvoiceRequest{
			TenantInfo:  tenantInfo,
			InvoiceID:   entity.ID,
			OrderID:     entity.OrderID,
			ShipmentIDs: entity.LegShipmentIDs(),
			PayerID:     entity.CustomerID,
		},
	); sweepErr != nil {
		return nil, sweepErr
	}

	return &postedBillingQueueResult{
		Previous: &previous,
		Updated:  updated,
	}, nil
}

func (s *Service) getExistingInvoiceByBillingQueueItem(
	ctx context.Context,
	req *servicesports.CreateInvoiceFromBillingQueueRequest,
) (*existingInvoiceLookupResult, error) {
	existing, err := s.repo.GetByBillingQueueItemID(
		ctx,
		repositories.GetInvoiceByBillingQueueItemIDRequest{
			BillingQueueItemID: req.BillingQueueItemID,
			TenantInfo:         req.TenantInfo,
		},
	)
	if err == nil {
		return &existingInvoiceLookupResult{
			Invoice: existing,
			Found:   true,
		}, nil
	}
	if errortypes.IsNotFoundError(err) {
		return &existingInvoiceLookupResult{}, nil
	}

	return nil, err
}

func (s *Service) getApprovedBillingQueueItem(
	ctx context.Context,
	req *servicesports.CreateInvoiceFromBillingQueueRequest,
) (*billingqueue.BillingQueueItem, error) {
	item, err := s.billingQueueRepo.GetByID(ctx, &repositories.GetBillingQueueItemByIDRequest{
		ItemID:                req.BillingQueueItemID,
		TenantInfo:            req.TenantInfo,
		ExpandShipmentDetails: true,
	})
	if err != nil {
		return nil, err
	}

	if item.Status != billingqueue.StatusApproved {
		return nil, errortypes.NewValidationError(
			"billingQueueItemId",
			errortypes.ErrInvalidOperation,
			"Only approved billing queue items can create invoices",
		)
	}

	return item, nil
}

// resolveDependencyShipment resolves the leg (when the queue item carries one) and the
// billing customer. Order-scoped adjustment items (credit memo / replacement for a
// grouped invoice) have no single shipment; their customer comes from the source
// invoice.
func (s *Service) resolveDependencyShipment(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	item *billingqueue.BillingQueueItem,
) (*shipment.Shipment, pulid.ID, error) {
	if !item.ShipmentID.IsNil() {
		shp := item.Shipment
		if shp == nil || shp.ID != item.ShipmentID || shp.CustomerID.IsNil() {
			var err error
			shp, err = s.shipmentRepo.GetByID(
				ctx,
				expandedShipmentByIDRequest(item.ShipmentID, tenantInfo),
			)
			if err != nil {
				return nil, pulid.Nil, err
			}
		}
		if item.BillToCustomerID.IsNotNil() {
			return shp, item.BillToCustomerID, nil
		}
		return shp, shp.PayerID(), nil
	}

	if item.BillToCustomerID.IsNotNil() &&
		(item.SourceInvoiceID == nil || item.SourceInvoiceID.IsNil()) {
		return nil, item.BillToCustomerID, nil
	}

	if item.SourceInvoiceID == nil || item.SourceInvoiceID.IsNil() {
		return nil, pulid.Nil, errortypes.NewValidationError(
			"billingQueueItemId",
			errortypes.ErrInvalidOperation,
			"Billing queue item has no shipment or source invoice to bill from",
		)
	}
	src, err := s.repo.GetByID(ctx, repositories.GetInvoiceByIDRequest{
		ID:         *item.SourceInvoiceID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, pulid.Nil, err
	}
	return nil, src.CustomerID, nil
}

func (s *Service) getInvoiceDependencies(
	ctx context.Context,
	req *servicesports.CreateInvoiceFromBillingQueueRequest,
	item *billingqueue.BillingQueueItem,
) (*invoiceDependencies, error) {
	shp, customerID, err := s.resolveDependencyShipment(ctx, req.TenantInfo, item)
	if err != nil {
		return nil, err
	}
	if err = invoicelines.HydrateAccessorials(ctx, s.accessorialRepo, req.TenantInfo, shp); err != nil {
		return nil, err
	}

	cus, err := s.customerRepo.GetByID(ctx, repositories.GetCustomerByIDRequest{
		ID:         customerID,
		TenantInfo: req.TenantInfo,
		CustomerFilterOptions: repositories.CustomerFilterOptions{
			IncludeBillingProfile: true,
			IncludeState:          true,
		},
	})
	if err != nil {
		return nil, err
	}

	control, err := s.billingRepo.GetByOrgID(ctx, req.TenantInfo.OrgID)
	if err != nil && !errortypes.IsNotFoundError(err) {
		return nil, err
	}

	orderID := item.OrderID
	if orderID.IsNil() && shp != nil {
		orderID = shp.OrderID
	}
	var ord *order.Order
	if !orderID.IsNil() {
		ord, err = s.orderRepo.GetByID(ctx, repositories.GetOrderByIDRequest{
			ID:         orderID,
			TenantInfo: req.TenantInfo,
		})
		if err != nil && !errortypes.IsNotFoundError(err) {
			return nil, err
		}
	}

	deps := &invoiceDependencies{
		Shipment:       shp,
		Order:          ord,
		Customer:       cus,
		BillingControl: control,
	}
	if shp == nil {
		return deps, nil
	}

	resolution, err := shipment.ResolveShares(shp, shp.ChargeAllocations)
	if err != nil {
		return nil, err
	}
	deps.Share = resolution.ShareFor(cus.ID)
	if deps.Share == nil {
		return nil, errortypes.NewValidationError(
			"billingQueueItemId",
			errortypes.ErrInvalidOperation,
			"{0} no longer has a share of shipment {1}",
			cus.Name,
			shp.ProNumber,
		)
	}
	deps.IsSplitBill = resolution.IsSplit
	if shp.CustomerID != cus.ID {
		deps.Shipper, err = s.customerRepo.GetByID(ctx, repositories.GetCustomerByIDRequest{
			ID:         shp.CustomerID,
			TenantInfo: req.TenantInfo,
		})
		if err != nil {
			return nil, err
		}
	}

	return deps, nil
}

func (s *Service) EnqueueAutoPost(
	ctx context.Context,
	entity *invoice.Invoice,
	actor *servicesports.RequestActor,
) error {
	if entity == nil {
		return errortypes.NewValidationError(
			"invoice",
			errortypes.ErrRequired,
			"Invoice is required",
		)
	}

	if !s.workflowStarter.Enabled() {
		return servicesports.ErrWorkflowStarterDisabled
	}

	auditActor := actor.AuditActor()
	payload := &billingjobs.AutoPostInvoicePayload{
		BasePayload: temporaltype.BasePayload{
			OrganizationID: entity.OrganizationID,
			BusinessUnitID: entity.BusinessUnitID,
			UserID:         auditActor.UserID,
			Timestamp:      timeutils.NowUnix(),
		},
		InvoiceID:     entity.ID,
		PrincipalType: auditActor.PrincipalType,
		PrincipalID:   auditActor.PrincipalID,
		APIKeyID:      auditActor.APIKeyID,
	}

	workflowID := fmt.Sprintf(
		"invoice-auto-post-%s-%s-%s",
		entity.OrganizationID.String(),
		entity.BusinessUnitID.String(),
		entity.ID.String(),
	)

	_, err := s.workflowStarter.StartWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID:            workflowID,
			TaskQueue:     temporaltype.TaskQueueBilling.String(),
			StaticSummary: "Auto-post invoice " + entity.Number,
		},
		billingjobs.AutoPostInvoiceWorkflowName,
		payload,
	)
	return err
}

// buildInvoiceParams is what an invoice is built from, whatever its scope.
//
// Grouped by a struct rather than passed positionally because the shapes differ
// only in which identity the header carries, and a six-argument signature that
// means different things per scope is how the two builders drifted apart in the
// first place.
type buildInvoiceParams struct {
	Anchor       *billingqueue.BillingQueueItem
	Scope        invoice.Scope
	Customer     *customer.Customer
	Control      *tenant.BillingControl
	Legs         []*shipment.Shipment
	Order        *order.Order
	OrderCharges []*order.OrderCharge
	RunID        pulid.ID
	PeriodStart  *int64
	PeriodEnd    *int64
	InvoiceDate  int64
	Number       string
	// OffCycleReason is set only when this invoice takes freight off a customer's
	// periodic statement.
	OffCycleReason string
	// Shipper is the ordering customer when the payer is somebody else. Shares
	// are each leg's slice for the payer; a leg with no entry is billed in full.
	// OrderChargeShare is the payer's slice of the order's own charges.
	Shipper          *customer.Customer
	Shares           map[pulid.ID]*shipment.PayerShare
	OrderChargeShare *shipment.PayerShare
	IsSplitBill      bool
}

// buildInvoiceEntity builds every shape of invoice from its legs.
//
// One builder, because a single-shipment invoice is the one-leg case of a
// grouped one and a consolidated invoice is the many-order case. Only the header
// differs: which identity it carries, and whether it names a period.
func (s *Service) buildInvoiceEntity(p *buildInvoiceParams) *invoice.Invoice {
	if p.Anchor.IsAdjustmentOrigin {
		if entity := s.buildAdjustmentOriginInvoiceEntity(
			p.Anchor,
			firstLeg(p.Legs),
			p.Customer,
			p.Control,
		); entity != nil {
			if p.Order != nil {
				entity.OrderID = p.Order.ID
				entity.OrderNumber = p.Order.OrderNumber
			}
			return entity
		}
	}

	if len(p.Legs) == 0 {
		return nil
	}

	invoiceDate := p.InvoiceDate
	if invoiceDate == 0 {
		invoiceDate = timeutils.NowUnix()
	}
	paymentTerm := resolvePaymentTerm(p.Customer, p.Control)
	if paymentTerm == "" {
		paymentTerm = invoice.PaymentTermNet30
	}

	number := p.Number
	if number == "" {
		number = p.Anchor.Number
	}

	lines := make([]*invoice.InvoiceLine, 0, len(p.Legs))
	nextLineNumber := 1
	for _, leg := range p.Legs {
		legLines := invoicelines.ForShipmentShare(
			p.Anchor.BillType,
			leg,
			p.Shares[leg.ID],
			nextLineNumber,
		)
		lines = append(lines, legLines...)
		nextLineNumber += len(legLines)
	}

	// Order-level charges (customs brokerage, an order-wide fuel surcharge) carry
	// no leg attribution, which is what puts them in the trailing section when the
	// invoice is rendered.
	switch {
	case p.OrderChargeShare != nil:
		for _, charge := range p.OrderChargeShare.Charges {
			if charge.Kind != shipment.ChargeAllocationKindOrderCharge {
				continue
			}
			lines = append(lines, invoicelines.ForOrderChargeShare(
				p.Anchor.BillType,
				charge,
				nextLineNumber,
			))
			nextLineNumber++
		}
	default:
		for _, charge := range p.OrderCharges {
			if charge == nil {
				continue
			}
			amount := invoicelines.SignedAmount(p.Anchor.BillType, charge.Amount)
			lines = append(lines, &invoice.InvoiceLine{
				LineNumber:  nextLineNumber,
				Type:        invoice.InvoiceLineTypeAccessorial,
				Description: charge.Description,
				Quantity:    decimal.NewFromInt(1),
				UnitPrice:   amount,
				Amount:      amount,
			})
			nextLineNumber++
		}
	}

	entity := &invoice.Invoice{
		OrganizationID:     p.Anchor.OrganizationID,
		BusinessUnitID:     p.Anchor.BusinessUnitID,
		BillingQueueItemID: p.Anchor.ID,
		Scope:              p.Scope,
		CustomerID:         p.Customer.ID,
		Number:             number,
		BillType:           p.Anchor.BillType,
		Status:             invoice.StatusDraft,
		PaymentTerm:        paymentTerm,
		CurrencyCode:       billingCurrencyFromCustomer(p.Customer),
		InvoiceDate:        invoiceDate,
		DueDate:            invoice.DueDateFromPaymentTerm(invoiceDate, paymentTerm),
		ShipmentCount:      len(p.Legs),
		BillToName:         p.Customer.Name,
		BillToCode:         p.Customer.Code,
		BillToAddressLine1: p.Customer.AddressLine1,
		BillToAddressLine2: p.Customer.AddressLine2,
		BillToCity:         p.Customer.City,
		BillToPostalCode:   p.Customer.PostalCode,
		AppliedAmount:      decimal.Zero,
		SettlementStatus:   invoice.SettlementStatusUnpaid,
		DisputeStatus:      invoice.DisputeStatusNone,
		OffCycleReason:     p.OffCycleReason,
		IsSplitBill:        p.IsSplitBill,
		Lines:              lines,
	}

	applyInvoiceDetail(entity, p.Customer)
	applyInvoiceScopeHeader(entity, p)
	applyShipper(entity, p)

	if p.Customer.State != nil {
		entity.BillToState = p.Customer.State.Abbreviation
		entity.BillToCountry = p.Customer.State.CountryName
	}

	syncInvoiceTotalsFromLines(entity)
	entity.SyncMinorAmounts()

	return entity
}

// applyInvoiceScopeHeader stamps the identity fields that belong to this scope,
// and only those. A grouped invoice deliberately carries no header shipment, and
// a consolidated one carries neither shipment nor order — the lines hold the
// attribution and Scope says so outright.
func applyInvoiceScopeHeader(entity *invoice.Invoice, p *buildInvoiceParams) {
	switch p.Scope {
	case invoice.ScopeShipment:
		leg := firstLeg(p.Legs)
		entity.ShipmentID = leg.ID
		entity.ShipmentProNumber = leg.ProNumber
		entity.ShipmentBOL = leg.BOL
		entity.ServiceDate = serviceDateFromShipment(leg)
		entity.OrderID = orderIDFromOrder(p.Order)
		entity.OrderNumber = orderNumberFromOrder(p.Order)
	case invoice.ScopeOrder:
		entity.OrderID = orderIDFromOrder(p.Order)
		entity.OrderNumber = orderNumberFromOrder(p.Order)
	case invoice.ScopeConsolidated:
		entity.InvoiceRunID = p.RunID
		entity.PeriodStart = p.PeriodStart
		entity.PeriodEnd = p.PeriodEnd
	case invoice.ScopeAdjustment:
	}
}

// applyShipper records on whose behalf a single-shipment or order invoice bills
// when the payer is not the ordering customer. A consolidated invoice spans
// shippers, so its lines carry that attribution instead of the header.
func applyShipper(entity *invoice.Invoice, p *buildInvoiceParams) {
	if p.Shipper == nil || p.Shipper.ID == entity.CustomerID {
		return
	}
	switch p.Scope {
	case invoice.ScopeShipment, invoice.ScopeOrder:
		entity.ShipperCustomerID = p.Shipper.ID
	case invoice.ScopeConsolidated, invoice.ScopeAdjustment:
	}
}

func sharesForLeg(
	shp *shipment.Shipment,
	share *shipment.PayerShare,
) map[pulid.ID]*shipment.PayerShare {
	if shp == nil || share == nil {
		return nil
	}

	return map[pulid.ID]*shipment.PayerShare{shp.ID: share}
}

func legsFromShipment(shp *shipment.Shipment) []*shipment.Shipment {
	if shp == nil {
		return nil
	}
	return []*shipment.Shipment{shp}
}

// applyInvoiceDetail copies the customer's presentation preference onto the
// invoice, so how it reads is fixed at the moment it was billed.
func applyInvoiceDetail(entity *invoice.Invoice, cus *customer.Customer) {
	entity.Detail = customer.InvoiceDetailDetailed
	entity.SectionBy = customer.InvoiceSectionKeyShipment

	profile := billingProfileOf(cus)
	if profile == nil {
		return
	}
	if profile.InvoiceDetail.IsValid() {
		entity.Detail = profile.InvoiceDetail
	}
	if profile.SectionBy.IsValid() {
		entity.SectionBy = profile.SectionBy
	}
}

func firstLeg(legs []*shipment.Shipment) *shipment.Shipment {
	if len(legs) == 0 {
		return nil
	}
	return legs[0]
}

func orderIDFromOrder(ord *order.Order) pulid.ID {
	if ord == nil {
		return pulid.Nil
	}
	return ord.ID
}

func orderNumberFromOrder(ord *order.Order) string {
	if ord == nil {
		return ""
	}
	return ord.OrderNumber
}

type adjustmentInvoiceContext struct {
	ReplacementLines   []*invoice.InvoiceLine `json:"replacementLines"`
	SubtotalAmount     decimal.Decimal        `json:"subtotalAmount"`
	OtherAmount        decimal.Decimal        `json:"otherAmount"`
	TotalAmount        decimal.Decimal        `json:"totalAmount"`
	AccountingDate     int64                  `json:"accountingDate"`
	SourceInvoiceID    pulid.ID               `json:"sourceInvoiceId"`
	CorrectionGroupID  pulid.ID               `json:"correctionGroupId"`
	SourceAdjustmentID pulid.ID               `json:"sourceAdjustmentId"`
}

func (s *Service) buildAdjustmentOriginInvoiceEntity(
	item *billingqueue.BillingQueueItem,
	shp *shipment.Shipment,
	cus *customer.Customer,
	control *tenant.BillingControl,
) *invoice.Invoice {
	if len(item.AdjustmentContext) == 0 {
		return nil
	}

	raw, err := sonic.Marshal(item.AdjustmentContext)
	if err != nil {
		return nil
	}

	var ctx adjustmentInvoiceContext
	if err = sonic.Unmarshal(raw, &ctx); err != nil {
		return nil
	}

	invoiceDate := ctx.AccountingDate
	if invoiceDate == 0 {
		invoiceDate = timeutils.NowUnix()
	}
	paymentTerm := resolvePaymentTerm(cus, control)
	if paymentTerm == "" {
		paymentTerm = invoice.PaymentTermNet30
	}
	lines := ctx.ReplacementLines
	if len(lines) == 0 {
		if shp == nil {
			return nil
		}
		lines = buildInvoiceLines(item.BillType, shp)
	}

	entity := &invoice.Invoice{
		OrganizationID:            item.OrganizationID,
		BusinessUnitID:            item.BusinessUnitID,
		BillingQueueItemID:        item.ID,
		Scope:                     invoice.ScopeAdjustment,
		CustomerID:                cus.ID,
		Number:                    item.Number,
		BillType:                  item.BillType,
		Status:                    invoice.StatusDraft,
		PaymentTerm:               paymentTerm,
		CurrencyCode:              billingCurrencyFromCustomer(cus),
		InvoiceDate:               invoiceDate,
		DueDate:                   invoice.DueDateFromPaymentTerm(invoiceDate, paymentTerm),
		BillToName:                cus.Name,
		BillToCode:                cus.Code,
		BillToAddressLine1:        cus.AddressLine1,
		BillToAddressLine2:        cus.AddressLine2,
		BillToCity:                cus.City,
		BillToPostalCode:          cus.PostalCode,
		AppliedAmount:             decimal.Zero,
		SettlementStatus:          invoice.SettlementStatusUnpaid,
		DisputeStatus:             invoice.DisputeStatusNone,
		CorrectionGroupID:         ctx.CorrectionGroupID,
		SupersedesInvoiceID:       ctx.SourceInvoiceID,
		SourceInvoiceAdjustmentID: ctx.SourceAdjustmentID,
		Lines:                     lines,
	}
	if shp != nil {
		entity.ShipmentID = shp.ID
		entity.ShipmentProNumber = shp.ProNumber
		entity.ShipmentBOL = shp.BOL
		entity.ServiceDate = serviceDateFromShipment(shp)
		if shp.CustomerID != cus.ID {
			entity.ShipperCustomerID = shp.CustomerID
		}
	}
	for _, line := range lines {
		if line != nil && line.IsPartialShare() {
			entity.IsSplitBill = true
			break
		}
	}
	if cus.State != nil {
		entity.BillToState = cus.State.Abbreviation
		entity.BillToCountry = cus.State.CountryName
	}
	syncInvoiceTotalsFromLines(entity)
	entity.SyncMinorAmounts()
	return entity
}

func syncInvoiceTotalsFromLines(entity *invoice.Invoice) {
	entity.SubtotalAmount = sumLinesByType(entity.Lines, invoice.InvoiceLineTypeFreight)
	entity.OtherAmount = sumLinesByType(entity.Lines, invoice.InvoiceLineTypeAccessorial)
	entity.TotalAmount = sumLinesByType(entity.Lines, "")
}

func sumLinesByType(
	lines []*invoice.InvoiceLine,
	lineType invoice.InvoiceLineType,
) decimal.Decimal {
	total := decimal.Zero
	for _, line := range lines {
		if line == nil {
			continue
		}
		if lineType == "" || line.Type == lineType {
			total = total.Add(line.Amount)
		}
	}
	return total
}

func buildInvoiceLines(
	billType billingqueue.BillType,
	shp *shipment.Shipment,
) []*invoice.InvoiceLine {
	return invoicelines.ForShipment(billType, shp, 1)
}

func paymentTermFromCustomer(cus *customer.Customer) invoice.PaymentTerm {
	if cus == nil || cus.BillingProfile == nil {
		return ""
	}

	return invoice.PaymentTerm(cus.BillingProfile.PaymentTerm)
}

func paymentTermFromBillingControl(control *tenant.BillingControl) invoice.PaymentTerm {
	if control == nil {
		return ""
	}

	return invoice.PaymentTerm(control.DefaultPaymentTerm)
}

func resolvePaymentTerm(
	cus *customer.Customer,
	control *tenant.BillingControl,
) invoice.PaymentTerm {
	if term := paymentTermFromCustomer(cus); term != "" {
		return term
	}

	return paymentTermFromBillingControl(control)
}

func billingCurrencyFromCustomer(cus *customer.Customer) string {
	if cus == nil || cus.BillingProfile == nil ||
		strings.TrimSpace(cus.BillingProfile.BillingCurrency) == "" {
		return "USD"
	}

	return cus.BillingProfile.BillingCurrency
}

func serviceDateFromShipment(shp *shipment.Shipment) *int64 {
	switch {
	case shp.ActualDeliveryDate != nil:
		return shp.ActualDeliveryDate
	case shp.ActualShipDate != nil:
		return shp.ActualShipDate
	default:
		return nil
	}
}

func (s *Service) shouldAutoPost(
	ctx context.Context,
	orgID pulid.ID,
	cus *customer.Customer,
) bool {
	control, err := s.billingRepo.GetByOrgID(ctx, orgID)
	if err == nil && control != nil {
		return s.billingPolicyService().CanAutoPostInvoice(control, cus)
	}

	return s.resolveAutoPost(cus)
}

func (s *Service) resolveAutoPost(cus *customer.Customer) bool {
	return cus != nil && cus.BillingProfile != nil && cus.BillingProfile.AutoBill
}

func (s *Service) accountingPolicyService() *accountingcontrolpolicyservice.Service {
	if s.accountingPolicy != nil {
		return s.accountingPolicy
	}
	return accountingcontrolpolicyservice.New(
		accountingcontrolpolicyservice.Params{Logger: zap.NewNop()},
	)
}

func (s *Service) billingPolicyService() *billingcontrolpolicyservice.Service {
	if s.billingPolicy != nil {
		return s.billingPolicy
	}
	return billingcontrolpolicyservice.New(billingcontrolpolicyservice.Params{Logger: zap.NewNop()})
}

func (s *Service) logAction(
	entity *invoice.Invoice,
	actor servicesports.AuditActor,
	op permission.Operation,
	previous any,
	current any,
	comment string,
) {
	params := &servicesports.LogActionParams{
		Resource:       permission.ResourceInvoice,
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

	opts := []servicesports.LogOption{
		auditservice.WithComment(comment),
		auditservice.WithMetadata(map[string]any{
			"shipmentId":         entity.ShipmentID.String(),
			"billingQueueItemId": entity.BillingQueueItemID.String(),
			"invoiceNumber":      entity.Number,
			"status":             entity.Status,
		}),
	}
	if previous != nil && current != nil {
		opts = append(opts, auditservice.WithDiff(previous, current))
	}

	if err := s.auditService.LogAction(params, opts...); err != nil {
		s.l.Error("failed to log invoice action", zap.Error(err))
	}
}

func (s *Service) publishInvalidation(
	ctx context.Context,
	entity *invoice.Invoice,
	actor servicesports.AuditActor,
	action string,
	payload any,
) {
	if err := realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: entity.OrganizationID,
		BusinessUnitID: entity.BusinessUnitID,
		ActorUserID:    actor.UserID,
		ActorType:      actor.PrincipalType,
		ActorID:        actor.PrincipalID,
		ActorAPIKeyID:  actor.APIKeyID,
		Resource:       permission.ResourceInvoice.String(),
		Action:         action,
		RecordID:       entity.ID,
		Entity:         payload,
	}); err != nil {
		s.l.Warn("failed to publish invoice invalidation", zap.Error(err))
	}
}

func (s *Service) logBillingQueuePosted(
	result *postedBillingQueueResult,
	actor servicesports.AuditActor,
) {
	if result == nil || result.Previous == nil || result.Updated == nil {
		return
	}

	if err := s.auditService.LogAction(&servicesports.LogActionParams{
		Resource:       permission.ResourceBillingQueue,
		ResourceID:     result.Updated.ID.String(),
		Operation:      permission.OpUpdate,
		UserID:         actor.UserID,
		APIKeyID:       actor.APIKeyID,
		PrincipalType:  actor.PrincipalType,
		PrincipalID:    actor.PrincipalID,
		OrganizationID: result.Updated.OrganizationID,
		BusinessUnitID: result.Updated.BusinessUnitID,
		CurrentState:   jsonutils.MustToJSON(result.Updated),
		PreviousState:  jsonutils.MustToJSON(result.Previous),
	},
		auditservice.WithComment("Billing queue item marked posted from invoice"),
		auditservice.WithDiff(result.Previous, result.Updated),
	); err != nil {
		s.l.Error("failed to log billing queue invoice posting", zap.Error(err))
	}
}

func (s *Service) publishBillingQueueInvalidation(
	ctx context.Context,
	result *postedBillingQueueResult,
	actor servicesports.AuditActor,
) {
	if result == nil || result.Previous == nil || result.Updated == nil {
		return
	}

	if err := realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: result.Updated.OrganizationID,
		BusinessUnitID: result.Updated.BusinessUnitID,
		ActorUserID:    actor.UserID,
		ActorType:      actor.PrincipalType,
		ActorID:        actor.PrincipalID,
		ActorAPIKeyID:  actor.APIKeyID,
		Resource:       permission.ResourceBillingQueue.String(),
		Action:         "updated",
		RecordID:       result.Updated.ID,
		Entity:         result.Updated,
	}); err != nil {
		s.l.Warn("failed to publish billing queue invalidation", zap.Error(err))
	}
}
