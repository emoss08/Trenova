package carrierintelservice

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeControlRepo struct {
	repositories.CarrierIntelControlRepository
	control *carrierintel.CarrierIntelControl
}

func (f *fakeControlRepo) GetOrCreate(
	context.Context,
	pagination.TenantInfo,
) (*carrierintel.CarrierIntelControl, error) {
	return f.control, nil
}

type fakeSnapshotRepo struct {
	repositories.CarrierIntelSnapshotRepository
	byCarrier map[pulid.ID]*carrierintel.CarrierIntelSnapshot
}

func (f *fakeSnapshotRepo) GetCurrentByCarrierIDs(
	_ context.Context,
	_ pagination.TenantInfo,
	carrierIDs []pulid.ID,
) ([]*carrierintel.CarrierIntelSnapshot, error) {
	out := make([]*carrierintel.CarrierIntelSnapshot, 0, len(carrierIDs))
	for _, id := range carrierIDs {
		if snap, ok := f.byCarrier[id]; ok {
			out = append(out, snap)
		}
	}
	return out, nil
}

type fakeOverrideRepo struct {
	repositories.CarrierIntelOverrideRepository
	overrides []*carrierintel.CarrierIntelOverride
}

func (f *fakeOverrideRepo) ListActiveByCarrierIDs(
	_ context.Context,
	_ pagination.TenantInfo,
	carrierIDs []pulid.ID,
	now int64,
) ([]*carrierintel.CarrierIntelOverride, error) {
	wanted := make(map[pulid.ID]bool, len(carrierIDs))
	for _, id := range carrierIDs {
		wanted[id] = true
	}
	out := make([]*carrierintel.CarrierIntelOverride, 0, len(f.overrides))
	for _, o := range f.overrides {
		if wanted[o.CarrierID] && o.IsActive(now) {
			out = append(out, o)
		}
	}
	return out, nil
}

type fakeSubjectRepo struct {
	repositories.CarrierIntelSubjectRepository
	requested [][]pulid.ID
}

func (f *fakeSubjectRepo) ListCarrierSubjects(
	_ context.Context,
	req *repositories.ListCarrierIntelSubjectsRequest,
) ([]repositories.CarrierIntelSubject, error) {
	f.requested = append(f.requested, req.CarrierIDs)
	out := make([]repositories.CarrierIntelSubject, 0, len(req.CarrierIDs))
	for _, id := range req.CarrierIDs {
		out = append(out, repositories.CarrierIntelSubject{
			SubjectType: carrierintel.SubjectTypeCarrier,
			SubjectID:   id.String(),
			CarrierID:   id,
			DOTNumber:   "818175",
		})
	}
	return out, nil
}

type gateHarness struct {
	*harness
	snapshots *fakeSnapshotRepo
	overrides *fakeOverrideRepo
	subjects  *fakeSubjectRepo
}

func newGateHarness(t *testing.T) *gateHarness {
	t.Helper()
	base := newHarness(t, newCarrierOKConnector())
	g := &gateHarness{
		harness: base,
		snapshots: &fakeSnapshotRepo{
			byCarrier: make(map[pulid.ID]*carrierintel.CarrierIntelSnapshot),
		},
		overrides: &fakeOverrideRepo{},
		subjects:  &fakeSubjectRepo{},
	}
	base.svc.controlRepo = &fakeControlRepo{control: base.control}
	base.svc.snapshotRepo = g.snapshots
	base.svc.overrideRepo = g.overrides
	base.svc.subjectRepo = g.subjects
	base.setPrimary(integration.TypeCarrierOK)
	return g
}

func outOfServiceProfile() *carrierintel.Profile {
	profile := &carrierintel.Profile{
		Identity: &carrierintel.Identity{DOTNumber: "818175", USDOTStatus: "ACTIVE"},
		Safety:   &carrierintel.Safety{OutOfServiceOrder: new(true)},
	}
	profile.NormalizeCoverage()
	return profile
}

