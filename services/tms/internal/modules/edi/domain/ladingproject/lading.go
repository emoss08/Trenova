package ladingproject

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/pkg/lading"
	"github.com/emoss08/trenova/pkg/wasmmodule"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
)

type LadingVersionStatus string

const (
	LadingVersionStatusDraft      = LadingVersionStatus("Draft")
	LadingVersionStatusCertified  = LadingVersionStatus("Certified")
	LadingVersionStatusActive     = LadingVersionStatus("Active")
	LadingVersionStatusShadow     = LadingVersionStatus("Shadow")
	LadingVersionStatusSuperseded = LadingVersionStatus("Superseded")
	LadingVersionStatusArchived   = LadingVersionStatus("Archived")
)

func (s LadingVersionStatus) IsValid() bool {
	switch s {
	case LadingVersionStatusDraft,
		LadingVersionStatusCertified,
		LadingVersionStatusActive,
		LadingVersionStatusShadow,
		LadingVersionStatusSuperseded,
		LadingVersionStatusArchived:
		return true
	default:
		return false
	}
}

type FlowInstanceStatus string

const (
	FlowInstanceStatusOpen      = FlowInstanceStatus("Open")
	FlowInstanceStatusHeld      = FlowInstanceStatus("Held")
	FlowInstanceStatusEscalated = FlowInstanceStatus("Escalated")
	FlowInstanceStatusClosed    = FlowInstanceStatus("Closed")
)

func (s FlowInstanceStatus) IsValid() bool {
	switch s {
	case FlowInstanceStatusOpen,
		FlowInstanceStatusHeld,
		FlowInstanceStatusEscalated,
		FlowInstanceStatusClosed:
		return true
	default:
		return false
	}
}

type LadingReplayStatus string

const (
	LadingReplayStatusRunning   = LadingReplayStatus("Running")
	LadingReplayStatusCompleted = LadingReplayStatus("Completed")
	LadingReplayStatusFailed    = LadingReplayStatus("Failed")
)

func (s LadingReplayStatus) IsValid() bool {
	switch s {
	case LadingReplayStatusRunning, LadingReplayStatusCompleted, LadingReplayStatusFailed:
		return true
	default:
		return false
	}
}

type LadingSampleRef struct {
	Path   string `json:"path"`
	Digest string `json:"digest"`
	Size   int    `json:"size"`
}

type LadingReadiness struct {
	Errors               int   `json:"errors"`
	Failing              int   `json:"failing"`
	Tests                int   `json:"tests"`
	UncoveredFlowClauses int   `json:"uncoveredFlowClauses"`
	Ready                bool  `json:"ready"`
	CheckedAt            int64 `json:"checkedAt"`
}

type LadingProject struct {
	keel.Table `keel:"edi_lading_projects"`

	ID                pulid.ID       `json:"id"                keel:"id,pk,autoid:ldp_"`
	BusinessUnitID    pulid.ID       `json:"businessUnitId"    keel:"business_unit_id,pk,tenant"`
	OrganizationID    pulid.ID       `json:"organizationId"    keel:"organization_id,pk,tenant"`
	EDIPartnerID      pulid.ID       `json:"ediPartnerId"      keel:"edi_partner_id"`
	ActiveVersionID   pulid.ID       `json:"activeVersionId"   keel:"active_version_id,nullzero"`
	ShadowVersionID   pulid.ID       `json:"shadowVersionId"   keel:"shadow_version_id,nullzero"`
	Settings          map[string]any `json:"settings"          keel:"settings,jsonb"`
	NextVersionNumber int64          `json:"nextVersionNumber" keel:"next_version_number"`
	Version           int64          `json:"version"           keel:"version,version"`
	CreatedAt         int64          `json:"createdAt"         keel:"created_at,autocreate"`
	UpdatedAt         int64          `json:"updatedAt"         keel:"updated_at,autoupdate"`

	Partner       *edi.EDIPartner `json:"partner,omitempty"       keel:"belongs:edi_partner_id"`
	ActiveVersion *LadingVersion  `json:"activeVersion,omitempty" keel:"belongs:active_version_id"`
}

func (p *LadingProject) GetID() pulid.ID {
	return p.ID
}

func (p *LadingProject) GetOrganizationID() pulid.ID {
	return p.OrganizationID
}

func (p *LadingProject) GetBusinessUnitID() pulid.ID {
	return p.BusinessUnitID
}

