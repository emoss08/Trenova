package agentdefinitionservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type inTxKey struct{}

type txConnection struct {
	dbtest.NopConnection
}

func (txConnection) WithTx(
	ctx context.Context,
	_ ports.TxOptions,
	fn func(context.Context, bun.Tx) error,
) error {
	return fn(context.WithValue(ctx, inTxKey{}, true), bun.Tx{})
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
	repositories.AgentDefinitionRepository
	log      *deleteLog
	existing *agentdefinition.Definition
}

func (d *deletableDefinitions) GetByID(
	context.Context,
	repositories.GetAgentDefinitionByIDRequest,
) (*agentdefinition.Definition, error) {
	return d.existing, nil
}

func (d *deletableDefinitions) Delete(
	ctx context.Context,
	_ repositories.DeleteAgentDefinitionRequest,
) error {
	d.log.record(ctx, "delete agent")
	return nil
}

type withdrawnProposals struct {
	repositories.AgentProposalRepository
	log *deleteLog
	req repositories.ExpireAgentProposalsByDefinitionRequest
	err error
}

func (p *withdrawnProposals) ExpirePendingByDefinition(
	ctx context.Context,
	req repositories.ExpireAgentProposalsByDefinitionRequest,
) (int, error) {
	p.log.record(ctx, "withdraw proposals")
	p.req = req
	return 2, p.err
}

type withdrawnPlans struct {
	repositories.AgentPlanRepository
	log *deleteLog
	req repositories.ExpireAgentPlansByDefinitionRequest
}

func (p *withdrawnPlans) ExpirePendingByDefinition(
	ctx context.Context,
	req repositories.ExpireAgentPlansByDefinitionRequest,
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
	serviceports.AuditService
	logged int
}

func (a *silentAudit) LogAction(*serviceports.LogActionParams, ...serviceports.LogOption) error {
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

	err := f.svc.Delete(t.Context(), repositories.DeleteAgentDefinitionRequest{
		ID:         f.agent.ID,
		TenantInfo: tenantInfo,
	}, &serviceports.RequestActor{})
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

	err := f.svc.Delete(t.Context(), repositories.DeleteAgentDefinitionRequest{
		ID:         f.agent.ID,
		TenantInfo: testTenant(),
	}, &serviceports.RequestActor{})

	require.Error(t, err)
	assert.Equal(t, []string{"withdraw proposals in tx"}, f.log.steps)
	assert.Empty(t, f.schedules.removed)
	assert.Zero(t, f.audit.logged)
}

func TestDeleteRefusesASystemAgentWithoutWithdrawingAnything(t *testing.T) {
	t.Parallel()

	f := newDeleteFixture()
	f.agent.SystemKey = "load_monitor"

	err := f.svc.Delete(t.Context(), repositories.DeleteAgentDefinitionRequest{
		ID:         f.agent.ID,
		TenantInfo: testTenant(),
	}, &serviceports.RequestActor{})

	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
	assert.Empty(t, f.log.steps)
}
