package pagination

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newTestAuthContext() *authctx.AuthContext {
	return &authctx.AuthContext{
		UserID:         pulid.MustNew("usr"),
		BusinessUnitID: pulid.MustNew("bu"),
		OrganizationID: pulid.MustNew("org"),
	}
}

func createTestContextWithParams(queryParams url.Values) *gin.Context {
	req := httptest.NewRequest(http.MethodGet, "/?"+queryParams.Encode(), nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	return c
}

func TestNewQueryOptions_DefaultValues(t *testing.T) {
	t.Parallel()

	params := url.Values{}
	c := createTestContextWithParams(params)
	authCtx := newTestAuthContext()

	opts := NewQueryOptions(c, authCtx)

	require.NotNil(t, opts)
	assert.Equal(t, authCtx.OrganizationID, opts.TenantInfo.OrgID)
	assert.Equal(t, authCtx.BusinessUnitID, opts.TenantInfo.BuID)
	assert.Equal(t, authCtx.UserID, opts.TenantInfo.UserID)
	assert.Equal(t, DefaultLimit, opts.Pagination.Limit)
	assert.Equal(t, DefaultOffset, opts.Pagination.Offset)
	assert.Empty(t, opts.Query)
}

func TestNewQueryOptions_WithQueryParam(t *testing.T) {
	t.Parallel()

	params := url.Values{}
	params.Set("query", "search term")
	c := createTestContextWithParams(params)
	authCtx := newTestAuthContext()

	opts := NewQueryOptions(c, authCtx)

	assert.Equal(t, "search term", opts.Query)
}

func TestNewQueryOptions_WithPagination(t *testing.T) {
	t.Parallel()

	params := url.Values{}
	params.Set("limit", "50")
	params.Set("offset", "10")
	c := createTestContextWithParams(params)
	authCtx := newTestAuthContext()

	opts := NewQueryOptions(c, authCtx)

	assert.Equal(t, 50, opts.Pagination.Limit)
	assert.Equal(t, 10, opts.Pagination.Offset)
}

func TestNewQueryOptions_EmptyQuery(t *testing.T) {
	t.Parallel()

	params := url.Values{}
	params.Set("query", "")
	c := createTestContextWithParams(params)
	authCtx := newTestAuthContext()

	opts := NewQueryOptions(c, authCtx)

	assert.Empty(t, opts.Query)
}

func TestNewSelectQueryRequest_DefaultValues(t *testing.T) {
	t.Parallel()

	params := url.Values{}
	c := createTestContextWithParams(params)
	authCtx := newTestAuthContext()

	req := NewSelectQueryRequest(c, authCtx)

	require.NotNil(t, req)
	assert.Equal(t, authCtx.OrganizationID, req.TenantInfo.OrgID)
	assert.Equal(t, authCtx.BusinessUnitID, req.TenantInfo.BuID)
	assert.Equal(t, authCtx.UserID, req.TenantInfo.UserID)
	assert.Equal(t, DefaultLimit, req.Pagination.Limit)
	assert.Equal(t, DefaultOffset, req.Pagination.Offset)
	assert.Empty(t, req.Query)
}

func TestNewSelectQueryRequest_WithQueryParam(t *testing.T) {
	t.Parallel()

	params := url.Values{}
	params.Set("query", "find me")
	c := createTestContextWithParams(params)
	authCtx := newTestAuthContext()

	req := NewSelectQueryRequest(c, authCtx)

	assert.Equal(t, "find me", req.Query)
}

func TestNewSelectQueryRequest_WithPagination(t *testing.T) {
	t.Parallel()

	params := url.Values{}
	params.Set("limit", "25")
	params.Set("offset", "5")
	c := createTestContextWithParams(params)
	authCtx := newTestAuthContext()

	req := NewSelectQueryRequest(c, authCtx)

	assert.Equal(t, 25, req.Pagination.Limit)
	assert.Equal(t, 5, req.Pagination.Offset)
}

func TestParams_DefaultValues(t *testing.T) {
	t.Parallel()

	params := url.Values{}
	c := createTestContextWithParams(params)

	info, err := Params(c)

	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, DefaultOffset, info.Offset)
	assert.Equal(t, DefaultLimit, info.Limit)
}

func TestParams_CustomValues(t *testing.T) {
	t.Parallel()

	params := url.Values{}
	params.Set("limit", "50")
	params.Set("offset", "20")
	c := createTestContextWithParams(params)

	info, err := Params(c)

	require.NoError(t, err)
	assert.Equal(t, 20, info.Offset)
	assert.Equal(t, 50, info.Limit)
}

