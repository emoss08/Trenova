package transactions

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/shared/edi/internal/segments"
	"github.com/emoss08/trenova/shared/edi/internal/x12"
)

// TX204Builder builds 204 Motor Carrier Load Tender transactions
type TX204Builder struct {
	*BaseBuilder
}

// NewTX204Builder creates a new 204 builder
func NewTX204Builder(
	registry *segments.SegmentRegistry,
	version string,
	delims x12.Delimiters,
) *TX204Builder {
	return &TX204Builder{
		BaseBuilder: NewBaseBuilder(registry, version, delims),
	}
}

// GetTransactionType returns "204"
func (b *TX204Builder) GetTransactionType() string {
	return "204"
}

// GetVersion returns the X12 version
func (b *TX204Builder) GetVersion() string {
	return b.version
}

// LoadTender204 represents a 204 load tender transaction
type LoadTender204 struct {
	// Transaction header
	ControlNumber string
	Purpose       string // "00" = Original, "01" = Cancellation, etc.

	// B2 - Beginning Segment
	TariffServiceCode            string
	StandardCarrierAlphaCode     string
	ShipmentIdentificationNumber string
	ShipmentMethodOfPayment      string

	// B2A - Set Purpose
	TransactionSetPurposeCode string
	ApplicationType           string

	// Parties (N1 loops)
	Shipper   Party
	Consignee Party
	BillTo    Party

	// Stops (S5 loops)
	Stops []Stop

	// Equipment
	EquipmentType   string
	EquipmentNumber string
	EquipmentLength int
	EquipmentHeight int
	EquipmentWidth  int

	// Dates and Times
	PickupDate   time.Time
	DeliveryDate time.Time

	// Weights and Quantities
	TotalWeight  float64
	WeightUnit   string
	TotalPieces  int
	TotalPallets int

	// Special Instructions
	SpecialInstructions []string

	// Reference Numbers
	ReferenceNumbers []ReferenceNumber

	// Line Items
	LineItems []LineItem
}

// Party represents a party in the transaction (shipper, consignee, etc.)
type Party struct {
	EntityIdentifierCode string // "SH" = Shipper, "CN" = Consignee, "BT" = Bill To
	Name                 string
	IdentificationCode   string
	Address1             string
	Address2             string
	City                 string
	State                string
	PostalCode           string
	Country              string
	ContactName          string
	ContactPhone         string
	ContactEmail         string
}

// Stop represents a stop in the shipment
type Stop struct {
	StopSequenceNumber int
	StopReasonCode     string // "CL" = Complete Load, "CU" = Complete Unload
	Date               time.Time
	Time               time.Time
	LocationQualifier  string
	LocationIdentifier string
	City               string
	State              string
	PostalCode         string
	Country            string
}

// ReferenceNumber represents a reference number
type ReferenceNumber struct {
	Qualifier string // "BM" = Bill of Lading, "PO" = Purchase Order, etc.
	Number    string
}

// LineItem represents a line item in the shipment
type LineItem struct {
	LineNumber    int
	Description   string
	Quantity      float64
	Unit          string
	Weight        float64
	WeightUnit    string
	Volume        float64
	VolumeUnit    string
	CommodityCode string
	PackagingCode string
	HazmatCode    string
}

