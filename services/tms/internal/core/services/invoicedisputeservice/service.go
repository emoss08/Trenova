package invoicedisputeservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	servicesports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const maxDisputeNotesLength = 2000

type Params struct {
	fx.In

	Logger         *zap.Logger
	DB             ports.DBConnection
	Repo           repositories.InvoiceDisputeRepository
	InvoiceRepo    repositories.InvoiceRepository
	AdjustmentRepo repositories.InvoiceAdjustmentRepository
	AuditService   servicesports.AuditService
	Realtime       servicesports.RealtimeService
}

// Service runs the dispute case lifecycle. The invoice's DisputeStatus flag
// is derived: Disputed while a case is Open, None otherwise, and the two are
// changed together under the invoice row lock.
type Service struct {
	l              *zap.Logger
	db             ports.DBConnection
	repo           repositories.InvoiceDisputeRepository
	invoiceRepo    repositories.InvoiceRepository
	adjustmentRepo repositories.InvoiceAdjustmentRepository
	auditService   servicesports.AuditService
	realtime       servicesports.RealtimeService
}

func New(p Params) servicesports.InvoiceDisputeService {
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
	req *servicesports.OpenInvoiceDisputeRequest,
	actor *servicesports.RequestActor,
) (*invoice.InvoiceDispute, error) {
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

	var created *invoice.InvoiceDispute
	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		inv, txErr := s.invoiceRepo.LockForUpdate(txCtx, repositories.GetInvoiceByIDRequest{
			ID:         req.InvoiceID,
			TenantInfo: req.TenantInfo,
		})
		if txErr != nil {
			return txErr
		}
		if multiErr := validateDisputableInvoice(inv, req.DisputedAmount); multiErr != nil {
			return multiErr
		}
		if _, txErr = s.repo.GetOpenByInvoiceID(txCtx, repositories.GetOpenInvoiceDisputeRequest{
			InvoiceID:  inv.ID,
			TenantInfo: req.TenantInfo,
		}); txErr == nil {
			return errortypes.NewValidationError(
				"invoiceId",
				errortypes.ErrInvalidOperation,
				"Invoice {0} already has an open dispute",
				inv.Number,
			)
		} else if !errortypes.IsNotFoundError(txErr) {
			return txErr
		}

		now := timeutils.NowUnix()
		entity := &invoice.InvoiceDispute{
			OrganizationID: req.TenantInfo.OrgID,
			BusinessUnitID: req.TenantInfo.BuID,
			InvoiceID:      inv.ID,
			CustomerID:     inv.CustomerID,
			Status:         invoice.DisputeCaseStatusOpen,
			ReasonCode:     req.ReasonCode,
			DisputedAmount: req.DisputedAmount,
			Notes:          strings.TrimSpace(req.Notes),
			OpenedByID:     actor.UserID,
			OpenedAt:       now,
		}
		entity.SyncMinor()
		if multiErr := validateEntity(entity); multiErr != nil {
			return multiErr
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
	req *servicesports.ResolveInvoiceDisputeRequest,
	actor *servicesports.RequestActor,
) (*invoice.InvoiceDispute, error) {
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

	var previous, updated *invoice.InvoiceDispute
	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		entity, txErr := s.repo.GetByID(txCtx, repositories.GetInvoiceDisputeByIDRequest{
			ID:         req.DisputeID,
			TenantInfo: req.TenantInfo,
		})
		if txErr != nil {
			return txErr
		}
		if !entity.IsOpen() {
			return errortypes.NewValidationError(
				"disputeId",
				errortypes.ErrInvalidOperation,
				"Only an open dispute can be resolved",
			)
		}
		inv, txErr := s.invoiceRepo.LockForUpdate(txCtx, repositories.GetInvoiceByIDRequest{
			ID:         entity.InvoiceID,
			TenantInfo: req.TenantInfo,
		})
		if txErr != nil {
			return txErr
		}
		if txErr = s.validateResolutionAdjustment(txCtx, req, inv); txErr != nil {
			return txErr
		}

		snapshot := *entity
		previous = &snapshot
		now := timeutils.NowUnix()
		entity.Status = invoice.DisputeCaseStatusResolved
		entity.Resolution = req.Resolution
		entity.ResolutionAdjustmentID = req.ResolutionAdjustmentID
		entity.ResolutionNotes = strings.TrimSpace(req.ResolutionNotes)
		entity.ResolvedByID = actor.UserID
		entity.ResolvedAt = &now
		if multiErr := validateEntity(entity); multiErr != nil {
			return multiErr
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
	req *servicesports.WithdrawInvoiceDisputeRequest,
	actor *servicesports.RequestActor,
) (*invoice.InvoiceDispute, error) {
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
	if req.DisputeID.IsNil() {
		return nil, errortypes.NewValidationError(
			"disputeId",
			errortypes.ErrRequired,
			"Dispute is required",
		)
	}
	if len(req.Notes) > maxDisputeNotesLength {
		return nil, errortypes.NewValidationError(
			"notes",
			errortypes.ErrInvalid,
			"Notes must be at most {0} characters",
			maxDisputeNotesLength,
		)
	}

	var previous, updated *invoice.InvoiceDispute
	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		entity, txErr := s.repo.GetByID(txCtx, repositories.GetInvoiceDisputeByIDRequest{
			ID:         req.DisputeID,
			TenantInfo: req.TenantInfo,
		})
		if txErr != nil {
			return txErr
		}
		if !entity.IsOpen() {
			return errortypes.NewValidationError(
				"disputeId",
				errortypes.ErrInvalidOperation,
				"Only an open dispute can be withdrawn",
			)
		}
		inv, txErr := s.invoiceRepo.LockForUpdate(txCtx, repositories.GetInvoiceByIDRequest{
			ID:         entity.InvoiceID,
			TenantInfo: req.TenantInfo,
		})
		if txErr != nil {
			return txErr
		}

		snapshot := *entity
		previous = &snapshot
		now := timeutils.NowUnix()
		entity.Status = invoice.DisputeCaseStatusWithdrawn
		entity.ResolvedByID = actor.UserID
		entity.ResolvedAt = &now
		if notes := strings.TrimSpace(req.Notes); notes != "" {
			entity.ResolutionNotes = notes
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
	req *servicesports.ResolveInvoiceDisputeRequest,
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

	adjustment, err := s.adjustmentRepo.GetByID(ctx, repositories.GetInvoiceAdjustmentRequest{
		ID:         req.ResolutionAdjustmentID,
		TenantInfo: req.TenantInfo,
	})
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
	actor *servicesports.RequestActor,
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

	params := &servicesports.LogActionParams{
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
	entity *invoice.InvoiceDispute,
	actor *servicesports.RequestActor,
	op permission.Operation,
	previous *invoice.InvoiceDispute,
	comment string,
) {
	if entity == nil {
		return
	}
	params := &servicesports.LogActionParams{
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
	opts := []servicesports.LogOption{auditservice.WithComment(comment)}
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
	entity *invoice.InvoiceDispute,
	actor *servicesports.RequestActor,
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

func validateOpenRequest(req *servicesports.OpenInvoiceDisputeRequest) *errortypes.MultiError {
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
	req *servicesports.ResolveInvoiceDisputeRequest,
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

func validateEntity(entity *invoice.InvoiceDispute) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

var _ = pulid.Nil
