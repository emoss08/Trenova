package proposalpreviewservice

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdecision"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type fakeDB struct {
	mu   sync.Mutex
	opts []kernelports.TxOptions
}

func (f *fakeDB) Transact(
	ctx context.Context,
	opts kernelports.TxOptions,
	fn func(context.Context) error,
) error {
	f.mu.Lock()
	f.opts = append(f.opts, opts)
	f.mu.Unlock()

	return fn(ctx)
}

type fakeRegistry struct{ tools []agenttoolports.AgentTool }

func (r fakeRegistry) Get(name string) (agenttoolports.AgentTool, bool) {
	for _, tool := range r.tools {
		if tool.Name() == name {
			return tool, true
		}
	}

	return nil, false
}

func (r fakeRegistry) All() []agenttoolports.AgentTool { return r.tools }

func (r fakeRegistry) Descriptors() []agenttoolports.AgentToolDescriptor { return nil }

type baseTool struct {
	name     string
	resource permission.Resource
	scope    agenttooltrust.ToolScope
}

func (t *baseTool) Name() string        { return t.name }
func (t *baseTool) Description() string { return "A test tool." }
func (t *baseTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"shipmentId": map[string]any{"type": "string"},
			"status":     map[string]any{"type": "string"},
			"bankAccount": map[string]any{
				"type": "string",
			},
		},
	}
}

func (t *baseTool) Policy() agenttoolports.ToolPolicy {
	scope := t.scope
	if scope == "" {
		scope = agenttooltrust.ToolScopeTenant
	}

	return agenttoolports.ToolPolicy{
		Name:          t.name,
		Kind:          agenttooltrust.ToolKindAction,
		Resource:      t.resource,
		Operation:     permission.OpUpdate,
		Scope:         scope,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierActWithApproval,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Effect:        agenttooltrust.ToolEffectChange,
		ReadsExternal: agenttooltrust.ExternalReadNever,
		Rationale:     "A test tool.",
	}
}

func (t *baseTool) Execute(context.Context, agenttoolports.ToolExecuteParams) error {
	return fmt.Errorf("%s must never run in a preview", t.name)
}

func (t *baseTool) Target(params map[string]any) (agenttoolports.ToolTarget, bool) {
	raw, _ := params["shipmentId"].(string)
	id, err := pulid.Parse(raw)
	if err != nil {
		return agenttoolports.ToolTarget{}, false
	}

	return agenttoolports.ToolTarget{Resource: permission.ResourceShipment, ID: id}, true
}

// shipmentState is what the fake "database" holds for the previewing tool.
type shipmentState struct {
	ID         pulid.ID        `json:"id"`
	ProNumber  string          `json:"proNumber"`
	Status     string          `json:"status"`
	CustomerID pulid.ID        `json:"customerId"`
	RateAmount decimal.Decimal `json:"rateAmount"`
	Version    int64           `json:"version"`
}

type previewingTool struct {
	baseTool

	state   *shipmentState
	block   bool
	asked   []agenttoolports.ToolExecuteParams
	money   bool
	failErr error
	refusal error
}

func (t *previewingTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams,
) (*agentpreview.ToolPreview, error) {
	t.asked = append(t.asked, params)
	if t.block {
		<-ctx.Done()

		return nil, ctx.Err()
	}
	if t.failErr != nil {
		return nil, t.failErr
	}
	if t.refusal != nil {
		preview := toolpreview.Build("Would set PRO " + t.state.ProNumber + ".")
		preview.AddWarning(toolpreview.WouldFail(t.refusal))

		return preview, nil
	}

	status, _ := params.Params["status"].(string)
	version := t.state.Version
	change, err := toolpreview.Update(
		toolpreview.Record{
			Resource: permission.ResourceShipment,
			ID:       t.state.ID,
			Version:  &version,
		},
		t.state,
		func(s *shipmentState) error {
			s.Status = status
			if raw, ok := params.Params["customerId"].(string); ok {
				s.CustomerID = pulid.ID(raw)
			}
			s.Version++
			return nil
		},
		toolpreview.WithRefs(map[string]permission.Resource{
			"customerId": permission.ResourceCustomer,
		}),
	)
	if err != nil {
		return nil, err
	}
	if t.money {
		change.Money = toolpreview.MoneyBlock("USD", agentpreview.MoneyLine{
			Label:  "Linehaul",
			Before: decimal.NewNullDecimal(decimal.RequireFromString("100")),
			After:  decimal.NewNullDecimal(decimal.RequireFromString("120")),
		})
		change.Money.Sensitivity = permission.SensitivityRestricted
	}

	return toolpreview.Build("Would set PRO "+t.state.ProNumber+" to "+status+".", change), nil
}

