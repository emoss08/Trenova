package conversation

import (
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	MaxWorkingSet      = 12
	maxWorkingLabelLen = 120
)

type WorkingSource string

const (
	WorkingSourceSubject WorkingSource = "subject"
	WorkingSourceMention WorkingSource = "mention"
	WorkingSourcePage    WorkingSource = "page"
	WorkingSourceRead    WorkingSource = "read"
	WorkingSourceWrote   WorkingSource = "wrote"
)

type WorkingRecord struct {
	Kind      permission.RecordKind `json:"kind"`
	ID        string                `json:"id"`
	Label     string                `json:"label,omitempty"`
	Source    WorkingSource         `json:"source"`
	TouchedAt int64                 `json:"touchedAt"`
}

func (r WorkingRecord) Ref() agent.EntityRef {
	return agent.EntityRef{Type: string(r.Kind), ID: r.ID, Label: r.Label}
}

func NewWorkingRecord(id, label string, source WorkingSource, at int64) (WorkingRecord, bool) {
	id = strings.TrimSpace(id)
	prefix, _, found := strings.Cut(id, "_")
	if !found || prefix == "" {
		return WorkingRecord{}, false
	}
	kind, known := permission.RecordKindOfIDPrefix(prefix + "_")
	if !known {
		return WorkingRecord{}, false
	}

	return WorkingRecord{
		Kind:      kind,
		ID:        id,
		Label:     stringutils.OneLine(label, maxWorkingLabelLen),
		Source:    source,
		TouchedAt: at,
	}, true
}

func sourceRank(source WorkingSource) int {
	switch source {
	case WorkingSourceSubject:
		return 4
	case WorkingSourceWrote:
		return 3
	case WorkingSourceMention, WorkingSourcePage:
		return 2
	default:
		return 1
	}
}

func Touch(current []WorkingRecord, touched ...WorkingRecord) []WorkingRecord {
	merged := make([]WorkingRecord, 0, min(len(current)+len(touched), MaxWorkingSet*2))
	merged = append(merged, current...)
	for _, record := range touched {
		if record.ID == "" {
			continue
		}
		idx := slices.IndexFunc(merged, func(known WorkingRecord) bool { return known.ID == record.ID })
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

	slices.SortStableFunc(merged, func(a, b WorkingRecord) int {
		if a.Source == WorkingSourceSubject && b.Source != WorkingSourceSubject {
			return -1
		}
		if b.Source == WorkingSourceSubject && a.Source != WorkingSourceSubject {
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

func Forget(current []WorkingRecord, id string) []WorkingRecord {
	return slices.DeleteFunc(slices.Clone(current), func(record WorkingRecord) bool {
		return record.ID == id
	})
}
