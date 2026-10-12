package detentioncreditservice

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceadjustmentports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	maxAttempts    = 20
	refusalField   = "decision"
	idempotencyKey = "detention-waiver:%s:%s:%d"
)

type invoiceReader interface {
	NetBilledByCharge(
		ctx context.Context,
		req *invoiceports.NetBilledByChargeRequest,
	) (map[pulid.ID]decimal.Decimal, error)
	ListByShipmentIDs(
		ctx context.Context,
		req invoiceports.ListInvoicesByShipmentIDsRequest,
	) (map[pulid.ID][]*invoice.Invoice, error)
	GetByID(ctx context.Context, req invoiceports.GetInvoiceByIDRequest) (*invoice.Invoice, error)
}

type adjustmentSubmitter interface {
	Preview(
		ctx context.Context,
		req *invoiceadjustmentports.InvoiceAdjustmentRequest,
		actor *permissionports.RequestActor,
	) (*invoiceadjustmentports.InvoiceAdjustmentPreview, error)
	Submit(
		ctx context.Context,
		req *invoiceadjustmentports.InvoiceAdjustmentRequest,
		actor *permissionports.RequestActor,
	) (*invoiceadjustment.InvoiceAdjustment, error)
}

type adjustmentFinder interface {
	GetByIdempotencyKey(
		ctx context.Context,
		req invoiceadjustmentports.GetInvoiceAdjustmentByIdempotencyRequest,
	) (*invoiceadjustment.InvoiceAdjustment, error)
}

type permissionChecker interface {
	Check(
		ctx context.Context,
		req *permissionports.PermissionCheckRequest,
	) (*permissionports.PermissionCheckResult, error)
}

type Params struct {
	fx.In

	Logger         *zap.Logger
	Invoices       invoiceports.InvoiceRepository
	Adjustments    invoiceadjustmentports.InvoiceAdjustmentService
	AdjustmentRepo invoiceadjustmentports.InvoiceAdjustmentRepository
	Permissions    permissionports.PermissionEngine
}

type Service struct {
	l           *zap.Logger
	invoices    invoiceReader
	adjustments adjustmentSubmitter
	finder      adjustmentFinder
	permissions permissionChecker
}

var _ detentionports.BilledDetentionCreditor = (*Service)(nil)

func New(p Params) *Service { //nolint:gocritic // fx params are passed by value
	return &Service{
		l:           p.Logger.Named("service.detention-credit"),
		invoices:    p.Invoices,
		adjustments: p.Adjustments,
		finder:      p.AdjustmentRepo,
		permissions: p.Permissions,
	}
}

type billedLines struct {
	invoice *invoice.Invoice
	lines   []*invoice.InvoiceLine
}

func (s *Service) CreditBilledCharge(
	ctx context.Context,
	req *detentionports.CreditBilledDetentionRequest,
) (*detentionports.DetentionCredit, error) {
	if req == nil || req.ChargeID.IsNil() {
		return nil, nil //nolint:nilnil // nothing was billed, so nothing is credited
	}

	net, err := s.invoices.NetBilledByCharge(ctx, &invoiceports.NetBilledByChargeRequest{
		TenantInfo: req.TenantInfo,
		ChargeIDs:  []pulid.ID{req.ChargeID},
	})
	if err != nil {
		return nil, err
	}
	if !net[req.ChargeID].IsPositive() {
		return nil, nil //nolint:nilnil // no standing invoice bills the charge
	}

	if err = s.authorize(ctx, req.Actor); err != nil {
		return nil, err
	}

	billed, err := s.billedInvoices(ctx, req)
	if err != nil {
		return nil, err
	}
	if len(billed) == 0 {
		return nil, errortypes.NewBusinessError(
			"An invoice still bills this detention charge, but none of its lines can be credited. Adjust the invoice in billing.",
		)
	}

	credit := &detentionports.DetentionCredit{Amount: decimal.Zero}
	for _, entry := range billed {
		item, creditErr := s.creditInvoice(ctx, req, entry)
		if creditErr != nil {
			return nil, creditErr
		}
		if item == nil {
			continue
		}
		credit.Amount = credit.Amount.Add(item.Amount)
		credit.Currency = entry.invoice.CurrencyCode
		credit.Invoices = append(credit.Invoices, *item)
	}
	if len(credit.Invoices) == 0 {
		return nil, errortypes.NewBusinessError(
			"The invoices that bill this detention charge have nothing left to credit. Check the adjustments waiting in billing.",
		)
	}

	return credit, nil
}

func (s *Service) authorize(ctx context.Context, actor *permissionports.RequestActor) error {
	if actor == nil {
		return errInvoicePermission()
	}

	result, err := s.permissions.Check(ctx, &permissionports.PermissionCheckRequest{
		PrincipalType:  actor.PrincipalType,
		PrincipalID:    actor.PrincipalID,
		UserID:         actor.UserID,
		APIKeyID:       actor.APIKeyID,
		BusinessUnitID: actor.BusinessUnitID,
		OrganizationID: actor.OrganizationID,
		Resource:       permission.ResourceInvoice.String(),
		Operation:      permission.OpUpdate,
	})
	if err != nil {
		return fmt.Errorf("authorize the detention credit: %w", err)
	}
	if result == nil || !result.Allowed {
		return errInvoicePermission()
	}

	return nil
}

