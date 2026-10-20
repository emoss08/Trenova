// A Desk conversation with the EDI desk setting up a trading partner from its
// implementation guide. The `ediDraft` scenario field picks how far it has
// got: "held" (the first draft, one value left for the person), "ready" (drafted
// again with their answer), "proposed" (saving it waits on the approval card)
// or "revised" (a later guide, drafted as a change set against the partner).
// The draft each artifact carries is in fixtures/desk-edi-draft.mjs.
import {
  EDI_DRAFT_AGENT_ID,
  EDI_DRAFT_PROPOSAL_ID,
  EDI_DRAFT_THREAD_ID,
  EDI_GUIDES,
  ediConnectionDraft,
} from "../fixtures/desk-edi-draft.mjs";

const THREAD = EDI_DRAFT_THREAD_ID;
const STAGES = ["held", "ready", "proposed", "revised"];
const FIRST_QUESTION =
  "Acme Retail sent over their EDI guide. Can you set them up as a trading partner?";
const REVISED_QUESTION = "Acme sent a revised EDI guide, version 4.0. Update their setup to match.";
const RESOLUTION = `For the EDI connection drafted from ${EDI_GUIDES.original.fileName}, use "https://as2-test.acmeretail.example/as2/receive" for AS2 endpoint URL (page 2) and draft it again.`;
const PROPOSE = `Propose saving the EDI connection drafted from ${EDI_GUIDES.original.fileName}, with document digest ${EDI_GUIDES.original.digest}.`;

function stageOf(state) {
  const stage = state.scenario.ediDraft ?? "ready";
  return STAGES.includes(stage) ? stage : "ready";
}

function message(id, sequence, role, content, createdAt, extra = {}) {
  return {
    id,
    threadId: THREAD,
    sequence,
    role,
    content,
    kind: "Message",
    toolFailed: false,
    refused: false,
    createdAt,
    ...extra,
  };
}

function attachment(guide) {
  return {
    documentId: guide.documentId,
    fileName: guide.fileName,
    contentType: "application/pdf",
    fileSize: guide.fileSize,
  };
}

/** One assistant step that calls tools, and each call's result, as the runtime saves them. */
function toolTurn(prefix, sequence, at, calls) {
  const asking = message(`${prefix}_calls`, sequence, "Assistant", "", at, {
    toolCalls: calls.map((call) => ({
      id: call.id,
      name: call.name,
      arguments: call.args,
      effect: call.effect,
    })),
  });
  const results = calls.map((call, index) =>
    message(`${prefix}_r${index + 1}`, sequence + index + 1, "Tool", call.content ?? "{}", at, {
      toolCallId: call.id,
      toolName: call.name,
      effect: call.effect,
      toolVerdict: call.verdict ?? "ran",
      summary: call.summary,
    }),
  );
  return [asking, ...results];
}

function artifactLink(id, title) {
  return `[${title}](artifact:${id})`;
}

const ARTIFACTS = {
  held: {
    id: "aart_edi_acme_1",
    slug: "acme-retail-connection",
    title: "EDI connection · Acme Retail Group",
  },
  ready: {
    id: "aart_edi_acme_2",
    slug: "acme-retail-connection-2",
    title: "EDI connection · Acme Retail Group",
  },
  revised: {
    id: "aart_edi_acme_3",
    slug: "acme-retail-connection-v4",
    title: "EDI connection · Acme Retail Group, revised",
  },
};

function draftCalls(guide, callId, extra = {}) {
  return {
    id: callId,
    name: "edi_draft_connection",
    effect: "lookup",
    args: { documentId: guide.documentId, ...extra },
  };
}

