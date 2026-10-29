package invoicedisputeservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoicedispute"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceadjustmentports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const maxDisputeNotesLength = 2000

type Params struct {
	fx.In

	Logger         *zap.Logger
	DB             kernelports.Transactor
	Repo           invoiceports.InvoiceDisputeRepository
	InvoiceRepo    invoiceports.InvoiceRepository
	AdjustmentRepo invoiceadjustmentports.InvoiceAdjustmentRepository
	AuditService   auditports.AuditService
	Realtime       realtimeports.RealtimeService
}

// Service runs the dispute case lifecycle. The invoice's DisputeStatus flag
// is derived: Disputed while a case is Open, None otherwise, and the two are
// changed together under the invoice row lock.
type Service struct {
	l              *zap.Logger
	db             kernelports.Transactor
	repo           invoiceports.InvoiceDisputeRepository
	invoiceRepo    invoiceports.InvoiceRepository
	adjustmentRepo invoiceadjustmentports.InvoiceAdjustmentRepository
	auditService   auditports.AuditService
	realtime       realtimeports.RealtimeService
}

func New(p Params) invoiceports.InvoiceDisputeService {
	return &Service{
		l:              p.Logger.Named("service.invoice-dispute"),
		db:             p.DB,
		repo:           p.Repo,
		invoiceRepo:    p.InvoiceRepo,
		adjustmentRepo: p.AdjustmentRepo,
		auditService:   p.AuditService,
		realtime:       p.Realtime,
	}
}

func (s *Service) Open(
	ctx context.Context,
	req *invoiceports.OpenInvoiceDisputeRequest,
	actor *permissionports.RequestActor,
) (*invoicedispute.InvoiceDispute, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Request is required",
		)
	}
	if actor == nil || actor.UserID.IsNil() {
		return nil, errortypes.NewAuthorizationError(
			"Opening a dispute requires an authenticated user",
		)
	}
	if multiErr := validateOpenRequest(req); multiErr != nil {
		return nil, multiErr
	}

	var created *invoicedispute.InvoiceDispute
	err := s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		inv, txErr := s.invoiceRepo.LockForUpdate(txCtx, invoiceports.GetInvoiceByIDRequest{
			ID:         req.InvoiceID,
			TenantInfo: req.TenantInfo,
		})
		if txErr != nil {
			return txErr
		}
		entity, txErr := s.planOpen(txCtx, req, inv, actor, timeutils.NowUnix())
		if txErr != nil {
			return txErr
		}
		created, txErr = s.repo.Create(txCtx, entity)
		if txErr != nil {
			return txErr
		}

		return s.setInvoiceDisputeFlag(txCtx, inv, invoice.DisputeStatusDisputed, actor)
	})
	if err != nil {
		return nil, err
	}

	s.logAudit(created, actor, permission.OpCreate, nil, "Invoice dispute opened")
	s.publish(ctx, created, actor, "created")

	return created, nil
}

func (s *Service) Resolve(
	ctx context.Context,
	req *invoiceports.ResolveInvoiceDisputeRequest,
	actor *permissionports.RequestActor,
) (*invoicedispute.InvoiceDispute, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Request is required",
		)
	}
	if actor == nil || actor.UserID.IsNil() {
		return nil, errortypes.NewAuthorizationError(
			"Resolving a dispute requires an authenticated user",
		)
	}
	if multiErr := validateResolveRequest(req); multiErr != nil {
		return nil, multiErr
	}

	var previous, updated *invoicedispute.InvoiceDispute
	err := s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		entity, inv, txErr := s.loadCase(txCtx, req.DisputeID, req.TenantInfo, true, verbResolved)
		if txErr != nil {
			return txErr
		}

		snapshot := *entity
		previous = &snapshot
		if txErr = s.planResolve(
			txCtx,
			req,
			entity,
			inv,
			actor,
			timeutils.NowUnix(),
		); txErr != nil {
			return txErr
		}
		updated, txErr = s.repo.Update(txCtx, entity)
		if txErr != nil {
			return txErr
		}

		return s.setInvoiceDisputeFlag(txCtx, inv, invoice.DisputeStatusNone, actor)
	})
	if err != nil {
		return nil, err
	}

	s.logAudit(updated, actor, permission.OpApprove, previous, "Invoice dispute resolved")
	s.publish(ctx, updated, actor, "updated")

	return updated, nil
}