func TestParams_NegativeOffset(t *testing.T) {
	t.Parallel()

	params := url.Values{}
	params.Set("offset", "-5")
	c := createTestContextWithParams(params)

	info, err := Params(c)

	require.NoError(t, err)
	assert.Equal(t, 0, info.Offset)
}

func TestParams_LimitBelowMin(t *testing.T) {
	t.Parallel()

	params := url.Values{}
	params.Set("limit", "0")
	c := createTestContextWithParams(params)

	info, err := Params(c)

	require.NoError(t, err)
	assert.Equal(t, 1, info.Limit)
}

func TestParams_LimitAboveMax(t *testing.T) {
	t.Parallel()

	params := url.Values{}
	params.Set("limit", "500")
	c := createTestContextWithParams(params)

	info, err := Params(c)

	require.NoError(t, err)
	assert.Equal(t, 100, info.Limit)
}

func TestParams_InvalidLimit(t *testing.T) {
	t.Parallel()

	params := url.Values{}
	params.Set("limit", "abc")
	c := createTestContextWithParams(params)

	info, err := Params(c)

	require.NoError(t, err)
	assert.Equal(t, DefaultLimit, info.Limit)
}

func TestParams_InvalidOffset(t *testing.T) {
	t.Parallel()

	params := url.Values{}
	params.Set("offset", "abc")
	c := createTestContextWithParams(params)

	info, err := Params(c)

	require.NoError(t, err)
	assert.Equal(t, 0, info.Offset)
}

func TestBuildPageURL_HTTP(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/items?existing=param", nil)

	result := buildPageURL(req, 20, 10)

	assert.Contains(t, result, "http://")
	assert.Contains(t, result, "example.com")
	assert.Contains(t, result, "/api/items")
	assert.Contains(t, result, "offset=20")
	assert.Contains(t, result, "limit=10")
	assert.Contains(t, result, "existing=param")
}

func TestBuildPageURL_HTTPS_TLS(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "https://example.com/api/items", nil)
	req.TLS = &tls.ConnectionState{}

	result := buildPageURL(req, 10, 20)

	assert.Contains(t, result, "https://")
}

func TestBuildPageURL_HTTPS_XForwardedProto(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/items", nil)
	req.Header.Set("X-Forwarded-Proto", "https")

	result := buildPageURL(req, 10, 20)

	assert.Contains(t, result, "https://")
}

func TestBuildPageURL_OverwritesExistingPagination(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(
		http.MethodGet,
		"http://example.com/api/items?offset=0&limit=10",
		nil,
	)

	result := buildPageURL(req, 30, 15)

	assert.Contains(t, result, "offset=30")
	assert.Contains(t, result, "limit=15")
}

func TestGetNextPageURL_MiddlePage(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/items", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	result := GetNextPageURL(c, 10, 20, 100)

	assert.NotEmpty(t, result)
	assert.Contains(t, result, "offset=30")
	assert.Contains(t, result, "limit=10")
}

func TestGetNextPageURL_LastPage(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/items", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	result := GetNextPageURL(c, 10, 90, 100)

	assert.Empty(t, result)
}

func TestGetNextPageURL_ExactlyAtEnd(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/items", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	result := GetNextPageURL(c, 10, 100, 100)

	assert.Empty(t, result)
}

func TestGetNextPageURL_FirstPage(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/items", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	result := GetNextPageURL(c, 10, 0, 100)

	assert.NotEmpty(t, result)
	assert.Contains(t, result, "offset=10")
}

func TestGetNextPageURL_SingleItem(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/items", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	result := GetNextPageURL(c, 10, 0, 1)

	assert.Empty(t, result)
}

func TestGetPreviousPageURL_MiddlePage(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/items", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	result := GetPreviousPageURL(c, 10, 30)

	assert.NotEmpty(t, result)
	assert.Contains(t, result, "offset=20")
	assert.Contains(t, result, "limit=10")
}

func TestGetPreviousPageURL_SecondPage(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/items", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	result := GetPreviousPageURL(c, 10, 10)

	assert.NotEmpty(t, result)
	assert.Contains(t, result, "offset=0")
}

func TestGetPreviousPageURL_OffsetLessThanLimit(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/items", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	result := GetPreviousPageURL(c, 10, 5)

	assert.NotEmpty(t, result)
	assert.Contains(t, result, "offset=0")
}

func TestGetPreviousPageURL_AtStart(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/items", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	result := GetPreviousPageURL(c, 10, 0)

	assert.Empty(t, result)
}

func TestNormalizeFilterValue_Float64WholeNumber(t *testing.T) {
	t.Parallel()

	result := normalizeFilterValue(float64(42), "eq")

	assert.Equal(t, int64(42), result)
}

func TestNormalizeFilterValue_Float64Decimal(t *testing.T) {
	t.Parallel()

	result := normalizeFilterValue(float64(3.14), "eq")

	assert.Equal(t, 3.14, result)
}

