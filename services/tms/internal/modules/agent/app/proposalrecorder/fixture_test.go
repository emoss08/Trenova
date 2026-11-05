package proposalrecorder

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrunstep"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentproposalports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttrustports"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
)

type quietNotifier struct {
	agentproposalports.AgentProposalNotifier
}

func (quietNotifier) NotifyPending(
	context.Context,
	agentproposalports.PendingProposalsNotice,
) error {
	return nil
}

type quietTrust struct {
	agenttrustports.AgentTrustService
}

func (quietTrust) RecordExecutionFailure(context.Context, *agent.AgentProposal) error {
	return nil
}

type echoedPlans struct{}

func (echoedPlans) Create(
	_ context.Context,
	plan *agentplan.AgentPlan,
) (*agentplan.AgentPlan, error) {
	return plan, nil
}

type noSteps struct{}

func (noSteps) Get(
	context.Context,
	*agentruntimeports.GetAgentRunStepRequest,
) (*agentrunstep.AgentRunStep, error) {
	return nil, nil
}

func recorderOn(runs RunOpener, proposals ProposalStore) *Service {
	svc := NewWithStores(nil, runs, proposals).
		WithTransactor(dbtest.NopConnection{}).
		WithPlans(echoedPlans{}).
		WithScriptSteps(noSteps{})
	svc.notifier = quietNotifier{}
	svc.trust = quietTrust{}
	svc.activity = agenttest.QuietActivity{}
	svc.watchtower = agenttest.QuietWatchtower{}

	return svc
}