func (g *gateHarness) addSnapshot(
	carrierID pulid.ID,
	ageSeconds int64,
) *carrierintel.CarrierIntelSnapshot {
	profile := outOfServiceProfile()
	snap := &carrierintel.CarrierIntelSnapshot{
		CarrierID:     carrierID,
		SubjectType:   carrierintel.SubjectTypeCarrier,
		SubjectID:     carrierID.String(),
		Provider:      integration.TypeCarrierOK,
		Depth:         carrierintel.LookupDepthFull,
		Profile:       profile,
		PolicyVersion: g.control.PolicyVersion,
		FetchedAt:     g.now - ageSeconds,
		Findings: carrierintel.EvaluateFindings(&carrierintel.EvaluateInput{
			Profile:  profile,
			Now:      g.now,
			Subject:  carrierintel.SubjectTypeCarrier,
			Settings: g.control.Rules,
		}),
	}
	g.snapshots.byCarrier[carrierID] = snap
	return snap
}

func gateFinding(gate *carrier.IntelGate, code carrierintel.RuleCode) *carrier.IntelFinding {
	for idx := range gate.Findings {
		if gate.Findings[idx].Code == code.String() {
			return &gate.Findings[idx]
		}
	}
	return nil
}

func TestGateFor(t *testing.T) {
	t.Parallel()

	t.Run("no primary provider leaves carriers ungated", func(t *testing.T) {
		t.Parallel()
		g := newGateHarness(t)
		g.control.PrimaryProvider = nil

		gates, err := g.svc.GateFor(t.Context(), g.tenant, []pulid.ID{pulid.MustNew("car_")})
		require.NoError(t, err)
		assert.Empty(t, gates)
	})

	t.Run("a carrier never looked up is stale past the hard max age", func(t *testing.T) {
		t.Parallel()
		g := newGateHarness(t)
		carrierID := pulid.MustNew("car_")

		gates, err := g.svc.GateFor(t.Context(), g.tenant, []pulid.ID{carrierID})
		require.NoError(t, err)
		gate := gates[carrierID]
		require.NotNil(t, gate)
		assert.True(t, gate.Evaluated)
		assert.True(t, gate.Stale)
		assert.True(t, gate.PastHardMaxAge)
		assert.Empty(t, gate.Findings)
		assert.Equal(t, integration.TypeCarrierOK.String(), gate.Provider)
	})

	t.Run("current snapshots gate from stored findings without recomputing", func(t *testing.T) {
		t.Parallel()
		g := newGateHarness(t)
		carrierID := pulid.MustNew("car_")
		g.addSnapshot(carrierID, 3600)

		gates, err := g.svc.GateFor(t.Context(), g.tenant, []pulid.ID{carrierID})
		require.NoError(t, err)
		gate := gates[carrierID]
		assert.False(t, gate.Stale)
		assert.False(t, gate.PastHardMaxAge)
		assert.Empty(t, g.subjects.requested)

		finding := gateFinding(gate, carrierintel.RuleSafetyOOSOrder)
		require.NotNil(t, finding)
		assert.Equal(t, carrier.IntelAction(carrierintel.RuleActionBlock), finding.Action)
		for _, f := range gate.Findings {
			assert.NotEqual(t, carrier.IntelAction(carrierintel.RuleActionOff), f.Action)
		}
	})

	t.Run("freshness is measured against the pre-tender and hard limits", func(t *testing.T) {
		t.Parallel()
		g := newGateHarness(t)
		stale := pulid.MustNew("car_")
		expired := pulid.MustNew("car_")
		g.addSnapshot(stale, 2*timeutils.SecondsPerDay)
		g.addSnapshot(expired, 8*timeutils.SecondsPerDay)

		gates, err := g.svc.GateFor(t.Context(), g.tenant, []pulid.ID{stale, expired})
		require.NoError(t, err)
		assert.True(t, gates[stale].Stale)
		assert.False(t, gates[stale].PastHardMaxAge)
		assert.True(t, gates[expired].Stale)
		assert.True(t, gates[expired].PastHardMaxAge)
	})

	t.Run("a confirmation refreshes freshness without a new fetch", func(t *testing.T) {
		t.Parallel()
		g := newGateHarness(t)
		carrierID := pulid.MustNew("car_")
		snap := g.addSnapshot(carrierID, 8*timeutils.SecondsPerDay)
		snap.ConfirmedAt = new(g.now - 60)

		gates, err := g.svc.GateFor(t.Context(), g.tenant, []pulid.ID{carrierID})
		require.NoError(t, err)
		assert.False(t, gates[carrierID].Stale)
		assert.Equal(t, g.now-60, gates[carrierID].FetchedAt)
	})

	t.Run("a policy change re-evaluates stored profiles", func(t *testing.T) {
		t.Parallel()
		g := newGateHarness(t)
		carrierID := pulid.MustNew("car_")
		g.addSnapshot(carrierID, 3600)

		g.control.Rules = carrierintel.RuleSettings{
			carrierintel.RuleSafetyOOSOrder: {Action: carrierintel.RuleActionWarn},
		}
		g.control.PolicyVersion++

		gates, err := g.svc.GateFor(t.Context(), g.tenant, []pulid.ID{carrierID})
		require.NoError(t, err)
		require.Len(t, g.subjects.requested, 1)
		finding := gateFinding(gates[carrierID], carrierintel.RuleSafetyOOSOrder)
		require.NotNil(t, finding)
		assert.Equal(t, carrier.IntelAction(carrierintel.RuleActionWarn), finding.Action)
	})

	t.Run("rules turned off drop out of the gate", func(t *testing.T) {
		t.Parallel()
		g := newGateHarness(t)
		carrierID := pulid.MustNew("car_")
		g.addSnapshot(carrierID, 3600)

		g.control.Rules = carrierintel.RuleSettings{
			carrierintel.RuleSafetyOOSOrder: {Action: carrierintel.RuleActionOff},
		}
		g.control.PolicyVersion++

		gates, err := g.svc.GateFor(t.Context(), g.tenant, []pulid.ID{carrierID})
		require.NoError(t, err)
		assert.Nil(t, gateFinding(gates[carrierID], carrierintel.RuleSafetyOOSOrder))
	})

	t.Run("active overrides mark the block overridden until expiry", func(t *testing.T) {
		t.Parallel()
		g := newGateHarness(t)
		carrierID := pulid.MustNew("car_")
		g.addSnapshot(carrierID, 3600)
		expiresAt := g.now + 7*timeutils.SecondsPerDay
		g.overrides.overrides = []*carrierintel.CarrierIntelOverride{{
			ID:        pulid.MustNew("ciovr_"),
			CarrierID: carrierID,
			RuleCode:  carrierintel.RuleSafetyOOSOrder,
			ExpiresAt: expiresAt,
		}}

		gates, err := g.svc.GateFor(t.Context(), g.tenant, []pulid.ID{carrierID})
		require.NoError(t, err)
		finding := gateFinding(gates[carrierID], carrierintel.RuleSafetyOOSOrder)
		require.NotNil(t, finding)
		assert.True(t, finding.Overridden)
		require.NotNil(t, finding.OverrideExpiresAt)
		assert.Equal(t, expiresAt, *finding.OverrideExpiresAt)

		result := carrier.EvaluateEligibility(carrier.EligibilityInput{
			Carrier: &carrier.Carrier{
				Status:           carrier.StatusActive,
				ComplianceStatus: carrier.ComplianceStatusQualified,
			},
			Now:   g.now,
			Intel: gates[carrierID],
		})
		for _, blocker := range result.Blockers {
			assert.NotContains(t, blocker, finding.Message)
		}
		assert.Empty(t, result.Warnings)
	})

	t.Run("an open breaker reports the provider unavailable", func(t *testing.T) {
		t.Parallel()
		g := newGateHarness(t)
		carrierID := pulid.MustNew("car_")
		g.addSnapshot(carrierID, 3600)
		key := breakerKey(g.tenant, integration.TypeCarrierOK)
		for range breakerFailureThreshold {
			g.svc.breaker.record(key, true, time.Unix(g.now, 0))
		}

		gates, err := g.svc.GateFor(t.Context(), g.tenant, []pulid.ID{carrierID})
		require.NoError(t, err)
		assert.True(t, gates[carrierID].ProviderUnavailable)
		assert.NotNil(t, gateFinding(gates[carrierID], carrierintel.RuleSafetyOOSOrder))
	})
}
