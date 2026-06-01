//go:build invoice_pdf_preview

package invoiceservice

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/commodity"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
	portstorage "github.com/emoss08/trenova/internal/core/ports/storage"
	"github.com/emoss08/trenova/shared/pulid"
	storagetest "github.com/emoss08/trenova/shared/testutil/storage"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestWriteInvoicePDFPreview(t *testing.T) {
	outputDir := os.Getenv("INVOICE_PDF_PREVIEW_DIR")
	if outputDir == "" {
		t.Skip("INVOICE_PDF_PREVIEW_DIR is required")
	}

	require.NoError(t, os.MkdirAll(outputDir, 0o755))
	content, err := renderInvoicePDF(
		t.Context(),
		previewInvoicePDFEntity(),
		previewInvoicePDFDeliveryProfile(),
		previewInvoicePDFStorage(t),
	)
	require.NoError(t, err)
	require.NotEmpty(t, content)
	require.NoError(t, os.WriteFile(filepath.Join(outputDir, "invoice-preview.pdf"), content, 0o644))
}

func previewInvoicePDFEntity() *invoice.Invoice {
	invoiceDate := previewUnixDate(2026, time.May, 31)
	dueDate := previewUnixDate(2026, time.June, 30)
	serviceDate := previewUnixDate(2026, time.May, 28)

	return &invoice.Invoice{
		ID:                     pulid.MustNew("inv_"),
		OrganizationID:         pulid.MustNew("org_"),
		BusinessUnitID:         pulid.MustNew("bu_"),
		BillingQueueItemID:     pulid.MustNew("bqi_"),
		ShipmentID:             pulid.MustNew("shp_"),
		CustomerID:             pulid.MustNew("cus_"),
		Number:                 "INV2605000001",
		BillType:               billingqueue.BillTypeInvoice,
		Status:                 invoice.StatusDraft,
		PaymentTerm:            invoice.PaymentTermNet30,
		CurrencyCode:           "USD",
		InvoiceDate:            invoiceDate,
		DueDate:                &dueDate,
		ServiceDate:            &serviceDate,
		ShipmentProNumber:      "SEED-SHP-005",
		ShipmentBOL:            "BOL-2026-0005",
		BillToName:             "Acme Manufacturing",
		BillToAddressLine1:     "400 W Superior St",
		BillToCity:             "Chicago",
		BillToState:            "IL",
		BillToPostalCode:       "60654",
		RemittanceInstructions: "",
		SubtotalAmount:         decimal.NewFromInt(2800),
		OtherAmount:            decimal.NewFromInt(150),
		TotalAmount:            decimal.NewFromInt(2950),
		Lines: []*invoice.InoviceLine{
			{
				LineNumber:  1,
				Type:        invoice.InvoiceLineTypeFreight,
				Description: "Freight charge",
				Quantity:    decimal.NewFromInt(1),
				UnitPrice:   decimal.NewFromInt(2800),
				Amount:      decimal.NewFromInt(2800),
			},
			{
				LineNumber:  2,
				Type:        invoice.InvoiceLineTypeAccessorial,
				Description: "Detention Fee",
				Quantity:    decimal.NewFromInt(2),
				UnitPrice:   decimal.NewFromInt(75),
				Amount:      decimal.NewFromInt(150),
			},
		},
	}
}

