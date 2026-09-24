package agent

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*Memory)(nil)
	_ validationframework.TenantedEntity = (*Memory)(nil)
)

// MaxMemoryContentChars bounds one memory. Every active memory is read into
// a prompt, so a long one costs every run the organization makes.
const MaxMemoryContentChars = 2000

type MemoryKind string

const (
	// MemoryKindInstruction is a standing rule a person wants agents to follow.
	MemoryKindInstruction = MemoryKind("Instruction")
	// MemoryKindFact is something an agent was told or worked out and should
	// weigh, not obey.
	MemoryKindFact = MemoryKind("Fact")
	// MemoryKindCorrection is what a decision on a proposal taught: what a
	// person changed or refused, and why.
	MemoryKindCorrection = MemoryKind("Correction")
)

func (k MemoryKind) IsValid() bool {
	switch k {
	case MemoryKindInstruction, MemoryKindFact, MemoryKindCorrection:
		return true
	default:
		return false
	}
}

// Rank orders kinds in a prompt: what to follow before what to weigh.
func (k MemoryKind) Rank() int {
	switch k {
	case MemoryKindInstruction:
		return 0
	case MemoryKindCorrection:
		return 1
	default:
		return 2
	}
}

type MemorySource string

const (
	MemorySourceUser     = MemorySource("User")
	MemorySourceAgent    = MemorySource("Agent")
	MemorySourceDecision = MemorySource("Decision")
	MemorySourceFeedback = MemorySource("Feedback")
)

func (s MemorySource) IsValid() bool {
	switch s {
	case MemorySourceUser, MemorySourceAgent, MemorySourceDecision, MemorySourceFeedback:
		return true
	default:
		return false
	}
}

func AllMemorySources() []MemorySource {
	return []MemorySource{
		MemorySourceUser,
		MemorySourceAgent,
		MemorySourceDecision,
		MemorySourceFeedback,
	}
}

func AllMemoryKinds() []MemoryKind {
	return []MemoryKind{MemoryKindInstruction, MemoryKindFact, MemoryKindCorrection}
}

type MemoryStatus string

const (
	MemoryStatusActive    = MemoryStatus("Active")
	MemoryStatusRetired   = MemoryStatus("Retired")
	MemoryStatusSuggested = MemoryStatus("Suggested")
	MemoryStatusDismissed = MemoryStatus("Dismissed")
)

func (s MemoryStatus) IsValid() bool {
	switch s {
	case MemoryStatusActive, MemoryStatusRetired, MemoryStatusSuggested, MemoryStatusDismissed:
		return true
	default:
		return false
	}
}

func (s MemoryStatus) IsSuggestion() bool {
	return s == MemoryStatusSuggested || s == MemoryStatusDismissed
}

func AllMemoryStatuses() []MemoryStatus {
	return []MemoryStatus{
		MemoryStatusActive,
		MemoryStatusRetired,
		MemoryStatusSuggested,
		MemoryStatusDismissed,
	}
}

type MemoryEvidence struct {
	FeedbackIDs     []pulid.ID `json:"feedbackIds"`
	PatternKey      string     `json:"patternKey"`
	RatingCount     int        `json:"ratingCount"`
	DistinctUsers   int        `json:"distinctUsers"`
	DistinctThreads int        `json:"distinctThreads"`
	Reason          string     `json:"reason"`
	Quotes          []string   `json:"quotes,omitempty"`
	FirstRatedAt    int64      `json:"firstRatedAt"`
	LastRatedAt     int64      `json:"lastRatedAt"`
}

func (e *MemoryEvidence) Count() int {
	if e == nil {
		return 0
	}

	return len(e.FeedbackIDs)
}

// MemorySubjectType is what a memory can be about beyond the organization as
// a whole. These are the records a standing instruction is usually about:
// "this customer", "this dock", "this driver", "this carrier".
type MemorySubjectType string

