package agentrunservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/temporaljobs/agentjobs"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

type fakeAgentRunRepo struct {
	repositories.AgentRunRepository
	create func(ctx context.Context, entity *agent.AgentRun) (*agent.AgentRun, error)
	update func(ctx context.Context, entity *agent.AgentRun) (*agent.AgentRun, error)
}

func (f *fakeAgentRunRepo) Create(
	ctx context.Context,
	entity *agent.AgentRun,
) (*agent.AgentRun, error) {
	return f.create(ctx, entity)
}

func (f *fakeAgentRunRepo) Update(
	ctx context.Context,
	entity *agent.AgentRun,
) (*agent.AgentRun, error) {
	return f.update(ctx, entity)
}

type fakeDefinitionRepo struct {
	repositories.AgentDefinitionRepository
	byID  func(ctx context.Context, req repositories.GetAgentDefinitionByIDRequest) (*agentdefinition.Definition, error)
	byKey func(ctx context.Context, req repositories.GetAgentDefinitionBySystemKeyRequest) (*agentdefinition.Definition, error)
}

func (f *fakeDefinitionRepo) GetByID(
	ctx context.Context,
	req repositories.GetAgentDefinitionByIDRequest,
) (*agentdefinition.Definition, error) {
	return f.byID(ctx, req)
}

func (f *fakeDefinitionRepo) GetBySystemKey(
	ctx context.Context,
	req repositories.GetAgentDefinitionBySystemKeyRequest,
) (*agentdefinition.Definition, error) {
	return f.byKey(ctx, req)
}

type fakeWorkflowStarter struct {
	serviceports.WorkflowStarter
	enabled       bool
	startWorkflow func(
		ctx context.Context,
		options client.StartWorkflowOptions,
		workflow any,
		args ...any,
	) (client.WorkflowRun, error)
}

func (f *fakeWorkflowStarter) Enabled() bool { return f.enabled }

func (f *fakeWorkflowStarter) StartWorkflow(
	ctx context.Context,
	options client.StartWorkflowOptions,
	workflow any,
	args ...any,
) (client.WorkflowRun, error) {
	return f.startWorkflow(ctx, options, workflow, args...)
}

type fakeAuditService struct {
	serviceports.AuditService
	logged []*serviceports.LogActionParams
}

func (f *fakeAuditService) LogAction(
	params *serviceports.LogActionParams,
	_ ...serviceports.LogOption,
) error {
	f.logged = append(f.logged, params)
	return nil
}

var testTenant = pagination.TenantInfo{
	OrgID: pulid.MustNew("org_"),
	BuID:  pulid.MustNew("bu_"),
}

func definitionFixture(enabled bool) *agentdefinition.Definition {
	return &agentdefinition.Definition{
		ID:             pulid.MustNew("agd_"),
		OrganizationID: testTenant.OrgID,
		BusinessUnitID: testTenant.BuID,
		Name:           "Billing exceptions",
		SystemKey:      SystemKeyBillingException,
		Enabled:        enabled,
	}
}

func definitionRepoFor(def *agentdefinition.Definition) *fakeDefinitionRepo {
	return &fakeDefinitionRepo{
		byID: func(_ context.Context, req repositories.GetAgentDefinitionByIDRequest) (*agentdefinition.Definition, error) {
			if req.ID != def.ID || req.TenantInfo.OrgID != def.OrganizationID {
				return nil, errortypes.NewNotFoundError("Agent definition not found")
			}
			return def, nil
		},
		byKey: func(_ context.Context, req repositories.GetAgentDefinitionBySystemKeyRequest) (*agentdefinition.Definition, error) {
			if req.SystemKey != def.SystemKey || req.TenantInfo.OrgID != def.OrganizationID {
				return nil, errortypes.NewNotFoundError("Agent definition not found")
			}
			return def, nil
		},
	}
}

func startRequest(def *agentdefinition.Definition) *serviceports.StartAgentRunForDefinitionRequest {
	return &serviceports.StartAgentRunForDefinitionRequest{
		DefinitionID: def.ID,
		SubjectType:  agent.SubjectBillingQueueItem,
		SubjectID:    pulid.MustNew("bqi_"),
		Trigger:      agent.RunTriggerEvent,
		TenantInfo:   testTenant,
	}
}

func TestStartForDefinitionRejectsDisabledDefinition(t *testing.T) {
	t.Parallel()

	createCalled := false
	startCalled := false
	def := definitionFixture(false)
	svc := &Service{
		l: zap.NewNop(),
		repo: &fakeAgentRunRepo{
			create: func(_ context.Context, entity *agent.AgentRun) (*agent.AgentRun, error) {
				createCalled = true
				return entity, nil
			},
		},
		definitions: definitionRepoFor(def),
		workflows: &fakeWorkflowStarter{
			enabled: true,
			startWorkflow: func(
				context.Context,
				client.StartWorkflowOptions,
				any,
				...any,
			) (client.WorkflowRun, error) {
				startCalled = true
				return nil, nil
			},
		},
		audit: &fakeAuditService{},
	}

	run, err := svc.StartForDefinition(t.Context(), startRequest(def), nil)
	if err == nil {
		t.Fatalf("expected error when the definition is disabled")
	}
	if run != nil {
		t.Fatalf("expected no run, got %+v", run)
	}

	var businessErr *errortypes.BusinessError
	if !errors.As(err, &businessErr) {
		t.Fatalf("expected BusinessError, got %T: %v", err, err)
	}
	if createCalled {
		t.Fatalf("expected no run row when the definition is disabled")
	}
	if startCalled {
		t.Fatalf("expected no workflow when the definition is disabled")
	}
}