// Build constructs a 204 load tender
func (b *TX204Builder) Build(ctx context.Context, data interface{}) (string, error) {
	tender, ok := data.(*LoadTender204)
	if !ok {
		return "", fmt.Errorf("invalid data type for 204 builder")
	}

	var txContent strings.Builder

	stValues := map[string]string{
		"1": "204",
		"2": tender.ControlNumber,
	}
	stSegment, err := b.BuildSegment("ST", stValues)
	if err != nil {
		return "", fmt.Errorf("failed to build ST: %w", err)
	}
	txContent.WriteString(stSegment)
	txContent.WriteByte(b.delims.Segment)

	b2Values := map[string]string{
		"1": tender.TariffServiceCode,
		"2": tender.StandardCarrierAlphaCode,
		"3": tender.ShipmentIdentificationNumber,
		"4": tender.ShipmentMethodOfPayment,
	}
	b2Segment, err := b.BuildSegment("B2", b2Values)
	if err != nil {
		return "", fmt.Errorf("failed to build B2: %w", err)
	}
	txContent.WriteString(b2Segment)
	txContent.WriteByte(b.delims.Segment)

	if tender.TransactionSetPurposeCode != "" {
		b2aValues := map[string]string{
			"1": tender.TransactionSetPurposeCode,
			"2": tender.ApplicationType,
		}
		b2aSegment, err := b.BuildSegment("B2A", b2aValues)
		if err == nil {
			txContent.WriteString(b2aSegment)
			txContent.WriteByte(b.delims.Segment)
		}
	}

	for _, ref := range tender.ReferenceNumbers {
		l11Values := map[string]string{
			"1": ref.Number,
			"2": ref.Qualifier,
		}
		l11Segment, err := b.BuildSegment("L11", l11Values)
		if err == nil {
			txContent.WriteString(l11Segment)
			txContent.WriteByte(b.delims.Segment)
		}
	}

	if !tender.PickupDate.IsZero() {
		g62Values := map[string]string{
			"1": "10", // Requested Pick-up Date
			"2": tender.PickupDate.Format("20060102"),
			"3": "1", // Date Qualifier
		}
		g62Segment, err := b.BuildSegment("G62", g62Values)
		if err == nil {
			txContent.WriteString(g62Segment)
			txContent.WriteByte(b.delims.Segment)
		}
	}

	for _, instruction := range tender.SpecialInstructions {
		nteValues := map[string]string{
			"1": "SPH", // Special Handling
			"2": instruction,
		}
		nteSegment, err := b.BuildSegment("NTE", nteValues)
		if err == nil {
			txContent.WriteString(nteSegment)
			txContent.WriteByte(b.delims.Segment)
		}
	}

	if tender.Shipper.Name != "" {
		if err := b.buildPartyLoop(&txContent, tender.Shipper); err != nil {
			return "", fmt.Errorf("failed to build shipper: %w", err)
		}
	}

	if tender.Consignee.Name != "" {
		if err := b.buildPartyLoop(&txContent, tender.Consignee); err != nil {
			return "", fmt.Errorf("failed to build consignee: %w", err)
		}
	}

	if tender.BillTo.Name != "" {
		if err := b.buildPartyLoop(&txContent, tender.BillTo); err != nil {
			return "", fmt.Errorf("failed to build bill-to: %w", err)
		}
	}

	for _, stop := range tender.Stops {
		if err := b.buildStopLoop(&txContent, stop); err != nil {
			return "", fmt.Errorf("failed to build stop: %w", err)
		}
	}

	for _, item := range tender.LineItems {
		if err := b.buildLineItemLoop(&txContent, item); err != nil {
			return "", fmt.Errorf("failed to build line item: %w", err)
		}
	}

	segmentCount := strings.Count(txContent.String(), string(b.delims.Segment)) + 1
	seValues := map[string]string{
		"1": fmt.Sprintf("%d", segmentCount),
		"2": tender.ControlNumber,
	}
	seSegment, err := b.BuildSegment("SE", seValues)
	if err != nil {
		return "", fmt.Errorf("failed to build SE: %w", err)
	}
	txContent.WriteString(seSegment)
	txContent.WriteByte(b.delims.Segment)

	return txContent.String(), nil
}

func (b *TX204Builder) buildPartyLoop(content *strings.Builder, party Party) error {
	n1Values := map[string]string{
		"1": party.EntityIdentifierCode,
		"2": party.Name,
		"3": "93", // Code qualifier (if code provided)
		"4": party.IdentificationCode,
	}
	n1Segment, err := b.BuildSegment("N1", n1Values)
	if err != nil {
		return fmt.Errorf("failed to build N1: %w", err)
	}
	content.WriteString(n1Segment)
	content.WriteByte(b.delims.Segment)

	if party.Address1 != "" {
		n3Values := map[string]string{
			"1": party.Address1,
			"2": party.Address2,
		}
		n3Segment, err := b.BuildSegment("N3", n3Values)
		if err == nil {
			content.WriteString(n3Segment)
			content.WriteByte(b.delims.Segment)
		}
	}

	if party.City != "" {
		n4Values := map[string]string{
			"1": party.City,
			"2": party.State,
			"3": party.PostalCode,
			"4": party.Country,
		}
		n4Segment, err := b.BuildSegment("N4", n4Values)
		if err == nil {
			content.WriteString(n4Segment)
			content.WriteByte(b.delims.Segment)
		}
	}

	if party.ContactName != "" {
		g61Values := map[string]string{
			"1": "IC", // Information Contact
			"2": party.ContactName,
			"3": "TE", // Telephone
			"4": party.ContactPhone,
		}
		g61Segment, err := b.BuildSegment("G61", g61Values)
		if err == nil {
			content.WriteString(g61Segment)
			content.WriteByte(b.delims.Segment)
		}
	}

	return nil
}

