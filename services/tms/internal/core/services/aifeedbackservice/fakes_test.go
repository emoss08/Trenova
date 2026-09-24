package aifeedbackservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/aifeedback"
	"github.com/emoss08/trenova/internal/core/domain/briefing"
	"github.com/emoss08/trenova/internal/core/domain/insight"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/watchtower"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

type fakeStore struct {
	upserted  []*aifeedback.Feedback
	deleted   []repositories.DeleteAIFeedbackRequest
	listed    []repositories.ListAIFeedbackForTargetsRequest
	negatives []*aifeedback.Feedback
	purges    []repositories.PurgeAIFeedbackRequest
	purgeRows []int64
}

func (f *fakeStore) Upsert(
	_ context.Context,
	entity *aifeedback.Feedback,
) (*aifeedback.Feedback, error) {
	if entity.ID.IsNil() {
		entity.ID = pulid.MustNew("aifb_")
	}
	f.upserted = append(f.upserted, entity)

	return entity, nil
}

func (f *fakeStore) Delete(
	_ context.Context,
	req repositories.DeleteAIFeedbackRequest,
) (bool, error) {
	f.deleted = append(f.deleted, req)

	return true, nil
}

func (f *fakeStore) ListForTargets(
	_ context.Context,
	req repositories.ListAIFeedbackForTargetsRequest,
) ([]*aifeedback.Feedback, error) {
	f.listed = append(f.listed, req)

	return []*aifeedback.Feedback{}, nil
}

func (f *fakeStore) ListByIDs(
	context.Context,
	repositories.ListAIFeedbackByIDsRequest,
) ([]*aifeedback.Feedback, error) {
	return []*aifeedback.Feedback{}, nil
}

func (f *fakeStore) ListConnection(
	context.Context,
	*repositories.ListAIFeedbackConnectionRequest,
) (*pagination.CursorListResult[*aifeedback.Feedback], error) {
	return &pagination.CursorListResult[*aifeedback.Feedback]{}, nil
}

func (f *fakeStore) DailySatisfaction(
	context.Context,
	repositories.AIFeedbackWindowRequest,
) ([]*repositories.AIFeedbackDay, error) {
	return []*repositories.AIFeedbackDay{}, nil
}

func (f *fakeStore) WorstRated(
	context.Context,
	repositories.AIFeedbackWindowRequest,
) ([]*repositories.AIFeedbackTargetScore, error) {
	return []*repositories.AIFeedbackTargetScore{}, nil
}

func (f *fakeStore) ListNegativeSince(
	context.Context,
	repositories.ListNegativeAIFeedbackRequest,
) ([]*aifeedback.Feedback, error) {
	return f.negatives, nil
}

func (f *fakeStore) PurgeBefore(
	_ context.Context,
	req repositories.PurgeAIFeedbackRequest,
) (int64, error) {
	f.purges = append(f.purges, req)
	if len(f.purgeRows) == 0 {
		return 0, nil
	}
	next := f.purgeRows[0]
	f.purgeRows = f.purgeRows[1:]

	return next, nil
}

type fakeMessages struct {
	context *repositories.AIFeedbackMessageContext
	err     error
}

func (f *fakeMessages) GetMessageContext(
	context.Context,
	repositories.GetAIFeedbackMessageRequest,
) (*repositories.AIFeedbackMessageContext, error) {
	return f.context, f.err
}

type fakeBriefings struct{ entity *briefing.Briefing }

func (f *fakeBriefings) GetByID(
	context.Context,
	repositories.GetBriefingByIDRequest,
) (*briefing.Briefing, error) {
	if f.entity == nil {
		return nil, errortypes.NewNotFoundError("Briefing not found")
	}

	return f.entity, nil
}

type fakeInsights struct{ byID map[pulid.ID]*insight.Insight }

func (f *fakeInsights) GetByID(
	_ context.Context,
	req repositories.GetInsightByIDRequest,
) (*insight.Insight, error) {
	if entity, ok := f.byID[req.ID]; ok {
		return entity, nil
	}

	return nil, errortypes.NewNotFoundError("Insight not found")
}

type fakeWatchtower struct{ item *watchtower.Item }

func (f *fakeWatchtower) GetByID(
	context.Context,
	repositories.GetWatchtowerItemRequest,
) (*watchtower.Item, error) {
	if f.item == nil {
		return nil, errortypes.NewNotFoundError("Item not found")
	}

	return f.item, nil
}

type fakeProposals struct{ proposal *agent.AgentProposal }

func (f *fakeProposals) GetByID(
	context.Context,
	repositories.GetAgentProposalByIDRequest,
) (*agent.AgentProposal, error) {
	return f.proposal, nil
}

type fakePlans struct{ plan *agent.AgentPlan }

func (f *fakePlans) GetByID(
	context.Context,
	repositories.GetAgentPlanByIDRequest,
) (*agent.AgentPlan, error) {
	return f.plan, nil
}

type fakeRuns struct{ run *agent.AgentRun }

func (f *fakeRuns) GetByID(
	context.Context,
	repositories.GetAgentRunByIDRequest,
) (*agent.AgentRun, error) {
	if f.run == nil {
		return nil, errortypes.NewNotFoundError("Run not found")
	}

	return f.run, nil
}

