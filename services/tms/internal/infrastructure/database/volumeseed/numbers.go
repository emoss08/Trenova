package volumeseed

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/seqgen"
	"github.com/emoss08/trenova/shared/pulid"
)

type numberer struct {
	generator seqgen.Generator
	formats   seqgen.FormatProvider
	orgID     pulid.ID
	buID      pulid.ID
	buCode    string
}

type numberSlot struct {
	At           int64
	LocationCode string
}

type numberGroup struct {
	at      time.Time
	code    string
	indexes []int
}

// allocate issues one number per slot from the tenant's real sequence, in the
// order the slots happened. Slots are grouped by every date part and the
// location code the tenant's format prints, so each number reads exactly as
// the application would have issued it at that moment.
func (n *numberer) allocate(
	ctx context.Context,
	typ tenant.SequenceType,
	slots []numberSlot,
) ([]string, error) {
	if len(slots) == 0 {
		return nil, nil
	}

	format, err := n.formats.GetFormat(ctx, typ, n.orgID, n.buID)
	if err != nil {
		return nil, fmt.Errorf("get %s format: %w", typ, err)
	}
	byDay, byWeek, byLocation := formatParts(format)

	groups := make(map[string]*numberGroup)
	for i, slot := range slots {
		at := time.Unix(slot.At, 0).UTC()
		key := at.Format("2006-01")
		switch {
		case byDay:
			key = at.Format("2006-01-02")
		case byWeek:
			year, week := at.ISOWeek()
			key = fmt.Sprintf("%s-w%d-%d", key, year, week)
		}
		code := ""
		if byLocation {
			code = slot.LocationCode
			key += "|" + code
		}
		group, ok := groups[key]
		if !ok {
			group = &numberGroup{at: at, code: code}
			groups[key] = group
		}
		group.indexes = append(group.indexes, i)
	}

	keys := make([]string, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	out := make([]string, len(slots))
	for _, key := range keys {
		group := groups[key]
		sort.SliceStable(group.indexes, func(a, b int) bool {
			return slots[group.indexes[a]].At < slots[group.indexes[b]].At
		})
		numbers, genErr := n.generator.GenerateBatch(ctx, &seqgen.GenerateRequest{
			Type:             typ,
			OrgID:            n.orgID,
			BuID:             n.buID,
			Count:            len(group.indexes),
			Time:             group.at,
			LocationCode:     group.code,
			BusinessUnitCode: n.buCode,
		})
		if genErr != nil {
			return nil, fmt.Errorf("generate %s numbers: %w", typ, genErr)
		}
		for i, idx := range group.indexes {
			out[idx] = numbers[i]
		}
	}

	return out, nil
}

func formatParts(format *tenant.SequenceFormat) (byDay, byWeek, byLocation bool) {
	if format.AllowCustomFormat && format.CustomFormat != "" {
		return strings.Contains(format.CustomFormat, "{D}"),
			strings.Contains(format.CustomFormat, "{W}"),
			strings.Contains(format.CustomFormat, "{L}")
	}

	return format.IncludeDay, format.IncludeWeekNumber, format.IncludeLocationCode
}
