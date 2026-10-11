package sim

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/bytedance/sonic"
)

const (
	defaultPageSize   = 512
	pageCursorVersion = 1
)

type pagePolicy struct {
	Size       int
	Max        int
	LimitParam bool
}

func limitPage(size int) pagePolicy {
	return pagePolicy{Size: size, Max: size, LimitParam: true}
}

func fixedPage(size int) pagePolicy {
	return pagePolicy{Size: size, Max: size}
}

var endpointPagePolicies = newRouteTable([]routeEntry[pagePolicy]{
	{http.MethodGet, "/addresses", limitPage(512)},
	{http.MethodGet, "/assets", fixedPage(300)},
	{http.MethodGet, "/assets/location-and-speed/stream", limitPage(512)},
	{http.MethodGet, "/beta/fleet/drivers/efficiency", fixedPage(512)},
	{http.MethodGet, "/dvirs/stream", limitPage(200)},
	{http.MethodGet, "/fleet/driver-vehicle-assignments", fixedPage(512)},
	{http.MethodGet, "/fleet/drivers", limitPage(512)},
	{http.MethodGet, "/fleet/drivers/tachograph-activity/history", fixedPage(512)},
	{http.MethodGet, "/fleet/drivers/tachograph-files/history", fixedPage(512)},
	{http.MethodGet, "/fleet/drivers/vehicle-assignments", fixedPage(512)},
	{http.MethodGet, "/fleet/drivers/workflows", limitPage(512)},
	{http.MethodGet, "/fleet/dvirs/history", limitPage(512)},
	{http.MethodGet, "/fleet/equipment", limitPage(512)},
	{http.MethodGet, "/fleet/equipment/locations", fixedPage(512)},
	{http.MethodGet, "/fleet/equipment/locations/feed", fixedPage(512)},
	{http.MethodGet, "/fleet/equipment/locations/history", fixedPage(512)},
	{http.MethodGet, "/fleet/equipment/stats", fixedPage(512)},
	{http.MethodGet, "/fleet/equipment/stats/feed", fixedPage(512)},
	{http.MethodGet, "/fleet/equipment/stats/history", fixedPage(512)},
	{http.MethodGet, "/fleet/hos/clocks", limitPage(512)},
	{http.MethodGet, "/fleet/hos/daily-logs", fixedPage(512)},
	{http.MethodGet, "/fleet/hos/logs", fixedPage(512)},
	{http.MethodGet, "/fleet/hos/violations", fixedPage(512)},
	{http.MethodGet, "/fleet/routes", limitPage(512)},
	{http.MethodGet, "/fleet/trailers", limitPage(512)},
	{http.MethodGet, "/fleet/trailers/stats", fixedPage(512)},
	{http.MethodGet, "/fleet/trailers/stats/feed", fixedPage(512)},
	{http.MethodGet, "/fleet/trailers/stats/history", fixedPage(512)},
	{http.MethodGet, "/fleet/vehicles", limitPage(512)},
	{http.MethodGet, "/fleet/vehicles/driver-assignments", fixedPage(512)},
	{http.MethodGet, "/fleet/vehicles/immobilizer/stream", fixedPage(512)},
	{http.MethodGet, "/fleet/vehicles/locations", fixedPage(512)},
	{http.MethodGet, "/fleet/vehicles/locations/feed", fixedPage(512)},
	{http.MethodGet, "/fleet/vehicles/locations/history", fixedPage(512)},
	{http.MethodGet, "/fleet/vehicles/stats", fixedPage(512)},
	{http.MethodGet, "/fleet/vehicles/stats/feed", fixedPage(512)},
	{http.MethodGet, "/fleet/vehicles/stats/history", fixedPage(512)},
	{http.MethodGet, "/fleet/vehicles/tachograph-files/history", fixedPage(512)},
	{http.MethodGet, "/fleet/document-types", fixedPage(512)},
	{http.MethodGet, "/fleet/documents", fixedPage(512)},
	{http.MethodGet, "/form-submissions/stream", fixedPage(512)},
	{http.MethodGet, "/form-templates", fixedPage(512)},
	{http.MethodGet, "/live-shares", limitPage(100)},
	{http.MethodGet, "/tags", limitPage(512)},
	{http.MethodGet, "/users", limitPage(512)},
	{http.MethodGet, "/webhooks", limitPage(512)},
})

func pagePolicyFor(request *http.Request) pagePolicy {
	if policy, ok := endpointPagePolicies.lookup(requestRouteKey(request)); ok {
		return policy
	}
	return limitPage(defaultPageSize)
}