type LadingVersion struct {
	keel.Table `keel:"edi_lading_versions"`

	ID              pulid.ID            `json:"id"              keel:"id,pk,autoid:ldv_"`
	BusinessUnitID  pulid.ID            `json:"businessUnitId"  keel:"business_unit_id,pk,tenant"`
	OrganizationID  pulid.ID            `json:"organizationId"  keel:"organization_id,pk,tenant"`
	ProjectID       pulid.ID            `json:"projectId"       keel:"project_id"`
	Number          int64               `json:"number"          keel:"number"`
	Status          LadingVersionStatus `json:"status"          keel:"status"`
	ParentVersionID pulid.ID            `json:"parentVersionId" keel:"parent_version_id,nullzero"`
	Files           []lading.File       `json:"files"           keel:"files,jsonb"`
	Samples         []LadingSampleRef   `json:"samples"         keel:"samples,jsonb"`
	Manifest        lading.Check        `json:"manifest"        keel:"manifest,jsonb"`
	Readiness       *LadingReadiness    `json:"readiness"       keel:"readiness,jsonb"`
	Notes           string              `json:"notes"           keel:"notes,nullzero"`
	ErrorCount      int                 `json:"errorCount"      keel:"error_count"`
	WarningCount    int                 `json:"warningCount"    keel:"warning_count"`
	ShadowUntil     *int64              `json:"shadowUntil"     keel:"shadow_until"`
	CreatedByID     pulid.ID            `json:"createdById"     keel:"created_by_id,nullzero"`
	CertifiedByID   pulid.ID            `json:"certifiedById"   keel:"certified_by_id,nullzero"`
	CertifiedAt     *int64              `json:"certifiedAt"     keel:"certified_at"`
	ActivatedByID   pulid.ID            `json:"activatedById"   keel:"activated_by_id,nullzero"`
	ActivatedAt     *int64              `json:"activatedAt"     keel:"activated_at"`
	SupersededAt    *int64              `json:"supersededAt"    keel:"superseded_at"`
	ArchivedAt      *int64              `json:"archivedAt"      keel:"archived_at"`
	Version         int64               `json:"version"         keel:"version,version"`
	CreatedAt       int64               `json:"createdAt"       keel:"created_at,autocreate"`
	UpdatedAt       int64               `json:"updatedAt"       keel:"updated_at,autoupdate"`
}

func (v *LadingVersion) GetID() pulid.ID {
	return v.ID
}

func (v *LadingVersion) GetOrganizationID() pulid.ID {
	return v.OrganizationID
}

func (v *LadingVersion) GetBusinessUnitID() pulid.ID {
	return v.BusinessUnitID
}

func (v *LadingVersion) Editable() bool {
	return v.Status == LadingVersionStatusDraft
}

func (v *LadingVersion) Runnable() bool {
	switch v.Status {
	case LadingVersionStatusActive, LadingVersionStatusShadow, LadingVersionStatusCertified:
		return true
	case LadingVersionStatusDraft, LadingVersionStatusSuperseded, LadingVersionStatusArchived:
	}
	return false
}

type LadingSample struct {
	keel.Table `keel:"edi_lading_samples"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:lds_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	ProjectID      pulid.ID `json:"projectId"      keel:"project_id"`
	Digest         string   `json:"digest"         keel:"digest"`
	Size           int      `json:"size"           keel:"size"`
	Content        string   `json:"content"        keel:"content"`
	CreatedAt      int64    `json:"createdAt"      keel:"created_at,autocreate"`
}

type FlowInstance struct {
	keel.Table `keel:"edi_flow_instances"`

	ID             pulid.ID            `json:"id"             keel:"id,pk,autoid:edfi_"`
	BusinessUnitID pulid.ID            `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID            `json:"organizationId" keel:"organization_id,pk,tenant"`
	EDIPartnerID   pulid.ID            `json:"ediPartnerId"   keel:"edi_partner_id"`
	ProjectID      pulid.ID            `json:"projectId"      keel:"project_id,nullzero"`
	VersionID      pulid.ID            `json:"versionId"      keel:"version_id,nullzero"`
	Flow           string              `json:"flow"           keel:"flow"`
	Key            string              `json:"key"            keel:"key"`
	State          jsonutils.RawJSON   `json:"state"          keel:"state,jsonb"`
	Status         FlowInstanceStatus  `json:"status"         keel:"status"`
	Reason         *wasmmodule.Message `json:"reason"         keel:"reason,jsonb"`
	WakeAt         *int64              `json:"wakeAt"         keel:"wake_at"`
	ShipmentID     pulid.ID            `json:"shipmentId"     keel:"shipment_id,nullzero"`
	TransferID     pulid.ID            `json:"transferId"     keel:"transfer_id,nullzero"`
	LastEventAt    int64               `json:"lastEventAt"    keel:"last_event_at"`
	HeldAt         *int64              `json:"heldAt"         keel:"held_at"`
	ResolvedByID   pulid.ID            `json:"resolvedById"   keel:"resolved_by_id,nullzero"`
	ResolvedAt     *int64              `json:"resolvedAt"     keel:"resolved_at"`
	Version        int64               `json:"version"        keel:"version,version"`
	CreatedAt      int64               `json:"createdAt"      keel:"created_at,autocreate"`
	UpdatedAt      int64               `json:"updatedAt"      keel:"updated_at,autoupdate"`

	Partner *edi.EDIPartner `json:"partner,omitempty" keel:"belongs:edi_partner_id"`
}

