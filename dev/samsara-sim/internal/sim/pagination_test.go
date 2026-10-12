package sim

import (
	"encoding/base64"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
)

func collectPages(t *testing.T, records []Record, policy pagePolicy) []Record {
	t.Helper()

	collected := make([]Record, 0, len(records))
	after := ""
	for guard := 0; guard <= len(records)+1; guard++ {
		values := url.Values{}
		if after != "" {
			values.Set("after", after)
		}
		page, pagination, err := paginateRecords(records, values, policy)
		if err != nil {
			t.Fatalf("paginate: %v", err)
		}
		collected = append(collected, page...)
		hasNext, _ := pagination["hasNextPage"].(bool)
		endCursor, _ := pagination["endCursor"].(string)
		if !hasNext {
			if endCursor != "" {
				t.Fatalf("expected empty endCursor on the last page, got %q", endCursor)
			}
			return collected
		}
		if endCursor == "" {
			t.Fatal("expected non-empty endCursor while hasNextPage is true")
		}
		if len(page) == 0 {
			t.Fatal("expected a non-empty page while hasNextPage is true")
		}
		after = endCursor
	}
	t.Fatal("pagination did not terminate")
	return nil
}

func TestPagePolicyTableMatchesSpec(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path string
		want pagePolicy
	}{
		{path: "/addresses", want: limitPage(512)},
		{path: "/assets", want: fixedPage(300)},
		{path: "/assets/location-and-speed/stream", want: limitPage(512)},
		{path: "/dvirs/stream", want: limitPage(200)},
		{path: "/fleet/drivers", want: limitPage(512)},
		{path: "/fleet/dvirs/history", want: limitPage(512)},
		{path: "/fleet/hos/clocks", want: limitPage(512)},
		{path: "/fleet/hos/daily-logs", want: fixedPage(512)},
		{path: "/fleet/hos/logs", want: fixedPage(512)},
		{path: "/fleet/hos/violations", want: fixedPage(512)},
		{path: "/fleet/routes", want: limitPage(512)},
		{path: "/fleet/vehicles/stats", want: fixedPage(512)},
		{path: "/fleet/vehicles/stats/feed", want: fixedPage(512)},
		{path: "/fleet/vehicles/stats/history", want: fixedPage(512)},
		{path: "/form-submissions/stream", want: fixedPage(512)},
		{path: "/form-templates", want: fixedPage(512)},
		{path: "/live-shares", want: limitPage(100)},
		{path: "/webhooks", want: limitPage(512)},
	}
	for _, testCase := range tests {
		request := &http.Request{
			Method:  http.MethodGet,
			Pattern: "GET " + testCase.path,
			URL:     &url.URL{Path: testCase.path},
		}
		if got := pagePolicyFor(request); got != testCase.want {
			t.Fatalf("expected %s policy %+v, got %+v", testCase.path, testCase.want, got)
		}
	}

	unknown := &http.Request{Method: http.MethodGet, URL: &url.URL{Path: "/fleet/new-list"}}
	if got := pagePolicyFor(unknown); got != limitPage(defaultPageSize) {
		t.Fatalf("expected default policy for unlisted endpoints, got %+v", got)
	}
}

func TestEveryPaginatedSimEndpointHasPagePolicy(t *testing.T) {
	t.Parallel()

	srv := newEventTestServer(t, "")
	for _, path := range []string{
		"/addresses",
		"/assets",
		"/assets/location-and-speed/stream",
		"/fleet/drivers",
		"/fleet/dvirs/history",
		"/fleet/hos/clocks",
		"/fleet/hos/daily-logs",
		"/fleet/hos/logs",
		"/fleet/hos/violations",
		"/fleet/drivers/tachograph-files/history",
		"/fleet/vehicles/tachograph-files/history",
		"/fleet/routes",
		"/fleet/vehicles/stats",
		"/fleet/vehicles/stats/feed",
		"/fleet/vehicles/stats/history",
		"/form-submissions/stream",
		"/form-templates",
		"/live-shares",
		"/webhooks",
	} {
		request := &http.Request{Method: http.MethodGet, URL: &url.URL{Path: path}}
		_, pattern := srv.mux.Handler(request)
		if pattern == "" {
			t.Fatalf("expected %s to be routed", path)
		}
		if _, ok := endpointPagePolicies.lookup(patternRouteKey(pattern, http.MethodGet)); !ok {
			t.Fatalf("expected a page policy for %s", pattern)
		}
	}
}

func TestPaginateRejectsOutOfRangeLimitOnlyWhenDeclared(t *testing.T) {
	t.Parallel()

	records := []Record{{"id": "1"}, {"id": "2"}, {"id": "3"}}
	for _, raw := range []string{"0", "-4", "6", "two", "1.5"} {
		_, _, err := paginateRecords(records, url.Values{"limit": {raw}}, limitPage(5))
		if !errors.Is(err, ErrLimitInvalid) {
			t.Fatalf("expected ErrLimitInvalid for limit=%s, got %v", raw, err)
		}
	}

	page, _, err := paginateRecords(records, url.Values{"limit": {"0"}}, fixedPage(2))
	if err != nil {
		t.Fatalf("expected an undeclared limit to be ignored, got %v", err)
	}
	if len(page) != 2 {
		t.Fatalf("expected the fixed page size, got %d", len(page))
	}
}

