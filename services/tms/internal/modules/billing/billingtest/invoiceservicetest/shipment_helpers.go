package invoiceservicetest

import (
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

func Leg(pro string, freight int64) *shipment.Shipment {
	serviceDate := int64(1_700_000_000)
	return &shipment.Shipment{
		ID:                  pulid.MustNew("shp_"),
		CustomerID:          pulid.MustNew("cus_"),
		ProNumber:           pro,
		BOL:                 "BOL-" + pro,
		ActualDeliveryDate:  &serviceDate,
		FreightChargeAmount: decimal.NewNullDecimal(decimal.NewFromInt(freight)),
		TotalChargeAmount:   decimal.NewNullDecimal(decimal.NewFromInt(freight)),
	}
}

func FreightSplit(leg *shipment.Shipment, payer, other pulid.ID, payerPercent int64) {
	leg.ChargeAllocations = []*shipment.ChargeAllocation{
		{
			ID:               pulid.MustNew("chal_"),
			ChargeKind:       shipmentvocab.ChargeAllocationKindFreight,
			BillToCustomerID: payer,
			Method:           shipmentvocab.ChargeAllocationMethodPercent,
			Percent:          decimal.NewNullDecimal(decimal.NewFromInt(payerPercent)),
		},
		{
			ID:               pulid.MustNew("chal_"),
			ChargeKind:       shipmentvocab.ChargeAllocationKindFreight,
			BillToCustomerID: other,
			Method:           shipmentvocab.ChargeAllocationMethodPercent,
			Percent:          decimal.NewNullDecimal(decimal.NewFromInt(100 - payerPercent)),
			Sequence:         1,
		},
	}
}
