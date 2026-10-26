// The EDI messages the Desk's EDI message card is drawn for, as
// `GET /edi/messages/:id/` returns them, and each one's inspection as
// `GET /edi/messages/:id/inspect/` returns it: the X12 split into segments with
// the dictionary's names for them, the envelope's control numbers and counts,
// and the validator's diagnostics. Every time is relative to the scenario's anchor.

const HOUR = 3_600;
const DAY = 86_400;

/** The names the X12 dictionary gives the segments these messages carry, and their elements. */
const DICTIONARY = {
  ISA: [
    "Interchange Control Header",
    [
      "Authorization Information Qualifier",
      "Authorization Information",
      "Security Information Qualifier",
      "Security Information",
      "Interchange ID Qualifier",
      "Interchange Sender ID",
      "Interchange ID Qualifier",
      "Interchange Receiver ID",
      "Interchange Date",
      "Interchange Time",
      "Repetition Separator",
      "Interchange Control Version Number",
      "Interchange Control Number",
      "Acknowledgment Requested",
      "Usage Indicator",
      "Component Element Separator",
    ],
  ],
  GS: [
    "Functional Group Header",
    [
      "Functional Identifier Code",
      "Application Sender's Code",
      "Application Receiver's Code",
      "Date",
      "Time",
      "Group Control Number",
      "Responsible Agency Code",
      "Version / Release / Industry Identifier Code",
    ],
  ],
  ST: [
    "Transaction Set Header",
    ["Transaction Set Identifier Code", "Transaction Set Control Number"],
  ],
  SE: [
    "Transaction Set Trailer",
    ["Number of Included Segments", "Transaction Set Control Number"],
  ],
  GE: ["Functional Group Trailer", ["Number of Transaction Sets Included", "Group Control Number"]],
  IEA: [
    "Interchange Control Trailer",
    ["Number of Included Functional Groups", "Interchange Control Number"],
  ],
  B10: [
    "Beginning Segment for Transportation Carrier Shipment Status Message",
    ["Reference Identification", "Shipment Identification Number", "Standard Carrier Alpha Code"],
  ],
  LX: ["Assigned Number", ["Assigned Number"]],
  AT7: [
    "Shipment Status Details",
    [
      "Shipment Status Code",
      "Shipment Status or Appointment Reason Code",
      "Shipment Appointment Status Code",
      "Shipment Status or Appointment Reason Code",
      "Date",
      "Time",
      "Time Code",
    ],
  ],
  MS1: [
    "Equipment, Shipment, or Real Property Location",
    ["City Name", "State or Province Code", "Country Code"],
  ],
  MS2: [
    "Equipment or Container Owner and Type",
    ["Standard Carrier Alpha Code", "Equipment Number"],
  ],
  L11: [
    "Business Instructions and Reference Number",
    ["Reference Identification", "Reference Identification Qualifier"],
  ],
  N1: [
    "Party Identification",
    ["Entity Identifier Code", "Name", "Identification Code Qualifier", "Identification Code"],
  ],
  N3: ["Party Location", ["Address Information", "Address Information"]],
  N4: [
    "Geographic Location",
    ["City Name", "State or Province Code", "Postal Code", "Country Code"],
  ],
  B3: [
    "Beginning Segment for Carrier's Invoice",
    [
      "Shipment Qualifier",
      "Invoice Number",
      "Shipment Identification Number",
      "Shipment Method of Payment",
      "Weight Unit Code",
      "Date",
      "Net Amount Due",
      "Correction Indicator",
      "Delivery Date",
      "Date/Time Qualifier",
      "Standard Carrier Alpha Code",
    ],
  ],
  C3: ["Currency", ["Currency Code"]],
  L5: ["Description, Marks and Numbers", ["Lading Line Item Number", "Lading Description"]],
  L0: [
    "Line Item - Quantity and Weight",
    ["Lading Line Item Number", "Billed/Rated-as Quantity", "Billed/Rated-as Qualifier", "Weight"],
  ],
  L1: [
    "Rate and Charges",
    [
      "Lading Line Item Number",
      "Freight Rate",
      "Rate/Value Qualifier",
      "Charge",
      "Advances",
      "Prepaid Amount",
      "Rate Combination Point Code",
      "Special Charge or Allowance Code",
    ],
  ],
  L3: [
    "Total Weight and Charges",
    ["Weight", "Weight Qualifier", "Freight Rate", "Rate/Value Qualifier", "Charge"],
  ],
  B2: [
    "Beginning Segment for Shipment Information Transaction",
    [
      "Tariff Service Code",
      "Standard Carrier Alpha Code",
      "Standard Point Location Code",
      "Shipment Identification Number",
      "Weight Unit Code",
      "Shipment Method of Payment",
    ],
  ],
  B2A: ["Set Purpose", ["Transaction Set Purpose Code", "Application Type"]],
  S5: [
    "Stop Off Details",
    [
      "Stop Sequence Number",
      "Stop Reason Code",
      "Weight",
      "Weight Unit Code",
      "Number of Units Shipped",
    ],
  ],
  G62: ["Date/Time", ["Date Qualifier", "Date", "Time Qualifier", "Time"]],
  B1: [
    "Beginning Segment for Booking or Pick-up/Delivery",
    [
      "Standard Carrier Alpha Code",
      "Shipment Identification Number",
      "Date",
      "Reservation Action Code",
    ],
  ],
  K1: ["Remarks", ["Free-form Message", "Free-form Message"]],
  AK1: [
    "Functional Group Response Header",
    [
      "Functional Identifier Code",
      "Group Control Number",
      "Version / Release / Industry Identifier Code",
    ],
  ],
  AK2: [
    "Transaction Set Response Header",
    ["Transaction Set Identifier Code", "Transaction Set Control Number"],
  ],
  AK3: [
    "Data Segment Note",
    [
      "Segment ID Code",
      "Segment Position in Transaction Set",
      "Loop Identifier Code",
      "Segment Syntax Error Code",
    ],
  ],
  AK4: [
    "Data Element Note",
    [
      "Position in Segment",
      "Data Element Reference Number",
      "Data Element Syntax Error Code",
      "Copy of Bad Data Element",
    ],
  ],
  AK5: [
    "Transaction Set Response Trailer",
    ["Transaction Set Acknowledgment Code", "Transaction Set Syntax Error Code"],
  ],
  AK9: [
    "Functional Group Response Trailer",
    [
      "Functional Group Acknowledge Code",
      "Number of Transaction Sets Included",
      "Number of Received Transaction Sets",
      "Number of Accepted Transaction Sets",
    ],
  ],
};

