package netsuite_test

import (
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/emoss08/trenova/shared/netsuite"
	"github.com/emoss08/trenova/shared/netsuiteerrors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSuiteQLPath = "/services/rest/query/v1/suiteql"

type transactionRow struct {
	ID           string `json:"id"`
	TranID       string `json:"tranid"`
	ExternalID   string `json:"externalid"`
	LastModified string `json:"lastmodified"`
	ForeignTotal string `json:"foreigntotal"`
}

func TestQueryRequestShapeAndDecoding(t *testing.T) {
	t.Parallel()

	const sql = "SELECT id, tranid FROM transaction WHERE (lastmodifieddate > ? OR " +
		"(lastmodifieddate = ? AND id > ?)) ORDER BY lastmodifieddate, id"
	client, _ := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, testSuiteQLPath, r.URL.Path)
		assert.Equal(t, "2", r.URL.Query().Get("limit"))
		assert.Empty(t, r.URL.Query().Get("offset"))
		assert.Equal(t, "transient", r.Header.Get("Prefer"))
		assertAPIHeaders(t, r)
		body := readJSON(t, r)
		assert.Equal(t, sql, body["q"])
		assert.Equal(t, []any{"2026-10-13 14:00:00", "2026-10-13 14:00:00", float64(1200)},
			body["params"])
		_, _ = w.Write(fixture(t, "suiteql_transactions.json"))
	})

	rows := []transactionRow{{ID: "stale"}}
	page, err := client.Query(t.Context(), netsuite.Query{
		SQL:    sql,
		Params: []any{"2026-10-13 14:00:00", "2026-10-13 14:00:00", 1200},
		Limit:  2,
	}, &rows)
	require.NoError(t, err)
	assert.Equal(t, &netsuite.QueryPage{HasMore: true, Count: 2, TotalResults: 5}, page)
	require.Len(t, rows, 2)
	assert.Equal(t, transactionRow{
		ID: "1201", TranID: "INV-1001", ExternalID: "trn-0a1b2c3d",
		LastModified: "2026-10-13 14:04:05", ForeignTotal: "1250.00",
	}, rows[0])
	assert.Equal(t, "980.5", rows[1].ForeignTotal)
}

func TestQueryWithoutParamsDecodesIntoMaps(t *testing.T) {
	t.Parallel()

	client, _ := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "1000", r.URL.Query().Get("limit"))
		body := readJSON(t, r)
		assert.NotContains(t, body, "params")
		_, _ = w.Write(fixture(t, "suiteql_transactions.json"))
	})
	var rows []map[string]any
	page, err := client.Query(t.Context(), netsuite.Query{SQL: "SELECT id FROM transaction"}, &rows)
	require.NoError(t, err)
	assert.True(t, page.HasMore)
	require.Len(t, rows, 2)
	assert.Equal(t, "INV-1002", rows[1]["tranid"])
}

func TestQueryValidates(t *testing.T) {
	t.Parallel()

	client, _ := newAPIClient(t, func(http.ResponseWriter, *http.Request) {
		t.Error("no request is sent")
	})
	var rows []map[string]any
	_, err := client.Query(t.Context(), netsuite.Query{SQL: " "}, &rows)
	require.ErrorIs(t, err, netsuiteerrors.ErrQueryRequired)
	_, err = client.Query(t.Context(), netsuite.Query{SQL: "SELECT 1", Limit: 1001}, &rows)
	require.ErrorIs(t, err, netsuiteerrors.ErrInvalidLimit)
	_, err = client.Query(t.Context(), netsuite.Query{SQL: "SELECT 1", Limit: -1}, &rows)
	require.ErrorIs(t, err, netsuiteerrors.ErrInvalidLimit)
	for _, bad := range []any{nil, rows, &struct{}{}, (*[]map[string]any)(nil)} {
		_, err = client.Query(t.Context(), netsuite.Query{SQL: "SELECT 1"}, bad)
		require.ErrorIs(t, err, netsuiteerrors.ErrInvalidRows)
	}
}

func TestQueryIsRetriedAsARead(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	client, _ := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			serveFixture(t, http.StatusServiceUnavailable, "error_500.json")(w, r)
			return
		}
		_, _ = w.Write(fixture(t, "suiteql_transactions.json"))
	})
	var rows []transactionRow
	_, err := client.Query(t.Context(), netsuite.Query{SQL: "SELECT id FROM transaction"}, &rows)
	require.NoError(t, err)
	assert.Len(t, rows, 2)
	assert.Equal(t, int32(2), calls.Load())
}

func TestQueryUnsupportedTable(t *testing.T) {
	t.Parallel()

	client, _ := newAPIClient(t,
		serveFixture(t, http.StatusBadRequest, "error_unsupported_table.json"))
	var rows []map[string]any
	_, err := client.Query(t.Context(), netsuite.Query{SQL: "SELECT id FROM subsidiary"}, &rows)
	require.Error(t, err)
	assert.True(t, netsuiteerrors.IsUnsupportedTable(err))

	client, _ = newAPIClient(t, serveFixture(t, http.StatusBadRequest, "error_unknown_field.json"))
	_, err = client.Query(t.Context(), netsuite.Query{SQL: "SELECT bogus FROM transaction"}, &rows)
	require.Error(t, err)
	assert.False(t, netsuiteerrors.IsUnsupportedTable(err), "an unknown field is a query bug")
}

func TestQueryTimes(t *testing.T) {
	t.Parallel()

	parsed, err := netsuite.ParseQueryTime("2026-10-13 14:04:05")
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 10, 13, 14, 4, 5, 0, time.UTC), parsed)
	assert.Equal(t, "2026-10-13 14:04:05", netsuite.FormatQueryTime(parsed))

	date, err := netsuite.ParseQueryDate("2026-10-13")
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC), date)

	for _, bad := range []string{"", "10/13/2026 2:04 pm", "2026-10-13T14:04:05Z", "2026-10-13"} {
		_, err = netsuite.ParseQueryTime(bad)
		require.ErrorIs(t, err, netsuiteerrors.ErrInvalidQueryTime, bad)
	}
	_, err = netsuite.ParseQueryDate("10/13/2026")
	require.ErrorIs(t, err, netsuiteerrors.ErrInvalidQueryTime)
}