func (s *Service) Withdraw(
	ctx context.Context,
	req *invoiceports.WithdrawInvoiceDisputeRequest,
	actor *permissionports.RequestActor,
) (*invoicedispute.InvoiceDispute, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Request is required",
		)
	}
	if actor == nil || actor.UserID.IsNil() {
		return nil, errortypes.NewAuthorizationError(
			"Withdrawing a dispute requires an authenticated user",
		)
	}
	if multiErr := validateWithdrawRequest(req); multiErr != nil {
		return nil, multiErr
	}

	var previous, updated *invoicedispute.InvoiceDispute
	err := s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		entity, inv, txErr := s.loadCase(txCtx, req.DisputeID, req.TenantInfo, true, verbWithdrawn)
		if txErr != nil {
			return txErr
		}

		snapshot := *entity
		previous = &snapshot
		if txErr = planWithdraw(req, entity, actor, timeutils.NowUnix()); txErr != nil {
			return txErr
		}
		updated, txErr = s.repo.Update(txCtx, entity)
		if txErr != nil {
			return txErr
		}

		return s.setInvoiceDisputeFlag(txCtx, inv, invoice.DisputeStatusNone, actor)
	})
	if err != nil {
		return nil, err
	}

	s.logAudit(updated, actor, permission.OpCancel, previous, "Invoice dispute withdrawn")
	s.publish(ctx, updated, actor, "updated")

	return updated, nil
}

// validateResolutionAdjustment checks that a credit or write-off resolution
// points at an adjustment that actually executed against this invoice.
func (s *Service) validateResolutionAdjustment(
	ctx context.Context,
	req *invoiceports.ResolveInvoiceDisputeRequest,
	inv *invoice.Invoice,
) error {
	if req.ResolutionAdjustmentID.IsNil() {
		if req.Resolution.RequiresAdjustment() {
			return errortypes.NewValidationError(
				"resolutionAdjustmentId",
				errortypes.ErrRequired,
				"Name the executed adjustment that settled this dispute",
			)
		}
		return nil
	}

	adjustment, err := s.adjustmentRepo.GetByID(
		ctx,
		invoiceadjustmentports.GetInvoiceAdjustmentRequest{
			ID:         req.ResolutionAdjustmentID,
			TenantInfo: req.TenantInfo,
		},
	)
	if err != nil {
		return err
	}
	if adjustment.OriginalInvoiceID != inv.ID {
		return errortypes.NewValidationError(
			"resolutionAdjustmentId",
			errortypes.ErrInvalid,
			"The adjustment does not belong to invoice {0}",
			inv.Number,
		)
	}
	if adjustment.Status != invoiceadjustment.StatusExecuted {
		return errortypes.NewValidationError(
			"resolutionAdjustmentId",
			errortypes.ErrInvalidOperation,
			"The adjustment has not executed yet",
		)
	}

	return nil
}

// setInvoiceDisputeFlag keeps the invoice's quick-read flag in step with its
// open case. It writes only when the flag changes.
func (s *Service) setInvoiceDisputeFlag(
	ctx context.Context,
	inv *invoice.Invoice,
	status invoice.DisputeStatus,
	actor *permissionports.RequestActor,
) error {
	if inv.DisputeStatus == status {
		return nil
	}
	previous := *inv
	inv.DisputeStatus = status
	updated, err := s.invoiceRepo.Update(ctx, inv)
	if err != nil {
		return err
	}

	params := &auditports.LogActionParams{
		Resource:       permission.ResourceInvoice,
		ResourceID:     updated.ID.String(),
		Operation:      permission.OpUpdate,
		UserID:         actor.UserID,
		APIKeyID:       actor.APIKeyID,
		PrincipalType:  actor.PrincipalType,
		PrincipalID:    actor.PrincipalID,
		OrganizationID: updated.OrganizationID,
		BusinessUnitID: updated.BusinessUnitID,
		CurrentState:   jsonutils.MustToJSON(updated),
		PreviousState:  jsonutils.MustToJSON(&previous),
	}
	if logErr := s.auditService.LogAction(
		params,
		auditservice.WithComment("Invoice dispute status changed to "+string(status)),
		auditservice.WithDiff(&previous, updated),
	); logErr != nil {
		s.l.Warn("failed to log invoice dispute flag audit", zap.Error(logErr))
	}
	if pubErr := realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: updated.OrganizationID,
		BusinessUnitID: updated.BusinessUnitID,
		ActorUserID:    actor.UserID,
		ActorType:      actor.PrincipalType,
		ActorID:        actor.PrincipalID,
		ActorAPIKeyID:  actor.APIKeyID,
		Resource:       permission.ResourceInvoice.String(),
		Action:         "updated",
		RecordID:       updated.ID,
		Entity:         updated,
	}); pubErr != nil {
		s.l.Warn("failed to publish invoice invalidation", zap.Error(pubErr))
	}

	return nil
}

