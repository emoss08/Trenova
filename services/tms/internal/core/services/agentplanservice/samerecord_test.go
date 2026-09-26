package agentplanservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordDecider decides each step the way the executor runs it: the record
// must be at the version the step expects, the pinned one unless the plan
// says an earlier step moved it, and each write moves it one version on.
type recordDecider struct {
	fakeDecider

	versions map[pulid.ID]int64
	pins     map[pulid.ID]int64
	targets  map[pulid.ID]pulid.ID
}

func (d *recordDecider) DecideWithOutcome(
	ctx context.Context,
	req *services.DecideAgentProposalRequest,
	actor *services.RequestActor,
) (*services.DecisionOutcome, error) {
	outcome, err := d.fakeDecider.DecideWithOutcome(ctx, req, actor)
	if err != nil {
		return nil, err
	}

	record := d.targets[req.ProposalID]
	expected := d.pins[req.ProposalID]
	if req.ExpectedTargetVersion != nil {
		expected = *req.ExpectedTargetVersion
	}
	if d.versions[record] != expected {
		outcome.ExecutionError = errors.New("the record changed since this was proposed")

		return outcome, nil
	}

	d.versions[record]++
	after := d.versions[record]
	outcome.ExecutedTargetVersion = &after

	return outcome, nil
}

// Two steps of one plan on the same shipment used to fail at the second:
// both were pinned at version 3 before either ran, and the first step's
// write moved the shipment to 4.
func TestDecide_TwoStepsOnOneRecordBothRun(t *testing.T) {
	t.Parallel()

	h := newHarness(t, 3, false)
	shipment, other := pulid.MustNew("shp_"), pulid.MustNew("shp_")
	decider := &recordDecider{
		versions: map[pulid.ID]int64{shipment: 3, other: 7},
		pins:     map[pulid.ID]int64{},
		targets:  map[pulid.ID]pulid.ID{},
	}
	for i, step := range h.steps.steps {
		record, pinned := shipment, int64(3)
		if i == 1 {
			record, pinned = other, 7
		}
		step.TargetResource = string(permission.ResourceShipment)
		step.TargetID = record
		step.TargetVersion = pinned
		decider.pins[step.ID] = pinned
		decider.targets[step.ID] = record
	}
	h.svc.decisions = decider

	plan, err := h.svc.Decide(t.Context(), h.req, h.actor)
	require.NoError(t, err)

	assert.Equal(t, agent.PlanStatusCompleted, plan.Status, plan.FailureError)
	assert.Equal(t, 3, plan.CompletedSteps)
	require.Len(t, decider.decided, 3)
	assert.Nil(t, decider.decided[0].ExpectedTargetVersion)
	assert.Nil(t, decider.decided[1].ExpectedTargetVersion, "another record keeps its pin")
	require.NotNil(t, decider.decided[2].ExpectedTargetVersion)
	assert.Equal(t, int64(4), *decider.decided[2].ExpectedTargetVersion,
		"the third step runs against the version the first left")
	assert.Equal(t, int64(5), decider.versions[shipment])
}

// A record changed by somebody else between the steps is still refused: only
// the plan's own earlier write is allowed for.
func TestDecide_ARecordMovedByAnotherHandStillStopsThePlan(t *testing.T) {
	t.Parallel()

	h := newHarness(t, 2, false)
	shipment := pulid.MustNew("shp_")
	decider := &recordDecider{
		versions: map[pulid.ID]int64{shipment: 3},
		pins:     map[pulid.ID]int64{},
		targets:  map[pulid.ID]pulid.ID{},
	}
	for _, step := range h.steps.steps {
		step.TargetResource = string(permission.ResourceShipment)
		step.TargetID = shipment
		step.TargetVersion = 3
		decider.pins[step.ID] = 3
		decider.targets[step.ID] = shipment
	}
	decider.versions[shipment] = 9
	h.svc.decisions = decider

	plan, err := h.svc.Decide(t.Context(), h.req, h.actor)
	require.NoError(t, err)

	assert.Equal(t, agent.PlanStatusFailed, plan.Status)
	require.NotNil(t, plan.FailedStep)
	assert.Equal(t, 1, *plan.FailedStep)
}
