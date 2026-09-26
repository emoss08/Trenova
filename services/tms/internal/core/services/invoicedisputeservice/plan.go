package invoicedisputeservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	servicesports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

func (s *Service) planOpen(
	ctx context.Context,
	req *servicesports.OpenInvoiceDisputeRequest,
	inv *invoice.Invoice,
	actor *servicesports.RequestActor,
	now int64,
) (*invoice.InvoiceDispute, error) {
	if multiErr := validateDisputableInvoice(inv, req.DisputedAmount); multiErr != nil {
		return nil, multiErr
	}
	if _, err := s.repo.GetOpenByInvoiceID(ctx, repositories.GetOpenInvoiceDisputeRequest{
		InvoiceID:  inv.ID,
		TenantInfo: req.TenantInfo,
	}); err == nil {
		return nil, errortypes.NewValidationError(
			"invoiceId",
			errortypes.ErrInvalidOperation,
			"Invoice {0} already has an open dispute",
			inv.Number,
		)
	} else if !errortypes.IsNotFoundError(err) {
		return nil, err
	}

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
		return nil, multiErr
	}

	return entity, nil
}

func (s *Service) planResolve(
	ctx context.Context,
	req *servicesports.ResolveInvoiceDisputeRequest,
	entity *invoice.InvoiceDispute,
	inv *invoice.Invoice,
	actor *servicesports.RequestActor,
	now int64,
) error {
	if err := refuseClosedCase(entity, verbResolved); err != nil {
		return err
	}
	if err := s.validateResolutionAdjustment(ctx, req, inv); err != nil {
		return err
	}

	entity.Status = invoice.DisputeCaseStatusResolved
	entity.Resolution = req.Resolution
	entity.ResolutionAdjustmentID = req.ResolutionAdjustmentID
	entity.ResolutionNotes = strings.TrimSpace(req.ResolutionNotes)
	entity.ResolvedByID = actor.UserID
	entity.ResolvedAt = &now
	if multiErr := validateEntity(entity); multiErr != nil {
		return multiErr
	}

	return nil
}

func planWithdraw(
	req *servicesports.WithdrawInvoiceDisputeRequest,
	entity *invoice.InvoiceDispute,
	actor *servicesports.RequestActor,
	now int64,
) error {
	if err := refuseClosedCase(entity, verbWithdrawn); err != nil {
		return err
	}

	entity.Status = invoice.DisputeCaseStatusWithdrawn
	entity.ResolvedByID = actor.UserID
	entity.ResolvedAt = &now
	if notes := strings.TrimSpace(req.Notes); notes != "" {
		entity.ResolutionNotes = notes
	}

	return nil
}

func validateWithdrawRequest(
	req *servicesports.WithdrawInvoiceDisputeRequest,
) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	if req.DisputeID.IsNil() {
		multiErr.Add("disputeId", errortypes.ErrRequired, "Dispute is required")
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

const (
	verbResolved  = "resolved"
	verbWithdrawn = "withdrawn"
)

func refuseClosedCase(entity *invoice.InvoiceDispute, verb string) error {
	if entity.IsOpen() {
		return nil
	}

	return errortypes.NewValidationError(
		"disputeId",
		errortypes.ErrInvalidOperation,
		"Only an open dispute can be "+verb,
	)
}

func (s *Service) loadCase(
	ctx context.Context,
	disputeID pulid.ID,
	tenantInfo pagination.TenantInfo,
	lock bool,
	verb string,
) (*invoice.InvoiceDispute, *invoice.Invoice, error) {
	entity, err := s.repo.GetByID(ctx, repositories.GetInvoiceDisputeByIDRequest{
		ID:         disputeID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, nil, err
	}
	if err = refuseClosedCase(entity, verb); err != nil {
		return nil, nil, err
	}

	invoiceReq := repositories.GetInvoiceByIDRequest{ID: entity.InvoiceID, TenantInfo: tenantInfo}
	var inv *invoice.Invoice
	if lock {
		inv, err = s.invoiceRepo.LockForUpdate(ctx, invoiceReq)
	} else {
		inv, err = s.invoiceRepo.GetByID(ctx, invoiceReq)
	}
	if err != nil {
		return nil, nil, err
	}

	return entity, inv, nil
}

func (s *Service) PreviewOpen(
	ctx context.Context,
	req *servicesports.OpenInvoiceDisputeRequest,
	actor *servicesports.RequestActor,
) (*servicesports.InvoiceDisputePreview, error) {
	if err := requireRequestAndUser(req == nil, actor); err != nil {
		return nil, err
	}
	if multiErr := validateOpenRequest(req); multiErr != nil {
		return nil, multiErr
	}

	inv, err := s.invoiceRepo.GetByID(ctx, repositories.GetInvoiceByIDRequest{
		ID:         req.InvoiceID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	entity, err := s.planOpen(ctx, req, inv, actor, timeutils.NowUnix())
	if err != nil {
		return nil, err
	}

	return disputePreview(nil, entity, inv, invoice.DisputeStatusDisputed), nil
}

func (s *Service) PreviewResolve(
	ctx context.Context,
	req *servicesports.ResolveInvoiceDisputeRequest,
	actor *servicesports.RequestActor,
) (*servicesports.InvoiceDisputePreview, error) {
	if err := requireRequestAndUser(req == nil, actor); err != nil {
		return nil, err
	}
	if multiErr := validateResolveRequest(req); multiErr != nil {
		return nil, multiErr
	}

	entity, inv, err := s.loadCase(ctx, req.DisputeID, req.TenantInfo, false, verbResolved)
	if err != nil {
		return nil, err
	}
	before := *entity
	if err = s.planResolve(ctx, req, entity, inv, actor, timeutils.NowUnix()); err != nil {
		return nil, err
	}

	return disputePreview(&before, entity, inv, invoice.DisputeStatusNone), nil
}

func (s *Service) PreviewWithdraw(
	ctx context.Context,
	req *servicesports.WithdrawInvoiceDisputeRequest,
	actor *servicesports.RequestActor,
) (*servicesports.InvoiceDisputePreview, error) {
	if err := requireRequestAndUser(req == nil, actor); err != nil {
		return nil, err
	}
	if multiErr := validateWithdrawRequest(req); multiErr != nil {
		return nil, multiErr
	}

	entity, inv, err := s.loadCase(ctx, req.DisputeID, req.TenantInfo, false, verbWithdrawn)
	if err != nil {
		return nil, err
	}
	before := *entity
	if err = planWithdraw(req, entity, actor, timeutils.NowUnix()); err != nil {
		return nil, err
	}

	return disputePreview(&before, entity, inv, invoice.DisputeStatusNone), nil
}

func requireRequestAndUser(missing bool, actor *servicesports.RequestActor) error {
	if missing {
		return errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Request is required",
		)
	}
	if actor == nil || actor.UserID.IsNil() {
		return errortypes.NewAuthorizationError(
			"Changing a dispute requires an authenticated user",
		)
	}

	return nil
}

func disputePreview(
	before, after *invoice.InvoiceDispute,
	inv *invoice.Invoice,
	flag invoice.DisputeStatus,
) *servicesports.InvoiceDisputePreview {
	invoiceAfter := *inv
	invoiceAfter.DisputeStatus = flag

	return &servicesports.InvoiceDisputePreview{
		Before:        before,
		After:         after,
		InvoiceBefore: inv,
		InvoiceAfter:  &invoiceAfter,
	}
}
