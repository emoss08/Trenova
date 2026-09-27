package captureservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/capture"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

const (
	expireBatchSize       = 200
	maxFailureMessageLen  = 500
	requestActionCreated  = "created"
	requestActionUpdated  = "updated"
	actionCreated         = "created"
	actionUpdated         = "updated"
	actionDeleted         = "deleted"
	defaultTargetRequests = 20
)

type CreateRequestInput struct {
	TenantInfo     pagination.TenantInfo `json:"-"`
	DeviceID       pulid.ID              `json:"deviceId"`
	Mode           capture.RequestMode   `json:"mode"`
	TargetType     string                `json:"targetType"`
	TargetID       pulid.ID              `json:"targetId"`
	DocumentTypeID *pulid.ID             `json:"documentTypeId"`
	ProfileID      *pulid.ID             `json:"profileId"`
	SourceName     string                `json:"sourceName"`
}

// CreateRequest asks one of the person's own devices to capture into a
// record. Only their own: a request is someone standing at a scanner, and
// starting a scan on a machine somebody else is sitting at is not a thing
// this should be able to do.
func (s *Service) CreateRequest(
	ctx context.Context,
	in *CreateRequestInput,
) (*capture.CaptureRequest, error) {
	if _, err := s.require(
		ctx,
		in.TenantInfo,
		permission.ResourceCaptureBatch,
		permission.OpCreate,
	); err != nil {
		return nil, err
	}
	if _, err := s.requireEnabled(ctx, in.TenantInfo); err != nil {
		return nil, err
	}

	device, err := s.ownDevice(ctx, in.TenantInfo, in.DeviceID)
	if err != nil {
		return nil, err
	}

	target := capture.Target{
		ResourceType:   in.TargetType,
		ResourceID:     &in.TargetID,
		DocumentTypeID: in.DocumentTypeID,
	}
	if err = s.checkTarget(ctx, in.TenantInfo, target, "targetType", "targetId"); err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	req := &capture.CaptureRequest{
		OrganizationID: in.TenantInfo.OrgID,
		BusinessUnitID: in.TenantInfo.BuID,
		UserID:         in.TenantInfo.UserID,
		DeviceID:       device.ID,
		Mode:           in.Mode,
		Status:         capture.RequestPending,
		TargetType:     in.TargetType,
		TargetID:       in.TargetID,
		DocumentTypeID: in.DocumentTypeID,
		SourceName:     strings.TrimSpace(in.SourceName),
		ExpiresAt:      now + in.Mode.Lifetime(),
	}

	if in.Mode == capture.RequestModeScan {
		profileID, found, profileErr := s.resolveProfile(ctx, in.TenantInfo, in.ProfileID)
		if profileErr != nil {
			return nil, profileErr
		}
		if found {
			req.ProfileID = &profileID
		}
	} else {
		req.ProfileID = in.ProfileID
	}

	multiErr := errortypes.NewMultiError()
	req.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	if in.Mode == capture.RequestModePrint {
		s.cancelArmedPrints(ctx, in.TenantInfo, device.ID)
	}

	created, err := s.requests.Create(ctx, req)
	if err != nil {
		return nil, err
	}

	s.signalRequest(ctx, created, requestActionCreated)

	return created, nil
}

// cancelArmedPrints disarms any print destination already waiting on the
// device. Only one can be armed: the next print goes to exactly one place,
// and the place is the one the person chose last.
func (s *Service) cancelArmedPrints(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	deviceID pulid.ID,
) {
	open, err := s.requests.ListOpen(ctx, repositories.ListOpenCaptureRequestsRequest{
		TenantInfo: tenantInfo,
		DeviceID:   deviceID,
	})
	if err != nil {
		s.l.Warn(
			"could not read armed prints",
			zap.String("deviceId", deviceID.String()),
			zap.Error(err),
		)

		return
	}

	now := timeutils.NowUnix()
	for _, req := range open {
		if req.Mode != capture.RequestModePrint || req.BatchID != nil {
			continue
		}
		if !req.Transition(capture.RequestCanceled, now) {
			continue
		}
		if _, err = s.requests.Update(ctx, req); err != nil {
			s.l.Warn(
				"could not disarm a print",
				zap.String("requestId", req.ID.String()),
				zap.Error(err),
			)

			continue
		}
		s.signalRequest(ctx, req, requestActionUpdated)
	}
}

// ownDevice is one of the caller's active devices.
func (s *Service) ownDevice(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	deviceID pulid.ID,
) (*capture.CaptureDevice, error) {
	device, err := s.devices.GetByID(ctx, repositories.GetCaptureDeviceByIDRequest{
		ID:         deviceID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, errortypes.NewValidationError(
				"deviceId",
				errortypes.ErrInvalid,
				"Device not found",
			)
		}

		return nil, err
	}
	if device.UserID != tenantInfo.UserID {
		return nil, errortypes.NewValidationError("deviceId", errortypes.ErrInvalid,
			"You can only capture with your own devices")
	}
	if !device.IsActive() {
		return nil, errortypes.NewValidationError("deviceId", errortypes.ErrInvalid,
			"That device has been revoked; pair it again to use it")
	}

	return device, nil
}

