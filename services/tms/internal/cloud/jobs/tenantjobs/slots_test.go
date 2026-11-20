package tenantjobs

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/cloud/ports/cloudtenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type recordingSlots struct {
	held     []kernelports.HeavyJob
	released int
	refuse   error
}

func (s *recordingSlots) Hold(_ context.Context, job kernelports.HeavyJob) (func(), error) {
	if s.refuse != nil {
		return nil, s.refuse
	}
	s.held = append(s.held, job)
	return func() { s.released++ }, nil
}

func (s *recordingSlots) TryHold(context.Context, kernelports.HeavyJob) (func(), bool) {
	return func() {}, true
}

type countingPurge struct {
	cloudtenantports.Purge
	purged  int
	deleted int
	beats   bool
}

func (p *countingPurge) PurgeRows(
	_ context.Context,
	req *cloudtenantports.PurgeRowsRequest,
) (*cloudtenantports.PurgeRowsResult, error) {
	p.purged++
	p.beats = req.Beat != nil
	return &cloudtenantports.PurgeRowsResult{Complete: true}, nil
}

func (p *countingPurge) DeleteOrganization(
	context.Context,
	dbscope.Tenant,
) (*cloudtenantports.DeleteResult, error) {
	p.deleted++
	return &cloudtenantports.DeleteResult{Deleted: true}, nil
}

func TestHeavyRemovalStepsHoldARemovalSlot(t *testing.T) {
	t.Parallel()
	slots := &recordingSlots{}
	purge := &countingPurge{}
	a := NewActivities(ActivitiesParams{Purge: purge, Slots: slots, Logger: zap.NewNop()})

	_, err := a.PurgeRowsActivity(t.Context(), payload())
	require.NoError(t, err)
	_, err = a.DeleteOrganizationActivity(t.Context(), payload())
	require.NoError(t, err)

	assert.Equal(
		t,
		[]kernelports.HeavyJob{kernelports.HeavyJobRemoval, kernelports.HeavyJobRemoval},
		slots.held,
	)
	assert.Equal(t, 2, slots.released)
	assert.True(t, purge.beats)
}

func TestARemovalStepWithNoSlotDeletesNothing(t *testing.T) {
	t.Parallel()
	busy := errors.New("busy")
	purge := &countingPurge{}
	a := NewActivities(ActivitiesParams{
		Purge:  purge,
		Slots:  &recordingSlots{refuse: busy},
		Logger: zap.NewNop(),
	})

	_, err := a.PurgeRowsActivity(t.Context(), payload())
	require.ErrorIs(t, err, busy)
	_, err = a.PurgeStoresActivity(t.Context(), payload())
	require.ErrorIs(t, err, busy)
	assert.Zero(t, purge.purged)
}