func (s *Service) logAudit(
	entity *invoicedispute.InvoiceDispute,
	actor *permissionports.RequestActor,
	op permission.Operation,
	previous *invoicedispute.InvoiceDispute,
	comment string,
) {
	if entity == nil {
		return
	}
	params := &auditports.LogActionParams{
		Resource:       permission.ResourceInvoiceDispute,
		ResourceID:     entity.ID.String(),
		Operation:      op,
		UserID:         actor.UserID,
		APIKeyID:       actor.APIKeyID,
		PrincipalType:  actor.PrincipalType,
		PrincipalID:    actor.PrincipalID,
		OrganizationID: entity.OrganizationID,
		BusinessUnitID: entity.BusinessUnitID,
		CurrentState:   jsonutils.MustToJSON(entity),
	}
	opts := []auditports.LogOption{auditservice.WithComment(comment)}
	if previous != nil {
		params.PreviousState = jsonutils.MustToJSON(previous)
		opts = append(opts, auditservice.WithDiff(previous, entity))
	}
	if err := s.auditService.LogAction(params, opts...); err != nil {
		s.l.Warn("failed to log invoice dispute audit", zap.Error(err))
	}
}

func (s *Service) publish(
	ctx context.Context,
	entity *invoicedispute.InvoiceDispute,
	actor *permissionports.RequestActor,
	action string,
) {
	if err := realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: entity.OrganizationID,
		BusinessUnitID: entity.BusinessUnitID,
		ActorUserID:    actor.UserID,
		ActorType:      actor.PrincipalType,
		ActorID:        actor.PrincipalID,
		ActorAPIKeyID:  actor.APIKeyID,
		Resource:       permission.ResourceInvoiceDispute.String(),
		Action:         action,
		RecordID:       entity.ID,
		Entity:         entity,
	}); err != nil {
		s.l.Warn("failed to publish invoice dispute invalidation", zap.Error(err))
	}
}

func validateOpenRequest(req *invoiceports.OpenInvoiceDisputeRequest) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	if req.InvoiceID.IsNil() {
		multiErr.Add("invoiceId", errortypes.ErrRequired, "Invoice is required")
	}
	if !req.ReasonCode.IsValid() {
		multiErr.Add("reasonCode", errortypes.ErrInvalid, "Choose a dispute reason")
	}
	if req.DisputedAmount.LessThanOrEqual(decimal.Zero) {
		multiErr.Add(
			"disputedAmount",
			errortypes.ErrInvalid,
			"Disputed amount must be greater than zero",
		)
	}
	if len(req.Notes) > maxDisputeNotesLength {
		multiErr.Add(
			"notes",
			errortypes.ErrInvalid,
			"Notes must be at most {0} characters",
			maxDisputeNotesLength,
		)
	}
	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func validateResolveRequest(
	req *invoiceports.ResolveInvoiceDisputeRequest,
) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	if req.DisputeID.IsNil() {
		multiErr.Add("disputeId", errortypes.ErrRequired, "Dispute is required")
	}
	if !req.Resolution.IsValid() {
		multiErr.Add("resolution", errortypes.ErrInvalid, "Choose how the dispute was resolved")
	}
	if len(req.ResolutionNotes) > maxDisputeNotesLength {
		multiErr.Add(
			"resolutionNotes",
			errortypes.ErrInvalid,
			"Notes must be at most {0} characters",
			maxDisputeNotesLength,
		)
	}
	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

// validateDisputableInvoice says whether an invoice can carry a dispute: it
// must be a posted invoice or debit memo with something still owed, and the
// disputed amount cannot exceed that.
func validateDisputableInvoice(
	inv *invoice.Invoice,
	amount decimal.Decimal,
) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	switch {
	case inv.Status == invoice.StatusVoided:
		multiErr.Add(
			"invoiceId",
			errortypes.ErrInvalidOperation,
			"A voided invoice cannot be disputed",
		)
	case inv.Status != invoice.StatusPosted:
		multiErr.Add(
			"invoiceId",
			errortypes.ErrInvalidOperation,
			"Only a posted invoice can be disputed",
		)
	case inv.BillType != billingqueue.BillTypeInvoice && inv.BillType != billingqueue.BillTypeDebitMemo:
		multiErr.Add(
			"invoiceId",
			errortypes.ErrInvalidOperation,
			"Only invoices and debit memos can be disputed",
		)
	case inv.OpenBalanceMinor() <= 0:
		multiErr.Add(
			"invoiceId",
			errortypes.ErrInvalidOperation,
			"Invoice {0} has no open balance to dispute",
			inv.Number,
		)
	case money.MinorUnits(amount) > inv.OpenBalanceMinor():
		multiErr.Add(
			"disputedAmount",
			errortypes.ErrInvalid,
			"Disputed amount exceeds the open balance of {0}",
			inv.OpenBalanceAmount().StringFixed(2),
		)
	}
	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func validateEntity(entity *invoicedispute.InvoiceDispute) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

var _ = pulid.Nil
