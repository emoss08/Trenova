package carrierintel

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*CarrierMonitoringEnrollment)(nil)

type CarrierMonitoringEnrollment struct {
	bun.BaseModel `bun:"table:carrier_monitoring_enrollments,alias:cienr" json:"-"`
	keel.Table    `keel:"carrier_monitoring_enrollments"`

	pagination.CursorValueSet `json:"-" bun:",embed"`

	ID              pulid.ID         `json:"id"              bun:"id,type:VARCHAR(100),pk,notnull" keel:"id,pk,autoid:cienr_"`
	BusinessUnitID  pulid.ID         `json:"businessUnitId"  bun:"business_unit_id,type:VARCHAR(100),pk,notnull" keel:"business_unit_id,pk,tenant"`
	OrganizationID  pulid.ID         `json:"organizationId"  bun:"organization_id,type:VARCHAR(100),pk,notnull" keel:"organization_id,pk,tenant"`
	SubjectType     SubjectType      `json:"subjectType"     bun:"subject_type,type:VARCHAR(20),notnull" keel:"subject_type"`
	SubjectID       string           `json:"subjectId"       bun:"subject_id,type:VARCHAR(100),notnull" keel:"subject_id"`
	CarrierID       pulid.ID         `json:"carrierId"       bun:"carrier_id,type:VARCHAR(100),nullzero" keel:"carrier_id,nullzero"`
	SubjectName     string           `json:"subjectName"     bun:"subject_name,type:VARCHAR(255),nullzero" keel:"subject_name,nullzero"`
	DOTNumber       string           `json:"dotNumber"       bun:"dot_number,type:VARCHAR(12),notnull" keel:"dot_number"`
	DocketNumber    string           `json:"docketNumber"    bun:"docket_number,type:VARCHAR(12),nullzero" keel:"docket_number,nullzero"`
	Provider        integration.Type `json:"provider"        bun:"provider,type:integration_type,notnull" keel:"provider"`
	ProviderRef     string           `json:"providerRef"     bun:"provider_ref,type:VARCHAR(64),nullzero" keel:"provider_ref,nullzero"`
	Mode            EnrollmentMode   `json:"mode"            bun:"mode,type:VARCHAR(20),notnull" keel:"mode"`
	DesiredState    DesiredState     `json:"desiredState"    bun:"desired_state,type:VARCHAR(20),notnull" keel:"desired_state"`
	VendorState     VendorState      `json:"vendorState"     bun:"vendor_state,type:VARCHAR(20),notnull" keel:"vendor_state"`
	Reason          EnrollmentReason `json:"reason"          bun:"reason,type:VARCHAR(30),notnull" keel:"reason"`
	OwnedByTrenova  bool             `json:"ownedByTrenova"  bun:"owned_by_trenova,type:BOOLEAN,notnull" keel:"owned_by_trenova"`
	EnrolledAt      *int64           `json:"enrolledAt"      bun:"enrolled_at,type:BIGINT,nullzero" keel:"enrolled_at"`
	UnenrolledAt    *int64           `json:"unenrolledAt"    bun:"unenrolled_at,type:BIGINT,nullzero" keel:"unenrolled_at"`
	LastSyncedAt    *int64           `json:"lastSyncedAt"    bun:"last_synced_at,type:BIGINT,nullzero" keel:"last_synced_at"`
	LastConfirmedAt *int64           `json:"lastConfirmedAt" bun:"last_confirmed_at,type:BIGINT,nullzero" keel:"last_confirmed_at"`
	LastUsedAt      *int64           `json:"lastUsedAt"      bun:"last_used_at,type:BIGINT,nullzero" keel:"last_used_at"`
	FailureCount    int              `json:"failureCount"    bun:"failure_count,type:INTEGER,notnull" keel:"failure_count"`
	LastError       string           `json:"lastError"       bun:"last_error,type:TEXT,nullzero" keel:"last_error,nullzero"`
	Version         int64            `json:"version"         bun:"version,type:BIGINT" keel:"version,version"`
	CreatedAt       int64            `json:"createdAt"       bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt       int64            `json:"updatedAt"       bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`
}

func (e *CarrierMonitoringEnrollment) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(e,
		validation.Field(&e.SubjectType,
			validation.Required.Error("Subject type is required"),
			domainvalidation.ValidEnum[SubjectType]("Subject type is invalid"),
		),
		validation.Field(&e.SubjectID, validation.Required.Error("Subject is required")),
		validation.Field(&e.DOTNumber,
			validation.Required.Error("A DOT number is required to monitor a subject"),
			validation.Length(1, 12).Error("DOT number must be at most 12 characters"),
		),
		validation.Field(&e.Mode,
			validation.Required.Error("Mode is required"),
			domainvalidation.ValidEnum[EnrollmentMode]("Mode is invalid"),
		),
		validation.Field(&e.DesiredState,
			validation.Required.Error("Desired state is required"),
			domainvalidation.ValidEnum[DesiredState]("Desired state is invalid"),
		),
		validation.Field(&e.VendorState,
			validation.Required.Error("Vendor state is required"),
			domainvalidation.ValidEnum[VendorState]("Vendor state is invalid"),
		),
		validation.Field(&e.Reason,
			validation.Required.Error("Reason is required"),
			domainvalidation.ValidEnum[EnrollmentReason]("Reason is invalid"),
		),
	))
}

func (e *CarrierMonitoringEnrollment) WantsEnrolled() bool {
	return e.DesiredState == DesiredStateEnrolled
}

func (e *CarrierMonitoringEnrollment) IsActive() bool {
	return e.WantsEnrolled() && e.VendorState == VendorStateActive
}

func (e *CarrierMonitoringEnrollment) NeedsSync() bool {
	switch e.DesiredState {
	case DesiredStateEnrolled:
		return e.VendorState != VendorStateActive
	case DesiredStateNotEnrolled:
		return e.VendorState == VendorStateActive || e.VendorState == VendorStatePendingRemove ||
			e.VendorState == VendorStatePendingAdd
	default:
		return false
	}
}

func (e *CarrierMonitoringEnrollment) MarkDesired(
	state DesiredState,
	reason EnrollmentReason,
	now int64,
) bool {
	if e.DesiredState == state && (state == DesiredStateNotEnrolled || e.Reason == reason) {
		return false
	}
	e.DesiredState = state
	if state == DesiredStateEnrolled {
		e.Reason = reason
		if e.VendorState != VendorStateActive {
			e.VendorState = VendorStatePendingAdd
		}
		return true
	}
	if e.VendorState == VendorStateActive || e.VendorState == VendorStatePendingAdd {
		e.VendorState = VendorStatePendingRemove
	}
	e.UnenrolledAt = &now
	return true
}

func (e *CarrierMonitoringEnrollment) MarkSynced(now int64) {
	e.LastSyncedAt = &now
	e.FailureCount = 0
	e.LastError = ""
	if e.DesiredState == DesiredStateEnrolled {
		e.VendorState = VendorStateActive
		e.OwnedByTrenova = true
		if e.EnrolledAt == nil {
			e.EnrolledAt = &now
		}
		return
	}
	e.VendorState = VendorStateRemoved
}

func (e *CarrierMonitoringEnrollment) MarkFailed(message string, now int64) {
	e.LastSyncedAt = &now
	e.FailureCount++
	e.LastError = message
	if e.FailureCount >= 5 {
		e.VendorState = VendorStateFailed
	}
}

func (e *CarrierMonitoringEnrollment) GetID() pulid.ID { return e.ID }

func (e *CarrierMonitoringEnrollment) GetCreatedAt() int64 { return e.CreatedAt }

func (e *CarrierMonitoringEnrollment) GetOrganizationID() pulid.ID { return e.OrganizationID }

func (e *CarrierMonitoringEnrollment) GetBusinessUnitID() pulid.ID { return e.BusinessUnitID }

func (e *CarrierMonitoringEnrollment) GetTableName() string { return "carrier_monitoring_enrollments" }

func (e *CarrierMonitoringEnrollment) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias: "cienr",
		SearchableFields: []domaintypes.SearchableField{
			{
				Name:   "subject_name",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightA,
			},
			{
				Name:   "dot_number",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightA,
			},
			{
				Name:   "provider_ref",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
		},
	}
}

func (e *CarrierMonitoringEnrollment) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if e.ID.IsNil() {
			e.ID = pulid.MustNew("cienr_")
		}
		if e.VendorState == "" {
			e.VendorState = VendorStateUnknown
		}
		e.CreatedAt = now
		e.UpdatedAt = now
	case *bun.UpdateQuery:
		e.UpdatedAt = now
	}

	return nil
}

type FeedCursor struct {
	DateMin string `json:"dateMin,omitempty"`
	Page    int    `json:"page,omitempty"`
	AfterID string `json:"afterId,omitempty"`
}

type CarrierIntelFeedState struct {
	bun.BaseModel `bun:"table:carrier_intel_feed_states,alias:cifeed" json:"-"`
	keel.Table    `keel:"carrier_intel_feed_states"`

	OrganizationID pulid.ID         `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID         `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull" keel:"business_unit_id,pk,tenant"`
	Provider       integration.Type `json:"provider"       bun:"provider,type:integration_type,pk,notnull" keel:"provider,pk"`
	FeedType       FeedType         `json:"feedType"       bun:"feed_type,type:VARCHAR(32),pk,notnull" keel:"feed_type,pk"`
	Cursor         FeedCursor       `json:"cursor"         bun:"cursor,type:JSONB,notnull" keel:"cursor,jsonb"`
	LastPolledAt   *int64           `json:"lastPolledAt"   bun:"last_polled_at,type:BIGINT,nullzero" keel:"last_polled_at"`
	LastSuccessAt  *int64           `json:"lastSuccessAt"  bun:"last_success_at,type:BIGINT,nullzero" keel:"last_success_at"`
	NextPollAfter  *int64           `json:"nextPollAfter"  bun:"next_poll_after,type:BIGINT,nullzero" keel:"next_poll_after"`
	PausedReason   FeedPauseReason  `json:"pausedReason"   bun:"paused_reason,type:VARCHAR(30),nullzero" keel:"paused_reason,nullzero"`
	PausedAt       *int64           `json:"pausedAt"       bun:"paused_at,type:BIGINT,nullzero" keel:"paused_at"`
	FailureCount   int              `json:"failureCount"   bun:"failure_count,type:INTEGER,notnull" keel:"failure_count"`
	LastError      string           `json:"lastError"      bun:"last_error,type:TEXT,nullzero" keel:"last_error,nullzero"`
}

func (f *CarrierIntelFeedState) IsPaused() bool {
	return f.PausedReason != FeedPauseReasonNone
}

func (f *CarrierIntelFeedState) Pause(reason FeedPauseReason, message string, now int64) {
	f.PausedReason = reason
	f.PausedAt = &now
	f.LastError = message
}

func (f *CarrierIntelFeedState) Resume() {
	f.PausedReason = FeedPauseReasonNone
	f.PausedAt = nil
	f.FailureCount = 0
	f.LastError = ""
}

func (f *CarrierIntelFeedState) RecordSuccess(now, nextPollAfter int64) {
	f.LastPolledAt = &now
	f.LastSuccessAt = &now
	f.NextPollAfter = &nextPollAfter
	f.FailureCount = 0
	f.LastError = ""
}

func (f *CarrierIntelFeedState) RecordFailure(message string, now, nextPollAfter int64) {
	f.LastPolledAt = &now
	f.NextPollAfter = &nextPollAfter
	f.FailureCount++
	f.LastError = message
}

func (f *CarrierIntelFeedState) DueAt(now int64) bool {
	if f.IsPaused() {
		return false
	}
	return f.NextPollAfter == nil || *f.NextPollAfter <= now
}
