export type X12ElementDefinition = {
  position: number;
  label: string;
  required?: boolean;
};

export type X12SegmentDefinition = {
  id: string;
  label: string;
  control?: boolean;
  elements: X12ElementDefinition[];
};

const controlSegments: Record<string, X12SegmentDefinition> = {
  ISA: {
    id: "ISA",
    label: "Interchange Control Header",
    control: true,
    elements: [
      { position: 1, label: "Authorization Information Qualifier", required: true },
      { position: 2, label: "Authorization Information" },
      { position: 3, label: "Security Information Qualifier", required: true },
      { position: 4, label: "Security Information" },
      { position: 5, label: "Interchange ID Qualifier", required: true },
      { position: 6, label: "Interchange Sender ID", required: true },
      { position: 7, label: "Interchange ID Qualifier", required: true },
      { position: 8, label: "Interchange Receiver ID", required: true },
      { position: 9, label: "Interchange Date", required: true },
      { position: 10, label: "Interchange Time", required: true },
      { position: 11, label: "Repetition Separator", required: true },
      { position: 12, label: "Interchange Control Version", required: true },
      { position: 13, label: "Interchange Control Number", required: true },
      { position: 14, label: "Acknowledgment Requested", required: true },
      { position: 15, label: "Usage Indicator", required: true },
      { position: 16, label: "Component Element Separator", required: true },
    ],
  },
  GS: {
    id: "GS",
    label: "Functional Group Header",
    control: true,
    elements: [
      { position: 1, label: "Functional Identifier Code", required: true },
      { position: 2, label: "Application Sender Code", required: true },
      { position: 3, label: "Application Receiver Code", required: true },
      { position: 4, label: "Group Date", required: true },
      { position: 5, label: "Group Time", required: true },
      { position: 6, label: "Group Control Number", required: true },
      { position: 7, label: "Responsible Agency Code", required: true },
      { position: 8, label: "Version", required: true },
    ],
  },
  ST: {
    id: "ST",
    label: "Transaction Set Header",
    control: true,
    elements: [
      { position: 1, label: "Transaction Set Identifier", required: true },
      { position: 2, label: "Transaction Control Number", required: true },
    ],
  },
  SE: {
    id: "SE",
    label: "Transaction Set Trailer",
    control: true,
    elements: [
      { position: 1, label: "Segment Count", required: true },
      { position: 2, label: "Transaction Control Number", required: true },
    ],
  },
  GE: {
    id: "GE",
    label: "Functional Group Trailer",
    control: true,
    elements: [
      { position: 1, label: "Number of Transaction Sets", required: true },
      { position: 2, label: "Group Control Number", required: true },
    ],
  },
  IEA: {
    id: "IEA",
    label: "Interchange Control Trailer",
    control: true,
    elements: [
      { position: 1, label: "Number of Functional Groups", required: true },
      { position: 2, label: "Interchange Control Number", required: true },
    ],
  },
};

