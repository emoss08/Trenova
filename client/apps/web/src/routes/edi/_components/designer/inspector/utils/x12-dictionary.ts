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
    label: "Interchange control header",
    control: true,
    elements: [
      { position: 1, label: "Authorization information qualifier", required: true },
      { position: 2, label: "Authorization information" },
      { position: 3, label: "Security information qualifier", required: true },
      { position: 4, label: "Security information" },
      { position: 5, label: "Interchange ID qualifier", required: true },
      { position: 6, label: "Interchange sender ID", required: true },
      { position: 7, label: "Interchange ID qualifier", required: true },
      { position: 8, label: "Interchange receiver ID", required: true },
      { position: 9, label: "Interchange date", required: true },
      { position: 10, label: "Interchange time", required: true },
      { position: 11, label: "Repetition separator", required: true },
      { position: 12, label: "Interchange control version", required: true },
      { position: 13, label: "Interchange control number", required: true },
      { position: 14, label: "Acknowledgment requested", required: true },
      { position: 15, label: "Usage indicator", required: true },
      { position: 16, label: "Component element separator", required: true },
    ],
  },
  GS: {
    id: "GS",
    label: "Functional group header",
    control: true,
    elements: [
      { position: 1, label: "Functional identifier code", required: true },
      { position: 2, label: "Application sender code", required: true },
      { position: 3, label: "Application receiver code", required: true },
      { position: 4, label: "Group date", required: true },
      { position: 5, label: "Group time", required: true },
      { position: 6, label: "Group control number", required: true },
      { position: 7, label: "Responsible agency code", required: true },
      { position: 8, label: "Version", required: true },
    ],
  },
  ST: {
    id: "ST",
    label: "Transaction set header",
    control: true,
    elements: [
      { position: 1, label: "Transaction set identifier", required: true },
      { position: 2, label: "Transaction control number", required: true },
    ],
  },
  SE: {
    id: "SE",
    label: "Transaction set trailer",
    control: true,
    elements: [
      { position: 1, label: "Segment count", required: true },
      { position: 2, label: "Transaction control number", required: true },
    ],
  },
  GE: {
    id: "GE",
    label: "Functional group trailer",
    control: true,
    elements: [
      { position: 1, label: "Number of transaction sets", required: true },
      { position: 2, label: "Group control number", required: true },
    ],
  },
  IEA: {
    id: "IEA",
    label: "Interchange control trailer",
    control: true,
    elements: [
      { position: 1, label: "Number of functional groups", required: true },
      { position: 2, label: "Interchange control number", required: true },
    ],
  },
};

const transaction204Segments: Record<string, X12SegmentDefinition> = {
  B2: {
    id: "B2",
    label: "Beginning segment for shipment information",
    elements: [
      { position: 1, label: "Standard carrier alpha code" },
      { position: 2, label: "Shipment identification number", required: true },
      { position: 3, label: "Shipment method of payment" },
      { position: 4, label: "Shipment method of payment" },
    ],
  },
  B2A: {
    id: "B2A",
    label: "Set purpose",
    elements: [{ position: 1, label: "Transaction set purpose code", required: true }],
  },
  L11: {
    id: "L11",
    label: "Reference identification",
    elements: [
      { position: 1, label: "Reference identification" },
      { position: 2, label: "Reference identification qualifier" },
    ],
  },
  G62: {
    id: "G62",
    label: "Date time",
    elements: [
      { position: 1, label: "Date qualifier" },
      { position: 2, label: "Date" },
      { position: 3, label: "Time qualifier" },
      { position: 4, label: "Time" },
    ],
  },
  NTE: {
    id: "NTE",
    label: "Note",
    elements: [
      { position: 1, label: "Note reference code" },
      { position: 2, label: "Description" },
    ],
  },
  N1: {
    id: "N1",
    label: "Name",
    elements: [
      { position: 1, label: "Entity identifier code" },
      { position: 2, label: "Name" },
    ],
  },
  N3: {
    id: "N3",
    label: "Address",
    elements: [
      { position: 1, label: "Address information" },
      { position: 2, label: "Address information" },
    ],
  },
  N4: {
    id: "N4",
    label: "Geographic location",
    elements: [
      { position: 1, label: "City name" },
      { position: 2, label: "State or province code" },
      { position: 3, label: "Postal code" },
    ],
  },
  G61: {
    id: "G61",
    label: "Contact",
    elements: [
      { position: 1, label: "Contact function code" },
      { position: 2, label: "Name" },
      { position: 3, label: "Communication number qualifier" },
      { position: 4, label: "Communication number" },
    ],
  },
  S5: {
    id: "S5",
    label: "Stop off details",
    elements: [
      { position: 1, label: "Stop sequence number", required: true },
      { position: 2, label: "Stop reason code", required: true },
      { position: 3, label: "Weight" },
      { position: 4, label: "Weight unit code" },
      { position: 5, label: "Number of units shipped" },
      { position: 6, label: "Unit or basis for measurement code" },
    ],
  },
  AT8: {
    id: "AT8",
    label: "Shipment weight packaging and quantity data",
    elements: [
      { position: 1, label: "Weight qualifier" },
      { position: 2, label: "Weight unit code" },
      { position: 3, label: "Weight" },
      { position: 4, label: "Lading quantity" },
    ],
  },
  L5: {
    id: "L5",
    label: "Description marks and numbers",
    elements: [
      { position: 1, label: "Lading line item number" },
      { position: 2, label: "Lading description" },
    ],
  },
  L3: {
    id: "L3",
    label: "Total weight and charges",
    elements: [
      { position: 1, label: "Weight" },
      { position: 2, label: "Weight qualifier" },
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
