package agentreflection

import (
	"context"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmemory"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ validationframework.TenantedEntity = (*Reflection)(nil)
)

const (
	MaxReflectionLessons      = 5
	MaxReflectionNotesChars   = 500
	MaxReflectionErrorChars   = 1000
	MaxReflectionLessonChars  = 1200
	MaxReflectionChangeReason = 300
)

type ReflectionSubject string

const (
	ReflectionSubjectThread = ReflectionSubject("Thread")
	ReflectionSubjectRun    = ReflectionSubject("Run")
)

func (s ReflectionSubject) IsValid() bool {
	return s == ReflectionSubjectThread || s == ReflectionSubjectRun
}

func AllReflectionSubjects() []ReflectionSubject {
	return []ReflectionSubject{ReflectionSubjectThread, ReflectionSubjectRun}
}

type ReflectionStatus string

const (
	ReflectionStatusRunning   = ReflectionStatus("Running")
	ReflectionStatusSkipped   = ReflectionStatus("Skipped")
	ReflectionStatusCompleted = ReflectionStatus("Completed")
	ReflectionStatusFailed    = ReflectionStatus("Failed")
)

func (s ReflectionStatus) IsValid() bool {
	switch s {
	case ReflectionStatusRunning,
		ReflectionStatusSkipped,
		ReflectionStatusCompleted,
		ReflectionStatusFailed:
		return true
	default:
		return false
	}
}

func (s ReflectionStatus) Settled() bool {
	return s == ReflectionStatusSkipped || s == ReflectionStatusCompleted
}

func AllReflectionStatuses() []ReflectionStatus {
	return []ReflectionStatus{
		ReflectionStatusRunning,
		ReflectionStatusSkipped,
		ReflectionStatusCompleted,
		ReflectionStatusFailed,
	}
}

type ReflectionSkip string

const (
	ReflectionSkipNoSignal         = ReflectionSkip("NoSignal")
	ReflectionSkipNothingToRead    = ReflectionSkip("NothingToRead")
	ReflectionSkipLearningOff      = ReflectionSkip("LearningOff")
	ReflectionSkipAgentUnavailable = ReflectionSkip("AgentUnavailable")
	ReflectionSkipOverBudget       = ReflectionSkip("OverBudget")
)

func (s ReflectionSkip) IsValid() bool {
	switch s {
	case ReflectionSkipNoSignal,
		ReflectionSkipNothingToRead,
		ReflectionSkipLearningOff,
		ReflectionSkipAgentUnavailable,
		ReflectionSkipOverBudget:
		return true
	default:
		return false
	}
}

func AllReflectionSkips() []ReflectionSkip {
	return []ReflectionSkip{
		ReflectionSkipNoSignal,
		ReflectionSkipNothingToRead,
		ReflectionSkipLearningOff,
		ReflectionSkipAgentUnavailable,
		ReflectionSkipOverBudget,
	}
}

type ReflectionSignalKind string

const (
	ReflectionSignalToolRecovered    = ReflectionSignalKind("ToolRecovered")
	ReflectionSignalToolFailed       = ReflectionSignalKind("ToolFailed")
	ReflectionSignalPersonCorrected  = ReflectionSignalKind("PersonCorrected")
	ReflectionSignalStandingRequest  = ReflectionSignalKind("StandingRequest")
	ReflectionSignalProposalModified = ReflectionSignalKind("ProposalModified")
	ReflectionSignalProposalRejected = ReflectionSignalKind("ProposalRejected")
	ReflectionSignalNegativeFeedback = ReflectionSignalKind("NegativeFeedback")
	ReflectionSignalLongTask         = ReflectionSignalKind("LongTask")
)

func (k ReflectionSignalKind) IsValid() bool {
	switch k {
	case ReflectionSignalToolRecovered,
		ReflectionSignalToolFailed,
		ReflectionSignalPersonCorrected,
		ReflectionSignalStandingRequest,
		ReflectionSignalProposalModified,
		ReflectionSignalProposalRejected,
		ReflectionSignalNegativeFeedback,
		ReflectionSignalLongTask:
		return true
	default:
		return false
	}
}

