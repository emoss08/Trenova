package watchtowersources

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/checkcallports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeCheckCalls struct {
	checkcallports.CheckCallService

	overdue []*checkcallports.OverdueMove
	now     int64
	err     error
}

func (f *fakeCheckCalls) ListOverdue(
	_ context.Context,
	_ pagination.TenantInfo,
	now int64,
) ([]*checkcallports.OverdueMove, error) {
	f.now = now
	return f.overdue, f.err
}

type resolveCall struct {
	tenant   pagination.TenantInfo
	kind     watchtower.SourceKind
	sourceID string
}

type fakeProjector struct {
	resolved []resolveCall
}

func (p *fakeProjector) Upsert(context.Context, watchtowerports.WatchtowerItemInput) {}

func (p *fakeProjector) Resolve(
	_ context.Context,
	tenant pagination.TenantInfo,
	kind watchtower.SourceKind,
	sourceID string,
) {
	p.resolved = append(p.resolved, resolveCall{tenant: tenant, kind: kind, sourceID: sourceID})
}

func overdueMove(overdueBy int64) *checkcallports.OverdueMove {
	return &checkcallports.OverdueMove{
		MoveID:        pulid.MustNew("smv_"),
		ShipmentID:    pulid.MustNew("shp_"),
		ProNumber:     "PRO-42",
		CarrierName:   "Redline Freight",
		LastContactAt: 1_800_000_000,
		IntervalHours: 4,
		OverdueBy:     overdueBy,
	}
}

func TestDescribeNoCheckCall_WarnsThenEscalatesAtTwiceTheWindow(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	move := overdueMove(30 * 60)

	item := DescribeNoCheckCall(tenant, move)

	assert.Equal(t, watchtower.SourceNoCheckCall, item.SourceKind)
	assert.Equal(t, move.MoveID.String(), item.SourceID)
	assert.Equal(t, watchtower.SeverityWarning, item.Severity)
	assert.Equal(t, "No check call on PRO-42", item.Title)
	assert.Contains(t, item.Summary, "Redline Freight")
	assert.Contains(t, item.Summary, "4h")
	assert.Equal(t, agent.SubjectShipmentMove, item.SubjectType)
	assert.Equal(t, move.MoveID, item.SubjectID)
	assert.Contains(t, item.Path, move.ShipmentID.String())
	assert.Equal(t, int64(1_800_000_000+4*60*60), item.OccurredAt)
	assert.Equal(t, tenant, item.TenantInfo)

	critical := DescribeNoCheckCall(tenant, overdueMove(4*60*60))
	assert.Equal(t, watchtower.SeverityCritical, critical.Severity)
	assert.Contains(t, critical.Summary, "twice")
}

func TestDescribeNoCheckCall_NamesNoCarrierWhenNoneIsKnown(t *testing.T) {
	t.Parallel()

	move := overdueMove(60)
	move.CarrierName = ""
	move.ProNumber = ""

	item := DescribeNoCheckCall(pagination.TenantInfo{}, move)

	assert.Equal(t, "No check call", item.Title)
	assert.Contains(t, item.Summary, "The carrier")
}

func TestNoCheckCallSource_SnapshotsTheOverdueLoads(t *testing.T) {
	t.Parallel()

	checkCalls := &fakeCheckCalls{
		overdue: []*checkcallports.OverdueMove{overdueMove(60), nil, overdueMove(120)},
	}
	source := &NoCheckCallSource{checkCalls: checkCalls, now: func() int64 { return 42 }}

	items, err := source.Snapshot(t.Context(), pagination.TenantInfo{})

	require.NoError(t, err)
	assert.Len(t, items, 2)
	assert.Equal(t, int64(42), checkCalls.now)
	assert.Equal(t, watchtower.SourceNoCheckCall, source.Kind())
}

func TestNoCheckCallSource_PassesAReadFailureUp(t *testing.T) {
	t.Parallel()

	source := &NoCheckCallSource{
		checkCalls: &fakeCheckCalls{err: assert.AnError},
		now:        func() int64 { return 42 },
	}

	_, err := source.Snapshot(t.Context(), pagination.TenantInfo{})

	require.ErrorIs(t, err, assert.AnError)
}

func TestNoCheckCallObserver_ResolvesTheMoveItem(t *testing.T) {
	t.Parallel()

	projector := &fakeProjector{}
	observer := NewNoCheckCallObserver(projector)
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	moveID := pulid.MustNew("smv_")

	observer.CheckCallRecorded(t.Context(), tenant, moveID)
	observer.CheckCallRecorded(t.Context(), tenant, pulid.Nil)

	require.Len(t, projector.resolved, 1)
	assert.Equal(t, resolveCall{
		tenant:   tenant,
		kind:     watchtower.SourceNoCheckCall,
		sourceID: moveID.String(),
	}, projector.resolved[0])
}
