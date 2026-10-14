import {
  ediMessageInspectionSchema,
  ediMessageSchema,
  type EDIMessage,
  type EDIMessageInspection,
} from "@trenova/shared/types/edi";

export const NOW = 1_791_720_000;
export const HOUR = 3_600;

/**
 * An outbound 214 as `GET /edi/messages/:id/` returns it: the Go struct's JSON,
 * empty ids as empty strings, the partner and its validation errors joined.
 * Delivered over AS2 and accepted by the partner's 997.
 */
export function wireMessage(overrides: Record<string, unknown> = {}): Record<string, unknown> {
  return {
    id: "edimsg_1",
    businessUnitId: "bu_1",
    organizationId: "org_1",
    ediPartnerId: "edip_1",
    documentTypeId: "edidt_x12_214_outbound",
    partnerDocumentProfileId: "edipdp_1",
    templateId: "editpl_1",
    templateVersionId: "editplv_1",
    shipmentId: "shp_1",
    invoiceId: "",
    transferId: "",
    inboundFileId: "",
    direction: "Outbound",
    standard: "X12",
    transactionSet: "214",
    x12Version: "004010",
    status: "Generated",
    validationMode: "Strict",
    interchangeControlNumber: "000000417",
    groupControlNumber: "417",
    transactionControlNumber: "0001",
    segmentCount: 6,
    rawX12: RAW_214,
    payloadSnapshot: {
      transactionSet: "214",
      shipmentStatus: {
        shipmentId: "shp_1",
        bol: "BOL-55120",
        proNumber: "S2610-1366",
        statusCode: "X1",
        statusReasonCode: "NS",
        eventDate: NOW - 2 * HOUR,
        stopType: "Delivery",
        stopSequence: 3,
        locationName: "Tyson Foods DC",
        city: "Springdale",
        stateCode: "AR",
        postalCode: "72762",
        appointmentNumber: "APT-7781",
        equipmentNumber: "TRL-5521",
      },
    },
    rawPurgedAt: null,
    deliveryStatus: "Sent",
    deliveryRemotePath: "as2://tyson/inbound",
    deliveryAttempts: 1,
    deliveryLastAttemptAt: NOW - 2 * HOUR + 30,
    deliverySentAt: NOW - 2 * HOUR + 30,
    deliveryLastError: "",
    as2MessageId: "<trenova-417@trenova>",
    as2Mic: "",
    ackStatus: "Accepted",
    ackMessageId: "edimsg_ack",
    ackReceivedAt: NOW - 2 * HOUR + 300,
    ackLastError: "",
    generatedById: "usr_1",
    generatedAt: NOW - 2 * HOUR,
    version: 3,
    createdAt: NOW - 2 * HOUR,
    updatedAt: NOW - 2 * HOUR + 300,
    diagnosticCount: 0,
    partner: {
      id: "edip_1",
      businessUnitId: "bu_1",
      organizationId: "org_1",
      kind: "External",
      status: "Active",
      code: "TYSON",
      name: "Tyson Foods",
      description: "",
      internalOrganizationId: "",
      ediConnectionId: "",
      customerId: "cus_1",
      defaultTransportId: "",
      defaultMappingProfileId: "",
      timezone: "America/Chicago",
      country: "US",
      contactName: "",
      contactEmail: "",
      contactPhone: "",
      enabledForInbound: true,
      enabledForOutbound: true,
      settings: {},
      version: 1,
      createdAt: NOW - 400 * 24 * HOUR,
      updatedAt: NOW - 30 * 24 * HOUR,
    },
    ...overrides,
  };
}

/**
 * The wire form of the message's partner document profile, as edi.EDIPartnerDocumentProfile
 * marshals it. Its acknowledgment column defaults to '{}', so a profile saved without
 * acknowledgment expectations sends every field at its zero value: an empty type and an
 * empty severity.
 */
export function wireDocumentProfile(
  acknowledgment: Record<string, unknown> = {
    expected: false,
    type: "",
    slaInMinutes: 0,
    missingAckSeverity: "",
  },
): Record<string, unknown> {
  return {
    id: "edipdp_1",
    businessUnitId: "bu_1",
    organizationId: "org_1",
    ediPartnerId: "edip_1",
    documentTypeId: "edidt_1",
    templateId: "editpl_1",
    templateVersionId: null,
    name: "Tyson 214",
    status: "Active",
    direction: "Outbound",
    standard: "X12",
    transactionSet: "214",
    x12VersionOverride: "",
    functionalGroupId: "QM",
    envelope: {
      interchangeSenderQualifier: "ZZ",
      interchangeSenderId: "TRENOVA",
      interchangeReceiverQualifier: "ZZ",
      interchangeReceiverId: "TYSON",
      applicationSenderCode: "TRENOVA",
      applicationReceiverCode: "TYSON",
      interchangeUsageIndicator: "P",
      elementSeparator: "*",
      segmentTerminator: "~",
      componentSeparator: ">",
      repetitionSeparator: "^",
    },
    acknowledgment,
    validationMode: "Strict",
    partnerSettings: {},
    partnerSettingsSchemaId: null,
    partnerSettingsSchemaVersion: 0,
    version: 1,
    createdAt: NOW - 30 * 24 * HOUR,
    updatedAt: NOW - 30 * 24 * HOUR,
  };
}

export const RAW_214 = [
  "ISA*00*          *00*          *ZZ*TRENOVA        *ZZ*TYSONFOODS     *261009*1415*U*00401*000000417*0*P*>",
  "GS*QM*TRENOVA*TYSONFOODS*20261009*1415*417*X*004010",
  "ST*214*0001",
  "B10*S2610-1366*BOL-55120*TRNV",
  "LX*1",
  "AT7*X1*NS***20261009*1415*LT",
  "MS1*SPRINGDALE*AR",
  "SE*6*0001",
  "GE*1*417",
  "IEA*1*000000417",
].join("~\n");

