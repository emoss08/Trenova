package agentdefinitionresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttemplate"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentbudgetports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttrustports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubDefinitions struct {
	agentdefinitionports.AgentDefinitionService

	definition *agentdefinition.Definition
	byID       []agentdefinitionports.GetAgentDefinitionByIDRequest
	created    []*agentdefinitionports.SaveAgentDefinitionRequest
	updated    []*agentdefinitionports.SaveAgentDefinitionRequest
	deleted    []agentdefinitionports.DeleteAgentDefinitionRequest
	actors     []*permissionports.RequestActor
}

func (s *stubDefinitions) GetByID(
	_ context.Context,
	req agentdefinitionports.GetAgentDefinitionByIDRequest,
) (*agentdefinition.Definition, error) {
	s.byID = append(s.byID, req)
	return s.definition, nil
}

func (s *stubDefinitions) Create(
	_ context.Context,
	req *agentdefinitionports.SaveAgentDefinitionRequest,
	actor *permissionports.RequestActor,
) (*agentdefinition.Definition, error) {
	s.created = append(s.created, req)
	s.actors = append(s.actors, actor)
	return &agentdefinition.Definition{ID: pulid.MustNew("agdef_"), Name: req.Name}, nil
}

func (s *stubDefinitions) Update(
	_ context.Context,
	req *agentdefinitionports.SaveAgentDefinitionRequest,
	actor *permissionports.RequestActor,
) (*agentdefinition.Definition, error) {
	s.updated = append(s.updated, req)
	s.actors = append(s.actors, actor)
	return &agentdefinition.Definition{ID: req.ID, Name: req.Name}, nil
}

func (s *stubDefinitions) Delete(
	_ context.Context,
	req agentdefinitionports.DeleteAgentDefinitionRequest,
	actor *permissionports.RequestActor,
) error {
	s.deleted = append(s.deleted, req)
	s.actors = append(s.actors, actor)
	return nil
}

func (s *stubDefinitions) EventKinds() []agenttrigger.EventDescriptor {
	return []agenttrigger.EventDescriptor{{
		Kind:        agenttrigger.EventBillingQueueItemException,
		SubjectType: agent.SubjectBillingQueueItem,
		Label:       "Billing item blocked",
	}}
}

type stubBudgets struct {
	agentbudgetports.AgentBudgetService

	asked []*agentdefinition.Definition
}

func (s *stubBudgets) Status(
	_ context.Context,
	definition *agentdefinition.Definition,
) (*agentbudgetports.AgentBudgetStatus, error) {
	s.asked = append(s.asked, definition)
	return &agentbudgetports.AgentBudgetStatus{SpentUSD: "4.20", RunsToday: 3}, nil
}

type stubTrust struct {
	agenttrustports.AgentTrustService

	requests []agenttoolports.ListToolTrustRequest
}

func (s *stubTrust) ListForDefinition(
	_ context.Context,
	req agenttoolports.ListToolTrustRequest,
) ([]*agenttooltrust.ToolTrust, error) {
	s.requests = append(s.requests, req)
	return []*agenttooltrust.ToolTrust{
		{ToolName: "customer_update", Streak: 4, EarnedTier: agent.TierActWithApproval},
		{ToolName: "shipment_get", Streak: 1},
	}, nil
}

func minimalSaveInput() gqlmodel.SaveAgentDefinitionInput {
	return gqlmodel.SaveAgentDefinitionInput{
		Name:              "Payroll",
		AutonomyCeiling:   agent.TierPropose,
		DataAccessCeiling: agentdefinition.DataAccessInternal,
		TriggerMode:       agentdefinition.TriggerChat,
		OutputMode:        agentdefinition.OutputConversational,
	}
}

