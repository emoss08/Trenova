package conform

import (
	"maps"
	"strings"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/edi-partner-sim/internal/partner"
	"github.com/emoss08/trenova/edi-partner-sim/internal/partner/reply"
)

const tmsEDI = "services/tms/internal/modules/edi/app/"

func TestPartnerReferenceMirrorsTheTMS(t *testing.T) {
	t.Parallel()

	long := strings.Repeat("K", maxExternalReference+5)
	cases := []struct {
		source string
		tender readTender
		want   string
	}{
		{
			source: tmsEDI + "editendermapping/build_shipment.go:externalShipmentReference",
			tender: readTender{Key: "  KVW-1001  ", BOL: "8100"},
			want:   "KVW-1001",
		},
		{
			source: tmsEDI + "editendermapping/build_shipment.go:externalShipmentReference",
			tender: readTender{Key: long},
			want:   long[:maxExternalReference],
		},
		{
			source: tmsEDI + "edipayloadbuild/payload.go:partnerReference",
			tender: readTender{Key: "   ", BOL: "8100"},
			want:   "8100",
		},
	}
	for _, tc := range cases {
		if got := partnerReference(&tc.tender); got != tc.want {
			t.Errorf("%s: %+v gives %q, want %q", tc.source, tc.tender, got, tc.want)
		}
	}
}

func TestStatusAndInvoiceKeysMirrorTheTMS(t *testing.T) {
	t.Parallel()

	cases := []struct {
		source string
		tender readTender
		want   string
	}{
		{
			source: tmsEDI + "edioutbound/document.go:freightInvoiceKey",
			tender: readTender{Key: "KVW-1001", BOL: "8100"},
			want:   "KVW-1001",
		},
		{
			source: tmsEDI + "edioutbound/document.go:freightInvoiceKey",
			tender: readTender{BOL: "8100"},
			want:   "8100",
		},
		{
			source: tmsEDI + "edioutbound/document.go:shipmentStatusKey",
			tender: readTender{},
			want:   "PRO0000001",
		},
		{
			source: tmsEDI + "edioutbound/document.go:freightInvoiceKey",
			tender: readTender{},
			want:   "PRO0000001",
		},
	}
	for _, tc := range cases {
		if got := shipmentKey(&tc.tender, proNumber(1)); got != tc.want {
			t.Errorf("%s: %+v gives %q, want %q", tc.source, tc.tender, got, tc.want)
		}
	}
}

func TestStopStatusCodesMirrorTheTMS(t *testing.T) {
	t.Parallel()

	const source = tmsEDI + "edipayloadbuild/payload.go:stopStatusCode"
	cases := []struct {
		pickup, departed, final bool
		code, kind, event       string
	}{
		{pickup: true, code: "X3", kind: "ArrivedPickup", event: eventStopArrived},
		{
			pickup: true, departed: true,
			code: "AF", kind: "DepartedPickup", event: eventStopDeparted,
		},
		{
			pickup: true, departed: true, final: true,
			code: "AF", kind: "DepartedPickup", event: eventStopDeparted,
		},
		{code: "X1", kind: "ArrivedDelivery", event: eventStopArrived},
		{final: true, code: "X1", kind: "ArrivedDelivery", event: eventStopArrived},
		{departed: true, code: "CD", kind: "DepartedDelivery", event: eventStopDeparted},
		{departed: true, final: true, code: "D1", kind: "Delivered", event: eventDelivered},
	}
	for _, tc := range cases {
		code := stopStatusCode(tc.pickup, tc.departed, tc.final)
		if code != tc.code {
			t.Errorf("%s: pickup %t departed %t final %t gives %s, want %s",
				source, tc.pickup, tc.departed, tc.final, code, tc.code)
		}
		if kind := statusKinds[code]; kind != tc.kind {
			t.Errorf("%sladingdoc/status.go:EventKindOf(%s) is %s, want %s",
				tmsEDI, code, kind, tc.kind)
		}
		if event := flowEventOf(tc.kind); event != tc.event {
			t.Errorf("%sedioutbound/document.go:statusEvent(%s) is %s, want %s",
				tmsEDI, code, event, tc.event)
		}
	}
}

