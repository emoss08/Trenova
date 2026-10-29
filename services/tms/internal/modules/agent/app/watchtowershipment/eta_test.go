package watchtowershipment

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/domain/stopeta"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchcontrolports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/stopetaports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeStopEtas struct {
	stopetaports.Repository

	stops    []*stopetaports.AttentionStop
	requests []*stopetaports.ListAttentionRequest
}

func (f *fakeStopEtas) ListAttention(
	_ context.Context,
	req *stopetaports.ListAttentionRequest,
) ([]*stopetaports.AttentionStop, error) {
	f.requests = append(f.requests, req)
	return f.stops, nil
}

type fakeEtaControls struct {
	dispatchcontrolports.DispatchControlRepository

	control *dispatchcontrol.DispatchControl
}

func (f *fakeEtaControls) GetByOrgID(
	context.Context,
	dispatchcontrolports.GetDispatchControlRequest,
) (*dispatchcontrol.DispatchControl, error) {
	return f.control, nil
}

type recordingProjector struct {
	upserted []watchtowerports.WatchtowerItemInput
	resolved []string
}

func (p *recordingProjector) Upsert(_ context.Context, input watchtowerports.WatchtowerItemInput) {
	p.upserted = append(p.upserted, input)
}

func (p *recordingProjector) Resolve(
	_ context.Context,
	_ pagination.TenantInfo,
	kind watchtower.SourceKind,
	sourceID string,
) {
	if kind == watchtower.SourceEtaAtRisk {
		p.resolved = append(p.resolved, sourceID)
	}
}

func attentionStop(
	shipmentID pulid.ID,
	seq int64,
	verdict stopeta.Verdict,
) *stopetaports.AttentionStop {
	windowEnd := int64(1_800_010_000)
	return &stopetaports.AttentionStop{
		ETA: &stopeta.StopETA{
			ShipmentID:       shipmentID,
			StopID:           pulid.MustNew("stp_"),
			StopSequence:     seq,
			Verdict:          verdict,
			WindowEnd:        &windowEnd,
			ThresholdMinutes: 60,
			ComputedAt:       1_800_000_000,
		},
		ProNumber:    "PRO-7",
		StopType:     shipmentvocab.StopTypeDelivery,
		LocationName: "Houston DC",
		City:         "Houston",
		StateCode:    "TX",
	}
}

func TestDescribeEtaRisks_OneItemPerShipmentForTheWorstStop(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	shipmentID := pulid.MustNew("shp_")
	risky := attentionStop(shipmentID, 1, stopeta.VerdictAtRisk)
	late := attentionStop(shipmentID, 2, stopeta.VerdictLate)

	items := DescribeEtaRisks(tenant, []*stopetaports.AttentionStop{risky, late})
	require.Len(t, items, 1)
	item := items[0]
	assert.Equal(t, watchtower.SourceEtaAtRisk, item.SourceKind)
	assert.Equal(t, shipmentID.String(), item.SourceID)
	assert.Equal(t, watchtower.SeverityCritical, item.Severity)
	assert.Equal(t, "Projected late on PRO-7", item.Title)
	assert.Equal(
		t,
		"Projected to reach the delivery at Houston DC in Houston, TX after its window closes.",
		item.Summary,
	)
	assert.Equal(t, agent.SubjectShipment, item.SubjectType)
	assert.Equal(t, int64(1_800_000_000), item.OccurredAt)
}

func TestDescribeEtaRisk_AtRiskIsAWarning(t *testing.T) {
	t.Parallel()

	stop := attentionStop(pulid.MustNew("shp_"), 1, stopeta.VerdictAtRisk)
	stop.ETA.InputStale = true
	item := DescribeEtaRisk(pagination.TenantInfo{}, stop)

	assert.Equal(t, watchtower.SeverityWarning, item.Severity)
	assert.Equal(t, "ETA at risk on PRO-7", item.Title)
	assert.Contains(t, item.Summary, "less than 60 minutes to spare")
	assert.Contains(t, item.Summary, "old information")
}

func TestEtaAtRiskSource_SnapshotHonoursTheSwitch(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	etas := &fakeStopEtas{stops: []*stopetaports.AttentionStop{
		attentionStop(pulid.MustNew("shp_"), 1, stopeta.VerdictLate),
	}}
	control := dispatchcontrol.NewDefaultDispatchControl(tenant.OrgID, tenant.BuID)
	source := NewEtaAtRiskSource(EtaAtRiskParams{
		StopEtas: etas,
		Controls: &fakeEtaControls{control: control},
	})

	items, err := source.Snapshot(t.Context(), tenant)
	require.NoError(t, err)
	assert.Len(t, items, 1)

	control.EnableEtaAlerts = false
	items, err = source.Snapshot(t.Context(), tenant)
	require.NoError(t, err)
	assert.Empty(t, items)
}

func TestEtaAtRiskObserver_RaisesAndClears(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	risky := pulid.MustNew("shp_")
	recovered := pulid.MustNew("shp_")
	etas := &fakeStopEtas{stops: []*stopetaports.AttentionStop{
		attentionStop(risky, 1, stopeta.VerdictAtRisk),
	}}
	projector := &recordingProjector{}
	observer := NewEtaAtRiskObserver(EtaAtRiskObserverParams{
		StopEtas: etas,
		Controls: &fakeEtaControls{
			control: dispatchcontrol.NewDefaultDispatchControl(tenant.OrgID, tenant.BuID),
		},
		Projector: projector,
		Logger:    zap.NewNop(),
	})

	observer.StopEtasRefreshed(t.Context(), tenant, []pulid.ID{risky, recovered})

	require.Len(t, projector.upserted, 1)
	assert.Equal(t, risky.String(), projector.upserted[0].SourceID)
	assert.Equal(t, []string{recovered.String()}, projector.resolved)
	require.Len(t, etas.requests, 1)
	assert.Equal(t, []pulid.ID{risky, recovered}, etas.requests[0].ShipmentIDs)
}