function firstTurn(anchor) {
  const guide = EDI_GUIDES.original;
  const at = anchor - 14 * 60;
  return [
    message("amsg_edi_u1", 1, "User", FIRST_QUESTION, at, { attachments: [attachment(guide)] }),
    ...toolTurn("amsg_edi_t1", 2, at + 20, [
      {
        id: "call_edi_read",
        name: "agent_read_attachment",
        effect: "lookup",
        args: { documentId: guide.documentId },
        summary: `Read 7 pages of ${guide.fileName}`,
      },
      {
        id: "call_edi_svct",
        name: "masterdata_list_service_types",
        effect: "lookup",
        args: { query: "standard expedited" },
        summary: "Standard and Expedited",
      },
      {
        id: "call_edi_cust",
        name: "customer_list",
        effect: "lookup",
        args: { query: "Acme Retail" },
        summary: "Acme Retail Group",
      },
      {
        ...draftCalls(guide, "call_edi_draft_1", {
          values: 31,
          requirements: 14,
          mappings: 4,
          gaps: 1,
        }),
        summary: "Drafted a new trading partner; 1 value to decide",
      },
    ]),
    message(
      "amsg_edi_a1",
      7,
      "Assistant",
      `I've drafted **Acme Retail Group** as a new trading partner from their guide: ${artifactLink(ARTIFACTS.held.id, ARTIFACTS.held.title)}. It covers the AS2 connection, the 204 they tender and the 990, 214 and 210 we send back, with every value tied to the page it came from.\n\nOne thing the guide leaves open: it gives a **production** AS2 URL and a **test** one, both in section 2.1. It also says to begin in the test environment, but that's your call, not mine. Which endpoint should the connection send to?`,
      at + 48,
    ),
  ];
}

function secondTurn(anchor) {
  const guide = EDI_GUIDES.original;
  const at = anchor - 9 * 60;
  return [
    message("amsg_edi_u2", 8, "User", RESOLUTION, at),
    ...toolTurn("amsg_edi_t2", 9, at + 6, [
      {
        ...draftCalls(guide, "call_edi_draft_2", {
          values: 31,
          requirements: 14,
          mappings: 4,
          gaps: 1,
          resolutions: 1,
        }),
        summary: "Drafted a new trading partner; ready to approve",
      },
    ]),
    message(
      "amsg_edi_a2",
      11,
      "Assistant",
      `Drafted again with the test endpoint, and it's ready to approve: ${artifactLink(ARTIFACTS.ready.id, ARTIFACTS.ready.title)}.\n\nBefore Acme can trade with you:\n\n- **MS1-01 (page 5)**: Acme marks MS1 mandatory on every 214, and the starter 214 template doesn't write it. That's a change in the template designer.\n- **Appendix B (page 6)**: the 210 charge codes for detention and accessorials aren't in this guide, so ask Acme for them.\n- **Credentials**: Acme's signing and encryption certificates, plus your own certificate and its private key, go on the AS2 profile before you turn it on.\n\nThe guide printed a keystore password for testing; I didn't take it, and nothing secret is in the draft. Everything is saved inactive, and nothing is saved until you approve it.`,
      at + 22,
    ),
  ];
}

function thirdTurn(anchor) {
  const at = anchor - 2 * 60;
  return [
    message("amsg_edi_u3", 12, "User", PROPOSE, at),
    ...toolTurn("amsg_edi_t3", 13, at + 5, [
      {
        id: "call_edi_apply",
        name: "edi_apply_connection_draft",
        effect: "change",
        verdict: "proposed",
        args: {
          documentId: EDI_GUIDES.original.documentId,
          documentDigest: EDI_GUIDES.original.digest,
        },
        summary: "Proposed saving the connection",
      },
    ]),
    message(
      "amsg_edi_a3",
      15,
      "Assistant",
      "I've proposed saving it. The approval below shows each record that will be written, all of them inactive: the partner, the AS2 profile, a document profile for each of the four document types, the interchange control numbers starting at 1000, and the code mappings.",
      at + 12,
    ),
  ];
}

