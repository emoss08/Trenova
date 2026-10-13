package watchtowersources

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/checkcallports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

type NoCheckCallSource struct {
	checkCalls checkcallports.CheckCallService
	now        func() int64
}

func NewNoCheckCallSource(
	checkCalls checkcallports.CheckCallService,
) watchtowerports.WatchtowerSource {
	return &NoCheckCallSource{checkCalls: checkCalls, now: timeutils.NowUnix}
}

func (s *NoCheckCallSource) Kind() watchtower.SourceKind {
	return watchtower.SourceNoCheckCall
}

func (s *NoCheckCallSource) Snapshot(
	ctx context.Context,
	tenant pagination.TenantInfo,
) ([]watchtowerports.WatchtowerItemInput, error) {
	overdue, err := s.checkCalls.ListOverdue(ctx, tenant, s.now())
	if err != nil {
		return nil, err
	}

	return DescribeNoCheckCalls(tenant, overdue), nil
}

func DescribeNoCheckCalls(
	tenant pagination.TenantInfo,
	overdue []*checkcallports.OverdueMove,
) []watchtowerports.WatchtowerItemInput {
	items := make([]watchtowerports.WatchtowerItemInput, 0, len(overdue))
	for _, move := range overdue {
		if move == nil || move.MoveID.IsNil() {
			continue
		}
		items = append(items, DescribeNoCheckCall(tenant, move))
	}

	return items
}

type NoCheckCallObserver struct {
	projector watchtowerports.WatchtowerProjector
}

func NewNoCheckCallObserver(
	projector watchtowerports.WatchtowerProjector,
) checkcallports.CheckCallObserver {
	return &NoCheckCallObserver{projector: projector}
}

func (o *NoCheckCallObserver) CheckCallRecorded(
	ctx context.Context,
	tenant pagination.TenantInfo,
	_, moveID pulid.ID,
) {
	if moveID.IsNil() {
		return
	}
	o.projector.Resolve(ctx, tenant, watchtower.SourceNoCheckCall, moveID.String())
}
