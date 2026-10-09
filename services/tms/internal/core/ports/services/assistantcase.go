package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentwait"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/domain/deskcase"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// CaseThreadRequest names one of the person's conversations.
type CaseThreadRequest struct {
	ThreadID   pulid.ID
	TenantInfo pagination.TenantInfo
	Actor      *RequestActor
}

// BindCaseRequest makes a conversation a case about a shipment, invoice or
// dispute.
type BindCaseRequest struct {
	CaseThreadRequest
	SubjectType agent.SubjectType
	SubjectID   pulid.ID
}

// SnoozeCaseRequest snoozes a case until a time, the shipment's next
// appointment or its ETA. Until is the time for a Time snooze, and the
// fallback for an ETA snooze while the shipment has no estimate.
type SnoozeCaseRequest struct {
	CaseThreadRequest
	Anchor deskcase.SnoozeAnchor
	Until  int64
}

// AwaitCaseReplyRequest parks a case until the record's customer or one of
// its carriers replies.
type AwaitCaseReplyRequest struct {
	CaseThreadRequest
	Party            deskcase.WaitingOn
	PartyID          pulid.ID
	GiveUpAfterHours int
}

// AssistantCaseStates works out where the cases among a page of the
// person's conversations stand, in a fixed number of queries.
type AssistantCaseStates interface {
	Attach(
		ctx context.Context,
		tenant pagination.TenantInfo,
		userID pulid.ID,
		threads []*conversation.Thread,
	) error
}

type AssistantCaseService interface {
	AssistantCaseStates
	Get(ctx context.Context, req *CaseThreadRequest) (*deskcase.View, error)
	Bind(ctx context.Context, req *BindCaseRequest) (*conversation.Thread, error)
	Unbind(ctx context.Context, req *CaseThreadRequest) (*conversation.Thread, error)
	Snooze(ctx context.Context, req *SnoozeCaseRequest) (*conversation.Thread, error)
	Wake(ctx context.Context, req *CaseThreadRequest) (*conversation.Thread, error)
	AwaitReply(ctx context.Context, req *AwaitCaseReplyRequest) (*agentwait.Wait, error)
	Tick(ctx context.Context, req *TickCaseItemRequest) (*deskcase.View, error)
}

// TickCaseItemRequest ticks, or unticks, a step the organization added to
// the checklist of the record a case is about.
type TickCaseItemRequest struct {
	CaseThreadRequest
	ItemKey deskcase.ItemKey
	Ticked  bool
}

// CaseChecklistTemplates is how an organization lays out one kind of case
// checklist: its own template, the default when it saved none, and each
// customer's that replaces it.
type CaseChecklistTemplates struct {
	Kind         deskcase.ChecklistKind        `json:"kind"`
	Organization *deskcase.ChecklistTemplate   `json:"organization"`
	Customers    []*deskcase.ChecklistTemplate `json:"customers"`
	// Locked are the built-in steps whose mode follows a rule set
	// elsewhere; they can be moved but stay required.
	Locked []deskcase.ItemKey `json:"locked"`
}

type ListCaseChecklistTemplatesRequest struct {
	TenantInfo pagination.TenantInfo
	Kind       deskcase.ChecklistKind
}

// SaveCaseChecklistTemplateRequest saves the organization's template of a kind,
// or a customer's. ID and Version name a template being changed; without an
// ID one is made.
type SaveCaseChecklistTemplateRequest struct {
	TenantInfo pagination.TenantInfo
	UserID     pulid.ID
	ID         pulid.ID
	Version    int64
	Kind       deskcase.ChecklistKind
	CustomerID pulid.ID
	Items      deskcase.TemplateItems
}

type DeleteCaseChecklistTemplateRequest struct {
	TenantInfo pagination.TenantInfo
	UserID     pulid.ID
	ID         pulid.ID
}

// CaseChecklistService keeps how case checklists are laid out.
type CaseChecklistService interface {
	List(ctx context.Context, req *ListCaseChecklistTemplatesRequest) (*CaseChecklistTemplates, error)
	Save(ctx context.Context, req *SaveCaseChecklistTemplateRequest) (*deskcase.ChecklistTemplate, error)
	Delete(ctx context.Context, req *DeleteCaseChecklistTemplateRequest) error
}
