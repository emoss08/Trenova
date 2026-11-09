package agentplan

import (
	"testing"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestMostTargeted_PicksTheRecordMostStepsChange(t *testing.T) {
	shipment := pulid.ID("shp_01JABCDEFGHJKMNPQRSTVWXYZ0")
	customer := pulid.ID("cus_01JABCDEFGHJKMNPQRSTVWXYZ0")

	assert.Equal(t, shipment, MostTargeted([]pulid.ID{customer, shipment, pulid.Nil, shipment}))
}

func TestMostTargeted_KeepsTheFirstOnATie(t *testing.T) {
	first := pulid.ID("shp_01JABCDEFGHJKMNPQRSTVWXYZ0")
	second := pulid.ID("shp_01JABCDEFGHJKMNPQRSTVWXYZ1")

	assert.Equal(t, first, MostTargeted([]pulid.ID{first, second}))
}

func TestMostTargeted_IsNilWhenNoStepNamesARecord(t *testing.T) {
	assert.True(t, MostTargeted([]pulid.ID{pulid.Nil, ""}).IsNil())
}
