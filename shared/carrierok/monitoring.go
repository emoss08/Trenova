package carrierok

import (
	"cmp"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/emoss08/trenova/shared/jsonflex"
)

const (
	suffixCurrent = "_current"
	suffixPrior   = "_prior"
	suffixChanged = "_changed"
)

type MonitoringWriteResult struct {
	StatusCode int
	Status     string
	Added      int64
	Removed    int64
	Error      string
	Failed     []MonitoringFailure
	Partial    bool
	Raw        []byte
}

type MonitoringFailure struct {
	ProfileID string
	Error     string
}

type FieldChange struct {
	Field   string
	Current []byte
	Prior   []byte
	Changed bool
}

type MonitoredProfile struct {
	ProfileID string
	DOTNumber string
	Docket    string
	Changes   []FieldChange
	Meta      map[string]string
}

type MonitoringListResult struct {
	TotalCount int64
	Profiles   []MonitoredProfile
	Raw        []byte
}

func ProfileID(dot, docketPrefix, docketNumber string) string {
	dot = strings.TrimSpace(dot)
	number := strings.ToUpper(strings.TrimSpace(docketNumber))
	if number == "" {
		return dot
	}
	prefix := strings.ToUpper(strings.TrimSpace(docketPrefix))
	if prefix != "" && !strings.HasPrefix(number, prefix) {
		number = prefix + number
	}
	return dot + "-" + number
}

func SplitProfileID(profileID string) (dot, docket string) {
	trimmed := strings.TrimSpace(profileID)
	dot, docket, _ = strings.Cut(trimmed, "-")
	return dot, docket
}

func decodeMonitoringWrite(status int, body []byte) (*MonitoringWriteResult, error) {
	result := &MonitoringWriteResult{
		StatusCode: status,
		Raw:        body,
	}
	if len(body) == 0 || jsonflex.IsAbsent(body) {
		result.Partial = status == http.StatusMultiStatus
		return result, nil
	}

	obj, err := jsonflex.DecodeObject(body)
	if err != nil {
		return nil, fmt.Errorf("%w: monitoring response: %w", ErrUnexpectedPayload, err)
	}

	result.Status = obj.Text("status")
	result.Added = obj.Int("added").Value()
	result.Removed = obj.Int("removed").Value()
	result.Error = obj.Text("error", "message")

	if items, ok := obj.Array("failed", "failures", "errors"); ok {
		result.Failed = make([]MonitoringFailure, 0, len(items))
		for _, item := range items {
			if failure, valid := decodeMonitoringFailure(item); valid {
				result.Failed = append(result.Failed, failure)
			}
		}
	}
	result.Partial = status == http.StatusMultiStatus || len(result.Failed) > 0
	return result, nil
}

func decodeMonitoringFailure(raw []byte) (MonitoringFailure, bool) {
	switch jsonflex.KindOf(raw) {
	case jsonflex.KindObject:
		item, err := jsonflex.DecodeObject(raw)
		if err != nil {
			return MonitoringFailure{}, false
		}
		failure := MonitoringFailure{
			ProfileID: item.Text("profile_id", "profileId", "id"),
			Error:     item.Text("error", "message", "reason"),
		}
		return failure, failure.ProfileID != "" || failure.Error != ""
	case jsonflex.KindString:
		value, ok := jsonflex.ParseString(raw)
		if !ok {
			return MonitoringFailure{}, false
		}
		return MonitoringFailure{ProfileID: value.Value()}, true
	default:
		return MonitoringFailure{}, false
	}
}