function revisedTurn(anchor) {
  const guide = EDI_GUIDES.revised;
  const at = anchor - 6 * 60;
  return [
    message("amsg_edi_v1", 1, "User", REVISED_QUESTION, at, { attachments: [attachment(guide)] }),
    ...toolTurn("amsg_edi_vt", 2, at + 15, [
      {
        id: "call_edi_vread",
        name: "agent_read_attachment",
        effect: "lookup",
        args: { documentId: guide.documentId },
        summary: `Read 7 pages of ${guide.fileName}`,
      },
      {
        ...draftCalls(guide, "call_edi_vdraft", {
          ediPartnerId: "edip_acme_retail",
          values: 31,
          requirements: 14,
          mappings: 4,
          gaps: 1,
        }),
        summary: "Drafted 6 changes to Acme Retail Group",
      },
    ]),
    message(
      "amsg_edi_va",
      5,
      "Assistant",
      `Version 4.0 changes six things on the partner you already have: ${artifactLink(ARTIFACTS.revised.id, ARTIFACTS.revised.title)}.\n\n- Contact name from Dana Whitfield to **Priya Raman** (page 1), with her email and phone\n- Interchange control numbers on the 990, 214 and 210 outbound start at **5000** instead of 1000 (page 3)\n\nMS1 is now optional on the 214 (page 5), so the starter template meets every rule. Everything else is the same as you have it, nothing is turned off, and no credentials change.`,
      at + 40,
    ),
  ];
}

function messages(state) {
  const stage = stageOf(state);
  const anchor = state.anchor;
  if (stage === "revised") return revisedTurn(anchor);
  const out = firstTurn(anchor);
  if (stage === "held") return out;
  out.push(...secondTurn(anchor));
  if (stage === "proposed") out.push(...thirdTurn(anchor));
  return out;
}

function artifact(state, key, stage, messageId, callId, turn, createdAt) {
  const meta = ARTIFACTS[key];
  return {
    id: meta.id,
    threadId: THREAD,
    messageId,
    runId: "",
    proposalId: "",
    planId: "",
    kind: "edi_connection_draft",
    status: "Ready",
    title: meta.title,
    payload: ediConnectionDraft(stage),
    sourceToolCallId: callId,
    pinned: false,
    lineageId: "",
    lineageSeq: 1,
    slug: meta.slug,
    turn,
    createdAt,
    updatedAt: createdAt,
  };
}

function artifacts(state) {
  const stage = stageOf(state);
  const anchor = state.anchor;
  if (stage === "revised") {
    return [
      artifact(
        state,
        "revised",
        "revised",
        "amsg_edi_va",
        "call_edi_vdraft",
        REVISED_QUESTION,
        anchor - 6 * 60 + 40,
      ),
    ];
  }
  const out = [
    artifact(
      state,
      "held",
      "held",
      "amsg_edi_a1",
      "call_edi_draft_1",
      FIRST_QUESTION,
      anchor - 14 * 60 + 48,
    ),
  ];
  if (stage !== "held") {
    out.push(
      artifact(
        state,
        "ready",
        "ready",
        "amsg_edi_a2",
        "call_edi_draft_2",
        RESOLUTION,
        anchor - 9 * 60 + 22,
      ),
    );
  }
  return out;
}

function proposals(state) {
  if (stageOf(state) !== "proposed") return [];
  const at = state.anchor - 2 * 60 + 5;
  return [
    {
      id: EDI_DRAFT_PROPOSAL_ID,
      runId: "arun_edi_acme",
      toolName: "edi_apply_connection_draft",
      arguments: {
        documentId: EDI_GUIDES.original.documentId,
        customerId: "cus_acme_retail",
        values: Array.from({ length: 31 }, (_, index) => ({ field: `field_${index}` })),
        requirements: Array.from({ length: 14 }, (_, index) => ({ segment: `seg_${index}` })),
        mappings: Array.from({ length: 4 }, (_, index) => ({ sourceId: `code_${index}` })),
        gaps: [{ kind: "Ambiguous" }],
        resolutions: [{ field: "as2.endpointUrl", source: "Candidate" }],
        documentDigest: EDI_GUIDES.original.digest,
      },
      rationale:
        "Save the trading partner drafted from Acme Retail Group's EDI guide, as reviewed, all of it inactive.",
      autonomyTier: "Propose",
      status: "Pending",
      sourceMessageId: "amsg_edi_t3_calls",
      confidence: 0.94,
      expiresAt: state.anchor + 7 * 86400,
      fields: [],
      agentId: EDI_DRAFT_AGENT_ID,
      agentName: "EDI",
      createdAt: at,
      decisionNote: "",
    },
  ];
}

