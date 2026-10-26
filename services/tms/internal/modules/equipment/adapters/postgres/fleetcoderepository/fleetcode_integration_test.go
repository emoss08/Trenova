//go:build integration

package fleetcoderepository

import (
	"cmp"
	"fmt"
	"slices"
	"testing"

	"github.com/emoss08/trenova/internal/modules/equipment/domain/fleetcode"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/fleetcode/fleetcodedb"
	"github.com/emoss08/trenova/internal/modules/equipment/equipmenttest"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/fleetcodeports"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fleetCodeFixture struct {
	db    *equipmenttest.Database
	repo  fleetcodeports.FleetCodeRepository
	codes []*fleetcode.FleetCode
	other *fleetcode.FleetCode
}

func newFleetCodeFixture(t *testing.T) *fleetCodeFixture {
	t.Helper()

	db := equipmenttest.NewDatabase(t)
	f := &fleetCodeFixture{db: db, repo: New(Params{DB: db.Conn, Logger: zap.NewNop()})}

	for i, name := range []string{"Alpha", "Bravo", "Charlie", "Delta", "Echo"} {
		created, err := f.repo.Create(db.A.Ctx, &fleetcode.FleetCode{
			ManagerID:   db.A.Info.UserID,
			Status:      domaintypes.StatusActive,
			Code:        fmt.Sprintf("FC%d", i),
			Description: name + " fleet",
		})
		require.NoError(t, err)
		f.codes = append(f.codes, created)
	}

	other, err := f.repo.Create(db.B.Ctx, &fleetcode.FleetCode{
		ManagerID:   db.B.Info.UserID,
		Status:      domaintypes.StatusActive,
		Code:        "FC0",
		Description: "Alpha fleet",
	})
	require.NoError(t, err)
	f.other = other

	return f
}

func (f *fleetCodeFixture) connection(
	t *testing.T,
	filter *pagination.QueryOptions,
	limit int,
	after string,
) *pagination.CursorListResult[*fleetcode.FleetCode] {
	t.Helper()

	cursor, err := pagination.NewCursorInfo(limit, after)
	require.NoError(t, err)

	result, err := f.repo.ListConnection(f.db.A.Ctx, &fleetcodeports.ListFleetCodeConnectionRequest{
		Filter: filter,
		Cursor: cursor,
	})
	require.NoError(t, err)

	return result
}

func newestFirst(items []*fleetcode.FleetCode) []string {
	sorted := slices.Clone(items)
	slices.SortFunc(sorted, func(a, b *fleetcode.FleetCode) int {
		if a.CreatedAt != b.CreatedAt {
			return cmp.Compare(b.CreatedAt, a.CreatedAt)
		}
		return cmp.Compare(b.ID.String(), a.ID.String())
	})

	return codesOf(sorted)
}

func codesOf(items []*fleetcode.FleetCode) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.Code)
	}

	return out
}

