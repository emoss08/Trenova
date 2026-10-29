package aituneup

import (
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/shared/pulid"
)

const IDPrefix = "aitu_"

type Kind string

const (
	KindRaiseToolTier    = Kind("RaiseToolTier")
	KindReorderProviders = Kind("ReorderProviders")
	KindLeaveShadow      = Kind("LeaveShadow")
	KindAssignTask       = Kind("AssignTask")
	KindTurnOffIdleAgent = Kind("TurnOffIdleAgent")
)

func (k Kind) IsValid() bool {
	switch k {
	case KindRaiseToolTier,
		KindReorderProviders,
		KindLeaveShadow,
		KindAssignTask,
		KindTurnOffIdleAgent:
		return true
	default:
		return false
	}
}

type Status string

const (
	StatusOpen      = Status("Open")
	StatusApplied   = Status("Applied")
	StatusDismissed = Status("Dismissed")
)

const (
	WindowDays                   = 30
	DefaultDismissDays           = 30
	MaxDismissDays               = 180
	ShadowMinRecorded            = 20
	ShadowMinMatchRate           = 0.85
	IdleDays                     = 14
	ReorderMinFailures           = 10
	ReorderMinFailureRate        = 0.2
	ReorderMinRescues            = 5
	ReorderMaxRescuerFailureRate = 0.05
	AppliedRetentionDays         = 30
	FreshForSeconds              = 26 * 60 * 60
)

type Evidence struct {
	FromTier         agent.AutonomyTier `json:"fromTier,omitempty"`
	ToTier           agent.AutonomyTier `json:"toTier,omitempty"`
	Streak           int                `json:"streak,omitempty"`
	Approvals        int                `json:"approvals,omitempty"`
	ApprovalsPerWeek float64            `json:"approvalsPerWeek,omitempty"`
	Rejections       int                `json:"rejections,omitempty"`
	Calls            int                `json:"calls,omitempty"`
	Failed           int                `json:"failed,omitempty"`
	Rescued          int                `json:"rescued,omitempty"`
	Recorded         int                `json:"recorded,omitempty"`
	MatchRate        float64            `json:"matchRate,omitempty"`
	WouldFail        int                `json:"wouldFail,omitempty"`
	Tasks            []aitask.Task      `json:"tasks,omitempty"`
	Model            string             `json:"model,omitempty"`
	LastRunAt        *int64             `json:"lastRunAt,omitempty"`
	IdleSince        int64              `json:"idleSince,omitempty"`
	Tools            int                `json:"tools,omitempty"`
}

type TuneUp struct {
	keel.Table `keel:"ai_tune_ups"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:aitu_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	Kind              Kind        `json:"kind"              keel:"kind"`
	Fingerprint       string      `json:"fingerprint"       keel:"fingerprint"`
	AgentDefinitionID pulid.ID    `json:"agentDefinitionId" keel:"agent_definition_id,nullzero"`
	ProviderID        pulid.ID    `json:"providerId"        keel:"provider_id,nullzero"`
	OtherProviderID   pulid.ID    `json:"otherProviderId"   keel:"other_provider_id,nullzero"`
	ToolName          string      `json:"toolName"          keel:"tool_name,nullzero"`
	Task              aitask.Task `json:"task"              keel:"task,nullzero"`
	Evidence          Evidence    `json:"evidence"          keel:"evidence,jsonb"`

	Status         Status   `json:"status"         keel:"status"`
	DismissedUntil *int64   `json:"dismissedUntil" keel:"dismissed_until"`
	DecidedByID    pulid.ID `json:"decidedById"    keel:"decided_by_id,nullzero"`
	DecidedAt      *int64   `json:"decidedAt"      keel:"decided_at"`
	ComputedAt     int64    `json:"computedAt"     keel:"computed_at"`

	Version   int64 `json:"version"   keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`
}

func (t *TuneUp) Visible(now int64) bool {
	switch t.Status {
	case StatusOpen:
		return true
	case StatusDismissed:
		return t.DismissedUntil != nil && *t.DismissedUntil <= now
	default:
		return false
	}
}

func Fingerprint(kind Kind, parts ...string) string {
	return string(kind) + ":" + strings.Join(parts, ":")
}

func ClampDismissDays(days int) int {
	switch {
	case days <= 0:
		return DefaultDismissDays
	case days > MaxDismissDays:
		return MaxDismissDays
	default:
		return days
	}
}
