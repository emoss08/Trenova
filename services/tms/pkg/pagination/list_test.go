package pagination

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newTestErrorHandler() *helpers.ErrorHandler {
	logger := zap.NewNop()
	cfg := &config.Config{
		App: config.AppConfig{
			Name:    "test",
			Env:     "test",
			Version: "1.0.0",
			Debug:   true,
		},
	}
	return helpers.NewErrorHandler(helpers.ErrorHandlerParams{
		Logger: logger,
		Config: cfg,
	})
}

func TestList_Success(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/items?limit=10", nil)

	eh := newTestErrorHandler()
	opts := &QueryOptions{
		Pagination: Info{Limit: 10, Offset: 0},
	}

	List[string](c, opts, eh, func() (*ListResult[string], error) {
		return &ListResult[string]{
			Items: []string{"item1", "item2", "item3"},
			Total: 3,
		}, nil
	})

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"count":3`)
	assert.Contains(t, w.Body.String(), `"results":["item1","item2","item3"]`)
}

func TestList_FunctionError(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/items?limit=10", nil)

	eh := newTestErrorHandler()
	opts := &QueryOptions{
		Pagination: Info{Limit: 10, Offset: 0},
	}

	List[string](c, opts, eh, func() (*ListResult[string], error) {
		return nil, errors.New("database error")
	})

	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestList_EmptyResult(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/items?limit=10", nil)

	eh := newTestErrorHandler()
	opts := &QueryOptions{
		Pagination: Info{Limit: 10, Offset: 0},
	}

	List[string](c, opts, eh, func() (*ListResult[string], error) {
		return &ListResult[string]{
			Items: []string{},
			Total: 0,
		}, nil
	})

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"count":0`)
	assert.Contains(t, w.Body.String(), `"results":[]`)
}

func TestList_WithPagination(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(
		http.MethodGet,
		"http://example.com/api/items?limit=2",
		nil,
	)

	eh := newTestErrorHandler()
	opts := &QueryOptions{
		Pagination: Info{Limit: 2, Offset: 0},
	}

	List[string](c, opts, eh, func() (*ListResult[string], error) {
		return &ListResult[string]{
			Items: []string{"a", "b"},
			Total: 10,
		}, nil
	})

	require.Equal(t, http.StatusOK, w.Code)

	var response CursorResponse[[]string]
	require.NoError(t, sonic.Unmarshal(w.Body.Bytes(), &response))
	assert.Equal(t, 10, response.Count)
	require.NotNil(t, response.TotalCount)
	assert.Equal(t, 10, *response.TotalCount)
	assert.Len(t, response.Results, 2)

	var fields map[string]any
	require.NoError(t, sonic.Unmarshal(w.Body.Bytes(), &fields))
	assert.Contains(t, fields, "next")
}

func TestList_RejectsOffset(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(
		http.MethodGet,
		"http://example.com/api/items?limit=10&offset=90",
		nil,
	)

	eh := newTestErrorHandler()
	opts := &QueryOptions{
		Pagination: Info{Limit: 10, Offset: 90},
	}

	List[string](c, opts, eh, func() (*ListResult[string], error) {
		return &ListResult[string]{
			Items: []string{"last"},
			Total: 91,
		}, nil
	})

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Offset is not supported")
}

func TestList_WithFilters(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(
		http.MethodGet,
		`/api/items?limit=10&fieldFilters=[{"field":"name","operator":"eq","value":"test"}]`,
		nil,
	)

	eh := newTestErrorHandler()
	opts := &QueryOptions{
		Pagination: Info{Limit: 10, Offset: 0},
	}

	List[string](c, opts, eh, func() (*ListResult[string], error) {
		return &ListResult[string]{
			Items: []string{"test"},
			Total: 1,
		}, nil
	})

	assert.Equal(t, http.StatusOK, w.Code)
	require.Len(t, opts.FieldFilters, 1)
	assert.Equal(t, "name", opts.FieldFilters[0].Field)
}

