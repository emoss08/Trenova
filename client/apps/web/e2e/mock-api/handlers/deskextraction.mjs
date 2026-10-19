// A Desk conversation about a file a person attached: the agent reads it with
// intake_get_draft and the extraction artifact shows what was read. The
// `extraction` scenario field picks the file and how it was read (see
// EXTRACTION_VARIANTS), and `extractionStage` how far the shipment has got:
// "idle" (not asked for), "proposed" (waiting on approval), "created" (the
// approval ran), "failed" (it ran and failed), "rejected", or "attached" (the
// draft was already a shipment when it was read), or "replying" (the
// conversation has a reply under way on something else).
import {
  EXTRACTION_AGENT_ID,
  EXTRACTION_PROPOSAL_ID,
  EXTRACTION_SHIPMENT_ID,
  EXTRACTION_THREAD_ID,
  EXTRACTION_VARIANTS,
  documentPdf,
  extractionDocument,
  extractionPayload,
  pageSvg,
} from "../fixtures/desk-extraction.mjs";

const THREAD = EXTRACTION_THREAD_ID;
const STAGES = ["idle", "proposed", "created", "failed", "rejected", "attached", "replying"];
const SLUG = "northline-rate-confirmation";
const ARTIFACT_ID = "aart_extract_ratecon";

function variantOf(state) {
  const variant = state.scenario.extraction ?? "ratecon";
  return EXTRACTION_VARIANTS.includes(variant) ? variant : "ratecon";
}

