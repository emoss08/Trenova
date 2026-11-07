package agentwaitcontract

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentwait"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
)

const WorkflowName = "AgentWaitWorkflow"

// MetSignal tells a wait that what it waits for has happened.
const MetSignal = "agent-wait-met"

// RescheduleSignal tells a wait that when it comes due moved, as a
// driver's projected drive time does with each clock read.
const RescheduleSignal = "agent-wait-reschedule"

// ErrConversationBusy is a wait that ended while its conversation was
// answering something else.
var ErrConversationBusy = errors.New("the conversation is answering something else")

func WorkflowIDFor(waitID pulid.ID) string { return "agent-wait:" + waitID.String() }

type Payload struct {
	temporaltype.BasePayload

	WaitID pulid.ID `json:"waitId"`
}

func (p *Payload) Tenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: p.OrganizationID, BuID: p.BusinessUnitID}
}

// Met is what ended a wait, as the event or poll that ended it said. Key
// names what the notice is about, so a settling wait keeps the latest notice
// per record rather than every edit.
type Met struct {
	Detail string `json:"detail"`
	At     int64  `json:"at"`
	Key    string `json:"key,omitempty"`
}

// Schedule is when a wait next needs looking at: when it comes due, and when
// it runs out. Open is false for one already closed.
//
// SettleSeconds and SettleMaxSeconds, for a kind that settles, are how long
// a met wait gathers further notices before it picks the work up: until it
// has been quiet that long, and never longer than the limit.
type Schedule struct {
	Open             bool   `json:"open"`
	DueAt            *int64 `json:"dueAt,omitempty"`
	ExpiresAt        int64  `json:"expiresAt"`
	SettleSeconds    int64  `json:"settleSeconds,omitempty"`
	SettleMaxSeconds int64  `json:"settleMaxSeconds,omitempty"`
}

// Check is a timed wait read again when it came due: met, due again later
// because the appointment or the free time moved, or closed meanwhile.
type Check struct {
	Met    bool   `json:"met"`
	Detail string `json:"detail,omitempty"`
	DueAt  *int64 `json:"dueAt,omitempty"`
	Closed bool   `json:"closed,omitempty"`
}

// FinishInput closes a wait. FirstMetAt is when the first notice of a
// settled wait arrived: what the conversation may already have taken in.
type FinishInput struct {
	Payload    *Payload         `json:"payload"`
	Status     agentwait.Status `json:"status"`
	Outcome    string           `json:"outcome"`
	FirstMetAt int64            `json:"firstMetAt,omitempty"`
}

// Worker is what the workflow asks of the waits.
type Worker interface {
	Schedule(ctx context.Context, Tenant pagination.TenantInfo, id pulid.ID) (*Schedule, error)
	Check(ctx context.Context, Tenant pagination.TenantInfo, id pulid.ID, now int64) (*Check, error)
	Finish(ctx context.Context, in *FinishInput) error
	// ReconcileOverdue closes and picks up every open wait past its expiry
	// whose workflow did not, and says how many.
	ReconcileOverdue(ctx context.Context) (int, error)
}