func (p pagePolicy) parseLimit(values url.Values) (int, error) {
	if !p.LimitParam {
		return p.Size, nil
	}
	rawLimit := strings.TrimSpace(values.Get("limit"))
	if rawLimit == "" {
		return p.Size, nil
	}
	limit, err := strconv.Atoi(rawLimit)
	if err != nil || limit < 1 || limit > p.Max {
		return 0, fmt.Errorf(
			"%w: it must be an integer between 1 and %d",
			ErrLimitInvalid,
			p.Max,
		)
	}
	return limit, nil
}

type pageCursor struct {
	Version int    `json:"v"`
	Key     string `json:"k,omitempty"`
	Ordinal int    `json:"n,omitempty"`
	Offset  int    `json:"o"`
}

func encodePageCursor(cursor pageCursor) (string, error) {
	cursor.Version = pageCursorVersion
	raw, err := sonic.Marshal(cursor)
	if err != nil {
		return "", fmt.Errorf("encode page cursor: %w", err)
	}
	return base64.URLEncoding.EncodeToString(raw), nil
}

func decodePageCursor(value string) (pageCursor, error) {
	clean := strings.TrimSpace(value)
	raw, err := base64.URLEncoding.DecodeString(clean)
	if err != nil {
		raw, err = base64.StdEncoding.DecodeString(clean)
		if err != nil {
			return pageCursor{}, ErrCursorInvalid
		}
	}
	cursor := pageCursor{}
	if err = sonic.Unmarshal(raw, &cursor); err != nil {
		return pageCursor{}, ErrCursorInvalid
	}
	if cursor.Version != pageCursorVersion || cursor.Offset < 0 || cursor.Ordinal < 0 {
		return pageCursor{}, ErrCursorInvalid
	}
	return cursor, nil
}

func cursorAfter(records []Record, index int) pageCursor {
	key := recordIdentityKey(records[index])
	ordinal := 0
	if key != "" {
		for idx := 0; idx < index; idx++ {
			if recordIdentityKey(records[idx]) == key {
				ordinal++
			}
		}
	}
	return pageCursor{Key: key, Ordinal: ordinal, Offset: index + 1}
}

func resumeIndex(records []Record, cursor pageCursor) int {
	if cursor.Key != "" {
		seen := 0
		for idx, record := range records {
			if recordIdentityKey(record) != cursor.Key {
				continue
			}
			if seen == cursor.Ordinal {
				return idx + 1
			}
			seen++
		}
	}
	return min(cursor.Offset, len(records))
}

func paginate(
	records []Record,
	request *http.Request,
) (page []Record, pagination map[string]any, err error) {
	return paginateRecords(records, request.URL.Query(), pagePolicyFor(request))
}

func paginateRecords(
	records []Record,
	values url.Values,
	policy pagePolicy,
) (page []Record, pagination map[string]any, err error) {
	limit, err := policy.parseLimit(values)
	if err != nil {
		return nil, nil, err
	}

	workingSet := records
	start := 0
	if after := strings.TrimSpace(values.Get("after")); after != "" {
		cursor, decodeErr := decodePageCursor(after)
		if decodeErr != nil {
			return nil, nil, decodeErr
		}
		start = resumeIndex(workingSet, cursor)
	}

	end := min(start+limit, len(workingSet))
	page = cloneRecords(workingSet[start:end])
	pagination = map[string]any{
		"endCursor":   "",
		"hasNextPage": false,
	}
	if end >= len(workingSet) {
		return page, pagination, nil
	}

	endCursor, err := encodePageCursor(cursorAfter(workingSet, end-1))
	if err != nil {
		return nil, nil, err
	}
	pagination["endCursor"] = endCursor
	pagination["hasNextPage"] = true
	return page, pagination, nil
}

var (
	identityReferenceKeys = []string{"driver", "vehicle", "trailer", "asset"}
	identityTimeKeys      = []string{
		"happenedAtTime",
		"logStartTime",
		"startTime",
		"time",
		"violationStartTime",
	}
)

func recordIdentityKey(record Record) string {
	if id := recordID(record); id != "" {
		return "id:" + id
	}

	var builder strings.Builder
	for _, reference := range identityReferenceKeys {
		id := nestedString(record, reference, "id")
		if id == "" {
			continue
		}
		if builder.Len() > 0 {
			builder.WriteByte('|')
		}
		builder.WriteString(reference)
		builder.WriteByte(':')
		builder.WriteString(id)
	}
	for _, field := range identityTimeKeys {
		value := stringValue(record, field)
		if value == "" {
			continue
		}
		if builder.Len() > 0 {
			builder.WriteByte('|')
		}
		builder.WriteString(field)
		builder.WriteByte('@')
		builder.WriteString(value)
		break
	}
	return builder.String()
}