const commonSegments: Record<string, X12SegmentDefinition> = {
  N1: {
    id: "N1",
    label: "Name",
    elements: [
      { position: 1, label: "Entity Identifier Code", required: true },
      { position: 2, label: "Name" },
      { position: 3, label: "Identification Code Qualifier" },
      { position: 4, label: "Identification Code" },
    ],
  },
  N2: {
    id: "N2",
    label: "Additional Name Information",
    elements: [
      { position: 1, label: "Name", required: true },
      { position: 2, label: "Name" },
    ],
  },
  N3: {
    id: "N3",
    label: "Address Information",
    elements: [
      { position: 1, label: "Address Information", required: true },
      { position: 2, label: "Address Information" },
    ],
  },
  N4: {
    id: "N4",
    label: "Geographic Location",
    elements: [
      { position: 1, label: "City Name" },
      { position: 2, label: "State or Province Code" },
      { position: 3, label: "Postal Code" },
      { position: 4, label: "Country Code" },
    ],
  },
  N9: {
    id: "N9",
    label: "Reference Identification",
    elements: [
      { position: 1, label: "Reference Identification Qualifier", required: true },
      { position: 2, label: "Reference Identification" },
      { position: 3, label: "Free-form Description" },
      { position: 4, label: "Date" },
      { position: 5, label: "Time" },
    ],
  },
  L11: {
    id: "L11",
    label: "Business Instructions and Reference Number",
    elements: [
      { position: 1, label: "Reference Identification" },
      { position: 2, label: "Reference Identification Qualifier" },
      { position: 3, label: "Description" },
    ],
  },
  G61: {
    id: "G61",
    label: "Contact",
    elements: [
      { position: 1, label: "Contact Function Code", required: true },
      { position: 2, label: "Name", required: true },
      { position: 3, label: "Communication Number Qualifier" },
      { position: 4, label: "Communication Number" },
    ],
  },
  G62: {
    id: "G62",
    label: "Date/Time",
    elements: [
      { position: 1, label: "Date Qualifier" },
      { position: 2, label: "Date" },
      { position: 3, label: "Time Qualifier" },
      { position: 4, label: "Time" },
      { position: 5, label: "Time Code" },
    ],
  },
  NTE: {
    id: "NTE",
    label: "Note / Special Instruction",
    elements: [
      { position: 1, label: "Note Reference Code" },
      { position: 2, label: "Description", required: true },
    ],
  },
  LX: {
    id: "LX",
    label: "Transaction Set Line Number",
    elements: [{ position: 1, label: "Assigned Number", required: true }],
  },
  L5: {
    id: "L5",
    label: "Description, Marks and Numbers",
    elements: [
      { position: 1, label: "Lading Line Item Number" },
      { position: 2, label: "Lading Description" },
      { position: 3, label: "Commodity Code" },
      { position: 4, label: "Commodity Code Qualifier" },
    ],
  },
  L3: {
    id: "L3",
    label: "Total Weight and Charges",
    elements: [
      { position: 1, label: "Weight" },
      { position: 2, label: "Weight Qualifier" },
      { position: 3, label: "Freight Rate" },
      { position: 4, label: "Rate/Value Qualifier" },
      { position: 5, label: "Charge" },
      { position: 11, label: "Lading Quantity" },
    ],
  },
  AT8: {
    id: "AT8",
    label: "Shipment Weight, Packaging and Quantity Data",
    elements: [
      { position: 1, label: "Weight Qualifier" },
      { position: 2, label: "Weight Unit Code" },
      { position: 3, label: "Weight" },
      { position: 4, label: "Lading Quantity" },
    ],
  },
};

const shipmentTender204Segments: Record<string, X12SegmentDefinition> = {
  B2: {
    id: "B2",
    label: "Beginning Segment for Shipment Information",
    elements: [
      { position: 1, label: "Standard Carrier Alpha Code" },
      { position: 2, label: "Shipment Identification Number", required: true },
      { position: 3, label: "Standard Point Location Code" },
      { position: 4, label: "Shipment Method of Payment" },
    ],
  },
  B2A: {
    id: "B2A",
    label: "Set Purpose",
    elements: [
      { position: 1, label: "Transaction Set Purpose Code", required: true },
      { position: 2, label: "Application Type" },
    ],
  },
  S5: {
    id: "S5",
    label: "Stop-off Details",
    elements: [
      { position: 1, label: "Stop Sequence Number", required: true },
      { position: 2, label: "Stop Reason Code", required: true },
      { position: 3, label: "Weight" },
      { position: 4, label: "Weight Unit Code" },
      { position: 5, label: "Number of Units Shipped" },
      { position: 6, label: "Unit or Basis for Measurement Code" },
    ],
  },
};

const tenderResponse990Segments: Record<string, X12SegmentDefinition> = {
  B1: {
    id: "B1",
    label: "Beginning Segment for Booking or Pick-up/Delivery",
    elements: [
      { position: 1, label: "Standard Carrier Alpha Code" },
      { position: 2, label: "Shipment Identification Number", required: true },
      { position: 3, label: "Shipment Method of Payment", required: true },
      { position: 4, label: "Reservation Action Code" },
    ],
  },
  K1: {
    id: "K1",
    label: "Remarks",
    elements: [
      { position: 1, label: "Free-form Information", required: true },
      { position: 2, label: "Free-form Information" },
    ],
  },
};

