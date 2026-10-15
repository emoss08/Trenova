package carriersettlement

import (
	"cmp"
	"slices"

	"github.com/emoss08/trenova/internal/modules/settlement/domain/settlementload"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/typeutils"
)

type CostGroup struct {
	EventType     CostEventType
	Description   string
	AmountMinor   int64
	LineCount     int
	ShipmentCount int
	Manual        bool
}

type costGroupKey struct {
	eventType   CostEventType
	description string
	manual      bool
}

type costGroupTally struct {
	group     CostGroup
	first     int
	shipments map[pulid.ID]struct{}
}

var costEventTypeRank = map[CostEventType]int{
	CostEventTypeLinehaulCost:  0,
	CostEventTypeFuelSurcharge: 1,
	CostEventTypeAccessorial:   2,
	CostEventTypeAdjustment:    3,
}

func (l *CarrierSettlementLine) IsManual() bool {
	return l.EventType == CostEventTypeAdjustment && typeutils.DerefID(l.CostEventID).IsNil()
}

func CostGroups(lines []*CarrierSettlementLine) []CostGroup {
	tallies := make([]*costGroupTally, 0, len(lines))
	at := make(map[costGroupKey]*costGroupTally, len(lines))
	for idx, line := range lines {
		if line == nil {
			continue
		}
		key := costGroupKey{
			eventType:   line.EventType,
			description: line.Description,
			manual:      line.IsManual(),
		}
		tally, seen := at[key]
		if !seen {
			tally = &costGroupTally{
				group: CostGroup{
					EventType:   line.EventType,
					Description: line.Description,
					Manual:      key.manual,
				},
				first:     idx,
				shipments: make(map[pulid.ID]struct{}),
			}
			at[key] = tally
			tallies = append(tallies, tally)
		}
		tally.group.AmountMinor += line.AmountMinor
		tally.group.LineCount++
		if id := typeutils.DerefID(line.ShipmentID); !id.IsNil() {
			tally.shipments[id] = struct{}{}
		}
	}

	slices.SortStableFunc(tallies, func(a, b *costGroupTally) int {
		return cmp.Or(
			cmp.Compare(costEventTypeRank[a.group.EventType], costEventTypeRank[b.group.EventType]),
			cmp.Compare(a.first, b.first),
		)
	})

	groups := make([]CostGroup, 0, len(tallies))
	for _, tally := range tallies {
		group := tally.group
		group.ShipmentCount = len(tally.shipments)
		groups = append(groups, group)
	}
	return groups
}

func LoadLines(lines []*CarrierSettlementLine) []settlementload.Line {
	out := make([]settlementload.Line, 0, len(lines))
	for _, line := range lines {
		if line == nil {
			continue
		}
		out = append(out, settlementload.Line{
			ShipmentID:  line.ShipmentID,
			MoveID:      line.MoveID,
			ProNumber:   line.ProNumber,
			AmountMinor: line.AmountMinor,
		})
	}
	return out
}

type MoveInvoice struct {
	MoveID     pulid.ID
	ShipmentID *pulid.ID
	ProNumber  string
	CostMinor  int64
	Match      *InvoiceMatch
}

func MoveIDs(lines []*CarrierSettlementLine) []pulid.ID {
	ids := make([]pulid.ID, 0, len(lines))
	for _, line := range lines {
		if line == nil {
			continue
		}
		if id := typeutils.DerefID(line.MoveID); !id.IsNil() && !slices.Contains(ids, id) {
			ids = append(ids, id)
		}
	}
	return ids
}

func MoveInvoices(
	lines []*CarrierSettlementLine,
	matchesByMove map[pulid.ID][]*InvoiceMatch,
) []MoveInvoice {
	invoices := make([]MoveInvoice, 0, len(lines))
	at := make(map[pulid.ID]int, len(lines))
	for _, line := range lines {
		if line == nil {
			continue
		}
		moveID := typeutils.DerefID(line.MoveID)
		if moveID.IsNil() {
			continue
		}
		idx, seen := at[moveID]
		if !seen {
			idx = len(invoices)
			at[moveID] = idx
			invoices = append(invoices, MoveInvoice{
				MoveID: moveID,
				Match:  standingMatch(matchesByMove[moveID]),
			})
		}
		invoice := &invoices[idx]
		invoice.CostMinor += line.AmountMinor
		if invoice.ShipmentID == nil {
			invoice.ShipmentID = line.ShipmentID
		}
		if invoice.ProNumber == "" {
			invoice.ProNumber = line.ProNumber
		}
	}
	return invoices
}

func standingMatch(matches []*InvoiceMatch) *InvoiceMatch {
	var standing *InvoiceMatch
	for _, match := range matches {
		if match == nil || match.Status == InvoiceMatchStatusRejected {
			continue
		}
		if standing == nil || outranks(match, standing) {
			standing = match
		}
	}
	return standing
}

func outranks(candidate, current *InvoiceMatch) bool {
	candidateOriginal := typeutils.DerefID(candidate.DuplicateOfMatchID).IsNil()
	currentOriginal := typeutils.DerefID(current.DuplicateOfMatchID).IsNil()
	if candidateOriginal != currentOriginal {
		return candidateOriginal
	}
	if candidate.CreatedAt != current.CreatedAt {
		return candidate.CreatedAt > current.CreatedAt
	}
	return candidate.ID > current.ID
}
