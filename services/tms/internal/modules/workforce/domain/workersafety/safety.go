package workersafety

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

var (
	ErrInvalidSafetyEventKind   = errors.New("invalid safety event kind")
	ErrInvalidSafetySeverity    = errors.New("invalid safety severity")
	ErrInvalidSafetyEventStatus = errors.New("invalid safety event status")
	ErrInvalidInspectionResult  = errors.New("invalid inspection result")
)

type SafetyEventKind string

const (
	SafetyEventAccident   = SafetyEventKind("Accident")
	SafetyEventIncident   = SafetyEventKind("Incident")
	SafetyEventNearMiss   = SafetyEventKind("NearMiss")
	SafetyEventCitation   = SafetyEventKind("Citation")
	SafetyEventInspection = SafetyEventKind("Inspection")
)

func (k SafetyEventKind) String() string { return string(k) }

func SafetyEventKindValues() []SafetyEventKind {
	return []SafetyEventKind{
		SafetyEventAccident,
		SafetyEventIncident,
		SafetyEventNearMiss,
		SafetyEventCitation,
		SafetyEventInspection,
	}
}

func (k SafetyEventKind) IsValid() bool {
	switch k {
	case SafetyEventAccident, SafetyEventIncident, SafetyEventNearMiss, SafetyEventCitation,
		SafetyEventInspection:
		return true
	default:
		return false
	}
}

// CountsAgainstRecord reports whether the kind is something that happened to
// the worker's driving record rather than a check that went well.
func (k SafetyEventKind) CountsAgainstRecord() bool {
	return k == SafetyEventAccident || k == SafetyEventIncident || k == SafetyEventCitation
}

type SafetySeverity string

const (
	SafetySeverityMinor    = SafetySeverity("Minor")
	SafetySeverityModerate = SafetySeverity("Moderate")
	SafetySeverityMajor    = SafetySeverity("Major")
	SafetySeverityCritical = SafetySeverity("Critical")
)

func (s SafetySeverity) String() string { return string(s) }

func SafetySeverityValues() []SafetySeverity {
	return []SafetySeverity{
		SafetySeverityMinor,
		SafetySeverityModerate,
		SafetySeverityMajor,
		SafetySeverityCritical,
	}
}

func (s SafetySeverity) IsValid() bool {
	switch s {
	case SafetySeverityMinor, SafetySeverityModerate, SafetySeverityMajor, SafetySeverityCritical:
		return true
	default:
		return false
	}
}

func (s SafetySeverity) rank() int {
	switch s {
	case SafetySeverityMinor:
		return 1
	case SafetySeverityModerate:
		return 2
	case SafetySeverityMajor:
		return 3
	case SafetySeverityCritical:
		return 4
	default:
		return 0
	}
}

type SafetyEventStatus string

const (
	SafetyEventStatusOpen        = SafetyEventStatus("Open")
	SafetyEventStatusUnderReview = SafetyEventStatus("UnderReview")
	SafetyEventStatusClosed      = SafetyEventStatus("Closed")
)

func (s SafetyEventStatus) String() string { return string(s) }

func (s SafetyEventStatus) IsValid() bool {
	switch s {
	case SafetyEventStatusOpen, SafetyEventStatusUnderReview, SafetyEventStatusClosed:
		return true
	default:
		return false
	}
}

type InspectionResult string

const (
	InspectionResultNone         = InspectionResult("")
	InspectionResultPass         = InspectionResult("Pass")
	InspectionResultFail         = InspectionResult("Fail")
	InspectionResultOutOfService = InspectionResult("OutOfService")
)

func (r InspectionResult) String() string { return string(r) }

func InspectionResultValues() []InspectionResult {
	return []InspectionResult{
		InspectionResultPass,
		InspectionResultFail,
		InspectionResultOutOfService,
	}
}

func (r InspectionResult) IsValid() bool {
	switch r {
	case InspectionResultNone, InspectionResultPass, InspectionResultFail,
		InspectionResultOutOfService:
		return true
	default:
		return false
	}
}

func (r InspectionResult) IsSet() bool { return r != InspectionResultNone }

// SafetyPointsRetentionMonths is how long an event's points stay on the
// scorecard, matching the FMCSA CSA look-back.
const SafetyPointsRetentionMonths = 24

