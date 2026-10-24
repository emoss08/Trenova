package capture

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

const (
	// ScanRequestLifetimeSeconds is how long a scan request waits for its device.
	// A person who clicked Scan is standing at the scanner; a request that
	// reaches the device ten minutes later would start a scan nobody is there
	// to feed.
	ScanRequestLifetimeSeconds = 2 * 60
	// PrintRequestLifetimeSeconds is how long an armed print destination holds.
	// It is longer because finding the right file and printing it takes longer
	// than feeding a scanner.
	PrintRequestLifetimeSeconds = 10 * 60
	maxFailureMessageLength     = 500
	maxSourceNameLength         = 255
)

// CaptureRequest is a person asking, from the web app, for their device to capture
// pages into a particular record. It never reaches the device directly: the
// device learns of it from its stream and fetches it, so the browser never
// talks to anything on the person's machine.
type CaptureRequest struct {
	keel.Table `keel:"capture_requests"`

	ID             pulid.ID           `json:"id" keel:"id,pk,autoid:creq_"`
	BusinessUnitID pulid.ID           `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID           `json:"organizationId" keel:"organization_id,pk,tenant"`
	UserID         pulid.ID           `json:"userId" keel:"user_id"`
	DeviceID       pulid.ID           `json:"deviceId" keel:"device_id"`
	Mode           RequestMode        `json:"mode" keel:"mode"`
	Status         RequestStatus      `json:"status" keel:"status"`
	TargetType     string             `json:"targetType" keel:"target_type"`
	TargetID       pulid.ID           `json:"targetId" keel:"target_id"`
	DocumentTypeID *pulid.ID          `json:"documentTypeId" keel:"document_type_id"`
	ProfileID      *pulid.ID          `json:"profileId" keel:"profile_id"`
	SourceName     string             `json:"sourceName" keel:"source_name,nullzero"`
	BatchID        *pulid.ID          `json:"batchId" keel:"batch_id"`
	FailureCode    RequestFailureCode `json:"failureCode" keel:"failure_code,nullzero"`
	FailureMessage string             `json:"failureMessage" keel:"failure_message,nullzero"`
	ExpiresAt      int64              `json:"expiresAt" keel:"expires_at"`
	DeliveredAt    *int64             `json:"deliveredAt" keel:"delivered_at"`
	CompletedAt    *int64             `json:"completedAt" keel:"completed_at"`
	Version        int64              `json:"version" keel:"version,version"`
	CreatedAt      int64              `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt      int64              `json:"updatedAt" keel:"updated_at,autoupdate"`

	CaptureProfile *CaptureProfile      `json:"profile,omitempty" keel:"belongs:profile_id"`
	Organization   *tenant.Organization `json:"organization,omitempty"`
	BusinessUnit   *tenant.BusinessUnit `json:"businessUnit,omitempty"`
}

func (r *CaptureRequest) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(r,
		validation.Field(&r.UserID, validation.Required.Error("User is required")),
		validation.Field(&r.DeviceID, validation.Required.Error("Device is required")),
		validation.Field(&r.Mode,
			validation.Required.Error("Mode is required"),
			domainvalidation.ValidEnum[RequestMode]("Mode must be Scan or Print"),
		),
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[RequestStatus]("Status is not a request status"),
		),
		validation.Field(&r.SourceName, validation.Length(0, maxSourceNameLength)),
		validation.Field(
			&r.FailureCode,
			domainvalidation.ValidEnum[RequestFailureCode](
				"Failure code is not one Trenova recognises",
			),
		),
		validation.Field(&r.FailureMessage, validation.Length(0, maxFailureMessageLength)),
	))

	r.Target().Validate(multiErr, targetFields, true)

	if r.Mode == RequestModePrint && r.ProfileID != nil {
		multiErr.Add("profileId", errortypes.ErrInvalid, "A print has no scan profile")
	}
}

// Target is where the request's pages are to be filed.
func (r *CaptureRequest) Target() Target {
	id := r.TargetID
	return Target{ResourceType: r.TargetType, ResourceID: &id, DocumentTypeID: r.DocumentTypeID}
}

// Lifetime is how long a request of this mode waits for its device.
func (m RequestMode) Lifetime() int64 {
	if m == RequestModePrint {
		return PrintRequestLifetimeSeconds
	}

	return ScanRequestLifetimeSeconds
}

// IsExpired reports whether a request still waiting on its device has run out
// of time. One the device has started is not expired by the clock: a scan of
// two hundred pages outlives any lifetime short enough to be useful.
func (r *CaptureRequest) IsExpired(now int64) bool {
	switch r.Status {
	case RequestPending, RequestDelivered:
		return now >= r.ExpiresAt
	case RequestInProgress, RequestCompleted, RequestCanceled, RequestExpired, RequestFailed:
		return false
	default:
		return false
	}
}

// IsOpen reports whether the device may still act on the request.
func (r *CaptureRequest) IsOpen(now int64) bool {
	return !r.Status.Terminal() && !r.IsExpired(now)
}

// Transition moves the request, refusing a move its lifecycle does not allow.
func (r *CaptureRequest) Transition(next RequestStatus, now int64) bool {
	if !r.Status.CanMoveTo(next) {
		return false
	}

	r.Status = next
	switch next {
	case RequestDelivered:
		r.DeliveredAt = &now
	case RequestCompleted, RequestCanceled, RequestExpired, RequestFailed:
		r.CompletedAt = &now
	case RequestPending, RequestInProgress:
	}

	return true
}

func (r *CaptureRequest) GetID() pulid.ID      { return r.ID }
func (r *CaptureRequest) GetCreatedAt() int64  { return r.CreatedAt }
func (r *CaptureRequest) GetTableName() string { return "capture_requests" }

func (r *CaptureRequest) BeforeCreate(ctx context.Context) error {
	if r.Status == "" {
		r.Status = RequestPending
	}

	return nil
}
