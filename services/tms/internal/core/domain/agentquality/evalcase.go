package agentquality

import (
	"context"
	"slices"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/hashutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*EvalCase)(nil)
	_ validationframework.TenantedEntity = (*EvalCase)(nil)
	_ domaintypes.PostgresSearchable     = (*EvalCase)(nil)
	_ pagination.CursorEntity            = (*EvalCase)(nil)
)

const (
	MaxInputChars   = 20000
	MaxRubricChars  = 4000
	MaxTitleChars   = 200
	MaxHistory      = 200
	MaxToolFixtures = 100

	RestrictedPlaceholderPrefix = "[restricted:"
)

type CaseSource string

const (
	CaseSourceDecidedProposal = CaseSource("DecidedProposal")
	CaseSourceThumbsUp        = CaseSource("ThumbsUp")
	CaseSourceCurated         = CaseSource("Curated")
)

func AllCaseSources() []CaseSource {
	return []CaseSource{CaseSourceDecidedProposal, CaseSourceThumbsUp, CaseSourceCurated}
}

func (s CaseSource) IsValid() bool {
	return slices.Contains(AllCaseSources(), s)
}

func (s CaseSource) Weight() float64 {
	if s == CaseSourceThumbsUp {
		return 0.6
	}

	return 1.0
}

type CaseStatus string

const (
	CaseStatusCandidate   = CaseStatus("Candidate")
	CaseStatusActive      = CaseStatus("Active")
	CaseStatusQuarantined = CaseStatus("Quarantined")
	CaseStatusRetired     = CaseStatus("Retired")
)

func AllCaseStatuses() []CaseStatus {
	return []CaseStatus{
		CaseStatusCandidate,
		CaseStatusActive,
		CaseStatusQuarantined,
		CaseStatusRetired,
	}
}

func (s CaseStatus) IsValid() bool {
	return slices.Contains(AllCaseStatuses(), s)
}

var caseTransitions = map[CaseStatus][]CaseStatus{
	CaseStatusCandidate:   {CaseStatusActive, CaseStatusRetired},
	CaseStatusActive:      {CaseStatusQuarantined, CaseStatusRetired},
	CaseStatusQuarantined: {CaseStatusActive, CaseStatusRetired},
	CaseStatusRetired:     {CaseStatusActive},
}

func (s CaseStatus) CanBecome(next CaseStatus) bool {
	return slices.Contains(caseTransitions[s], next)
}

func (s CaseStatus) Runs() bool {
	return s == CaseStatusActive
}

type HistoryMessage struct {
	Role       conversation.Role             `json:"role"`
	Content    string                        `json:"content,omitempty"`
	ToolCalls  []conversation.ToolCallRecord `json:"toolCalls,omitempty"`
	ToolCallID string                        `json:"toolCallId,omitempty"`
	ToolName   string                        `json:"toolName,omitempty"`
	ToolFailed bool                          `json:"toolFailed,omitempty"`
	CreatedAt  int64                         `json:"createdAt,omitempty"`
}

func HistoryFromMessages(messages []conversation.Message) []HistoryMessage {
	history := make([]HistoryMessage, 0, len(messages))
	for i := range messages {
		message := &messages[i]
		calls := make([]conversation.ToolCallRecord, 0, len(message.ToolCalls))
		for _, call := range message.ToolCalls {
			calls = append(calls, conversation.ToolCallRecord{
				ID:        call.ID,
				Name:      call.Name,
				Arguments: call.Arguments,
				Effect:    call.Effect,
			})
		}
		history = append(history, HistoryMessage{
			Role:       message.Role,
			Content:    message.Content,
			ToolCalls:  calls,
			ToolCallID: message.ToolCallID,
			ToolName:   message.ToolName,
			ToolFailed: message.ToolFailed,
			CreatedAt:  message.CreatedAt,
		})
	}

	return history
}

func (h HistoryMessage) Message() conversation.Message {
	return conversation.Message{
		Role:       h.Role,
		Kind:       conversation.MessageKindMessage,
		Content:    h.Content,
		ToolCalls:  h.ToolCalls,
		ToolCallID: h.ToolCallID,
		ToolName:   h.ToolName,
		ToolFailed: h.ToolFailed,
		CreatedAt:  h.CreatedAt,
	}
}

