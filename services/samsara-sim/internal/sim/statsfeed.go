package sim

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const maxFeedSamplesPerVehicle = maxAssetSamplesPerAsset

type feedRequestWindow struct {
	feedWindow
	EndCursor string
	HasNext   bool
}

func parseFeedWindow(
	request *http.Request,
	now time.Time,
	assetCount int,
	totalCap int,
) (feedRequestWindow, string, error) {
	gridNow := now.Truncate(defaultAssetSampleStep)
	after := queryValue(request, "after")
	if after == "" {
		return feedRequestWindow{
			feedWindow: feedWindow{
				Times:      []time.Time{gridNow},
				EventFrom:  gridNow.Add(-statEventLookback),
				EventTo:    gridNow,
				LatestOnly: true,
			},
			EndCursor: encodeStatsFeedCursor(gridNow, 0),
		}, after, nil
	}
	cursorTime, _, err := decodeStatsFeedCursor(after)
	if err != nil {
		return feedRequestWindow{}, after, err
	}
	times, hasNext := statsFeedSampleTimes(cursorTime, gridNow, assetCount, totalCap)
	if len(times) == 0 {
		return feedRequestWindow{EndCursor: after}, after, nil
	}
	last := times[len(times)-1]
	return feedRequestWindow{
		feedWindow: feedWindow{Times: times, EventFrom: cursorTime, EventTo: last},
		EndCursor:  encodeStatsFeedCursor(last, 0),
		HasNext:    hasNext,
	}, after, nil
}

func parseHistoryWindow(request *http.Request, now time.Time) (*feedWindow, error) {
	startTime, endTime, err := parseTimeRange(request)
	if err != nil {
		return nil, err
	}
	if startTime == nil || endTime == nil {
		return nil, ErrTimeRangeRequired
	}
	end := minTime(*endTime, now)
	window := &feedWindow{EventFrom: startTime.Add(-time.Nanosecond), EventTo: end}
	if end.Before(*startTime) {
		window.Times = []time.Time{}
		return window, nil
	}
	window.Times = sampleTimes(*startTime, end, defaultAssetSampleStep, maxFeedSamplesPerVehicle)
	return window, nil
}

func (s *Server) respondFeed(
	writer http.ResponseWriter,
	request *http.Request,
	records []Record,
	endCursor string,
	hasNextPage bool,
	signature string,
) {
	payload := map[string]any{
		keyData: recordsAsAny(records),
		keyPagination: map[string]any{
			"endCursor":   endCursor,
			"hasNextPage": hasNextPage,
		},
	}
	s.respondJSON(writer, request, requestSignature(request)+signature, payload)
}

func encodeStatsFeedCursor(at time.Time, offset int) string {
	raw := fmt.Sprintf("t=%d;i=%d", at.UTC().UnixMilli(), offset)
	return base64.StdEncoding.EncodeToString([]byte(raw))
}

func decodeStatsFeedCursor(cursor string) (time.Time, int, error) {
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(cursor))
	if err != nil {
		return time.Time{}, 0, ErrCursorInvalid
	}

	unixMilli := int64(-1)
	offset := 0
	for _, part := range strings.Split(string(decoded), ";") {
		key, value, found := strings.Cut(part, "=")
		if !found {
			return time.Time{}, 0, ErrCursorInvalid
		}
		switch key {
		case "t":
			parsed, parseErr := strconv.ParseInt(value, 10, 64)
			if parseErr != nil || parsed < 0 {
				return time.Time{}, 0, ErrCursorInvalid
			}
			unixMilli = parsed
		case "i":
			parsed, parseErr := strconv.Atoi(value)
			if parseErr != nil || parsed < 0 {
				return time.Time{}, 0, ErrCursorInvalid
			}
			offset = parsed
		default:
			return time.Time{}, 0, ErrCursorInvalid
		}
	}
	if unixMilli < 0 {
		return time.Time{}, 0, ErrCursorInvalid
	}
	return time.UnixMilli(unixMilli).UTC(), offset, nil
}

func statsFeedSampleTimes(
	cursorTime time.Time,
	gridNow time.Time,
	vehicleCount int,
	totalCap int,
) (times []time.Time, hasNext bool) {
	if !gridNow.After(cursorTime) {
		return []time.Time{}, false
	}

	available := int(gridNow.Sub(cursorTime) / defaultAssetSampleStep)
	if available <= 0 {
		return []time.Time{}, false
	}
	perVehicle := available
	if perVehicle > maxFeedSamplesPerVehicle {
		perVehicle = maxFeedSamplesPerVehicle
	}
	if totalCap > 0 && vehicleCount > 0 && perVehicle*vehicleCount > totalCap {
		perVehicle = totalCap / vehicleCount
		if perVehicle < 1 {
			perVehicle = 1
		}
	}

	times = make([]time.Time, 0, perVehicle)
	for idx := 1; idx <= perVehicle; idx++ {
		times = append(times, cursorTime.Add(time.Duration(idx)*defaultAssetSampleStep).UTC())
	}
	return times, perVehicle < available
}

func formattedLocationFromAddress(address map[string]any) string {
	if len(address) == 0 {
		return ""
	}
	record := Record(address)
	if formatted := stringValue(record, "formattedAddress"); formatted != "" {
		return formatted
	}

	street := strings.TrimSpace(
		stringValue(record, "streetNumber") + " " + stringValue(record, "street"),
	)
	parts := make([]string, 0, 3)
	for _, part := range []string{street, stringValue(record, "city"), stringValue(record, keyState)} {
		if part != "" {
			parts = append(parts, part)
		}
	}
	return strings.Join(parts, ", ")
}