func (k ReflectionSignalKind) Describe() string {
	switch k {
	case ReflectionSignalToolRecovered:
		return "a tool call failed and a later call to the same tool worked"
	case ReflectionSignalToolFailed:
		return "a tool call failed or was refused as invalid"
	case ReflectionSignalPersonCorrected:
		return "the person corrected or redirected the agent"
	case ReflectionSignalStandingRequest:
		return "the person said how they want something done from now on"
	case ReflectionSignalProposalModified:
		return "a person changed a write the agent proposed before approving it"
	case ReflectionSignalProposalRejected:
		return "a person refused a write the agent proposed"
	case ReflectionSignalNegativeFeedback:
		return "a person rated a reply as unhelpful"
	case ReflectionSignalLongTask:
		return "the agent took many steps to finish the task"
	default:
		return string(k)
	}
}

type ReflectionSignal struct {
	Kind   ReflectionSignalKind `json:"kind"`
	Count  int                  `json:"count"`
	Detail string               `json:"detail,omitempty"`
}

type ReflectionSignals []ReflectionSignal

func (s ReflectionSignals) Has(kind ReflectionSignalKind) bool {
	for _, signal := range s {
		if signal.Kind == kind && signal.Count > 0 {
			return true
		}
	}

	return false
}

func (s ReflectionSignals) Kinds() []string {
	kinds := make([]string, 0, len(s))
	for _, signal := range s {
		if signal.Count > 0 {
			kinds = append(kinds, string(signal.Kind))
		}
	}

	return kinds
}

type ReflectionAction string

const (
	ReflectionActionSaved     = ReflectionAction("Saved")
	ReflectionActionSuggested = ReflectionAction("Suggested")
	ReflectionActionRefreshed = ReflectionAction("Refreshed")
	ReflectionActionRefused   = ReflectionAction("Refused")
)

type ReflectionChange struct {
	Action       ReflectionAction        `json:"action"`
	MemoryID     *pulid.ID               `json:"memoryId,omitempty"`
	SupersedesID *pulid.ID               `json:"supersedesId,omitempty"`
	Kind         agentmemory.MemoryKind  `json:"kind"`
	Scope        agentmemory.MemoryScope `json:"scope,omitempty"`
	Content      string                  `json:"content"`
	Reason       string                  `json:"reason,omitempty"`
}

func (c *ReflectionChange) Kept() bool {
	return c.MemoryID != nil && c.MemoryID.IsNotNil() &&
		(c.Action == ReflectionActionSaved || c.Action == ReflectionActionSuggested)
}

type Reflection struct {
	keel.Table `keel:"agent_reflections"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:arfl_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	AgentDefinitionID pulid.ID          `json:"agentDefinitionId" keel:"agent_definition_id"`
	SubjectType       ReflectionSubject `json:"subjectType"       keel:"subject_type"`
	ThreadID          *pulid.ID         `json:"threadId"          keel:"thread_id"`
	RunID             *pulid.ID         `json:"runId"             keel:"run_id"`
	UserID            *pulid.ID         `json:"userId"            keel:"user_id"`
	FromSequence      int               `json:"fromSequence"      keel:"from_sequence"`
	ThroughSequence   int               `json:"throughSequence"   keel:"through_sequence"`

	Status     ReflectionStatus   `json:"status"     keel:"status"`
	SkipReason ReflectionSkip     `json:"skipReason" keel:"skip_reason,nullzero"`
	Signals    ReflectionSignals  `json:"signals"    keel:"signals,jsonb"`
	Changes    []ReflectionChange `json:"changes"    keel:"changes,jsonb"`
	Notes      string             `json:"notes"      keel:"notes,nullzero"`
	Tainted    bool               `json:"tainted"    keel:"tainted"`

	Model        string   `json:"model"        keel:"model,nullzero"`
	ProviderID   pulid.ID `json:"providerId"   keel:"provider_id,nullzero"`
	InputTokens  int      `json:"inputTokens"  keel:"input_tokens"`
	OutputTokens int      `json:"outputTokens" keel:"output_tokens"`
	ErrorMessage string   `json:"errorMessage" keel:"error_message,nullzero"`
	FinishedAt   *int64   `json:"finishedAt"   keel:"finished_at"`

	Version   int64 `json:"version"   keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit `json:"-"`
	Organization *tenant.Organization `json:"-"`
}

