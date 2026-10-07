package agentdraftservice

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/aiusage"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type stubCompleter struct {
	reply string
	err   error
	calls int
	saw   *serviceports.StructuredCompletionRequest
}

func (s *stubCompleter) CompleteStructured(
	_ context.Context,
	req *serviceports.StructuredCompletionRequest,
) (*serviceports.StructuredCompletionResult, error) {
	s.calls++
	s.saw = req
	if s.err != nil {
		return nil, s.err
	}

	return &serviceports.StructuredCompletionResult{Text: s.reply, ModelIdentifier: "stub"}, nil
}

type stubDefinitions struct {
	serviceports.AgentDefinitionService
	drafted *serviceports.SaveAgentDefinitionRequest
}

func (s *stubDefinitions) ToolCatalog(
	context.Context,
	pagination.TenantInfo,
) ([]serviceports.ToolCatalogEntry, error) {
	return testCatalog(), nil
}

func (s *stubDefinitions) Draft(
	_ context.Context,
	req *serviceports.SaveAgentDefinitionRequest,
) (*agentdefinition.Definition, error) {
	s.drafted = req
	definition := &agentdefinition.Definition{
		OrganizationID:    req.TenantInfo.OrgID,
		BusinessUnitID:    req.TenantInfo.BuID,
		Name:              req.Name,
		Description:       req.Description,
		Instructions:      req.Instructions,
		Guardrails:        req.Guardrails,
		ToolNames:         req.ToolNames,
		ToolTiers:         req.ToolTiers,
		AutonomyCeiling:   req.AutonomyCeiling,
		DataAccessCeiling: req.DataAccessCeiling,
		Enabled:           req.Enabled,
		ShadowMode:        req.ShadowMode,
		TriggerMode:       req.TriggerMode,
		CronExpression:    req.CronExpression,
		CronTimezone:      req.CronTimezone,
		EventKinds:        req.EventKinds,
		IntervalSeconds:   req.IntervalSeconds,
		Icon:              req.Icon,
		Accent:            req.Accent,
		OutputMode:        req.OutputMode,
	}
	definition.ApplyDefaults()

	return definition, nil
}

type stubProviders struct {
	repositories.AIProviderRepository
	providers []*aiprovider.Provider
	err       error
	asked     aiprovider.Task
}

func (s *stubProviders) ListForTask(
	_ context.Context,
	req repositories.ListAIProvidersForTaskRequest,
) ([]*aiprovider.Provider, error) {
	s.asked = req.Task
	if s.err != nil {
		return nil, s.err
	}

	return s.providers, nil
}

type stubOrganizations struct {
	repositories.OrganizationRepository
	timezone string
}

func (s *stubOrganizations) GetByID(
	context.Context,
	repositories.GetOrganizationByIDRequest,
) (*tenant.Organization, error) {
	return &tenant.Organization{Timezone: s.timezone}, nil
}

func testTenant() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}
}

func newTestService(
	completer *stubCompleter,
	definitions *stubDefinitions,
	providers *stubProviders,
) *Service {
	return New(Params{
		Logger:        zap.NewNop(),
		Completion:    completer,
		Definitions:   definitions,
		Providers:     providers,
		Organizations: &stubOrganizations{timezone: "America/Chicago"},
	})
}

const validDraftReply = `{
  "name": "Delivery desk",
  "description": "Tells customers where their loads are.",
  "icon": "headset",
  "accent": "teal",
  "instructions": "Look the shipment up before you answer.",
  "guardrails": ["Never promise a delivery date."],
  "triggerMode": "Chat",
  "cronExpression": "",
  "cronTimezone": "",
  "eventKinds": [],
  "intervalSeconds": 0,
  "tools": [
    {"name": "get_shipment", "tier": "Propose"},
    {"name": "send_customer_email", "tier": "AutoExecute"},
    {"name": "exec_shell", "tier": "AutoExecute"}
  ],
  "autonomyCeiling": "AutoExecute",
  "outputMode": "Conversational"
}`

