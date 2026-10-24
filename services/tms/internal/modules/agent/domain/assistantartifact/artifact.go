package assistantartifact

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

const (
	maxTitleChars = 200
	// MaxPayloadBytes bounds what one artifact stores. A preview is already
	// capped by the tool that made it; the bound is what keeps a runaway
	// payload from turning a conversation into a warehouse.
	MaxPayloadBytes = 256 * 1024
)

// SummaryPayloadKeys are the payload keys a summary keeps: the tool that made
// the artifact, the entity it shows, the page it points at and the document
// it was read from, which are what a list needs to name and draw it without
// its contents, and what lets a file the person attached open what was read
// off it.
func SummaryPayloadKeys() []string {
	return []string{"tool", "entity", "path", "documentId"}
}

// Artifact is what a turn produced besides words: the rows a report preview
// returned, a run to download, the email an agent wants to send, the plan it
// wants to carry out, the record it looked up. The transcript refers to it;
// the Desk renders it beside the conversation.
//
// A draft or a plan is a view over the proposal or plan that carries the
// decision. The artifact holds what to show; approving, rejecting and
// changing happen on the proposal, so there is one place a decision lives.
type Artifact struct {
	keel.Table `keel:"assistant_artifacts"`

	ID             pulid.ID `json:"id" keel:"id,pk,autoid:art_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	ThreadID   pulid.ID `json:"threadId" keel:"thread_id"`
	MessageID  pulid.ID `json:"messageId" keel:"message_id,nullzero"`
	RunID      pulid.ID `json:"runId" keel:"run_id,nullzero"`
	ProposalID pulid.ID `json:"proposalId" keel:"proposal_id,nullzero"`
	PlanID     pulid.ID `json:"planId" keel:"plan_id,nullzero"`

	Kind   Kind   `json:"kind" keel:"kind"`
	Status Status `json:"status" keel:"status"`
	Title  string `json:"title" keel:"title"`
	// Payload is kind-specific and bounded; each kind's shape is declared
	// with the mapper that produces it.
	Payload map[string]any `json:"payload" keel:"payload,jsonb"`
	// SourceToolCallID ties the artifact to the tool call that produced it,
	// so a retried turn updates the artifact rather than adding a second.
	SourceToolCallID string `json:"sourceToolCallId" keel:"source_tool_call_id"`
	Pinned           bool   `json:"pinned" keel:"pinned"`

	LineageKey string   `json:"lineageKey" keel:"lineage_key"`
	LineageID  pulid.ID `json:"lineageId" keel:"lineage_id,nullzero"`
	LineageSeq int      `json:"lineageSeq" keel:"lineage_seq"`
	// Slug names the lineage in a link, the same for every version of it and
	// unique among the conversation's artifacts.
	Slug string `json:"slug" keel:"slug"`

	Version   int64 `json:"version" keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit `json:"-"`
	Organization *tenant.Organization `json:"-"`
}

func (a *Artifact) BeforeCreate(_ context.Context) error {
	if a.Status == "" {
		a.Status = StatusReady
	}
	if a.Payload == nil {
		a.Payload = map[string]any{}
	}
	if a.LineageSeq == 0 {
		a.LineageSeq = 1
	}

	return nil
}

func (a *Artifact) GetID() pulid.ID { return a.ID }

func (a *Artifact) GetTableName() string { return "assistant_artifacts" }

func (a *Artifact) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(a,
		validation.Field(&a.OrganizationID, validation.Required.Error("Organization is required")),
		validation.Field(&a.BusinessUnitID, validation.Required.Error("Business unit is required")),
		validation.Field(&a.ThreadID, validation.Required.Error("Thread is required")),
		validation.Field(&a.Kind,
			validation.Required.Error("Kind is required"),
			domainvalidation.ValidEnum[Kind]("Kind is invalid"),
		),
		validation.Field(&a.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[Status]("Status is invalid"),
		),
		validation.Field(&a.Title,
			validation.Required.Error("Title is required"),
			validation.Length(1, maxTitleChars).Error("Title cannot be longer than 200 characters"),
		),
	))

	if a.Kind == KindEmailDraft && a.ProposalID.IsNil() {
		multiErr.Add(
			"proposalId",
			errortypes.ErrRequired,
			"An email draft is a view over a proposal",
		)
	}
	if a.Kind == KindDecisionRequest && a.ProposalID.IsNil() {
		multiErr.Add(
			"proposalId",
			errortypes.ErrRequired,
			"A decision request is a view over a proposal",
		)
	}
	if a.Kind == KindPlan && a.PlanID.IsNil() {
		multiErr.Add("planId", errortypes.ErrRequired, "A plan artifact is a view over a plan")
	}
}
