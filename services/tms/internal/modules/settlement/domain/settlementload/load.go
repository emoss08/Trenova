package settlementload

import (
	"slices"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/typeutils"
)

type Line struct {
	ShipmentID  *pulid.ID
	MoveID      *pulid.ID
	ProNumber   string
	AmountMinor int64
}

type Share struct {
	ShipmentID  pulid.ID
	MoveIDs     []pulid.ID
	ProNumber   string
	AmountMinor int64
	LineCount   int
}

func Shares(lines []Line) []Share {
	shares := make([]Share, 0, len(lines))
	at := make(map[pulid.ID]int, len(lines))
	for _, line := range lines {
		shipmentID := typeutils.DerefID(line.ShipmentID)
		if shipmentID.IsNil() {
			continue
		}
		idx, seen := at[shipmentID]
		if !seen {
			idx = len(shares)
			at[shipmentID] = idx
			shares = append(shares, Share{ShipmentID: shipmentID})
		}
		share := &shares[idx]
		share.AmountMinor += line.AmountMinor
		share.LineCount++
		if share.ProNumber == "" {
			share.ProNumber = line.ProNumber
		}
		if moveID := typeutils.DerefID(line.MoveID); !moveID.IsNil() &&
			!slices.Contains(share.MoveIDs, moveID) {
			share.MoveIDs = append(share.MoveIDs, moveID)
		}
	}
	return shares
}

func ShipmentIDs(lines []Line) []pulid.ID {
	shares := Shares(lines)
	ids := make([]pulid.ID, 0, len(shares))
	for idx := range shares {
		ids = append(ids, shares[idx].ShipmentID)
	}
	return ids
}
