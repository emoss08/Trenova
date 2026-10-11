package edix12inspect

import "fmt"

type segmentMetadata struct {
	Name     string
	Type     string
	Loop     string
	Elements map[int]elementMetadata
}

type elementMetadata struct {
	Label    string
	Required bool
}

var x12Dictionary = map[string]segmentMetadata{
	"ISA": {
		Name: "Interchange Control Header",
		Type: "interchange",
		Elements: map[int]elementMetadata{
			1:  {Label: "Authorization Information Qualifier", Required: true},
			2:  {Label: "Authorization Information", Required: true},
			3:  {Label: "Security Information Qualifier", Required: true},
			4:  {Label: "Security Information", Required: true},
			5:  {Label: "Interchange ID Qualifier", Required: true},
			6:  {Label: "Interchange Sender ID", Required: true},
			7:  {Label: "Interchange ID Qualifier", Required: true},
			8:  {Label: "Interchange Receiver ID", Required: true},
			9:  {Label: "Interchange Date", Required: true},
			10: {Label: "Interchange Time", Required: true},
			11: {Label: "Repetition Separator", Required: true},
			12: {Label: "Interchange Control Version Number", Required: true},
			13: {Label: "Interchange Control Number", Required: true},
			14: {Label: "Acknowledgment Requested", Required: true},
			15: {Label: "Usage Indicator", Required: true},
			16: {Label: "Component Element Separator", Required: true},
		},
	},
	"GS": {
		Name: "Functional Group Header",
		Type: "group",
		Elements: map[int]elementMetadata{
			1: {Label: "Functional Identifier Code", Required: true},
			2: {Label: "Application Sender Code", Required: true},
			3: {Label: "Application Receiver Code", Required: true},
			4: {Label: "Date", Required: true},
			5: {Label: "Time", Required: true},
			6: {Label: "Group Control Number", Required: true},
			7: {Label: "Responsible Agency Code", Required: true},
			8: {Label: "Version / Release / Industry Identifier Code", Required: true},
		},
	},
	"ST": {
		Name: "Transaction Set Header",
		Type: "transaction",
		Elements: map[int]elementMetadata{
			1: {Label: "Transaction Set Identifier Code", Required: true},
			2: {Label: "Transaction Set Control Number", Required: true},
		},
	},
	"B2": {
		Name: "Beginning Segment for Shipment Information",
		Type: "detail",
		Elements: map[int]elementMetadata{
			1: {Label: "Tariff Service Code"},
			2: {Label: "Standard Carrier Alpha Code"},
			3: {Label: "Standard Point Location Code"},
			4: {Label: "Shipment Identification Number"},
			5: {Label: "Weight Unit Code"},
			6: {Label: "Shipment Method of Payment", Required: true},
		},
	},
	"B2A": {
		Name: "Set Purpose",
		Type: "detail",
		Elements: map[int]elementMetadata{
			1: {Label: "Transaction Set Purpose Code", Required: true},
			2: {Label: "Application Type"},
		},
	},
	"L11": {
		Name: "Business Instructions and Reference Number",
		Type: "detail",
		Elements: map[int]elementMetadata{
			1: {Label: "Reference Identification"},
			2: {Label: "Reference Identification Qualifier"},
			3: {Label: "Description"},
		},
	},
	"G62": {
		Name: "Date / Time",
		Type: "detail",
		Elements: map[int]elementMetadata{
			1: {Label: "Date Qualifier"},
			2: {Label: "Date"},
			3: {Label: "Time Qualifier"},
			4: {Label: "Time"},
		},
	},
	"N1": {
		Name: "Name",
		Type: "loop",
		Loop: "N1",
		Elements: map[int]elementMetadata{
			1: {Label: "Entity Identifier Code", Required: true},
			2: {Label: "Name"},
			3: {Label: "Identification Code Qualifier"},
			4: {Label: "Identification Code"},
		},
	},
	"N3": {
		Name: "Address Information",
		Type: "loop",
		Loop: "N1",
		Elements: map[int]elementMetadata{
			1: {Label: "Address Information", Required: true},
			2: {Label: "Address Information"},
		},
	},
	"N4": {
		Name: "Geographic Location",
		Type: "loop",
		Loop: "N1",
		Elements: map[int]elementMetadata{
			1: {Label: "City Name"},
			2: {Label: "State or Province Code"},
			3: {Label: "Postal Code"},
			4: {Label: "Country Code"},
		},
	},
	"S5": {
		Name: "Stop-off Details",
		Type: "loop",
		Loop: "S5",
		Elements: map[int]elementMetadata{
			1: {Label: "Stop Sequence Number", Required: true},
			2: {Label: "Stop Reason Code", Required: true},
		},
	},
	"L5": {
		Name: "Description, Marks and Numbers",
		Type: "loop",
		Loop: "S5",
		Elements: map[int]elementMetadata{
			1: {Label: "Lading Line Item Number"},
			2: {Label: "Lading Description"},
			3: {Label: "Commodity Code"},
			4: {Label: "Commodity Code Qualifier"},
		},
	},
	"L3": {
		Name: "Total Weight and Charges",
		Type: "summary",
		Elements: map[int]elementMetadata{
			1:  {Label: "Weight"},
			2:  {Label: "Weight Qualifier"},
			5:  {Label: "Charge"},
			11: {Label: "Lading Quantity"},
		},
	},
	"SE": {
		Name: "Transaction Set Trailer",
		Type: "transaction",
		Elements: map[int]elementMetadata{
			1: {Label: "Number of Included Segments", Required: true},
			2: {Label: "Transaction Set Control Number", Required: true},
		},
	},
	"GE": {
		Name: "Functional Group Trailer",
		Type: "group",
		Elements: map[int]elementMetadata{
			1: {Label: "Number of Transaction Sets Included", Required: true},
			2: {Label: "Group Control Number", Required: true},
		},
	},
	"IEA": {
		Name: "Interchange Control Trailer",
		Type: "interchange",
		Elements: map[int]elementMetadata{
			1: {Label: "Number of Included Functional Groups", Required: true},
			2: {Label: "Interchange Control Number", Required: true},
		},
	},
}

func segmentName(segmentID string) string {
	if metadata, ok := x12Dictionary[segmentID]; ok {
		return metadata.Name
	}
	return "Unknown Segment"
}

func segmentType(segmentID string) string {
	if metadata, ok := x12Dictionary[segmentID]; ok {
		return metadata.Type
	}
	return "detail"
}

func segmentLoop(segmentID string) string {
	if metadata, ok := x12Dictionary[segmentID]; ok {
		return metadata.Loop
	}
	return ""
}

func elementLabel(segmentID string, position int) (string, bool) {
	metadata, ok := x12Dictionary[segmentID]
	if !ok {
		return fallbackElementLabel(position), false
	}
	element, ok := metadata.Elements[position]
	if !ok {
		return fallbackElementLabel(position), false
	}
	return element.Label, true
}

func elementRequired(segmentID string, position int) bool {
	metadata, ok := x12Dictionary[segmentID]
	if !ok {
		return false
	}
	return metadata.Elements[position].Required
}

func fallbackElementLabel(position int) string {
	return fmt.Sprintf("Element %02d", position)
}