const (
	MemorySubjectCustomer = MemorySubjectType("Customer")
	MemorySubjectLocation = MemorySubjectType("Location")
	MemorySubjectWorker   = MemorySubjectType("Worker")
	MemorySubjectCarrier  = MemorySubjectType("Carrier")
)

func (t MemorySubjectType) IsValid() bool {
	switch t {
	case MemorySubjectCustomer, MemorySubjectLocation, MemorySubjectWorker, MemorySubjectCarrier:
		return true
	default:
		return false
	}
}

func AllMemorySubjectTypes() []MemorySubjectType {
	return []MemorySubjectType{
		MemorySubjectCustomer,
		MemorySubjectLocation,
		MemorySubjectWorker,
		MemorySubjectCarrier,
	}
}

// Memory is one thing an organization keeps for its agents between runs.
//
// It is scoped three ways, each optional: to a subject (a customer, a
// location, a driver, a carrier), to a tool (a correction to how that tool
// was proposed), or to neither, which makes it organization-wide and read
// into every prompt. It expires if it says so and is retired rather than
// deleted, so what an agent was told last month can still be read.
type Memory struct {
	bun.BaseModel `bun:"table:agent_memories,alias:amem" json:"-"`

	pagination.CursorValueSet `json:"-" bun:",embed"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,notnull,type:VARCHAR(100)"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,notnull,type:VARCHAR(100)"`

	Kind   MemoryKind   `json:"kind"   bun:"kind,type:VARCHAR(20),notnull"`
	Source MemorySource `json:"source" bun:"source,type:VARCHAR(20),notnull"`
	Status MemoryStatus `json:"status" bun:"status,type:VARCHAR(20),notnull,default:'Active'"`

	SubjectType  MemorySubjectType `json:"subjectType"  bun:"subject_type,type:VARCHAR(30),nullzero"`
	SubjectID    *pulid.ID         `json:"subjectId"    bun:"subject_id,type:VARCHAR(100),nullzero"`
	SubjectLabel string            `json:"subjectLabel" bun:"subject_label,type:VARCHAR(200),nullzero"`
	ToolName     string            `json:"toolName"     bun:"tool_name,type:VARCHAR(100),nullzero"`

	Content string `json:"content" bun:"content,type:TEXT,notnull"`

	AgentDefinitionID *pulid.ID `json:"agentDefinitionId" bun:"agent_definition_id,type:VARCHAR(100),nullzero"`
	SourceRunID       *pulid.ID `json:"sourceRunId"       bun:"source_run_id,type:VARCHAR(100),nullzero"`
	SourceProposalID  *pulid.ID `json:"sourceProposalId"  bun:"source_proposal_id,type:VARCHAR(100),nullzero"`
	CreatedByUserID   *pulid.ID `json:"createdByUserId"   bun:"created_by_user_id,type:VARCHAR(100),nullzero"`
	RetiredByUserID   *pulid.ID `json:"retiredByUserId"   bun:"retired_by_user_id,type:VARCHAR(100),nullzero"`
	RetiredAt         *int64    `json:"retiredAt"         bun:"retired_at,type:BIGINT,nullzero"`
	ExpiresAt         *int64    `json:"expiresAt"         bun:"expires_at,type:BIGINT,nullzero"`

	UseCount   int    `json:"useCount"   bun:"use_count,type:INTEGER,notnull,default:0"`
	LastUsedAt *int64 `json:"lastUsedAt" bun:"last_used_at,type:BIGINT,nullzero"`

	Evidence *MemoryEvidence `json:"evidence" bun:"evidence,type:JSONB,nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit *tenant.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *tenant.Organization `bun:"rel:belongs-to,join:organization_id=id"  json:"-"`
}

func (m *Memory) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(m,
		validation.Field(&m.OrganizationID, validation.Required.Error("Organization is required")),
		validation.Field(&m.BusinessUnitID, validation.Required.Error("Business unit is required")),
		validation.Field(&m.Kind,
			validation.Required.Error("Kind is required"),
			domainvalidation.ValidEnum[MemoryKind]("Kind is invalid"),
		),
		validation.Field(&m.Source,
			validation.Required.Error("Source is required"),
			domainvalidation.ValidEnum[MemorySource]("Source is invalid"),
		),
		validation.Field(&m.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[MemoryStatus]("Status is invalid"),
		),
		validation.Field(&m.SubjectType,
			validation.When(m.SubjectType != "",
				domainvalidation.ValidEnum[MemorySubjectType]("Subject type is invalid"),
			),
		),
		validation.Field(&m.Content,
			validation.Required.Error("Content is required"),
			validation.Length(1, MaxMemoryContentChars).
				Error("Content must be at most 2000 characters"),
		),
	))

	hasType := m.SubjectType != ""
	hasID := m.SubjectID != nil && m.SubjectID.IsNotNil()
	switch {
	case hasType && !hasID:
		multiErr.Add(
			"subjectId",
			errortypes.ErrRequired,
			"A subject type needs the record it names",
		)
	case hasID && !hasType:
		multiErr.Add("subjectType", errortypes.ErrRequired, "A subject id needs its type")
	}

	if m.ExpiresAt != nil && *m.ExpiresAt <= 0 {
		multiErr.Add("expiresAt", errortypes.ErrInvalid, "Expiry must be a time")
	}

	if m.Source == MemorySourceFeedback && m.Evidence.Count() == 0 {
		multiErr.Add(
			"evidence",
			errortypes.ErrRequired,
			"A memory drawn from feedback needs the ratings it was drawn from",
		)
	}
	if m.Status.IsSuggestion() && m.Source != MemorySourceFeedback {
		multiErr.Add("status", errortypes.ErrInvalid, "Only feedback can suggest a memory")
	}
}

// Active reports whether the memory should still be read: not retired and
// not past its expiry, when it has one.
func (m *Memory) Active(now int64) bool {
	if m.Status != MemoryStatusActive {
		return false
	}

	return m.ExpiresAt == nil || *m.ExpiresAt > now
}

// OrganizationWide reports whether the memory is about the organization as a
// whole rather than one record or one tool, and so belongs in every prompt.
func (m *Memory) OrganizationWide() bool {
	return m.SubjectType == "" && strings.TrimSpace(m.ToolName) == ""
}

// Scope is the memory's subject or tool in words, for a prompt line.
func (m *Memory) Scope() string {
	switch {
	case m.SubjectType != "" && m.SubjectLabel != "":
		return m.SubjectLabel + " (" + strings.ToLower(string(m.SubjectType)) + ")"
	case m.SubjectType != "" && m.SubjectID != nil:
		return strings.ToLower(string(m.SubjectType)) + " " + m.SubjectID.String()
	case m.ToolName != "":
		return "tool " + m.ToolName
	default:
		return ""
	}
}

func (m *Memory) GetID() pulid.ID { return m.ID }

func (m *Memory) GetCreatedAt() int64 { return m.CreatedAt }

func (m *Memory) GetOrganizationID() pulid.ID { return m.OrganizationID }

func (m *Memory) GetBusinessUnitID() pulid.ID { return m.BusinessUnitID }

func (m *Memory) GetTableName() string { return "agent_memories" }

func (m *Memory) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "amem",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "content", Type: domaintypes.FieldTypeText},
			{Name: "subject_label", Type: domaintypes.FieldTypeText},
			{Name: "tool_name", Type: domaintypes.FieldTypeText},
			{Name: "kind", Type: domaintypes.FieldTypeEnum},
			{Name: "status", Type: domaintypes.FieldTypeEnum},
		},
	}
}

func (m *Memory) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if m.ID.IsNil() {
			m.ID = pulid.MustNew("amem_")
		}
		if m.Status == "" {
			m.Status = MemoryStatusActive
		}
		m.CreatedAt = now
		m.UpdatedAt = now
	case *bun.UpdateQuery:
		m.UpdatedAt = now
	}

	return nil
}