func TestStartForDefinitionLaunchesWorkflow(t *testing.T) {
	t.Parallel()

	def := definitionFixture(true)
	var started client.StartWorkflowOptions
	var startedName any
	var payload *agentjobs.AgentRunPayload
	var createdAfterStart bool
	audit := &fakeAuditService{}
	svc := &Service{
		l:           zap.NewNop(),
		validator:   NewValidator(ValidatorParams{}),
		definitions: definitionRepoFor(def),
		repo: &fakeAgentRunRepo{
			create: func(_ context.Context, entity *agent.AgentRun) (*agent.AgentRun, error) {
				createdAfterStart = started.ID != ""
				return entity, nil
			},
			update: func(_ context.Context, entity *agent.AgentRun) (*agent.AgentRun, error) {
				return entity, nil
			},
		},
		workflows: &fakeWorkflowStarter{
			enabled: true,
			startWorkflow: func(
				_ context.Context,
				options client.StartWorkflowOptions,
				name any,
				args ...any,
			) (client.WorkflowRun, error) {
				started = options
				startedName = name
				if len(args) == 1 {
					payload, _ = args[0].(*agentjobs.AgentRunPayload)
				}
				return nil, nil
			},
		},
		audit: audit,
	}

	req := startRequest(def)
	run, err := svc.StartForDefinition(t.Context(), req, nil)
	if err != nil {
		t.Fatalf("expected start to succeed, got %v", err)
	}
	if run == nil {
		t.Fatalf("expected a run to be returned")
	}
	if !createdAfterStart {
		t.Fatalf("expected the run to be recorded only once its workflow had started")
	}
	runID := run.ID

	// An event run is keyed by its subject, so a second event about a
	// subject whose run is still open is refused by Temporal.
	wantWorkflowID := workflowIDPrefix + def.ID.String() + "-subject-" + req.SubjectID.String()
	if started.ID != wantWorkflowID {
		t.Fatalf("expected workflow id %s, got %s", wantWorkflowID, started.ID)
	}
	if started.WorkflowIDConflictPolicy != enums.WORKFLOW_ID_CONFLICT_POLICY_FAIL {
		t.Fatalf(
			"expected a second open run to be refused, got %v",
			started.WorkflowIDConflictPolicy,
		)
	}
	// A run belongs on the background queue, not the one a person's chat turn
	// waits in: a research run that takes ten minutes must not hold a slot
	// somebody is watching for.
	if started.TaskQueue != temporaltype.TaskQueueAgentBackground.String() {
		t.Fatalf("expected the agent background task queue, got %s", started.TaskQueue)
	}
	if started.WorkflowIDReusePolicy != enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE {
		t.Fatalf("expected a subject to take a new run once its last one ended, got %v",
			started.WorkflowIDReusePolicy)
	}
	if startedName != agentjobs.AgentRunWorkflowName {
		t.Fatalf("expected workflow %s, got %v", agentjobs.AgentRunWorkflowName, startedName)
	}
	if payload == nil {
		t.Fatalf("expected an agent run payload")
	}
	if payload.RunID != runID || payload.DefinitionID != def.ID {
		t.Fatalf("expected payload for run %s / definition %s, got %+v", runID, def.ID, payload)
	}
	if payload.OrganizationID != testTenant.OrgID || payload.BusinessUnitID != testTenant.BuID {
		t.Fatalf("expected payload tenant to come from the request, got %+v", payload.BasePayload)
	}
	if run.WorkflowID != wantWorkflowID {
		t.Fatalf("expected run workflow id %s, got %s", wantWorkflowID, run.WorkflowID)
	}
	if run.AgentDefinitionID != def.ID {
		t.Fatalf("expected run linked to definition %s, got %s", def.ID, run.AgentDefinitionID)
	}
	if run.AgentType != agent.TypeBillingException {
		t.Fatalf("expected billing exception agent type, got %s", run.AgentType)
	}
	if run.Trigger != agent.RunTriggerEvent {
		t.Fatalf("expected event trigger, got %s", run.Trigger)
	}
	if len(audit.logged) != 1 {
		t.Fatalf("expected one audit entry, got %d", len(audit.logged))
	}
}

