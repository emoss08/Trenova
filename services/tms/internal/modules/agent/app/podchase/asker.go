package podchase

import (
	"context"
	"errors"

	"go.uber.org/fx"

	"github.com/emoss08/trenova/internal/modules/agent/app/codedrequest"
	"github.com/emoss08/trenova/internal/modules/agent/app/requestemail"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/podchaseports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const promptVersion = "routine.pod-chase.v1"

// ErrNobodyToAsk is a shipment whose proof of delivery nobody can be asked
// for: no driver Dash, text or email reaches and no carrier with an address.
// No model can ask either, so nothing is run; a person uploads the paperwork
// or adds a way to reach them.
var ErrNobodyToAsk = errors.New("nobody can be asked for the proof of delivery")

type AskerParams struct {
	fx.In

	Filer    *codedrequest.Filer
	Requests *requestemail.Sender
}

// Asker files the billing agent's own request for a delivered shipment's
// signed proof of delivery, when the event that announces a missing one
// reaches an agent that takes it. Who is asked and in what words is the
// same plan a person sends from the Watchtower (planAsk); here it is put
// through the agent runtime's policy as that agent's proposal, so its
// tier, holds and shadow apply. No model reads the record.
type Asker struct {
	filer    *codedrequest.Filer
	requests *requestemail.Sender
}

func NewAsker(p AskerParams) *Asker { //nolint:gocritic // fx requires an fx.In parameter struct by value
	return &Asker{filer: p.Filer, requests: p.Requests}
}

// Chasing is what requests are filed with for one agent, and what the
// organization can send with.
type Chasing struct {
	*codedrequest.Filing

	can reach
}

// For loads the agent and what its requests need. Nil when the agent is
// gone or off, so nothing is asked for it.
func (a *Asker) For(
	ctx context.Context,
	tenant pagination.TenantInfo,
	definitionID pulid.ID,
) (*Chasing, error) {
	filing, err := a.filer.For(ctx, tenant, definitionID)
	if err != nil || filing == nil {
		return nil, err
	}

	return &Chasing{
		Filing: filing,
		can:    reach{email: a.requests.CanSend(ctx, tenant), texts: filing.Texts},
	}, nil
}

// Ask files one shipment's request. ErrNobodyToAsk when nobody can be
// reached, and nothing is run.
func (a *Asker) Ask(
	ctx context.Context,
	c *Chasing,
	fact *podchaseports.Facts,
) (*agent.AgentRun, error) {
	planned := planAsk(fact, c.can, c.Timezone)
	if planned == nil {
		return nil, ErrNobodyToAsk
	}
	call := planned.call(fact, c.Timezone)

	return a.filer.File(ctx, c.Filing, &codedrequest.Request{
		AgentType:     agent.TypeBillingException,
		SubjectType:   agent.SubjectShipment,
		SubjectID:     fact.ShipmentID,
		PromptVersion: promptVersion,
		Summary:       "Asked for the signed proof of delivery",
		Trigger:       agent.RunTriggerEvent,
		Tool:          call.tool,
		Arguments:     call.arguments,
		Rationale:     call.rationale,
	})
}