func TestFleetCodeRepository(t *testing.T) {
	f := newFleetCodeFixture(t)
	a := f.db.A

	t.Run("create stamps the id, the tenant and the timestamps", func(t *testing.T) {
		created := f.codes[0]
		assert.True(t, created.ID.IsNotNil())
		assert.Equal(t, a.Info.OrgID, created.OrganizationID)
		assert.Equal(t, a.Info.BuID, created.BusinessUnitID)
		assert.Positive(t, created.CreatedAt)
		assert.Equal(t, created.CreatedAt, created.UpdatedAt)
	})

	t.Run("get by id reads only the caller's tenant", func(t *testing.T) {
		got, err := f.repo.GetByID(a.Ctx, fleetcodeports.GetFleetCodeByIDRequest{ID: f.codes[1].ID})
		require.NoError(t, err)
		assert.Equal(t, "FC1", got.Code)

		_, err = f.repo.GetByID(a.Ctx, fleetcodeports.GetFleetCodeByIDRequest{ID: f.other.ID})
		assert.True(t, errortypes.IsNotFoundError(err))
	})

	t.Run("update checks the version", func(t *testing.T) {
		current, err := f.repo.GetByID(
			a.Ctx,
			fleetcodeports.GetFleetCodeByIDRequest{ID: f.codes[2].ID},
		)
		require.NoError(t, err)
		stale := *current

		current.Color = "#112233"
		updated, err := f.repo.Update(a.Ctx, current)
		require.NoError(t, err)
		assert.Equal(t, stale.Version+1, updated.Version)

		stale.Color = "#445566"
		_, err = f.repo.Update(a.Ctx, &stale)
		var validation *errortypes.Error
		require.ErrorAs(t, err, &validation)
		assert.Equal(t, errortypes.ErrVersionMismatch, validation.Code)
	})

	t.Run("a missing manager is a validation error on the field", func(t *testing.T) {
		_, err := f.repo.Create(a.Ctx, &fleetcode.FleetCode{
			ManagerID: pulid.MustNew("usr_"),
			Status:    domaintypes.StatusActive,
			Code:      "NOPE",
		})
		var validation *errortypes.Error
		require.ErrorAs(t, err, &validation)
		assert.Equal(t, "managerId", validation.Field)
	})

	t.Run("a connection pages newest first with every cursor resumable", func(t *testing.T) {
		order := newestFirst(f.codes)

		first := f.connection(t, &pagination.QueryOptions{}, 2, "")
		assert.Equal(t, order[0:2], codesOf(first.Items))
		assert.True(t, first.HasNextPage)
		require.NotNil(t, first.TotalCount)
		assert.Equal(t, 5, *first.TotalCount)

		next, err := first.NextCursor()
		require.NoError(t, err)
		second := f.connection(t, &pagination.QueryOptions{}, 2, next)
		assert.Equal(t, order[2:4], codesOf(second.Items))

		afterFirstEdge := f.connection(t, &pagination.QueryOptions{}, 2, first.EdgeCursors[0])
		assert.Equal(t, order[1:3], codesOf(afterFirstEdge.Items))

		afterThat := f.connection(t, &pagination.QueryOptions{}, 2, afterFirstEdge.EdgeCursors[0])
		assert.Equal(t, order[2:4], codesOf(afterThat.Items))

		last, err := second.NextCursor()
		require.NoError(t, err)
		third := f.connection(t, &pagination.QueryOptions{}, 2, last)
		assert.Equal(t, order[4:], codesOf(third.Items))
		assert.False(t, third.HasNextPage)
	})

	t.Run("a connection sorted by a column pages in that order", func(t *testing.T) {
		filter := &pagination.QueryOptions{Sort: []domaintypes.SortField{
			{Field: "code", Direction: dbtype.SortDirectionAsc},
		}}

		first := f.connection(t, filter, 3, "")
		assert.Equal(t, []string{"FC0", "FC1", "FC2"}, codesOf(first.Items))

		next, err := first.NextCursor()
		require.NoError(t, err)
		second := f.connection(t, filter, 3, next)
		assert.Equal(t, []string{"FC3", "FC4"}, codesOf(second.Items))

		cursor, err := pagination.NewCursorInfo(3, next)
		require.NoError(t, err)
		_, err = f.repo.ListConnection(a.Ctx, &fleetcodeports.ListFleetCodeConnectionRequest{
			Filter: &pagination.QueryOptions{},
			Cursor: cursor,
		})
		require.Error(t, err)
	})

	t.Run("filters, groups and search narrow the connection and its total", func(t *testing.T) {
		filtered := f.connection(t, &pagination.QueryOptions{
			FieldFilters: []domaintypes.FieldFilter{
				{Field: "description", Operator: dbtype.OpContains, Value: "a fleet"},
			},
			FilterGroups: []domaintypes.FilterGroup{{Filters: []domaintypes.FieldFilter{
				{Field: "code", Operator: dbtype.OpEqual, Value: "FC0"},
				{Field: "code", Operator: dbtype.OpEqual, Value: "FC2"},
				{Field: "code", Operator: dbtype.OpEqual, Value: "FC3"},
			}}},
		}, 10, "")
		assert.ElementsMatch(t, []string{"FC0", "FC3"}, codesOf(filtered.Items))
		assert.Equal(t, 2, *filtered.TotalCount)

		searched := f.connection(t, &pagination.QueryOptions{Query: "char"}, 10, "")
		assert.Equal(t, []string{"FC2"}, codesOf(searched.Items))
	})

	t.Run("the connection reads only the requested columns and the manager", func(t *testing.T) {
		cursor, err := pagination.NewCursorInfo(10, "")
		require.NoError(t, err)
		result, err := f.repo.ListConnection(a.Ctx, &fleetcodeports.ListFleetCodeConnectionRequest{
			Filter:           &pagination.QueryOptions{},
			Cursor:           cursor,
			FleetCodeColumns: []string{"id", "code", "created_at"},
		})
		require.NoError(t, err)
		require.Len(t, result.Items, 5)
		assert.Empty(t, result.Items[0].Description)
		assert.NotEmpty(t, result.Items[0].Code)
		require.NotNil(t, result.Items[0].Manager)
		assert.Equal(t, a.Info.UserID, result.Items[0].Manager.ID)
	})

	t.Run("an offset list counts every match", func(t *testing.T) {
		result, err := f.repo.List(a.Ctx, &fleetcodeports.ListFleetCodesRequest{
			Filter: &pagination.QueryOptions{
				Pagination: pagination.Info{Limit: 2, Offset: 1},
				Sort: []domaintypes.SortField{
					{Field: "code", Direction: dbtype.SortDirectionDesc},
				},
			},
			IncludeManagerDetails: true,
		})
		require.NoError(t, err)
		assert.Equal(t, 5, result.Total)
		assert.True(t, result.TotalExact)
		assert.Equal(t, []string{"FC3", "FC2"}, codesOf(result.Items))
		assert.NotNil(t, result.Items[0].Manager)
	})

	t.Run("select options search the active codes", func(t *testing.T) {
		result, err := f.repo.SelectOptions(a.Ctx, &pagination.SelectQueryRequest{
			Query:      "delta",
			Pagination: pagination.Info{Limit: 10},
		})
		require.NoError(t, err)
		assert.Equal(t, 1, result.Total)
		assert.True(t, result.TotalExact)
		assert.Equal(t, []string{"FC3"}, codesOf(result.Items))
	})

	t.Run("the keel checker answers uniqueness and references", func(t *testing.T) {
		checker := validationframework.NewKeelChecker(f.db.Conn, fleetcodedb.FleetCodesSchema())

		taken, err := checker.CheckUniqueness(a.Ctx, &validationframework.UniquenessRequest{
			TableName:      "fleet_codes",
			OrganizationID: a.Info.OrgID,
			BusinessUnitID: a.Info.BuID,
			Fields:         []validationframework.FieldCheck{{Column: "code", Value: "fc1"}},
		})
		require.NoError(t, err)
		assert.True(t, taken)

		self, err := checker.CheckUniqueness(a.Ctx, &validationframework.UniquenessRequest{
			TableName: "fleet_codes",
			ExcludeID: f.codes[1].ID,
			Fields:    []validationframework.FieldCheck{{Column: "code", Value: "FC1"}},
		})
		require.NoError(t, err)
		assert.False(t, self)

		manager, err := checker.CheckReference(a.Ctx, &validationframework.ReferenceRequest{
			TableName: "users",
			ID:        a.Info.UserID,
		})
		require.NoError(t, err)
		assert.True(t, manager)

		foreign, err := checker.CheckReference(a.Ctx, &validationframework.ReferenceRequest{
			TableName: "fleet_codes",
			ID:        f.other.ID,
		})
		require.NoError(t, err)
		assert.False(t, foreign)
	})
}
