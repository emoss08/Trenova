package logs

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type lokiStub struct {
	mu      sync.Mutex
	queries []map[string]string
	fail    int
	status  int
}

func (s *lokiStub) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /loki/api/v1/query_range", func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		q := map[string]string{}
		for k := range r.URL.Query() {
			q[k] = r.URL.Query().Get(k)
		}
		s.queries = append(s.queries, q)
		fail := s.fail > 0
		if fail {
			s.fail--
		}
		s.mu.Unlock()
		if fail {
			http.Error(w, "too many outstanding requests", s.status)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"streams","result":[` +
			`{"stream":{"component":"api","pod":"trenova-api-1","container":"api"},` +
			`"values":[["1794960060000000000","second"],["1794960000000000000","first"]]},` +
			`{"stream":{"component":"api","pod":"trenova-api-2","container":"api"},` +
			`"values":[["1794960030000000000","between"]]}],"stats":{}}}`))
	})
	mux.HandleFunc("GET /loki/api/v1/label/component/values", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":"success","data":["worker","api","api"]}`))
	})
	return mux
}

func newLokiStub(t *testing.T) (*lokiStub, *Loki) {
	t.Helper()
	stub := &lokiStub{}
	srv := httptest.NewServer(stub.handler())
	t.Cleanup(srv.Close)
	loki, err := NewLoki(srv.URL, srv.Client())
	require.NoError(t, err)
	return stub, loki
}

func TestLokiIsAskedNewestFirstForOneComponentAndItsStreamsAreMerged(t *testing.T) {
	t.Parallel()
	stub, loki := newLokiStub(t)
	start := time.Unix(0, 1794950000000000000)
	end := time.Unix(0, 1794970000000000000)

	got, err := loki.NewestFirst(t.Context(), PageQuery{Component: "api", Start: start, End: end, Limit: 2000})

	require.NoError(t, err)
	require.Len(t, got, 3)
	assert.Equal(t, []string{"second", "between", "first"}, []string{got[0].Line, got[1].Line, got[2].Line})
	assert.Equal(t, "trenova-api-2", got[1].Pod)
	assert.Equal(t, "api", got[1].Container)
	q := stub.queries[0]
	assert.Equal(t, `{component="api"}`, q["query"])
	assert.Equal(t, "backward", q["direction"])
	assert.Equal(t, "2000", q["limit"])
	assert.Equal(t, strconv.FormatInt(start.UnixNano(), 10), q["start"])
	assert.Equal(t, strconv.FormatInt(end.UnixNano(), 10), q["end"])
}

func TestLokiIsRetriedWhenBusyAndNotWhenTheQueryIsWrong(t *testing.T) {
	t.Parallel()
	stub, loki := newLokiStub(t)
	stub.fail, stub.status = 2, http.StatusTooManyRequests
	q := PageQuery{Component: "api", Start: time.Unix(0, 0), End: time.Now(), Limit: 10}

	_, err := loki.NewestFirst(t.Context(), q)
	require.NoError(t, err)
	assert.Len(t, stub.queries, 3)

	stub.fail, stub.status = 1, http.StatusBadRequest
	_, err = loki.NewestFirst(t.Context(), q)
	require.Error(t, err)
	assert.Len(t, stub.queries, 4, "a bad request is not retried")

	_, err = loki.NewestFirst(t.Context(), PageQuery{Component: `api"} or {x="`, Limit: 10})
	require.Error(t, err, "a component name never reaches the query unchecked")
}

func TestLokiListsEachSourceOnce(t *testing.T) {
	t.Parallel()
	_, loki := newLokiStub(t)

	got, err := loki.Components(t.Context(), time.Unix(0, 0), time.Now())

	require.NoError(t, err)
	assert.Equal(t, []string{"api", "worker"}, got)
}