const ENVELOPE = new Set(["ISA", "IEA", "GS", "GE", "ST", "SE"]);

/** An X12 interchange around one transaction set's body, with its trailers counted. */
function interchange({ sender, receiver, icn, gcn, stcn, set, group, date, time, body }) {
  const pad = (value, width) => value.padEnd(width, " ");
  const transaction = [`ST*${set}*${stcn}`, ...body];
  return [
    `ISA*00*${pad("", 10)}*00*${pad("", 10)}*ZZ*${pad(sender, 15)}*ZZ*${pad(receiver, 15)}*${date.slice(2)}*${time}*U*00401*${icn}*0*P*>`,
    `GS*${group}*${sender}*${receiver}*${date}*${time}*${gcn}*X*004010`,
    ...transaction,
    `SE*${transaction.length + 1}*${stcn}`,
    `GE*1*${gcn}`,
    `IEA*1*${icn}`,
  ].join("~\n");
}

/** The X12 as the inspector reads it: each segment's elements under the dictionary's names. */
function inspect(raw, diagnostics = []) {
  let offset = 0;
  const lines = raw.split("~\n");
  const segments = lines.map((line, index) => {
    const [id, ...values] = line.split("*");
    const [name, labels] = DICTIONARY[id] ?? [id, []];
    const start = offset;
    offset += line.length + 2;
    return {
      index,
      segmentId: id,
      name,
      type: ENVELOPE.has(id) ? "envelope" : "body",
      raw: line,
      rawWithTerminator: `${line}~`,
      startOffset: start,
      endOffset: start + line.length,
      elements: values.map((value, position) => ({
        position: position + 1,
        label: labels[position] ?? `Element ${String(position + 1).padStart(2, "0")}`,
        value,
        empty: value.trim() === "",
        required: false,
        known: position < labels.length,
        startOffset: start,
        endOffset: start + line.length,
        components: [],
      })),
      malformed: false,
    };
  });
  const element = (id, position) =>
    segments.find((segment) => segment.segmentId === id)?.elements[position - 1]?.value ?? "";
  const st = segments.findIndex((segment) => segment.segmentId === "ST");
  const se = segments.findIndex((segment) => segment.segmentId === "SE");
  const errors = diagnostics.filter((entry) => entry.severity === "Error").length;
  const warnings = diagnostics.filter((entry) => entry.severity === "Warning").length;
  return {
    rawX12: raw,
    transactionSet: element("ST", 1),
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
      segmentCount: segments.length,
      groupCount: 1,
      transactionCount: 1,
      errorCount: errors,
      warningCount: warnings,
      infoCount: diagnostics.length - errors - warnings,
    },
    envelope: {
      isaControlNumber: element("ISA", 13),
      ieaControlNumber: element("IEA", 2),
      expectedGroups: Number(element("IEA", 1)),
      actualGroups: 1,
    },
    groups: [
      {
        index: 0,
        functionalIdCode: element("GS", 1),
        gsControlNumber: element("GS", 6),
        geControlNumber: element("GE", 2),
        expectedCount: Number(element("GE", 1)),
        actualCount: 1,
        startSegmentIndex: 1,
        endSegmentIndex: segments.length - 2,
      },
    ],
    transactions: [
      {
        index: 0,
        transactionSet: element("ST", 1),
        stControlNumber: element("ST", 2),
        seControlNumber: element("SE", 2),
        expectedSegments: Number(element("SE", 1)),
        actualSegments: se - st + 1,
        startSegmentIndex: st,
        endSegmentIndex: se,
      },
    ],
    segments,
    formatted: raw,
    diagnostics,
  };
}