// buildStopLoop builds an S5 loop for a stop
func (b *TX204Builder) buildStopLoop(content *strings.Builder, stop Stop) error {
	s5Values := map[string]string{
		"1": fmt.Sprintf("%d", stop.StopSequenceNumber),
		"2": stop.StopReasonCode,
	}
	s5Segment, err := b.BuildSegment("S5", s5Values)
	if err != nil {
		return fmt.Errorf("failed to build S5: %w", err)
	}
	content.WriteString(s5Segment)
	content.WriteByte(b.delims.Segment)

	if !stop.Date.IsZero() {
		g62Values := map[string]string{
			"1": "68", // Scheduled Delivery
			"2": stop.Date.Format("20060102"),
			"3": "1", // Date Qualifier
			"4": stop.Time.Format("1504"),
		}
		g62Segment, err := b.BuildSegment("G62", g62Values)
		if err == nil {
			content.WriteString(g62Segment)
			content.WriteByte(b.delims.Segment)
		}
	}

	if stop.LocationIdentifier != "" {
		n1Values := map[string]string{
			"1": "ST", // Ship To
			"2": "",   // Name
			"3": stop.LocationQualifier,
			"4": stop.LocationIdentifier,
		}
		n1Segment, err := b.BuildSegment("N1", n1Values)
		if err == nil {
			content.WriteString(n1Segment)
			content.WriteByte(b.delims.Segment)
		}
	}

	if stop.City != "" {
		n4Values := map[string]string{
			"1": stop.City,
			"2": stop.State,
			"3": stop.PostalCode,
			"4": stop.Country,
		}
		n4Segment, err := b.BuildSegment("N4", n4Values)
		if err == nil {
			content.WriteString(n4Segment)
			content.WriteByte(b.delims.Segment)
		}
	}

	return nil
}

func (b *TX204Builder) buildLineItemLoop(content *strings.Builder, item LineItem) error {
	oidValues := map[string]string{
		"1": "", // Reference Identification
		"2": "", // Purchase Order Number
		"3": fmt.Sprintf("%d", item.LineNumber),
		"4": "", // Release Number
		"5": "", // Change Order Sequence Number
		"6": fmt.Sprintf("%.2f", item.Quantity),
		"7": item.Unit,
		"8": fmt.Sprintf("%.2f", item.Weight),
		"9": item.WeightUnit,
	}
	oidSegment, err := b.BuildSegment("OID", oidValues)
	if err != nil {
		return fmt.Errorf("failed to build OID: %w", err)
	}
	content.WriteString(oidSegment)
	content.WriteByte(b.delims.Segment)

	// Build LAD - Lading Detail
	if item.Description != "" {
		ladValues := map[string]string{
			"1": item.PackagingCode,
			"2": fmt.Sprintf("%.0f", item.Quantity),
			"3": "", // Weight Qualifier
			"4": fmt.Sprintf("%.2f", item.Weight),
			"5": item.WeightUnit,
			"6": item.Description,
		}
		ladSegment, err := b.BuildSegment("LAD", ladValues)
		if err == nil {
			content.WriteString(ladSegment)
			content.WriteByte(b.delims.Segment)
		}
	}

	return nil
}

// Parse parses raw segments into 204 structure
func (b *TX204Builder) Parse(ctx context.Context, segments []x12.Segment) (interface{}, error) {
	// This would parse incoming 204 load tenders
	// Implementation would extract all relevant segments and build LoadTender204 structure
	return nil, fmt.Errorf("204 parsing not yet implemented")
}