func TestList_WithSort(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(
		http.MethodGet,
		`/api/items?limit=10&sort=[{"field":"name","direction":"desc"}]`,
		nil,
	)

	eh := newTestErrorHandler()
	opts := &QueryOptions{
		Pagination: Info{Limit: 10, Offset: 0},
	}

	List[string](c, opts, eh, func() (*ListResult[string], error) {
		return &ListResult[string]{
			Items: []string{"z", "a"},
			Total: 2,
		}, nil
	})

	assert.Equal(t, http.StatusOK, w.Code)
	require.Len(t, opts.Sort, 1)
	assert.Equal(t, "name", opts.Sort[0].Field)
}

func TestList_WithGeoFilters(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(
		http.MethodGet,
		`/api/items?limit=10&geoFilters=[{"field":"location","center":{"latitude":40.7,"longitude":-74.0},"radiusKm":10}]`,
		nil,
	)

	eh := newTestErrorHandler()
	opts := &QueryOptions{
		Pagination: Info{Limit: 10, Offset: 0},
	}

	List[string](c, opts, eh, func() (*ListResult[string], error) {
		return &ListResult[string]{
			Items: []string{"nearby"},
			Total: 1,
		}, nil
	})

	assert.Equal(t, http.StatusOK, w.Code)
	require.Len(t, opts.GeoFilters, 1)
}

func TestList_WithAggregateFilters(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(
		http.MethodGet,
		`/api/items?limit=10&aggregateFilters=[{"relation":"stops","operator":"countgt","value":2}]`,
		nil,
	)

	eh := newTestErrorHandler()
	opts := &QueryOptions{
		Pagination: Info{Limit: 10, Offset: 0},
	}

	List[string](c, opts, eh, func() (*ListResult[string], error) {
		return &ListResult[string]{
			Items: []string{"multi-stop"},
			Total: 1,
		}, nil
	})

	assert.Equal(t, http.StatusOK, w.Code)
	require.Len(t, opts.AggregateFilters, 1)
}

func TestList_WithFilterGroups(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(
		http.MethodGet,
		`/api/items?limit=10&filterGroups=[{"filters":[{"field":"status","operator":"eq","value":"active"}]}]`,
		nil,
	)

	eh := newTestErrorHandler()
	opts := &QueryOptions{
		Pagination: Info{Limit: 10, Offset: 0},
	}

	List[string](c, opts, eh, func() (*ListResult[string], error) {
		return &ListResult[string]{
			Items: []string{"active-item"},
			Total: 1,
		}, nil
	})

	assert.Equal(t, http.StatusOK, w.Code)
	require.Len(t, opts.FilterGroups, 1)
}

func TestList_OffsetDoesNotReturnPreviousPageURL(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(
		http.MethodGet,
		"http://example.com/api/items?limit=10&offset=20",
		nil,
	)

	eh := newTestErrorHandler()
	opts := &QueryOptions{
		Pagination: Info{Limit: 10, Offset: 20},
	}

	List[string](c, opts, eh, func() (*ListResult[string], error) {
		return &ListResult[string]{
			Items: []string{"item"},
			Total: 100,
		}, nil
	})

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Offset is not supported")
}

func TestSelectOptions_Success(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/select-options", nil)

	eh := newTestErrorHandler()
	req := &SelectQueryRequest{
		Pagination: Info{Limit: 10, Offset: 0},
	}

	SelectOptions[string](c, req, eh, func() (*ListResult[string], error) {
		return &ListResult[string]{
			Items: []string{"opt1", "opt2"},
			Total: 2,
		}, nil
	})

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"count":2`)
	assert.Contains(t, w.Body.String(), `"results":["opt1","opt2"]`)
}

func TestSelectOptions_FunctionError(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/select-options", nil)

	eh := newTestErrorHandler()
	req := &SelectQueryRequest{
		Pagination: Info{Limit: 10, Offset: 0},
	}

	SelectOptions[string](c, req, eh, func() (*ListResult[string], error) {
		return nil, errors.New("fetch error")
	})

	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestSelectOptions_EmptyResult(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/select-options", nil)

	eh := newTestErrorHandler()
	req := &SelectQueryRequest{
		Pagination: Info{Limit: 10, Offset: 0},
	}

	SelectOptions[string](c, req, eh, func() (*ListResult[string], error) {
		return &ListResult[string]{
			Items: []string{},
			Total: 0,
		}, nil
	})

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"count":0`)
	assert.Contains(t, w.Body.String(), `"results":[]`)
}