func decodeMonitoringList(body []byte) (*MonitoringListResult, error) {
	result := &MonitoringListResult{Raw: body}
	if len(body) == 0 || jsonflex.IsAbsent(body) {
		return result, nil
	}

	obj, err := jsonflex.DecodeObject(body)
	if err != nil {
		return nil, fmt.Errorf("%w: monitoring list: %w", ErrUnexpectedPayload, err)
	}
	result.TotalCount = obj.Int("total_count", "totalCount").Value()

	items, ok := obj.Object("items")
	if !ok {
		return result, nil
	}

	result.Profiles = make([]MonitoredProfile, 0, len(items))
	for profileID, raw := range items {
		profile, decodeErr := decodeMonitoredProfile(profileID, raw)
		if decodeErr != nil {
			return nil, decodeErr
		}
		result.Profiles = append(result.Profiles, profile)
	}
	slices.SortFunc(result.Profiles, func(a, b MonitoredProfile) int {
		return cmp.Compare(a.ProfileID, b.ProfileID)
	})
	return result, nil
}

func decodeMonitoredProfile(profileID string, raw []byte) (MonitoredProfile, error) {
	dot, docket := SplitProfileID(profileID)
	profile := MonitoredProfile{
		ProfileID: profileID,
		DOTNumber: dot,
		Docket:    docket,
		Meta:      make(map[string]string),
	}

	var entries [][]byte
	switch jsonflex.KindOf(raw) {
	case jsonflex.KindArray:
		items, err := jsonflex.DecodeArray(raw)
		if err != nil {
			return MonitoredProfile{}, fmt.Errorf(
				"%w: monitoring item %s: %w",
				ErrUnexpectedPayload,
				profileID,
				err,
			)
		}
		entries = make([][]byte, 0, len(items))
		for _, item := range items {
			entries = append(entries, item)
		}
	case jsonflex.KindObject:
		entries = [][]byte{raw}
	case jsonflex.KindNull:
		return profile, nil
	default:
		return MonitoredProfile{}, fmt.Errorf(
			"%w: monitoring item %s is not an object or array",
			ErrUnexpectedPayload,
			profileID,
		)
	}

	for _, entry := range entries {
		if jsonflex.KindOf(entry) != jsonflex.KindObject {
			continue
		}
		obj, err := jsonflex.DecodeObject(entry)
		if err != nil {
			return MonitoredProfile{}, fmt.Errorf(
				"%w: monitoring item %s: %w",
				ErrUnexpectedPayload,
				profileID,
				err,
			)
		}
		profile.Changes = append(profile.Changes, groupFieldChanges(obj, profile.Meta)...)
	}
	return profile, nil
}

func groupFieldChanges(obj jsonflex.Object, meta map[string]string) []FieldChange {
	indexes := make(map[string]int, len(obj)/3)
	changes := make([]FieldChange, 0, len(obj)/3)

	changeFor := func(field string) *FieldChange {
		if idx, ok := indexes[field]; ok {
			return &changes[idx]
		}
		indexes[field] = len(changes)
		changes = append(changes, FieldChange{Field: field})
		return &changes[len(changes)-1]
	}

	for _, key := range obj.Keys() {
		raw := obj[key]
		switch {
		case strings.HasSuffix(key, suffixCurrent):
			changeFor(strings.TrimSuffix(key, suffixCurrent)).Current = normalizedRaw(raw)
		case strings.HasSuffix(key, suffixPrior):
			changeFor(strings.TrimSuffix(key, suffixPrior)).Prior = normalizedRaw(raw)
		case strings.HasSuffix(key, suffixChanged):
			change := changeFor(strings.TrimSuffix(key, suffixChanged))
			if value, ok := jsonflex.ParseBool(raw); ok {
				change.Changed = value.Value()
			}
		default:
			meta[key] = metaValue(raw)
		}
	}

	slices.SortFunc(changes, func(a, b FieldChange) int {
		return cmp.Compare(a.Field, b.Field)
	})
	return changes
}

func normalizedRaw(raw []byte) []byte {
	if jsonflex.KindOf(raw) == jsonflex.KindNull {
		return nil
	}
	return cloneRaw(raw)
}

func metaValue(raw []byte) string {
	if value, ok := jsonflex.ParseString(raw); ok {
		return value.Value()
	}
	if jsonflex.IsAbsent(raw) {
		return ""
	}
	return strings.TrimSpace(string(raw))
}