func TestAgentDefinitionAdmin_ChecksTheAgentDefinitionPermission(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("agdef_").String()
	calls := []struct {
		name      string
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{"agentBudgetStatus", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).AgentBudgetStatus(ctx, id)
			return err
		}},
		{"agentToolTrust", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).AgentToolTrust(ctx, id)
			return err
		}},
		{"agentToolCatalog", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).AgentToolCatalog(ctx)
			return err
		}},
		{"agentEventKinds", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).AgentEventKinds(ctx)
			return err
		}},
		{"createAgentDefinition", permission.OpCreate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).CreateAgentDefinition(ctx, minimalSaveInput())
			return err
		}},
		{"updateAgentDefinition", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).UpdateAgentDefinition(ctx, id, minimalSaveInput())
			return err
		}},
		{"deleteAgentDefinition", permission.OpDelete, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).DeleteAgentDefinition(ctx, id)
			return err
		}},
	}

	for _, call := range calls {
		t.Run(call.name, func(t *testing.T) {
			t.Parallel()

			engine := resolvertest.RefusingEngine()
			deps := &Deps{Core: &resolvercore.Core{PermissionEngine: engine}}
			ctx := resolvertest.AuthedContext(t, resolvertest.UserAuthContext())

			err := call.run(ctx, deps)

			resolvertest.RequireRefused(
				t,
				err,
				engine,
				permission.ResourceAgentDefinition,
				call.operation,
			)
		})
	}
}

func TestCreateAgentDefinition_SavesAsTheCallerInTheirTenant(t *testing.T) {
	t.Parallel()

	definitions := &stubDefinitions{}
	deps := &Deps{
		Core: &resolvercore.Core{
			PermissionEngine: &mocks.AllowAllPermissionEngine{},
		},
		AgentDefinitionService: definitions,
	}
	authCtx := resolvertest.UserAuthContext()
	ctx := resolvertest.AuthedContext(t, authCtx)

	created, err := (&MutationResolver{deps}).CreateAgentDefinition(ctx, minimalSaveInput())
	require.NoError(t, err)

	require.Len(t, definitions.created, 1)
	req := definitions.created[0]
	assert.True(t, req.ID.IsNil(), "a new agent carries no id")
	assert.Equal(t, authCtx.OrganizationID, req.TenantInfo.OrgID)
	assert.Equal(t, authCtx.BusinessUnitID, req.TenantInfo.BuID)
	assert.Equal(t, authCtx.UserID, definitions.actors[0].UserID)
	assert.Equal(t, "Payroll", created.Name)
}

func TestUpdateAgentDefinition_SavesTheNamedAgent(t *testing.T) {
	t.Parallel()

	definitions := &stubDefinitions{}
	deps := &Deps{
		Core: &resolvercore.Core{
			PermissionEngine: &mocks.AllowAllPermissionEngine{},
		},
		AgentDefinitionService: definitions,
	}
	ctx := resolvertest.AuthedContext(t, resolvertest.UserAuthContext())
	id := pulid.MustNew("agdef_")

	_, err := (&MutationResolver{deps}).UpdateAgentDefinition(ctx, id.String(), minimalSaveInput())
	require.NoError(t, err)

	require.Len(t, definitions.updated, 1)
	assert.Equal(t, id, definitions.updated[0].ID)
}

func TestDeleteAgentDefinition_DeletesTheNamedAgentInTheCallersTenant(t *testing.T) {
	t.Parallel()

	definitions := &stubDefinitions{}
	deps := &Deps{
		Core: &resolvercore.Core{
			PermissionEngine: &mocks.AllowAllPermissionEngine{},
		},
		AgentDefinitionService: definitions,
	}
	authCtx := resolvertest.UserAuthContext()
	ctx := resolvertest.AuthedContext(t, authCtx)
	id := pulid.MustNew("agdef_")

	deleted, err := (&MutationResolver{deps}).DeleteAgentDefinition(ctx, id.String())
	require.NoError(t, err)

	assert.True(t, deleted)
	require.Len(t, definitions.deleted, 1)
	assert.Equal(t, id, definitions.deleted[0].ID)
	assert.Equal(t, authCtx.OrganizationID, definitions.deleted[0].TenantInfo.OrgID)
}

