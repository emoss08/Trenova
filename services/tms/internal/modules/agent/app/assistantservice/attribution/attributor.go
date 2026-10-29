package attribution

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"go.uber.org/zap"
)

// AgentNames reads the names of the agents with the given ids, in one query.
// A name that cannot be read is left out: the record still stands under its
// id, and a thread is not refused over a label.
func (s *Attributor) AgentNames(
	ctx context.Context,
	tenant pagination.TenantInfo,
	ids []pulid.ID,
) map[pulid.ID]string {
	definitions := s.agentDefinitions(ctx, tenant, ids)
	names := make(map[pulid.ID]string, len(definitions))
	for _, definition := range definitions {
		names[definition.ID] = definition.Name
	}

	return names
}

// AgentIdentities reads the name and mark of the agents with the given ids,
// in one query. The icon is the agent's own or its starter's, empty when it
// has neither, so a reader draws its initials; the accent is always one.
func (s *Attributor) AgentIdentities(
	ctx context.Context,
	tenant pagination.TenantInfo,
	ids []pulid.ID,
) map[pulid.ID]AgentIdentity {
	definitions := s.agentDefinitions(ctx, tenant, ids)
	identities := make(map[pulid.ID]AgentIdentity, len(definitions))
	for _, definition := range definitions {
		identities[definition.ID] = AgentIdentity{
			Name:   definition.Name,
			Icon:   definition.ChosenIcon(),
			Accent: definition.ResolvedAccent(),
		}
	}

	return identities
}

// agentDefinitions reads the agents with the given ids in the tenant, in one
// query. None are returned when they cannot be read: what is being served
// still stands without them.
func (s *Attributor) agentDefinitions(
	ctx context.Context,
	tenant pagination.TenantInfo,
	ids []pulid.ID,
) []*agentdefinition.Definition {
	if s.Definitions == nil || len(ids) == 0 {
		return nil
	}

	definitions, err := s.Definitions.ListByIDs(
		ctx,
		agentdefinitionports.ListAgentDefinitionsByIDsRequest{
			IDs:        ids,
			TenantInfo: tenant,
		},
	)
	if err != nil {
		s.Logger.Warn("could not read the agents in a conversation", zap.Error(err))

		return nil
	}

	return definitions
}

// ProposersOf reads which agent raised each run's proposals: the
// conversation's own agent, or one it handed a task to. Two queries whatever
// the number of runs.
func (s *Attributor) ProposersOf(
	ctx context.Context,
	tenant pagination.TenantInfo,
	runIDs []pulid.ID,
) map[pulid.ID]proposer {
	proposers := make(map[pulid.ID]proposer, len(runIDs))
	if s.Runs == nil || len(runIDs) == 0 {
		return proposers
	}

	runs, err := s.Runs.ListByIDs(ctx, &agentruntimeports.ListAgentRunsByIDsRequest{
		IDs:        sliceutils.Dedupe(runIDs),
		TenantInfo: tenant,
	})
	if err != nil {
		s.Logger.Warn("could not read which agents raised a conversation's proposals",
			zap.Error(err))

		return proposers
	}

	definitionIDs := make([]pulid.ID, 0, len(runs))
	for _, run := range runs {
		if run != nil && run.AgentDefinitionID.IsNotNil() {
			definitionIDs = append(definitionIDs, run.AgentDefinitionID)
		}
	}
	names := s.AgentNames(ctx, tenant, sliceutils.Dedupe(definitionIDs))

	for _, run := range runs {
		if run == nil || run.AgentDefinitionID.IsNil() {
			continue
		}
		proposers[run.ID] = proposer{
			Id:   run.AgentDefinitionID,
			Name: names[run.AgentDefinitionID],
		}
	}

	return proposers
}

// NameDelegatedSteps names the agent behind each step another agent took on
// a task this conversation's agent handed it, and gives it the agent's mark,
// as the thread is served. A delegate_task result's account is given the same
// mark, so the hand-off and its steps are drawn alike; an agent that no
// longer exists keeps the mark the turn recorded. One query for the whole
// page.
func (s *Attributor) NameDelegatedSteps(
	ctx context.Context,
	tenant pagination.TenantInfo,
	messages []conversation.Message,
) {
	ids := make([]pulid.ID, 0, 2)
	for idx := range messages {
		if id := messages[idx].AgentDefinitionID; id.IsNotNil() {
			ids = append(ids, id)
		}
		if report := messages[idx].DelegateReport; report != nil && report.AgentID.IsNotNil() {
			ids = append(ids, report.AgentID)
		}
	}
	if len(ids) == 0 {
		return
	}

	identities := s.AgentIdentities(ctx, tenant, sliceutils.Dedupe(ids))
	for idx := range messages {
		message := &messages[idx]
		if message.AgentDefinitionID.IsNotNil() {
			identity := identities[message.AgentDefinitionID]
			message.AgentName = identity.Name
			message.AgentIcon = identity.Icon
			message.AgentAccent = identity.Accent
		}
		if report := message.DelegateReport; report != nil {
			if identity, ok := identities[report.AgentID]; ok {
				report.Icon = identity.Icon
				report.Accent = identity.Accent
			}
		}
	}
}

// PersonName is a person's name for a sentence such as "Jordan Pike turned
// off this agent", empty when there is nobody to name or no way to look.
func (s *Attributor) PersonName(
	ctx context.Context,
	id *pulid.ID,
	tenant pagination.TenantInfo,
) string {
	if id == nil || id.IsNil() || s.Users == nil {
		return ""
	}
	user, err := s.Users.GetByID(ctx, tenantports.GetUserByIDRequest{
		TenantInfo:   tenant,
		LookupUserID: *id,
	})
	if err != nil || user == nil {
		return ""
	}

	return user.Name
}

type Attributor struct {
	Logger      *zap.Logger
	Definitions agentdefinitionports.AgentDefinitionRepository
	Runs        agentruntimeports.AgentRunRepository
	Users       tenantports.UserRepository
}
