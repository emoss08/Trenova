// An EDI connection drafted on the Desk from a trading partner's implementation
// guide: Acme Retail Group's guide (deskbench/fixtures/attachments/edi-guide.html,
// version 3.2 and the revised 4.0), read by the EDI desk and drafted with
// draft_edi_connection. Each draft is the review the artifact carries, shaped
// as assistantservice.connectionDraftArtifact stores it. Every time is relative
// to the scenario's anchor.

export const EDI_DRAFT_THREAD_ID = "athr_desk_edi_guide";
export const EDI_DRAFT_AGENT_ID = "agdef_edi";
export const EDI_DRAFT_PROPOSAL_ID = "aprop_edi_acme";

const GUIDE = {
  documentId: "doc_acme_guide_v3",
  fileName: "acme-retail-edi-guide-v3.2.pdf",
  fileSize: 188_416,
  digest: "sha256:5f1c9a0b7e3d2c1f4a8e6b0d9c2f7a1e3b5d8c0f6a4e2d9b7c1f3a5e8d0b6c4a",
};

const REVISED_GUIDE = {
  documentId: "doc_acme_guide_v4",
  fileName: "acme-retail-edi-guide-v4.0.pdf",
  fileSize: 190_464,
  digest: "sha256:a83d27c5e9f14b06d2c8e5a17f3b90d4c6e2a8f1b3d5c7e9a0f2b4d6c8e1a3f5",
};

const PRODUCTION_URL = "https://as2.acmeretail.example/as2/receive";
const TEST_URL = "https://as2-test.acmeretail.example/as2/receive";

function doc(page, quote, section = "") {
  return { kind: "Document", page, quote, section, note: "" };
}

function value(field, label, val, source, extra = {}) {
  return {
    field,
    label,
    value: val,
    current: "",
    change: "Add",
    status: "Accepted",
    required: false,
    confidence: 0.96,
    source,
    candidates: [],
    reason: "",
    ...extra,
  };
}

function same(entry, current = entry.value) {
  return { ...entry, current, change: "Same" };
}

function changed(entry, current) {
  return { ...entry, current, change: "Change" };
}

function partnerFields(revised) {
  const base = [
    value(
      "partner.code",
      "Partner code",
      "ACMERETAIL",
      doc(1, "Trading Partner Code: ACMERETAIL", "1"),
      { required: true, confidence: 0.99 },
    ),
    value(
      "partner.name",
      "Partner name",
      "Acme Retail Group",
      doc(1, "Trading Partner Name: Acme Retail Group", "1"),
      { required: true, confidence: 0.99 },
    ),
    value(
      "partner.contactName",
      "Contact name",
      "Dana Whitfield",
      doc(1, "EDI Coordinator: Dana Whitfield", "1"),
    ),
    value(
      "partner.contactEmail",
      "Contact email",
      "edi@acmeretail.example",
      doc(1, "Email: edi@acmeretail.example", "1"),
    ),
    value(
      "partner.contactPhone",
      "Contact phone",
      "+1 312 555 0142",
      doc(1, "Phone: +1 312 555 0142", "1"),
    ),
    value(
      "partner.timezone",
      "Time zone",
      "America/Chicago",
      doc(1, "All times in this guide are Central Time (America/Chicago).", "1"),
    ),
    value(
      "partner.country",
      "Country",
      "US",
      { kind: "Default", page: 0, quote: "", section: "", note: "" },
      { status: "Defaulted", confidence: 1 },
    ),
  ];
  if (!revised) return base;
  return base.map((entry) => {
    switch (entry.field) {
      case "partner.contactName":
        return changed(
          { ...entry, value: "Priya Raman", source: doc(1, "EDI Coordinator: Priya Raman", "1") },
          "Dana Whitfield",
        );
      case "partner.contactEmail":
        return changed(
          {
            ...entry,
            value: "edi-ops@acmeretail.example",
            source: doc(1, "Email: edi-ops@acmeretail.example", "1"),
          },
          "edi@acmeretail.example",
        );
      case "partner.contactPhone":
        return changed(
          { ...entry, value: "+1 312 555 0177", source: doc(1, "Phone: +1 312 555 0177", "1") },
          "+1 312 555 0142",
        );
      default:
        return same(entry);
    }
  });
}