function field(path, label, after, valueType = "text") {
  return {
    path,
    label,
    valueType,
    before: null,
    after,
    beforeRef: null,
    afterRef: null,
    withheld: false,
    volatile: false,
    truncated: false,
    changedSinceProposed: false,
    proposedBefore: null,
    projectedFromStep: 0,
  };
}

function created(resource, label, fields) {
  return {
    resource,
    record: null,
    entityId: null,
    label,
    operation: "Create",
    version: null,
    withheld: false,
    dependsOnStep: 0,
    fields,
    omittedFields: 0,
    message: null,
    money: null,
  };
}

/** The proposal's preview: every record the save would create, recomputed from the guide as it is now. */
function preview(state) {
  const profile = (set, direction, template) =>
    created("edi_partner_document_profile", `Acme Retail Group ${set} ${direction.toLowerCase()}`, [
      field("transactionSet", "Document", `${set} ${direction}`),
      field("template", "Template", template),
      field("isaReceiverId", "Interchange", "01/007912345 ↔ 02/TRNV"),
      field("status", "Status", "Inactive", "status"),
    ]);
  return {
    proposalId: EDI_DRAFT_PROPOSAL_ID,
    tool: "edi_apply_connection_draft",
    summary:
      "Would set up Acme Retail Group from its guide: 7 records, all inactive until you enter the credentials and turn them on.",
    coverage: "Full",
    changes: [
      created("edi_partner", "Acme Retail Group", [
        field("code", "Code", "ACMERETAIL"),
        field("name", "Name", "Acme Retail Group"),
        field("contactName", "Contact", "Dana Whitfield · edi@acmeretail.example"),
        field("timezone", "Time zone", "America/Chicago"),
        field("status", "Status", "Inactive", "status"),
      ]),
      created("edi_communication_profile", "Acme Retail Group AS2", [
        field("method", "Method", "AS2"),
        field("endpointUrl", "Endpoint", "https://as2-test.acmeretail.example/as2/receive"),
        field("as2Ids", "AS2 IDs", "ACMERETAILAS2 ↔ TRENOVA-AS2"),
        field("mdn", "MDN", "Synchronous, signed · SHA-256 · AES-256 (CBC)"),
        field("status", "Status", "Inactive", "status"),
      ]),
      profile("204", "Inbound", "Load tender 204"),
      profile("990", "Outbound", "Tender response 990"),
      profile("214", "Outbound", "Shipment status 214"),
      profile("210", "Outbound", "Freight invoice 210"),
      created("edi_mapping_profile", "Acme Retail Group code mappings", [
        field("items", "Mappings", "STD, EXP, CL, ACMERETAIL"),
      ]),
    ],
    warnings: [],
    stale: false,
    staleness: { pinned: true, proposedVersion: 0, currentVersion: 0, missing: false },
    targetVersion: null,
    withheldCount: 0,
    omittedRecords: 0,
    recorded: false,
    computedAt: state.anchor - 60,
    digest: "sha256:4e1b7c90d2a6f3e8",
    reviewRequired: false,
  };
}

function thread(state) {
  const list = messages(state);
  const stage = stageOf(state);
  const anchor = state.anchor;
  return {
    id: THREAD,
    businessUnitId: "bu_mock",
    organizationId: "org_mock",
    userId: "usr_mock",
    agentDefinitionId: EDI_DRAFT_AGENT_ID,
    title: stage === "revised" ? REVISED_QUESTION : FIRST_QUESTION,
    status: "Active",
    lastMessageAt: list.at(-1).createdAt,
    lastReadAt: anchor,
    origin: "Desk",
    pinned: false,
    canContinue: true,
    version: 2,
    createdAt: list[0].createdAt,
    updatedAt: list.at(-1).createdAt,
  };
}

