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
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	servicesports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/accountingcontrolpolicyservice"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/core/services/billingcontrolpolicyservice"
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

	Logger             *zap.Logger
	DB                 ports.DBConnection
	Repo               repositories.InvoiceRepository
	BillingQueueRepo   repositories.BillingQueueRepository
	ShipmentRepo       repositories.ShipmentRepository
	CustomerRepo       repositories.CustomerRepository
	CustomerLedgerRepo repositories.CustomerLedgerProjectionRepository
	BillingRepo        repositories.BillingControlRepository
	AccountingRepo     repositories.AccountingControlRepository
	JournalRepo        repositories.JournalPostingRepository
	AdjustmentRepo     repositories.InvoiceAdjustmentRepository
	NotificationRepo   repositories.NotificationRepository
	Validator          *Validator
	AuditService       servicesports.AuditService
	Realtime           servicesports.RealtimeService
	WorkflowStarter    servicesports.WorkflowStarter
	SequenceGenerator  seqgen.Generator
	AccountingPolicy   *accountingcontrolpolicyservice.Service
	BillingPolicy      *billingcontrolpolicyservice.Service
}

type Service struct {
	l                  *zap.Logger
	db                 ports.DBConnection
	repo               repositories.InvoiceRepository
	billingQueueRepo   repositories.BillingQueueRepository
	shipmentRepo       repositories.ShipmentRepository
	customerRepo       repositories.CustomerRepository
	customerLedgerRepo repositories.CustomerLedgerProjectionRepository
	billingRepo        repositories.BillingControlRepository
	accountingRepo     repositories.AccountingControlRepository
	journalRepo        repositories.JournalPostingRepository
	adjustmentRepo     repositories.InvoiceAdjustmentRepository
	notificationRepo   repositories.NotificationRepository
	validator          *Validator
	auditService       servicesports.AuditService
	realtime           servicesports.RealtimeService
	workflowStarter    servicesports.WorkflowStarter
	sequenceGenerator  seqgen.Generator
	accountingPolicy   *accountingcontrolpolicyservice.Service
	billingPolicy      *billingcontrolpolicyservice.Service
}

type existingInvoiceLookupResult struct {
	Invoice *invoice.Invoice
	Found   bool
}

type invoiceDependencies struct {
	Shipment       *shipment.Shipment
	Customer       *customer.Customer
	BillingControl *tenant.BillingControl
}

type postedBillingQueueResult struct {
	Previous *billingqueue.BillingQueueItem
	Updated  *billingqueue.BillingQueueItem
}

var _ servicesports.InvoiceService = (*Service)(nil)

//nolint:gocritic // dependency injection
func New(p Params) servicesports.InvoiceService {
	return &Service{
		l:                  p.Logger.Named("service.invoice"),
		db:                 p.DB,
		repo:               p.Repo,
		billingQueueRepo:   p.BillingQueueRepo,
		shipmentRepo:       p.ShipmentRepo,
		customerRepo:       p.CustomerRepo,
		customerLedgerRepo: p.CustomerLedgerRepo,
		billingRepo:        p.BillingRepo,
		accountingRepo:     p.AccountingRepo,
		journalRepo:        p.JournalRepo,
		adjustmentRepo:     p.AdjustmentRepo,
		notificationRepo:   p.NotificationRepo,
		validator:          p.Validator,
		auditService:       p.AuditService,
		realtime:           p.Realtime,
		workflowStarter:    p.WorkflowStarter,
		sequenceGenerator:  p.SequenceGenerator,
		accountingPolicy:   p.AccountingPolicy,
		billingPolicy:      p.BillingPolicy,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListInvoicesRequest,
) (*pagination.ListResult[*invoice.Invoice], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) GetByID(
	ctx context.Context,
	req repositories.GetInvoiceByIDRequest,
) (*invoice.Invoice, error) {
	return s.repo.GetByID(ctx, req)
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

	entity := s.buildInvoiceEntity(
		item,
		dependencies.Shipment,
		dependencies.Customer,
		dependencies.BillingControl,
	)
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

	if s.adjustmentRepo != nil && item.CorrectionGroupID != nil && item.CorrectionGroupID.IsNotNil() {
		group, err := s.adjustmentRepo.GetCorrectionGroup(ctx, repositories.GetCorrectionGroupRequest{
			ID: *item.CorrectionGroupID,
			TenantInfo: pagination.TenantInfo{
				OrgID: item.OrganizationID,
				BuID:  item.BusinessUnitID,
			},
		})
		if err == nil && group != nil {
			group.CurrentInvoiceID = created.ID
			if _, err = s.adjustmentRepo.UpdateCorrectionGroup(ctx, group); err != nil {
				s.l.Warn("failed to update correction group current invoice", zap.Error(err))
			}
		}
	}
}

func (s *Service) Post(
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
				if policyErr := s.billingPolicyService().ValidateInvoicePosting(control, req.TriggeredBy); policyErr != nil {
					return policyErr
				}
			}
		}

		auditActor := actor.AuditActor()

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

		if multiErr := s.validator.ValidatePost(txCtx, entity, req.TenantInfo, now); multiErr != nil {
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

		shp, shipErr := s.shipmentRepo.GetByID(txCtx, &repositories.GetShipmentByIDRequest{
			ID:         updated.ShipmentID,
			TenantInfo: req.TenantInfo,
		})
		if shipErr != nil {
			return shipErr
		}

		shp.Status = shipment.StatusInvoiced
		shp.BilledAt = &now
		if _, shipErr = s.shipmentRepo.UpdateDerivedState(txCtx, shp); shipErr != nil {
			return shipErr
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
		s.notifyReconciliationWarning(txCtx, updated, shp)

		return nil
	})
	if err != nil {
		return nil, err
	}

	return posted, nil
}

