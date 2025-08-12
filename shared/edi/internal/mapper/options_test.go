package mapper

import (
	"testing"

	tx204 "github.com/emoss08/trenova/shared/edi/internal/tx/tx204"
)

func TestOptions_PartyRoles_StopTypeMap_SCACFallback(t *testing.T) {
	lt := tx204.LoadTender{}
	lt.Header.CarrierSCAC = ""
	lt.Parties = map[string]tx204.Party{
		"SF": {Code: "SF", Name: "Shipper From"},
		"CN": {Code: "CN", Name: "Consignee"},
		"BT": {Code: "BT", Name: "BillTo"},
	}
	lt.Stops = []tx204.Stop{{Sequence: 1, Type: "CL"}, {Sequence: 2, Type: "CU"}}

	opts := DefaultOptions()
	opts.PartyRoles = map[string][]string{
		"shipper":   {"SF"},
		"consignee": {"CN"},
		"bill_to":   {"BT"},
	}
	opts.StopTypeMap = map[string]string{"CL": "pickup", "CU": "delivery"}
	opts.CarrierSCACFallback = "MYSC"

	out := ToShipmentWithOptions(lt, opts)
	if out.CarrierSCAC != "MYSC" {
		t.Fatalf("expected SCAC fallback 'MYSC', got %q", out.CarrierSCAC)
	}
	if out.Shipper == nil || out.Shipper.Name != "Shipper From" {
		t.Fatalf("expected shipper from SF, got %#v", out.Shipper)
	}
	if out.Consignee == nil || out.Consignee.Name != "Consignee" {
		t.Fatalf("expected consignee from CN, got %#v", out.Consignee)
	}
	if out.BillTo == nil || out.BillTo.Name != "BillTo" {
		t.Fatalf("expected bill_to from BT, got %#v", out.BillTo)
	}
	if len(out.Stops) != 2 || out.Stops[0].Type != "pickup" || out.Stops[1].Type != "delivery" {
		t.Fatalf("unexpected stop types: %#v", out.Stops)
	}
}
