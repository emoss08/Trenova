package telematicsservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/telematics"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/telematicsports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type hosLinkRepo struct {
	telematicsports.TelematicsRepository
	externalIDs map[pulid.ID]string
	states      map[pulid.ID]*telematics.WorkerHOSState
	open        []*telematics.WorkerHOSLog
	mappingAsks int
}

func (r *hosLinkRepo) GetWorkerExternalID(
	_ context.Context,
	_ pagination.TenantInfo,
	workerID pulid.ID,
) (string, error) {
	r.mappingAsks++
	return r.externalIDs[workerID], nil
}

func (r *hosLinkRepo) ListWorkerMappings(
	context.Context,
	pagination.TenantInfo,
) ([]telematicsports.WorkerTelematicsMapping, error) {
	panic("a single worker's link never reads every mapping in the tenant")
}

func (r *hosLinkRepo) GetWorkerHOSState(
	_ context.Context,
	req telematicsports.GetWorkerHOSStateRequest,
) (*telematics.WorkerHOSState, error) {
	state, ok := r.states[req.WorkerID]
	if !ok {
		return nil, errortypes.NewNotFoundError("no hos state")
	}
	copied := *state
	return &copied, nil
}

func (r *hosLinkRepo) ListOpenWorkerHOSLogs(
	_ context.Context,
	_ pagination.TenantInfo,
	workerIDs []pulid.ID,
) ([]*telematics.WorkerHOSLog, error) {
	wanted := make(map[pulid.ID]bool, len(workerIDs))
	for _, id := range workerIDs {
		wanted[id] = true
	}
	out := make([]*telematics.WorkerHOSLog, 0, len(r.open))
	for _, segment := range r.open {
		if wanted[segment.WorkerID] {
			out = append(out, segment)
		}
	}
	return out, nil
}

type connectionFactory struct {
	fixedProviderFactory
	connected bool
}

func (f connectionFactory) ConnectedType(
	context.Context,
	pagination.TenantInfo,
) (integration.Type, bool, error) {
	return integration.TypeSamsara, f.connected, nil
}

func hosLinkService(
	repo *hosLinkRepo,
	connected bool,
	gate *plantest.Gate,
) *Service {
	return New(Params{
		Repo:            repo,
		ProviderFactory: connectionFactory{connected: connected},
		Features:        gate,
		Logger:          zap.NewNop(),
	})
}