func TestOnlyStopArrivalsAndDeparturesAnswerWithAStatus(t *testing.T) {
	t.Parallel()

	const source = tmsEDI + "edioutbound/flowevents.go:statusShipmentEvent"
	want := []string{"StopArrived", "StopDeparted"}
	if len(stopEvents) != len(want) {
		t.Fatalf("%s: %d events, want %v", source, len(stopEvents), want)
	}
	for index, e := range stopEvents {
		if e.eventType != want[index] {
			t.Errorf("%s: event %d is %s, want %s", source, index, e.eventType, want[index])
		}
	}
}

func TestTheFinalStopIsTheLastBySequence(t *testing.T) {
	t.Parallel()

	const source = tmsEDI + "edipayloadbuild/payload.go:completesShipment"
	stops := []shipmentStop{{sequence: 1, pickup: true}, {sequence: 3}, {sequence: 2}}
	if got := finalStop(stops); got != 1 {
		t.Errorf("%s: the final stop is index %d, want 1 (sequence 3)", source, got)
	}
}

func TestFreightTermsMirrorTheTMS(t *testing.T) {
	t.Parallel()

	cases := []struct {
		source, tendered, want string
	}{
		{tmsEDI + "ladingdoc/outbound.go:paymentOf", "Collect", "Collect"},
		{tmsEDI + "ladingdoc/outbound.go:paymentOf", "ThirdParty", "ThirdParty"},
		{"operations/domain/shipmentnormalize/normalize.go:FreightTerms", "", "Prepaid"},
		{"operations/domain/shipmentnormalize/normalize.go:FreightTerms", "PP", "Prepaid"},
	}
	for _, tc := range cases {
		if got := freightTerms(tc.tendered); got != tc.want {
			t.Errorf("%s: %q bills %q, want %q", tc.source, tc.tendered, got, tc.want)
		}
	}
}

func TestWholePoundsMirrorsTheTMS(t *testing.T) {
	t.Parallel()

	const source = tmsEDI + "ladingdoc/inbound.go:wholePounds"
	cases := []struct {
		value string
		want  string
	}{
		{"38000", "38000"},
		{"38000.5", "38001"},
		{"38000.4", "38000"},
		{"0", ""},
		{"-3", ""},
		{"heavy", ""},
	}
	for _, tc := range cases {
		got := wholePounds(&measure{Value: tc.value, Unit: unitPounds})
		switch {
		case tc.want == "" && got != nil:
			t.Errorf("%s: %q gives %+v, want none", source, tc.value, got)
		case tc.want != "" && (got == nil || got.Value != tc.want):
			t.Errorf("%s: %q gives %+v, want %s", source, tc.value, got, tc.want)
		}
	}
}