func (s *Service) notifyReconciliationWarning(
	ctx context.Context,
	entity *invoice.Invoice,
	shp *shipment.Shipment,
) {
	if s.accountingRepo == nil || s.notificationRepo == nil || entity == nil || shp == nil {
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

	expectedTotal := signedAmount(entity.BillType, shp.TotalChargeAmount.Decimal)
	discrepancy := entity.TotalAmount.Sub(expectedTotal).Abs()
	if !discrepancy.GreaterThan(control.ReconciliationToleranceAmount) {
		return
	}

	if _, err = s.notificationRepo.Create(ctx, &notification.Notification{
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
		RelatedEntities: map[string]any{
			"invoiceId":  entity.ID.String(),
			"shipmentId": entity.ShipmentID.String(),
		},
		Source: "invoiceservice.Post",
	}); err != nil {
		s.l.Warn("failed to create reconciliation warning notification", zap.Error(err))
	}
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

func (s *Service) getInvoiceDependencies(
	ctx context.Context,
	req *servicesports.CreateInvoiceFromBillingQueueRequest,
	item *billingqueue.BillingQueueItem,
) (*invoiceDependencies, error) {
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

	cus, err := s.customerRepo.GetByID(ctx, repositories.GetCustomerByIDRequest{
		ID:         shp.CustomerID,
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

	return &invoiceDependencies{
		Shipment:       shp,
		Customer:       cus,
		BillingControl: control,
	}, nil
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

func (s *Service) buildInvoiceEntity(
	item *billingqueue.BillingQueueItem,
	shp *shipment.Shipment,
	cus *customer.Customer,
	control *tenant.BillingControl,
) *invoice.Invoice {
	if item.IsAdjustmentOrigin {
		if entity := s.buildAdjustmentOriginInvoiceEntity(item, shp, cus, control); entity != nil {
			return entity
		}
	}

	invoiceDate := timeutils.NowUnix()
	paymentTerm := resolvePaymentTerm(cus, control)
	if paymentTerm == "" {
		paymentTerm = invoice.PaymentTermNet30
	}

	entity := &invoice.Invoice{
		OrganizationID:     item.OrganizationID,
		BusinessUnitID:     item.BusinessUnitID,
		BillingQueueItemID: item.ID,
		ShipmentID:         shp.ID,
		CustomerID:         cus.ID,
		Number:             item.Number,
		BillType:           item.BillType,
		Status:             invoice.StatusDraft,
		PaymentTerm:        paymentTerm,
		CurrencyCode:       billingCurrencyFromCustomer(cus),
		InvoiceDate:        invoiceDate,
		DueDate:            invoice.DueDateFromPaymentTerm(invoiceDate, paymentTerm),
		ShipmentProNumber:  shp.ProNumber,
		ShipmentBOL:        shp.BOL,
		ServiceDate:        serviceDateFromShipment(shp),
		BillToName:         cus.Name,
		BillToCode:         cus.Code,
		BillToAddressLine1: cus.AddressLine1,
		BillToAddressLine2: cus.AddressLine2,
		BillToCity:         cus.City,
		BillToPostalCode:   cus.PostalCode,
		SubtotalAmount:     signedAmount(item.BillType, shp.FreightChargeAmount.Decimal),
		OtherAmount:        signedAmount(item.BillType, shp.OtherChargeAmount.Decimal),
		TotalAmount:        signedAmount(item.BillType, shp.TotalChargeAmount.Decimal),
		AppliedAmount:      decimal.Zero,
		SettlementStatus:   invoice.SettlementStatusUnpaid,
		DisputeStatus:      invoice.DisputeStatusNone,
		Lines:              buildInvoiceLines(item.BillType, shp),
	}

	if cus.State != nil {
		entity.BillToState = cus.State.Abbreviation
		entity.BillToCountry = cus.State.CountryName
	}

	entity.SyncMinorAmounts()

	return entity
}

type adjustmentInvoiceContext struct {
	ReplacementLines   []*invoice.Line `json:"replacementLines"`
	SubtotalAmount     decimal.Decimal `json:"subtotalAmount"`
	OtherAmount        decimal.Decimal `json:"otherAmount"`
	TotalAmount        decimal.Decimal `json:"totalAmount"`
	AccountingDate     int64           `json:"accountingDate"`
	SourceInvoiceID    pulid.ID        `json:"sourceInvoiceId"`
	CorrectionGroupID  pulid.ID        `json:"correctionGroupId"`
	SourceAdjustmentID pulid.ID        `json:"sourceAdjustmentId"`
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
		lines = buildInvoiceLines(item.BillType, shp)
	}

	entity := &invoice.Invoice{
		OrganizationID:            item.OrganizationID,
		BusinessUnitID:            item.BusinessUnitID,
		BillingQueueItemID:        item.ID,
		ShipmentID:                shp.ID,
		CustomerID:                cus.ID,
		Number:                    item.Number,
		BillType:                  item.BillType,
		Status:                    invoice.StatusDraft,
		PaymentTerm:               paymentTerm,
		CurrencyCode:              billingCurrencyFromCustomer(cus),
		InvoiceDate:               invoiceDate,
		DueDate:                   invoice.DueDateFromPaymentTerm(invoiceDate, paymentTerm),
		ShipmentProNumber:         shp.ProNumber,
		ShipmentBOL:               shp.BOL,
		ServiceDate:               serviceDateFromShipment(shp),
		BillToName:                cus.Name,
		BillToCode:                cus.Code,
		BillToAddressLine1:        cus.AddressLine1,
		BillToAddressLine2:        cus.AddressLine2,
		BillToCity:                cus.City,
		BillToPostalCode:          cus.PostalCode,
		SubtotalAmount:            ctx.SubtotalAmount,
		OtherAmount:               ctx.OtherAmount,
		TotalAmount:               ctx.TotalAmount,
		AppliedAmount:             decimal.Zero,
		SettlementStatus:          invoice.SettlementStatusUnpaid,
		DisputeStatus:             invoice.DisputeStatusNone,
		CorrectionGroupID:         ctx.CorrectionGroupID,
		SupersedesInvoiceID:       ctx.SourceInvoiceID,
		SourceInvoiceAdjustmentID: ctx.SourceAdjustmentID,
		Lines:                     lines,
	}
	if ctx.SubtotalAmount.IsZero() && ctx.OtherAmount.IsZero() {
		entity.SubtotalAmount = sumLinesByType(lines, invoice.LineTypeFreight)
		entity.OtherAmount = sumLinesByType(lines, invoice.LineTypeAccessorial)
	}
	if ctx.TotalAmount.IsZero() {
		entity.TotalAmount = sumLinesByType(lines, "")
	}
	if cus.State != nil {
		entity.BillToState = cus.State.Abbreviation
		entity.BillToCountry = cus.State.CountryName
	}
	entity.SyncMinorAmounts()
	return entity
}

func sumLinesByType(lines []*invoice.Line, lineType invoice.LineType) decimal.Decimal {
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
) []*invoice.Line {
	lines := make([]*invoice.Line, 0, 1+len(shp.AdditionalCharges))
	freightAmount := signedAmount(billType, shp.FreightChargeAmount.Decimal)
	lines = append(lines, &invoice.Line{
		LineNumber:  1,
		Type:        invoice.LineTypeFreight,
		Description: "Freight charge",
		Quantity:    decimal.NewFromInt(1),
		UnitPrice:   freightAmount,
		Amount:      freightAmount,
	})

	for idx, charge := range shp.AdditionalCharges {
		if charge == nil {
			continue
		}

		quantity := decimal.NewFromInt(int64(charge.Unit))
		if quantity.LessThanOrEqual(decimal.Zero) {
			quantity = decimal.NewFromInt(1)
		}

		amount := signedAmount(billType, charge.Amount)
		unitPrice := amount
		if !quantity.IsZero() {
			unitPrice = amount.Div(quantity)
		}

		description := "Accessorial charge"
		if charge.AccessorialCharge != nil &&
			strings.TrimSpace(charge.AccessorialCharge.Description) != "" {
			description = charge.AccessorialCharge.Description
		}

		lines = append(lines, &invoice.Line{
			LineNumber:  idx + 2,
			Type:        invoice.LineTypeAccessorial,
			Description: description,
			Quantity:    quantity,
			UnitPrice:   unitPrice,
			Amount:      amount,
		})
	}

	return lines
}

func signedAmount(
	billType billingqueue.BillType,
	amount decimal.Decimal,
) decimal.Decimal {
	if billType == billingqueue.BillTypeCreditMemo {
		return amount.Neg()
	}

	return amount
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

func resolveEffectiveAutoPost(
	control *tenant.BillingControl,
	cus *customer.Customer,
) bool {
	if control == nil {
		return false
	}

	if control.InvoicePostingMode != tenant.InvoicePostingModeAutomaticWhenNoBlockingExceptions {
		return false
	}

	if cus == nil || cus.BillingProfile == nil {
		return true
	}

	return cus.BillingProfile.AutoBill
}

func (s *Service) accountingPolicyService() *accountingcontrolpolicyservice.Service {
	if s.accountingPolicy != nil {
		return s.accountingPolicy
	}
	return accountingcontrolpolicyservice.New(accountingcontrolpolicyservice.Params{Logger: zap.NewNop()})
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
