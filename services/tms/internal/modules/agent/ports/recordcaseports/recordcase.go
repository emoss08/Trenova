package recordcaseports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdecision"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
	"github.com/emoss08/trenova/internal/modules/agent/domain/recordcase"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type CaseRequest struct {
	TenantInfo pagination.TenantInfo
	Kind       recordkind.Kind
	ID         pulid.ID
	Timezone   string
}

type ChooseOptionRequest struct {
	TenantInfo pagination.TenantInfo
	Kind       recordkind.Kind
	ID         pulid.ID
	OptionID   pulid.ID
}

type AcceptRiskRequest struct {
	TenantInfo pagination.TenantInfo
	Kind       recordkind.Kind
	ID         pulid.ID
	Reason     string
}

type PutAwayRequest struct {
	TenantInfo pagination.TenantInfo
	Kind       recordkind.Kind
	ID         pulid.ID
	// Until is when the case comes back if nothing new happens first.
	Until int64
}

type AssignCaseRequest struct {
	TenantInfo pagination.TenantInfo
	Kind       recordkind.Kind
	ID         pulid.ID
	// OwnerID is who the case goes to; the actor themselves claims it.
	OwnerID pulid.ID
}

type OfferOptionsRequest struct {
	TenantInfo        pagination.TenantInfo
	Kind              recordkind.Kind
	ID                pulid.ID
	AgentDefinitionID pulid.ID
	RunID             pulid.ID
	Options           []*recordcase.Option
}

type RecordCaseService interface {
	Case(
		ctx context.Context,
		req *CaseRequest,
		actor *permissionports.RequestActor,
	) (*recordcase.Case, error)
	ChooseOption(
		ctx context.Context,
		req *ChooseOptionRequest,
		actor *permissionports.RequestActor,
	) error
	AcceptRisk(
		ctx context.Context,
		req *AcceptRiskRequest,
		actor *permissionports.RequestActor,
	) (int, error)
	// PutAway snoozes what is still open about the record until a time, so
	// the case leaves Needs you. It refuses while a change waits to be
	// decided. Returns how many items it put away.
	PutAway(
		ctx context.Context,
		req *PutAwayRequest,
		actor *permissionports.RequestActor,
	) (int, error)
	// BringBack wakes what PutAway snoozed. Returns how many items it woke.
	BringBack(
		ctx context.Context,
		req *CaseRequest,
		actor *permissionports.RequestActor,
	) (int, error)
	// AssignCase gives the case to an owner: the actor themselves when no
	// owner is named, which claims it.
	AssignCase(
		ctx context.Context,
		req *AssignCaseRequest,
		actor *permissionports.RequestActor,
	) error
	// ReleaseCase leaves the case unowned.
	ReleaseCase(
		ctx context.Context,
		req *CaseRequest,
		actor *permissionports.RequestActor,
	) error
}

type OptionOfferer interface {
	OfferOptions(ctx context.Context, req *OfferOptionsRequest) ([]*recordcase.Option, error)
}

type RecordRequest struct {
	TenantInfo pagination.TenantInfo
	RecordID   pulid.ID
}

type ReplaceOptionsRequest struct {
	TenantInfo pagination.TenantInfo
	RecordID   pulid.ID
	Options    []*recordcase.Option
}

type ChooseRequest struct {
	TenantInfo pagination.TenantInfo
	RecordID   pulid.ID
	OptionID   pulid.ID
}

type SetDraftRequest struct {
	TenantInfo  pagination.TenantInfo
	OptionID    pulid.ID
	RunID       pulid.ID
	RequestedAt int64
}

type DecisionsRequest struct {
	TenantInfo pagination.TenantInfo
	RecordID   pulid.ID
	RunID      pulid.ID
	// HandedRuns are the runs of the agents the record's conditions were
	// handed to. What they propose is the record's to decide whatever record
	// the change is made on: a detention desk put on a load's detention
	// proposes on the detention occurrence.
	HandedRuns []pulid.ID
	Since      int64
}

type Decisions struct {
	Plans     []*agentplan.AgentPlan
	Proposals []*agent.AgentProposal
	Decided   map[pulid.ID]*agentdecision.AgentDecision
}

type ReadingRequest struct {
	TenantInfo  pagination.TenantInfo
	RecordID    pulid.ID
	Fingerprint string
}

type ClaimReadingRequest struct {
	TenantInfo  pagination.TenantInfo
	RecordKind  recordkind.Kind
	RecordID    pulid.ID
	Fingerprint string
	Now         int64
	StaleBefore int64
	// ForgetBefore is when a reading of the record from other signals was
	// last asked for to be worth keeping; older ones are deleted with the claim.
	ForgetBefore int64
}

type RecordCaseRepository interface {
	Options(ctx context.Context, req RecordRequest) ([]*recordcase.Option, error)
	GetOption(
		ctx context.Context,
		tenant pagination.TenantInfo,
		id pulid.ID,
	) (*recordcase.Option, error)
	ReplaceOptions(ctx context.Context, req *ReplaceOptionsRequest) ([]*recordcase.Option, error)
	Choose(ctx context.Context, req ChooseRequest) (*recordcase.Option, error)
	SetDraft(ctx context.Context, req SetDraftRequest) error
	Reading(ctx context.Context, req ReadingRequest) (*recordcase.Reading, error)
	ClaimReading(ctx context.Context, req *ClaimReadingRequest) (bool, error)
	SaveReading(ctx context.Context, reading *recordcase.Reading) error
	Decisions(ctx context.Context, req *DecisionsRequest) (*Decisions, error)
}

type WriteReadingRequest struct {
	TenantInfo  pagination.TenantInfo
	RecordKind  recordkind.Kind
	RecordID    pulid.ID
	Fingerprint string
	Timezone    string
	UserID      pulid.ID
}

type ReadingStarter interface {
	StartReading(ctx context.Context, req *WriteReadingRequest) error
}

type ReadingWriter interface {
	WriteReading(ctx context.Context, req *WriteReadingRequest) error
}
