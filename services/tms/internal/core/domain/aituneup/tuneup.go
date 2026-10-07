package aituneup

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
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
	case KindRaiseToolTier, KindReorderProviders, KindLeaveShadow, KindAssignTask, KindTurnOffIdleAgent:
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
	Tasks            []aiprovider.Task  `json:"tasks,omitempty"`
	Model            string             `json:"model,omitempty"`
	LastRunAt        *int64             `json:"lastRunAt,omitempty"`
	IdleSince        int64              `json:"idleSince,omitempty"`
	Tools            int                `json:"tools,omitempty"`
}

type TuneUp struct {
	bun.BaseModel `bun:"table:ai_tune_ups,alias:aitu" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`

	Kind              Kind            `json:"kind" bun:"kind,type:VARCHAR(40),notnull"`
	Fingerprint       string          `json:"fingerprint"       bun:"fingerprint,type:VARCHAR(400),notnull"`
	AgentDefinitionID pulid.ID        `json:"agentDefinitionId" bun:"agent_definition_id,type:VARCHAR(100),nullzero"`
	ProviderID        pulid.ID        `json:"providerId"        bun:"provider_id,type:VARCHAR(100),nullzero"`
	OtherProviderID   pulid.ID        `json:"otherProviderId"   bun:"other_provider_id,type:VARCHAR(100),nullzero"`
	ToolName          string          `json:"toolName"          bun:"tool_name,type:VARCHAR(100),nullzero"`
	Task              aiprovider.Task `json:"task"              bun:"task,type:VARCHAR(100),nullzero"`
	Evidence          Evidence        `json:"evidence"          bun:"evidence,type:JSONB,notnull,default:'{}'"`

	Status         Status   `json:"status"         bun:"status,type:VARCHAR(20),notnull"`
	DismissedUntil *int64   `json:"dismissedUntil" bun:"dismissed_until,type:BIGINT,nullzero"`
	DecidedByID    pulid.ID `json:"decidedById"    bun:"decided_by_id,type:VARCHAR(100),nullzero"`
	DecidedAt      *int64   `json:"decidedAt"      bun:"decided_at,type:BIGINT,nullzero"`
	ComputedAt     int64    `json:"computedAt"     bun:"computed_at,type:BIGINT,notnull"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (t *TuneUp) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if t.ID.IsNil() {
			t.ID = pulid.MustNew(IDPrefix)
		}
		t.CreatedAt = now
		t.UpdatedAt = now
	case *bun.UpdateQuery:
		t.UpdatedAt = now
	}
	return nil
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