func TestSelectOptions_WithNextPage(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(
		http.MethodGet,
		"http://example.com/api/select-options?limit=5",
		nil,
	)

	eh := newTestErrorHandler()
	req := &SelectQueryRequest{
		Pagination: Info{Limit: 5, Offset: 0},
	}

	SelectOptions[string](c, req, eh, func() (*ListResult[string], error) {
		return &ListResult[string]{
			Items: []string{"a", "b", "c", "d", "e"},
			Total: 20,
		}, nil
	})

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"next"`)
}

func TestSelectOptions_WithPreviousPage(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(
		http.MethodGet,
		"http://example.com/api/select-options?limit=5&offset=10",
		nil,
	)

	eh := newTestErrorHandler()
	req := &SelectQueryRequest{
		Pagination: Info{Limit: 5, Offset: 10},
	}

	SelectOptions[string](c, req, eh, func() (*ListResult[string], error) {
		return &ListResult[string]{
			Items: []string{"f", "g"},
			Total: 12,
		}, nil
	})

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"previous"`)
}

func TestSelectOptions_NoMorePages(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(
		http.MethodGet,
		"http://example.com/api/select-options?limit=10",
		nil,
	)

	eh := newTestErrorHandler()
	req := &SelectQueryRequest{
		Pagination: Info{Limit: 10, Offset: 0},
	}

	SelectOptions[string](c, req, eh, func() (*ListResult[string], error) {
		return &ListResult[string]{
			Items: []string{"only"},
			Total: 1,
		}, nil
	})

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"next":""`)
	assert.Contains(t, w.Body.String(), `"previous":""`)
}

type testItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type cursorTestItem struct {
	ID        pulid.ID `json:"id"`
	CreatedAt int64    `json:"createdAt"`
	Name      string   `json:"name"`
}

type cursorTestItemWithoutID struct {
	Name string `json:"name"`
}

func TestList_StructType(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/items?limit=10", nil)

	eh := newTestErrorHandler()
	opts := &QueryOptions{
		Pagination: Info{Limit: 10, Offset: 0},
	}

	List[testItem](c, opts, eh, func() (*ListResult[testItem], error) {
		return &ListResult[testItem]{
			Items: []testItem{
				{ID: "1", Name: "First"},
				{ID: "2", Name: "Second"},
			},
			Total: 2,
		}, nil
	})

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"id":"1"`)
	assert.Contains(t, w.Body.String(), `"name":"First"`)
}

func TestList_UsesQueryOptionsCursorSortForEndCursor(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(
		http.MethodGet,
		`http://example.com/api/items?limit=2&sort=[{"field":"name","direction":"asc"}]`,
		nil,
	)

	eh := newTestErrorHandler()
	opts := &QueryOptions{
		Pagination: Info{Limit: 2, Offset: 0},
	}
	lastID := pulid.MustNew("item_")

	List[cursorTestItem](c, opts, eh, func() (*ListResult[cursorTestItem], error) {
		opts.CursorSort = []CursorSortField{
			{Field: "name", Direction: "asc"},
			{Field: "id", Direction: "asc"},
		}

		return &ListResult[cursorTestItem]{
			Items: []cursorTestItem{
				{ID: pulid.MustNew("item_"), CreatedAt: 1710000000000, Name: "Alpha"},
				{ID: lastID, CreatedAt: 1710000001000, Name: "Beta"},
			},
			Total: 2,
		}, nil
	})

	require.Equal(t, http.StatusOK, w.Code)

	var response struct {
		EndCursor string `json:"endCursor"`
	}
	require.NoError(t, sonic.Unmarshal(w.Body.Bytes(), &response))
	require.NotEmpty(t, response.EndCursor)

	cursor, err := DecodeCursor(response.EndCursor)
	require.NoError(t, err)
	assert.Equal(t, lastID, cursor.ID)
	assert.Equal(t, []CursorSortField{
		{Field: "name", Direction: "asc"},
		{Field: "id", Direction: "asc"},
	}, cursor.Sort)
	assert.Equal(t, []any{"Beta", lastID.String()}, cursor.Values)
}