func (r *Reflection) BeforeCreate(_ context.Context) error {
	if r.Status == "" {
		r.Status = ReflectionStatusRunning
	}
	if r.Signals == nil {
		r.Signals = ReflectionSignals{}
	}
	if r.Changes == nil {
		r.Changes = []ReflectionChange{}
	}

	return nil
}

func (r *Reflection) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(r,
		validation.Field(&r.OrganizationID, validation.Required.Error("Organization is required")),
		validation.Field(&r.BusinessUnitID, validation.Required.Error("Business unit is required")),
		validation.Field(&r.AgentDefinitionID, validation.Required.Error("Agent is required")),
		validation.Field(&r.SubjectType,
			validation.Required.Error("Subject type is required"),
			domainvalidation.ValidEnum[ReflectionSubject]("Subject type is invalid"),
		),
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[ReflectionStatus]("Status is invalid"),
		),
		validation.Field(&r.SkipReason,
			validation.When(r.SkipReason != "",
				domainvalidation.ValidEnum[ReflectionSkip]("Skip reason is invalid"),
			),
		),
	))

	switch r.SubjectType {
	case ReflectionSubjectThread:
		if r.ThreadID == nil || r.ThreadID.IsNil() {
			multiErr.Add(
				"threadId",
				errortypes.ErrRequired,
				"A look back at a conversation needs it",
			)
		}
	case ReflectionSubjectRun:
		if r.RunID == nil || r.RunID.IsNil() {
			multiErr.Add("runId", errortypes.ErrRequired, "A look back at a run needs it")
		}
	}
	if (r.Status == ReflectionStatusSkipped) != (r.SkipReason != "") {
		multiErr.Add(
			"skipReason",
			errortypes.ErrInvalid,
			"Only a skipped look back says why it was skipped",
		)
	}
	if r.FromSequence < 0 || r.ThroughSequence < r.FromSequence {
		multiErr.Add("throughSequence", errortypes.ErrInvalid, "The stretch read must run forward")
	}
}

func (r *Reflection) Skip(reason ReflectionSkip, at int64) {
	r.Status = ReflectionStatusSkipped
	r.SkipReason = reason
	r.FinishedAt = &at
}

func (r *Reflection) Fail(message string, at int64) {
	r.Status = ReflectionStatusFailed
	r.SkipReason = ""
	r.ErrorMessage = stringutils.TruncateRunes(strings.TrimSpace(message), MaxReflectionErrorChars)
	r.FinishedAt = &at
}

func (r *Reflection) Complete(changes []ReflectionChange, notes string, at int64) {
	r.Status = ReflectionStatusCompleted
	r.SkipReason = ""
	r.ErrorMessage = ""
	r.Changes = changes
	r.Notes = stringutils.TruncateRunes(strings.TrimSpace(notes), MaxReflectionNotesChars)
	r.FinishedAt = &at
}

func (r *Reflection) KeptMemoryIDs() []pulid.ID {
	ids := make([]pulid.ID, 0, len(r.Changes))
	for _, change := range r.Changes {
		if change.Kept() {
			ids = append(ids, *change.MemoryID)
		}
	}

	return ids
}

func (r *Reflection) GetID() pulid.ID { return r.ID }

func (r *Reflection) GetOrganizationID() pulid.ID { return r.OrganizationID }

func (r *Reflection) GetBusinessUnitID() pulid.ID { return r.BusinessUnitID }

func (r *Reflection) GetTableName() string { return "agent_reflections" }

func (r *Reflection) GetCreatedAt() int64 { return r.CreatedAt }

func (r *Reflection) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "arfl",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "notes", Type: domaintypes.FieldTypeText},
			{Name: "status", Type: domaintypes.FieldTypeEnum},
			{Name: "subject_type", Type: domaintypes.FieldTypeEnum},
		},
	}
}
