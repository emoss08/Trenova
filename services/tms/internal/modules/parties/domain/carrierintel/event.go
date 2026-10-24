package carrierintel

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type CarrierIntelEvent struct {
	keel.Table `                                                      keel:"carrier_intel_events"`

	ID               pulid.ID         `json:"id" keel:"id,pk,autoid:cievt_"`
	BusinessUnitID   pulid.ID         `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID   pulid.ID         `json:"organizationId" keel:"organization_id,pk,tenant"`
	SubjectType      SubjectType      `json:"subjectType" keel:"subject_type"`
	SubjectID        string           `json:"subjectId" keel:"subject_id"`
	CarrierID        pulid.ID         `json:"carrierId" keel:"carrier_id,nullzero"`
	DOTNumber        string           `json:"dotNumber" keel:"dot_number"`
	SubjectName      string           `json:"subjectName" keel:"subject_name,nullzero"`
	Provider         integration.Type `json:"provider" keel:"provider"`
	Source           EventSource      `json:"source" keel:"source"`
	Category         Section          `json:"category" keel:"category"`
	FieldPath        string           `json:"fieldPath" keel:"field_path,nullzero"`
	RuleCode         RuleCode         `json:"ruleCode" keel:"rule_code,nullzero"`
	Severity         Severity         `json:"severity" keel:"severity"`
	Action           RuleAction       `json:"action" keel:"action,nullzero"`
	PriorValue       any              `json:"priorValue" keel:"prior_value,jsonb,nullzero"`
	CurrentValue     any              `json:"currentValue" keel:"current_value,jsonb,nullzero"`
	Summary          string           `json:"summary" keel:"summary"`
	VendorChangedAt  *int64           `json:"vendorChangedAt" keel:"vendor_changed_at"`
	DetectedAt       int64            `json:"detectedAt" keel:"detected_at"`
	Fingerprint      string           `json:"fingerprint" keel:"fingerprint"`
	Status           EventStatus      `json:"status" keel:"status"`
	AcknowledgedByID pulid.ID         `json:"acknowledgedById" keel:"acknowledged_by_id,nullzero"`
	AcknowledgedAt   *int64           `json:"acknowledgedAt" keel:"acknowledged_at"`
	ResolvedByID     pulid.ID         `json:"resolvedById" keel:"resolved_by_id,nullzero"`
	ResolvedAt       *int64           `json:"resolvedAt" keel:"resolved_at"`
	Resolution       EventResolution  `json:"resolution" keel:"resolution,nullzero"`
	ResolutionNote   string           `json:"resolutionNote" keel:"resolution_note,nullzero"`
	SnapshotID       pulid.ID         `json:"snapshotId" keel:"snapshot_id,nullzero"`
	NotifiedAt       *int64           `json:"notifiedAt" keel:"notified_at"`
	Version          int64            `json:"version" keel:"version,version"`
	CreatedAt        int64            `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt        int64            `json:"updatedAt" keel:"updated_at,autoupdate"`
}

func (e *CarrierIntelEvent) BeforeCreate(_ context.Context) error {
	if e.Status == "" {
		e.Status = EventStatusOpen
	}
	if e.DetectedAt == 0 {
		e.DetectedAt = e.CreatedAt
	}

	return nil
}

func (e *CarrierIntelEvent) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(e,
		validation.Field(&e.SubjectType,
			validation.Required.Error("Subject type is required"),
			domainvalidation.ValidEnum[SubjectType]("Subject type is invalid"),
		),
		validation.Field(&e.Source,
			validation.Required.Error("Source is required"),
			domainvalidation.ValidEnum[EventSource]("Source is invalid"),
		),
		validation.Field(&e.Severity,
			validation.Required.Error("Severity is required"),
			domainvalidation.ValidEnum[Severity]("Severity is invalid"),
		),
		validation.Field(&e.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[EventStatus]("Status is invalid"),
		),
		validation.Field(&e.Summary, validation.Required.Error("Summary is required")),
		validation.Field(&e.Resolution,
			domainvalidation.ValidEnum[EventResolution]("Resolution is invalid"),
		),
	))
}

func (e *CarrierIntelEvent) Acknowledge(userID pulid.ID, now int64) bool {
	if e.Status != EventStatusOpen {
		return false
	}
	e.Status = EventStatusAcknowledged
	e.AcknowledgedByID = userID
	e.AcknowledgedAt = &now
	return true
}

func (e *CarrierIntelEvent) Resolve(
	userID pulid.ID,
	resolution EventResolution,
	note string,
	now int64,
) error {
	if e.Status.IsClosed() {
		return errortypes.NewBusinessError("This event is already closed")
	}
	if !resolution.IsValid() {
		return errortypes.NewValidationError("resolution", errortypes.ErrInvalid,
			"Resolution is invalid")
	}
	status := EventStatusResolved
	if resolution == EventResolutionFalsePositive || resolution == EventResolutionNoActionRequired {
		status = EventStatusDismissed
	}
	if e.AcknowledgedAt == nil {
		e.AcknowledgedByID = userID
		e.AcknowledgedAt = &now
	}
	e.Status = status
	e.Resolution = resolution
	e.ResolutionNote = strings.TrimSpace(note)
	e.ResolvedByID = userID
	e.ResolvedAt = &now
	return nil
}

func (e *CarrierIntelEvent) GetID() pulid.ID { return e.ID }

func (e *CarrierIntelEvent) GetCreatedAt() int64 { return e.CreatedAt }

func (e *CarrierIntelEvent) GetOrganizationID() pulid.ID { return e.OrganizationID }

func (e *CarrierIntelEvent) GetBusinessUnitID() pulid.ID { return e.BusinessUnitID }

func (e *CarrierIntelEvent) GetTableName() string { return "carrier_intel_events" }

func (e *CarrierIntelEvent) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias: "cievt",
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
			{Name: "summary", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightB},
		},
	}
}

type EventFingerprintInput struct {
	Provider    integration.Type
	SubjectType SubjectType
	SubjectID   string
	Discriminat string
	Prior       any
	Current     any
	Day         int64
}

func EventFingerprint(in EventFingerprintInput) string {
	prior, _ := sonic.Marshal(in.Prior)
	current, _ := sonic.Marshal(in.Current)
	raw := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%d",
		in.Provider, in.SubjectType, in.SubjectID, in.Discriminat, prior, current, in.Day)
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