type fakeExceptions struct{ exception *agent.AgentException }

func (f *fakeExceptions) GetByID(
	context.Context,
	repositories.GetAgentExceptionByIDRequest,
) (*agent.AgentException, error) {
	return f.exception, nil
}

type fakeDefinitions struct{ definition *agentdefinition.Definition }

func (f *fakeDefinitions) GetByID(
	context.Context,
	repositories.GetAgentDefinitionByIDRequest,
) (*agentdefinition.Definition, error) {
	if f.definition == nil {
		return nil, errortypes.NewNotFoundError("Agent not found")
	}

	return f.definition, nil
}

type fakeMemories struct {
	existing []*agent.Memory
	created  []*agent.Memory
	asked    []repositories.ListAgentMemorySuggestionContextRequest
}

func (f *fakeMemories) Create(_ context.Context, entity *agent.Memory) (*agent.Memory, error) {
	entity.ID = pulid.MustNew("amem_")
	f.created = append(f.created, entity)

	return entity, nil
}

func (f *fakeMemories) ListSuggestionContext(
	_ context.Context,
	req repositories.ListAgentMemorySuggestionContextRequest,
) ([]*agent.Memory, error) {
	f.asked = append(f.asked, req)

	return f.existing, nil
}

type fakeRetention struct{ row *tenant.DataRetention }

func (f *fakeRetention) Get(
	context.Context,
	repositories.GetDataRetentionRequest,
) (*tenant.DataRetention, error) {
	if f.row == nil {
		return nil, errortypes.NewNotFoundError("Data Retention not found")
	}

	return f.row, nil
}

type fakeOrganizations struct{}

func (fakeOrganizations) GetByID(context.Context, pulid.ID) (*tenant.Organization, error) {
	return &tenant.Organization{Timezone: "America/Chicago"}, nil
}

type fakePermissions struct {
	allowed map[permission.Resource]bool
	asked   []string
}

func (f *fakePermissions) Check(
	_ context.Context,
	req *services.PermissionCheckRequest,
) (*services.PermissionCheckResult, error) {
	f.asked = append(f.asked, req.Resource)

	return &services.PermissionCheckResult{
		Allowed: f.allowed[permission.Resource(req.Resource)],
	}, nil
}

type fakeToolResources map[string]permission.Resource

func (f fakeToolResources) Resource(name string) (permission.Resource, bool) {
	resource, ok := f[name]

	return resource, ok
}

type fakeSensitivity map[string]permission.FieldSensitivity

func (f fakeSensitivity) GetFieldSensitivity(_, field string) permission.FieldSensitivity {
	if sensitivity, ok := f[field]; ok {
		return sensitivity
	}

	return permission.SensitivityInternal
}

type harness struct {
	svc         *Service
	store       *fakeStore
	messages    *fakeMessages
	briefings   *fakeBriefings
	insights    *fakeInsights
	watchtower  *fakeWatchtower
	proposals   *fakeProposals
	plans       *fakePlans
	runs        *fakeRuns
	exceptions  *fakeExceptions
	definitions *fakeDefinitions
	memories    *fakeMemories
	retention   *fakeRetention
	permissions *fakePermissions
}

func newHarness() *harness {
	h := &harness{
		store:       &fakeStore{},
		messages:    &fakeMessages{},
		briefings:   &fakeBriefings{},
		insights:    &fakeInsights{byID: map[pulid.ID]*insight.Insight{}},
		watchtower:  &fakeWatchtower{},
		proposals:   &fakeProposals{},
		plans:       &fakePlans{},
		runs:        &fakeRuns{},
		exceptions:  &fakeExceptions{},
		definitions: &fakeDefinitions{},
		memories:    &fakeMemories{},
		retention:   &fakeRetention{},
		permissions: &fakePermissions{allowed: map[permission.Resource]bool{}},
	}
	h.svc = &Service{
		l:             zap.NewNop(),
		repo:          h.store,
		sources:       h.messages,
		briefings:     h.briefings,
		insights:      h.insights,
		watchtower:    h.watchtower,
		proposals:     h.proposals,
		plans:         h.plans,
		runs:          h.runs,
		exceptions:    h.exceptions,
		definitions:   h.definitions,
		memories:      h.memories,
		retention:     h.retention,
		organizations: fakeOrganizations{},
		permissions:   h.permissions,
		redactor:      &redactor{tools: fakeToolResources{}, registry: fakeSensitivity{}},
		now:           func() int64 { return 1_760_000_000 },
	}

	return h
}

func testTenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
}

func person(tenant pagination.TenantInfo) *services.RequestActor {
	userID := pulid.MustNew("usr_")

	return &services.RequestActor{
		PrincipalType:  services.PrincipalTypeUser,
		PrincipalID:    userID,
		UserID:         userID,
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
	}
}

func (f *fakeStore) TotalsByAgent(
	context.Context,
	repositories.AIFeedbackAgentTotalsRequest,
) ([]*repositories.AIFeedbackAgentTotals, error) {
	return nil, nil
}