// DefaultSafetyPoints suggests the points an event carries so the office
// starts from a consistent scale; it can be overridden per event.
func DefaultSafetyPoints(
	kind SafetyEventKind,
	severity SafetySeverity,
	preventable bool,
	result InspectionResult,
) int32 {
	base := int32(0)
	switch kind {
	case SafetyEventAccident:
		base = int32(severity.rank()) * 2 //nolint:gosec // rank is 1..4
		if preventable {
			base += 2
		}
	case SafetyEventIncident:
		base = int32(severity.rank()) //nolint:gosec // rank is 1..4
	case SafetyEventCitation:
		base = int32(severity.rank()) + 1 //nolint:gosec // rank is 1..4
	case SafetyEventInspection:
		switch result {
		case InspectionResultFail:
			base = 2
		case InspectionResultOutOfService:
			base = 5
		case InspectionResultPass, InspectionResultNone:
			base = 0
		}
	case SafetyEventNearMiss:
		base = 0
	}
	return base
}

var (
	_ validationframework.TenantedEntity = (*WorkerSafetyEvent)(nil)
)

type WorkerSafetyEvent struct {
	keel.Table `keel:"worker_safety_events"`

	ID               pulid.ID            `json:"id"               keel:"id,pk,autoid:wsev_"`
	BusinessUnitID   pulid.ID            `json:"businessUnitId"   keel:"business_unit_id,pk,tenant"`
	OrganizationID   pulid.ID            `json:"organizationId"   keel:"organization_id,pk,tenant"`
	WorkerID         pulid.ID            `json:"workerId"         keel:"worker_id"`
	Kind             SafetyEventKind     `json:"kind"             keel:"kind"`
	Severity         SafetySeverity      `json:"severity"         keel:"severity"`
	Status           SafetyEventStatus   `json:"status"           keel:"status"`
	OccurredAt       int64               `json:"occurredAt"       keel:"occurred_at"`
	Location         string              `json:"location"         keel:"location,nullzero"`
	Description      string              `json:"description"      keel:"description"`
	Preventable      bool                `json:"preventable"      keel:"preventable"`
	Points           int32               `json:"points"           keel:"points"`
	PointsExpireAt   *int64              `json:"pointsExpireAt"   keel:"points_expire_at"`
	ReferenceNumber  string              `json:"referenceNumber"  keel:"reference_number,nullzero"`
	ShipmentID       pulid.ID            `json:"shipmentId"       keel:"shipment_id,nullzero"`
	InspectionLevel  *int16              `json:"inspectionLevel"  keel:"inspection_level"`
	InspectionResult InspectionResult    `json:"inspectionResult" keel:"inspection_result,nullzero"`
	OutOfService     bool                `json:"outOfService"     keel:"out_of_service"`
	FineAmount       decimal.NullDecimal `json:"fineAmount"       keel:"fine_amount"`
	CostAmount       decimal.NullDecimal `json:"costAmount"       keel:"cost_amount"`
	DocumentID       pulid.ID            `json:"documentId"       keel:"document_id,nullzero"`
	RecordedByID     pulid.ID            `json:"recordedById"     keel:"recorded_by_id,nullzero"`
	ClosedByID       pulid.ID            `json:"closedById"       keel:"closed_by_id,nullzero"`
	ClosedAt         *int64              `json:"closedAt"         keel:"closed_at"`
	Resolution       string              `json:"resolution"       keel:"resolution,nullzero"`
	Version          int64               `json:"version"          keel:"version,version"`
	CreatedAt        int64               `json:"createdAt"        keel:"created_at,autocreate"`
	UpdatedAt        int64               `json:"updatedAt"        keel:"updated_at,autoupdate"`

	Worker     *worker.Worker     `json:"worker,omitempty"     keel:"belongs:worker_id"`
	Document   *document.Document `json:"document,omitempty"   keel:"belongs:document_id"`
	RecordedBy *tenant.User       `json:"recordedBy,omitempty" keel:"belongs:recorded_by_id"`
	ClosedBy   *tenant.User       `json:"closedBy,omitempty"   keel:"belongs:closed_by_id"`
}