// checkTarget confirms a target is somewhere this person may file: a real
// record in their organization, of a kind captures go onto, that they can see
// and add documents to, with a document type that exists.
func (s *Service) checkTarget(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	target capture.Target,
	typeField, idField string,
) error {
	multiErr := errortypes.NewMultiError()
	target.Validate(multiErr, capture.TargetFields{Type: typeField, ID: idField}, true)
	if multiErr.HasErrors() {
		return multiErr
	}

	if _, err := s.require(
		ctx,
		tenantInfo,
		document.OwnerResource(target.ResourceType),
		permission.OpRead,
	); err != nil {
		return err
	}
	if _, err := s.require(
		ctx,
		tenantInfo,
		permission.ResourceDocument,
		permission.OpCreate,
	); err != nil {
		return err
	}

	exists, err := s.records.Exists(ctx, tenantInfo, target.ResourceType, *target.ResourceID)
	if err != nil {
		return err
	}
	if !exists {
		return errortypes.NewValidationError(idField, errortypes.ErrInvalid, "Record not found")
	}

	if target.DocumentTypeID != nil {
		if _, err = s.documentTypes.GetByID(ctx, repositories.GetDocumentTypeByIDRequest{
			ID:         *target.DocumentTypeID,
			TenantInfo: tenantInfo,
		}); err != nil {
			if errortypes.IsNotFoundError(err) {
				return errortypes.NewValidationError("documentTypeId", errortypes.ErrInvalid,
					"Document type not found")
			}

			return err
		}
	}

	return nil
}

// resolveProfile is the profile a scan uses: the one asked for, or the
// tenant's default. It reports none found when there is no default, in which
// case the device uses its own settings.
func (s *Service) resolveProfile(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	profileID *pulid.ID,
) (pulid.ID, bool, error) {
	if profileID == nil {
		profile, err := s.profiles.GetDefault(ctx, tenantInfo)
		if err != nil {
			if errortypes.IsNotFoundError(err) {
				return pulid.Nil, false, nil
			}

			return pulid.Nil, false, err
		}

		return profile.ID, true, nil
	}

	profile, err := s.profiles.GetByID(ctx, repositories.GetCaptureProfileByIDRequest{
		ID:         *profileID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return pulid.Nil, false, errortypes.NewValidationError(
				"profileId",
				errortypes.ErrInvalid,
				"Profile not found",
			)
		}

		return pulid.Nil, false, err
	}
	if profile.Status != capture.ProfileActive {
		return pulid.Nil, false, errortypes.NewValidationError("profileId", errortypes.ErrInvalid,
			"That profile is no longer offered")
	}

	return profile.ID, true, nil
}