const partners = {
  tyson: {
    id: "edip_tyson",
    businessUnitId: "bu_1",
    organizationId: "org_1",
    kind: "External",
    status: "Active",
    code: "TYSON",
    name: "Tyson Foods",
    description: "",
    country: "US",
    enabledForInbound: true,
    enabledForOutbound: true,
    settings: {},
    version: 4,
  },
  acme: {
    id: "edip_acme",
    businessUnitId: "bu_1",
    organizationId: "org_1",
    kind: "External",
    status: "Active",
    code: "ACME",
    name: "Acme Foods",
    description: "",
    country: "US",
    enabledForInbound: true,
    enabledForOutbound: true,
    settings: {},
    version: 2,
  },
};

/** A message as the API returns it, every id the server leaves empty as an empty string. */
function message(fields) {
  const raw = fields.rawX12 ?? "";
  return {
    businessUnitId: "bu_1",
    organizationId: "org_1",
    ladingVersionId: "",
    flowInstanceId: "",
    shipmentId: "",
    invoiceId: "",
    transferId: "",
    inboundFileId: "",
    standard: "X12",
    x12Version: "004010",
    status: "Generated",
    segmentCount: raw ? raw.split("~\n").length - 4 : 0,
    rawPurgedAt: null,
    deliveryStatus: "",
    deliveryRemotePath: "",
    deliveryAttempts: 0,
    deliveryLastAttemptAt: null,
    deliverySentAt: null,
    deliveryLastError: "",
    ackStatus: "NotExpected",
    ackMessageId: "",
    ackReceivedAt: null,
    ackLastError: "",
    generatedById: "",
    version: 1,
    diagnosticCount: 0,
    validationErrors: [],
    ...fields,
    rawX12: raw,
  };
}