func TestDraftFromDescriptionAsksTheChatProvidersAndSavesNothing(t *testing.T) {
	t.Parallel()

	completer := &stubCompleter{reply: validDraftReply}
	definitions := &stubDefinitions{}
	service := newTestService(completer, definitions, &stubProviders{})
	tenantInfo := testTenant()

	draft, err := service.DraftFromDescription(t.Context(), &serviceports.DraftAgentRequest{
		TenantInfo:  tenantInfo,
		Description: "Answer customers asking where their loads are.",
	})
	require.NoError(t, err)

	require.NotNil(t, completer.saw)
	assert.Equal(t, aiprovider.TaskAssistantChat, completer.saw.Task)
	assert.Equal(t, aiusage.FeatureAgentDrafting, completer.saw.Attribution.Feature)
	assert.Equal(t, tenantInfo.UserID, completer.saw.Attribution.UserID)
	assert.NotEmpty(t, completer.saw.OutputSchema)

	var described bool
	for _, section := range completer.saw.Context.Sections {
		if strings.Contains(section.Content, "Answer customers asking where their loads are.") {
			described = true
			assert.False(t, section.Trusted, "the person's description is never trusted")
		}
	}
	assert.True(t, described)

	require.NotNil(t, definitions.drafted)
	assert.True(t, definitions.drafted.ID.IsNil())
	assert.Equal(t, "Delivery desk", draft.Name)
	assert.Equal(t, []string{"get_shipment", "send_customer_email"}, draft.ToolNames)
	assert.Equal(
		t,
		map[string]agent.AutonomyTier{"send_customer_email": agent.TierActWithApproval},
		draft.ToolTiers,
	)
	assert.True(t, draft.ShadowMode)
	assert.Equal(t, agentdefinition.DefaultDecisionTimeoutSeconds, draft.DecisionTimeoutSeconds)
	assert.ElementsMatch(
		t,
		[]string{"toolNames", "toolTiers.send_customer_email"},
		noteFields(draft.Notes),
	)
}

func TestDraftFromDescriptionRefusesBeforeAskingTheModel(t *testing.T) {
	t.Parallel()

	completer := &stubCompleter{reply: validDraftReply}
	service := newTestService(completer, &stubDefinitions{}, &stubProviders{})

	_, err := service.DraftFromDescription(t.Context(), &serviceports.DraftAgentRequest{
		TenantInfo:  testTenant(),
		Description: "   ",
	})
	require.Error(t, err)

	_, err = service.DraftFromDescription(t.Context(), &serviceports.DraftAgentRequest{
		TenantInfo:  testTenant(),
		Description: strings.Repeat("a", maxDescriptionRunes+1),
	})
	require.Error(t, err)

	assert.Zero(t, completer.calls)
}

func TestDraftFromDescriptionFailsClearlyOnAnEmptyReply(t *testing.T) {
	t.Parallel()

	replies := []string{
		"",
		"I can't help with that.",
		`{"name":"","instructions":"","tools":[]}`,
	}
	for _, reply := range replies {
		completer := &stubCompleter{reply: reply}
		definitions := &stubDefinitions{}
		service := newTestService(completer, definitions, &stubProviders{})

		_, err := service.DraftFromDescription(t.Context(), &serviceports.DraftAgentRequest{
			TenantInfo:  testTenant(),
			Description: "Watch for late loads.",
		})

		require.Error(t, err, reply)
		assert.True(t, errortypes.IsBusinessError(err), reply)
		assert.Nil(t, definitions.drafted, reply)
	}
}

func TestDraftFromDescriptionPassesOnTheRoutersError(t *testing.T) {
	t.Parallel()

	routerErr := errortypes.NewBusinessError("No AI provider is configured for AssistantChat").
		WithInternal(serviceports.ErrNoProviderConfigured)
	service := newTestService(&stubCompleter{err: routerErr}, &stubDefinitions{}, &stubProviders{})

	_, err := service.DraftFromDescription(t.Context(), &serviceports.DraftAgentRequest{
		TenantInfo:  testTenant(),
		Description: "Watch for late loads.",
	})

	require.ErrorIs(t, err, serviceports.ErrNoProviderConfigured)
}

