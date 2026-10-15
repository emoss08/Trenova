package invoiceservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/billing/app/invoicelines"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/reference/ports/accessorialchargeports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

const (
	maxMemoReasonLength = 1000
	maxMemoLines        = 200
)

// CreateMemo raises a credit or debit memo against a customer with nothing
// shipped behind it: a goodwill credit, a returned-cheque fee, a late charge. It
// sits on its own billing queue item so the queue, the posting sweep and the
// ledger treat it like any other bill.
func (s *Service) CreateMemo(
	ctx context.Context,
	req *invoiceports.CreateMemoRequest,
	actor *permissionports.RequestActor,
) (*invoice.Invoice, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Request is required",
		)
	}
	if actor == nil {
		return nil, errortypes.NewValidationError(
			"actor",
			errortypes.ErrRequired,
			"Actor is required",
		)
	}
	plan, err := s.planMemo(ctx, req)
	if err != nil {
		return nil, err
	}
	cus, control, lines := plan.customer, plan.control, plan.lines

	probe := s.buildMemoEntity(req, &billingqueue.BillingQueueItem{}, cus, control, lines)
	if err = s.refuseAutoPostingMemo(ctx, req, probe); err != nil {
		return nil, err
	}

	number, err := s.generateMemoNumber(ctx, req.TenantInfo, req.BillType)
	if err != nil {
		return nil, err
	}

	var created *invoice.Invoice
	err = s.db.WithTx(ctx, kernelports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		queued := memoQueueItem(req, cus, number)
		queued.AdjustmentContext[memoRaisedByKey] = actor.UserID.String()
		item, txErr := s.billingQueueRepo.Create(txCtx, queued)
		if txErr != nil {
			return txErr
		}

		entity := s.buildMemoEntity(req, item, cus, control, lines)
		if multiErr := s.validator.ValidateCreate(txCtx, entity); multiErr != nil {
			return multiErr
		}

		created, txErr = s.repo.Create(txCtx, entity)
		if txErr != nil {
			return txErr
		}

		auditActor := actor.AuditActor()
		s.logAction(
			created,
			auditActor,
			permission.OpCreate,
			nil,
			created,
			memoAuditComment(req.BillType),
		)
		s.publishInvalidation(txCtx, created, auditActor, "created", created)

		if !req.AutoPost {
			return nil
		}
		posted, postErr := s.Post(txCtx, &invoiceports.PostInvoiceRequest{
			InvoiceID:   created.ID,
			TenantInfo:  req.TenantInfo,
			TriggeredBy: "memo-auto-post",
			Automatic:   req.AutoPostByPolicy,
		}, actor)
		if postErr != nil {
			return postErr
		}
		created = posted
		return nil
	})
	if err != nil {
		return nil, err
	}

	return created, nil
}

func validateMemoRequest(req *invoiceports.CreateMemoRequest) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	if req.CustomerID.IsNil() {
		multiErr.Add("customerId", errortypes.ErrRequired, "Customer is required")
	}
	if req.BillType != billingqueue.BillTypeCreditMemo &&
		req.BillType != billingqueue.BillTypeDebitMemo {
		multiErr.Add(
			"billType",
			errortypes.ErrInvalid,
			"A memo must be a credit memo or a debit memo",
		)
	}
	reason := strings.TrimSpace(req.Reason)
	switch {
	case reason == "":
		multiErr.Add("reason", errortypes.ErrRequired, "Say why the memo is being raised")
	case len(reason) > maxMemoReasonLength:
		multiErr.Add(
			"reason",
			errortypes.ErrInvalid,
			"Reason must be at most {0} characters",
			maxMemoReasonLength,
		)
	}
	if req.MemoKind != "" && !req.MemoKind.IsValid() {
		multiErr.Add("memoKind", errortypes.ErrInvalid, "Invalid memo kind")
	}
	switch {
	case len(req.Lines) == 0:
		multiErr.Add("lines", errortypes.ErrRequired, "A memo needs at least one line")
	case len(req.Lines) > maxMemoLines:
		multiErr.Add(
			"lines",
			errortypes.ErrInvalid,
			"A memo may carry at most {0} lines",
			maxMemoLines,
		)
	}
	for idx, line := range req.Lines {
		if line == nil {
			multiErr.WithIndex("lines", idx).
				Add("description", errortypes.ErrRequired, "Line is required")
			continue
		}
		if strings.TrimSpace(line.Description) == "" {
			multiErr.WithIndex("lines", idx).
				Add("description", errortypes.ErrRequired, "Description is required")
		}
		if line.Amount.LessThanOrEqual(decimal.Zero) {
			multiErr.WithIndex("lines", idx).
				Add("amount", errortypes.ErrInvalid, "Amount must be greater than zero")
		}
		if !line.Quantity.IsZero() && line.Quantity.LessThanOrEqual(decimal.Zero) {
			multiErr.WithIndex("lines", idx).
				Add("quantity", errortypes.ErrInvalid, "Quantity must be greater than zero")
		}
	}
	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

