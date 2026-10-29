package agentquerytoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/equipmentquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/masterdataquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytooltest"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/trailer"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/equipmenttypeports"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/tractorports"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/trailerports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type cursorRepo[T pagination.CursorEntity] struct {
	items    []T
	received pagination.CursorInfo
}

func newCursorRepo[T pagination.CursorEntity](count int, build func(id pulid.ID) T) *cursorRepo[T] {
	items := make([]T, 0, count)
	for range count {
		items = append(items, build(pulid.MustNew("tst_")))
	}

	return &cursorRepo[T]{items: items}
}

func (r *cursorRepo[T]) page(cursor pagination.CursorInfo) *pagination.CursorListResult[T] {
	r.received = cursor

	return querytooltest.PageAfter(r.items, cursor)
}

type pagedTractorRepo struct {
	tractorports.TractorRepository
	*cursorRepo[*tractor.Tractor]
}

func (r *pagedTractorRepo) List(
	_ context.Context,
	req *tractorports.ListTractorsRequest,
) (*pagination.CursorListResult[*tractor.Tractor], error) {
	return r.page(req.Cursor), nil
}

type pagedTrailerRepo struct {
	trailerports.TrailerRepository
	*cursorRepo[*trailer.Trailer]
}

func (r *pagedTrailerRepo) List(
	_ context.Context,
	req *trailerports.ListTrailersRequest,
) (*pagination.CursorListResult[*trailer.Trailer], error) {
	return r.page(req.Cursor), nil
}

type pagedEquipmentTypeRepo struct {
	equipmenttypeports.EquipmentTypeRepository
	*cursorRepo[*equipmenttype.EquipmentType]
}

func (r *pagedEquipmentTypeRepo) List(
	_ context.Context,
	req *equipmenttypeports.ListEquipmentTypesRequest,
) (*pagination.CursorListResult[*equipmenttype.EquipmentType], error) {
	return r.page(req.Cursor), nil
}

type cursorListCase struct {
	tool     agenttoolports.AgentQueryTool
	received func() pagination.CursorInfo
	ids      []pulid.ID
}

func cursorCase[T pagination.CursorEntity](
	tool agenttoolports.AgentQueryTool,
	repo *cursorRepo[T],
) cursorListCase {
	return cursorListCase{
		tool:     tool,
		received: func() pagination.CursorInfo { return repo.received },
		ids:      querytooltest.CursorIDs(repo.items),
	}
}

const cursorCaseRows = 7

func TestCursorListCatalog_PagesByAfterAndNextCursor(t *testing.T) {
	t.Parallel()

	cases := map[string]func() cursorListCase{
		"tractors": func() cursorListCase {
			repo := newCursorRepo(cursorCaseRows, func(id pulid.ID) *tractor.Tractor {
				return &tractor.Tractor{ID: id, Code: "T"}
			})
			return cursorCase(
				equipmentquerytools.NewListTractorsTool(
					&pagedTractorRepo{cursorRepo: repo},
					&querytooltest.FakePermissions{},
				),
				repo,
			)
		},
		"trailers": func() cursorListCase {
			repo := newCursorRepo(cursorCaseRows, func(id pulid.ID) *trailer.Trailer {
				return &trailer.Trailer{ID: id, Code: "R"}
			})
			return cursorCase(
				equipmentquerytools.NewListTrailersTool(&pagedTrailerRepo{cursorRepo: repo}),
				repo,
			)
		},
		"equipment types": func() cursorListCase {
			repo := newCursorRepo(cursorCaseRows, func(id pulid.ID) *equipmenttype.EquipmentType {
				return &equipmenttype.EquipmentType{ID: id, Code: "DV"}
			})
			return cursorCase(
				masterdataquerytools.NewListEquipmentTypesTool(
					&pagedEquipmentTypeRepo{cursorRepo: repo},
				),
				repo,
			)
		},
	}

	for name, build := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			c := build()
			params := querytooltest.TestParams(nil)
			querytooltest.AssertCursorPaging(t, querytooltest.CursorPagingCase{
				Query: func(args map[string]any) (any, error) {
					params.Params = args
					return c.tool.Query(t.Context(), params)
				},
				Received: c.received,
				Args:     map[string]any{"query": "a"},
				Changed:  map[string]any{"query": "b"},
				IDs:      c.ids,
				Limit:    3,
			})
		})
	}
}