func TestTightenInstructionsKeepsPlaceholdersAndAsksTheChatProviders(t *testing.T) {
	t.Parallel()

	completer := &stubCompleter{reply: `{"instructions":"Help {{user.name}}. Look loads up first."}`}
	service := newTestService(completer, &stubDefinitions{}, &stubProviders{})

	result, err := service.TightenInstructions(
		t.Context(),
		&serviceports.TightenAgentInstructionsRequest{
			TenantInfo:   testTenant(),
			Instructions: "You should help {{user.name}}. Always look the loads up before anything else.",
		},
	)
	require.NoError(t, err)

	assert.Equal(t, "Help {{user.name}}. Look loads up first.", result.Instructions)
	assert.True(t, result.Changed)
	assert.Equal(t, aiprovider.TaskAssistantChat, completer.saw.Task)
	assert.Equal(t, aiusage.FeatureAgentDrafting, completer.saw.Attribution.Feature)
}

func TestTightenInstructionsRefusesAReplyThatLostAPlaceholder(t *testing.T) {
	t.Parallel()

	completer := &stubCompleter{reply: `{"instructions":"Help the person."}`}
	service := newTestService(completer, &stubDefinitions{}, &stubProviders{})

	_, err := service.TightenInstructions(
		t.Context(),
		&serviceports.TightenAgentInstructionsRequest{
			TenantInfo:   testTenant(),
			Instructions: "You should help {{user.name}}.",
		},
	)

	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
}

func TestTightenInstructionsRefusesEmptyAndOverlongInstructions(t *testing.T) {
	t.Parallel()

	completer := &stubCompleter{reply: `{"instructions":"x"}`}
	service := newTestService(completer, &stubDefinitions{}, &stubProviders{})

	for _, text := range []string{" ", strings.Repeat("x", agentdefinition.MaxInstructionsRunes+1)} {
		_, err := service.TightenInstructions(
			t.Context(),
			&serviceports.TightenAgentInstructionsRequest{TenantInfo: testTenant(), Instructions: text},
		)
		require.Error(t, err)
	}
	assert.Zero(t, completer.calls)
}

func TestAvailableWhenAnEnabledProviderTakesAssistantChat(t *testing.T) {
	t.Parallel()

	chat := &aiprovider.Provider{
		Name:    "Chat",
		Enabled: true,
		Tasks:   []aiprovider.Task{aiprovider.TaskAssistantChat},
	}
	disabled := &aiprovider.Provider{
		Name:    "Off",
		Enabled: false,
		Tasks:   []aiprovider.Task{aiprovider.TaskAssistantChat},
	}
	other := &aiprovider.Provider{
		Name:    "Docs",
		Enabled: true,
		Tasks:   []aiprovider.Task{aiprovider.TaskDocumentExtraction},
	}

	cases := []struct {
		name      string
		providers []*aiprovider.Provider
		want      bool
	}{
		{name: "none", want: false},
		{name: "disabled", providers: []*aiprovider.Provider{disabled}, want: false},
		{name: "another task", providers: []*aiprovider.Provider{other}, want: false},
		{name: "covered", providers: []*aiprovider.Provider{disabled, other, chat}, want: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			providers := &stubProviders{providers: tc.providers}
			service := newTestService(&stubCompleter{}, &stubDefinitions{}, providers)

			available, err := service.Available(t.Context(), testTenant())

			require.NoError(t, err)
			assert.Equal(t, tc.want, available)
			assert.Equal(t, aiprovider.TaskAssistantChat, providers.asked)
		})
	}
}

func TestAvailablePassesOnALookupFailure(t *testing.T) {
	t.Parallel()

	lookupErr := errors.New("database is down")
	service := newTestService(
		&stubCompleter{},
		&stubDefinitions{},
		&stubProviders{err: lookupErr},
	)

	_, err := service.Available(t.Context(), testTenant())

	require.ErrorIs(t, err, lookupErr)
}
