package equipmentquerytools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/fieldgatetest"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytooltest"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/tractorports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubTractorList struct {
	tractorports.TractorRepository

	items []*tractor.Tractor
	last  *tractorports.ListTractorsRequest
}

func (s *stubTractorList) List(
	_ context.Context,
	req *tractorports.ListTractorsRequest,
) (*pagination.CursorListResult[*tractor.Tractor], error) {
	s.last = req

	return &pagination.CursorListResult[*tractor.Tractor]{Items: s.items}, nil
}

func TestListTractors_NamesTheDriverOnlyToSomeoneWhoMayReadWorkers(t *testing.T) {
	t.Parallel()

	unit := &tractor.Tractor{
		ID:            pulid.MustNew("tr_"),
		Code:          "T-104",
		PrimaryWorker: &worker.Worker{FirstName: "Dana", LastName: "Whitfield"},
	}

	for _, tc := range []struct {
		name    string
		allowed bool
		want    string
	}{
		{name: "reads workers", allowed: true, want: "Dana Whitfield"},
		{name: "does not read workers", allowed: false, want: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			repo := &stubTractorList{items: []*tractor.Tractor{unit}}
			params := querytooltest.ChatParams(map[string]any{}, "")
			result, err := NewListTractorsTool(
				repo,
				&querytooltest.FakePermissions{Allowed: tc.allowed},
			).
				Query(t.Context(), params)
			require.NoError(t, err)

			rows, _ := fieldgatetest.ListedRows[equipmentRow](t, result)
			require.Len(t, rows, 1)
			assert.Equal(t, "T-104", rows[0].Code)
			assert.Equal(t, tc.want, rows[0].AssignedTo)
			assert.Equal(t, tc.allowed, repo.last.IncludePrimaryWorker,
				"a driver nobody may be told about is not loaded")
		})
	}
}
