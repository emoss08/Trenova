package agentdefinitionservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type inTxKey struct{}

type txConnection struct{}

func (txConnection) Transact(
	ctx context.Context,
	_ kernelports.TxOptions,
	fn func(context.Context) error,
) error {
	return fn(context.WithValue(ctx, inTxKey{}, true))
}

func inTx(ctx context.Context) bool {
	in, _ := ctx.Value(inTxKey{}).(bool)
	return in
}

type deleteLog struct {
	steps []string
}

func (l *deleteLog) record(ctx context.Context, step string) {
	if inTx(ctx) {
		step += " in tx"
	}
	l.steps = append(l.steps, step)
}

type deletableDefinitions struct {
	agentdefinitionports.AgentDefinitionRepository
	log      *deleteLog
	existing *agentdefinition.Definition
}

func (d *deletableDefinitions) GetByID(
	context.Context,
	agentdefinitionports.GetAgentDefinitionByIDRequest,
) (*agentdefinition.Definition, error) {
	return d.existing, nil
}

func (d *deletableDefinitions) Delete(
	ctx context.Context,
	_ agentdefinitionports.DeleteAgentDefinitionRequest,
) error {
	d.log.record(ctx, "delete agent")
	return nil
}

type withdrawnProposals struct {
	agentports.AgentProposalRepository
	log *deleteLog
	req agentports.ExpireAgentProposalsByDefinitionRequest
	err error
}

func (p *withdrawnProposals) ExpirePendingByDefinition(
	ctx context.Context,
	req agentports.ExpireAgentProposalsByDefinitionRequest,
) (int, error) {
	p.log.record(ctx, "withdraw proposals")
	p.req = req
	return 2, p.err
}

type withdrawnPlans struct {
	agentports.AgentPlanRepository
	log *deleteLog
	req agentports.ExpireAgentPlansByDefinitionRequest
}

func (p *withdrawnPlans) ExpirePendingByDefinition(
	ctx context.Context,
	req agentports.ExpireAgentPlansByDefinitionRequest,
) (int, error) {
	p.log.record(ctx, "withdraw plans")
	p.req = req
	return 1, nil
}

type removedSchedules struct {
	removed []pulid.ID
}

func (s *removedSchedules) Sync(context.Context, *agentdefinition.Definition) {}

func (s *removedSchedules) Remove(_ context.Context, id pulid.ID) {
	s.removed = append(s.removed, id)
}

type silentAudit struct {
	auditports.AuditService
	logged int
}

func (a *silentAudit) LogAction(*auditports.LogActionParams, ...auditports.LogOption) error {
	a.logged++
	return nil
}

type deleteFixture struct {
	svc       *Service
	log       *deleteLog
	agent     *agentdefinition.Definition
	proposals *withdrawnProposals
	plans     *withdrawnPlans
	schedules *removedSchedules
	audit     *silentAudit
}

func newDeleteFixture() *deleteFixture {
	log := &deleteLog{}
	agent := agentFixture()
	f := &deleteFixture{
		log:       log,
		agent:     agent,
		proposals: &withdrawnProposals{log: log},
		plans:     &withdrawnPlans{log: log},
		schedules: &removedSchedules{},
		audit:     &silentAudit{},
	}
	f.svc = &Service{
		l:          zap.NewNop(),
		db:         txConnection{},
		repo:       &deletableDefinitions{log: log, existing: agent},
		proposals:  f.proposals,
		agentPlans: f.plans,
		schedules:  f.schedules,
		audit:      f.audit,
	}

	return f
}

func TestDeleteWithdrawsOpenProposalsAndPlansWithTheAgent(t *testing.T) {
	t.Parallel()

	f := newDeleteFixture()
	tenantInfo := testTenant()

	err := f.svc.Delete(t.Context(), agentdefinitionports.DeleteAgentDefinitionRequest{
		ID:         f.agent.ID,
		TenantInfo: tenantInfo,
	}, &permissionports.RequestActor{})
	require.NoError(t, err)

	assert.Equal(t, []string{
		"withdraw proposals in tx",
		"withdraw plans in tx",
		"delete agent in tx",
	}, f.log.steps)
	assert.Equal(t, f.agent.ID, f.proposals.req.AgentDefinitionID)
	assert.Equal(t, tenantInfo, f.proposals.req.TenantInfo)
	assert.Equal(t, f.agent.ID, f.plans.req.AgentDefinitionID)
	assert.Equal(t, tenantInfo, f.plans.req.TenantInfo)
	assert.Equal(t, []pulid.ID{f.agent.ID}, f.schedules.removed)
	assert.Equal(t, 1, f.audit.logged)
}

func TestDeleteKeepsTheAgentWhenItsProposalsCannotBeWithdrawn(t *testing.T) {
	t.Parallel()

	f := newDeleteFixture()
	f.proposals.err = errors.New("database is down")

	err := f.svc.Delete(t.Context(), agentdefinitionports.DeleteAgentDefinitionRequest{
		ID:         f.agent.ID,
		TenantInfo: testTenant(),
	}, &permissionports.RequestActor{})

	require.Error(t, err)
	assert.Equal(t, []string{"withdraw proposals in tx"}, f.log.steps)
	assert.Empty(t, f.schedules.removed)
	assert.Zero(t, f.audit.logged)
}

func TestDeleteRefusesASystemAgentWithoutWithdrawingAnything(t *testing.T) {
	t.Parallel()

	f := newDeleteFixture()
	f.agent.SystemKey = "load_monitor"

	err := f.svc.Delete(t.Context(), agentdefinitionports.DeleteAgentDefinitionRequest{
		ID:         f.agent.ID,
		TenantInfo: testTenant(),
	}, &permissionports.RequestActor{})

	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
	assert.Empty(t, f.log.steps)
}