func TestAgentBudgetStatus_ReadsTheAgentThenWhereItStands(t *testing.T) {
	t.Parallel()

	definition := &agentdefinition.Definition{ID: pulid.MustNew("agdef_"), Name: "Rates"}
	definitions := &stubDefinitions{definition: definition}
	budgets := &stubBudgets{}
	deps := &Deps{
		Core: &resolvercore.Core{
			PermissionEngine: &mocks.AllowAllPermissionEngine{},
		},
		AgentDefinitionService: definitions,
		AgentBudgetService:     budgets,
	}
	authCtx := resolvertest.UserAuthContext()
	ctx := resolvertest.AuthedContext(t, authCtx)

	status, err := (&QueryResolver{deps}).AgentBudgetStatus(ctx, definition.ID.String())
	require.NoError(t, err)

	require.Len(t, definitions.byID, 1)
	assert.Equal(t, definition.ID, definitions.byID[0].ID)
	assert.Equal(t, authCtx.OrganizationID, definitions.byID[0].TenantInfo.OrgID)
	assert.Equal(t, []*agentdefinition.Definition{definition}, budgets.asked)
	assert.Equal(t, "4.20", status.SpentUSD)
	assert.Equal(t, 3, status.RunsToday)
}

func TestAgentToolTrust_ReadsTheLedgerForTheAgent(t *testing.T) {
	t.Parallel()

	trust := &stubTrust{}
	deps := &Deps{
		Core:              &resolvercore.Core{PermissionEngine: &mocks.AllowAllPermissionEngine{}},
		AgentTrustService: trust,
	}
	authCtx := resolvertest.UserAuthContext()
	ctx := resolvertest.AuthedContext(t, authCtx)
	id := pulid.MustNew("agdef_")

	rows, err := (&QueryResolver{deps}).AgentToolTrust(ctx, id.String())
	require.NoError(t, err)

	require.Len(t, trust.requests, 1)
	assert.Equal(t, id, trust.requests[0].AgentDefinitionID)
	assert.Equal(t, authCtx.OrganizationID, trust.requests[0].TenantInfo.OrgID)
	require.Len(t, rows, 2)
	assert.Equal(t, "customer_update", rows[0].ToolName)
	assert.Equal(t, 4, rows[0].Streak)
	require.NotNil(t, rows[0].EarnedTier)
	assert.Equal(t, agent.TierActWithApproval, *rows[0].EarnedTier)
	assert.Nil(t, rows[1].EarnedTier, "a tier a person chose is not one the ledger granted")
}

func TestAgentEventKinds_ListsWhatCanStartAnAgent(t *testing.T) {
	t.Parallel()

	deps := &Deps{
		Core: &resolvercore.Core{
			PermissionEngine: &mocks.AllowAllPermissionEngine{},
		},
		AgentDefinitionService: &stubDefinitions{},
	}
	ctx := resolvertest.AuthedContext(t, resolvertest.UserAuthContext())

	kinds, err := (&QueryResolver{deps}).AgentEventKinds(ctx)
	require.NoError(t, err)

	require.Len(t, kinds, 1)
	assert.Equal(t, agenttrigger.EventBillingQueueItemException, kinds[0].Kind)
	assert.Equal(t, agent.SubjectBillingQueueItem, kinds[0].SubjectType)
}

func TestSaveRequestFromInput_MapsTheBuildersInput(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	template := agenttemplate.TemplateBillingException
	budget := "125.50"
	endsAt := 1_800_000_000
	memory := 9000
	provider := pulid.MustNew("aip_")
	providerRaw := provider.String()
	input := minimalSaveInput()
	input.Template = &template
	input.ToolTiers = []*gqlmodel.AgentToolTierInput{
		{Tool: "customer_update", Tier: agent.TierActWithApproval},
	}
	input.ToolDailyLimits = []*gqlmodel.AgentToolDailyLimitInput{
		{Tool: "customer_update", Limit: 20},
	}
	input.EventKinds = []string{string(agenttrigger.EventBillingQueueItemException)}
	input.MonthlyBudgetUsd = &budget
	input.EndsAt = &endsAt
	input.MemoryTokenBudget = &memory
	input.PreferredProviderID = &providerRaw
	input.DataAccessCeiling = agentdefinition.DataAccessRestricted
	input.Version = 7

	req, err := saveRequestFromInput(pulid.Nil, &input, tenant)
	require.NoError(t, err)

	assert.Equal(t, tenant, req.TenantInfo)
	assert.Equal(t, agenttemplate.TemplateBillingException, req.Template)
	assert.Equal(t,
		map[string]agent.AutonomyTier{"customer_update": agent.TierActWithApproval},
		req.ToolTiers,
	)
	assert.Equal(t, map[string]int{"customer_update": 20}, req.ToolDailyLimits)
	assert.Equal(t,
		[]agenttrigger.EventKind{agenttrigger.EventBillingQueueItemException},
		req.EventKinds,
	)
	require.NotNil(t, req.MonthlyBudgetUSD)
	assert.True(t, decimal.RequireFromString("125.50").Equal(*req.MonthlyBudgetUSD))
	require.NotNil(t, req.EndsAt)
	assert.Equal(t, int64(1_800_000_000), *req.EndsAt)
	assert.Equal(t, &memory, req.MemoryTokenBudget)
	assert.Equal(t, provider, req.PreferredProviderID)
	assert.Equal(t, agentdefinition.DataAccessRestricted, req.DataAccessCeiling)
	assert.Equal(t, int64(7), req.Version)
	assert.Nil(t, req.DelegateIDs, "absent delegates keep the ones the agent has")
	assert.Nil(t, req.Access, "absent access keeps who may use the agent")
}

