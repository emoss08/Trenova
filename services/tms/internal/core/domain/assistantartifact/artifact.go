package assistantartifact

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

const (
	maxTitleChars = 200
	// MaxPayloadBytes bounds what one artifact stores. A preview is already
	// capped by the tool that made it; the bound is what keeps a runaway
	// payload from turning a conversation into a warehouse.
	MaxPayloadBytes = 256 * 1024
)

// Artifact is what a turn produced besides words: the rows a report preview
// returned, a run to download, the email an agent wants to send, the plan it
// wants to carry out, the record it looked up. The transcript refers to it;
// the Desk renders it beside the conversation.
//
// A draft or a plan is a view over the proposal or plan that carries the
// decision. The artifact holds what to show; approving, rejecting and
// changing happen on the proposal, so there is one place a decision lives.
type Artifact struct {
	bun.BaseModel `bun:"table:assistant_artifacts,alias:aart" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`

	ThreadID   pulid.ID `json:"threadId"   bun:"thread_id,type:VARCHAR(100),notnull"`
	MessageID  pulid.ID `json:"messageId"  bun:"message_id,type:VARCHAR(100),nullzero"`
	RunID      pulid.ID `json:"runId"      bun:"run_id,type:VARCHAR(100),nullzero"`
	ProposalID pulid.ID `json:"proposalId" bun:"proposal_id,type:VARCHAR(100),nullzero"`
	PlanID     pulid.ID `json:"planId"     bun:"plan_id,type:VARCHAR(100),nullzero"`

	Kind   Kind   `json:"kind"   bun:"kind,type:VARCHAR(40),notnull"`
	Status Status `json:"status" bun:"status,type:VARCHAR(20),notnull,default:'Ready'"`
	Title  string `json:"title"  bun:"title,type:VARCHAR(200),notnull"`
	// Payload is kind-specific and bounded; each kind's shape is declared
	// with the mapper that produces it.
	Payload map[string]any `json:"payload" bun:"payload,type:JSONB,notnull,default:'{}'"`
	// SourceToolCallID ties the artifact to the tool call that produced it,
	// so a retried turn updates the artifact rather than adding a second.
	SourceToolCallID string `json:"sourceToolCallId" bun:"source_tool_call_id,type:VARCHAR(200),notnull,default:''"`
	Pinned           bool   `json:"pinned"           bun:"pinned,type:BOOLEAN,notnull,default:false"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT,notnull"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit *tenant.BusinessUnit `json:"-" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"-" bun:"rel:belongs-to,join:organization_id=id"`
}

func (a *Artifact) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if a.ID.IsNil() {
			a.ID = pulid.MustNew("art_")
		}
		if a.Status == "" {
			a.Status = StatusReady
		}
		if a.Payload == nil {
			a.Payload = map[string]any{}
		}
		a.CreatedAt = now
		a.UpdatedAt = now
	case *bun.UpdateQuery:
		a.UpdatedAt = now
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
		multiErr.Add("proposalId", errortypes.ErrRequired, "An email draft is a view over a proposal")
	}
	if a.Kind == KindPlan && a.PlanID.IsNil() {
		multiErr.Add("planId", errortypes.ErrRequired, "A plan artifact is a view over a plan")
	}
}