/** The conversation this file serves, for the rail's list and shelf counts. */
export function ediDraftThreads(state) {
  return [thread(state)];
}

function budget(state) {
  const tomorrow = Math.floor(state.anchor / 86400) * 86400 + 86400;
  return {
    agentName: "EDI",
    spentUsd: "6.20",
    limitUsd: "100.00",
    share: 0.062,
    near: false,
    monthStart: Math.floor(state.anchor / 86400) * 86400 - 15 * 86400,
    resetsAt: tomorrow + 15 * 86400,
    runsToday: 4,
    dailyRunLimit: 100,
    budgetUsed: false,
    dailyUsed: false,
    dayResetsAt: tomorrow,
    disabledBy: "",
    disabledAt: 0,
    person: { used: 9, limit: 500, resetsAt: tomorrow },
  };
}

/** The guide as document intelligence read it: seven pages of text, indexed. */
function documentContent(state, path) {
  const guide = Object.values(EDI_GUIDES).find((entry) => path.includes(`/${entry.documentId}/`));
  const at = state.anchor - 15 * 60;
  return {
    id: `dcon_${guide.documentId}`,
    documentId: guide.documentId,
    organizationId: "org_mock",
    businessUnitId: "bu_mock",
    status: "Indexed",
    contentText: "",
    pageCount: 7,
    sourceKind: "Native",
    detectedLanguage: "en",
    detectedDocumentKind: "Other",
    classificationConfidence: 0.92,
    structuredData: {},
    failureCode: null,
    failureMessage: null,
    version: 1,
    createdAt: at,
    updatedAt: at,
    lastExtractedAt: at,
    pages: [],
  };
}

const route = (rest = "") => new RegExp(`^/api/v1/assistant/threads/${THREAD}${rest}/?$`);

export const EDI_DRAFT_ROUTES = [
  ["GET", route(), thread],
  [
    "GET",
    route("/messages"),
    (state) => {
      const list = messages(state);
      return { results: list, hasMore: false, total: list.length, limit: 50 };
    },
  ],
  [
    "GET",
    route("/artifacts"),
    (state) => {
      const list = artifacts(state);
      return {
        results: list,
        total: list.length,
        nextCursor: "",
        counts: { all: list.length, pinned: 0, families: { work: list.length } },
      };
    },
  ],
  [
    "GET",
    route("/artifacts/by-slug/([a-z0-9-]+)"),
    (state, path) => {
      const slug = path.match(/by-slug\/([a-z0-9-]+)/)?.[1] ?? "";
      return { results: artifacts(state).filter((entry) => entry.slug === slug) };
    },
  ],
  [
    "GET",
    route("/artifacts/(aart_[a-z0-9_]+)"),
    (state, path) => {
      const id = path.match(/artifacts\/(aart_[a-z0-9_]+)/)?.[1] ?? "";
      return { results: artifacts(state).filter((entry) => entry.id === id) };
    },
  ],
  ["GET", route("/proposals"), (state) => ({ results: proposals(state) })],
  ["GET", route("/plans"), () => ({ results: [] })],
  ["GET", route("/schedules"), () => ({ items: [], total: 0 })],
  ["GET", route("/waits"), () => ({ items: [] })],
  ["GET", route("/queue"), () => ({ items: [] })],
  ["GET", route("/budget"), budget],
  ["GET", route("/turns/active"), () => ({ turn: null })],
  ["GET", /^\/api\/v1\/documents\/doc_acme_guide_v[34]\/content\/?$/, documentContent],
];

export const EDI_DRAFT_HANDLERS = {
  AgentProposalPreview: (state, variables) => {
    if (variables.id !== EDI_DRAFT_PROPOSAL_ID || stageOf(state) !== "proposed") {
      throw new Error(`no preview for ${variables.id}`);
    }
    return { agentProposalPreview: preview(state) };
  },
};
