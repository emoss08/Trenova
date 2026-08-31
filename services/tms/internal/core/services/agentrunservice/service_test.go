package agentrunservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
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

type fakeAgentControlService struct {
	serviceports.AgentControlService
	get func(ctx context.Context, tenantInfo pagination.TenantInfo) (*tenant.AgentControl, error)
}

func (f *fakeAgentControlService) Get(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*tenant.AgentControl, error) {
	return f.get(ctx, tenantInfo)
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

func startRequest() *serviceports.StartAgentRunRequest {
	return &serviceports.StartAgentRunRequest{
		AgentType:   agent.TypeBillingException,
		SubjectType: agent.SubjectBillingQueueItem,
		SubjectID:   pulid.MustNew("bqi_"),
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
	}
}

func TestStartRejectsWhenBillingAgentDisabled(t *testing.T) {
	t.Parallel()

	createCalled := false
	startCalled := false
	svc := &Service{
		l: zap.NewNop(),
		repo: &fakeAgentRunRepo{
			create: func(_ context.Context, entity *agent.AgentRun) (*agent.AgentRun, error) {
				createCalled = true
				return entity, nil
			},
		},
		control: &fakeAgentControlService{
			get: func(context.Context, pagination.TenantInfo) (*tenant.AgentControl, error) {
				return &tenant.AgentControl{BillingAgentEnabled: false}, nil
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
				startCalled = true
				return nil, nil
			},
		},
		audit: &fakeAuditService{},
	}

	run, err := svc.Start(t.Context(), startRequest(), nil)
	if err == nil {
		t.Fatalf("expected error when billing agent is disabled")
	}
	if run != nil {
		t.Fatalf("expected no run when billing agent is disabled, got %+v", run)
	}

	var businessErr *errortypes.BusinessError
	if !errors.As(err, &businessErr) {
		t.Fatalf("expected BusinessError, got %T: %v", err, err)
	}
	if createCalled {
		t.Fatalf("expected no run row to be created when billing agent is disabled")
	}
	if startCalled {
		t.Fatalf("expected no workflow to start when billing agent is disabled")
	}
}

func TestStartLaunchesWorkflowWhenBillingAgentEnabled(t *testing.T) {
	t.Parallel()

	runID := pulid.MustNew("ar_")
	var startedWorkflowID string
	audit := &fakeAuditService{}
	svc := &Service{
		l: zap.NewNop(),
		repo: &fakeAgentRunRepo{
			create: func(_ context.Context, entity *agent.AgentRun) (*agent.AgentRun, error) {
				entity.ID = runID
				return entity, nil
			},
			update: func(_ context.Context, entity *agent.AgentRun) (*agent.AgentRun, error) {
				return entity, nil
			},
		},
		control: &fakeAgentControlService{
			get: func(context.Context, pagination.TenantInfo) (*tenant.AgentControl, error) {
				return &tenant.AgentControl{
					BillingAgentEnabled:    true,
					ShadowMode:             true,
					DecisionTimeoutSeconds: 300,
				}, nil
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
				startedWorkflowID = options.ID
				return nil, nil
			},
		},
		audit: audit,
	}

	run, err := svc.Start(t.Context(), startRequest(), nil)
	if err != nil {
		t.Fatalf("expected start to succeed when billing agent is enabled, got %v", err)
	}
	if run == nil {
		t.Fatalf("expected a run to be returned")
	}

	wantWorkflowID := workflowIDPrefix + runID.String()
	if startedWorkflowID != wantWorkflowID {
		t.Fatalf("expected workflow id %s, got %s", wantWorkflowID, startedWorkflowID)
	}
	if run.WorkflowID != wantWorkflowID {
		t.Fatalf("expected run workflow id %s, got %s", wantWorkflowID, run.WorkflowID)
	}
	if len(audit.logged) != 1 {
		t.Fatalf("expected one audit entry, got %d", len(audit.logged))
	}
}

func TestStartRejectsWhenWorkflowEngineUnavailable(t *testing.T) {
	t.Parallel()

	svc := &Service{
		l:         zap.NewNop(),
		workflows: &fakeWorkflowStarter{enabled: false},
	}

	if _, err := svc.Start(t.Context(), startRequest(), nil); err == nil {
		t.Fatalf("expected error when workflow engine is unavailable")
	}
}
