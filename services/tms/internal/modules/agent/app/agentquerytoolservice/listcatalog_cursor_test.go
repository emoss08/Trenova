package agentquerytoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/trailer"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/equipmenttypeports"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/tractorports"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/trailerports"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/fuelpurchase"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/ifta"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/fuelpurchaseports"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/iftaports"
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

	return pageAfter(r.items, cursor)
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

type pagedFuelPurchaseRepo struct {
	fuelpurchaseports.FuelPurchaseRepository
	purchases *cursorRepo[*fuelpurchase.FuelPurchase]
	cards     *cursorRepo[*fuelpurchase.FuelCard]
	batches   *cursorRepo[*fuelpurchase.ImportBatch]
}

func (r *pagedFuelPurchaseRepo) ListPurchases(
	_ context.Context,
	req *fuelpurchaseports.ListFuelPurchasesRequest,
) (*pagination.CursorListResult[*fuelpurchase.FuelPurchase], error) {
	return r.purchases.page(req.Cursor), nil
}

func (r *pagedFuelPurchaseRepo) ListCards(
	_ context.Context,
	req *fuelpurchaseports.ListFuelCardsRequest,
) (*pagination.CursorListResult[*fuelpurchase.FuelCard], error) {
	return r.cards.page(req.Cursor), nil
}

func (r *pagedFuelPurchaseRepo) ListImportBatches(
	_ context.Context,
	req *fuelpurchaseports.ListImportBatchesRequest,
) (*pagination.CursorListResult[*fuelpurchase.ImportBatch], error) {
	return r.batches.page(req.Cursor), nil
}

type pagedIFTARepo struct {
	iftaports.IFTARepository
	returns *cursorRepo[*ifta.Return]
	entries *cursorRepo[*ifta.JurisdictionMileageEntry]
}

func (r *pagedIFTARepo) ListReturns(
	_ context.Context,
	req *iftaports.ListReturnsRequest,
) (*pagination.CursorListResult[*ifta.Return], error) {
	return r.returns.page(req.Cursor), nil
}

func (r *pagedIFTARepo) ListMileageEntries(
	_ context.Context,
	req *iftaports.ListMileageEntriesRequest,
) (*pagination.CursorListResult[*ifta.JurisdictionMileageEntry], error) {
	return r.entries.page(req.Cursor), nil
}

type cursorListCase struct {
	tool     agentports.AgentQueryTool
	received func() pagination.CursorInfo
	ids      []pulid.ID
}

func cursorCase[T pagination.CursorEntity](
	tool agentports.AgentQueryTool,
	repo *cursorRepo[T],
) cursorListCase {
	return cursorListCase{
		tool:     tool,
		received: func() pagination.CursorInfo { return repo.received },
		ids:      cursorIDs(repo.items),
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
				newListTractorsTool(&pagedTractorRepo{cursorRepo: repo}, &fakePermissions{}),
				repo,
			)
		},
		"trailers": func() cursorListCase {
			repo := newCursorRepo(cursorCaseRows, func(id pulid.ID) *trailer.Trailer {
				return &trailer.Trailer{ID: id, Code: "R"}
			})
			return cursorCase(newListTrailersTool(&pagedTrailerRepo{cursorRepo: repo}), repo)
		},
		"equipment types": func() cursorListCase {
			repo := newCursorRepo(cursorCaseRows, func(id pulid.ID) *equipmenttype.EquipmentType {
				return &equipmenttype.EquipmentType{ID: id, Code: "DV"}
			})
			return cursorCase(
				newListEquipmentTypesTool(&pagedEquipmentTypeRepo{cursorRepo: repo}),
				repo,
			)
		},
		"fuel purchases": func() cursorListCase {
			repo := newCursorRepo(cursorCaseRows, func(id pulid.ID) *fuelpurchase.FuelPurchase {
				return &fuelpurchase.FuelPurchase{ID: id}
			})
			return cursorCase(
				newListFuelPurchasesTool(&pagedFuelPurchaseRepo{purchases: repo}),
				repo,
			)
		},
		"fuel cards": func() cursorListCase {
			repo := newCursorRepo(cursorCaseRows, func(id pulid.ID) *fuelpurchase.FuelCard {
				return &fuelpurchase.FuelCard{ID: id}
			})
			return cursorCase(newListFuelCardsTool(&pagedFuelPurchaseRepo{cards: repo}), repo)
		},
		"fuel purchase imports": func() cursorListCase {
			repo := newCursorRepo(cursorCaseRows, func(id pulid.ID) *fuelpurchase.ImportBatch {
				return &fuelpurchase.ImportBatch{ID: id}
			})
			return cursorCase(
				newListFuelPurchaseImportsTool(&pagedFuelPurchaseRepo{batches: repo}),
				repo,
			)
		},
		"IFTA returns": func() cursorListCase {
			repo := newCursorRepo(cursorCaseRows, func(id pulid.ID) *ifta.Return {
				return &ifta.Return{ID: id}
			})
			return cursorCase(newListIFTAReturnsTool(&pagedIFTARepo{returns: repo}), repo)
		},
		"IFTA mileage entries": func() cursorListCase {
			repo := newCursorRepo(
				cursorCaseRows,
				func(id pulid.ID) *ifta.JurisdictionMileageEntry {
					return &ifta.JurisdictionMileageEntry{ID: id}
				},
			)
			return cursorCase(newListIFTAMileageEntriesTool(&pagedIFTARepo{entries: repo}), repo)
		},
	}

	for name, build := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			c := build()
			params := testParams(nil)
			assertCursorPaging(t, cursorPagingCase{
				query: func(args map[string]any) (any, error) {
					params.Params = args
					return c.tool.Query(t.Context(), params)
				},
				received: c.received,
				args:     map[string]any{"query": "a"},
				changed:  map[string]any{"query": "b"},
				ids:      c.ids,
				limit:    3,
			})
		})
	}
}