function stageOf(state) {
  const stage = state.scenario.extractionStage ?? "idle";
  return STAGES.includes(stage) ? stage : "idle";
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

function question(document) {
  return document.contentType === "application/pdf"
    ? "Northline sent this rate con over. What's on it?"
    : "Here's the BOL the driver photographed. What does it say?";
}

const CREATE_ASK = (document) =>
  `Create a shipment from ${document.fileName} (document ${document.documentId}), using what was read from it.`;

function messages(state) {
  const document = extractionDocument(variantOf(state));
  const stage = stageOf(state);
  const at = state.anchor - 12 * 60;
  const title = document.contentType === "application/pdf" ? "Rate confirmation" : "Bill of lading";
  const list = [
    message("amsg_ex_u1", 1, "User", question(document), at, {
      attachments: [
        {
          documentId: document.documentId,
          fileName: document.fileName,
          contentType: document.contentType,
          fileSize: document.fileSize,
        },
      ],
    }),
    message("amsg_ex_t1_calls", 2, "Assistant", "", at + 8, {
      toolCalls: [
        {
          id: "call_ex_draft",
          name: "intake_get_draft",
          arguments: { documentId: document.documentId },
          effect: "lookup",
        },
      ],
    }),
    message("amsg_ex_t1_r1", 3, "Tool", "{}", at + 9, {
      toolCallId: "call_ex_draft",
      toolName: "intake_get_draft",
      effect: "lookup",
      toolVerdict: "ran",
      summary: `Read ${document.fileName}`,
    }),
    message(
      "amsg_ex_a1",
      4,
      "Assistant",
      `Here's what I read off it: [${title} · ${document.fileName}](artifact:${ARTIFACT_ID}). Most of it is clear; a few values need a look before it becomes a shipment.`,
      at + 20,
    ),
  ];
  if (stage === "idle" || stage === "attached" || stage === "replying") return list;
  const asked = state.anchor - 4 * 60;
  list.push(
    message("amsg_ex_u2", 5, "User", CREATE_ASK(document), asked),
    message("amsg_ex_t2_calls", 6, "Assistant", "", asked + 6, {
      toolCalls: [
        {
          id: "call_ex_create",
          name: "shipment_create",
          arguments: { sourceDocumentId: document.documentId },
          effect: "change",
        },
      ],
    }),
    message("amsg_ex_t2_r1", 7, "Tool", "{}", asked + 7, {
      toolCallId: "call_ex_create",
      toolName: "shipment_create",
      effect: "change",
      toolVerdict: "proposed",
      summary: "Proposed creating the shipment",
    }),
    message(
      "amsg_ex_a2",
      8,
      "Assistant",
      "I've drafted the shipment from the rate confirmation. It's waiting on your approval below.",
      asked + 14,
    ),
  );
  return list;
}

function proposals(state) {
  const stage = stageOf(state);
  if (stage === "idle" || stage === "attached" || stage === "replying") return [];
  const document = extractionDocument(variantOf(state));
  const at = state.anchor - 4 * 60 + 6;
  const status = {
    proposed: "Pending",
    created: "Executed",
    failed: "ExecutionFailed",
    rejected: "Rejected",
  }[stage];
  return [
    {
      id: EXTRACTION_PROPOSAL_ID,
      runId: "",
      toolName: "shipment_create",
      arguments: { sourceDocumentId: document.documentId, proNumber: "" },
      rationale: "Create the shipment read from Northline's rate confirmation.",
      autonomyTier: "Propose",
      status,
      sourceMessageId: "amsg_ex_t2_calls",
      confidence: 0.86,
      executedAt: stage === "created" || stage === "failed" ? state.anchor - 60 : null,
      executionError: stage === "failed" ? "The customer Northline Logistics is inactive." : "",
      executionResult:
        stage === "created"
          ? {
              action: "Created",
              kind: "shipment",
              name: "S-104260",
              ids: { shipment: EXTRACTION_SHIPMENT_ID },
              record: { entityType: "shipment", id: EXTRACTION_SHIPMENT_ID },
            }
          : null,
      expiresAt: state.anchor + 7 * 86400,
      fields: [],
      agentId: EXTRACTION_AGENT_ID,
      agentName: "Dispatch",
      createdAt: at,
      decisionNote: "",
    },
  ];
}

function artifacts(state) {
  const variant = variantOf(state);
  const document = extractionDocument(variant);
  const createdAt = state.anchor - 12 * 60 + 20;
  const kind = document.contentType === "application/pdf" ? "Rate confirmation" : "Bill of lading";
  return [
    {
      id: ARTIFACT_ID,
      threadId: THREAD,
      messageId: "amsg_ex_a1",
      runId: "",
      proposalId: "",
      planId: "",
      kind: "extraction",
      status: "Ready",
      title: `${kind} · ${document.fileName}`,
      payload: extractionPayload(variant, stageOf(state) === "attached"),
      sourceToolCallId: "call_ex_draft",
      pinned: false,
      lineageId: "",
      lineageSeq: 1,
      slug: SLUG,
      turn: question(document),
      createdAt,
      updatedAt: createdAt,
    },
  ];
}

function thread(state) {
  const list = messages(state);
  const document = extractionDocument(variantOf(state));
  return {
    id: THREAD,
    businessUnitId: "bu_mock",
    organizationId: "org_mock",
    userId: "usr_mock",
    agentDefinitionId: EXTRACTION_AGENT_ID,
    title: question(document),
    status: "Active",
    lastMessageAt: list.at(-1).createdAt,
    lastReadAt: state.anchor,
    origin: "Desk",
    pinned: false,
    canContinue: true,
    version: 1,
    createdAt: list[0].createdAt,
    updatedAt: list.at(-1).createdAt,
  };
}

/** The conversation this file serves, for the rail's list and shelf counts. */
export function extractionThreads(state) {
  return [thread(state)];
}

function documentFor(path) {
  const id = path.match(/\/documents\/(doc_[A-Za-z0-9]+)\//)?.[1] ?? "";
  for (const variant of EXTRACTION_VARIANTS) {
    const document = extractionDocument(variant);
    if (document.documentId === id) return document;
  }
  return null;
}

const NOT_FOUND = {
  __status: 404,
  body: {
    type: "not-found",
    title: "Not Found",
    status: 404,
    detail: "This page is not in the document",
  },
};

/** One page drawn as the server draws it; the "broken" variant fails every page. */
function pageImage(state, path) {
  const document = documentFor(path);
  const page = Number(path.match(/\/pages\/(\d+)\/image/)?.[1] ?? 0);
  if (!document || page < 1) return NOT_FOUND;
  if (variantOf(state) === "broken") {
    return {
      __status: 500,
      body: {
        type: "internal",
        title: "Internal Server Error",
        status: 500,
        detail: "render failed",
      },
    };
  }
  const svg = pageSvg(document, page);
  if (svg === null) return NOT_FOUND;
  return { __raw: svg, contentType: "image/svg+xml" };
}

/** The file as document intelligence read it: its pages, indexed. */
function documentReading(state, path) {
  const document = documentFor(path);
  if (!document) return NOT_FOUND;
  const at = state.anchor - 13 * 60;
  return {
    id: `dcon_${document.documentId}`,
    documentId: document.documentId,
    organizationId: "org_mock",
    businessUnitId: "bu_mock",
    status: "Indexed",
    contentText: "",
    pageCount: document.pages.length,
    sourceKind: document.contentType === "application/pdf" ? "Native" : "OCR",
    detectedLanguage: "en",
    detectedDocumentKind:
      document.contentType === "application/pdf" ? "RateConfirmation" : "BillOfLading",
    classificationConfidence: 0.94,
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

/** The upload's stored preview of page 1, which the card shows while the full page is drawn. */
function documentPreview(state, path) {
  const document = documentFor(path);
  if (!document || variantOf(state) === "broken") return NOT_FOUND;
  return { __raw: pageSvg(document, 1), contentType: "image/svg+xml" };
}

/** Every reply under way across the person's conversations: this one's, while it is replying. */
function activeTurns(state) {
  if (stageOf(state) !== "replying") return { items: [] };
  return {
    items: [
      {
        turnId: "atrn_extract_live",
        threadId: THREAD,
        threadTitle: thread(state).title,
        origin: "Desk",
        startedAt: state.anchor - 20,
      },
    ],
  };
}

function documentContent(_state, path) {
  const document = documentFor(path);
  if (!document) return NOT_FOUND;
  if (document.contentType === "application/pdf") {
    return { __raw: documentPdf(document), contentType: "application/pdf" };
  }
  return { __raw: pageSvg(document, 1), contentType: "image/svg+xml" };
}

function budget(state) {
  const tomorrow = Math.floor(state.anchor / 86400) * 86400 + 86400;
  return {
    agentName: "Dispatch",
    spentUsd: "3.10",
    limitUsd: "100.00",
    share: 0.031,
    near: false,
    monthStart: Math.floor(state.anchor / 86400) * 86400 - 15 * 86400,
    resetsAt: tomorrow + 15 * 86400,
    runsToday: 2,
    dailyRunLimit: 100,
    budgetUsed: false,
    dailyUsed: false,
    dayResetsAt: tomorrow,
    disabledBy: "",
    disabledAt: 0,
    person: { used: 4, limit: 500, resetsAt: tomorrow },
  };
}

const route = (rest = "") => new RegExp(`^/api/v1/assistant/threads/${THREAD}${rest}/?$`);

export const EXTRACTION_ROUTES = [
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
  ["GET", /^\/api\/v1\/assistant\/turns\/active\/?$/, activeTurns],
  ["GET", /^\/api\/v1\/documents\/doc_01JR[A-Z0-9]+\/pages\/\d+\/image\/?$/, pageImage],
  ["GET", /^\/api\/v1\/documents\/doc_01JR[A-Z0-9]+\/preview\/?$/, documentPreview],
  ["GET", /^\/api\/v1\/documents\/doc_01JR[A-Z0-9]+\/content\/?$/, documentReading],
  ["GET", /^\/api\/v1\/documents\/doc_01JR[A-Z0-9]+\/(view|download)\/?$/, documentContent],
];

export { EXTRACTION_SHIPMENT_ID };