func TestSaveRequestFromInput_AnEmptyDelegateListClearsThem(t *testing.T) {
	t.Parallel()

	input := minimalSaveInput()
	input.DelegateIds = []string{}

	req, err := saveRequestFromInput(pulid.Nil, &input, pagination.TenantInfo{})
	require.NoError(t, err)

	require.NotNil(t, req.DelegateIDs)
	assert.Empty(t, *req.DelegateIDs)
}

/*
A save says who may use the agent with both fields or with neither. Neither
keeps who may use it; both set it with the save; one without the other is
refused on the missing field rather than read as "no roles" or "Everyone".
*/
func TestSaveRequestFromInput_AccessIsBothFieldsOrNeither(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	role := pulid.MustNew("rol_")
	restricted := agentdefinition.AccessRoles

	both := minimalSaveInput()
	both.AccessMode = &restricted
	both.AccessRoleIds = []string{role.String()}
	req, err := saveRequestFromInput(pulid.Nil, &both, tenant)
	require.NoError(t, err)
	require.NotNil(t, req.Access)
	assert.Equal(t, agentdefinition.AccessRoles, req.Access.Mode)
	assert.Equal(t, []pulid.ID{role}, req.Access.RoleIDs)

	var invalid *errortypes.Error
	modeOnly := minimalSaveInput()
	modeOnly.AccessMode = &restricted
	_, err = saveRequestFromInput(pulid.Nil, &modeOnly, tenant)
	require.ErrorAs(t, err, &invalid)
	assert.Equal(t, "accessRoleIds", invalid.Field)

	rolesOnly := minimalSaveInput()
	rolesOnly.AccessRoleIds = []string{role.String()}
	_, err = saveRequestFromInput(pulid.Nil, &rolesOnly, tenant)
	require.ErrorAs(t, err, &invalid)
	assert.Equal(t, "accessMode", invalid.Field)
}

func TestSaveRequestFromInput_RefusesTwoTiersForOneTool(t *testing.T) {
	t.Parallel()

	input := minimalSaveInput()
	input.ToolTiers = []*gqlmodel.AgentToolTierInput{
		{Tool: "customer_update", Tier: agent.TierPropose},
		{Tool: "customer_update", Tier: agent.TierAutoExecute},
	}

	_, err := saveRequestFromInput(pulid.Nil, &input, pagination.TenantInfo{})

	var invalid *errortypes.Error
	require.ErrorAs(t, err, &invalid)
	assert.Equal(t, "toolTiers", invalid.Field)
}

func TestSaveRequestFromInput_NamesAMalformedBudget(t *testing.T) {
	t.Parallel()

	input := minimalSaveInput()
	budget := "lots"
	input.MonthlyBudgetUsd = &budget

	_, err := saveRequestFromInput(pulid.Nil, &input, pagination.TenantInfo{})

	var invalid *errortypes.Error
	require.ErrorAs(t, err, &invalid)
	assert.Equal(t, "monthlyBudgetUsd", invalid.Field)
}
