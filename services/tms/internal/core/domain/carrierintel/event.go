package carrierintel

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*CarrierIntelEvent)(nil)

type CarrierIntelEvent struct {
	bun.BaseModel `bun:"table:carrier_intel_events,alias:cievt" json:"-"`

	pagination.CursorValueSet `json:"-" bun:",embed"`

	ID               pulid.ID         `json:"id"               bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID   pulid.ID         `json:"businessUnitId"   bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID   pulid.ID         `json:"organizationId"   bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	SubjectType      SubjectType      `json:"subjectType"      bun:"subject_type,type:VARCHAR(20),notnull"`
	SubjectID        string           `json:"subjectId"        bun:"subject_id,type:VARCHAR(100),notnull"`
	CarrierID        pulid.ID         `json:"carrierId"        bun:"carrier_id,type:VARCHAR(100),nullzero"`
	DOTNumber        string           `json:"dotNumber"        bun:"dot_number,type:VARCHAR(12),notnull"`
	SubjectName      string           `json:"subjectName"      bun:"subject_name,type:VARCHAR(255),nullzero"`
	Provider         integration.Type `json:"provider"         bun:"provider,type:integration_type,notnull"`
	Source           EventSource      `json:"source"           bun:"source,type:VARCHAR(30),notnull"`
	Category         Section          `json:"category"         bun:"category,type:VARCHAR(30),notnull"`
	FieldPath        string           `json:"fieldPath"        bun:"field_path,type:VARCHAR(200),nullzero"`
	RuleCode         RuleCode         `json:"ruleCode"         bun:"rule_code,type:VARCHAR(100),nullzero"`
	Severity         Severity         `json:"severity"         bun:"severity,type:VARCHAR(20),notnull"`
	Action           RuleAction       `json:"action"           bun:"action,type:VARCHAR(10),nullzero"`
	PriorValue       any              `json:"priorValue"       bun:"prior_value,type:JSONB,nullzero"`
	CurrentValue     any              `json:"currentValue"     bun:"current_value,type:JSONB,nullzero"`
	Summary          string           `json:"summary"          bun:"summary,type:TEXT,notnull"`
	VendorChangedAt  *int64           `json:"vendorChangedAt"  bun:"vendor_changed_at,type:BIGINT,nullzero"`
	DetectedAt       int64            `json:"detectedAt"       bun:"detected_at,type:BIGINT,notnull"`
	Fingerprint      string           `json:"fingerprint"      bun:"fingerprint,type:VARCHAR(64),notnull"`
	Status           EventStatus      `json:"status"           bun:"status,type:VARCHAR(20),notnull"`
	AcknowledgedByID pulid.ID         `json:"acknowledgedById" bun:"acknowledged_by_id,type:VARCHAR(100),nullzero"`
	AcknowledgedAt   *int64           `json:"acknowledgedAt"   bun:"acknowledged_at,type:BIGINT,nullzero"`
	ResolvedByID     pulid.ID         `json:"resolvedById"     bun:"resolved_by_id,type:VARCHAR(100),nullzero"`
	ResolvedAt       *int64           `json:"resolvedAt"       bun:"resolved_at,type:BIGINT,nullzero"`
	Resolution       EventResolution  `json:"resolution"       bun:"resolution,type:VARCHAR(30),nullzero"`
	ResolutionNote   string           `json:"resolutionNote"   bun:"resolution_note,type:TEXT,nullzero"`
	SnapshotID       pulid.ID         `json:"snapshotId"       bun:"snapshot_id,type:VARCHAR(100),nullzero"`
	NotifiedAt       *int64           `json:"notifiedAt"       bun:"notified_at,type:BIGINT,nullzero"`
	Version          int64            `json:"version"          bun:"version,type:BIGINT"`
	CreatedAt        int64            `json:"createdAt"        bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt        int64            `json:"updatedAt"        bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
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

func (e *CarrierIntelEvent) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if e.ID.IsNil() {
			e.ID = pulid.MustNew("cievt_")
		}
		if e.Status == "" {
			e.Status = EventStatusOpen
		}
		if e.DetectedAt == 0 {
			e.DetectedAt = now
		}
		e.CreatedAt = now
		e.UpdatedAt = now
	case *bun.UpdateQuery:
		e.UpdatedAt = now
	}

	return nil
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
