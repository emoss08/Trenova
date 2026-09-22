package agentscorecardservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime/agentruntimetest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const fixedNow = 1_700_000_000

type stubScorecards struct {
	repositories.AgentScorecardRepository

	asked  repositories.ScorecardRequest
	totals *agent.ScorecardTotals
}

func (s *stubScorecards) Aggregate(
	_ context.Context,
	req repositories.ScorecardRequest,
) (*agent.ScorecardTotals, error) {
	s.asked = req
	if s.totals == nil {
		return &agent.ScorecardTotals{CostUSD: decimal.Zero}, nil
	}

	return s.totals, nil
}

type stubTrust struct {
	repositories.AgentToolTrustRepository

	rows []*agent.ToolTrust
}

func (s *stubTrust) ListByDefinition(
	_ context.Context,
	_ repositories.ListToolTrustRequest,
) ([]*agent.ToolTrust, error) {
	return s.rows, nil
}

type stubDefinitions struct {
	repositories.AgentDefinitionRepository

	missing bool
}

func (s *stubDefinitions) GetByID(
	_ context.Context,
	req repositories.GetAgentDefinitionByIDRequest,
) (*agentdefinition.Definition, error) {
	if s.missing {
		return nil, errortypes.NewNotFoundError("Agent not found")
	}

	return &agentdefinition.Definition{ID: req.ID}, nil
}

func newService(
	perms *agentruntimetest.StubPermissions,
	cards *stubScorecards,
	trust *stubTrust,
	defs *stubDefinitions,
) *Service {
	svc := New(Params{
		Logger:      zap.NewNop(),
		Scorecards:  cards,
		Trust:       trust,
		Definitions: defs,
		Permissions: perms,
	})
	svc.now = func() int64 { return fixedNow }

	return svc
}

func actor() *services.RequestActor {
	return &services.RequestActor{
		PrincipalType:  services.PrincipalTypeUser,
		PrincipalID:    pulid.MustNew("usr_"),
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}
}

func request(window agent.ScorecardWindow) services.AgentScorecardRequest {
	return services.AgentScorecardRequest{
		TenantInfo:        pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		AgentDefinitionID: pulid.MustNew("agd_"),
		Window:            window,
	}
}

// A scorecard is a summary of rows the reader could already list one by one,
// so it needs what reading the agent needs — but it does need that, because
// it says what an agent has been writing to the organization's records.
func TestGet_RefusesAReaderWhoCannotReadTheAgent(t *testing.T) {
	t.Parallel()

	perms := &agentruntimetest.StubPermissions{Denied: map[string]bool{"agent_definition:read": true}}
	cards := &stubScorecards{}
	svc := newService(perms, cards, &stubTrust{}, &stubDefinitions{})

	_, err := svc.Get(t.Context(), request(agent.ScorecardWindow30d), actor())

	require.Error(t, err)
	assert.Zero(t, cards.asked.AgentDefinitionID, "counted rows for a reader who was refused")
}

/*
An id from another organization has to be not found rather than counted. The
aggregate is tenant-scoped, so it would come back all zeros — which reads as a
real agent that has never done anything, and tells the asker that an agent
with that id exists.
*/
func TestGet_AnAgentFromAnotherOrganizationIsNotFound(t *testing.T) {
	t.Parallel()

	cards := &stubScorecards{}
	svc := newService(
		&agentruntimetest.StubPermissions{},
		cards,
		&stubTrust{},
		&stubDefinitions{missing: true},
	)

	_, err := svc.Get(t.Context(), request(agent.ScorecardWindow30d), actor())

	require.Error(t, err)
	assert.Zero(t, cards.asked.AgentDefinitionID, "counted rows before the agent was found")
}

func TestGet_CountsFromTheStartOfTheWindow(t *testing.T) {
	t.Parallel()

	cards := &stubScorecards{}
	svc := newService(&agentruntimetest.StubPermissions{}, cards, &stubTrust{}, &stubDefinitions{})

	result, err := svc.Get(t.Context(), request(agent.ScorecardWindow7d), actor())

	require.NoError(t, err)
	assert.Equal(t, int64(fixedNow-7*secondsPerDay), cards.asked.Since)
	assert.Equal(t, int64(fixedNow-7*secondsPerDay), result.Scorecard.Since)
}

// A window nobody recognises covers a month rather than no time at all: a
// scorecard over zero seconds reads as an agent that has done nothing.
func TestGet_AnUnknownWindowFallsBackToAMonth(t *testing.T) {
	t.Parallel()

	cards := &stubScorecards{}
	svc := newService(&agentruntimetest.StubPermissions{}, cards, &stubTrust{}, &stubDefinitions{})

	result, err := svc.Get(t.Context(), request(agent.ScorecardWindow("Forever")), actor())

	require.NoError(t, err)
	assert.Equal(t, agent.ScorecardWindow30d, result.Scorecard.Window)
	assert.Equal(t, int64(fixedNow-30*secondsPerDay), cards.asked.Since)
}

/*
The ladder is not windowed and must not be. A tier is earned over the agent's
whole life and taken back the same way, so a ladder filtered to the last seven
days would describe a different ladder from the one deciding, right now, what
runs without anybody being asked.
*/
func TestGet_TheLadderIgnoresTheWindow(t *testing.T) {
	t.Parallel()

	trust := &stubTrust{rows: []*agent.ToolTrust{{ToolName: "assign_move", Streak: 9}}}
	svc := newService(
		&agentruntimetest.StubPermissions{},
		&stubScorecards{},
		trust,
		&stubDefinitions{},
	)

	short, err := svc.Get(t.Context(), request(agent.ScorecardWindow7d), actor())
	require.NoError(t, err)
	long, err := svc.Get(t.Context(), request(agent.ScorecardWindow90d), actor())
	require.NoError(t, err)

	assert.Equal(t, short.ToolTrust, long.ToolTrust)
	require.Len(t, short.ToolTrust, 1)
	assert.Equal(t, 9, short.ToolTrust[0].Streak)
}
