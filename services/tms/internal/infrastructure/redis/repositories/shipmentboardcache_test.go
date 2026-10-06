package repositories

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestShipmentBoardKeys(t *testing.T) {
	t.Parallel()

	org := pulid.MustNew("org_")
	bu := pulid.MustNew("bu_")

	assert.Equal(
		t,
		"shipment-board:epoch:"+org.String()+":"+bu.String(),
		ShipmentBoardEpochKey(org, bu),
	)

	key := &repositories.ShipmentBoardCacheKey{
		Section:    repositories.ShipmentBoardSectionBriefing,
		TenantInfo: pagination.TenantInfo{OrgID: org, BuID: bu},
		Timezone:   "America/Chicago",
		Epoch:      7,
	}
	assert.Equal(
		t,
		"shipment-board:briefing:"+org.String()+":"+bu.String()+":America/Chicago:7",
		ShipmentBoardSectionKey(key),
	)

	key.Variant = "usr_1"
	assert.Equal(
		t,
		"shipment-board:briefing:"+org.String()+":"+bu.String()+":America/Chicago:7:usr_1",
		ShipmentBoardSectionKey(key),
	)
}
