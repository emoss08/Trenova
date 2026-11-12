package toolpreview

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
)

func TestDescribeNamesRecordsAndNeverShowsAnID(t *testing.T) {
	t.Parallel()

	workerPrefix, ok := recordkind.ResourcePrefix(permission.ResourceWorker)
	require.True(t, ok)
	shipmentPrefix, ok := recordkind.ResourcePrefix(permission.ResourceShipment)
	require.True(t, ok)
	worker := pulid.MustNew(workerPrefix)
	first, second := pulid.MustNew(shipmentPrefix), pulid.MustNew(shipmentPrefix)
	unknown := pulid.MustNew("zzzq_")

	preview := Describe("request_worker_credential_renewal", map[string]any{
		"workerId":    worker.String(),
		"shipmentIds": []any{first.String(), second.String()},
		"traceId":     unknown.String(),
		"dueWithin":   30,
		"credentials": []any{"Medical card", "Hazmat endorsement"},
		"_internal":   "left out",
	})

	require.Len(t, preview.Changes, 1)
	type shown struct {
		path, label string
		after       any
		ref         pulid.ID
	}
	got := make([]shown, 0, len(preview.Changes[0].Fields))
	for _, field := range preview.Changes[0].Fields {
		entry := shown{path: field.Path, label: field.Label, after: field.After}
		if field.AfterRef != nil {
			entry.ref = field.AfterRef.ID
		}
		got = append(got, entry)
	}

	require.Equal(t, []shown{
		{path: "credentials", label: "Credentials", after: []any{"Medical card", "Hazmat endorsement"}},
		{path: "dueWithin", label: "Due within", after: 30},
		{path: "shipmentIds[0]", label: "Shipment", ref: first},
		{path: "shipmentIds[1]", label: "Shipment", ref: second},
		{path: "workerId", label: "Worker", ref: worker},
	}, got)
	for _, field := range preview.Changes[0].Fields {
		if field.AfterRef != nil {
			require.Nil(t, field.After, "a record is named by its reference, never by its id")
		}
	}
	require.Equal(t, permission.ResourceWorker, preview.Changes[0].Fields[4].AfterRef.Resource)
}
