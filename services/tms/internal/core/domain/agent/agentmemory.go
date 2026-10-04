package agent

import (
	"context"
	"strings"
	"unicode/utf8"

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

// MaxMemoryContentChars bounds one memory. A prompt shows at most
// MemoryPromptExcerptChars of it and names its id, so the rest is read through
// recall_memory rather than paid for by every run.
const MaxMemoryContentChars = 4000

const (
	MemoryPromptExcerptChars = 1200
	MaxMemoryCandidates      = 500
	MemoryActiveSoftCap      = 5000
	MemoryActiveWarnAt       = MemoryActiveSoftCap * 4 / 5
	DefaultMemoryRecallLimit = 10
	MaxMemoryRecallLimit     = 50
)

// MemoryStaleAfterSeconds is how long a memory that belongs in every prompt
// may go untouched, while the memories kept for the same readers are being
// used, before the prompt stops carrying it. A stale memory is not retired:
// recall_memory still finds it and the Memory page still lists it, and the
// first time it is recalled, restated or edited it is fresh again.
const MemoryStaleAfterSeconds = 90 * 24 * 60 * 60

type MemoryMatch string

const (
	MemoryMatchWords   = MemoryMatch("words")
	MemoryMatchMeaning = MemoryMatch("meaning")
	MemoryMatchBoth    = MemoryMatch("both")
)

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

// MemoryScope is who a memory is read for. Organization and Agent are about
// which agents read it; User and Role are about whose conversations it reaches:
// one person's own ("Just you" on the Desk), or everyone holding one role (the
// Desk calls that the person's team, since roles are how people are grouped).
type MemoryScope string

const (
	MemoryScopeOrganization = MemoryScope("Organization")
	MemoryScopeAgent        = MemoryScope("Agent")
	MemoryScopeUser         = MemoryScope("User")
	MemoryScopeRole         = MemoryScope("Role")
)

func (s MemoryScope) IsValid() bool {
	switch s {
	case MemoryScopeOrganization, MemoryScopeAgent, MemoryScopeUser, MemoryScopeRole:
		return true
	default:
		return false
	}
}

// Personal reports whether the scope narrows a memory to some people rather
// than to some agents.
func (s MemoryScope) Personal() bool {
	return s == MemoryScopeUser || s == MemoryScopeRole
}

func AllMemoryScopes() []MemoryScope {
	return []MemoryScope{
		MemoryScopeOrganization,
		MemoryScopeAgent,
		MemoryScopeUser,
		MemoryScopeRole,
	}
}

// MemoryReader is the person a memory is read for: their own memories and
// their roles' reach them, nobody else's. A reader with no user, a run nobody
// is in, reads only what is kept for the organization or for its agent.
type MemoryReader struct {
	UserID  pulid.ID
	RoleIDs []pulid.ID
}

// Reads reports whether a memory of this scope reaches the reader. Agent
// scope is the agent's business, not the person's, and is decided elsewhere.
func (r MemoryReader) Reads(m *Memory) bool {
	switch m.Scope {
	case MemoryScopeUser:
		return r.UserID.IsNotNil() && m.OwnerUserID != nil && *m.OwnerUserID == r.UserID
	case MemoryScopeRole:
		if m.RoleID == nil {
			return false
		}
		for _, id := range r.RoleIDs {
			if id == *m.RoleID {
				return true
			}
		}

		return false
	default:
		return true
	}
}

// MemorySavingMode is how a person wants new memories an agent picks up in
// their conversations to be kept: saved and announced, or offered first.
type MemorySavingMode string

const (
	MemorySavingAutomatic = MemorySavingMode("Automatic")
	MemorySavingAskFirst  = MemorySavingMode("AskFirst")
)

func (m MemorySavingMode) IsValid() bool {
	return m == MemorySavingAutomatic || m == MemorySavingAskFirst
}

func AllMemorySavingModes() []MemorySavingMode {
	return []MemorySavingMode{MemorySavingAutomatic, MemorySavingAskFirst}
}

func AllMemoryKinds() []MemoryKind {
	return []MemoryKind{MemoryKindInstruction, MemoryKindFact, MemoryKindCorrection}
}

type MemoryStatus string

const (
	MemoryStatusActive = MemoryStatus("Active")
	// MemoryStatusPaused is a memory a person set aside without forgetting
	// it: kept, listed, and read by no agent until it is resumed.
	MemoryStatusPaused    = MemoryStatus("Paused")
	MemoryStatusRetired   = MemoryStatus("Retired")
	MemoryStatusSuggested = MemoryStatus("Suggested")
	MemoryStatusDismissed = MemoryStatus("Dismissed")
)

func (s MemoryStatus) IsValid() bool {
	switch s {
	case MemoryStatusActive,
		MemoryStatusPaused,
		MemoryStatusRetired,
		MemoryStatusSuggested,
		MemoryStatusDismissed:
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
		MemoryStatusPaused,
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

	AgentDefinitionID *pulid.ID   `json:"agentDefinitionId" bun:"agent_definition_id,type:VARCHAR(100),nullzero"`
	Scope             MemoryScope `json:"scope"             bun:"scope,type:VARCHAR(20),notnull,default:'Organization'"`
	// OwnerUserID is the person a User-scoped memory is kept for, and RoleID
	// the role a Role-scoped one is kept for. Each is empty for every other
	// scope.
	OwnerUserID *pulid.ID `json:"ownerUserId" bun:"owner_user_id,type:VARCHAR(100),nullzero"`
	RoleID      *pulid.ID `json:"roleId"      bun:"role_id,type:VARCHAR(100),nullzero"`
	// SourceThreadID is the conversation the run that recorded the memory
	// was answering, so a list can say which conversation it came from.
	SourceThreadID   *pulid.ID `json:"sourceThreadId" bun:"source_thread_id,type:VARCHAR(100),nullzero"`
	Tainted          bool      `json:"tainted"           bun:"tainted,type:BOOLEAN,notnull,default:false"`
	TaintRunID       *pulid.ID `json:"taintRunId"        bun:"taint_run_id,type:VARCHAR(100),nullzero"`
	SourceRunID      *pulid.ID `json:"sourceRunId"       bun:"source_run_id,type:VARCHAR(100),nullzero"`
	SourceProposalID *pulid.ID `json:"sourceProposalId"  bun:"source_proposal_id,type:VARCHAR(100),nullzero"`
	CreatedByUserID  *pulid.ID `json:"createdByUserId"   bun:"created_by_user_id,type:VARCHAR(100),nullzero"`
	RetiredByUserID  *pulid.ID `json:"retiredByUserId"   bun:"retired_by_user_id,type:VARCHAR(100),nullzero"`
	RetiredAt        *int64    `json:"retiredAt"         bun:"retired_at,type:BIGINT,nullzero"`
	ExpiresAt        *int64    `json:"expiresAt"         bun:"expires_at,type:BIGINT,nullzero"`

	UseCount   int    `json:"useCount"   bun:"use_count,type:INTEGER,notnull,default:0"`
	LastUsedAt *int64 `json:"lastUsedAt" bun:"last_used_at,type:BIGINT,nullzero"`

	Evidence *MemoryEvidence `json:"evidence" bun:"evidence,type:JSONB,nullzero"`

	SearchVector string `json:"-" bun:"search_vector,type:TSVECTOR,scanonly"`

	// Refreshed is set, never stored, on the memory a save returns when an
	// active one already said the same thing for the same readers: that one
	// was refreshed rather than a duplicate recorded beside it.
	Refreshed bool `json:"-" bun:"-"`

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
		validation.Field(&m.Scope,
			validation.When(m.Scope != "",
				domainvalidation.ValidEnum[MemoryScope]("Scope is invalid"),
			),
		),
		validation.Field(&m.SubjectType,
			validation.When(m.SubjectType != "",
				domainvalidation.ValidEnum[MemorySubjectType]("Subject type is invalid"),
			),
		),
		validation.Field(&m.Content, validation.Required.Error("Content is required")),
	))

	if utf8.RuneCountInString(m.Content) > MaxMemoryContentChars {
		multiErr.Add(
			"content",
			errortypes.ErrInvalid,
			"Content must be at most {0} characters",
			MaxMemoryContentChars,
		)
	}

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
	if m.AgentScoped() && (m.AgentDefinitionID == nil || m.AgentDefinitionID.IsNil()) {
		multiErr.Add("scope", errortypes.ErrInvalid, "A memory kept for one agent needs that agent")
	}
	switch m.Scope {
	case MemoryScopeUser:
		if m.OwnerUserID == nil || m.OwnerUserID.IsNil() {
			multiErr.Add("scope", errortypes.ErrInvalid, "A memory kept for one person needs that person")
		}
	case MemoryScopeRole:
		if m.RoleID == nil || m.RoleID.IsNil() {
			multiErr.Add("roleId", errortypes.ErrRequired, "A memory kept for a role needs the role")
		}
	}
	// A suggestion is drawn from feedback, or offered by an agent to the
	// person who asked to be asked first; nothing else waits to be accepted.
	if m.Status.IsSuggestion() && m.Source != MemorySourceFeedback &&
		m.Source != MemorySourceAgent {
		multiErr.Add("status", errortypes.ErrInvalid, "Only feedback or an agent can suggest a memory")
	}
}