func TestNormalizeFilterValue_Map(t *testing.T) {
	t.Parallel()

	input := map[string]any{"from": float64(100), "to": float64(200)}
	result := normalizeFilterValue(input, "daterange")

	resultMap, ok := result.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(100), resultMap["from"])
	assert.Equal(t, float64(200), resultMap["to"])
}

func TestNormalizeFilterValue_String_InOperator(t *testing.T) {
	t.Parallel()

	result := normalizeFilterValue("a,b,c", "in")

	arr, ok := result.([]string)
	require.True(t, ok)
	assert.Equal(t, []string{"a", "b", "c"}, arr)
}

func TestNormalizeFilterValue_String_DefaultOperator(t *testing.T) {
	t.Parallel()

	result := normalizeFilterValue("hello", "eq")

	assert.Equal(t, "hello", result)
}

func TestNormalizeFilterValue_UnknownType(t *testing.T) {
	t.Parallel()

	result := normalizeFilterValue(true, "eq")

	assert.Equal(t, true, result)
}

func TestNormalizeFilterValue_NilValue(t *testing.T) {
	t.Parallel()

	result := normalizeFilterValue(nil, "eq")

	assert.Nil(t, result)
}

func TestNormalizeSlice_SingleElement(t *testing.T) {
	t.Parallel()

	input := []any{"only"}
	result := normalizeSlice(input)

	strSlice, ok := result.([]string)
	require.True(t, ok)
	assert.Equal(t, []string{"only"}, strSlice)
}

func TestNormalizeSlice_SingleNumberElement(t *testing.T) {
	t.Parallel()

	input := []any{float64(99)}
	result := normalizeSlice(input)

	intSlice, ok := result.([]int64)
	require.True(t, ok)
	assert.Equal(t, []int64{99}, intSlice)
}

func TestTenantInfo_Fields(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org")
	buID := pulid.MustNew("bu")
	userID := pulid.MustNew("usr")

	info := TenantInfo{
		OrgID:  orgID,
		BuID:   buID,
		UserID: userID,
	}

	assert.Equal(t, orgID, info.OrgID)
	assert.Equal(t, buID, info.BuID)
	assert.Equal(t, userID, info.UserID)
}

func TestListResult_Fields(t *testing.T) {
	t.Parallel()

	result := ListResult[string]{
		Items: []string{"a", "b", "c"},
		Total: 3,
	}

	assert.Len(t, result.Items, 3)
	assert.Equal(t, 3, result.Total)
}

func TestListResult_Empty(t *testing.T) {
	t.Parallel()

	result := ListResult[int]{
		Items: []int{},
		Total: 0,
	}

	assert.Empty(t, result.Items)
	assert.Equal(t, 0, result.Total)
}

func TestInfo_Fields(t *testing.T) {
	t.Parallel()

	info := Info{
		Limit:  25,
		Offset: 50,
	}

	assert.Equal(t, 25, info.Limit)
	assert.Equal(t, 50, info.Offset)
}

func TestResponse_Fields(t *testing.T) {
	t.Parallel()

	resp := Response[[]string]{
		Results: []string{"a", "b"},
		Count:   10,
		Next:    "http://example.com?offset=10",
		Prev:    "http://example.com?offset=0",
	}

	assert.Len(t, resp.Results, 2)
	assert.Equal(t, 10, resp.Count)
	assert.NotEmpty(t, resp.Next)
	assert.NotEmpty(t, resp.Prev)
}

func TestResponse_EmptyResults(t *testing.T) {
	t.Parallel()

	resp := Response[[]string]{
		Results: []string{},
		Count:   0,
		Next:    "",
		Prev:    "",
	}

	assert.Empty(t, resp.Results)
	assert.Equal(t, 0, resp.Count)
	assert.Empty(t, resp.Next)
	assert.Empty(t, resp.Prev)
}

func TestSelectQueryRequest_Fields(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org")
	buID := pulid.MustNew("bu")
	userID := pulid.MustNew("usr")

	req := SelectQueryRequest{
		TenantInfo: TenantInfo{
			OrgID:  orgID,
			BuID:   buID,
			UserID: userID,
		},
		Pagination: Info{
			Limit:  30,
			Offset: 60,
		},
		Query: "test search",
	}

	assert.Equal(t, orgID, req.TenantInfo.OrgID)
	assert.Equal(t, 30, req.Pagination.Limit)
	assert.Equal(t, "test search", req.Query)
}

func TestConstants(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 20, DefaultLimit)
	assert.Equal(t, 0, DefaultOffset)
	assert.Equal(t, 100, MaxLimit)
}

