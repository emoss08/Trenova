package assistanttest

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/proposalrecorder"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentproposalports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttrustports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"go.uber.org/zap"
)

func Recorder(runs *StubRunRepo, proposals *StubProposalRepo) *proposalrecorder.Service {
	return RecorderOn(runs, proposals)
}

// RecorderOn is a recorder writing to the stores given, with nobody told.
func RecorderOn(
	runs agentruntimeports.AgentRunRepository,
	proposals agentproposalports.AgentProposalRepository,
) *proposalrecorder.Service {
	return proposalrecorder.New(proposalrecorder.Params{
		Logger:     zap.NewNop(),
		DB:         dbtest.NopConnection{},
		Runs:       runs,
		Proposals:  proposals,
		Trust:      quietTrust{},
		Notifier:   quietNotifier{},
		Plans:      echoedPlans{},
		Activity:   QuietActivity{},
		Watchtower: quietWatchtower{},
	})
}

type quietTrust struct {
	agenttrustports.AgentTrustService
}

func (quietTrust) RecordExecutionFailure(context.Context, *agent.AgentProposal) error {
	return nil
}

type quietNotifier struct {
	agentproposalports.AgentProposalNotifier
}

func (quietNotifier) NotifyPending(
	context.Context,
	agentproposalports.PendingProposalsNotice,
) error {
	return nil
}

type echoedPlans struct{}

func (echoedPlans) Create(
	_ context.Context,
	plan *agentplan.AgentPlan,
) (*agentplan.AgentPlan, error) {
	return plan, nil
}

type quietWatchtower struct {
	watchtowerports.WatchtowerProjector
}

func (quietWatchtower) Upsert(context.Context, watchtowerports.WatchtowerItemInput) {}