function interchange(revised) {
  const rows = [
    value(
      "isa.partnerQualifier",
      "Partner ISA qualifier",
      "01",
      doc(3, "ISA05/ISA06 Acme interchange sender: qualifier 01 (DUNS), ID 007912345", "3.1"),
      { required: true },
    ),
    value(
      "isa.partnerId",
      "Partner ISA ID",
      "007912345",
      doc(3, "ISA05/ISA06 Acme interchange sender: qualifier 01 (DUNS), ID 007912345", "3.1"),
      { required: true, confidence: 0.99 },
    ),
    value(
      "isa.ourQualifier",
      "Your ISA qualifier",
      "02",
      doc(3, "ISA07/ISA08 Carrier interchange receiver: qualifier 02 (SCAC), ID TRNV", "3.1"),
      { required: true },
    ),
    value(
      "isa.ourId",
      "Your ISA ID",
      "TRNV",
      doc(3, "ISA07/ISA08 Carrier interchange receiver: qualifier 02 (SCAC), ID TRNV", "3.1"),
      { required: true, confidence: 0.99 },
    ),
    value(
      "gs.partnerId",
      "Partner GS ID",
      "ACMERETAIL",
      doc(3, "GS02 Application Sender Code: ACMERETAIL", "3.2"),
      { required: true },
    ),
    value("gs.ourId", "Your GS ID", "TRNV", doc(3, "GS03 Application Receiver Code: TRNV", "3.2"), {
      required: true,
    }),
    value(
      "isa.usageIndicator",
      "Usage indicator",
      "T",
      doc(3, "ISA15 usage indicator: T during testing, P in production.", "3.1"),
    ),
    value("x12.version", "X12 version", "004010", doc(3, "GS08 Version: 004010", "3.2"), {
      required: true,
    }),
    value(
      "separators.element",
      "Element separator",
      "*",
      doc(3, "Element separator: * (asterisk).", "3.3"),
    ),
    value(
      "separators.segment",
      "Segment terminator",
      "~",
      doc(3, "Segment terminator: ~ (tilde).", "3.3"),
    ),
    value(
      "separators.component",
      "Component separator",
      ">",
      doc(3, "Sub-element separator: > (greater than).", "3.3"),
    ),
  ];
  return revised ? rows.map((entry) => same(entry)) : rows;
}

function endpoint(stage) {
  const label = "AS2 endpoint URL";
  if (stage === "held") {
    return value("as2.endpointUrl", label, "", doc(2, ""), {
      change: "None",
      status: "Held",
      required: true,
      confidence: 0,
      reason: "The guide gives two endpoints, one for production and one for testing.",
      candidates: [
        {
          value: PRODUCTION_URL,
          confidence: 0.91,
          source: doc(2, `Production AS2 URL: ${PRODUCTION_URL}`, "2.1"),
          problem: "",
        },
        {
          value: TEST_URL,
          confidence: 0.9,
          source: doc(2, `Test AS2 URL: ${TEST_URL}`, "2.1"),
          problem: "",
        },
      ],
    });
  }
  const answered = value(
    "as2.endpointUrl",
    label,
    TEST_URL,
    { kind: "Person", page: 2, quote: `Test AS2 URL: ${TEST_URL}`, section: "2.1", note: "" },
    { required: true, confidence: 1 },
  );
  return stage === "revised" ? same(answered) : answered;
}

function transport(stage) {
  const revised = stage === "revised";
  const rows = [
    value(
      "transport.method",
      "Connection method",
      "AS2",
      doc(2, "Acme exchanges all documents over AS2 (Applicability Statement 2).", "2.1"),
      { required: true, confidence: 0.99 },
    ),
    value(
      "transport.environment",
      "Environment",
      "test",
      doc(
        2,
        "Begin in the test environment; Acme moves the connection to production after certification.",
        "2.2",
      ),
    ),
    endpoint(stage),
    value(
      "as2.partnerAs2Id",
      "Partner AS2 ID",
      "ACMERETAILAS2",
      doc(2, "Acme AS2 Identifier: ACMERETAILAS2", "2.1"),
      { required: true },
    ),
    value(
      "as2.localAs2Id",
      "Your AS2 ID",
      "TRENOVA-AS2",
      doc(2, "Carrier AS2 Identifier: TRENOVA-AS2", "2.1"),
      { required: true },
    ),
    value("as2.mdnMode", "MDN mode", "sync", doc(2, "MDN: synchronous, signed.", "2.1")),
    value(
      "as2.signingAlgorithm",
      "Signing algorithm",
      "sha256",
      doc(2, "Signing algorithm: SHA-256.", "2.1"),
    ),
    value(
      "as2.encryptionAlgorithm",
      "Encryption algorithm",
      "aes256-cbc",
      doc(2, "Encryption algorithm: AES-256 (CBC).", "2.1"),
    ),
  ];
  return {
    profileId: revised ? "edicp_acme_as2" : "",
    version: revised ? 3 : 0,
    name: "Acme Retail Group AS2",
    method: "AS2",
    status: "Inactive",
    change: revised ? "Same" : "Add",
    fields: revised ? rows.map((entry) => (entry.change === "Same" ? entry : same(entry))) : rows,
    credentials: [
      {
        keys: ["partnerSigningCertificate"],
        label: "Partner signing certificate",
        kind: "Secret",
        required: true,
        missing: !revised,
      },
      {
        keys: ["partnerEncryptionCertificate"],
        label: "Partner encryption certificate",
        kind: "Secret",
        required: true,
        missing: !revised,
      },
      {
        keys: ["localCertificate"],
        label: "Your signing certificate",
        kind: "Secret",
        required: true,
        missing: !revised,
      },
      {
        keys: ["as2PrivateKey"],
        label: "AS2 private key for your signing certificate",
        kind: "Secret",
        required: true,
        missing: !revised,
      },
    ],
  };
}

