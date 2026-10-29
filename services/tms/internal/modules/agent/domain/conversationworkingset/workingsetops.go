package conversationworkingset

import (
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	MaxWorkingSet      = 12
	maxWorkingLabelLen = 120
)

func NewWorkingRecord(
	id, label string,
	source conversation.WorkingSource,
	at int64,
) (conversation.WorkingRecord, bool) {
	id = strings.TrimSpace(id)
	prefix, _, found := strings.Cut(id, "_")
	if !found || prefix == "" {
		return conversation.WorkingRecord{}, false
	}
	kind, known := recordkind.OfIDPrefix(prefix + "_")
	if !known {
		return conversation.WorkingRecord{}, false
	}

	return conversation.WorkingRecord{
		Kind:      kind,
		ID:        id,
		Label:     stringutils.OneLine(label, maxWorkingLabelLen),
		Source:    source,
		TouchedAt: at,
	}, true
}

func sourceRank(source conversation.WorkingSource) int {
	switch source {
	case conversation.WorkingSourceSubject:
		return 4
	case conversation.WorkingSourceWrote:
		return 3
	case conversation.WorkingSourceMention, conversation.WorkingSourcePage:
		return 2
	default:
		return 1
	}
}

func Touch(
	current []conversation.WorkingRecord,
	touched ...conversation.WorkingRecord,
) []conversation.WorkingRecord {
	merged := make([]conversation.WorkingRecord, 0, min(len(current)+len(touched), MaxWorkingSet*2))
	merged = append(merged, current...)
	for _, record := range touched {
		if record.ID == "" {
			continue
		}
		idx := slices.IndexFunc(
			merged,
			func(known conversation.WorkingRecord) bool { return known.ID == record.ID },
		)
		if idx < 0 {
			merged = append(merged, record)
			continue
		}
		known := &merged[idx]
		known.TouchedAt = max(known.TouchedAt, record.TouchedAt)
		if record.Label != "" {
			known.Label = record.Label
		}
		if sourceRank(record.Source) > sourceRank(known.Source) {
			known.Source = record.Source
		}
	}

	slices.SortStableFunc(merged, func(a, b conversation.WorkingRecord) int {
		if a.Source == conversation.WorkingSourceSubject &&
			b.Source != conversation.WorkingSourceSubject {
			return -1
		}
		if b.Source == conversation.WorkingSourceSubject &&
			a.Source != conversation.WorkingSourceSubject {
			return 1
		}
		switch {
		case a.TouchedAt > b.TouchedAt:
			return -1
		case a.TouchedAt < b.TouchedAt:
			return 1
		default:
			return 0
		}
	})
	if len(merged) > MaxWorkingSet {
		merged = merged[:MaxWorkingSet]
	}

	return merged
}

func Forget(current []conversation.WorkingRecord, id string) []conversation.WorkingRecord {
	return slices.DeleteFunc(slices.Clone(current), func(record conversation.WorkingRecord) bool {
		return record.ID == id
	})
}
