package assistantservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"go.uber.org/zap"
)

// proposer is the agent behind a proposal or a plan.
type proposer struct {
	id   pulid.ID
	name string
}

// agentNames reads the names of the agents with the given ids, in one query.
// A name that cannot be read is left out: the record still stands under its
// id, and a thread is not refused over a label.
func (s *Service) agentNames(
	ctx context.Context,
	tenant pagination.TenantInfo,
	ids []pulid.ID,
) map[pulid.ID]string {
	names := make(map[pulid.ID]string, len(ids))
	if s.definitions == nil || len(ids) == 0 {
		return names
	}

	definitions, err := s.definitions.ListByIDs(ctx, repositories.ListAgentDefinitionsByIDsRequest{
		IDs:        ids,
		TenantInfo: tenant,
	})
	if err != nil {
		s.logger.Warn("could not read the names of the agents in a conversation", zap.Error(err))

		return names
	}
	for _, definition := range definitions {
		names[definition.ID] = definition.Name
	}

	return names
}

// proposersOf reads which agent raised each run's proposals: the
// conversation's own agent, or one it handed a task to. Two queries whatever
// the number of runs.
func (s *Service) proposersOf(
	ctx context.Context,
	tenant pagination.TenantInfo,
	runIDs []pulid.ID,
) map[pulid.ID]proposer {
	proposers := make(map[pulid.ID]proposer, len(runIDs))
	if s.runs == nil || len(runIDs) == 0 {
		return proposers
	}

	runs, err := s.runs.ListByIDs(ctx, repositories.ListAgentRunsByIDsRequest{
		IDs:        sliceutils.Dedupe(runIDs),
		TenantInfo: tenant,
	})
	if err != nil {
		s.logger.Warn("could not read which agents raised a conversation's proposals",
			zap.Error(err))

		return proposers
	}

	definitionIDs := make([]pulid.ID, 0, len(runs))
	for _, run := range runs {
		if run != nil && run.AgentDefinitionID.IsNotNil() {
			definitionIDs = append(definitionIDs, run.AgentDefinitionID)
		}
	}
	names := s.agentNames(ctx, tenant, sliceutils.Dedupe(definitionIDs))

	for _, run := range runs {
		if run == nil || run.AgentDefinitionID.IsNil() {
			continue
		}
		proposers[run.ID] = proposer{
			id:   run.AgentDefinitionID,
			name: names[run.AgentDefinitionID],
		}
	}

	return proposers
}

// nameDelegatedSteps names the agent behind each step another agent took on
// a task this conversation's agent handed it, as the thread is served.
func (s *Service) nameDelegatedSteps(
	ctx context.Context,
	tenant pagination.TenantInfo,
	messages []conversation.Message,
) {
	ids := make([]pulid.ID, 0, 2)
	for idx := range messages {
		if id := messages[idx].AgentDefinitionID; id.IsNotNil() {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return
	}

	names := s.agentNames(ctx, tenant, sliceutils.Dedupe(ids))
	for idx := range messages {
		message := &messages[idx]
		if message.AgentDefinitionID.IsNotNil() {
			message.AgentName = names[message.AgentDefinitionID]
		}
	}
}