function control(revised) {
  const source = revised
    ? doc(
        3,
        "Carrier interchange control numbers (ISA13) must start at 5000 and increase by one.",
        "3.4",
      )
    : doc(
        3,
        "Carrier interchange control numbers (ISA13) must start at 1000 and increase by one.",
        "3.4",
      );
  return [
    {
      kind: "Interchange",
      start: revised ? 5000 : 1000,
      current: revised ? 1042 : 0,
      change: revised ? "Change" : "Add",
      source,
      problem: "",
    },
  ];
}

function template(name, met, unmet, extra = {}) {
  return {
    templateId: `edtpl_${name.toLowerCase().replace(/[^a-z0-9]+/g, "_")}`,
    name,
    starter: true,
    current: "",
    change: "Add",
    met,
    unmet,
    considered: [],
    unverified: [],
    note: "",
    ...extra,
  };
}

const MS1_UNMET = {
  segmentId: "MS1",
  elementPosition: 1,
  usage: "Mandatory",
  codes: [],
  problem: "The template never writes MS1.",
  source: doc(5, "MS1 Equipment Location is mandatory on every status.", "6"),
};

function setFields(set, direction, group, ack) {
  const name = `${set} ${direction.toLowerCase()}`;
  const prefix = `sets.${set}.${direction.toLowerCase()}.`;
  const rows = [
    value(
      `${prefix}functionalGroup`,
      `${name} functional group`,
      group,
      doc(
        4,
        `The carrier ${set === "990" ? "answers each tender with a 990" : set === "214" ? "sends the 214" : set === "210" ? "invoices with the 210" : "receives the 204"}`,
        "4.1",
      ),
      {
        status: "Defaulted",
        source: { kind: "Default", page: 0, quote: "", section: "", note: "" },
      },
    ),
  ];
  if (ack) {
    rows.push(
      value(
        `${prefix}ack`,
        `${name} functional acknowledgment`,
        "997",
        doc(
          4,
          "Acme expects a 997 Functional Acknowledgment for every 990, 214 and 210 within 4 hours.",
          "4.2",
        ),
      ),
      value(
        `${prefix}ackWindow`,
        `${name} acknowledgment window`,
        "240 min",
        doc(
          4,
          "Acme expects a 997 Functional Acknowledgment for every 990, 214 and 210 within 4 hours.",
          "4.2",
        ),
      ),
    );
  }
  return rows;
}

function documentProfiles(revised) {
  const status = "Inactive";
  const profile = (set, direction, group, tpl, ack, withControl) => {
    const fields = setFields(set, direction, group, ack);
    return {
      transactionSet: set,
      direction,
      profileId: revised ? `edidp_acme_${set}` : "",
      name: `Acme Retail Group ${set} ${direction.toLowerCase()}`,
      status,
      change: revised ? (withControl ? "Change" : "Same") : "Add",
      fields: revised ? fields.map((entry) => same(entry)) : fields,
      template: revised ? { ...tpl, change: "Same", current: tpl.name } : tpl,
      controlNumbers: withControl ? control(revised) : [],
    };
  };
  const ms1Template = revised
    ? template("Shipment status 214", 5, [])
    : template("Shipment status 214", 4, [MS1_UNMET]);
  return [
    profile("204", "Inbound", "SM", template("Load tender 204", 6, []), false, false),
    profile("990", "Outbound", "GF", template("Tender response 990", 3, []), true, true),
    profile("214", "Outbound", "QM", ms1Template, true, true),
    profile("210", "Outbound", "IM", template("Freight invoice 210", 4, []), true, true),
  ];
}

