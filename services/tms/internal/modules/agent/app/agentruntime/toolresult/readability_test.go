package toolresult

import (
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
A record reaches the model without what it carries for the database.

The shipment_get the Dispatch desk read repeated the tenant, a version and two
audit stamps on every commodity, stop, location and charge. The record's own
version stays, because a write that checks it reads it from there.
*/
func TestEncodeToolResult_TrimsWhatARecordCarriesForTheDatabase(t *testing.T) {
	t.Parallel()

	record := map[string]any{
		"id":             "shp_1",
		"organizationId": "org_1",
		"businessUnitId": "bu_1",
		"version":        3,
		"moves": []any{map[string]any{
			"id":             "sm_1",
			"organizationId": "org_1",
			"version":        0,
			"createdAt":      transcriptNow,
			"stops": []any{map[string]any{
				"id":             "stp_1",
				"businessUnitId": "bu_1",
				"updatedAt":      transcriptNow,
				"type":           "Pickup",
			}},
		}},
	}

	encoded, err := Encode(record, transcriptNow, "UTC")
	require.NoError(t, err)

	assert.NotContains(t, encoded, "organizationId")
	assert.NotContains(t, encoded, "businessUnitId")
	assert.NotContains(t, encoded, "createdAt")
	assert.NotContains(t, encoded, "updatedAt")
	assert.Equal(t, 1, strings.Count(encoded, `"version"`), "only the record's own version is kept")
	assert.Contains(t, encoded, `"type":"Pickup"`)
}

/*
An appointment window is read with its hour.

The window arrived as a raw epoch, and the model told the dispatcher the stop
had no window set. A window, an arrival and a departure keep the time of day;
a plain date field stays a date.
*/
func TestEncodeToolResult_RendersAnAppointmentWindowWithItsHour(t *testing.T) {
	t.Parallel()

	start := transcriptNow + 2*86400 + 14*3600
	encoded, err := Encode(map[string]any{
		"scheduledWindowStart": start,
		"actualArrival":        start,
		"deliveryDate":         start,
	}, transcriptNow, "UTC")
	require.NoError(t, err)

	var document map[string]string
	require.NoError(t, sonic.UnmarshalString(encoded, &document))
	assert.Equal(t, "2026-09-21 14:00 UTC (in 2 days)", document["scheduledWindowStart"])
	assert.Equal(t, "2026-09-21 14:00 UTC (in 2 days)", document["actualArrival"])
	assert.Equal(t, "2026-09-21 (in 2 days)", document["deliveryDate"])
}