func errInvoicePermission() error {
	return errortypes.NewAuthorizationError(
		"Waiving a billed detention charge credits the invoice, which needs permission to update invoices",
	)
}

func (s *Service) billedInvoices(
	ctx context.Context,
	req *detentionports.CreditBilledDetentionRequest,
) ([]billedLines, error) {
	byShipment, err := s.invoices.ListByShipmentIDs(
		ctx,
		invoiceports.ListInvoicesByShipmentIDsRequest{
			TenantInfo:  req.TenantInfo,
			ShipmentIDs: []pulid.ID{req.ShipmentID},
		},
	)
	if err != nil {
		return nil, err
	}

	invoices := byShipment[req.ShipmentID]
	billed := make([]billedLines, 0, len(invoices))
	for _, entity := range invoices {
		if entity == nil || entity.Status == invoice.StatusVoided ||
			entity.BillType == billingqueue.BillTypeCreditMemo {
			continue
		}

		lines := chargeLines(entity, req.ChargeID)
		if len(lines) == 0 {
			continue
		}
		if entity.Status != invoice.StatusPosted {
			return nil, errortypes.NewValidationError(
				refusalField,
				errortypes.ErrInvalidOperation,
				"Invoice {0} bills this detention charge and is still a draft. Take the charge off the draft invoice, then waive it.",
				invoiceLabel(entity),
			)
		}
		billed = append(billed, billedLines{invoice: entity, lines: lines})
	}

	slices.SortFunc(billed, func(a, b billedLines) int {
		return strings.Compare(a.invoice.ID.String(), b.invoice.ID.String())
	})

	return billed, nil
}

func chargeLines(entity *invoice.Invoice, chargeID pulid.ID) []*invoice.InvoiceLine {
	lines := make([]*invoice.InvoiceLine, 0, 1)
	for _, line := range entity.Lines {
		if line != nil && line.AdditionalChargeID == chargeID && line.Amount.IsPositive() {
			lines = append(lines, line)
		}
	}

	return lines
}

func (s *Service) creditInvoice(
	ctx context.Context,
	req *detentionports.CreditBilledDetentionRequest,
	entry billedLines,
) (*detentionports.DetentionCreditInvoice, error) {
	key, existing, err := s.attemptKey(ctx, req, entry.invoice.ID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return s.creditOf(ctx, req.TenantInfo, entry.invoice, existing)
	}

	adjustmentReq := &invoiceadjustmentports.InvoiceAdjustmentRequest{
		InvoiceID:      entry.invoice.ID,
		Kind:           invoiceadjustment.KindCreditOnly,
		Reason:         req.Reason,
		IdempotencyKey: key,
		Lines: make(
			[]*invoiceadjustmentports.InvoiceAdjustmentLineInput,
			0,
			len(entry.lines),
		),
		TenantInfo:       req.TenantInfo,
		DetentionWaivers: []detentionports.DetentionWaiver{req.Waiver},
	}
	for _, line := range entry.lines {
		adjustmentReq.Lines = append(
			adjustmentReq.Lines,
			&invoiceadjustmentports.InvoiceAdjustmentLineInput{
				OriginalLineID: line.ID,
				Description:    line.Description,
			},
		)
	}

	preview, err := s.adjustments.Preview(ctx, adjustmentReq, req.Actor)
	if err != nil {
		return nil, refusal(entry.invoice, err)
	}
	if len(preview.Errors) > 0 {
		return nil, refusalFromMessages(entry.invoice, preview.Errors)
	}
	if !preview.CreditTotalAmount.IsPositive() {
		return nil, nil //nolint:nilnil // every line was already credited
	}
	adjustmentReq.Lines = creditableLines(adjustmentReq.Lines, preview)

	adjustment, err := s.adjustments.Submit(ctx, adjustmentReq, req.Actor)
	if err != nil {
		return nil, refusal(entry.invoice, err)
	}

	return s.creditOf(ctx, req.TenantInfo, entry.invoice, adjustment)
}

func creditableLines(
	lines []*invoiceadjustmentports.InvoiceAdjustmentLineInput,
	preview *invoiceadjustmentports.InvoiceAdjustmentPreview,
) []*invoiceadjustmentports.InvoiceAdjustmentLineInput {
	creditable := make(map[pulid.ID]struct{}, len(preview.Lines))
	for _, line := range preview.Lines {
		if line != nil && line.RequestedCreditAmount.IsPositive() {
			creditable[line.OriginalLineID] = struct{}{}
		}
	}

	return slices.DeleteFunc(
		lines,
		func(line *invoiceadjustmentports.InvoiceAdjustmentLineInput) bool {
			_, ok := creditable[line.OriginalLineID]
			return !ok
		},
	)
}