function mappings(revised) {
  const row = (entityType, sourceId, sourceLabel, targetId, targetLabel, source) => ({
    entityType,
    sourceId,
    sourceLabel,
    targetId,
    targetLabel,
    currentTargetId: revised ? targetId : "",
    change: revised ? "Same" : "Add",
    status: "Accepted",
    confidence: 0.93,
    source,
    reason: "",
  });
  return [
    row(
      "ServiceType",
      "STD",
      "Standard",
      "svct_standard",
      "Standard",
      doc(6, "7.2 Service levels (L5): STD Standard, EXP Expedited.", "7.2"),
    ),
    row(
      "ServiceType",
      "EXP",
      "Expedited",
      "svct_expedited",
      "Expedited",
      doc(6, "7.2 Service levels (L5): STD Standard, EXP Expedited.", "7.2"),
    ),
    row(
      "ShipmentType",
      "CL",
      "Complete Load",
      "shpt_ftl",
      "Full truckload",
      doc(6, "7.1 Stop reason codes (S5-02): CL Complete Load, CU Complete Unload.", "7.1"),
    ),
    row(
      "Customer",
      "ACMERETAIL",
      "Acme Retail Group",
      "cus_acme_retail",
      "Acme Retail Group",
      doc(1, "Trading Partner Code: ACMERETAIL", "1"),
    ),
  ];
}

function samples(revised) {
  const check = (field, label, expected) => ({
    field,
    label,
    expected,
    found: expected,
    matches: true,
  });
  return [
    {
      page: 7,
      transactionSet: "204",
      direction: "Inbound",
      parsed: true,
      error: "",
      overview: {
        x12Version: "004010",
        summary: { segmentCount: 13, errorCount: 0, warningCount: 0 },
      },
      checks: [
        check("isaSenderId", "Sender", "01 / 007912345"),
        check("isaReceiverId", "Receiver", "02 / TRNV"),
        check("version", "Version", "00401"),
        check("usage", "Usage indicator", "T"),
        check("separators", "Separators", "* ~ >"),
      ],
      unmapped: revised
        ? []
        : [
            { entityType: "Location", sourceId: "ACME DC 14", sourceLabel: "Shipper" },
            { entityType: "Location", sourceId: "ACME STORE 220", sourceLabel: "Consignee" },
          ],
    },
  ];
}

const GAPS = [
  {
    kind: "Ambiguous",
    summary:
      "The 210 charge codes for detention and other accessorials are in Appendix B, which this guide does not include.",
    source: doc(6, "Appendix B is distributed separately to certified carriers.", "8"),
    problem: "",
  },
];

function heldBlockers() {
  return ["Decide AS2 endpoint URL"];
}

/**
 * The draft at one stage of the conversation:
 *   held     — first draft, the AS2 endpoint held between production and test
 *   ready    — drafted again with the person's answer, ready to approve
 *   revised  — version 4.0 of the guide, a change set against the saved partner
 */
export function ediConnectionDraft(stage) {
  const revised = stage === "revised";
  const guide = revised ? REVISED_GUIDE : GUIDE;
  const profiles = documentProfiles(revised);
  const transportProfile = transport(stage);
  const held = stage === "held";
  return {
    documentId: guide.documentId,
    fileName: guide.fileName,
    pageCount: 7,
    documentDigest: guide.digest,
    partner: {
      match: revised ? "Existing" : "New",
      partnerId: revised ? "edip_acme_retail" : "",
      version: revised ? 4 : 0,
      status: revised ? "Active" : "Inactive",
      reasons: revised ? ["Code ACMERETAIL", "ISA ID 007912345", "AS2 ID ACMERETAILAS2"] : [],
      candidates: [],
      customerId: "cus_acme_retail",
      change: revised ? "Change" : "Add",
      fields: partnerFields(revised),
    },
    connection: {
      method: "AS2",
      inbound: true,
      outbound: true,
      sets: [
        { transactionSet: "204", direction: "Inbound" },
        { transactionSet: "990", direction: "Outbound" },
        { transactionSet: "214", direction: "Outbound" },
        { transactionSet: "210", direction: "Outbound" },
      ],
    },
    interchange: interchange(revised),
    transport: {
      ...transportProfile,
      fields: transportProfile.fields.filter((field) => field.status !== "Held"),
    },
    documentProfiles: profiles,
    mappings: mappings(revised),
    mappingCount: 4,
    truncated: false,
    kept: revised ? [{ record: "Mapping", label: "Location ACME DC 9 (Joliet)" }] : [],
    samples: samples(revised),
    gaps: GAPS,
    decisions: held ? transportProfile.fields.filter((field) => field.status === "Held") : [],
    failures: [],
    blockers: held ? heldBlockers() : [],
    changes: revised ? 6 : 14,
    ready: !held,
  };
}

export const EDI_GUIDES = { original: GUIDE, revised: REVISED_GUIDE };
