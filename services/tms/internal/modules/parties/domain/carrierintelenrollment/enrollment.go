package carrierintelenrollment

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type CarrierMonitoringEnrollment struct {
	keel.Table `keel:"carrier_monitoring_enrollments"`

	ID              pulid.ID                    `json:"id"              keel:"id,pk,autoid:cienr_"`
	BusinessUnitID  pulid.ID                    `json:"businessUnitId"  keel:"business_unit_id,pk,tenant"`
	OrganizationID  pulid.ID                    `json:"organizationId"  keel:"organization_id,pk,tenant"`
	SubjectType     carrierintel.SubjectType    `json:"subjectType"     keel:"subject_type"`
	SubjectID       string                      `json:"subjectId"       keel:"subject_id"`
	CarrierID       pulid.ID                    `json:"carrierId"       keel:"carrier_id,nullzero"`
	SubjectName     string                      `json:"subjectName"     keel:"subject_name,nullzero"`
	DOTNumber       string                      `json:"dotNumber"       keel:"dot_number"`
	DocketNumber    string                      `json:"docketNumber"    keel:"docket_number,nullzero"`
	Provider        integration.Type            `json:"provider"        keel:"provider"`
	ProviderRef     string                      `json:"providerRef"     keel:"provider_ref,nullzero"`
	Mode            carrierintel.EnrollmentMode `json:"mode"            keel:"mode"`
	DesiredState    DesiredState                `json:"desiredState"    keel:"desired_state"`
	VendorState     VendorState                 `json:"vendorState"     keel:"vendor_state"`
	Reason          EnrollmentReason            `json:"reason"          keel:"reason"`
	OwnedByTrenova  bool                        `json:"ownedByTrenova"  keel:"owned_by_trenova"`
	EnrolledAt      *int64                      `json:"enrolledAt"      keel:"enrolled_at"`
	UnenrolledAt    *int64                      `json:"unenrolledAt"    keel:"unenrolled_at"`
	LastSyncedAt    *int64                      `json:"lastSyncedAt"    keel:"last_synced_at"`
	LastConfirmedAt *int64                      `json:"lastConfirmedAt" keel:"last_confirmed_at"`
	LastUsedAt      *int64                      `json:"lastUsedAt"      keel:"last_used_at"`
	FailureCount    int                         `json:"failureCount"    keel:"failure_count"`
	LastError       string                      `json:"lastError"       keel:"last_error,nullzero"`
	Version         int64                       `json:"version"         keel:"version,version"`
	CreatedAt       int64                       `json:"createdAt"       keel:"created_at,autocreate"`
	UpdatedAt       int64                       `json:"updatedAt"       keel:"updated_at,autoupdate"`
}

func (e *CarrierMonitoringEnrollment) BeforeCreate(_ context.Context) error {
	if e.VendorState == "" {
		e.VendorState = VendorStateUnknown
	}

	return nil
}

func (e *CarrierMonitoringEnrollment) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(e,
		validation.Field(&e.SubjectType,
			validation.Required.Error("Subject type is required"),
			domainvalidation.ValidEnum[carrierintel.SubjectType]("Subject type is invalid"),
		),
		validation.Field(&e.SubjectID, validation.Required.Error("Subject is required")),
		validation.Field(&e.DOTNumber,
			validation.Required.Error("A DOT number is required to monitor a subject"),
			validation.Length(1, 12).Error("DOT number must be at most 12 characters"),
		),
		validation.Field(&e.Mode,
			validation.Required.Error("Mode is required"),
			domainvalidation.ValidEnum[carrierintel.EnrollmentMode]("Mode is invalid"),
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

type FeedCursor struct {
	DateMin string `json:"dateMin,omitempty"`
	Page    int    `json:"page,omitempty"`
	AfterID string `json:"afterId,omitempty"`
}

type CarrierIntelFeedState struct {
	keel.Table `keel:"carrier_intel_feed_states"`

	OrganizationID pulid.ID         `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID         `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	Provider       integration.Type `json:"provider"       keel:"provider,pk"`
	FeedType       FeedType         `json:"feedType"       keel:"feed_type,pk"`
	Cursor         FeedCursor       `json:"cursor"         keel:"cursor,jsonb"`
	LastPolledAt   *int64           `json:"lastPolledAt"   keel:"last_polled_at"`
	LastSuccessAt  *int64           `json:"lastSuccessAt"  keel:"last_success_at"`
	NextPollAfter  *int64           `json:"nextPollAfter"  keel:"next_poll_after"`
	PausedReason   FeedPauseReason  `json:"pausedReason"   keel:"paused_reason,nullzero"`
	PausedAt       *int64           `json:"pausedAt"       keel:"paused_at"`
	FailureCount   int              `json:"failureCount"   keel:"failure_count"`
	LastError      string           `json:"lastError"      keel:"last_error,nullzero"`
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