func (e *WorkerSafetyEvent) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(e,
		validation.Field(&e.WorkerID, validation.Required.Error("Worker is required")),
		validation.Field(&e.Kind,
			validation.Required.Error("Kind is required"),
			domainvalidation.ValidEnum[SafetyEventKind](
				"kind must be one of: Accident, Incident, NearMiss, Citation, Inspection",
			),
		),
		validation.Field(&e.Severity,
			validation.Required.Error("Severity is required"),
			domainvalidation.ValidEnum[SafetySeverity](
				"severity must be one of: Minor, Moderate, Major, Critical",
			),
		),
		validation.Field(
			&e.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[SafetyEventStatus](
				"status must be Open, UnderReview or Closed",
			),
		),
		validation.Field(&e.OccurredAt, validation.Required.Error("When it happened is required")),
		validation.Field(&e.Description,
			validation.Required.Error("Describe what happened"),
			validation.Length(1, 4000).Error("Description cannot exceed 4000 characters"),
		),
		validation.Field(&e.Location,
			validation.Length(0, 255).Error("Location cannot exceed 255 characters"),
		),
		validation.Field(&e.ReferenceNumber,
			validation.Length(0, 100).Error("Reference cannot exceed 100 characters"),
		),
		validation.Field(&e.Points, validation.Min(int32(0)).Error("Points cannot be negative")),
	))

	if e.OccurredAt > timeutils.NowUnix()+timeutils.SecondsPerDay {
		multiErr.Add("occurredAt", errortypes.ErrInvalid, "An event cannot be in the future")
	}
	if !e.InspectionResult.IsValid() {
		multiErr.Add("inspectionResult", errortypes.ErrInvalid, "Unknown inspection result")
	}
	if e.Kind == SafetyEventInspection {
		if !e.InspectionResult.IsSet() {
			multiErr.Add(
				"inspectionResult",
				errortypes.ErrRequired,
				"Record how the inspection went",
			)
		}
		if e.InspectionLevel != nil && (*e.InspectionLevel < 1 || *e.InspectionLevel > 6) {
			multiErr.Add("inspectionLevel", errortypes.ErrInvalid, "Inspection level is 1 to 6")
		}
	} else if e.InspectionResult.IsSet() || e.InspectionLevel != nil {
		multiErr.Add("inspectionResult", errortypes.ErrInvalid, "Only inspections carry a result")
	}
	if e.FineAmount.Valid && e.FineAmount.Decimal.IsNegative() {
		multiErr.Add("fineAmount", errortypes.ErrInvalid, "Fine cannot be negative")
	}
	if e.CostAmount.Valid && e.CostAmount.Decimal.IsNegative() {
		multiErr.Add("costAmount", errortypes.ErrInvalid, "Cost cannot be negative")
	}
	if e.Status == SafetyEventStatusClosed && strings.TrimSpace(e.Resolution) == "" {
		multiErr.Add("resolution", errortypes.ErrRequired, "Say how the event was resolved")
	}
}

// IsClosed reports whether the event has been resolved.
func (e *WorkerSafetyEvent) IsClosed() bool { return e.Status == SafetyEventStatusClosed }

// ActivePoints returns the points still counting at now.
func (e *WorkerSafetyEvent) ActivePoints(now int64) int32 {
	if e.Points <= 0 {
		return 0
	}
	if e.PointsExpireAt != nil && *e.PointsExpireAt > 0 && *e.PointsExpireAt <= now {
		return 0
	}
	return e.Points
}

// DefaultPointsExpiry sets the expiry from the occurrence date when points
// are carried and no expiry was chosen.
func (e *WorkerSafetyEvent) DefaultPointsExpiry() {
	if e.Points <= 0 {
		e.PointsExpireAt = nil
		return
	}
	if e.PointsExpireAt == nil || *e.PointsExpireAt <= 0 {
		expiry := timeutils.AddMonthsUTC(e.OccurredAt, SafetyPointsRetentionMonths)
		e.PointsExpireAt = &expiry
	}
}

func (e *WorkerSafetyEvent) GetID() pulid.ID { return e.ID }

func (e *WorkerSafetyEvent) GetCreatedAt() int64 { return e.CreatedAt }

func (e *WorkerSafetyEvent) GetOrganizationID() pulid.ID { return e.OrganizationID }

func (e *WorkerSafetyEvent) GetBusinessUnitID() pulid.ID { return e.BusinessUnitID }

func (e *WorkerSafetyEvent) GetTableName() string { return "worker_safety_events" }

func (e *WorkerSafetyEvent) GetResourceType() string { return "worker_safety_event" }

func (e *WorkerSafetyEvent) GetResourceID() string { return e.ID.String() }

func (e *WorkerSafetyEvent) BeforeCreate(_ context.Context) error {
	if e.Status == "" {
		e.Status = SafetyEventStatusOpen
	}
	if e.Severity == "" {
		e.Severity = SafetySeverityMinor
	}

	return nil
}

func (WorkerSafetyEvent) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "fk_worker_safety_events_shipment",
			Message: "Shipment does not exist or belongs to a different organization",
		},
		{
			Name:    "fk_worker_safety_events_document",
			Message: "Document does not exist or belongs to a different organization",
		},
		{
			Name:    "fk_worker_safety_events_worker",
			Message: "Worker does not exist or belongs to a different organization",
		},
	}
}
