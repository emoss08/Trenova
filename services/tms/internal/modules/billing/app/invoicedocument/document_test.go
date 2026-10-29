package invoicedocument

import (
	"image"
	"image/color"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/billingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestBuildInvoicePDFDataMapsBillToAndRemitSeparately(t *testing.T) {
	t.Parallel()

	state := &usstate.UsState{Abbreviation: "TN", CountryName: "USA"}
	entity := &invoice.Invoice{
		Number:             "INV-1001",
		BillToName:         "Snapshot Customer",
		BillToAddressLine1: "100 Snapshot Ave",
		BillToCity:         "Nashville",
		BillToPostalCode:   "37201",
		RemittanceInstructions: strings.Join([]string{
			"ACH preferred",
			"Account ending 4321",
		}, "\n"),
		CurrencyCode: "USD",
	}
	cus := &customer.Customer{
		Name:         "Fallback Customer",
		AddressLine1: "999 Customer Rd",
		AddressLine2: "Suite AP",
		City:         "Memphis",
		PostalCode:   "38103",
		State:        state,
	}
	org := &tenant.Organization{
		Name:         "Carrier Organization",
		AddressLine1: "500 Remit St",
		City:         "Knoxville",
		PostalCode:   "37902",
		State:        state,
	}

	data := buildInvoicePDFData(entity, &DeliveryProfile{
		Customer:     cus,
		Organization: org,
	})

	require.Equal(t, "Snapshot Customer", data.BillTo.Name)
	require.Contains(t, data.BillTo.Lines, "100 Snapshot Ave")
	require.Contains(t, data.BillTo.Lines, "Suite AP")
	require.NotContains(
		t,
		append([]string{data.BillTo.Name}, data.BillTo.Lines...),
		"Carrier Organization",
	)
	require.Equal(t, "Carrier Organization", data.RemitTo.Name)
	require.Contains(t, data.RemitTo.Lines, "500 Remit St")
	require.Contains(t, data.RemitTo.Lines, "ACH preferred")
	require.NotContains(
		t,
		append([]string{data.RemitTo.Name}, data.RemitTo.Lines...),
		"Fallback Customer",
	)
}

func TestBuildInvoicePDFDataMapsShipmentStopsToShipperAndConsignee(t *testing.T) {
	t.Parallel()

	state := &usstate.UsState{Abbreviation: "GA", CountryName: "USA"}
	pickupDate := int64(1_700_000_000)
	deliveryDate := int64(1_700_086_400)
	shp := &shipment.Shipment{
		ActualShipDate:     &pickupDate,
		ActualDeliveryDate: &deliveryDate,
		Moves: []*shipment.ShipmentMove{
			{
				Stops: []*shipment.Stop{
					{
						Type:     shipmentvocab.StopTypeDelivery,
						Sequence: 2,
						Location: &location.Location{
							Name:         "Consignee DC",
							AddressLine1: "200 Delivery Ln",
							City:         "Atlanta",
							PostalCode:   "30301",
							State:        state,
						},
					},
					{
						Type:     shipmentvocab.StopTypeDelivery,
						Sequence: 3,
						Location: &location.Location{
							Name:         "Final Delivery",
							AddressLine1: "300 Final Ln",
							City:         "Macon",
							PostalCode:   "31201",
							State:        state,
						},
					},
					{
						Type:     shipmentvocab.StopTypePickup,
						Sequence: 1,
						Location: &location.Location{
							Name:         "Shipper Plant",
							AddressLine1: "100 Pickup Rd",
							City:         "Savannah",
							PostalCode:   "31401",
							State:        state,
						},
					},
				},
			},
		},
	}

	data := buildInvoicePDFData(
		&invoice.Invoice{CurrencyCode: "USD"},
		&DeliveryProfile{Shipment: shp},
	)

	require.Equal(t, "Shipper Plant", data.Shipper.Name)
	require.Contains(t, data.Shipper.Lines, "100 Pickup Rd")
	require.Equal(t, "Consignee DC", data.Consignee.Name)
	require.Contains(t, data.Consignee.Lines, "200 Delivery Ln")
	require.NotEqual(t, data.Shipper.Name, data.Consignee.Name)
	require.Equal(t, []invoicePDFKeyValue{
		{Label: "Pickup Date", Value: UnixDatePtr(&pickupDate)},
	}, data.Shipper.Details)
	require.Equal(t, []invoicePDFKeyValue{
		{Label: "Delivery Date", Value: UnixDatePtr(&deliveryDate)},
	}, data.Consignee.Details)
}

func TestShipmentStopSelectionHelpers(t *testing.T) {
	t.Parallel()

	shp := &shipment.Shipment{
		Moves: []*shipment.ShipmentMove{
			{
				Stops: []*shipment.Stop{
					{
						Type:     shipmentvocab.StopTypeDelivery,
						Sequence: 4,
						Location: &location.Location{
							Name:       "Final Delivery",
							City:       "Macon",
							PostalCode: "31201",
						},
					},
					{
						Type:     shipmentvocab.StopTypePickup,
						Sequence: 2,
						Location: &location.Location{
							Name:       "First Pickup",
							City:       "Savannah",
							PostalCode: "31401",
						},
					},
					{
						Type:     shipmentvocab.StopTypeDelivery,
						Sequence: 3,
						Location: &location.Location{
							Name:       "First Delivery",
							City:       "Atlanta",
							PostalCode: "30301",
						},
					},
					{
						Type:     shipmentvocab.StopTypePickup,
						Sequence: 5,
						Location: &location.Location{Name: "Later Pickup"},
					},
				},
			},
		},
	}

	require.Equal(t, int64(2), firstPickupStop(shp).Sequence)
	require.Equal(t, int64(3), firstDeliveryStop(shp).Sequence)
	require.Equal(t, int64(4), finalDeliveryStop(shp).Sequence)
	require.Equal(t, "First Pickup Savannah 31401", ShipmentOrigin(shp))
	require.Equal(t, "Final Delivery Macon 31201", ShipmentDestination(shp))
}

func TestBuildInvoicePDFDataMapsShipmentCommodities(t *testing.T) {
	t.Parallel()

	shp := &shipment.Shipment{
		Commodities: []*shipment.ShipmentCommodity{
			{
				Pieces: 20,
				Weight: 40000,
				Commodity: &commodity.Commodity{
					Name:         "Industrial Parts",
					Description:  "SKU 12345",
					FreightClass: commodity.FreightClass125,
				},
			},
			nil,
		},
	}

	data := buildInvoicePDFData(&invoice.Invoice{CurrencyCode: "USD"}, &DeliveryProfile{
		Shipment: shp,
	})

	require.Equal(t, []invoicePDFCommodityRow{
		{
			Quantity:         "20",
			DescriptionLines: []string{"Industrial Parts", "SKU 12345"},
			Weight:           "40,000",
			Class:            "125",
			PiecesValue:      20,
			WeightValue:      40000,
		},
	}, data.CommodityRows)
}

func TestBuildInvoicePDFDataOmitsMissingOptionalRows(t *testing.T) {
	t.Parallel()

	data := buildInvoicePDFData(&invoice.Invoice{
		Number:                 "INV-1001",
		CurrencyCode:           "USD",
		RemittanceInstructions: "\nACH only\n\n",
	}, &DeliveryProfile{
		Organization: &tenant.Organization{Name: "Carrier Organization"},
	})

	require.Equal(t, []string{"ACH only"}, data.RemitTo.Lines)
	require.Empty(t, data.Terms)
	require.NotContains(t, data.RemitTo.Lines, "")
	require.NotContains(t, data.BillTo.Lines, "")
	require.Empty(t, data.CommodityRows)
	require.Empty(t, data.InvoiceTerms)
	require.Empty(t, data.InvoiceFooter)
}

func TestBuildInvoicePDFDataUsesBillingControlTermsAndFooter(t *testing.T) {
	t.Parallel()

	data := buildInvoicePDFData(&invoice.Invoice{CurrencyCode: "USD"}, &DeliveryProfile{
		BillingControl: &billingcontrol.BillingControl{
			DefaultInvoiceTerms:  "\nFirst invoice term.\n\nSecond invoice term.\n",
			DefaultInvoiceFooter: "  Thank you for your business.  ",
		},
	})

	require.Equal(t, []string{"First invoice term.", "Second invoice term."}, data.InvoiceTerms)
	require.Equal(t, "Thank you for your business.", data.InvoiceFooter)
}

func TestBuildInvoicePDFDataUsesBillingControlInvoiceDisplayOptions(t *testing.T) {
	t.Parallel()

	dueDate := int64(1_700_086_400)
	entity := &invoice.Invoice{
		CurrencyCode:   "USD",
		PaymentTerm:    invoice.PaymentTermNet30,
		DueDate:        &dueDate,
		TotalAmount:    decimal.NewFromInt(1250),
		AppliedAmount:  decimal.NewFromInt(250),
		SubtotalAmount: decimal.NewFromInt(1250),
	}

	defaulted := buildInvoicePDFData(entity, nil)
	require.Equal(t, "2023-11-15", defaulted.DueDate)
	require.Equal(t, "USD 1000.00", defaulted.BalanceDue)
	require.Contains(t, defaulted.Terms, "Due Date: 2023-11-15")

	shown := buildInvoicePDFData(entity, &DeliveryProfile{
		BillingControl: &billingcontrol.BillingControl{
			ShowDueDateOnInvoice:    true,
			ShowBalanceDueOnInvoice: true,
		},
	})
	require.Equal(t, "2023-11-15", shown.DueDate)
	require.Equal(t, "USD 1000.00", shown.BalanceDue)
	require.Contains(t, shown.Terms, "Due Date: 2023-11-15")

	hidden := buildInvoicePDFData(entity, &DeliveryProfile{
		BillingControl: &billingcontrol.BillingControl{},
	})
	require.Empty(t, hidden.DueDate)
	require.Empty(t, hidden.BalanceDue)
	require.Equal(t, []string{"Payment Terms: Net30"}, hidden.Terms)
}

func TestBuildInvoicePDFDataChargesMatchInvoiceTotals(t *testing.T) {
	t.Parallel()

	entity := &invoice.Invoice{
		CurrencyCode:   "USD",
		SubtotalAmount: decimal.NewFromInt(100),
		OtherAmount:    decimal.NewFromInt(25),
		TotalAmount:    decimal.NewFromInt(125),
		Lines: []*invoice.InvoiceLine{
			{
				LineNumber:  1,
				Description: "Freight",
				Quantity:    decimal.NewFromInt(1),
				UnitPrice:   decimal.NewFromInt(100),
				Amount:      decimal.NewFromInt(100),
			},
			{
				LineNumber:  2,
				Description: "Detention",
				Quantity:    decimal.NewFromInt(1),
				UnitPrice:   decimal.NewFromInt(25),
				Amount:      decimal.NewFromInt(25),
			},
		},
	}

	data := buildInvoicePDFData(entity, nil)

	require.Len(t, data.ChargeRows, 2)
	require.Equal(t, "USD 100.00", data.Subtotal)
	require.Equal(t, "USD 25.00", data.Other)
	require.Equal(t, "USD 125.00", data.Total)
	require.Equal(t, "USD 100.00", data.ChargeRows[0].Amount)
	require.Equal(t, "USD 25.00", data.ChargeRows[1].Amount)
}

func TestBuildInvoicePDFDataHeaderRowsOmitUnavailableMetadata(t *testing.T) {
	t.Parallel()

	data := buildInvoicePDFData(&invoice.Invoice{
		CurrencyCode:      "USD",
		PaymentTerm:       invoice.PaymentTermNet30,
		ShipmentProNumber: "PRO123",
	}, &DeliveryProfile{
		Organization: &tenant.Organization{
			DOTNumber: "1234567",
			ScacCode:  "TRNV",
		},
	})

	require.Equal(t, []invoicePDFKeyValue{
		{Label: "DOT", Value: "1234567"},
		{Label: "SCAC", Value: "TRNV"},
		{Label: "Payment Terms", Value: "Net30"},
		{Label: "PRO", Value: "PRO123"},
	}, data.HeaderRows)

	missing := buildInvoicePDFData(
		&invoice.Invoice{CurrencyCode: "USD"},
		&DeliveryProfile{Organization: &tenant.Organization{}},
	)
	require.Empty(t, missing.HeaderRows)
}

func testInvoiceForPDF(invoiceID, orgID, buID pulid.ID) *invoice.Invoice {
	return &invoice.Invoice{
		ID:                 invoiceID,
		OrganizationID:     orgID,
		BusinessUnitID:     buID,
		BillingQueueItemID: pulid.MustNew("bqi_"),
		ShipmentID:         pulid.MustNew("shp_"),
		CustomerID:         pulid.MustNew("cus_"),
		Number:             "INV-1001",
		BillType:           billingqueue.BillTypeInvoice,
		Status:             invoice.StatusDraft,
		PaymentTerm:        invoice.PaymentTermNet30,
		CurrencyCode:       "USD",
		InvoiceDate:        1_700_000_000,
		BillToName:         "Acme Logistics",
		SubtotalAmount:     decimal.NewFromInt(100),
		TotalAmount:        decimal.NewFromInt(100),
	}
}

func testInvoicePDFDeliveryProfile() *DeliveryProfile {
	state := &usstate.UsState{Abbreviation: "IL", CountryName: "USA"}
	pickupDate := int64(1_699_900_000)
	deliveryDate := int64(1_700_100_000)

	return &DeliveryProfile{
		Organization: &tenant.Organization{
			Name:         "Trenova Logistics",
			AddressLine1: "1 Market Street",
			City:         "Chicago",
			State:        state,
			PostalCode:   "60654",
			DOTNumber:    "1234567",
			ScacCode:     "TRNV",
		},
		BillingControl: &billingcontrol.BillingControl{
			ShowDueDateOnInvoice:    true,
			ShowBalanceDueOnInvoice: true,
			DefaultInvoiceTerms: strings.Join([]string{
				"Carrier agrees that all services are performed subject to the terms and conditions previously executed between Carrier and Customer.",
				"Invoice charges are true and correct and transportation services were performed as described above.",
			}, "\n"),
			DefaultInvoiceFooter: "Thank you for your business. Call (630) 954-0200 or email apinvoices@trenova.com with any questions.",
		},
		Shipment: &shipment.Shipment{
			ProNumber:          "PRO123",
			BOL:                "BOL123",
			ActualShipDate:     &pickupDate,
			ActualDeliveryDate: &deliveryDate,
			Moves: []*shipment.ShipmentMove{
				{
					Stops: []*shipment.Stop{
						{
							Type:     shipmentvocab.StopTypePickup,
							Sequence: 1,
							Location: &location.Location{
								Name:         "Origin DC",
								AddressLine1: "100 Pickup Rd",
								City:         "Chicago",
								State:        state,
								PostalCode:   "60654",
							},
						},
						{
							Type:     shipmentvocab.StopTypeDelivery,
							Sequence: 2,
							Location: &location.Location{
								Name:         "Destination DC",
								AddressLine1: "200 Delivery Ln",
								City:         "Chicago",
								State:        state,
								PostalCode:   "60654",
							},
						},
					},
				},
			},
			Commodities: []*shipment.ShipmentCommodity{
				{
					Pieces: 20,
					Weight: 40000,
					Commodity: &commodity.Commodity{
						Name:         "Industrial Parts",
						Description:  "SKU 12345",
						FreightClass: commodity.FreightClass125,
					},
				},
			},
		},
	}
}

func testLogoImage() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 20, G: 80, B: 120, A: 255})
	img.Set(1, 0, color.RGBA{R: 20, G: 80, B: 120, A: 255})
	img.Set(0, 1, color.RGBA{R: 255, G: 255, B: 255, A: 255})
	img.Set(1, 1, color.RGBA{R: 20, G: 80, B: 120, A: 255})
	return img
}

func TestAutoSendsOnGenerationOnlyForPostedInvoices(t *testing.T) {
	t.Parallel()

	cus := &customer.Customer{BillingProfile: &customer.CustomerBillingProfile{
		AutoSendInvoiceOnGeneration: true,
		EmailInvoiceEnabled:         true,
	}}

	require.False(t, AutoSendsOnGeneration(cus, &invoice.Invoice{
		Status:     invoice.StatusDraft,
		SendStatus: invoice.SendStatusNotSent,
	}))
	require.True(t, AutoSendsOnGeneration(cus, &invoice.Invoice{
		Status:     invoice.StatusPosted,
		SendStatus: invoice.SendStatusNotSent,
	}))
	require.False(t, AutoSendsOnGeneration(cus, &invoice.Invoice{
		Status:     invoice.StatusPosted,
		SendStatus: invoice.SendStatusSent,
	}))
}
