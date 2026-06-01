package servicefailure

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

func TestReasonCodeValidateNormalizesCodeAndX12Defaults(t *testing.T) {
	reason := &ReasonCode{
		OrganizationID:       pulid.MustNew("org_"),
		BusinessUnitID:       pulid.MustNew("bu_"),
		Code:                 " late_delivery ",
		Label:                " Late Delivery ",
		Description:          " Facility missed appointment ",
		Category:             ReasonCategoryFacility,
		AppliesTo:            ReasonCodeAppliesToDelivery,
		DefaultStatusCode:    " sd ",
		DefaultReasonCode:    " ns ",
		DefaultExceptionCode: " a3 ",
		DefaultNote:          "  customer-safe note  ",
	}

	multiErr := errortypes.NewMultiError()
	reason.Validate(multiErr)
	if multiErr.HasErrors() {
		t.Fatalf("expected reason code to validate, got %v", multiErr)
	}

	if reason.Code != "LATE_DELIVERY" {
		t.Fatalf("expected normalized code, got %q", reason.Code)
	}
	if reason.DefaultStatusCode != "SD" {
		t.Fatalf("expected normalized status code, got %q", reason.DefaultStatusCode)
	}
	if reason.DefaultReasonCode != "NS" {
		t.Fatalf("expected normalized reason code, got %q", reason.DefaultReasonCode)
	}
	if reason.DefaultExceptionCode != "A3" {
		t.Fatalf("expected normalized exception code, got %q", reason.DefaultExceptionCode)
	}
	if reason.DefaultNote != "customer-safe note" {
		t.Fatalf("expected trimmed default note, got %q", reason.DefaultNote)
	}
}

func TestReasonCodeAppliesToAllowsExpectedStopTypes(t *testing.T) {
	tests := []struct {
		name     string
		applies  ReasonCodeAppliesTo
		stopType shipment.StopType
		want     bool
	}{
		{name: "pickup allows pickup", applies: ReasonCodeAppliesToPickup, stopType: shipment.StopTypePickup, want: true},
		{name: "pickup allows split pickup", applies: ReasonCodeAppliesToPickup, stopType: shipment.StopTypeSplitPickup, want: true},
		{name: "pickup rejects delivery", applies: ReasonCodeAppliesToPickup, stopType: shipment.StopTypeDelivery},
		{name: "delivery allows delivery", applies: ReasonCodeAppliesToDelivery, stopType: shipment.StopTypeDelivery, want: true},
		{name: "delivery allows split delivery", applies: ReasonCodeAppliesToDelivery, stopType: shipment.StopTypeSplitDelivery, want: true},
		{name: "both allows pickup", applies: ReasonCodeAppliesToBoth, stopType: shipment.StopTypePickup, want: true},
		{name: "both allows delivery", applies: ReasonCodeAppliesToBoth, stopType: shipment.StopTypeDelivery, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.applies.AllowsStopType(tt.stopType); got != tt.want {
				t.Fatalf("AllowsStopType() = %v, want %v", got, tt.want)
			}
		})
	}
}