// validateMemoReference checks the invoice a memo points at belongs to the same
// customer and is posted, so a credit never references a document that was
// never owed.
func (s *Service) validateMemoReference(
	ctx context.Context,
	req *invoiceports.CreateMemoRequest,
	cus *customer.Customer,
) error {
	if req.ReferenceInvoiceID.IsNil() {
		return nil
	}
	reference, err := s.repo.GetByID(ctx, invoiceports.GetInvoiceByIDRequest{
		ID:         req.ReferenceInvoiceID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return err
	}
	if reference.CustomerID != cus.ID {
		return errortypes.NewValidationError(
			"referenceInvoiceId",
			errortypes.ErrInvalid,
			"The referenced invoice belongs to another customer",
		)
	}
	if reference.Status != invoice.StatusPosted {
		return errortypes.NewValidationError(
			"referenceInvoiceId",
			errortypes.ErrInvalidOperation,
			"A memo can only reference a posted invoice",
		)
	}

	return nil
}

func (s *Service) generateMemoNumber(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	billType billingqueue.BillType,
) (string, error) {
	if billType == billingqueue.BillTypeCreditMemo {
		return s.sequenceGenerator.GenerateCreditMemoNumber(
			ctx,
			tenantInfo.OrgID,
			tenantInfo.BuID,
			"",
			"",
		)
	}

	return s.sequenceGenerator.GenerateDebitMemoNumber(
		ctx,
		tenantInfo.OrgID,
		tenantInfo.BuID,
		"",
		"",
	)
}

// memoLines turns the request lines into invoice lines. A line that names an
// accessorial carries its code so the memo reads like the charge it corrects;
// any other line is a plain memo line that counts only toward the total.
func (s *Service) memoLines(
	ctx context.Context,
	req *invoiceports.CreateMemoRequest,
) ([]*invoice.InvoiceLine, error) {
	lines := make([]*invoice.InvoiceLine, 0, len(req.Lines))
	for idx, input := range req.Lines {
		quantity := input.Quantity
		if quantity.IsZero() {
			quantity = decimal.NewFromInt(1)
		}
		amount := invoicelines.SignedAmount(req.BillType, input.Amount.RoundBank(2))
		line := &invoice.InvoiceLine{
			LineNumber:  idx + 1,
			Type:        invoice.InvoiceLineTypeMemo,
			Description: strings.TrimSpace(input.Description),
			Quantity:    quantity,
			UnitPrice:   amount.Div(quantity),
			Amount:      amount,
		}
		if input.AccessorialChargeID.IsNotNil() && s.accessorialRepo != nil {
			definition, err := s.accessorialRepo.GetByID(
				ctx,
				accessorialchargeports.GetAccessorialChargeByIDRequest{
					ID:         input.AccessorialChargeID,
					TenantInfo: &req.TenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			line.Type = invoice.InvoiceLineTypeAccessorial
			line.AccessorialChargeID = definition.ID
			line.ChargeCode = strings.TrimSpace(definition.Code)
		}
		lines = append(lines, line)
	}

	return lines, nil
}

func (s *Service) buildMemoEntity(
	req *invoiceports.CreateMemoRequest,
	item *billingqueue.BillingQueueItem,
	cus *customer.Customer,
	control *tenant.BillingControl,
	lines []*invoice.InvoiceLine,
) *invoice.Invoice {
	invoiceDate := req.InvoiceDate
	if invoiceDate == 0 {
		invoiceDate = timeutils.NowUnix()
	}
	paymentTerm := resolvePaymentTerm(cus, control)
	if paymentTerm == "" {
		paymentTerm = invoice.PaymentTermNet30
	}

	entity := &invoice.Invoice{
		ID:                 req.ID,
		OrganizationID:     req.TenantInfo.OrgID,
		BusinessUnitID:     req.TenantInfo.BuID,
		BillingQueueItemID: item.ID,
		Scope:              invoice.ScopeMemo,
		CustomerID:         cus.ID,
		Number:             item.Number,
		BillType:           req.BillType,
		Status:             invoice.StatusDraft,
		PaymentTerm:        paymentTerm,
		CurrencyCode:       memoCurrency(req, cus),
		InvoiceDate:        invoiceDate,
		AppliedAmount:      decimal.Zero,
		SettlementStatus:   invoice.SettlementStatusUnpaid,
		DisputeStatus:      invoice.DisputeStatusNone,
		ReferenceInvoiceID: req.ReferenceInvoiceID,
		MemoReason:         strings.TrimSpace(req.Reason),
		MemoKind:           memoKindOrDefault(req.MemoKind),
		Memo:               strings.TrimSpace(req.Memo),
		Lines:              lines,
	}
	if req.BillType == billingqueue.BillTypeCreditMemo {
		due := invoiceDate
		entity.DueDate = &due
	} else {
		entity.DueDate = invoice.DueDateFromPaymentTerm(invoiceDate, paymentTerm)
	}

	applyBillToSnapshot(entity, cus)
	applyInvoiceDetail(entity, cus)
	syncInvoiceTotalsFromLines(entity)
	entity.SyncMinorAmounts()

	return entity
}

// applyBillToSnapshot copies the payer's name and address onto the invoice, so
// the document keeps saying who was billed even after the customer record moves.
func applyBillToSnapshot(entity *invoice.Invoice, cus *customer.Customer) {
	if entity == nil || cus == nil {
		return
	}
	entity.BillToName = cus.Name
	entity.BillToCode = cus.Code
	entity.BillToAddressLine1 = cus.AddressLine1
	entity.BillToAddressLine2 = cus.AddressLine2
	entity.BillToCity = cus.City
	entity.BillToPostalCode = cus.PostalCode
	if cus.State != nil {
		entity.BillToState = cus.State.Abbreviation
		entity.BillToCountry = cus.State.CountryName
	}
}

func memoKindOrDefault(kind invoice.MemoKind) invoice.MemoKind {
	if kind.IsValid() {
		return kind
	}

	return invoice.MemoKindManual
}

func memoAuditComment(billType billingqueue.BillType) string {
	if billType == billingqueue.BillTypeCreditMemo {
		return "Credit memo created"
	}

	return "Debit memo created"
}

func memoCurrency(req *invoiceports.CreateMemoRequest, cus *customer.Customer) string {
	if code := strings.ToUpper(strings.TrimSpace(req.CurrencyCode)); code != "" {
		return code
	}

	return billingCurrencyFromCustomer(cus)
}
