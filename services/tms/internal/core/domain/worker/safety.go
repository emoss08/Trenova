package worker

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
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
	_ bun.BeforeAppendModelHook          = (*WorkerSafetyEvent)(nil)
	_ validationframework.TenantedEntity = (*WorkerSafetyEvent)(nil)
)

type WorkerSafetyEvent struct {
	bun.BaseModel `bun:"table:worker_safety_events,alias:wsev" json:"-"`

	ID               pulid.ID            `json:"id"               bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID   pulid.ID            `json:"businessUnitId"   bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID   pulid.ID            `json:"organizationId"   bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	WorkerID         pulid.ID            `json:"workerId"         bun:"worker_id,type:VARCHAR(100),notnull"`
	Kind             SafetyEventKind     `json:"kind"             bun:"kind,type:safety_event_kind_enum,notnull"`
	Severity         SafetySeverity      `json:"severity"         bun:"severity,type:safety_severity_enum,notnull,default:'Minor'"`
	Status           SafetyEventStatus   `json:"status"           bun:"status,type:safety_event_status_enum,notnull,default:'Open'"`
	OccurredAt       int64               `json:"occurredAt"       bun:"occurred_at,type:BIGINT,notnull"`
	Location         string              `json:"location"         bun:"location,type:VARCHAR(255),nullzero"`
	Description      string              `json:"description"      bun:"description,type:TEXT,notnull"`
	Preventable      bool                `json:"preventable"      bun:"preventable,type:BOOLEAN,notnull"`
	Points           int32               `json:"points"           bun:"points,type:INTEGER,notnull"`
	PointsExpireAt   *int64              `json:"pointsExpireAt"   bun:"points_expire_at,type:BIGINT,nullzero"`
	ReferenceNumber  string              `json:"referenceNumber"  bun:"reference_number,type:VARCHAR(100),nullzero"`
	ShipmentID       pulid.ID            `json:"shipmentId"       bun:"shipment_id,type:VARCHAR(100),nullzero"`
	InspectionLevel  *int16              `json:"inspectionLevel"  bun:"inspection_level,type:SMALLINT,nullzero"`
	InspectionResult InspectionResult    `json:"inspectionResult" bun:"inspection_result,type:inspection_result_enum,nullzero"`
	OutOfService     bool                `json:"outOfService"     bun:"out_of_service,type:BOOLEAN,notnull"`
	FineAmount       decimal.NullDecimal `json:"fineAmount"       bun:"fine_amount,type:NUMERIC(12,2),nullzero"`
	CostAmount       decimal.NullDecimal `json:"costAmount"       bun:"cost_amount,type:NUMERIC(12,2),nullzero"`
	DocumentID       pulid.ID            `json:"documentId"       bun:"document_id,type:VARCHAR(100),nullzero"`
	RecordedByID     pulid.ID            `json:"recordedById"     bun:"recorded_by_id,type:VARCHAR(100),nullzero"`
	ClosedByID       pulid.ID            `json:"closedById"       bun:"closed_by_id,type:VARCHAR(100),nullzero"`
	ClosedAt         *int64              `json:"closedAt"         bun:"closed_at,type:BIGINT,nullzero"`
	Resolution       string              `json:"resolution"       bun:"resolution,type:TEXT,nullzero"`
	Version          int64               `json:"version"          bun:"version,type:BIGINT"`
	CreatedAt        int64               `json:"createdAt"        bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt        int64               `json:"updatedAt"        bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Worker     *Worker            `json:"worker,omitempty"     bun:"rel:belongs-to,join:worker_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	Document   *document.Document `json:"document,omitempty"   bun:"rel:belongs-to,join:document_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	RecordedBy *tenant.User       `json:"recordedBy,omitempty" bun:"rel:belongs-to,join:recorded_by_id=id"`
	ClosedBy   *tenant.User       `json:"closedBy,omitempty"   bun:"rel:belongs-to,join:closed_by_id=id"`
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
		validation.Field(&e.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[SafetyEventStatus]("status must be Open, UnderReview or Closed"),
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

	if e.OccurredAt > timeutils.NowUnix()+secondsPerDay {
		multiErr.Add("occurredAt", errortypes.ErrInvalid, "An event cannot be in the future")
	}
	if !e.InspectionResult.IsValid() {
		multiErr.Add("inspectionResult", errortypes.ErrInvalid, "Unknown inspection result")
	}
	if e.Kind == SafetyEventInspection {
		if !e.InspectionResult.IsSet() {
			multiErr.Add("inspectionResult", errortypes.ErrRequired, "Record how the inspection went")
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

func (e *WorkerSafetyEvent) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if e.ID.IsNil() {
			e.ID = pulid.MustNew("wsev_")
		}
		if e.Status == "" {
			e.Status = SafetyEventStatusOpen
		}
		if e.Severity == "" {
			e.Severity = SafetySeverityMinor
		}
		e.CreatedAt = now
		e.UpdatedAt = now
	case *bun.UpdateQuery:
		e.UpdatedAt = now
	}

	return nil
}

// SafetyRating buckets a scorecard for chips and gauges.
type SafetyRating string

const (
	SafetyRatingExcellent = SafetyRating("Excellent")
	SafetyRatingGood      = SafetyRating("Good")
	SafetyRatingWatch     = SafetyRating("Watch")
	SafetyRatingAtRisk    = SafetyRating("AtRisk")
)

func (r SafetyRating) String() string { return string(r) }

func (r SafetyRating) IsValid() bool {
	switch r {
	case SafetyRatingExcellent, SafetyRatingGood, SafetyRatingWatch, SafetyRatingAtRisk:
		return true
	default:
		return false
	}
}

const (
	scorecardWindowMonths       = 12
	scorePenaltyPerPoint        = 5
	scorePenaltyPreventable     = 10
	scorePenaltyOutOfService    = 15
	scorePenaltyOpenDiscipline  = 5
	scoreExcellentFloor         = 90
	scoreGoodFloor              = 75
	scoreWatchFloor             = 50
	safetyPointsWatchThreshold  = 6
	safetyPointsAtRiskThreshold = 10
)

// SafetyScorecard is the twelve-month picture of a worker's record plus the
// points still active from the two-year look-back.
type SafetyScorecard struct {
	WorkerID              pulid.ID
	AsOf                  int64
	Score                 int32
	Rating                SafetyRating
	ActivePoints          int32
	PointsWatchThreshold  int32
	PointsAtRiskThreshold int32
	Accidents             int32
	PreventableAccidents  int32
	Incidents             int32
	NearMisses            int32
	Citations             int32
	Inspections           int32
	InspectionsPassed     int32
	InspectionsFailed     int32
	OutOfServiceOrders    int32
	CleanInspectionRate   *float64
	OpenEvents            int32
	ActiveDiscipline      int32
	HighestDiscipline     DisciplinaryLevel
	DaysSinceLastEvent    *int64
	LastEventAt           *int64
	Recognitions          int32
}

// BuildSafetyScorecard is the pure roll-up. Counts cover the trailing twelve
// months; points count until they expire; the score starts at 100 and loses
// ground for active points, preventable accidents, out-of-service orders and
// open discipline.
func BuildSafetyScorecard(
	workerID pulid.ID,
	events []*WorkerSafetyEvent,
	actions []*WorkerDisciplinaryAction,
	recognitions []*WorkerRecognition,
	now int64,
) *SafetyScorecard {
	card := &SafetyScorecard{
		WorkerID:              workerID,
		AsOf:                  now,
		PointsWatchThreshold:  safetyPointsWatchThreshold,
		PointsAtRiskThreshold: safetyPointsAtRiskThreshold,
	}
	windowStart := timeutils.AddMonthsUTC(now, -scorecardWindowMonths)

	var lastEvent int64
	for _, event := range events {
		if event == nil {
			continue
		}
		card.ActivePoints += event.ActivePoints(now)
		if !event.IsClosed() {
			card.OpenEvents++
		}
		if event.Kind.CountsAgainstRecord() && event.OccurredAt > lastEvent {
			lastEvent = event.OccurredAt
		}
		if event.OccurredAt < windowStart {
			continue
		}
		switch event.Kind {
		case SafetyEventAccident:
			card.Accidents++
			if event.Preventable {
				card.PreventableAccidents++
			}
		case SafetyEventIncident:
			card.Incidents++
		case SafetyEventNearMiss:
			card.NearMisses++
		case SafetyEventCitation:
			card.Citations++
		case SafetyEventInspection:
			card.Inspections++
			switch event.InspectionResult {
			case InspectionResultPass:
				card.InspectionsPassed++
			case InspectionResultFail:
				card.InspectionsFailed++
			case InspectionResultOutOfService:
				card.InspectionsFailed++
				card.OutOfServiceOrders++
			case InspectionResultNone:
			}
			if event.OutOfService && event.InspectionResult != InspectionResultOutOfService {
				card.OutOfServiceOrders++
			}
		}
	}
	if card.Inspections > 0 {
		rate := float64(card.InspectionsPassed) / float64(card.Inspections)
		card.CleanInspectionRate = &rate
	}
	if lastEvent > 0 {
		days := DaysUntil(now, lastEvent)
		card.LastEventAt = &lastEvent
		card.DaysSinceLastEvent = &days
	}

	for _, action := range actions {
		if action == nil || !action.IsActive(now) {
			continue
		}
		card.ActiveDiscipline++
		if action.Level.Rank() > card.HighestDiscipline.Rank() {
			card.HighestDiscipline = action.Level
		}
	}
	for _, recognition := range recognitions {
		if recognition != nil && recognition.OccurredAt >= windowStart {
			card.Recognitions++
		}
	}

	score := int32(100) -
		card.ActivePoints*scorePenaltyPerPoint -
		card.PreventableAccidents*scorePenaltyPreventable -
		card.OutOfServiceOrders*scorePenaltyOutOfService -
		card.ActiveDiscipline*scorePenaltyOpenDiscipline
	if score < 0 {
		score = 0
	}
	card.Score = score
	card.Rating = ratingFor(score, card.ActivePoints)
	return card
}

func ratingFor(score, points int32) SafetyRating {
	switch {
	case points >= safetyPointsAtRiskThreshold || score < scoreWatchFloor:
		return SafetyRatingAtRisk
	case points >= safetyPointsWatchThreshold || score < scoreGoodFloor:
		return SafetyRatingWatch
	case score < scoreExcellentFloor:
		return SafetyRatingGood
	default:
		return SafetyRatingExcellent
	}
}