func TestStartForDefinitionUsesSlotWorkflowID(t *testing.T) {
	t.Parallel()

	def := definitionFixture(true)
	def.SystemKey = ""
	var started client.StartWorkflowOptions
	svc := &Service{
		l:           zap.NewNop(),
		validator:   NewValidator(ValidatorParams{}),
		definitions: definitionRepoFor(def),
		repo: &fakeAgentRunRepo{
			create: func(_ context.Context, entity *agent.AgentRun) (*agent.AgentRun, error) {
				return entity, nil
			},
			update: func(_ context.Context, entity *agent.AgentRun) (*agent.AgentRun, error) {
				return entity, nil
			},
		},
		workflows: &fakeWorkflowStarter{
			enabled: true,
			startWorkflow: func(
				_ context.Context,
				options client.StartWorkflowOptions,
				_ any,
				_ ...any,
			) (client.WorkflowRun, error) {
				started = options
				return nil, nil
			},
		},
		audit: &fakeAuditService{},
	}

	req := &serviceports.StartAgentRunForDefinitionRequest{
		SystemKey:  def.SystemKey,
		Trigger:    agent.RunTriggerScheduled,
		Slot:       1_700_000_000,
		TenantInfo: testTenant,
	}
	req.DefinitionID = def.ID

	run, err := svc.StartForDefinition(t.Context(), req, nil)
	if err != nil {
		t.Fatalf("expected start to succeed, got %v", err)
	}
	want := workflowIDPrefix + def.ID.String() + "-1700000000"
	if started.ID != want {
		t.Fatalf("expected slot workflow id %s, got %s", want, started.ID)
	}
	if run.SubjectType != agent.SubjectOrganization || run.SubjectID != testTenant.OrgID {
		t.Fatalf(
			"expected organization subject by default, got %s %s",
			run.SubjectType,
			run.SubjectID,
		)
	}
	if run.AgentType != agent.TypeGeneral {
		t.Fatalf("expected general agent type for a custom definition, got %s", run.AgentType)
	}
}

// A run is recorded only once its workflow has started, so a start that fails
// leaves nothing behind for anybody to wonder about.
func TestStartForDefinitionRecordsNothingWhenWorkflowStartFails(t *testing.T) {
	t.Parallel()

	def := definitionFixture(true)
	created := false
	svc := &Service{
		l:           zap.NewNop(),
		validator:   NewValidator(ValidatorParams{}),
		definitions: definitionRepoFor(def),
		repo: &fakeAgentRunRepo{
			create: func(_ context.Context, entity *agent.AgentRun) (*agent.AgentRun, error) {
				created = true
				return entity, nil
			},
		},
		workflows: &fakeWorkflowStarter{
			enabled: true,
			startWorkflow: func(
				context.Context,
				client.StartWorkflowOptions,
				any,
				...any,
			) (client.WorkflowRun, error) {
				return nil, errors.New("temporal down")
			},
		},
		audit: &fakeAuditService{},
	}

	if _, err := svc.StartForDefinition(t.Context(), startRequest(def), nil); err == nil {
		t.Fatalf("expected the workflow start error to surface")
	}
	if created {
		t.Fatalf("expected no run to be recorded")
	}
}

// Two events about one subject, arriving together, start one run. The second
// is refused by Temporal on the run's workflow id, where a count of open runs
// read by both would have let both through.
func TestStartForDefinitionRefusesASecondRunForAnOpenSubject(t *testing.T) {
	t.Parallel()

	def := definitionFixture(true)
	created := false
	svc := &Service{
		l:           zap.NewNop(),
		validator:   NewValidator(ValidatorParams{}),
		definitions: definitionRepoFor(def),
		repo: &fakeAgentRunRepo{
			create: func(_ context.Context, entity *agent.AgentRun) (*agent.AgentRun, error) {
				created = true
				return entity, nil
			},
		},
		workflows: &fakeWorkflowStarter{
			enabled: true,
			startWorkflow: func(
				context.Context,
				client.StartWorkflowOptions,
				any,
				...any,
			) (client.WorkflowRun, error) {
				return nil, serviceerror.NewWorkflowExecutionAlreadyStarted(
					"already started", "", "",
				)
			},
		},
		audit: &fakeAuditService{},
	}

	_, err := svc.StartForDefinition(t.Context(), startRequest(def), nil)
	if !errors.Is(err, serviceports.ErrAgentRunAlreadyOpen) {
		t.Fatalf("expected the open run to refuse the start, got %v", err)
	}
	if created {
		t.Fatalf("expected no second run to be recorded")
	}
}

func TestStartForDefinitionRejectsWhenWorkflowEngineUnavailable(t *testing.T) {
	t.Parallel()

	svc := &Service{
		l:         zap.NewNop(),
		workflows: &fakeWorkflowStarter{enabled: false},
	}

	if _, err := svc.StartForDefinition(t.Context(), startRequest(definitionFixture(true)), nil); err == nil {
		t.Fatalf("expected error when workflow engine is unavailable")
	}
}

func TestStartForDefinitionRequiresIDOrSystemKey(t *testing.T) {
	t.Parallel()

	svc := &Service{
		l:           zap.NewNop(),
		workflows:   &fakeWorkflowStarter{enabled: true},
		definitions: &fakeDefinitionRepo{},
	}

	_, err := svc.StartForDefinition(t.Context(), &serviceports.StartAgentRunForDefinitionRequest{
		TenantInfo: testTenant,
	}, nil)
	if err == nil {
		t.Fatalf("expected a validation error without an id or system key")
	}
}