func TestList_OverfetchesAndTrimsCursorPage(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "http://example.com/api/items?limit=2", nil)

	eh := newTestErrorHandler()
	opts := &QueryOptions{
		Pagination: Info{Limit: 2, Offset: 0},
	}

	List[cursorTestItem](c, opts, eh, func() (*ListResult[cursorTestItem], error) {
		assert.Equal(t, 3, opts.Pagination.SafeLimit())

		return &ListResult[cursorTestItem]{
			Items: []cursorTestItem{
				{ID: pulid.MustNew("item_"), CreatedAt: 1710000000000, Name: "Alpha"},
				{ID: pulid.MustNew("item_"), CreatedAt: 1710000001000, Name: "Beta"},
				{ID: pulid.MustNew("item_"), CreatedAt: 1710000002000, Name: "Gamma"},
			},
			Total: 3,
		}, nil
	})

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"count":3`)
	assert.Contains(t, w.Body.String(), `"totalCount":3`)
	assert.Contains(t, w.Body.String(), `"hasNextPage":true`)
	assert.Contains(t, w.Body.String(), `"name":"Alpha"`)
	assert.Contains(t, w.Body.String(), `"name":"Beta"`)
	assert.NotContains(t, w.Body.String(), `"name":"Gamma"`)

	var response struct {
		Next string `json:"next"`
	}
	require.NoError(t, sonic.Unmarshal(w.Body.Bytes(), &response))
	assert.Contains(t, response.Next, "after=")
	assert.Contains(t, response.Next, "limit=2")
}

func TestCursorList_UsesTotalCountForRESTCursorResponseCount(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "http://example.com/api/items?limit=2", nil)

	eh := newTestErrorHandler()
	opts := &QueryOptions{
		Pagination: Info{Limit: 2, Offset: 0},
	}
	total := 42

	CursorList[cursorTestItem](c, opts, eh, func(_ CursorInfo) (*CursorListResult[cursorTestItem], error) {
		return &CursorListResult[cursorTestItem]{
			Items: []cursorTestItem{
				{ID: pulid.MustNew("item_"), CreatedAt: 1710000000000, Name: "Alpha"},
				{ID: pulid.MustNew("item_"), CreatedAt: 1710000001000, Name: "Beta"},
			},
			TotalCount: &total,
		}, nil
	})

	require.Equal(t, http.StatusOK, w.Code)

	var response CursorResponse[[]cursorTestItem]
	require.NoError(t, sonic.Unmarshal(w.Body.Bytes(), &response))
	assert.Equal(t, total, response.Count)
	require.NotNil(t, response.TotalCount)
	assert.Equal(t, total, *response.TotalCount)
	assert.Len(t, response.Results, 2)
}

func TestCursorList_FallsBackToPageLengthWhenTotalCountIsUnavailable(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "http://example.com/api/items?limit=2", nil)

	eh := newTestErrorHandler()
	opts := &QueryOptions{
		Pagination: Info{Limit: 2, Offset: 0},
	}

	CursorList[cursorTestItem](c, opts, eh, func(_ CursorInfo) (*CursorListResult[cursorTestItem], error) {
		return &CursorListResult[cursorTestItem]{
			Items: []cursorTestItem{
				{ID: pulid.MustNew("item_"), CreatedAt: 1710000000000, Name: "Alpha"},
				{ID: pulid.MustNew("item_"), CreatedAt: 1710000001000, Name: "Beta"},
			},
		}, nil
	})

	require.Equal(t, http.StatusOK, w.Code)

	var response CursorResponse[[]cursorTestItem]
	require.NoError(t, sonic.Unmarshal(w.Body.Bytes(), &response))
	assert.Equal(t, 2, response.Count)
	assert.Nil(t, response.TotalCount)
	assert.Contains(t, w.Body.String(), `"totalCount":null`)
}

func TestList_ReturnsCursorValidationError(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/items?limit=10", nil)

	eh := newTestErrorHandler()
	opts := &QueryOptions{
		Pagination: Info{Limit: 10, Offset: 0},
	}

	List[cursorTestItem](c, opts, eh, func() (*ListResult[cursorTestItem], error) {
		opts.CursorError = errors.New("cursor sort does not match request sort")

		return &ListResult[cursorTestItem]{
			Items: []cursorTestItem{
				{ID: pulid.MustNew("item_"), CreatedAt: 1710000000000, Name: "Alpha"},
			},
			Total: 1,
		}, nil
	})

	assert.NotEqual(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "cursor sort does not match request sort")
}

func TestList_ReturnsCursorEncodeError(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/items?limit=10", nil)

	eh := newTestErrorHandler()
	opts := &QueryOptions{
		Pagination: Info{Limit: 10, Offset: 0},
	}

	List[cursorTestItemWithoutID](c, opts, eh, func() (*ListResult[cursorTestItemWithoutID], error) {
		opts.CursorSort = []CursorSortField{
			{Field: "name", Direction: "asc"},
			{Field: "id", Direction: "asc"},
		}

		return &ListResult[cursorTestItemWithoutID]{
			Items: []cursorTestItemWithoutID{{Name: "Alpha"}},
			Total: 1,
		}, nil
	})

	assert.NotEqual(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "cursor field")
}

func TestSelectOptions_StructType(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/select-options", nil)

	eh := newTestErrorHandler()
	req := &SelectQueryRequest{
		Pagination: Info{Limit: 10, Offset: 0},
	}

	SelectOptions[testItem](c, req, eh, func() (*ListResult[testItem], error) {
		return &ListResult[testItem]{
			Items: []testItem{
				{ID: "x", Name: "Option X"},
			},
			Total: 1,
		}, nil
	})

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"id":"x"`)
	assert.Contains(t, w.Body.String(), `"name":"Option X"`)
}

func TestList_BindQueryError(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/items?limit=-5&offset=-10", nil)

	eh := newTestErrorHandler()
	opts := &QueryOptions{}

	List[string](c, opts, eh, func() (*ListResult[string], error) {
		return &ListResult[string]{
			Items: []string{"should not reach"},
			Total: 1,
		}, nil
	})

	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestNormalizeSlice_BooleanTypes(t *testing.T) {
	t.Parallel()

	input := []any{true, false, true}
	result := normalizeSlice(input)

	_, ok := result.([]any)
	assert.True(t, ok)
}

func TestNormalizeSlice_NumbersAndNils(t *testing.T) {
	t.Parallel()

	input := []any{float64(1), nil, float64(3)}
	result := normalizeSlice(input)

	intSlice, ok := result.([]int64)
	assert.True(t, ok)
	assert.Equal(t, []int64{1, 3}, intSlice)
}

func TestNormalizeSlice_StringsAndNils(t *testing.T) {
	t.Parallel()

	input := []any{"hello", nil, "world"}
	result := normalizeSlice(input)

	strSlice, ok := result.([]string)
	assert.True(t, ok)
	assert.Equal(t, []string{"hello", "world"}, strSlice)
}