type validatingTool struct {
	baseTool

	err error
}

func (t *validatingTool) Validate(context.Context, agenttoolports.ToolExecuteParams) error {
	return t.err
}

type fakeVersions struct {
	mu       sync.Mutex
	versions map[pulid.ID]int64
	missing  map[pulid.ID]bool
	asked    int
}

func (f *fakeVersions) Version(
	_ context.Context,
	_ pagination.TenantInfo,
	target agenttoolports.ToolTarget,
) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.asked++
	if f.missing[target.ID] {
		return 0, fmt.Errorf("read shipment: %w", sql.ErrNoRows)
	}

	return f.versions[target.ID], nil
}

type fakeLabeler struct {
	labels agentruntimeports.RecordLabels
	asked  []map[permission.Resource][]pulid.ID
}

func (f *fakeLabeler) Labels(
	_ context.Context,
	_ pagination.TenantInfo,
	refs map[permission.Resource][]pulid.ID,
) (agentruntimeports.RecordLabels, error) {
	f.asked = append(f.asked, refs)

	return f.labels, nil
}

type fakeBaselines struct {
	mu      sync.Mutex
	kept    []*agentpreview.ProposalBaseline
	stored  []*agentpreview.ProposalBaseline
	listErr error
}

func (f *fakeBaselines) Create(_ context.Context, baseline *agentpreview.ProposalBaseline) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.kept = append(f.kept, baseline)

	return nil
}

func (f *fakeBaselines) ListByProposals(
	_ context.Context,
	req agentports.ListProposalBaselinesRequest,
) ([]*agentpreview.ProposalBaseline, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	out := make([]*agentpreview.ProposalBaseline, 0, len(f.stored))
	for _, baseline := range f.stored {
		for _, id := range req.ProposalIDs {
			if baseline.ProposalID == id {
				out = append(out, baseline)
			}
		}
	}

	return out, nil
}

type fakeDecisions struct {
	decisions []*agentdecision.AgentDecision
}

func (f *fakeDecisions) ListByProposals(
	context.Context,
	agentports.ListAgentDecisionsByProposalsRequest,
) ([]*agentdecision.AgentDecision, error) {
	return f.decisions, nil
}

type fakeSteps struct {
	steps []*agent.AgentProposal
}

func (f *fakeSteps) ListByPlan(
	context.Context,
	agentports.ListAgentProposalsByPlanRequest,
) ([]*agent.AgentProposal, error) {
	return f.steps, nil
}

type fakeChecker struct {
	err      error
	review   bool
	byClass  bool
	reviewed map[string]any
}

func (f *fakeChecker) ReviewRequired(
	proposal *agent.AgentProposal,
	modifications map[string]any,
) bool {
	f.reviewed = modifications
	if f.byClass {
		return proposal.ApprovalNeedsReview()
	}

	return f.review
}

func (f *fakeChecker) CheckModifications(
	_ context.Context,
	proposal *agent.AgentProposal,
	modifications map[string]any,
	_ *permissionports.RequestActor,
) (map[string]any, error) {
	if f.err != nil {
		return nil, f.err
	}
	merged := make(map[string]any, len(proposal.ToolParams)+len(modifications))
	for key, value := range proposal.ToolParams {
		merged[key] = value
	}
	for key, value := range modifications {
		merged[key] = value
	}

	return merged, nil
}