func TestNewQueryOptions_LargeLimit(t *testing.T) {
	t.Parallel()

	params := url.Values{}
	params.Set("limit", "200")
	c := createTestContextWithParams(params)
	authCtx := newTestAuthContext()

	opts := NewQueryOptions(c, authCtx)

	assert.Equal(t, MaxLimit, opts.Pagination.Limit)
}

func TestNewQueryOptions_ZeroLimit(t *testing.T) {
	t.Parallel()

	params := url.Values{}
	params.Set("limit", "0")
	c := createTestContextWithParams(params)
	authCtx := newTestAuthContext()

	opts := NewQueryOptions(c, authCtx)

	require.NotNil(t, opts)
	assert.Equal(t, DefaultLimit, opts.Pagination.Limit)
}

func TestNewQueryOptions_NegativeOffset(t *testing.T) {
	t.Parallel()

	params := url.Values{}
	params.Set("offset", "-10")
	c := createTestContextWithParams(params)
	authCtx := newTestAuthContext()

	opts := NewQueryOptions(c, authCtx)

	require.NotNil(t, opts)
	assert.Equal(t, DefaultOffset, opts.Pagination.Offset)
}

func TestNewSelectQueryRequest_LargeOffset(t *testing.T) {
	t.Parallel()

	params := url.Values{}
	params.Set("offset", "10000")
	c := createTestContextWithParams(params)
	authCtx := newTestAuthContext()

	req := NewSelectQueryRequest(c, authCtx)

	assert.Equal(t, 10000, req.Pagination.Offset)
}

func TestNewSelectQueryRequest_LargeLimit(t *testing.T) {
	t.Parallel()

	params := url.Values{}
	params.Set("limit", "500")
	c := createTestContextWithParams(params)
	authCtx := newTestAuthContext()

	req := NewSelectQueryRequest(c, authCtx)

	assert.Equal(t, MaxLimit, req.Pagination.Limit)
}

func TestNewSelectQueryRequest_ZeroLimit(t *testing.T) {
	t.Parallel()

	params := url.Values{}
	params.Set("limit", "0")
	c := createTestContextWithParams(params)
	authCtx := newTestAuthContext()

	req := NewSelectQueryRequest(c, authCtx)

	assert.Equal(t, DefaultLimit, req.Pagination.Limit)
}

func TestNewSelectQueryRequest_NegativeOffset(t *testing.T) {
	t.Parallel()

	params := url.Values{}
	params.Set("offset", "-100")
	c := createTestContextWithParams(params)
	authCtx := newTestAuthContext()

	req := NewSelectQueryRequest(c, authCtx)

	assert.Equal(t, DefaultOffset, req.Pagination.Offset)
}

func TestClampLimit(t *testing.T) {
	t.Parallel()

	assert.Equal(t, DefaultLimit, ClampLimit(0))
	assert.Equal(t, DefaultLimit, ClampLimit(-1))
	assert.Equal(t, MaxLimit, ClampLimit(MaxLimit+1))
	assert.Equal(t, 25, ClampLimit(25))
}

func TestClampOffset(t *testing.T) {
	t.Parallel()

	assert.Equal(t, DefaultOffset, ClampOffset(-1))
	assert.Equal(t, 42, ClampOffset(42))
}

func TestQueryOptions_Fields(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org")
	buID := pulid.MustNew("bu")

	opts := QueryOptions{
		TenantInfo: TenantInfo{
			OrgID: orgID,
			BuID:  buID,
		},
		Pagination: Info{
			Limit:  10,
			Offset: 0,
		},
		Query: "test",
	}

	assert.Equal(t, orgID, opts.TenantInfo.OrgID)
	assert.Equal(t, buID, opts.TenantInfo.BuID)
	assert.Equal(t, 10, opts.Pagination.Limit)
	assert.Equal(t, "test", opts.Query)
	assert.Empty(t, opts.FieldFilters)
	assert.Empty(t, opts.FilterGroups)
	assert.Empty(t, opts.GeoFilters)
	assert.Empty(t, opts.AggregateFilters)
	assert.Empty(t, opts.Sort)
}

func TestBuildPageURL_PreservesQueryParams(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(
		http.MethodGet,
		"http://example.com/api/items?status=active&name=test",
		nil,
	)

	result := buildPageURL(req, 10, 20)

	assert.Contains(t, result, "status=active")
	assert.Contains(t, result, "name=test")
	assert.Contains(t, result, "offset=10")
	assert.Contains(t, result, "limit=20")
}

func TestBuildPageURL_ZeroOffset(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/items", nil)

	result := buildPageURL(req, 0, 10)

	assert.Contains(t, result, "offset=0")
	assert.Contains(t, result, "limit=10")
}