func TestWorkerHOSLink_TellsTheFourStatesApartFromStoredStateOnly(t *testing.T) {
	t.Parallel()

	tenant := plantest.Tenant()
	unlinked := pulid.MustNew("wrk_")
	silent := pulid.MustNew("wrk_")
	reporting := pulid.MustNew("wrk_")
	tractor := pulid.MustNew("tr_")
	repo := &hosLinkRepo{
		externalIDs: map[pulid.ID]string{
			unlinked:  "  ",
			silent:    "drv-silent",
			reporting: "drv-1",
		},
		states: map[pulid.ID]*telematics.WorkerHOSState{
			reporting: {
				WorkerID:            reporting,
				DutyStatus:          telematics.DutyStatusDriving,
				DriveRemainingMs:    18_000_000,
				ShiftRemainingMs:    25_000_000,
				CycleRemainingMs:    90_000_000,
				BreakRemainingMs:    7_000_000,
				RulesetCycle:        "USA 70 hour / 8 day",
				RulesetShift:        "US Interstate Property",
				RulesetJurisdiction: "US",
				CurrentTractorID:    tractor,
				RecordedAt:          1_800_000_000,
			},
		},
		open: []*telematics.WorkerHOSLog{{
			WorkerID:   reporting,
			LogStartAt: 1_799_990_000,
			DutyStatus: telematics.DutyStatusDriving,
		}},
	}

	cases := []struct {
		name      string
		workerID  pulid.ID
		connected bool
		gate      *plantest.Gate
		want      telematics.HOSLink
	}{
		{
			name:      "telematics is off the plan",
			workerID:  reporting,
			connected: true,
			gate:      plantest.Without(platformcatalog.FeatureSamsaraIntegration),
			want:      telematics.HOSLinkNotConnected,
		},
		{
			name:      "no provider is connected",
			workerID:  reporting,
			connected: false,
			gate:      plantest.AllFeatures(),
			want:      telematics.HOSLinkNotConnected,
		},
		{
			name:      "the driver has no provider id",
			workerID:  unlinked,
			connected: true,
			gate:      plantest.AllFeatures(),
			want:      telematics.HOSLinkNotLinked,
		},
		{
			name:      "linked with nothing stored",
			workerID:  silent,
			connected: true,
			gate:      plantest.AllFeatures(),
			want:      telematics.HOSLinkNoData,
		},
		{
			name:      "reporting",
			workerID:  reporting,
			connected: true,
			gate:      plantest.AllFeatures(),
			want:      telematics.HOSLinkReporting,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := hosLinkService(repo, tc.connected, tc.gate)
			link, err := svc.WorkerHOSLink(t.Context(), tenant, tc.workerID)
			require.NoError(t, err)
			assert.Equal(t, tc.want, link.Link)
			if tc.want != telematics.HOSLinkReporting {
				assert.Nil(t, link.DutyStatus)
				assert.Nil(t, link.DriveRemainingMs)
				assert.Nil(t, link.DutyStatusSince)
				assert.Nil(t, link.DriveLimitMs)
				assert.Nil(t, link.ShiftRemainingMs)
				assert.Nil(t, link.CycleLimitMs)
				assert.Nil(t, link.BreakRemainingMs)
				assert.True(t, link.CurrentTractorID.IsNil())
				return
			}
			limits := LimitsForRuleset(
				"USA 70 hour / 8 day",
				"US Interstate Property",
				"US",
			)
			assert.Equal(t, &limits.DriveMs, link.DriveLimitMs)
			assert.Equal(t, int64(25_000_000), *link.ShiftRemainingMs)
			assert.Equal(t, &limits.ShiftMs, link.ShiftLimitMs)
			assert.Equal(t, int64(90_000_000), *link.CycleRemainingMs)
			assert.Equal(t, &limits.CycleMs, link.CycleLimitMs)
			assert.Equal(t, int64(7_000_000), *link.BreakRemainingMs)
			assert.Equal(t, &limits.BreakMs, link.BreakLimitMs)
			assert.Equal(t, tractor, link.CurrentTractorID)
			require.NotNil(t, link.DutyStatus)
			assert.Equal(t, string(telematics.DutyStatusDriving), *link.DutyStatus)
			require.NotNil(t, link.DriveRemainingMs)
			assert.Equal(t, int64(18_000_000), *link.DriveRemainingMs)
			require.NotNil(t, link.RecordedAt)
			assert.Equal(t, int64(1_800_000_000), *link.RecordedAt)
			require.NotNil(t, link.DutyStatusSince)
			assert.Equal(t, int64(1_799_990_000), *link.DutyStatusSince)
		})
	}
}

// The start of the current status comes from the open segment only when the
// segment is that status: a stale segment never dates a status it is not.
func TestGetWorkerHOSState_DatesTheStatusFromTheMatchingOpenSegment(t *testing.T) {
	t.Parallel()

	tenant := plantest.Tenant()
	matching := pulid.MustNew("wrk_")
	stale := pulid.MustNew("wrk_")
	none := pulid.MustNew("wrk_")
	repo := &hosLinkRepo{
		states: map[pulid.ID]*telematics.WorkerHOSState{
			matching: {WorkerID: matching, DutyStatus: telematics.DutyStatusOnDuty},
			stale:    {WorkerID: stale, DutyStatus: telematics.DutyStatusDriving},
			none:     {WorkerID: none, DutyStatus: telematics.DutyStatusOffDuty},
		},
		open: []*telematics.WorkerHOSLog{
			{
				WorkerID:   matching,
				LogStartAt: 1_800_000_000,
				DutyStatus: telematics.DutyStatusOnDuty,
			},
			{WorkerID: stale, LogStartAt: 1_700_000_000, DutyStatus: telematics.DutyStatusOffDuty},
		},
	}
	svc := hosLinkService(repo, true, plantest.AllFeatures())

	state, err := svc.GetWorkerHOSState(t.Context(), tenant, matching)
	require.NoError(t, err)
	require.NotNil(t, state.DutyStatusSince)
	assert.Equal(t, int64(1_800_000_000), *state.DutyStatusSince)

	state, err = svc.GetWorkerHOSState(t.Context(), tenant, stale)
	require.NoError(t, err)
	assert.Nil(t, state.DutyStatusSince)

	state, err = svc.GetWorkerHOSState(t.Context(), tenant, none)
	require.NoError(t, err)
	assert.Nil(t, state.DutyStatusSince)
}