// SetAudience narrows the memory to one scope's readers, clearing whatever
// another scope had named.
func (m *Memory) SetAudience(scope MemoryScope, ownerID, roleID pulid.ID) {
	m.Scope = scope
	m.OwnerUserID = nil
	m.RoleID = nil
	switch scope {
	case MemoryScopeUser:
		if ownerID.IsNotNil() {
			m.OwnerUserID = &ownerID
		}
	case MemoryScopeRole:
		if roleID.IsNotNil() {
			m.RoleID = &roleID
		}
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

// LastTouchedAt is the last time anyone had a reason to keep the memory: it
// was recorded, changed, restated, recalled or carried by a prompt.
func (m *Memory) LastTouchedAt() int64 {
	touched := max(m.CreatedAt, m.UpdatedAt)
	if m.LastUsedAt != nil {
		touched = max(touched, *m.LastUsedAt)
	}

	return touched
}

// WithoutStaleMemories drops the memories a prompt should no longer carry
// because they have gone unused.
//
// Only a memory about nothing in particular can go stale. One about a
// customer or a tool rides along only when that record or tool comes up, so
// its going unused says the record was quiet, not that the memory is wrong.
//
// Staleness is measured against the newest touch among the memories kept for
// the same readers, not against the clock: a person back from three months
// away, or an organization that paused its agents, finds what it kept still
// in the prompt, while a memory that sat unused as its neighbors were read
// every day is left out.
func WithoutStaleMemories(memories []*Memory) []*Memory {
	newest := make(map[string]int64, len(memories))
	for _, memory := range memories {
		if memory == nil {
			continue
		}
		key := memory.audienceKey()
		newest[key] = max(newest[key], memory.LastTouchedAt())
	}

	kept := make([]*Memory, 0, len(memories))
	for _, memory := range memories {
		if memory == nil {
			continue
		}
		if memory.OrganizationWide() &&
			newest[memory.audienceKey()]-memory.LastTouchedAt() > MemoryStaleAfterSeconds {
			continue
		}
		kept = append(kept, memory)
	}

	return kept
}

// audienceKey names who reads the memory: the organization, one agent, one
// person or one role.
func (m *Memory) audienceKey() string {
	key := string(m.Scope)
	switch {
	case m.Scope == MemoryScopeAgent && m.AgentDefinitionID != nil:
		key += ":" + m.AgentDefinitionID.String()
	case m.Scope == MemoryScopeUser && m.OwnerUserID != nil:
		key += ":" + m.OwnerUserID.String()
	case m.Scope == MemoryScopeRole && m.RoleID != nil:
		key += ":" + m.RoleID.String()
	}

	return key
}

func (m *Memory) AgentScoped() bool {
	return m.Scope == MemoryScopeAgent
}

func (m *Memory) ApprovedByPerson() bool {
	return m != nil && m.SourceProposalID != nil && m.SourceProposalID.IsNotNil() &&
		m.CreatedByUserID != nil && m.CreatedByUserID.IsNotNil()
}

func (m *Memory) DrawnFromOutside() bool {
	return m != nil && m.Tainted && !m.ApprovedByPerson()
}

func (m *Memory) TaintedRecords() []RecordRef {
	if m == nil || !m.Tainted {
		return nil
	}

	return []RecordRef{{EntityType: TaintEntityAgentMemory, ID: m.ID.String()}}
}

// About is the memory's subject or tool in words, for a prompt line.
func (m *Memory) About() string {
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
		if m.Scope == "" {
			m.Scope = MemoryScopeOrganization
		}
		m.CreatedAt = now
		m.UpdatedAt = now
	case *bun.UpdateQuery:
		m.UpdatedAt = now
	}

	return nil
}
