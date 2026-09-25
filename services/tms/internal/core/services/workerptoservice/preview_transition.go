package workerptoservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
)

// PreviewReject is Reject without the write: the request as rejecting it
// would leave it, and what the driver would be told and texted.
func (s *Service) PreviewReject(
	ctx context.Context,
	req *repositories.UpdatePTOStatusRequest,
) (*services.WorkerPTOTransitionPreview, error) {
	if err := requireRejectionReason(req); err != nil {
		return nil, err
	}

	return s.previewTransition(ctx, transitionParams{
		req:          req,
		target:       worker.PTOStatusRejected,
		notifyWorker: true,
	})
}

// PreviewCancel is Cancel without the write.
func (s *Service) PreviewCancel(
	ctx context.Context,
	req *repositories.UpdatePTOStatusRequest,
) (*services.WorkerPTOTransitionPreview, error) {
	return s.previewTransition(ctx, transitionParams{
		req:          req,
		target:       worker.PTOStatusCancelled,
		notifyWorker: true,
	})
}

func requireRejectionReason(req *repositories.UpdatePTOStatusRequest) error {
	if strings.TrimSpace(req.Reason) == "" {
		return errortypes.NewValidationError(
			"reason",
			errortypes.ErrRequired,
			"A reason is required to reject a PTO request",
		)
	}

	return nil
}

func (s *Service) previewTransition(
	ctx context.Context,
	params transitionParams,
) (*services.WorkerPTOTransitionPreview, error) {
	request := *params.req
	params.req = &request

	current, err := s.planTransition(ctx, params)
	if err != nil {
		return nil, err
	}

	after := projectTransition(current, params.req)
	preview := &services.WorkerPTOTransitionPreview{
		Before:        current,
		After:         after,
		ReturnsLedger: releasesUsage(current, params.target),
	}
	if !params.notifyWorker {
		return preview, nil
	}

	if notice, ok := transitionNotice(params.target, params.req.Reason); ok && s.driverNotify != nil {
		preview.Driver, err = s.driverNotify.Preview(
			ctx,
			ptoDriverNotification(params.req.TenantInfo, after, notice),
		)
		if err != nil {
			return nil, err
		}
	}

	if s.sendsSMS() {
		preview.SMS, err = s.transitionSMS(ctx, params.req, after, params.target)
		if err != nil {
			return nil, err
		}
	}

	return preview, nil
}

// planTransition checks a status change the way the write does: the request
// must be able to move to the target, an approval must fit the balance, and
// the version the change is made against is the one read here.
func (s *Service) planTransition(
	ctx context.Context,
	params transitionParams,
) (*worker.WorkerPTO, error) {
	req := params.req
	current := params.current
	if current == nil {
		var err error
		current, err = s.repo.GetByID(ctx, &repositories.GetPTOByIDRequest{
			ID:         req.ID,
			TenantInfo: req.TenantInfo,
		})
		if err != nil {
			return nil, err
		}
	}

	if !current.Status.CanTransitionTo(params.target) {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"PTO is {0} and cannot be {1}",
			strings.ToLower(string(current.Status)),
			strings.ToLower(string(params.target)),
		)
	}

	req.Status = params.target
	req.Reason = strings.TrimSpace(req.Reason)
	if req.ExpectedVersion == 0 {
		req.ExpectedVersion = current.Version
	}

	if params.target == worker.PTOStatusApproved {
		if err := s.requireAvailability(ctx, current, current.ID); err != nil {
			return nil, err
		}
	}

	return current, nil
}

// projectTransition is the request as UpdateStatus leaves it: the status,
// who made the decision, and the reason recorded against it.
func projectTransition(
	current *worker.WorkerPTO,
	req *repositories.UpdatePTOStatusRequest,
) *worker.WorkerPTO {
	after := *current
	after.Status = req.Status
	switch req.Status { //nolint:exhaustive // Requested is never a transition target
	case worker.PTOStatusApproved:
		after.ApproverID = req.UserID
	case worker.PTOStatusRejected:
		after.RejectorID = req.UserID
		after.RejectionReason = req.Reason
	case worker.PTOStatusCancelled:
		after.CancelledByID = req.UserID
		after.CancellationReason = req.Reason
	}

	return &after
}

func releasesUsage(current *worker.WorkerPTO, target worker.PTOStatus) bool {
	return target == worker.PTOStatusCancelled && current.Status == worker.PTOStatusApproved
}

// transitionNotice is the Dash message a decision sends the driver.
func transitionNotice(target worker.PTOStatus, reason string) (driverPTONotice, bool) {
	switch target { //nolint:exhaustive // Requested is never a transition target
	case worker.PTOStatusApproved:
		return driverPTONotice{eventType: "dash.pto_reviewed", approved: true}, true
	case worker.PTOStatusRejected:
		return driverPTONotice{eventType: "dash.pto_reviewed", reason: reason}, true
	case worker.PTOStatusCancelled:
		return driverPTONotice{eventType: "dash.pto_cancelled", reason: reason}, true
	default:
		return driverPTONotice{}, false
	}
}

func (s *Service) sendsSMS() bool {
	return s.workflowStarter != nil && s.workflowStarter.Enabled()
}

// transitionSMS is the text a decision sends the driver's phone, signed with
// the name of the person who made it. A driver with no phone on file is
// sent none, and neither is a decision with no wording.
func (s *Service) transitionSMS(
	ctx context.Context,
	req *repositories.UpdatePTOStatusRequest,
	updated *worker.WorkerPTO,
	target worker.PTOStatus,
) (*services.DriverSMSPreview, error) {
	user, err := s.userRepo.GetByID(ctx, repositories.GetUserByIDRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  req.TenantInfo.OrgID,
			BuID:   req.TenantInfo.BuID,
			UserID: req.UserID,
		},
	})
	if err != nil {
		return nil, err
	}

	wrk, err := s.workerRepo.GetByID(ctx, repositories.GetWorkerByIDRequest{
		ID:         updated.WorkerID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if wrk.PhoneNumber == "" {
		return nil, nil
	}

	message, ok := transitionSMSMessage(user.Name, updated, target, req.Reason)
	if !ok {
		return nil, nil
	}

	return &services.DriverSMSPreview{PhoneNumber: wrk.PhoneNumber, Message: message}, nil
}