type ToolFixture struct {
	Tool   string         `json:"tool"`
	Args   map[string]any `json:"args,omitempty"`
	Result any            `json:"result,omitempty"`
	Failed bool           `json:"failed"`
}

type RedactedField struct {
	Tool        string `json:"tool"`
	Path        string `json:"path"`
	Sensitivity string `json:"sensitivity"`
}

type Redaction struct {
	Fields     []RedactedField `json:"fields"`
	RedactedAt int64           `json:"redactedAt"`
}

func RestrictedPlaceholder(field string) string {
	return RestrictedPlaceholderPrefix + field + "]"
}

type EvalCase struct {
	bun.BaseModel `bun:"table:agent_eval_cases,alias:aec" json:"-"`

	pagination.CursorValueSet `json:"-" bun:",embed"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,notnull,type:VARCHAR(100)"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,notnull,type:VARCHAR(100)"`

	AgentDefinitionID pulid.ID         `json:"agentDefinitionId" bun:"agent_definition_id,type:VARCHAR(100),notnull"`
	Title             string           `json:"title"             bun:"title,type:VARCHAR(200),nullzero"`
	Source            CaseSource       `json:"source"            bun:"source,type:VARCHAR(30),notnull"`
	Status            CaseStatus       `json:"status"            bun:"status,type:VARCHAR(20),notnull,default:'Candidate'"`
	Trigger           agent.RunTrigger `json:"trigger"           bun:"trigger,type:VARCHAR(20),notnull"`

	SourceRunID      *pulid.ID `json:"sourceRunId"      bun:"source_run_id,type:VARCHAR(100),nullzero"`
	SourceTurnID     *pulid.ID `json:"sourceTurnId"     bun:"source_turn_id,type:VARCHAR(100),nullzero"`
	SourceThreadID   *pulid.ID `json:"sourceThreadId"   bun:"source_thread_id,type:VARCHAR(100),nullzero"`
	SourceMessageID  *pulid.ID `json:"sourceMessageId"  bun:"source_message_id,type:VARCHAR(100),nullzero"`
	SourceProposalID *pulid.ID `json:"sourceProposalId" bun:"source_proposal_id,type:VARCHAR(100),nullzero"`
	SourceFeedbackID *pulid.ID `json:"sourceFeedbackId" bun:"source_feedback_id,type:VARCHAR(100),nullzero"`

	Input        string             `json:"input"        bun:"input,type:TEXT,notnull"`
	History      []HistoryMessage   `json:"history"      bun:"history,type:JSONB,notnull,default:'[]'"`
	PageContext  *agent.PageContext `json:"pageContext"  bun:"page_context,type:JSONB,nullzero"`
	Mentions     []agent.EntityRef  `json:"mentions"     bun:"mentions,type:JSONB,notnull,default:'[]'"`
	SubjectType  agent.SubjectType  `json:"subjectType"  bun:"subject_type,type:VARCHAR(50),nullzero"`
	SubjectID    pulid.ID           `json:"subjectId"    bun:"subject_id,type:VARCHAR(100),nullzero"`
	HeldTools    []string           `json:"heldTools"    bun:"held_tools,type:TEXT[],array,notnull,default:'{}'"`
	ToolFixtures []ToolFixture      `json:"toolFixtures" bun:"tool_fixtures,type:JSONB,notnull,default:'[]'"`

	Expected            Expected           `json:"expected"            bun:"expected,type:JSONB,notnull"`
	Rubric              string             `json:"rubric"              bun:"rubric,type:TEXT,nullzero"`
	Redaction           *Redaction         `json:"redaction"           bun:"redaction,type:JSONB,nullzero"`
	ContentHash         string             `json:"contentHash"         bun:"content_hash,type:VARCHAR(64),notnull"`
	CapturedFingerprint *agent.Fingerprint `json:"capturedFingerprint" bun:"captured_fingerprint,type:JSONB,nullzero"`
	ExpiresAt           *int64             `json:"expiresAt"           bun:"expires_at,type:BIGINT,nullzero"`
	CreatedByUserID     *pulid.ID          `json:"createdByUserId"     bun:"created_by_user_id,type:VARCHAR(100),nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit *tenant.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *tenant.Organization `bun:"rel:belongs-to,join:organization_id=id"  json:"-"`
}

func (c *EvalCase) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(c,
		validation.Field(&c.OrganizationID, validation.Required.Error("Organization is required")),
		validation.Field(&c.BusinessUnitID, validation.Required.Error("Business unit is required")),
		validation.Field(&c.AgentDefinitionID, validation.Required.Error("Agent is required")),
		validation.Field(&c.Title,
			validation.Length(0, MaxTitleChars).Error("Title is at most 200 characters"),
		),
		validation.Field(&c.Source,
			validation.Required.Error("Source is required"),
			domainvalidation.ValidEnum[CaseSource]("Source is invalid"),
		),
		validation.Field(&c.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[CaseStatus]("Status is invalid"),
		),
		validation.Field(&c.Trigger,
			validation.Required.Error("Trigger is required"),
			domainvalidation.ValidEnum[agent.RunTrigger]("Trigger is invalid"),
		),
		validation.Field(&c.Input,
			validation.Required.Error("The question the case asks is required"),
			validation.Length(0, MaxInputChars).Error("The question is at most 20000 characters"),
		),
		validation.Field(&c.History,
			validation.Length(0, MaxHistory).Error("A case keeps at most 200 earlier messages"),
		),
		validation.Field(&c.ToolFixtures,
			validation.Length(0, MaxToolFixtures).Error("A case keeps at most 100 tool results"),
		),
		validation.Field(&c.SubjectType,
			domainvalidation.ValidEnum[agent.SubjectType]("Subject type is invalid"),
		),
		validation.Field(&c.Rubric,
			validation.Length(0, MaxRubricChars).Error("Rubric is at most 4000 characters"),
		),
		validation.Field(&c.ContentHash, validation.Required.Error("Content hash is required")),
	))

	if strings.TrimSpace(c.Input) == "" && c.Input != "" {
		multiErr.Add("input", errortypes.ErrRequired, "The question the case asks is required")
	}
	if (c.SubjectType == "") != c.SubjectID.IsNil() {
		multiErr.Add(
			"subjectId",
			errortypes.ErrInvalid,
			"A subject type and id go together; give both or neither",
		)
	}
	c.validateSource(multiErr)
	c.validateHeldTools(multiErr)
	for i, fixture := range c.ToolFixtures {
		if strings.TrimSpace(fixture.Tool) == "" {
			multiErr.WithIndex("toolFixtures", i).Add(
				"tool",
				errortypes.ErrRequired,
				"A tool result names its tool",
			)
		}
	}
	agent.ValidateEntityRefs("mentions", c.Mentions, multiErr)
	if c.ExpiresAt != nil && *c.ExpiresAt <= 0 {
		multiErr.Add("expiresAt", errortypes.ErrInvalid, "Expiry must be a moment in time")
	}

	c.Expected.Validate(multiErr.WithPrefix("expected"))
	if c.Status == CaseStatusActive && c.Expected.IsEmpty() {
		multiErr.Add(
			"expected",
			errortypes.ErrRequired,
			"An active case must expect something to be scored against",
		)
	}
}

func (c *EvalCase) validateSource(multiErr *errortypes.MultiError) {
	switch c.Source {
	case CaseSourceDecidedProposal:
		if isNilID(c.SourceProposalID) {
			multiErr.Add(
				"sourceProposalId",
				errortypes.ErrRequired,
				"A case from a decision names the proposal that was decided",
			)
		}
	case CaseSourceThumbsUp:
		if isNilID(c.SourceMessageID) {
			multiErr.Add(
				"sourceMessageId",
				errortypes.ErrRequired,
				"A case from a liked reply names the reply",
			)
		}
	}
}

func (c *EvalCase) validateHeldTools(multiErr *errortypes.MultiError) {
	seen := make(map[string]struct{}, len(c.HeldTools))
	for i, name := range c.HeldTools {
		if strings.TrimSpace(name) == "" {
			multiErr.WithIndex("heldTools", i).Add(
				"name",
				errortypes.ErrRequired,
				"A held tool must be named",
			)
			continue
		}
		if _, dup := seen[name]; dup {
			multiErr.WithIndex("heldTools", i).Add(
				"name",
				errortypes.ErrDuplicate,
				"A tool is held once",
			)
		}
		seen[name] = struct{}{}
	}
}

func (c *EvalCase) Weight() float64 {
	return c.Source.Weight()
}

func (c *EvalCase) Conversation() []conversation.Message {
	messages := make([]conversation.Message, 0, len(c.History))
	for _, message := range c.History {
		messages = append(messages, message.Message())
	}

	return messages
}

func (c *EvalCase) IsChat() bool {
	return c.Trigger == agent.RunTriggerChat
}

type contentKey struct {
	AgentDefinitionID pulid.ID           `json:"agentDefinitionId"`
	Input             string             `json:"input"`
	History           []HistoryMessage   `json:"history"`
	PageContext       *agent.PageContext `json:"pageContext"`
	Mentions          []agent.EntityRef  `json:"mentions"`
	SubjectType       agent.SubjectType  `json:"subjectType"`
	SubjectID         pulid.ID           `json:"subjectId"`
}

func (c *EvalCase) ComputeContentHash() (string, error) {
	encoded, err := sonic.ConfigStd.Marshal(contentKey{
		AgentDefinitionID: c.AgentDefinitionID,
		Input:             strings.TrimSpace(c.Input),
		History:           c.History,
		PageContext:       c.PageContext,
		Mentions:          c.Mentions,
		SubjectType:       c.SubjectType,
		SubjectID:         c.SubjectID,
	})
	if err != nil {
		return "", err
	}

	return hashutils.SHA256BytesHex(encoded), nil
}

func (c *EvalCase) FixtureText() string {
	var builder strings.Builder
	for _, fixture := range c.ToolFixtures {
		if fixture.Result == nil {
			continue
		}
		encoded, err := sonic.MarshalString(fixture.Result)
		if err != nil {
			continue
		}
		builder.WriteString(encoded)
		builder.WriteByte('\n')
	}

	return builder.String()
}

func (c *EvalCase) HistoryText() string {
	var builder strings.Builder
	for _, message := range c.History {
		builder.WriteString(message.Content)
		builder.WriteByte('\n')
	}

	return builder.String()
}

func (c *EvalCase) PageText() string {
	if c.PageContext == nil {
		return ""
	}
	encoded, err := sonic.MarshalString(c.PageContext)
	if err != nil {
		return ""
	}

	return encoded
}

func (c *EvalCase) GetID() pulid.ID { return c.ID }

func (c *EvalCase) GetCreatedAt() int64 { return c.CreatedAt }

func (c *EvalCase) GetOrganizationID() pulid.ID { return c.OrganizationID }

func (c *EvalCase) GetBusinessUnitID() pulid.ID { return c.BusinessUnitID }

func (c *EvalCase) GetTableName() string { return "agent_eval_cases" }

func (c *EvalCase) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "aec",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "title", Type: domaintypes.FieldTypeText},
			{Name: "input", Type: domaintypes.FieldTypeText},
			{Name: "status", Type: domaintypes.FieldTypeEnum},
			{Name: "source", Type: domaintypes.FieldTypeEnum},
		},
	}
}

func (c *EvalCase) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if c.ID.IsNil() {
			c.ID = pulid.MustNew("aec_")
		}
		if c.Status == "" {
			c.Status = CaseStatusCandidate
		}
		if c.History == nil {
			c.History = []HistoryMessage{}
		}
		if c.Mentions == nil {
			c.Mentions = []agent.EntityRef{}
		}
		if c.HeldTools == nil {
			c.HeldTools = []string{}
		}
		if c.ToolFixtures == nil {
			c.ToolFixtures = []ToolFixture{}
		}
		c.Expected.Normalize()
		c.CreatedAt = now
		c.UpdatedAt = now
	case *bun.UpdateQuery:
		c.UpdatedAt = now
	}

	return nil
}

func FingerprintOf(
	definition *agentdefinition.Definition,
	promptVersion string,
) *agent.Fingerprint {
	if definition == nil {
		return nil
	}

	tools := slices.Clone(definition.EffectiveToolNames())
	slices.Sort(tools)

	return &agent.Fingerprint{
		DefinitionVersion: definition.Version,
		PromptVersion:     promptVersion,
		ProviderID:        definition.PreferredProviderID,
		InstructionsHash:  hashutils.SHA256Hex(definition.Instructions),
		Tools:             tools,
	}
}

func isNilID(id *pulid.ID) bool {
	return id == nil || id.IsNil()
}