type WireSegment = { id: string; name: string; elements: [string, string][] };

const SEGMENTS_214: WireSegment[] = [
  {
    id: "ISA",
    name: "Interchange Control Header",
    elements: [
      ["Authorization Information Qualifier", "00"],
      ["Interchange Sender ID", "TRENOVA"],
      ["Interchange Receiver ID", "TYSONFOODS"],
      ["Interchange Control Number", "000000417"],
    ],
  },
  {
    id: "GS",
    name: "Functional Group Header",
    elements: [
      ["Functional Identifier Code", "QM"],
      ["Application Sender's Code", "TRENOVA"],
      ["Group Control Number", "417"],
    ],
  },
  {
    id: "ST",
    name: "Transaction Set Header",
    elements: [
      ["Transaction Set Identifier Code", "214"],
      ["Transaction Set Control Number", "0001"],
    ],
  },
  {
    id: "B10",
    name: "Beginning Segment for Transportation Carrier Shipment Status Message",
    elements: [
      ["Reference Identification", "S2610-1366"],
      ["Shipment Identification Number", "BOL-55120"],
      ["Standard Carrier Alpha Code", "TRNV"],
    ],
  },
  { id: "LX", name: "Assigned Number", elements: [["Assigned Number", "1"]] },
  {
    id: "AT7",
    name: "Shipment Status Details",
    elements: [
      ["Shipment Status Code", "X1"],
      ["Shipment Status or Appointment Reason Code", "NS"],
      ["Date", "20261009"],
    ],
  },
  {
    id: "MS1",
    name: "Equipment, Shipment, or Real Property Location",
    elements: [
      ["City Name", "SPRINGDALE"],
      ["State or Province Code", "AR"],
    ],
  },
  {
    id: "SE",
    name: "Transaction Set Trailer",
    elements: [
      ["Number of Included Segments", "6"],
      ["Transaction Set Control Number", "0001"],
    ],
  },
  {
    id: "GE",
    name: "Functional Group Trailer",
    elements: [
      ["Number of Transaction Sets Included", "1"],
      ["Group Control Number", "417"],
    ],
  },
  {
    id: "IEA",
    name: "Interchange Control Trailer",
    elements: [
      ["Number of Included Functional Groups", "1"],
      ["Interchange Control Number", "000000417"],
    ],
  },
];

/**
 * `GET /edi/messages/:id/inspect/` for the 214: the inspector's JSON, with
 * `omitempty` fields left out where Go leaves them out (a zero segment index,
 * an absent count).
 */
export function wireInspection(
  overrides: { inspection?: Record<string, unknown>; message?: Record<string, unknown> } = {},
): Record<string, unknown> {
  let offset = 0;
  const segments = SEGMENTS_214.map((segment, index) => {
    const raw = [segment.id, ...segment.elements.map(([, value]) => value)].join("*");
    const start = offset;
    offset += raw.length + 2;
    return {
      index,
      ...(index >= 2 && index <= 7 ? { transactionIndex: 0 } : {}),
      segmentId: segment.id,
      name: segment.name,
      type: index < 3 || index > 6 ? "envelope" : "body",
      raw,
      rawWithTerminator: `${raw}~`,
      startOffset: start,
      endOffset: start + raw.length,
      elements: segment.elements.map(([label, value], position) => ({
        position: position + 1,
        label,
        value,
        empty: value === "",
        required: true,
        known: true,
        startOffset: start,
        endOffset: start + raw.length,
        components: [],
      })),
      malformed: false,
    };
  });
  return {
    message: wireMessage(overrides.message),
    inspection: {
      rawX12: RAW_214,
      transactionSet: "214",
      x12Version: "004010",
      separators: {
        element: "*",
        segment: "~",
        component: ">",
        repetition: "U",
        source: "isa",
        hasConflict: false,
      },
      summary: {
        segmentCount: 10,
        groupCount: 1,
        transactionCount: 1,
        errorCount: 0,
        warningCount: 0,
        infoCount: 0,
      },
      envelope: {
        isaControlNumber: "000000417",
        ieaControlNumber: "000000417",
        expectedGroups: 1,
        actualGroups: 1,
      },
      groups: [
        {
          index: 0,
          functionalIdCode: "QM",
          gsControlNumber: "417",
          geControlNumber: "417",
          expectedCount: 1,
          actualCount: 1,
          startSegmentIndex: 1,
          endSegmentIndex: 8,
        },
      ],
      transactions: [
        {
          index: 0,
          transactionSet: "214",
          stControlNumber: "0001",
          seControlNumber: "0001",
          expectedSegments: 6,
          actualSegments: 6,
          startSegmentIndex: 2,
          endSegmentIndex: 7,
        },
      ],
      segments,
      formatted: RAW_214,
      diagnostics: [],
      ...overrides.inspection,
    },
    provenance: {
      messageId: "edimsg_1",
      profileId: "edipdp_1",
      templateId: "editpl_1",
      templateVersionId: "editplv_1",
      generatedAt: NOW - 2 * HOUR,
      generatedById: "usr_1",
    },
  };
}

/** The 214 as the client reads it, through the same schema the service parses it with. */
export function ediMessage(overrides: Record<string, unknown> = {}): EDIMessage {
  return ediMessageSchema.parse(wireMessage(overrides));
}

/** The 214's inspection as the client reads it. */
export function ediInspection(
  overrides: Parameters<typeof wireInspection>[0] = {},
): EDIMessageInspection {
  return ediMessageInspectionSchema.parse(wireInspection(overrides));
}