// CancelRequest is the person withdrawing a request from the web app.
func (s *Service) CancelRequest(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	requestID pulid.ID,
) (*capture.CaptureRequest, error) {
	req, err := s.requests.GetByID(ctx, repositories.GetCaptureRequestByIDRequest{
		ID:         requestID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if req.UserID != tenantInfo.UserID {
		return nil, errortypes.NewAuthorizationError("You can only cancel your own requests")
	}
	if !req.Transition(capture.RequestCanceled, timeutils.NowUnix()) {
		return req, nil
	}

	updated, err := s.requests.Update(ctx, req)
	if err != nil {
		return nil, err
	}
	s.signalRequest(ctx, updated, requestActionUpdated)

	return updated, nil
}

// OpenRequests is what a device still has to do. Anything whose time ran out
// is expired on the way, so a device coming back from sleep never starts a
// scan nobody is waiting for.
func (s *Service) OpenRequests(
	ctx context.Context,
	principal *DevicePrincipal,
) ([]*capture.CaptureRequest, error) {
	open, err := s.requests.ListOpen(ctx, repositories.ListOpenCaptureRequestsRequest{
		TenantInfo: principal.TenantInfo(),
		DeviceID:   principal.Device.ID,
	})
	if err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	live := make([]*capture.CaptureRequest, 0, len(open))
	for _, req := range open {
		if req.IsExpired(now) {
			s.expire(ctx, req, now)

			continue
		}
		live = append(live, req)
	}

	return live, nil
}

type RequestStatusReport struct {
	RequestID      pulid.ID                   `json:"-"`
	Status         capture.RequestStatus      `json:"status"`
	FailureCode    capture.RequestFailureCode `json:"failureCode"`
	FailureMessage string                     `json:"failureMessage"`
}

// ReportRequestStatus is the device saying how a request is going. A report
// its lifecycle does not allow is ignored and the request returned as it is,
// so a late report after the person cancelled is harmless.
func (s *Service) ReportRequestStatus(
	ctx context.Context,
	principal *DevicePrincipal,
	report *RequestStatusReport,
) (*capture.CaptureRequest, error) {
	req, err := s.deviceRequest(ctx, principal, report.RequestID)
	if err != nil {
		return nil, err
	}

	switch report.Status {
	case capture.RequestDelivered,
		capture.RequestInProgress,
		capture.RequestFailed,
		capture.RequestCanceled:
	case capture.RequestPending, capture.RequestCompleted, capture.RequestExpired:
		return nil, errortypes.NewValidationError("status", errortypes.ErrInvalid,
			"A device may report a request delivered, in progress, failed or cancelled")
	default:
		return nil, errortypes.NewValidationError("status", errortypes.ErrInvalid, "Unknown status")
	}

	if report.Status == capture.RequestFailed || report.Status == capture.RequestCanceled {
		code := report.FailureCode
		if code == "" {
			code = capture.FailureInternal
			if report.Status == capture.RequestCanceled {
				code = capture.FailureCanceledByUser
			}
		}
		if !code.IsValid() {
			return nil, errortypes.NewValidationError("failureCode", errortypes.ErrInvalid,
				"Failure code is not one Trenova recognises")
		}
		req.FailureCode = code
		req.FailureMessage = stringutils.TruncateRunes(
			strings.TrimSpace(report.FailureMessage),
			maxFailureMessageLen,
		)
	}

	if !req.Transition(report.Status, timeutils.NowUnix()) {
		return req, nil
	}

	updated, err := s.requests.Update(ctx, req)
	if err != nil {
		return nil, err
	}
	s.signalRequest(ctx, updated, requestActionUpdated)

	return updated, nil
}

// deviceRequest is a request addressed to this device.
func (s *Service) deviceRequest(
	ctx context.Context,
	principal *DevicePrincipal,
	requestID pulid.ID,
) (*capture.CaptureRequest, error) {
	req, err := s.requests.GetByID(ctx, repositories.GetCaptureRequestByIDRequest{
		ID:         requestID,
		TenantInfo: principal.TenantInfo(),
	})
	if err != nil {
		return nil, err
	}
	if req.DeviceID != principal.Device.ID {
		return nil, errortypes.NewNotFoundError("Capture request not found")
	}

	return req, nil
}

type ListRequestsForTargetInput struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	TargetType string                `json:"targetType"`
	TargetID   pulid.ID              `json:"targetId"`
	Limit      int                   `json:"limit"`
}

// ListRequestsForTarget is the person's recent requests into one record, so
// the record's page can show a scan in flight.
func (s *Service) ListRequestsForTarget(
	ctx context.Context,
	in *ListRequestsForTargetInput,
) ([]*capture.CaptureRequest, error) {
	if _, err := s.require(
		ctx,
		in.TenantInfo,
		permission.ResourceCaptureBatch,
		permission.OpRead,
	); err != nil {
		return nil, err
	}

	limit := in.Limit
	if limit <= 0 {
		limit = defaultTargetRequests
	}

	return s.requests.ListForTarget(ctx, repositories.ListCaptureRequestsForTargetRequest{
		TenantInfo: in.TenantInfo,
		UserID:     in.TenantInfo.UserID,
		TargetType: in.TargetType,
		TargetID:   in.TargetID,
		Limit:      limit,
	})
}

// ExpireRequests closes requests whose device never picked them up.
func (s *Service) ExpireRequests(ctx context.Context) (int, error) {
	now := timeutils.NowUnix()
	expired, err := s.requests.ListExpired(ctx, now, expireBatchSize)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, req := range expired {
		if s.expire(ctx, req, now) {
			count++
		}
	}

	return count, nil
}

func (s *Service) expire(ctx context.Context, req *capture.CaptureRequest, now int64) bool {
	if !req.Transition(capture.RequestExpired, now) {
		return false
	}
	req.FailureCode = capture.FailureNotDelivered

	updated, err := s.requests.Update(ctx, req)
	if err != nil {
		s.l.Warn("could not expire a capture request",
			zap.String("requestId", req.ID.String()), zap.Error(err))

		return false
	}
	s.signalRequest(ctx, updated, requestActionUpdated)

	return true
}

// signalRequest tells the person's screens and their device that a request
// changed. It is addressed to the person, so the rest of the tenant never
// hears about it, and it names the device so the person's other devices
// ignore it.
func (s *Service) signalRequest(ctx context.Context, req *capture.CaptureRequest, action string) {
	tenantInfo := pagination.TenantInfo{
		OrgID:  req.OrganizationID,
		BuID:   req.BusinessUnitID,
		UserID: req.UserID,
	}
	s.publish(
		ctx,
		tenantInfo,
		permission.ResourceCaptureBatch,
		req.ID,
		"request."+action,
		req.UserID,
		DeviceSignal{DeviceID: req.DeviceID, RequestID: req.ID},
	)
}
