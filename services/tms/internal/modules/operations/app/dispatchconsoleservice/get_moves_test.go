package dispatchconsoleservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchboardports"
	"github.com/emoss08/trenova/internal/testutil/mocks/dispatchboardportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestGetMoves_ReadsTheNamedMovesWhateverTheirWindowOrStatus(t *testing.T) {
	t.Parallel()

	info := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	ids := []pulid.ID{pulid.MustNew("smv_"), pulid.MustNew("smv_")}
	console := dispatchboardportsmock.NewMockDispatchConsoleRepository(t)
	svc := &Service{consoleRepo: console, l: zap.NewNop()}

	console.EXPECT().
		ListBoardMoves(mock.Anything, mock.MatchedBy(func(f *dispatchboardports.DispatchBoardFilter) bool {
			return assert.ObjectsAreEqual(ids, f.MoveIDs) &&
				f.IncludeCovered &&
				f.WindowStart == 0 && f.WindowEnd == 0 &&
				assert.ObjectsAreEqual(shipmentvocab.MoveStatuses(), f.MoveStatuses) &&
				f.Limit == len(ids)
		})).
		Return([]*dispatchboardports.BoardMove{}, nil).
		Once()

	moves, err := svc.GetMoves(t.Context(), info, ids)
	require.NoError(t, err)
	assert.Empty(t, moves)
}

func TestGetMoves_ReadsNothingForNoneAndRefusesMoreThanItsBound(t *testing.T) {
	t.Parallel()

	console := dispatchboardportsmock.NewMockDispatchConsoleRepository(t)
	svc := &Service{consoleRepo: console, l: zap.NewNop()}

	moves, err := svc.GetMoves(t.Context(), pagination.TenantInfo{}, nil)
	require.NoError(t, err)
	assert.Empty(t, moves)

	tooMany := make([]pulid.ID, MaxMovesByID+1)
	for idx := range tooMany {
		tooMany[idx] = pulid.MustNew("smv_")
	}
	_, err = svc.GetMoves(t.Context(), pagination.TenantInfo{}, tooMany)
	require.Error(t, err)
}