const carrierInvoice210Segments: Record<string, X12SegmentDefinition> = {
  B3: {
    id: "B3",
    label: "Beginning Segment for Carrier's Invoice",
    elements: [
      { position: 1, label: "Shipment Qualifier" },
      { position: 2, label: "Invoice Number", required: true },
      { position: 3, label: "Shipment Identification Number" },
      { position: 4, label: "Shipment Method of Payment", required: true },
      { position: 5, label: "Weight Unit Code" },
      { position: 6, label: "Net Amount Due", required: true },
      { position: 7, label: "Correction Indicator" },
      { position: 10, label: "Standard Carrier Alpha Code" },
      { position: 11, label: "Currency Code" },
    ],
  },
  C3: {
    id: "C3",
    label: "Currency",
    elements: [
      { position: 1, label: "Currency Code", required: true },
      { position: 2, label: "Exchange Rate" },
      { position: 3, label: "Currency Code" },
    ],
  },
  L0: {
    id: "L0",
    label: "Line Item - Quantity and Weight",
    elements: [
      { position: 1, label: "Lading Line Item Number" },
      { position: 2, label: "Billed/Rated-as Quantity" },
      { position: 3, label: "Billed/Rated-as Qualifier" },
      { position: 4, label: "Weight" },
      { position: 5, label: "Weight Qualifier" },
      { position: 6, label: "Lading Quantity" },
      { position: 7, label: "Packaging Form Code" },
      { position: 8, label: "Dunnage Description" },
      { position: 9, label: "Volume" },
      { position: 10, label: "Volume Unit Qualifier" },
      { position: 11, label: "Lading Quantity" },
    ],
  },
  L1: {
    id: "L1",
    label: "Rate and Charges",
    elements: [
      { position: 1, label: "Lading Line Item Number" },
      { position: 2, label: "Freight Rate" },
      { position: 3, label: "Rate/Value Qualifier" },
      { position: 4, label: "Charge" },
      { position: 5, label: "Advances" },
      { position: 6, label: "Prepaid Amount" },
      { position: 7, label: "Rate Combination Point Code" },
      { position: 8, label: "Special Charge or Allowance Code" },
    ],
  },
};

const shipmentStatus214Segments: Record<string, X12SegmentDefinition> = {
  B10: {
    id: "B10",
    label: "Beginning Segment for Transportation Carrier Shipment Status Message",
    elements: [
      { position: 1, label: "Reference Identification" },
      { position: 2, label: "Shipment Identification Number" },
      { position: 3, label: "Standard Carrier Alpha Code", required: true },
      { position: 4, label: "Inquiry Request Number" },
    ],
  },
  AT7: {
    id: "AT7",
    label: "Shipment Status Details",
    elements: [
      { position: 1, label: "Shipment Status Code" },
      { position: 2, label: "Shipment Status Reason Code" },
      { position: 3, label: "Shipment Appointment Status Code" },
      { position: 4, label: "Shipment Status Reason Code" },
      { position: 5, label: "Date" },
      { position: 6, label: "Time" },
      { position: 7, label: "Time Code" },
    ],
  },
  MS1: {
    id: "MS1",
    label: "Equipment, Shipment, or Real Property Location",
    elements: [
      { position: 1, label: "City Name" },
      { position: 2, label: "State or Province Code" },
      { position: 3, label: "Country Code" },
    ],
  },
  MS2: {
    id: "MS2",
    label: "Equipment or Container Owner and Type",
    elements: [
      { position: 1, label: "Standard Carrier Alpha Code" },
      { position: 2, label: "Equipment Number" },
      { position: 3, label: "Equipment Description Code" },
    ],
  },
  Q7: {
    id: "Q7",
    label: "Lading Exception Code",
    elements: [
      { position: 1, label: "Lading Exception Code", required: true },
      { position: 2, label: "Packaging Form Code" },
      { position: 3, label: "Lading Quantity" },
    ],
  },
};

