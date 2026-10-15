package agentsuggestionservice

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentsuggestion"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentsuggestionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingRepository struct {
	inserted []*agentsuggestion.Suggestion
	settled  []agentsuggestionports.SettleAgentSuggestionRow
}

func (r *recordingRepository) Insert(
	_ context.Context,
	entity *agentsuggestion.Suggestion,
) (*agentsuggestion.Suggestion, error) {
	r.inserted = append(r.inserted, entity)

	return entity, nil
}

func (r *recordingRepository) Settle(
	_ context.Context,
	req *agentsuggestionports.SettleAgentSuggestionRow,
) (*agentsuggestion.Suggestion, error) {
	r.settled = append(r.settled, *req)

	return &agentsuggestion.Suggestion{ID: req.ID, Outcome: req.Outcome}, nil
}

type knownAgents struct {
	agentdefinitionports.AgentDefinitionRepository

	ids        []pulid.ID
	restricted []pulid.ID
}

func (k *knownAgents) ListByIDs(
	_ context.Context,
	req agentdefinitionports.ListAgentDefinitionsByIDsRequest,
) ([]*agentdefinition.Definition, error) {
	found := make([]*agentdefinition.Definition, 0, len(req.IDs))
	for _, id := range req.IDs {
		if slices.Contains(k.restricted, id) {
			found = append(found, &agentdefinition.Definition{
				ID:         id,
				AccessMode: agentdefinition.AccessRoles,
			})
			continue
		}
		for _, known := range k.ids {
			if id == known {
				found = append(found, &agentdefinition.Definition{
					ID:         id,
					AccessMode: agentdefinition.AccessEveryone,
				})
			}
		}
	}

	return found, nil
}

func person() *permissionports.RequestActor {
	userID := pulid.MustNew("usr_")

	return &permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeUser,
		PrincipalID:    userID,
		UserID:         userID,
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}
}

func fieldOf(t *testing.T, err error) string {
	t.Helper()

	var validation *errortypes.Error
	require.ErrorAs(t, err, &validation)

	return validation.Field
}

func TestRecordMine_RecordsTheSuggestionAsThePerson(t *testing.T) {
	t.Parallel()

	suggested, current := pulid.MustNew("agdef_"), pulid.MustNew("agdef_")
	repo := &recordingRepository{}
	service := New(Params{Repository: repo, Permissions: &agenttest.StubPermissions{}, Definitions: &knownAgents{ids: []pulid.ID{suggested, current}}})
	actor := person()

	_, err := service.RecordMine(t.Context(), &agentsuggestionports.RecordAgentSuggestionRequest{
		TenantInfo:       actor.TenantInfo(),
		SuggestedAgentID: suggested,
		CurrentAgentID:   current,
		Surface:          agentsuggestion.SurfaceConversation,
		Outcome:          agentsuggestion.OutcomeShown,
	}, actor)

	require.NoError(t, err)
	require.Len(t, repo.inserted, 1)
	assert.Equal(t, &agentsuggestion.Suggestion{
		OrganizationID:   actor.OrganizationID,
		BusinessUnitID:   actor.BusinessUnitID,
		UserID:           actor.UserID,
		SuggestedAgentID: suggested,
		CurrentAgentID:   &current,
		Surface:          agentsuggestion.SurfaceConversation,
		Outcome:          agentsuggestion.OutcomeShown,
	}, repo.inserted[0])
}

func TestRecordMine_RecordsAnAutomaticSwitchWithNoAgentBefore(t *testing.T) {
	t.Parallel()

	suggested := pulid.MustNew("agdef_")
	repo := &recordingRepository{}
	service := New(Params{Repository: repo, Permissions: &agenttest.StubPermissions{}, Definitions: &knownAgents{ids: []pulid.ID{suggested}}})
	actor := person()

	_, err := service.RecordMine(t.Context(), &agentsuggestionports.RecordAgentSuggestionRequest{
		TenantInfo:       actor.TenantInfo(),
		SuggestedAgentID: suggested,
		Surface:          agentsuggestion.SurfaceHome,
		Outcome:          agentsuggestion.OutcomeAutoSwitched,
	}, actor)

	require.NoError(t, err)
	require.Len(t, repo.inserted, 1)
	assert.Nil(t, repo.inserted[0].CurrentAgentID)
}

