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

const transaction204Segments: Record<string, X12SegmentDefinition> = {
  B2: {
    id: "B2",
    label: "Beginning Segment for Shipment Information",
    elements: [
      { position: 1, label: "Standard Carrier Alpha Code" },
      { position: 2, label: "Shipment Identification Number", required: true },
      { position: 3, label: "Shipment Method of Payment" },
      { position: 4, label: "Shipment Method of Payment" },
    ],
  },
  B2A: {
    id: "B2A",
    label: "Set Purpose",
    elements: [{ position: 1, label: "Transaction Set Purpose Code", required: true }],
  },
  L11: {
    id: "L11",
    label: "Reference Identification",
    elements: [
      { position: 1, label: "Reference Identification" },
      { position: 2, label: "Reference Identification Qualifier" },
    ],
  },
  G62: {
    id: "G62",
    label: "Date Time",
    elements: [
      { position: 1, label: "Date Qualifier" },
      { position: 2, label: "Date" },
      { position: 3, label: "Time Qualifier" },
      { position: 4, label: "Time" },
    ],
  },
  NTE: {
    id: "NTE",
    label: "Note",
    elements: [
      { position: 1, label: "Note Reference Code" },
      { position: 2, label: "Description" },
    ],
  },
  N1: {
    id: "N1",
    label: "Name",
    elements: [
      { position: 1, label: "Entity Identifier Code" },
      { position: 2, label: "Name" },
    ],
  },
  N3: {
    id: "N3",
    label: "Address",
    elements: [
      { position: 1, label: "Address Information" },
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
    ],
  },
  G61: {
    id: "G61",
    label: "Contact",
    elements: [
      { position: 1, label: "Contact Function Code" },
      { position: 2, label: "Name" },
      { position: 3, label: "Communication Number Qualifier" },
      { position: 4, label: "Communication Number" },
    ],
  },
  S5: {
    id: "S5",
    label: "Stop Off Details",
    elements: [
      { position: 1, label: "Stop Sequence Number", required: true },
      { position: 2, label: "Stop Reason Code", required: true },
      { position: 3, label: "Weight" },
      { position: 4, label: "Weight Unit Code" },
      { position: 5, label: "Number of Units Shipped" },
      { position: 6, label: "Unit or Basis for Measurement Code" },
    ],
  },
  AT8: {
    id: "AT8",
    label: "Shipment Weight Packaging and Quantity Data",
    elements: [
      { position: 1, label: "Weight Qualifier" },
      { position: 2, label: "Weight Unit Code" },
      { position: 3, label: "Weight" },
      { position: 4, label: "Lading Quantity" },
    ],
  },
  L5: {
    id: "L5",
    label: "Description Marks and Numbers",
    elements: [
      { position: 1, label: "Lading Line Item Number" },
      { position: 2, label: "Lading Description" },
    ],
  },
  L3: {
    id: "L3",
    label: "Total Weight and Charges",
    elements: [
      { position: 1, label: "Weight" },
      { position: 2, label: "Weight Qualifier" },
      { position: 5, label: "Charge" },
    ],
  },
};

export const x12Dictionary: Record<string, X12SegmentDefinition> = {
  ...controlSegments,
  ...transaction204Segments,
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
