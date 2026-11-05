package shipmentquerytools

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithOutsideRows_MarksShipmentsEnteredOverEDI(t *testing.T) {
	t.Parallel()

	edi := &shipment.Shipment{
		ID:          pulid.MustNew("shp_"),
		EntryMethod: shipmentvocab.EntryMethodEDI,
	}
	manual := &shipment.Shipment{
		ID:          pulid.MustNew("shp_"),
		EntryMethod: shipmentvocab.EntryMethodManual,
	}

	marked, ok := withOutsideRows(querytool.SearchOutcome{
		Items: []any{ToRow(edi), ToRow(manual)},
	}).(agent.TaintCarrier)
	require.True(t, ok)

	records := marked.TaintedRecords()
	require.Len(t, records, 1)
	assert.Equal(t, agent.TaintEntityShipment, records[0].EntityType)
	assert.Equal(t, edi.ID.String(), records[0].ID)
}

func TestWithOutsideRows_LeavesAManualListAlone(t *testing.T) {
	t.Parallel()

	outcome := querytool.SearchOutcome{
		Items: []Row{ToRow(&shipment.Shipment{
			ID:          pulid.MustNew("shp_"),
			EntryMethod: shipmentvocab.EntryMethodManual,
		})},
	}

	_, isPlain := withOutsideRows(outcome).(querytool.SearchOutcome)
	assert.True(t, isPlain)
}
