package messages

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

const maxTextLength = 2500

func (r CreateRequest) body() (createRequestBody, error) {
	if strings.TrimSpace(r.Text) == "" {
		return createRequestBody{}, ErrTextRequired
	}
	if utf8.RuneCountInString(r.Text) > maxTextLength {
		return createRequestBody{}, ErrTextTooLong
	}
	if len(r.DriverIDs) == 0 {
		return createRequestBody{}, ErrDriverIDsRequired
	}

	ids := make([]int64, 0, len(r.DriverIDs))
	seen := make(map[int64]struct{}, len(r.DriverIDs))
	for _, raw := range r.DriverIDs {
		id, err := parseDriverID(raw)
		if err != nil {
			return createRequestBody{}, err
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return createRequestBody{DriverIDs: ids, Text: r.Text}, nil
}

func parseDriverID(raw string) (int64, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || trimmed[0] == '+' || trimmed[0] == '-' {
		return 0, fmt.Errorf("%w: %q", ErrDriverIDInvalid, raw)
	}
	id, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("%w: %q", ErrDriverIDInvalid, raw)
	}
	return id, nil
}
