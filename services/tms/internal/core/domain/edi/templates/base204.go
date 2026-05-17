package templates

import (
	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func Base204Segments(
	tenantInfo pagination.TenantInfo,
	versionID pulid.ID,
) []*edi.EDITemplateSegment {
	builder := base204Builder{tenantInfo: tenantInfo, versionID: versionID}
	segments := make([]*edi.EDITemplateSegment, 0, 19)
	segments = append(segments, builder.envelopeSegments()...)
	segments = append(segments, builder.headerSegments()...)
	segments = append(segments, builder.stopSegments()...)
	segments = append(segments, builder.summarySegments()...)
	segments = append(segments, builder.trailerSegments()...)
	return segments
}

type base204Builder struct {
	tenantInfo pagination.TenantInfo
	versionID  pulid.ID
}

func (b base204Builder) segment(
	sequence int64,
	id, name, repeatPath string,
	required bool,
	elements []edi.TemplateElement,
) *edi.EDITemplateSegment {
	return &edi.EDITemplateSegment{
		BusinessUnitID:    b.tenantInfo.BuID,
		OrganizationID:    b.tenantInfo.OrgID,
		TemplateVersionID: b.versionID,
		SegmentID:         id,
		Name:              name,
		Sequence:          sequence,
		RepeatPath:        repeatPath,
		Required:          required,
		MaxUse:            1,
		Elements:          elements,
	}
}

func (base204Builder) el(
	position int,
	name string,
	source edi.TemplateElementSource,
	value string,
	required bool,
) edi.TemplateElement {
	return edi.TemplateElement{
		Position: position,
		Name:     name,
		Source:   source,
		Value:    value,
		Validation: edi.TemplateValidationRule{
			Required: required,
			Code:     "required",
			Message:  name + " is required",
		},
	}
}

func (b base204Builder) field(
	position int,
	name, path, fallback string,
	required bool,
) edi.TemplateElement {
	element := b.el(position, name, edi.TemplateElementSourceFieldPath, "", required)
	element.FieldPath = path
	element.Default = fallback
	return element
}

func (b base204Builder) partner(
	position int,
	name, path string,
	required bool,
) edi.TemplateElement {
	element := b.el(position, name, edi.TemplateElementSourcePartnerSetting, "", required)
	element.PartnerSettingPath = path
	return element
}

func (b base204Builder) repeat(
	position int,
	name, path, fallback string,
	required bool,
) edi.TemplateElement {
	element := b.el(position, name, edi.TemplateElementSourceRepeat, "", required)
	element.RepeatPath = path
	element.Default = fallback
	return element
}

func (b base204Builder) runtime(position int, name, key string, required bool) edi.TemplateElement {
	element := b.el(position, name, edi.TemplateElementSourceRuntime, "", required)
	element.RuntimeKey = key
	return element
}

func (b base204Builder) envelopeSegments() []*edi.EDITemplateSegment {
	return []*edi.EDITemplateSegment{
		b.segment(10, "ISA", "Interchange Control Header", "", true, []edi.TemplateElement{
			b.el(
				1,
				"Authorization Information Qualifier",
				edi.TemplateElementSourceConstant,
				"00",
				true,
			),
			b.el(
				2,
				"Authorization Information",
				edi.TemplateElementSourceConstant,
				"          ",
				false,
			),
			b.el(
				3,
				"Security Information Qualifier",
				edi.TemplateElementSourceConstant,
				"00",
				true,
			),
			b.el(4, "Security Information", edi.TemplateElementSourceConstant, "          ", false),
			b.el(5, "Interchange ID Qualifier", edi.TemplateElementSourceConstant, "ZZ", true),
			b.runtime(6, "Interchange Sender ID", "interchangeSenderId", true),
			b.el(7, "Interchange ID Qualifier", edi.TemplateElementSourceConstant, "ZZ", true),
			b.runtime(8, "Interchange Receiver ID", "interchangeReceiverId", true),
			b.runtime(9, "Interchange Date", "interchangeDate", true),
			b.runtime(10, "Interchange Time", "interchangeTime", true),
			b.runtime(11, "Repetition Separator", "repetitionSeparator", true),
			b.el(
				12,
				"Interchange Control Version",
				edi.TemplateElementSourceConstant,
				"00401",
				true,
			),
			b.runtime(13, "Interchange Control Number", "isaControlNumber", true),
			b.el(14, "Acknowledgment Requested", edi.TemplateElementSourceConstant, "0", true),
			b.runtime(15, "Usage Indicator", "usageIndicator", true),
			b.runtime(16, "Component Separator", "componentSeparator", true),
		}),
		b.segment(20, "GS", "Functional Group Header", "", true, []edi.TemplateElement{
			b.runtime(1, "Functional Identifier Code", "functionalGroupId", true),
			b.runtime(2, "Application Sender Code", "applicationSenderCode", true),
			b.runtime(3, "Application Receiver Code", "applicationReceiverCode", true),
			b.runtime(4, "Group Date", "groupDate", true),
			b.runtime(5, "Group Time", "groupTime", true),
			b.runtime(6, "Group Control Number", "groupControlNumber", true),
			b.el(7, "Responsible Agency Code", edi.TemplateElementSourceConstant, "X", true),
			b.runtime(8, "Version", "x12Version", true),
		}),
	}
}

func (b base204Builder) headerSegments() []*edi.EDITemplateSegment {
	return []*edi.EDITemplateSegment{
		b.segment(30, "ST", "Transaction Set Header", "", true, []edi.TemplateElement{
			b.el(1, "Transaction Set Identifier", edi.TemplateElementSourceConstant, "204", true),
			b.runtime(2, "Transaction Control Number", "transactionControlNumber", true),
		}),
		b.segment(
			40,
			"B2",
			"Beginning Segment for Shipment Information",
			"",
			true,
			[]edi.TemplateElement{
				b.partner(1, "Standard Carrier Alpha Code", "carrier.scac", false),
				b.field(2, "Shipment Identification Number", "shipmentId", "", true),
				b.field(4, "Shipment Method of Payment", "ratingDetail.paymentMethod", "PP", false),
			},
		),
		b.segment(50, "B2A", "Set Purpose", "", true, []edi.TemplateElement{
			b.el(1, "Transaction Set Purpose Code", edi.TemplateElementSourceConstant, "00", true),
		}),
		b.segment(60, "L11", "Reference Identification", "", false, []edi.TemplateElement{
			b.field(1, "Reference Identification", "bol", "", false),
			b.el(
				2,
				"Reference Identification Qualifier",
				edi.TemplateElementSourceConstant,
				"BM",
				false,
			),
		}),
		b.segment(70, "G62", "Date Time", "moves.0.stops", false, []edi.TemplateElement{
			b.el(1, "Date Qualifier", edi.TemplateElementSourceConstant, "37", false),
			b.repeat(2, "Date", "scheduledWindowStart", "", false),
			b.el(3, "Time Qualifier", edi.TemplateElementSourceConstant, "I", false),
			b.repeat(4, "Time", "scheduledWindowStart", "", false),
		}),
		b.segment(80, "NTE", "Note", "", false, []edi.TemplateElement{
			b.el(1, "Note Reference Code", edi.TemplateElementSourceConstant, "ADD", false),
			b.field(2, "Description", "ratingDetail.note", "", false),
		}),
	}
}

func (b base204Builder) stopSegments() []*edi.EDITemplateSegment {
	return []*edi.EDITemplateSegment{
		b.segment(90, "N1", "Name", "moves.0.stops", false, []edi.TemplateElement{
			b.repeat(1, "Entity Identifier Code", "type", "SF", false),
			b.repeat(2, "Name", "locationName", "", false),
		}),
		b.segment(100, "N3", "Address", "moves.0.stops", false, []edi.TemplateElement{
			b.repeat(1, "Address Information", "locationAddressLine1", "", false),
			b.repeat(2, "Address Information", "locationAddressLine2", "", false),
		}),
		b.segment(110, "N4", "Geographic Location", "moves.0.stops", false, []edi.TemplateElement{
			b.repeat(1, "City Name", "locationCity", "", false),
			b.repeat(2, "State or Province Code", "locationStateCode", "", false),
			b.repeat(3, "Postal Code", "locationPostalCode", "", false),
		}),
		b.segment(120, "G61", "Contact", "", false, []edi.TemplateElement{
			b.el(1, "Contact Function Code", edi.TemplateElementSourceConstant, "IC", false),
			b.partner(2, "Name", "contact.name", false),
			b.el(
				3,
				"Communication Number Qualifier",
				edi.TemplateElementSourceConstant,
				"TE",
				false,
			),
			b.partner(4, "Communication Number", "contact.phone", false),
		}),
		b.segment(130, "S5", "Stop Off Details", "moves.0.stops", true, []edi.TemplateElement{
			b.repeat(1, "Stop Sequence Number", "sequence", "", true),
			b.repeat(2, "Stop Reason Code", "type", "LD", true),
			b.repeat(3, "Weight", "weight", "", false),
			b.el(4, "Weight Unit Code", edi.TemplateElementSourceConstant, "L", false),
			b.repeat(5, "Number of Units Shipped", "pieces", "", false),
			b.el(
				6,
				"Unit or Basis for Measurement Code",
				edi.TemplateElementSourceConstant,
				"PCS",
				false,
			),
		}),
	}
}

func (b base204Builder) summarySegments() []*edi.EDITemplateSegment {
	return []*edi.EDITemplateSegment{
		b.segment(
			140,
			"AT8",
			"Shipment Weight Packaging and Quantity Data",
			"",
			false,
			[]edi.TemplateElement{
				b.el(1, "Weight Qualifier", edi.TemplateElementSourceConstant, "G", false),
				b.el(2, "Weight Unit Code", edi.TemplateElementSourceConstant, "L", false),
				b.field(3, "Weight", "weight", "", false),
				b.field(4, "Lading Quantity", "pieces", "", false),
			},
		),
		b.segment(
			150,
			"L5",
			"Description Marks and Numbers",
			"commodities",
			false,
			[]edi.TemplateElement{
				b.repeat(1, "Lading Line Item Number", "sequence", "", false),
				b.repeat(2, "Lading Description", "commodityDescription", "", false),
			},
		),
		b.segment(160, "L3", "Total Weight and Charges", "", false, []edi.TemplateElement{
			b.field(1, "Weight", "weight", "", false),
			b.el(2, "Weight Qualifier", edi.TemplateElementSourceConstant, "G", false),
			b.field(5, "Charge", "totalChargeAmount", "", false),
		}),
	}
}

func (b base204Builder) trailerSegments() []*edi.EDITemplateSegment {
	return []*edi.EDITemplateSegment{
		b.segment(170, "SE", "Transaction Set Trailer", "", true, []edi.TemplateElement{
			b.runtime(1, "Segment Count", "transactionSegmentCount", false),
			b.runtime(2, "Transaction Control Number", "transactionControlNumber", true),
		}),
		b.segment(180, "GE", "Functional Group Trailer", "", true, []edi.TemplateElement{
			b.el(1, "Number of Transaction Sets", edi.TemplateElementSourceConstant, "1", true),
			b.runtime(2, "Group Control Number", "groupControlNumber", true),
		}),
		b.segment(190, "IEA", "Interchange Control Trailer", "", true, []edi.TemplateElement{
			b.el(1, "Number of Functional Groups", edi.TemplateElementSourceConstant, "1", true),
			b.runtime(2, "Interchange Control Number", "isaControlNumber", true),
		}),
	}
}