func TestAShipperStatusAndInvoiceCarryWhatTheTMSWrites(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, time.March, 9, 14, 0, 0, 0, time.UTC)
	load := &partner.Load{
		Seed:       7,
		TotalCents: 125_050,
		BillTo:     partner.Party{Name: "Kestrel"},
		Stops: []partner.Stop{
			{Sequence: 1, Kind: partner.StopPickup, Window: partner.Window{Start: start}},
			{
				Sequence: 2,
				Kind:     partner.StopDelivery,
				Window:   partner.Window{Start: start.Add(24 * time.Hour)},
			},
			{
				Sequence: 3,
				Kind:     partner.StopDelivery,
				Window: partner.Window{
					Start: start.Add(48 * time.Hour),
					End:   start.Add(50 * time.Hour),
				},
			},
		},
	}
	pieces := int64(22)
	tender := &readTender{
		Key:      "KVW-1001",
		BOL:      "8100",
		PONumber: " 8978300638 ",
		Payment:  "ThirdParty",
		Weight:   &measure{Value: "38000.4", Unit: unitPounds},
		Pieces:   &pieces,
		Stops: []readStop{
			{Sequence: 1, Type: stopPickup},
			{Sequence: 2, Type: stopDelivery},
			{Sequence: 3, Type: stopDelivery},
		},
	}
	var kinds []string
	for _, doc := range statuses(load, tender, func(string) bool { return true }, nil) {
		status, ok := doc.document.(shipmentStatus)
		if !ok {
			t.Fatalf("%s is a %T", doc.name, doc.document)
		}
		if status.Key != "KVW-1001" || status.ShipperReference != "KVW-1001" {
			t.Errorf("%s: key %q reference %q, want the tender's KVW-1001",
				doc.name, status.Key, status.ShipperReference)
		}
		if status.Weight == nil || status.Weight.Value != "38000" || *status.Pieces != 22 {
			t.Errorf("%s: weight %+v pieces %v, want the tender's", doc.name, status.Weight,
				status.Pieces)
		}
		wantRefs := map[string]string{refBillOfLading: "8100", refCarrierPRO: proNumber(load.Seed)}
		if !maps.Equal(status.Refs, wantRefs) {
			t.Errorf("%sedipayloadbuild/payload.go:partnerReferences: %s refs %v, want %v",
				tmsEDI, doc.name, status.Refs, wantRefs)
		}
		kinds = append(kinds, status.Events[0].Kind)
	}
	want := "ArrivedPickup DepartedPickup ArrivedDelivery DepartedDelivery " +
		"ArrivedDelivery Delivered"
	if got := strings.Join(kinds, " "); got != want {
		t.Errorf("the statuses are %s, want %s", got, want)
	}

	billed := invoice(load, tender, time.UTC, nil).document.(freightInvoice)
	if billed.Key != "KVW-1001" || billed.Payment != "ThirdParty" {
		t.Errorf("%sedioutbound/document.go:freightInvoiceKey and "+
			"edipayloadbuild/payload.go:shipmentFreightTerms: key %q payment %q",
			tmsEDI, billed.Key, billed.Payment)
	}
	if got := billed.Refs[refPurchaseOrder]; got != "8978300638" {
		t.Errorf("%seditendermapping/build_shipment.go:purchaseOrder, "+
			"edipayloadbuild/payload.go:BuildFreightInvoicePayload and "+
			"ladingdoc/outbound.go:invoiceRefs: the invoice quotes PO %q, want the tender's",
			tmsEDI, got)
	}

	wantRefs := map[string]string{
		refBillOfLading:  "8100",
		refCarrierPRO:    proNumber(load.Seed),
		refPurchaseOrder: "8978300638",
	}
	if !maps.Equal(billed.Refs, wantRefs) {
		t.Errorf("%sedipayloadbuild/payload.go:BuildFreightInvoicePayload: refs %v, want %v "+
			"(keyed by X12 qualifier, never a Trenova record id)", tmsEDI, billed.Refs, wantRefs)
	}

	tender.PONumber = "   "
	unordered := invoice(load, tender, time.UTC, nil).document.(freightInvoice)
	if got, ok := unordered.Refs[refPurchaseOrder]; ok {
		t.Errorf("%sladingdoc/outbound.go:invoiceRefs: a tender without a PO quotes %q",
			tmsEDI, got)
	}
}

func TestACarrierOfferCarriesWhatTheTMSWrites(t *testing.T) {
	t.Parallel()

	const source = tmsEDI + "editenderservice/tenderoffer.go:buildCarrierOfferDocumentPayload"
	thirdParty := false
	for seed := uint64(1); seed <= 40; seed++ {
		offered := reply.Offer(seed)
		thirdParty = thirdParty || offered.Terms == "ThirdParty"
		document, _, err := tenderFromOffer(offered)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(document.Key, "_") || document.Key != offered.Key {
			t.Errorf("%sedioutbound/document.go:loadTenderKey: seed %d keys the offer %q; "+
				"Trenova keys it by the shipment's PRO number, never a record id", tmsEDI, seed,
				document.Key)
		}
		if document.Payment != paymentPrepaid {
			t.Errorf("%s: seed %d on %s terms pays %q, Trenova always offers Prepaid", source,
				seed, offered.Terms, document.Payment)
		}
		if document.Equipment != offered.Equipment || document.PONumber != offered.PONumber {
			t.Errorf("%s: seed %d equipment %q po %q, offered %q %q", source, seed,
				document.Equipment, document.PONumber, offered.Equipment, offered.PONumber)
		}
		if document.FreightCharge == nil || *document.FreightCharge != *document.Total {
			t.Errorf("%s: seed %d freight charge %+v is not the offered total %+v", source,
				seed, document.FreightCharge, document.Total)
		}
		encoded, err := sonic.Marshal(document)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(encoded), `"country"`) {
			t.Errorf("%sladingdoc/outbound.go:FromLoadTender writes no stop country: %s",
				tmsEDI, encoded)
		}
	}
	if !thirdParty {
		t.Errorf("%s: no offer was for a third-party shipment, so the rule went untested", source)
	}
}
