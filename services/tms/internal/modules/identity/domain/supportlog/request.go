package supportlog

import (
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	RequestIDPrefix         = "slr_"
	MaxFailureMessageLength = 500
	MaxRequesterNameLength  = 255
	MaxRequesterEmailLength = 254
	WaitingLimit            = 10
	RecentLimit             = 10
	DailyLimit              = 5
	DailyWindow             = 24 * time.Hour
	PickUpTimeout           = 30 * time.Minute
	CollectTimeout          = 2 * time.Hour
	RetentionDays           = 30
)

const (
	MessageNotPickedUp = "The instance did not start collecting the logs within 30 minutes. " +
		"Try again, and contact Trenova support if it happens again."
	MessageNotFinished = "Collecting the logs did not finish within 2 hours. " +
		"Try a shorter period, and contact Trenova support if it happens again."
	MessageFailedWithoutReason = "The instance could not collect the logs."
)

var (
	ErrRequestNotFound  = errors.New("log request not found")
	ErrFinished         = errors.New("the log request has already finished with another status")
	ErrBundleMismatch   = errors.New("the log request is already registered under another bundle")
	ErrChangedMeanwhile = errors.New("the log request changed while it was being updated")

	bundleIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,100}$`)
)

type LogRequest struct {
	keel.Table `keel:"support_log_requests"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:slr_"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	RequestedByID  pulid.ID `json:"requestedById"  keel:"requested_by_id"`
	RequesterName  string   `json:"requesterName"  keel:"requester_name"`
	RequesterEmail string   `json:"requesterEmail" keel:"requester_email"`
	Period         Period   `json:"period"         keel:"period"`
	Since          int64    `json:"since"          keel:"since"`
	Until          int64    `json:"until"          keel:"until"`
	Status         Status   `json:"status"         keel:"status"`
	BundleID       string   `json:"bundleId"       keel:"bundle_id,nullzero"`
	FailureMessage string   `json:"failureMessage" keel:"failure_message,nullzero"`
	CollectingAt   *int64   `json:"collectingAt"   keel:"collecting_at"`
	FinishedAt     *int64   `json:"finishedAt"     keel:"finished_at"`
	Version        int64    `json:"version"        keel:"version,version"`
	CreatedAt      int64    `json:"createdAt"      keel:"created_at,autocreate"`
	UpdatedAt      int64    `json:"updatedAt"      keel:"updated_at,autoupdate"`
}

func (LogRequest) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "uq_support_log_requests_in_flight",
			Field:   "period",
			Message: "Logs are already being sent to Trenova support. Wait until that request has finished",
		},
		{Name: "ck_support_log_requests_period", Field: "period", Message: "Choose the last hour, the last 24 hours or the last 7 days"},
	}
}

type NewRequest struct {
	RequestedByID  pulid.ID
	RequesterName  string
	RequesterEmail string
	Period         Period
	Now            int64
}

func New(req *NewRequest) *LogRequest {
	return &LogRequest{
		RequestedByID:  req.RequestedByID,
		RequesterName:  strings.TrimSpace(req.RequesterName),
		RequesterEmail: strings.TrimSpace(req.RequesterEmail),
		Period:         req.Period,
		Since:          req.Now - int64(req.Period.Duration()/time.Second),
		Until:          req.Now,
		Status:         StatusRequested,
		CreatedAt:      req.Now,
	}
}

func (r *LogRequest) Validate(multiErr *errortypes.MultiError) {
	if !r.Period.IsValid() {
		multiErr.Add("period", errortypes.ErrInvalid, "Choose the last hour, the last 24 hours or the last 7 days")
	}
	if r.RequestedByID.IsNil() {
		multiErr.Add("requestedById", errortypes.ErrRequired, "Only a person in this organization may send logs to Trenova support")
	}
	if r.RequesterName == "" || utf8.RuneCountInString(r.RequesterName) > MaxRequesterNameLength {
		multiErr.Add("requesterName", errortypes.ErrInvalid, "Your profile needs a name before you can send logs to Trenova support")
	}
	if r.RequesterEmail == "" || len(r.RequesterEmail) > MaxRequesterEmailLength {
		multiErr.Add("requesterEmail", errortypes.ErrInvalid, "Your profile needs an email address before you can send logs to Trenova support")
	}
	if !r.Status.IsValid() {
		multiErr.Add("status", errortypes.ErrInvalid, "The log request's status is not one this instance knows")
	}
	if r.Until <= r.Since {
		multiErr.Add("period", errortypes.ErrInvalid, "Choose the last hour, the last 24 hours or the last 7 days")
	}
}