/** Each EDI message the cards thread names, with its inspection. */
export function deskEdiMessages(anchor) {
  const day = (offset) => {
    const at = new Date((anchor + offset) * 1000);
    return `${at.getUTCFullYear()}${String(at.getUTCMonth() + 1).padStart(2, "0")}${String(at.getUTCDate()).padStart(2, "0")}`;
  };

  const status214 = interchange({
    sender: "TRENOVA",
    receiver: "TYSONFOODS",
    icn: "000000417",
    gcn: "417",
    stcn: "0001",
    set: "214",
    group: "QM",
    date: day(-2 * HOUR),
    time: "1415",
    body: [
      "B10*S2610-1366*BOL-55120*TRNV",
      "L11*PO-77120*PO",
      "N1*SH*ACME PLANT 2",
      "N1*CN*TYSON FOODS DC",
      "N3*2200 W DON TYSON PKWY",
      "N4*SPRINGDALE*AR*72762*US",
      "LX*1",
      `AT7*X1*NS***${day(-2 * HOUR)}*1415*LT`,
      "MS1*SPRINGDALE*AR*US",
      "MS2*TRNV*TRL-5521",
    ],
  });

  const ack997 = interchange({
    sender: "TYSONFOODS",
    receiver: "TRENOVA",
    icn: "000000902",
    gcn: "902",
    stcn: "0902",
    set: "997",
    group: "FA",
    date: day(-2 * HOUR + 300),
    time: "1420",
    body: ["AK1*QM*417*004010", "AK2*214*0001", "AK5*A", "AK9*A*1*1*1"],
  });

  const invoice210 = interchange({
    sender: "TRENOVA",
    receiver: "ACMEFOODS",
    icn: "000000433",
    gcn: "433",
    stcn: "0001",
    set: "210",
    group: "IM",
    date: day(-DAY),
    time: "0912",
    body: [
      `B3**INV-2610-0042*BOL-88213*PP*L*${day(-DAY)}*293050**${day(-2 * DAY)}*035*TRNV`,
      "C3*USD",
      "N1*BT*ACME FOODS",
      "N3*400 COMMERCE ST",
      "N4*DALLAS*TX*7520*US",
      "LX*1",
      "L5*1*FREIGHT ALL KINDS",
      "L0*1*42000*FR*42000",
      "L1*1*2.80*PM*280000",
      "L1*2***13050****FUE",
      "L3*42000*G***293050",
    ],
  });

  const tender204 = interchange({
    sender: "ACMEFOODS",
    receiver: "TRENOVA",
    icn: "000005518",
    gcn: "5518",
    stcn: "0001",
    set: "204",
    group: "SM",
    date: day(-3 * HOUR),
    time: "0805",
    body: [
      "B2**TRNV**BOL-88240**PP",
      "B2A*00*LT",
      "L11*PO-88240*PO",
      "S5*1*CL*42000*L*22",
      `G62*69*${day(DAY)}*U*0800`,
      "N1*SH*ACME PLANT 2",
      "N4*DALLAS*TX*75207",
      "S5*2*CU*42000*L*22",
      `G62*70*${day(2 * DAY)}*X*1400`,
      "N1*CN*ACME DC",
      "N4*HOUSTON*TX*77029",
      "L3*42000*G***288000",
    ],
  });

  const rejected997 = interchange({
    sender: "TYSONFOODS",
    receiver: "TRENOVA",
    icn: "000000931",
    gcn: "931",
    stcn: "0931",
    set: "997",
    group: "FA",
    date: day(-5 * HOUR),
    time: "1105",
    body: [
      "AK1*QM*421*004010",
      "AK2*214*0003",
      "AK3*AT7*8**8",
      "AK4*1*1650*7*ZZ",
      "AK3*N4*6**8",
      "AK4*3*116*1*7520",
      "AK5*R*5",
      "AK9*R*1*1*0",
    ],
  });

  const response990 = interchange({
    sender: "TRENOVA",
    receiver: "ACMEFOODS",
    icn: "000000452",
    gcn: "452",
    stcn: "0001",
    set: "990",
    group: "GF",
    date: day(-HOUR),
    time: "1012",
    body: [`B1*TRNV*BOL-88240*${day(-HOUR)}*D`, "K1*NO CAPACITY ON THE DALLAS HOUSTON LANE"],
  });

  const invoiceDiagnostics = [
    {
      severity: "Error",
      code: "X12_POSTAL_CODE",
      source: "validation",
      segmentId: "N4",
      segmentIndex: 8,
      elementPosition: 3,
      path: "N4[1].03",
      message: "Postal code must be 5 or 9 digits",
      suggestedFix: "Send 75201 for the bill-to's Dallas address",
    },
    {
      severity: "Warning",
      code: "PARTNER_REFERENCE",
      source: "partner_setting",
      message: "Acme asks for its PO on every invoice; none was found on the order",
    },
  ];

  return {
    edimsg_214_tyson: {
      message: message({
        id: "edimsg_214_tyson",
        ediPartnerId: "edip_tyson",
        documentTypeId: "edidt_x12_214_outbound",
        shipmentId: "shp_1366",
        direction: "Outbound",
        transactionSet: "214",
        interchangeControlNumber: "000000417",
        groupControlNumber: "417",
        transactionControlNumber: "0001",
        rawX12: status214,
        payloadSnapshot: {
          transactionSet: "214",
          shipmentStatus: {
            shipmentId: "shp_1366",
            bol: "BOL-55120",
            proNumber: "S2610-1366",
            statusCode: "X1",
            statusReasonCode: "NS",
            eventDate: anchor - 2 * HOUR,
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
        deliveryStatus: "Sent",
        deliveryRemotePath: "as2://tyson-foods/inbound/214",
        deliveryAttempts: 1,
        deliveryLastAttemptAt: anchor - 2 * HOUR + 30,
        deliverySentAt: anchor - 2 * HOUR + 30,
        ackStatus: "Accepted",
        ackMessageId: "edimsg_997_tyson",
        ackReceivedAt: anchor - 2 * HOUR + 300,
        generatedAt: anchor - 2 * HOUR,
        partner: partners.tyson,
      }),
      inspection: inspect(status214),
    },
    edimsg_997_tyson: {
      message: message({
        id: "edimsg_997_tyson",
        ediPartnerId: "edip_tyson",
        documentTypeId: "edidt_x12_997_inbound",
        inboundFileId: "ediinf_902",
        direction: "Inbound",
        transactionSet: "997",
        validationMode: "Disabled",
        interchangeControlNumber: "000000902",
        groupControlNumber: "902",
        transactionControlNumber: "0902",
        rawX12: ack997,
        payloadSnapshot: {
          transactionSet: "997",
          functionalAck: {
            sourceMessageId: "edimsg_214_tyson",
            originalFunctionalGroupId: "QM",
            originalGroupControlNumber: "417",
            originalTransactionSet: "214",
            originalTransactionControlNumber: "0001",
            groupAcknowledgmentCode: "A",
            transactionAcknowledgmentCode: "A",
            acceptedTransactionSetCount: 1,
            receivedTransactionSetCount: 1,
            includedTransactionSetCount: 1,
          },
        },
        generatedAt: anchor - 2 * HOUR + 300,
        partner: partners.tyson,
      }),
      inspection: inspect(ack997),
    },
    edimsg_210_acme: {
      message: message({
        id: "edimsg_210_acme",
        ediPartnerId: "edip_acme",
        documentTypeId: "edidt_x12_210_outbound",
        shipmentId: "shp_1342",
        invoiceId: "inv_2610_0042",
        direction: "Outbound",
        transactionSet: "210",
        interchangeControlNumber: "000000433",
        groupControlNumber: "433",
        transactionControlNumber: "0001",
        rawX12: invoice210,
        payloadSnapshot: {
          transactionSet: "210",
          invoice: {
            invoiceId: "inv_2610_0042",
            invoiceNumber: "INV-2610-0042",
            invoiceDate: anchor - DAY,
            deliveryDate: anchor - 2 * DAY,
            shipmentId: "shp_1342",
            bol: "BOL-88213",
            proNumber: "S2610-1342",
            billToName: "Acme Foods",
            billToCity: "Dallas",
            billToStateCode: "TX",
            billToPostalCode: "7520",
            currencyCode: "USD",
            totalAmount: "2930.50",
            lineCharges: [
              {
                sequence: 1,
                code: "400",
                description: "Line haul, 1,000 mi at $2.80",
                amount: "2800.00",
              },
              { sequence: 2, code: "FUE", description: "Fuel surcharge", amount: "130.50" },
            ],
          },
        },
        deliveryStatus: "DeadLettered",
        deliveryRemotePath: "sftp://edi.acmefoods.example/inbound/210",
        deliveryAttempts: 5,
        deliveryLastAttemptAt: anchor - 3 * HOUR,
        deliveryLastError:
          "SFTP upload refused: 553 file name not allowed by the partner's mailbox policy",
        ackStatus: "Pending",
        generatedAt: anchor - DAY,
        diagnosticCount: invoiceDiagnostics.length,
        validationErrors: invoiceDiagnostics.map(
          ({ source: _source, segmentIndex: _index, ...rest }) => rest,
        ),
        partner: partners.acme,
      }),
      inspection: inspect(invoice210, invoiceDiagnostics),
    },
    edimsg_214_old: {
      message: message({
        id: "edimsg_214_old",
        ediPartnerId: "edip_tyson",
        documentTypeId: "edidt_x12_214_outbound",
        direction: "Outbound",
        transactionSet: "214",
        interchangeControlNumber: "000000118",
        groupControlNumber: "118",
        transactionControlNumber: "0001",
        segmentCount: 9,
        rawX12: "",
        rawPurgedAt: anchor - 20 * DAY,
        payloadSnapshot: null,
        deliveryStatus: "Sent",
        deliveryRemotePath: "as2://tyson-foods/inbound/214",
        deliveryAttempts: 2,
        deliveryLastAttemptAt: anchor - 120 * DAY,
        deliverySentAt: anchor - 120 * DAY,
        ackStatus: "NotExpected",
        generatedAt: anchor - 120 * DAY,
        partner: partners.tyson,
      }),
      inspection: null,
    },
    edimsg_204_acme: {
      message: message({
        id: "edimsg_204_acme",
        ediPartnerId: "edip_acme",
        documentTypeId: "edidt_x12_204_inbound",
        inboundFileId: "ediinf_5518",
        transferId: "edilt_5518",
        direction: "Inbound",
        transactionSet: "204",
        validationMode: "Disabled",
        interchangeControlNumber: "000005518",
        groupControlNumber: "5518",
        transactionControlNumber: "0001",
        rawX12: tender204,
        payloadSnapshot: {
          transactionSet: "204",
          purposeCode: "00",
          loadTender: {
            bol: "BOL-88240",
            customerLabel: "Acme Foods",
            serviceTypeLabel: "Dry van",
            weight: 42000,
            pieces: 22,
            totalChargeAmount: "2880.00",
            moves: [
              {
                loaded: true,
                sequence: 1,
                distance: 239,
                stops: [
                  {
                    locationId: "loc_acme_plant2",
                    locationName: "Acme Plant 2",
                    locationCity: "Dallas",
                    locationStateCode: "TX",
                    type: "Pickup",
                    scheduleType: "Window",
                    sequence: 1,
                    scheduledWindowStart: anchor + DAY,
                  },
                  {
                    locationId: "loc_acme_dc",
                    locationName: "Acme DC",
                    locationCity: "Houston",
                    locationStateCode: "TX",
                    type: "Delivery",
                    scheduleType: "Appointment",
                    sequence: 2,
                    scheduledWindowStart: anchor + 2 * DAY,
                  },
                ],
              },
            ],
          },
        },
        generatedAt: anchor - 3 * HOUR,
        partner: partners.acme,
      }),
      inspection: inspect(tender204),
    },
    edimsg_997_rejected: {
      message: message({
        id: "edimsg_997_rejected",
        ediPartnerId: "edip_tyson",
        documentTypeId: "edidt_x12_997_inbound",
        inboundFileId: "ediinf_931",
        direction: "Inbound",
        transactionSet: "997",
        validationMode: "Disabled",
        interchangeControlNumber: "000000931",
        groupControlNumber: "931",
        transactionControlNumber: "0931",
        rawX12: rejected997,
        payloadSnapshot: {
          transactionSet: "997",
          functionalAck: {
            sourceMessageId: "edimsg_214_tyson",
            originalFunctionalGroupId: "QM",
            originalGroupControlNumber: "421",
            originalTransactionSet: "214",
            originalTransactionControlNumber: "0003",
            groupAcknowledgmentCode: "R",
            transactionAcknowledgmentCode: "R",
            acceptedTransactionSetCount: 0,
            receivedTransactionSetCount: 1,
            includedTransactionSetCount: 1,
            diagnostics: [
              {
                segmentId: "AT7",
                segmentPosition: 8,
                elementPosition: 1,
                errorCode: "7",
                message: "Invalid code value: ZZ is not a shipment status code",
              },
              {
                segmentId: "N4",
                segmentPosition: 6,
                elementPosition: 3,
                errorCode: "1",
                message: "Mandatory postal code was cut short",
              },
            ],
          },
        },
        generatedAt: anchor - 5 * HOUR,
        partner: partners.tyson,
      }),
      inspection: inspect(rejected997),
    },
    edimsg_990_acme: {
      message: message({
        id: "edimsg_990_acme",
        ediPartnerId: "edip_acme",
        documentTypeId: "edidt_x12_990_outbound",
        transferId: "edilt_5518",
        direction: "Outbound",
        transactionSet: "990",
        interchangeControlNumber: "000000452",
        groupControlNumber: "452",
        transactionControlNumber: "0001",
        rawX12: response990,
        payloadSnapshot: {
          transactionSet: "990",
          tenderResponse: {
            transferId: "edilt_5518",
            bol: "BOL-88240",
            responseCode: "D",
            reasonCode: "CAP",
            rejectionReason: "No capacity on the Dallas to Houston lane this week",
            status: "Rejected",
          },
        },
        deliveryStatus: "Sent",
        deliveryRemotePath: "sftp://edi.acmefoods.example/inbound/990",
        deliveryAttempts: 1,
        deliveryLastAttemptAt: anchor - HOUR + 40,
        deliverySentAt: anchor - HOUR + 40,
        ackStatus: "Pending",
        generatedAt: anchor - HOUR,
        partner: partners.acme,
      }),
      inspection: inspect(response990),
    },
  };
}

/** The cards thread's EDI conversations: each slug's message, its title, and what was asked and said. */
export const DESK_EDI_CONVERSATIONS = {
  "edi-214-tyson": {
    recordId: "edimsg_214_tyson",
    title: "EDI message 214 to Tyson Foods",
    question: "Did Tyson get our delivered status for S2610-1366?",
    reply:
      "Yes: the 214 went out over AS2 on the first attempt and Tyson's 997 accepted it five minutes later. [214 to Tyson Foods](artifact:art_edi-214-tyson) has the X12 and its acknowledgment.",
  },
  "edi-210-acme": {
    recordId: "edimsg_210_acme",
    title: "EDI message 210 to Acme Foods",
    question: "Why hasn't Acme received invoice INV-2610-0042?",
    reply:
      "The 210 was dead-lettered after five SFTP attempts: Acme's mailbox refused the file name. The X12 also carries a four-digit postal code the validator flagged. [210 to Acme Foods](artifact:art_edi-210-acme) shows both.",
  },
  "edi-214-old": {
    recordId: "edimsg_214_old",
    title: "EDI message 214 to Tyson Foods",
    question: "Pull up the 214 we sent Tyson in June.",
    reply:
      "It was delivered on the second attempt; its raw X12 has since been purged under the retention policy. [214 to Tyson Foods](artifact:art_edi-214-old) shows what is kept.",
  },
  "edi-gone": {
    recordId: "edimsg_gone",
    title: "EDI message 204 from Globex",
    question: "Show me the Globex 204 from last week.",
    reply:
      "That message has been removed since I looked it up. [204 from Globex](artifact:art_edi-gone) is what I found then.",
  },
  "edi-204-acme": {
    recordId: "edimsg_204_acme",
    title: "EDI message 204 from Acme Foods",
    question: "What's in the tender Acme just sent?",
    reply:
      "A dry van load from Acme Plant 2 in Dallas to their Houston DC tomorrow, 42,000 lb at $2,880.00. It is waiting in inbound tenders. [204 from Acme Foods](artifact:art_edi-204-acme) has the stops.",
  },
  "edi-997-rejected": {
    recordId: "edimsg_997_rejected",
    title: "EDI message 997 from Tyson Foods",
    question: "Did Tyson reject one of our status messages?",
    reply:
      "Yes: their 997 rejected 214 0003, naming an AT7 status code they don't accept and a short postal code. [997 from Tyson Foods](artifact:art_edi-997-rejected) lists both faults.",
  },
  "edi-990-acme": {
    recordId: "edimsg_990_acme",
    title: "EDI message 990 to Acme Foods",
    question: "Did we answer Acme's tender?",
    reply:
      "We declined it an hour ago for lack of capacity on the lane; the 990 was delivered and Acme has not acknowledged it yet. [990 to Acme Foods](artifact:art_edi-990-acme) has the answer.",
  },
};
