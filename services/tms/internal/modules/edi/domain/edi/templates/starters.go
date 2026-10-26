package templates

import (
	"fmt"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func StarterSegments(
	tenantInfo pagination.TenantInfo,
	versionID pulid.ID,
	transactionSet edi.TransactionSet,
) ([]*edi.EDITemplateSegment, error) {
	switch transactionSet {
	case edi.TransactionSet204:
		return Base204Segments(tenantInfo, versionID), nil
	case edi.TransactionSet210:
		return Base210Segments(tenantInfo, versionID), nil
	case edi.TransactionSet214:
		return Base214Segments(tenantInfo, versionID), nil
	case edi.TransactionSet990:
		return Base990Segments(tenantInfo, versionID), nil
	default:
		return nil, fmt.Errorf("unsupported X12 transaction set %q", transactionSet)
	}
}

//nolint:funlen // Starter templates are declarative segment definitions kept in one ordered list.
func Base210Segments(
	tenantInfo pagination.TenantInfo,
	versionID pulid.ID,
) []*edi.EDITemplateSegment {
	b := base204Builder{tenantInfo: tenantInfo, versionID: versionID}
	segments := make([]*edi.EDITemplateSegment, 0, 18)
	segments = append(segments, b.envelopeSegments()...)
	segments = append(
		segments,
		x12ST(b, edi.TransactionSet210),
		b.segment(
			40,
			"B3",
			"Beginning Segment for Carrier's Invoice",
			"",
			true,
			[]edi.TemplateElement{
				b.field(2, "Invoice Number", "invoice.invoiceNumber", "", true),
				b.field(3, "Shipment Identification Number", "invoice.shipmentId", ""),
				b.el(
					4,
					"Shipment Method of Payment",
					edi.TemplateElementSourceConstant,
					"PP",
					false,
				),
				b.date(6, "Date", "invoice.invoiceDate", true),
				b.amount(7, "Net Amount Due", "invoice.totalAmount", true),
				b.partner(11, "Standard Carrier Alpha Code", "carrier.scac"),
			},
		),
		b.segment(50, "C3", "Currency", "", false, []edi.TemplateElement{
			b.field(1, "Currency Code", "invoice.currencyCode", "USD"),
		}),
		b.segment(60, "N9", "Reference Identification", "", false, []edi.TemplateElement{
			b.field(1, "Reference Qualifier", "invoice.referenceNumbers.qualifier", "BM"),
			b.field(2, "Reference Identification", "invoice.bol", ""),
		}),
		// One reference loop per shipment on a consolidated invoice, and nothing at
		// all on a single-shipment one, where segment 60 above already carries the
		// BOL. Without it a receiver has an invoice total with no way to tell which
		// of their loads it settles.
		b.segment(
			65,
			"N9",
			"Shipment Reference",
			"invoice.shipments",
			false,
			[]edi.TemplateElement{
				b.el(1, "Reference Qualifier", edi.TemplateElementSourceConstant, "BM"),
				b.repeat(2, "Bill of Lading", "bol", ""),
				b.el(3, "Pro Number Qualifier", edi.TemplateElementSourceConstant, "CN"),
				b.repeat(4, "Pro Number", "proNumber", ""),
			},
		),
		b.segment(70, "G62", "Date Time", "", false, []edi.TemplateElement{
			b.el(1, "Date Qualifier", edi.TemplateElementSourceConstant, "86"),
			b.date(2, "Invoice Date", "invoice.invoiceDate"),
		}),
		b.segment(80, "N1", "Name", "", false, []edi.TemplateElement{
			b.el(1, "Entity Identifier Code", edi.TemplateElementSourceConstant, "BT"),
			b.field(2, "Bill-To Name", "invoice.billToName", ""),
		}),
		b.segment(90, "N3", "Address", "", false, []edi.TemplateElement{
			b.field(1, "Address", "invoice.billToAddressLine1", ""),
		}),
		b.segment(100, "N4", "Geographic Location", "", false, []edi.TemplateElement{
			b.field(1, "City", "invoice.billToCity", ""),
			b.field(2, "State", "invoice.billToStateCode", ""),
			b.field(3, "Postal Code", "invoice.billToPostalCode", ""),
		}),
		b.segment(
			110,
			"LX",
			"Transaction Set Line Number",
			"invoice.lineCharges",
			false,
			[]edi.TemplateElement{
				b.repeat(1, "Assigned Number", "sequence", "", true),
			},
		),
		b.segment(
			120,
			"L5",
			"Description Marks and Numbers",
			"invoice.lineCharges",
			false,
			[]edi.TemplateElement{
				b.repeat(1, "Lading Line Item Number", "sequence", ""),
				b.repeat(2, "Description", "description", ""),
				// Blank on a single-shipment invoice, so the segment is unchanged
				// for every 210 that is not consolidated.
				b.repeat(3, "Pro Number", "proNumber", ""),
			},
		),
		b.segment(
			130,
			"L0",
			"Line Item Quantity and Weight",
			"invoice.lineCharges",
			false,
			[]edi.TemplateElement{
				b.repeat(1, "Lading Line Item Number", "sequence", ""),
			},
		),
		b.segment(
			140,
			"L1",
			"Rate and Charges",
			"invoice.lineCharges",
			false,
			[]edi.TemplateElement{
				b.repeat(1, "Lading Line Item Number", "sequence", ""),
				b.repeatAmount(4, "Charge", "amount"),
				b.repeat(8, "Special Charge Code", "code", ""),
			},
		),
		b.segment(150, "L3", "Total Weight and Charges", "", true, []edi.TemplateElement{
			b.amount(5, "Charge", "invoice.totalAmount", true),
		}),
	)
	segments = append(segments, x12Trailers(b, 160)...)
	return segments
}

func Base214Segments(
	tenantInfo pagination.TenantInfo,
	versionID pulid.ID,
) []*edi.EDITemplateSegment {
	b := base204Builder{tenantInfo: tenantInfo, versionID: versionID}
	segments := append([]*edi.EDITemplateSegment{}, b.envelopeSegments()...)
	segments = append(
		segments,
		x12ST(b, edi.TransactionSet214),
		b.segment(
			40,
			"B10",
			"Beginning Segment for Transportation Carrier Shipment Status",
			"",
			true,
			[]edi.TemplateElement{
				b.field(1, "Reference Identification", "shipmentStatus.proNumber", ""),
				b.field(2, "Shipment Identification Number", "shipmentStatus.shipperReference", ""),
				b.partner(3, "Standard Carrier Alpha Code", "carrier.scac"),
			},
		),
		when("shipmentStatus.bol", b.segment(
			50,
			"L11",
			"Business Instructions and Reference Number",
			"",
			false,
			[]edi.TemplateElement{
				b.field(1, "Reference Identification", "shipmentStatus.bol", ""),
				b.el(
					2,
					"Reference Identification Qualifier",
					edi.TemplateElementSourceConstant,
					"BM",
					false,
				),
			},
		)),
		b.segment(60, "LX", "Transaction Set Line Number", "", true, []edi.TemplateElement{
			b.el(1, "Assigned Number", edi.TemplateElementSourceConstant, "1", true),
		}),
		b.segment(70, "AT7", "Shipment Status Details", "", true, []edi.TemplateElement{
			b.field(1, "Shipment Status Code", "shipmentStatus.statusCode", "X3", true),
			b.field(2, "Shipment Status Reason Code", "shipmentStatus.statusReasonCode", ""),
			b.field(5, "Date", "shipmentStatus.eventDate", ""),
			b.field(6, "Time", "shipmentStatus.eventTime", ""),
			b.field(7, "Time Code", "shipmentStatus.eventTimeCode", ""),
		}),
		b.segment(
			80,
			"MS1",
			"Equipment Shipment or Real Property Location",
			"",
			false,
			[]edi.TemplateElement{
				b.field(1, "City", "shipmentStatus.city", ""),
				b.field(2, "State", "shipmentStatus.stateCode", ""),
			},
		),
		b.segment(
			90,
			"MS2",
			"Equipment or Container Owner and Type",
			"",
			false,
			[]edi.TemplateElement{
				b.partner(1, "Standard Carrier Alpha Code", "carrier.scac"),
				b.field(2, "Equipment Number", "shipmentStatus.equipmentNumber", ""),
			},
		),
		when("shipmentStatus.weight", b.segment(
			100,
			"AT8",
			"Shipment Weight Packaging and Quantity Data",
			"",
			false,
			[]edi.TemplateElement{
				b.el(1, "Weight Qualifier", edi.TemplateElementSourceConstant, "G"),
				b.el(2, "Weight Unit Code", edi.TemplateElementSourceConstant, "L"),
				b.field(3, "Weight", "shipmentStatus.weight", ""),
				b.field(4, "Lading Quantity", "shipmentStatus.pieces", ""),
			},
		)),
		b.segment(110, "Q7", "Lading Exception Code", "", false, []edi.TemplateElement{
			b.field(1, "Lading Exception Code", "shipmentStatus.exceptionCode", ""),
		}),
	)
	segments = append(segments, x12Trailers(b, 120)...)
	return segments
}

func when(condition string, segment *edi.EDITemplateSegment) *edi.EDITemplateSegment {
	segment.Condition = condition

	return segment
}

func Base990Segments(
	tenantInfo pagination.TenantInfo,
	versionID pulid.ID,
) []*edi.EDITemplateSegment {
	b := base204Builder{tenantInfo: tenantInfo, versionID: versionID}
	segments := append([]*edi.EDITemplateSegment{}, b.envelopeSegments()...)
	segments = append(
		segments,
		x12ST(b, edi.TransactionSet990),
		b.segment(
			40,
			"B1",
			"Beginning Segment for Booking or Pickup Delivery",
			"",
			true,
			[]edi.TemplateElement{
				b.partner(1, "Standard Carrier Alpha Code", "carrier.scac"),
				b.field(2, "Shipment Identification Number", "tenderResponse.shipmentId", "", true),
				b.field(3, "Shipment Method of Payment", "tenderResponse.responseCode", "A", true),
			},
		),
		b.segment(50, "N9", "Reference Identification", "", false, []edi.TemplateElement{
			b.el(
				1,
				"Reference Identification Qualifier",
				edi.TemplateElementSourceConstant,
				"BM",
				false,
			),
			b.field(2, "Reference Identification", "tenderResponse.bol", ""),
		}),
		b.segment(60, "K1", "Remarks", "", false, []edi.TemplateElement{
			b.field(1, "Free-form Information", "tenderResponse.rejectionReason", ""),
		}),
	)
	segments = append(segments, x12Trailers(b, 70)...)
	return segments
}

func x12ST(
	b base204Builder,
	transactionSet edi.TransactionSet,
) *edi.EDITemplateSegment {
	return b.segment(30, "ST", "Transaction Set Header", "", true, []edi.TemplateElement{
		b.el(
			1,
			"Transaction Set Identifier",
			edi.TemplateElementSourceConstant,
			string(transactionSet),
			true,
		),
		b.runtime(2, "Transaction Control Number", "transactionControlNumber", true),
	})
}

func x12Trailers(b base204Builder, start int64) []*edi.EDITemplateSegment {
	return []*edi.EDITemplateSegment{
		b.segment(start, "SE", "Transaction Set Trailer", "", true, []edi.TemplateElement{
			b.runtime(1, "Segment Count", "transactionSegmentCount"),
			b.runtime(2, "Transaction Control Number", "transactionControlNumber", true),
		}),
		b.segment(start+10, "GE", "Functional Group Trailer", "", true, []edi.TemplateElement{
			b.el(1, "Number of Transaction Sets", edi.TemplateElementSourceConstant, "1", true),
			b.runtime(2, "Group Control Number", "groupControlNumber", true),
		}),
		b.segment(start+20, "IEA", "Interchange Control Trailer", "", true, []edi.TemplateElement{
			b.el(1, "Number of Functional Groups", edi.TemplateElementSourceConstant, "1", true),
			b.runtime(2, "Interchange Control Number", "isaControlNumber", true),
		}),
	}
}