func (i *FlowInstance) GetID() pulid.ID {
	return i.ID
}

func (i *FlowInstance) GetOrganizationID() pulid.ID {
	return i.OrganizationID
}

func (i *FlowInstance) GetBusinessUnitID() pulid.ID {
	return i.BusinessUnitID
}

func (i *FlowInstance) GetCreatedAt() int64 {
	return i.CreatedAt
}

type FlowEvent struct {
	keel.Table `keel:"edi_flow_events"`

	ID             pulid.ID            `json:"id"             keel:"id,pk,autoid:edfe_"`
	BusinessUnitID pulid.ID            `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID            `json:"organizationId" keel:"organization_id,pk,tenant"`
	InstanceID     pulid.ID            `json:"instanceId"     keel:"instance_id"`
	Kind           string              `json:"kind"           keel:"kind"`
	Detail         jsonutils.RawJSON   `json:"detail"         keel:"detail,jsonb"`
	Actions        []lading.FlowAction `json:"actions"        keel:"actions,jsonb"`
	Error          *wasmmodule.Message `json:"error"          keel:"error,jsonb"`
	MessageID      pulid.ID            `json:"messageId"      keel:"message_id,nullzero"`
	ActorID        pulid.ID            `json:"actorId"        keel:"actor_id,nullzero"`
	CreatedAt      int64               `json:"createdAt"      keel:"created_at,autocreate"`
}

type LadingReplay struct {
	keel.Table `keel:"edi_lading_replays"`

	ID               pulid.ID              `json:"id"               keel:"id,pk,autoid:ldr_"`
	BusinessUnitID   pulid.ID              `json:"businessUnitId"   keel:"business_unit_id,pk,tenant"`
	OrganizationID   pulid.ID              `json:"organizationId"   keel:"organization_id,pk,tenant"`
	ProjectID        pulid.ID              `json:"projectId"        keel:"project_id"`
	VersionID        pulid.ID              `json:"versionId"        keel:"version_id"`
	AgainstVersionID pulid.ID              `json:"againstVersionId" keel:"against_version_id"`
	Map              string                `json:"map"              keel:"map"`
	PeriodStart      int64                 `json:"periodStart"      keel:"period_start"`
	PeriodEnd        int64                 `json:"periodEnd"        keel:"period_end"`
	Status           LadingReplayStatus    `json:"status"           keel:"status"`
	Summary          *lading.ReplaySummary `json:"summary"          keel:"summary,jsonb"`
	Items            []LadingReplayItem    `json:"items"            keel:"items,jsonb"`
	Error            string                `json:"error"            keel:"error,nullzero"`
	CreatedByID      pulid.ID              `json:"createdById"      keel:"created_by_id,nullzero"`
	FinishedAt       *int64                `json:"finishedAt"       keel:"finished_at"`
	Version          int64                 `json:"version"          keel:"version,version"`
	CreatedAt        int64                 `json:"createdAt"        keel:"created_at,autocreate"`
	UpdatedAt        int64                 `json:"updatedAt"        keel:"updated_at,autoupdate"`
}

func (r *LadingReplay) GetID() pulid.ID {
	return r.ID
}

func (r *LadingReplay) GetOrganizationID() pulid.ID {
	return r.OrganizationID
}

func (r *LadingReplay) GetBusinessUnitID() pulid.ID {
	return r.BusinessUnitID
}

type LadingReplayItem struct {
	MessageID pulid.ID `json:"messageId"`
	lading.ReplayItem
}

type LadingShadowResult struct {
	keel.Table `keel:"edi_lading_shadow_results"`

	ID             pulid.ID                   `json:"id"             keel:"id,pk,autoid:ldsh_"`
	BusinessUnitID pulid.ID                   `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID                   `json:"organizationId" keel:"organization_id,pk,tenant"`
	VersionID      pulid.ID                   `json:"versionId"      keel:"version_id"`
	MessageID      pulid.ID                   `json:"messageId"      keel:"message_id"`
	TransactionSet edidocument.TransactionSet `json:"transactionSet" keel:"transaction_set"`
	Same           bool                       `json:"same"           keel:"same"`
	Differences    []lading.Difference        `json:"differences"    keel:"differences,jsonb"`
	NewOutcomes    []lading.ReplayOutcome     `json:"newOutcomes"    keel:"new_outcomes,jsonb"`
	GoneOutcomes   []lading.ReplayOutcome     `json:"goneOutcomes"   keel:"gone_outcomes,jsonb"`
	Error          string                     `json:"error"          keel:"error,nullzero"`
	CreatedAt      int64                      `json:"createdAt"      keel:"created_at,autocreate"`
}
