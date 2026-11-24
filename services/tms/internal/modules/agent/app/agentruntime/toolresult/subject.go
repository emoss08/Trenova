package toolresult

import (
	"maps"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/boardregistry"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	itemsKey        = "items"
	minSubjectRows  = 2
	subjectIDLookup = "id"
)

func Subject(content string) (pulid.ID, bool) {
	decoded, ok := decodeResult(content)
	if !ok {
		return pulid.Nil, false
	}
	record, isRecord := decoded.(map[string]any)
	if !isRecord {
		return pulid.Nil, false
	}
	rows, listed := record[itemsKey].([]any)
	if !listed {
		return subjectOfRecord(record)
	}

	return subjectOfRows(rows)
}

func subjectOfRows(rows []any) (pulid.ID, bool) {
	own := make([]string, 0, 1)
	held := make([]string, 0, 1)
	for _, row := range rows {
		fields, isRecord := row.(map[string]any)
		if !isRecord {
			return pulid.Nil, false
		}
		if id, _ := fields[subjectIDLookup].(string); isSubjectKind(id) {
			own = append(own, id)
			continue
		}
		subjects := subjectsOf(fields)
		if len(subjects) != 1 {
			return pulid.Nil, false
		}
		held = append(held, subjects[0])
	}
	switch {
	case len(own) == 1 && len(held) == 0:
		return pulid.ID(own[0]), true
	case len(own) == 0 && len(held) >= minSubjectRows:
		return soleSubject(slices.Compact(slices.Sorted(slices.Values(held))))
	}

	return pulid.Nil, false
}

func subjectOfRecord(record map[string]any) (pulid.ID, bool) {
	own := stringID(record[subjectIDLookup])
	switch {
	case isSubjectKind(own):
		return pulid.ID(own), true
	case own != "":
		return pulid.Nil, false
	}

	return soleSubject(subjectsOf(record))
}

func soleSubject(ids []string) (pulid.ID, bool) {
	if len(ids) != 1 {
		return pulid.Nil, false
	}

	return pulid.ID(ids[0]), true
}

func subjectsOf(fields map[string]any) []string {
	found := make([]string, 0, 1)
	add := func(id string) {
		if isSubjectKind(id) && !slices.Contains(found, id) {
			found = append(found, id)
		}
	}
	add(stringID(fields[subjectIDLookup]))
	for _, key := range slices.Sorted(maps.Keys(fields)) {
		switch typed := fields[key].(type) {
		case string:
			if key != subjectIDLookup {
				add(typed)
			}
		case map[string]any:
			add(stringID(typed[subjectIDLookup]))
		}
	}

	return found
}

func stringID(value any) string {
	text, _ := value.(string)

	return text
}

func isSubjectKind(id string) bool {
	separator := strings.IndexByte(id, '_')
	if separator <= 0 || !pulid.LooksLike(id) {
		return false
	}
	kind, known := recordkind.OfIDPrefix(id[:separator+1])
	if !known {
		return false
	}
	metrics := boardregistry.Catalog().Metrics()

	return slices.ContainsFunc(metrics, func(metric boardcatalog.Metric) bool {
		return metric.For == kind
	})
}