func TestRecordMine_Refuses(t *testing.T) {
	t.Parallel()

	known := pulid.MustNew("agdef_")
	other := pulid.MustNew("agdef_")
	actor := person()
	tenant := actor.TenantInfo()
	cases := []struct {
		name  string
		req   agentsuggestionports.RecordAgentSuggestionRequest
		field string
	}{
		{
			name: "an outcome a suggestion cannot start with",
			req: agentsuggestionports.RecordAgentSuggestionRequest{
				TenantInfo: tenant, SuggestedAgentID: known,
				Surface: agentsuggestion.SurfaceHome, Outcome: agentsuggestion.OutcomeKept,
			},
			field: "outcome",
		},
		{
			name: "an agent outside the organization",
			req: agentsuggestionports.RecordAgentSuggestionRequest{
				TenantInfo: tenant, SuggestedAgentID: other,
				Surface: agentsuggestion.SurfaceHome, Outcome: agentsuggestion.OutcomeShown,
			},
			field: "suggestedAgentId",
		},
		{
			name: "an agent suggested in place of itself",
			req: agentsuggestionports.RecordAgentSuggestionRequest{
				TenantInfo: tenant, SuggestedAgentID: known, CurrentAgentID: known,
				Surface: agentsuggestion.SurfaceHome, Outcome: agentsuggestion.OutcomeShown,
			},
			field: "suggestedAgentId",
		},
		{
			name: "a surface that does not exist",
			req: agentsuggestionports.RecordAgentSuggestionRequest{
				TenantInfo: tenant, SuggestedAgentID: known,
				Surface: agentsuggestion.Surface("Palette"), Outcome: agentsuggestion.OutcomeShown,
			},
			field: "surface",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			repo := &recordingRepository{}
			service := New(Params{Repository: repo, Permissions: &agenttest.StubPermissions{}, Definitions: &knownAgents{ids: []pulid.ID{known}}})

			_, err := service.RecordMine(t.Context(), &tc.req, actor)

			var multi *errortypes.MultiError
			if errors.As(err, &multi) {
				require.NotEmpty(t, multi.Errors)
				assert.Equal(t, tc.field, multi.Errors[0].Field)
			} else {
				assert.Equal(t, tc.field, fieldOf(t, err))
			}
			assert.Empty(t, repo.inserted)
		})
	}
}

func TestRecordMine_RefusesAnAgentThePersonMayNotUse(t *testing.T) {
	t.Parallel()

	restricted := pulid.MustNew("agdef_")
	actor := person()
	repo := &recordingRepository{}
	service := New(Params{
		Repository:  repo,
		Permissions: &agenttest.StubPermissions{},
		Definitions: &knownAgents{restricted: []pulid.ID{restricted}},
	})

	_, err := service.RecordMine(t.Context(), &agentsuggestionports.RecordAgentSuggestionRequest{
		TenantInfo:       actor.TenantInfo(),
		SuggestedAgentID: restricted,
		Surface:          agentsuggestion.SurfaceHome,
		Outcome:          agentsuggestion.OutcomeShown,
	}, actor)

	assert.Equal(t, "suggestedAgentId", fieldOf(t, err))
	assert.Empty(t, repo.inserted, "only an agent the person may use is ever suggested to them")
}

func TestRecordMine_RecordsOnlyAPersonsSuggestions(t *testing.T) {
	t.Parallel()

	repo := &recordingRepository{}
	service := New(Params{Repository: repo, Permissions: &agenttest.StubPermissions{}, Definitions: &knownAgents{}})
	actor := person()
	actor.PrincipalType = permissionports.PrincipalTypeAPIKey

	_, err := service.RecordMine(t.Context(), &agentsuggestionports.RecordAgentSuggestionRequest{
		TenantInfo:       pagination.TenantInfo{OrgID: actor.OrganizationID, BuID: actor.BusinessUnitID},
		SuggestedAgentID: pulid.MustNew("agdef_"),
		Surface:          agentsuggestion.SurfaceHome,
		Outcome:          agentsuggestion.OutcomeShown,
	}, actor)

	require.Error(t, err)
	assert.Empty(t, repo.inserted)
}

func TestSettleMine_SettlesThePersonsOwnSuggestion(t *testing.T) {
	t.Parallel()

	repo := &recordingRepository{}
	service := New(Params{Repository: repo, Permissions: &agenttest.StubPermissions{}, Definitions: &knownAgents{}})
	actor := person()
	id := pulid.MustNew("agsg_")

	settled, err := service.SettleMine(t.Context(), &agentsuggestionports.SettleAgentSuggestionRequest{
		TenantInfo: actor.TenantInfo(),
		ID:         id,
		Outcome:    agentsuggestion.OutcomeSwitched,
	}, actor)

	require.NoError(t, err)
	assert.Equal(t, agentsuggestion.OutcomeSwitched, settled.Outcome)
	assert.Equal(t, []agentsuggestionports.SettleAgentSuggestionRow{{
		TenantInfo: actor.TenantInfo(),
		ID:         id,
		UserID:     actor.UserID,
		Outcome:    agentsuggestion.OutcomeSwitched,
	}}, repo.settled)
}

func TestSettleMine_RefusesAnOutcomeThatDoesNotSettle(t *testing.T) {
	t.Parallel()

	repo := &recordingRepository{}
	service := New(Params{Repository: repo, Permissions: &agenttest.StubPermissions{}, Definitions: &knownAgents{}})
	actor := person()

	for _, outcome := range []agentsuggestion.Outcome{
		agentsuggestion.OutcomeShown,
		agentsuggestion.OutcomeAutoSwitched,
	} {
		_, err := service.SettleMine(t.Context(), &agentsuggestionports.SettleAgentSuggestionRequest{
			TenantInfo: actor.TenantInfo(),
			ID:         pulid.MustNew("agsg_"),
			Outcome:    outcome,
		}, actor)
		assert.Equal(t, "outcome", fieldOf(t, err))
	}

	_, err := service.SettleMine(t.Context(), &agentsuggestionports.SettleAgentSuggestionRequest{
		TenantInfo: actor.TenantInfo(),
		Outcome:    agentsuggestion.OutcomeKept,
	}, actor)
	assert.Equal(t, "id", fieldOf(t, err))
	assert.Empty(t, repo.settled)
}
