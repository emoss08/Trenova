package capture

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
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
	bun.BaseModel             `bun:"table:capture_requests,alias:creq" json:"-"`
	pagination.CursorValueSet `bun:",embed"                            json:"-"`

	ID             pulid.ID           `json:"id"             bun:"id,pk,type:VARCHAR(100)"`
	BusinessUnitID pulid.ID           `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID           `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	UserID         pulid.ID           `json:"userId"         bun:"user_id,type:VARCHAR(100),notnull"`
	DeviceID       pulid.ID           `json:"deviceId"       bun:"device_id,type:VARCHAR(100),notnull"`
	Mode           RequestMode        `json:"mode"           bun:"mode,type:VARCHAR(10),notnull"`
	Status         RequestStatus      `json:"status"         bun:"status,type:VARCHAR(20),notnull"`
	TargetType     string             `json:"targetType"     bun:"target_type,type:VARCHAR(50),notnull"`
	TargetID       pulid.ID           `json:"targetId"       bun:"target_id,type:VARCHAR(100),notnull"`
	DocumentTypeID *pulid.ID          `json:"documentTypeId" bun:"document_type_id,type:VARCHAR(100),nullzero"`
	ProfileID      *pulid.ID          `json:"profileId"      bun:"profile_id,type:VARCHAR(100),nullzero"`
	SourceName     string             `json:"sourceName"     bun:"source_name,type:VARCHAR(255),nullzero"`
	BatchID        *pulid.ID          `json:"batchId"        bun:"batch_id,type:VARCHAR(100),nullzero"`
	FailureCode    RequestFailureCode `json:"failureCode"    bun:"failure_code,type:VARCHAR(40),nullzero"`
	FailureMessage string             `json:"failureMessage" bun:"failure_message,type:VARCHAR(500),nullzero"`
	ExpiresAt      int64              `json:"expiresAt"      bun:"expires_at,type:BIGINT,notnull"`
	DeliveredAt    *int64             `json:"deliveredAt"    bun:"delivered_at,type:BIGINT,nullzero"`
	CompletedAt    *int64             `json:"completedAt"    bun:"completed_at,type:BIGINT,nullzero"`
	Version        int64              `json:"version"        bun:"version,type:BIGINT"`
	CreatedAt      int64              `json:"createdAt"      bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64              `json:"updatedAt"      bun:"updated_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`

	CaptureProfile *CaptureProfile      `json:"profile,omitempty"      bun:"rel:belongs-to,join:profile_id=id,join:business_unit_id=business_unit_id,join:organization_id=organization_id"`
	Organization   *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	BusinessUnit   *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
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

func (r *CaptureRequest) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if r.ID.IsNil() {
			r.ID = pulid.MustNew("creq_")
		}
		if r.Status == "" {
			r.Status = RequestPending
		}
		r.CreatedAt = now
		r.UpdatedAt = now
	case *bun.UpdateQuery:
		r.UpdatedAt = now
	}

	return nil
}

func (r *CaptureRequest) GetID() pulid.ID      { return r.ID }
func (r *CaptureRequest) GetCreatedAt() int64  { return r.CreatedAt }
func (r *CaptureRequest) GetTableName() string { return "capture_requests" }