func TestPaginateCursorResumesIDLessRecords(t *testing.T) {
	t.Parallel()

	simulator := NewLiveSimulator(loadDefaultFixtureStore(t), "pagination-seed")
	now := simulator.anchorTime.Add(6 * time.Hour)
	clocks := simulator.HOSClocks(now, nil)
	if len(clocks) < 3 {
		t.Fatalf("expected several HOS clocks, got %d", len(clocks))
	}
	for _, record := range clocks {
		if recordID(record) != "" {
			t.Fatal("expected HOS clock records without a top-level id")
		}
		if recordIdentityKey(record) == "" {
			t.Fatal("expected a derived identity key for HOS clocks")
		}
	}

	paged := collectPages(t, clocks, fixedPage(1))
	if len(paged) != len(clocks) {
		t.Fatalf("expected %d clocks across pages, got %d", len(clocks), len(paged))
	}
	for idx := range clocks {
		if nestedString(paged[idx], "driver", "id") != nestedString(clocks[idx], "driver", "id") {
			t.Fatalf("page walk diverged at %d", idx)
		}
	}

	logs := simulator.HOSLogs(now, nil, nil, nil)
	if got := collectPages(t, logs, fixedPage(2)); len(got) != len(logs) {
		t.Fatalf("expected %d HOS log records across pages, got %d", len(logs), len(got))
	}
}

func TestPaginateCursorHandlesDuplicateAndKeylessRecords(t *testing.T) {
	t.Parallel()

	records := make([]Record, 0, 9)
	for idx := 0; idx < 3; idx++ {
		records = append(records, Record{"driver": map[string]any{"id": "7"}, "seq": idx})
	}
	for idx := 3; idx < 6; idx++ {
		records = append(records, Record{"payload": idx})
	}
	for idx := 6; idx < 9; idx++ {
		records = append(records, Record{"id": strconv.Itoa(idx)})
	}

	for _, size := range []int{1, 2, 4} {
		paged := collectPages(t, records, fixedPage(size))
		if len(paged) != len(records) {
			t.Fatalf("size %d: expected %d records, got %d", size, len(records), len(paged))
		}
		for idx := range records {
			if !recordsEqual(paged[idx], records[idx]) {
				t.Fatalf("size %d: record %d out of order: %v", size, idx, paged[idx])
			}
		}
	}
}

func TestPaginateCursorSurvivesRemovalOfLastSeenRecord(t *testing.T) {
	t.Parallel()

	records := []Record{{"id": "a"}, {"id": "b"}, {"id": "c"}, {"id": "d"}}
	_, pagination, err := paginateRecords(records, url.Values{}, fixedPage(2))
	if err != nil {
		t.Fatalf("paginate: %v", err)
	}
	endCursor, _ := pagination["endCursor"].(string)

	remaining := []Record{{"id": "a"}, {"id": "c"}, {"id": "d"}}
	page, _, err := paginateRecords(remaining, url.Values{"after": {endCursor}}, fixedPage(2))
	if err != nil {
		t.Fatalf("resume after removal: %v", err)
	}
	if len(page) != 1 || recordID(page[0]) != "d" {
		t.Fatalf("expected resume by offset to continue at d, got %v", page)
	}

	extended := []Record{{"id": "new"}, {"id": "a"}, {"id": "b"}, {"id": "c"}, {"id": "d"}}
	page, _, err = paginateRecords(extended, url.Values{"after": {endCursor}}, fixedPage(2))
	if err != nil {
		t.Fatalf("resume after insertion: %v", err)
	}
	if len(page) != 2 || recordID(page[0]) != "c" {
		t.Fatalf("expected resume by key to continue after b, got %v", page)
	}
}

func TestPaginateRejectsInvalidCursors(t *testing.T) {
	t.Parallel()

	records := []Record{{"id": "a"}, {"id": "b"}}
	wrongVersion := base64.URLEncoding.EncodeToString([]byte(`{"v":9,"o":1}`))
	negative := base64.URLEncoding.EncodeToString([]byte(`{"v":1,"o":-1}`))
	notJSON := base64.URLEncoding.EncodeToString([]byte("plain"))
	for _, cursor := range []string{"missing-cursor", "%%%", wrongVersion, negative, notJSON} {
		_, _, err := paginateRecords(records, url.Values{"after": {cursor}}, fixedPage(1))
		if !errors.Is(err, ErrCursorInvalid) {
			t.Fatalf("expected ErrCursorInvalid for %q, got %v", cursor, err)
		}
	}
}

func TestPaginateLastPageHasEmptyCursor(t *testing.T) {
	t.Parallel()

	records := []Record{{"id": "a"}, {"id": "b"}}
	for _, size := range []int{2, 5} {
		page, pagination, err := paginateRecords(records, url.Values{}, fixedPage(size))
		if err != nil {
			t.Fatalf("paginate: %v", err)
		}
		if len(page) != 2 || pagination["hasNextPage"] != false || pagination["endCursor"] != "" {
			t.Fatalf("expected a complete final page, got %d %v", len(page), pagination)
		}
	}

	empty, pagination, err := paginateRecords([]Record{}, url.Values{}, fixedPage(2))
	if err != nil || len(empty) != 0 || pagination["endCursor"] != "" {
		t.Fatalf("expected an empty final page, got %v %v %v", empty, pagination, err)
	}
}

func recordsEqual(left, right Record) bool {
	leftJSON, leftErr := encodeForComparison(left)
	rightJSON, rightErr := encodeForComparison(right)
	return leftErr == nil && rightErr == nil && leftJSON == rightJSON
}

func TestPaginationIgnoresSortParameters(t *testing.T) {
	records := []Record{{"id": "3"}, {"id": "1"}, {"id": "2"}}
	values := url.Values{"sortBy": {"bogus"}, "sortOrder": {"sideways"}}
	page, _, err := paginateRecords(records, values, limitPage(5))
	if err != nil {
		t.Fatalf("expected unknown sort parameters to be ignored, got %v", err)
	}
	if got := listIDs(page); strings.Join(got, ",") != "3,1,2" {
		t.Fatalf("expected natural order, got %v", got)
	}
}