func previewInvoicePDFDeliveryProfile() *invoiceDeliveryProfile {
	texas := &usstate.UsState{Abbreviation: "TX", CountryName: "USA"}
	illinois := &usstate.UsState{Abbreviation: "IL", CountryName: "USA"}
	pickupDate := previewUnixDate(2026, time.May, 25)
	deliveryDate := previewUnixDate(2026, time.May, 28)
	pieces := int64(26)
	weight := int64(40000)

	return &invoiceDeliveryProfile{
		Organization: &tenant.Organization{
			Name:         "Trenova Logistics",
			LogoURL:      "logos/invoice-preview.png",
			AddressLine1: "1 Market Street",
			City:         "Los Angeles",
			PostalCode:   "90001",
			DOTNumber:    "1234567",
			ScacCode:     "TRNV",
		},
		BillingControl: &tenant.BillingControl{
			DefaultInvoiceTerms: strings.Join([]string{
				"Carrier agrees that all services are performed subject to the terms and conditions previously executed between Carrier and Customer.",
				"Invoice charges are true and correct and transportation services were performed as described above.",
			}, "\n"),
			DefaultInvoiceFooter: "Thank you for your business. Call (630) 954-0200 or email apinvoices@trenova.com with any questions.",
		},
		Shipment: &shipment.Shipment{
			ProNumber:          "SEED-SHP-005",
			BOL:                "BOL-2026-0005",
			ActualShipDate:     &pickupDate,
			ActualDeliveryDate: &deliveryDate,
			Pieces:             &pieces,
			Weight:             &weight,
			Moves: []*shipment.ShipmentMove{
				{
					Stops: []*shipment.Stop{
						{
							Type:     shipment.StopTypePickup,
							Sequence: 1,
							Location: &location.Location{
								Name:         "Dallas Warehouse",
								AddressLine1: "4500 Singleton Blvd",
								City:         "Dallas",
								State:        texas,
								PostalCode:   "75212",
							},
						},
						{
							Type:     shipment.StopTypeDelivery,
							Sequence: 2,
							Location: &location.Location{
								Name:         "Chicago Distribution Center",
								AddressLine1: "3901 S Ashland Ave",
								City:         "Chicago",
								State:        illinois,
								PostalCode:   "60609",
							},
						},
					},
				},
			},
			Commodities: []*shipment.ShipmentCommodity{
				{
					Pieces: 8,
					Weight: 40000,
					Commodity: &commodity.Commodity{
						Name:         "Steel Coils",
						Description:  "Coiled steel",
						FreightClass: commodity.FreightClass125,
					},
				},
			},
		},
	}
}

func previewInvoicePDFStorage(t *testing.T) *storagetest.MockStorageClient {
	t.Helper()

	logo := previewInvoicePDFLogo(t)
	storageClient := storagetest.NewMockStorageClient()
	storageClient.DownloadFunc = func(_ context.Context, key string) (*portstorage.DownloadResult, error) {
		require.Equal(t, "logos/invoice-preview.png", key)
		return &portstorage.DownloadResult{
			Body:        io.NopCloser(bytes.NewReader(logo)),
			ContentType: "image/png",
			Size:        int64(len(logo)),
		}, nil
	}
	return storageClient
}

func previewInvoicePDFLogo(t *testing.T) []byte {
	t.Helper()

	if logoPath := os.Getenv("INVOICE_PDF_PREVIEW_LOGO"); logoPath != "" {
		logo, err := os.ReadFile(logoPath)
		require.NoError(t, err)
		return logo
	}

	img := image.NewRGBA(image.Rect(0, 0, 320, 72))
	navy := color.RGBA{R: 8, G: 20, B: 42, A: 255}
	green := color.RGBA{R: 41, G: 168, B: 94, A: 255}
	blue := color.RGBA{R: 39, G: 92, B: 168, A: 255}
	for y := range 72 {
		for x := range 320 {
			img.Set(x, y, color.RGBA{R: 255, G: 255, B: 255, A: 255})
			if y > 24 && y < 36 && x > 8 && x < 250 {
				img.Set(x, y, navy)
			}
			if y > 38 && y < 50 && x > 8 && x < 210 {
				img.Set(x, y, blue)
			}
			if x-y > 232 && x-y < 246 && x > 238 {
				img.Set(x, y, green)
			}
		}
	}

	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

func previewUnixDate(year int, month time.Month, day int) int64 {
	return time.Date(year, month, day, 12, 0, 0, 0, time.UTC).Unix()
}
