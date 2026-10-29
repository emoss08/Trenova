package gqlext

import (
	"github.com/emoss08/trenova/internal/modules/billing/domain/costingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/protectedfield"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionsimulation"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workercredential"
	"github.com/shopspring/decimal"
)

// protectedType is what stands behind one GraphQL type's protected fields.
//
// resource is the kind that classifies them: the type's own resource, or the
// kind that carries it (a tender's offers are the tender's). It is empty for a
// computed payload or a record no kind carries, whose fields have no tier of
// their own; such a type gives the reason in graphQLOnly or names an entity.
//
// entity builds the Go value the REST routes write the same record as, with
// every protected field set, so the test can see WithholdFields clear them.
// jsonNames maps a GraphQL field to that entity's JSON key where they differ.
// graphQLOnly says why no REST response carries the type, when none does.
type protectedType struct {
	resource    permission.Resource
	entity      func() protectedfield.Withholder
	jsonNames   map[string]string
	graphQLOnly string
}

func amount(value string) decimal.NullDecimal {
	return decimal.NewNullDecimal(decimal.RequireFromString(value))
}

func protectedTypes() map[string]protectedType {
	return map[string]protectedType{
		"WorkerProfile": {
			resource: permission.ResourceWorker,
			entity: func() protectedfield.Withholder {
				return &worker.WorkerProfile{
					DOB:            315532800,
					LicenseNumber:  "D1234567",
					TWICCardNumber: "TWIC-998877",
					LastDrugTest:   1767225600,
				}
			},
		},
		"WorkerCredential": {
			resource: permission.ResourceWorkerCredential,
			entity: func() protectedfield.Withholder {
				return &workercredential.WorkerCredential{Number: "CDL-445566"}
			},
		},
		"RateAgreement": {
			resource: permission.ResourceRateAgreement,
			entity: func() protectedfield.Withholder {
				return &rateagreement.RateAgreement{
					MarginFloorPercent:  amount("12.5"),
					MaxPayPercentOfSell: amount("85"),
				}
			},
		},
		"CostingControl": {
			resource: permission.ResourceCostingControl,
			entity: func() protectedfield.Withholder {
				return &costingcontrol.CostingControl{TargetMarginPercent: amount("15")}
			},
		},
		"TenderOffer": {
			resource: permission.ResourceTender,
			entity: func() protectedfield.Withholder {
				return &tender.TenderOffer{CounterMargin: amount("8.25")}
			},
			jsonNames: map[string]string{"counterMarginPercent": "counterMargin"},
		},
		"Shipment": {
			resource: permission.ResourceShipment,
			graphQLOnly: "the profitability estimate is a GraphQL field resolver; no REST " +
				"shipment carries it",
		},
		"DetentionOccurrence": {
			entity: func() protectedfield.Withholder {
				return &detention.DetentionOccurrence{
					DriverPayAmount: amount("75"),
					NetMargin:       amount("150"),
				}
			},
		},
		"FacilityDetentionStat": {
			entity: func() protectedfield.Withholder {
				return &detentionports.FacilityDetentionStat{
					DriverPayAmount: amount("75"),
					NetMargin:       amount("150"),
				}
			},
		},
		"CustomerDetentionStat": {
			entity: func() protectedfield.Withholder {
				return &detentionports.CustomerDetentionStat{
					DriverPayAmount: amount("75"),
					NetMargin:       amount("150"),
				}
			},
		},
		"DetentionPreviewResult": {
			entity: func() protectedfield.Withholder {
				return &detentionsimulation.PreviewResult{
					DriverPayAmount: amount("75"),
					NetMargin:       amount("150"),
				}
			},
		},
		"DetentionBacktestBucket": {
			entity: func() protectedfield.Withholder {
				return &detentionsimulation.BacktestBucket{
					DriverPayAmount: amount("75"),
					NetMargin:       amount("150"),
				}
			},
		},
		"DetentionBacktestResult": {
			entity: func() protectedfield.Withholder {
				stops := 2
				return &detentionsimulation.BacktestResult{
					ProposedDriverPay:   amount("75"),
					ProposedNetMargin:   amount("150"),
					NegativeMarginStops: &stops,
				}
			},
		},
		"CapacityMatch": {
			graphQLOnly: "capacity matches are computed for the shipment board's GraphQL read",
		},
		"FleetCostSummary": {
			graphQLOnly: "the fleet cost summary is read only through GraphQL",
		},
		"ResolvedCostProfile": {
			graphQLOnly: "the resolved cost profile is read only through GraphQL",
		},
		"ShipmentProfitabilityAnalytics": {
			graphQLOnly: "the REST analytics card is gated by the shipment analytics provider " +
				"(shipmentprovider/access.go), which both surfaces read",
		},
		"Query": {
			graphQLOnly: "a root field: the guard answers it null without resolving it",
		},
	}
}
