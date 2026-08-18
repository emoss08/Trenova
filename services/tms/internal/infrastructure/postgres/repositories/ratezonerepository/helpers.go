package ratezonerepository

import (
	"github.com/emoss08/trenova/internal/core/domain/ratezone"
	"github.com/emoss08/trenova/shared/pulid"
)

// stampMembers re-seats the tenancy and parent on every member.
//
// None of those three are the caller's to supply: a payload that named a
// different zone would move a member between zones as a side effect of editing
// the one it arrived under.
func stampMembers(entity *ratezone.RateZone, resetIDs bool) {
	for _, member := range entity.Members {
		if member == nil {
			continue
		}

		if resetIDs {
			member.ID = pulid.Nil
		}

		member.RateZoneID = entity.ID
		member.OrganizationID = entity.OrganizationID
		member.BusinessUnitID = entity.BusinessUnitID
		member.ApplyMatchKey()
	}
}