func (s *Service) attemptKey(
	ctx context.Context,
	req *detentionports.CreditBilledDetentionRequest,
	invoiceID pulid.ID,
) (string, *invoiceadjustment.InvoiceAdjustment, error) {
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		key := fmt.Sprintf(idempotencyKey, req.Waiver.OccurrenceID, invoiceID, attempt)
		existing, err := s.finder.GetByIdempotencyKey(
			ctx,
			invoiceadjustmentports.GetInvoiceAdjustmentByIdempotencyRequest{
				IdempotencyKey: key,
				TenantInfo:     req.TenantInfo,
			},
		)
		if errortypes.IsNotFoundError(err) || (err == nil && existing == nil) {
			return key, nil, nil
		}
		if err != nil {
			return "", nil, err
		}
		if !settledAgainst(existing.Status) {
			return key, existing, nil
		}
	}

	return "", nil, errortypes.NewBusinessError(
		"This detention charge has been put up for credit too many times. Adjust the invoice in billing.",
	)
}

func settledAgainst(status invoiceadjustment.Status) bool {
	return status == invoiceadjustment.StatusRejected ||
		status == invoiceadjustment.StatusExecutionFailed
}

func (s *Service) creditOf(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	source *invoice.Invoice,
	adjustment *invoiceadjustment.InvoiceAdjustment,
) (*detentionports.DetentionCreditInvoice, error) {
	item := &detentionports.DetentionCreditInvoice{
		AdjustmentID:  adjustment.ID,
		InvoiceID:     source.ID,
		InvoiceNumber: invoiceLabel(source),
		Amount:        adjustment.CreditTotalAmount,
	}

	switch adjustment.Status {
	case invoiceadjustment.StatusPendingApproval:
		item.Status = detentionports.DetentionCreditPendingApproval

		return item, nil
	case invoiceadjustment.StatusExecuted:
		item.Status = detentionports.DetentionCreditIssued
	case invoiceadjustment.StatusApproved, invoiceadjustment.StatusExecuting:
		item.Status = detentionports.DetentionCreditPendingApproval

		return item, nil
	case invoiceadjustment.StatusExecutionFailed, invoiceadjustment.StatusRejected:
		return nil, errortypes.NewValidationError(
			refusalField,
			errortypes.ErrInvalidOperation,
			"Invoice {0} could not be credited for the waived detention: {1}",
			invoiceLabel(source),
			executionFailure(adjustment),
		)
	case invoiceadjustment.StatusDraft:
		return nil, errortypes.NewBusinessError(
			"The credit for invoice {0} is still a draft. Submit it in billing.",
			invoiceLabel(source),
		)
	}

	if adjustment.CreditMemoInvoiceID.IsNil() {
		return item, nil
	}

	memo, err := s.invoices.GetByID(ctx, invoiceports.GetInvoiceByIDRequest{
		ID:         adjustment.CreditMemoInvoiceID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		s.l.Warn("credited a waived detention charge but could not read the credit memo",
			zap.String("adjustmentId", adjustment.ID.String()), zap.Error(err))

		return item, nil
	}
	item.CreditMemoNumber = strings.TrimSpace(memo.Number)

	return item, nil
}

func executionFailure(adjustment *invoiceadjustment.InvoiceAdjustment) string {
	if message := strings.TrimSpace(adjustment.ExecutionError); message != "" {
		return message
	}
	if message := strings.TrimSpace(adjustment.RejectionReason); message != "" {
		return message
	}

	return string(adjustment.Status)
}

func refusal(source *invoice.Invoice, err error) error {
	if multiErr, ok := errors.AsType[*errortypes.MultiError](err); ok && multiErr.HasErrors() {
		messages := make([]string, 0, len(multiErr.Errors))
		for _, fieldErr := range multiErr.Errors {
			messages = append(messages, fieldErr.Error())
		}

		return refusalOf(source, messages)
	}
	if fieldErr, ok := errors.AsType[*errortypes.Error](err); ok {
		return refusalOf(source, []string{fieldErr.Error()})
	}
	if errortypes.IsBusinessError(err) {
		return refusalOf(source, []string{err.Error()})
	}

	return err
}

func refusalFromMessages(source *invoice.Invoice, byField map[string][]string) error {
	fields := make([]string, 0, len(byField))
	for field := range byField {
		fields = append(fields, field)
	}
	slices.Sort(fields)

	messages := make([]string, 0, len(byField))
	for _, field := range fields {
		messages = append(messages, byField[field]...)
	}

	return refusalOf(source, messages)
}

func refusalOf(source *invoice.Invoice, messages []string) error {
	return errortypes.NewValidationError(
		refusalField,
		errortypes.ErrInvalidOperation,
		"Invoice {0} could not be credited for the waived detention: {1}",
		invoiceLabel(source),
		strings.Join(slices.Compact(messages), "; "),
	)
}

func invoiceLabel(entity *invoice.Invoice) string {
	if number := strings.TrimSpace(entity.Number); number != "" {
		return number
	}

	return entity.ID.String()
}
