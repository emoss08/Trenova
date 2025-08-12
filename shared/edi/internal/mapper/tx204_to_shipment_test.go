package mapper

import (
	"os"
	"testing"

	tx204 "github.com/emoss08/trenova/shared/edi/internal/tx/tx204"
	"github.com/emoss08/trenova/shared/edi/internal/x12"
)

func TestToShipment_Sample1(t *testing.T) {
	raw, err := os.ReadFile("../../testdata/204/sample1.edi")
	if err != nil {
		t.Skipf("sample not available: %v", err)
	}
	delims, err := x12.DetectDelimiters(raw)
	if err != nil {
		t.Fatalf("delims: %v", err)
	}
	segs, err := x12.ParseSegments(raw, delims)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	lt := tx204.BuildFromSegments(segs)
	shp := ToShipment(lt)
	if shp.CarrierSCAC == "" || shp.ShipmentID == "" {
		t.Fatalf("expected SCAC and ShipmentID, got: %#v", shp)
	}
	if len(shp.Stops) != 2 {
		t.Fatalf("expected 2 stops, got %d", len(shp.Stops))
	}
	if shp.Stops[0].Type != "pickup" || shp.Stops[1].Type != "delivery" {
		t.Fatalf("unexpected stop types: %#v", shp.Stops)
	}
}

func TestOptions_RawL11_EquipmentNorm_ShipmentIDModes(t *testing.T) {
	// Construct a minimal LoadTender
	lt := tx204.LoadTender{}
	lt.Header.CarrierSCAC = ""
	lt.Header.ShipmentID = "B2ID"
	lt.Header.References = map[string][]string{
		"PO": {"POVAL"},
		"CR": {"REFID"},
		"ZZ": {"IGNORED"},
	}
	lt.Equipment.Type = "VEH"
	// Baseline opts
	opts := DefaultOptions()
	opts.IncludeRawL11 = true
	opts.RawL11Filter = []string{"PO", "CR"}
	opts.EquipmentTypeMap = map[string]string{"VEH": "trailer"}

	// ref_first: prefers CR over B2ID
	opts.ShipmentIDMode = "ref_first"
	opts.ShipmentIDQuals = []string{"CR", "SI"}
	out := ToShipmentWithOptions(lt, opts)
	if out.ShipmentID != "REFID" {
		t.Fatalf("expected shipment_id from CR, got %q", out.ShipmentID)
	}
	if out.Equipment.Type != "trailer" {
		t.Fatalf("expected normalized equipment type 'trailer', got %q", out.Equipment.Type)
	}
	if out.ReferencesRaw["PO"][0] != "POVAL" || out.ReferencesRaw["CR"][0] != "REFID" {
		t.Fatalf("unexpected references_raw: %#v", out.ReferencesRaw)
	}
	if _, ok := out.ReferencesRaw["ZZ"]; ok {
		t.Fatalf("ZZ should have been filtered out: %#v", out.ReferencesRaw)
	}

	// b2_first: prefers B2ID over CR
	opts.ShipmentIDMode = "b2_first"
	out = ToShipmentWithOptions(lt, opts)
	if out.ShipmentID != "B2ID" {
		t.Fatalf("expected shipment_id from B2-03, got %q", out.ShipmentID)
	}
	// ref_only: only from refs
	opts.ShipmentIDMode = "ref_only"
	out = ToShipmentWithOptions(lt, opts)
	if out.ShipmentID != "REFID" {
		t.Fatalf("expected shipment_id from refs only, got %q", out.ShipmentID)
	}
	// b2_only: only from B2
	opts.ShipmentIDMode = "b2_only"
	out = ToShipmentWithOptions(lt, opts)
	if out.ShipmentID != "B2ID" {
		t.Fatalf("expected shipment_id from B2 only, got %q", out.ShipmentID)
	}
}

func TestOptions_DateTimeNormalization(t *testing.T) {
	// Build minimal segments to flow through BuildFromSegments
	segs := []x12.Segment{
		{Tag: "ST"},
		{Tag: "S5", Elements: [][]string{{"1"}, {"LD"}}},
		{Tag: "DTM", Elements: [][]string{{"133"}, {"20240102"}, {"0800"}}},
		{Tag: "SE"},
	}
	lt := tx204.BuildFromSegments(segs)
	opts := DefaultOptions()
	opts.EmitISODateTime = true
	opts.Timezone = "UTC"
	shp := ToShipmentWithOptions(lt, opts)
	if len(shp.Stops) == 0 || len(shp.Stops[0].Appointments) == 0 {
		t.Fatalf("expected a normalized appointment")
	}
	if got := shp.Stops[0].Appointments[0].DateTime; got != "2024-01-02T08:00:00Z" {
		t.Fatalf("expected ISO datetime, got %q", got)
	}
}

func TestMapping_Totals_Commodities_FedEx(t *testing.T) {
	raw, err := os.ReadFile("../../testdata/204/fedex.edi")
	if err != nil {
		t.Skip("fedex sample not present")
	}
	delims, err := x12.DetectDelimiters(raw)
	if err != nil {
		t.Fatalf("delims: %v", err)
	}
	segs, err := x12.ParseSegments(raw, delims)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	lt := tx204.BuildFromSegments(segs)
	shp := ToShipmentWithOptions(lt, DefaultOptions())
	if shp.Totals.Weight == "" {
		t.Fatalf("expected totals weight populated from L3/AT8")
	}
	// FedEx sample uses pounds; unit may be 'L'
	if len(shp.Goods) == 0 {
		t.Fatalf("expected at least one commodity from L5")
	}
}

func TestMapping_ServiceLevel_Accessorials_FromRefs(t *testing.T) {
	// Build a minimal LT with header references for service level and accessorials
	lt := tx204.LoadTender{}
	lt.Header.References = map[string][]string{
		"SV": {"EXPRESS"},
		"AC": {"LIFTGATE", "INSIDE"},
	}
	opts := DefaultOptions()
	opts.ServiceLevelQuals = []string{"SV"}
	opts.ServiceLevelMap = map[string]string{"EXPRESS": "expedited"}
	opts.AccessorialQuals = []string{"AC"}
	opts.AccessorialMap = map[string]string{"LIFTGATE": "liftgate", "INSIDE": "inside_delivery"}
	shp := ToShipmentWithOptions(lt, opts)
	if shp.ServiceLevel != "expedited" {
		t.Fatalf("expected normalized service level, got %q", shp.ServiceLevel)
	}
	if len(shp.Accessorials) != 2 {
		t.Fatalf("expected 2 accessorials, got %d", len(shp.Accessorials))
	}
	codes := []string{shp.Accessorials[0].Code, shp.Accessorials[1].Code}
	names := []string{shp.Accessorials[0].Name, shp.Accessorials[1].Name}
	if codes[0] == codes[1] || names[0] == names[1] {
		t.Fatalf("accessorials not mapped as expected: %#v", shp.Accessorials)
	}
}