const acknowledgmentSegments: Record<string, X12SegmentDefinition> = {
  AK1: {
    id: "AK1",
    label: "Functional Group Response Header",
    elements: [
      { position: 1, label: "Functional Identifier Code", required: true },
      { position: 2, label: "Group Control Number", required: true },
    ],
  },
  AK2: {
    id: "AK2",
    label: "Transaction Set Response Header",
    elements: [
      { position: 1, label: "Transaction Set Identifier Code", required: true },
      { position: 2, label: "Transaction Set Control Number", required: true },
    ],
  },
  AK3: {
    id: "AK3",
    label: "Data Segment Note",
    elements: [
      { position: 1, label: "Segment ID Code", required: true },
      { position: 2, label: "Segment Position in Transaction Set", required: true },
      { position: 3, label: "Loop Identifier Code" },
      { position: 4, label: "Segment Syntax Error Code" },
    ],
  },
  AK4: {
    id: "AK4",
    label: "Data Element Note",
    elements: [
      { position: 1, label: "Position in Segment", required: true },
      { position: 2, label: "Data Element Reference Number" },
      { position: 3, label: "Data Element Syntax Error Code" },
      { position: 4, label: "Copy of Bad Data Element" },
    ],
  },
  AK5: {
    id: "AK5",
    label: "Transaction Set Response Trailer",
    elements: [
      { position: 1, label: "Transaction Set Acknowledgment Code", required: true },
      { position: 2, label: "Transaction Set Syntax Error Code" },
      { position: 3, label: "Transaction Set Syntax Error Code" },
      { position: 4, label: "Transaction Set Syntax Error Code" },
      { position: 5, label: "Transaction Set Syntax Error Code" },
      { position: 6, label: "Transaction Set Syntax Error Code" },
    ],
  },
  AK9: {
    id: "AK9",
    label: "Functional Group Response Trailer",
    elements: [
      { position: 1, label: "Functional Group Acknowledge Code", required: true },
      { position: 2, label: "Number of Transaction Sets Included", required: true },
      { position: 3, label: "Number of Received Transaction Sets", required: true },
      { position: 4, label: "Number of Accepted Transaction Sets", required: true },
    ],
  },
  IK3: {
    id: "IK3",
    label: "Error Identification",
    elements: [
      { position: 1, label: "Segment ID Code", required: true },
      { position: 2, label: "Segment Position in Transaction Set", required: true },
      { position: 3, label: "Loop Identifier Code" },
      { position: 4, label: "Implementation Segment Syntax Error Code" },
    ],
  },
  IK4: {
    id: "IK4",
    label: "Implementation Data Element Note",
    elements: [
      { position: 1, label: "Position in Segment", required: true },
      { position: 2, label: "Data Element Reference Number" },
      { position: 3, label: "Implementation Data Element Syntax Error Code" },
      { position: 4, label: "Copy of Bad Data Element" },
    ],
  },
  IK5: {
    id: "IK5",
    label: "Implementation Transaction Set Response Trailer",
    elements: [
      { position: 1, label: "Transaction Set Acknowledgment Code", required: true },
      { position: 2, label: "Implementation Transaction Set Syntax Error Code" },
      { position: 3, label: "Implementation Transaction Set Syntax Error Code" },
      { position: 4, label: "Implementation Transaction Set Syntax Error Code" },
      { position: 5, label: "Implementation Transaction Set Syntax Error Code" },
      { position: 6, label: "Implementation Transaction Set Syntax Error Code" },
    ],
  },
  CTX: {
    id: "CTX",
    label: "Context",
    elements: [
      { position: 1, label: "Context Identification", required: true },
      { position: 2, label: "Segment ID Code" },
      { position: 3, label: "Segment Position in Transaction Set" },
      { position: 4, label: "Loop Identifier Code" },
      { position: 5, label: "Position in Segment" },
      { position: 6, label: "Reference in Segment" },
    ],
  },
};

export const x12Dictionary: Record<string, X12SegmentDefinition> = {
  ...controlSegments,
  ...commonSegments,
  ...shipmentTender204Segments,
  ...tenderResponse990Segments,
  ...carrierInvoice210Segments,
  ...shipmentStatus214Segments,
  ...acknowledgmentSegments,
};

export function getSegmentDefinition(segmentId: string) {
  return x12Dictionary[segmentId];
}

export function getSegmentLabel(segmentId: string) {
  return getSegmentDefinition(segmentId)?.label ?? "Unknown segment";
}

export function getElementLabel(segmentId: string, position: number) {
  return (
    getSegmentDefinition(segmentId)?.elements.find((element) => element.position === position)
      ?.label ?? `Element ${String(position).padStart(2, "0")}`
  );
}

export function getElementRequirement(segmentId: string, position: number) {
  return getSegmentDefinition(segmentId)?.elements.find((element) => element.position === position)
    ?.required;
}

export function isControlSegment(segmentId: string) {
  return getSegmentDefinition(segmentId)?.control === true;
}
