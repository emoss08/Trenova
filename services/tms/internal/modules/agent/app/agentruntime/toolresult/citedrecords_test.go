package toolresult

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpagecontext"
	"github.com/emoss08/trenova/pkg/productguide"
	"github.com/stretchr/testify/require"
)

func TestCitedRecordsNamesEachRoutableRecordInAToolResultOnce(t *testing.T) {
	content := `{
		"shipments": [
			{"id": "shp_01", "proNumber": "S-240193", "customer": {"id": "cus_01", "name": "Acme Foods"}},
			{"id": "shp_02", "proNumber": "S-240201", "moves": [{"id": "sm_01", "assignment": {"primaryWorker": {"id": "wrk_01", "firstName": "Leon", "lastName": "Park"}}}]}
		],
		"again": {"id": "shp_01", "proNumber": "S-240193"},
		"unlabelled": {"id": "cus_02"},
		"notARecord": {"id": "xyz_01", "name": "Nothing"}
	}`

	refs := CitedRecords(content)

	expected := []agentpagecontext.EntityRef{
		{Type: "shipment", ID: "shp_01", Label: "S-240193"},
		{Type: "customer", ID: "cus_01", Label: "Acme Foods"},
		{Type: "shipment", ID: "shp_02", Label: "S-240201"},
		{Type: "worker", ID: "wrk_01", Label: "Leon Park"},
	}
	for _, want := range expected {
		_, routable := productguide.RecordPath(want.Type, want.ID)
		require.True(t, routable, want.Type)
	}
	require.ElementsMatch(t, expected, refs)
}

func TestCitedRecordsLabelsADocumentByItsFileName(t *testing.T) {
	refs := CitedRecords(`[{"id": "doc_01", "originalName": "BOL-88213.pdf", "status": "Active"}]`)
	require.Equal(
		t,
		[]agentpagecontext.EntityRef{{Type: "document", ID: "doc_01", Label: "BOL-88213.pdf"}},
		refs,
	)
}

// An invoice row carries the PRO of the shipment it bills beside its own
// number; the invoice is named by its number, and the shipment keeps its PRO.
func TestCitedRecordsNamesAnInvoiceByItsNumberAndNotItsShipmentsPRO(t *testing.T) {
	refs := CitedRecords(`{"items": [
		{"id": "inv_01", "number": "INV-1042", "proNumber": "S-240193", "billTo": "Acme Foods"},
		{"id": "shp_01", "proNumber": "S-240193", "number": "7"}
	]}`)

	require.ElementsMatch(t, []agentpagecontext.EntityRef{
		{Type: "invoice", ID: "inv_01", Label: "INV-1042"},
		{Type: "shipment", ID: "shp_01", Label: "S-240193"},
	}, refs)
}

// A record with nothing of its own to be called by is still cited, under the
// PRO of its shipment, rather than left out of what the answer was built from.
func TestCitedRecordsFallsBackToTheShipmentsPROWhenARecordHasNoNameOfItsOwn(t *testing.T) {
	refs := CitedRecords(`[{"id": "inv_02", "proNumber": "S-240193", "status": "Draft"}]`)

	require.Equal(
		t,
		[]agentpagecontext.EntityRef{{Type: "invoice", ID: "inv_02", Label: "S-240193"}},
		refs,
	)
}

func TestCitedRecordsIgnoresTextAndOversizedResults(t *testing.T) {
	require.Nil(t, CitedRecords("Shipment S-240193 (shp_01) is late."))
	require.Nil(t, CitedRecords(""))
	require.Nil(t, CitedRecords("{not json"))
	require.Nil(
		t,
		CitedRecords(
			`{"id":"shp_01","proNumber":"S-1","pad":"`+strings.Repeat(
				"x",
				maxCitedContentBytes,
			)+`"}`,
		),
	)
}

func TestCitedRecordsStopsAtTheCap(t *testing.T) {
	var b strings.Builder
	b.WriteString("[")
	for i := range maxCitedRecords + 10 {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(`{"id":"shp_`)
		b.WriteString(strings.Repeat("a", i+1))
		b.WriteString(`","proNumber":"P`)
		b.WriteString(strings.Repeat("1", i+1))
		b.WriteString(`"}`)
	}
	b.WriteString("]")

	require.Len(t, CitedRecords(b.String()), maxCitedRecords)
}