type fakeCeilings struct {
	levels map[permission.Resource]permission.FieldSensitivity
}

func (f fakeCeilings) For(
	_ context.Context,
	resource permission.Resource,
) permission.FieldSensitivity {
	if level, ok := f.levels[resource]; ok {
		return level
	}

	return permission.SensitivityRestricted
}

type fakeReads struct {
	denied map[permission.Resource]bool
}

func (f fakeReads) MayRead(_ context.Context, resource permission.Resource) bool {
	return !f.denied[resource]
}

type fixture struct {
	svc       *Service
	db        *fakeDB
	tool      *previewingTool
	versions  *fakeVersions
	labeler   *fakeLabeler
	baselines *fakeBaselines
	decisions *fakeDecisions
	steps     *fakeSteps
	checker   *fakeChecker
	org, bu   pulid.ID
	customer  pulid.ID
}

func newFixture(extra ...agenttoolports.AgentTool) *fixture {
	f := &fixture{
		db:        &fakeDB{},
		versions:  &fakeVersions{versions: map[pulid.ID]int64{}, missing: map[pulid.ID]bool{}},
		baselines: &fakeBaselines{},
		decisions: &fakeDecisions{},
		steps:     &fakeSteps{},
		checker:   &fakeChecker{},
		org:       pulid.MustNew("org_"),
		bu:        pulid.MustNew("bu_"),
		customer:  pulid.MustNew("cus_"),
	}
	f.tool = &previewingTool{
		baseTool: baseTool{name: "set_shipment_status", resource: permission.ResourceShipment},
		state: &shipmentState{
			ID:         pulid.MustNew("shp_"),
			ProNumber:  "PRO-100",
			Status:     "New",
			CustomerID: f.customer,
			RateAmount: decimal.RequireFromString("100"),
			Version:    3,
		},
	}
	f.versions.versions[f.tool.state.ID] = 3
	f.labeler = &fakeLabeler{labels: agentruntimeports.RecordLabels{
		permission.ResourceCustomer: {f.customer: "Acme Foods"},
	}}

	tools := append([]agenttoolports.AgentTool{f.tool}, extra...)
	f.svc = &Service{
		l:           zap.NewNop(),
		db:          f.db,
		tools:       fakeRegistry{tools: tools},
		versions:    f.versions,
		labeler:     f.labeler,
		baselines:   f.baselines,
		decisions:   f.decisions,
		steps:       f.steps,
		checker:     f.checker,
		now:         func() int64 { return 1767225600 },
		readTimeout: time.Second,
		fileTimeout: time.Second,
	}

	return f
}

func (f *fixture) proposal(params map[string]any) *agent.AgentProposal {
	return &agent.AgentProposal{
		ID:             pulid.MustNew("ap_"),
		OrganizationID: f.org,
		BusinessUnitID: f.bu,
		RunID:          pulid.MustNew("ar_"),
		ToolName:       f.tool.name,
		ToolParams:     params,
		Status:         agent.ProposalStatusPending,
		TargetResource: string(permission.ResourceShipment),
		TargetID:       f.tool.state.ID,
		TargetVersion:  3,
	}
}

func (f *fixture) cancelParams() map[string]any {
	return map[string]any{"shipmentId": f.tool.state.ID.String(), "status": "Cancelled"}
}

func (f *fixture) viewer() *agentruntimeports.PreviewViewer {
	return &agentruntimeports.PreviewViewer{
		Actor: &permissionports.RequestActor{
			PrincipalType:  permissionports.PrincipalTypeUser,
			PrincipalID:    pulid.MustNew("usr_"),
			UserID:         pulid.MustNew("usr_"),
			OrganizationID: f.org,
			BusinessUnitID: f.bu,
		},
		Ceilings: fakeCeilings{},
		Reads:    fakeReads{},
	}
}