type Report struct {
	Status   Status
	BundleID string
	Message  string
}

func (r *LogRequest) Apply(report *Report, now int64) (bool, error) {
	multiErr := errortypes.NewMultiError()
	if report == nil || !report.Status.IsValid() || report.Status == StatusRequested {
		multiErr.Add("status", errortypes.ErrInvalid, "status must be collecting, sent or failed")
		return false, multiErr.Err()
	}
	bundleID := strings.TrimSpace(report.BundleID)
	if bundleID != "" && !bundleIDPattern.MatchString(bundleID) {
		multiErr.Add(
			"bundleId",
			errortypes.ErrInvalid,
			"bundleId must be 1 to 100 letters, digits, dashes or underscores",
		)
		return false, multiErr.Err()
	}
	message := strings.TrimSpace(report.Message)
	if utf8.RuneCountInString(message) > MaxFailureMessageLength {
		multiErr.Add("message", errortypes.ErrInvalid, "message must be at most 500 characters")
		return false, multiErr.Err()
	}
	if bundleID != "" && r.BundleID != "" && bundleID != r.BundleID {
		return false, ErrBundleMismatch
	}

	if r.Status.Finished() {
		if report.Status != r.Status {
			return false, ErrFinished
		}
		return false, nil
	}

	changed := false
	if bundleID != "" && r.BundleID == "" {
		r.BundleID = bundleID
		changed = true
	}

	switch report.Status {
	case StatusCollecting:
		if r.Status == StatusRequested {
			r.Status = StatusCollecting
			r.CollectingAt = &now
			changed = true
		}
	case StatusSent:
		if r.BundleID == "" {
			multiErr.Add(
				"bundleId",
				errortypes.ErrRequired,
				"bundleId is required to mark a log request sent",
			)
			return false, multiErr.Err()
		}
		r.Status = StatusSent
		r.FinishedAt = &now
		changed = true
	case StatusFailed:
		if message == "" {
			message = MessageFailedWithoutReason
		}
		r.fail(message, now)
		changed = true
	case StatusRequested:
	}
	return changed, nil
}

func (r *LogRequest) Overdue(now int64) (string, bool) {
	switch r.Status {
	case StatusRequested:
		if r.CreatedAt <= now-int64(PickUpTimeout/time.Second) {
			return MessageNotPickedUp, true
		}
	case StatusCollecting:
		if r.CollectingAt != nil && *r.CollectingAt <= now-int64(CollectTimeout/time.Second) {
			return MessageNotFinished, true
		}
	case StatusSent, StatusFailed:
	}
	return "", false
}

func (r *LogRequest) FailOverdue(now int64) bool {
	message, overdue := r.Overdue(now)
	if !overdue {
		return false
	}
	r.fail(message, now)
	return true
}

func (r *LogRequest) fail(message string, now int64) {
	r.Status = StatusFailed
	r.FailureMessage = message
	r.FinishedAt = &now
}

func (r *LogRequest) GetID() pulid.ID {
	return r.ID
}

func (r *LogRequest) GetOrganizationID() pulid.ID {
	return r.OrganizationID
}

func (r *LogRequest) GetBusinessUnitID() pulid.ID {
	return r.BusinessUnitID
}

func (r *LogRequest) GetCreatedAt() int64 {
	return r.CreatedAt
}

func (r *LogRequest) GetTableName() string {
	return "support_log_requests"
}

type Load struct {
	InFlight bool
	Recent   int
}

func (l *Load) Remaining() int {
	if l == nil {
		return DailyLimit
	}
	return max(DailyLimit-l.Recent, 0)
}
