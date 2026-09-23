package proposalrecorder

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

type txMarker struct{}

// stagingTx stands in for the database: writes made inside it are staged and
// only land in committed when fn returns without an error.
type stagingTx struct {
	calls      int
	staged     []string
	committed  []string
	rolledBack bool
}

func (s *stagingTx) WithTx(
	ctx context.Context,
	_ ports.TxOptions,
	fn func(context.Context, bun.Tx) error,
) error {
	s.calls++
	s.staged = nil
	if err := fn(context.WithValue(ctx, txMarker{}, s), bun.Tx{}); err != nil {
		s.staged = nil
		s.rolledBack = true

		return err
	}
	s.committed = append(s.committed, s.staged...)
	s.staged = nil

	return nil
}

func stage(ctx context.Context, row string) error {
	tx, ok := ctx.Value(txMarker{}).(*stagingTx)
	if !ok {
		return errors.New(row + " written outside the transaction")
	}
	tx.staged = append(tx.staged, row)

	return nil
}

type stagingRuns struct{}

func (stagingRuns) Create(ctx context.Context, run *agent.AgentRun) (*agent.AgentRun, error) {
	if err := stage(ctx, "run"); err != nil {
		return nil, err
	}
	run.ID = pulid.MustNew("arun_")

	return run, nil
}

type stagingPlans struct{}

func (stagingPlans) Create(ctx context.Context, plan *agent.AgentPlan) (*agent.AgentPlan, error) {
	if err := stage(ctx, "plan"); err != nil {
		return nil, err
	}
	plan.ID = pulid.MustNew("apl_")

	return plan, nil
}

type stagingProposals struct {
	failOn int
	calls  int
}

var errProposalInsert = errors.New("insert agent proposal failed")

func (s *stagingProposals) Create(
	ctx context.Context,
	proposal *agent.AgentProposal,
) (*agent.AgentProposal, error) {
	s.calls++
	if s.calls == s.failOn {
		return nil, errProposalInsert
	}
	if err := stage(ctx, "proposal"); err != nil {
		return nil, err
	}

	return proposal, nil
}

type countingActivity struct {
	serviceports.AgentActivityPublisher
	proposals int
	plans     int
}

func (a *countingActivity) ProposalChanged(
	context.Context,
	*agent.AgentProposal,
	serviceports.AuditActor,
	string,
) {
	a.proposals++
}

func (a *countingActivity) PlanChanged(
	context.Context,
	*agent.AgentPlan,
	serviceports.AuditActor,
	string,
) {
	a.plans++
}

func recordInTx(
	t *testing.T,
	tx *stagingTx,
	proposals *stagingProposals,
	activity *countingActivity,
) (*RecordResult, error) {
	t.Helper()

	svc := NewWithStores(nil, stagingRuns{}, proposals).
		WithPlans(stagingPlans{}).
		WithTransactor(tx)
	svc.activity = activity

	return svc.Record(t.Context(), &RecordRequest{
		Actor: &serviceports.RequestActor{
			OrganizationID: pulid.MustNew("org_"),
			BusinessUnitID: pulid.MustNew("bu_"),
		},
		Open: &OpenRunRequest{
			AgentType:        agent.TypeAssistantChat,
			SubjectType:      agent.SubjectAssistantThread,
			SubjectID:        pulid.MustNew("athr_"),
			PromptVersion:    "v1",
			InputContextHash: "hash",
		},
		Actions: []serviceports.PendingAction{
			pendingAction("assign_move"),
			pendingAction("notify_driver"),
			pendingAction("add_shipment_comment"),
		},
	})
}

// A plan is one decision. When the second step cannot be stored, the run,
// the plan and the first step go with it, and nobody is told something is
// waiting.
func TestRecord_LeavesNothingBehindWhenAProposalFails(t *testing.T) {
	t.Parallel()

	tx := &stagingTx{}
	activity := &countingActivity{}
	result, err := recordInTx(t, tx, &stagingProposals{failOn: 2}, activity)

	require.ErrorIs(t, err, errProposalInsert)
	assert.Nil(t, result)
	assert.Equal(t, 1, tx.calls, "the run, plan and proposals share one transaction")
	assert.True(t, tx.rolledBack)
	assert.Empty(t, tx.committed, "no run, plan or proposal outlives the failure")
	assert.Zero(t, activity.proposals, "a rolled back proposal is never announced")
	assert.Zero(t, activity.plans, "a rolled back plan is never announced")
}

func TestRecord_CommitsTheRunPlanAndProposalsTogether(t *testing.T) {
	t.Parallel()

	tx := &stagingTx{}
	activity := &countingActivity{}
	result, err := recordInTx(t, tx, &stagingProposals{}, activity)

	require.NoError(t, err)
	require.NotNil(t, result.Plan)
	require.Len(t, result.Proposals, 3)
	assert.Equal(t, 1, tx.calls)
	assert.False(t, tx.rolledBack)
	assert.Equal(t, []string{"run", "plan", "proposal", "proposal", "proposal"}, tx.committed)
	assert.Equal(t, 3, activity.proposals)
	assert.Equal(t, 1, activity.plans)
}
