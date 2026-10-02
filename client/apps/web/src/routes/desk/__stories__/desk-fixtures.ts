/* eslint-disable */
// Fixture data for the Desk stories: one organization's morning, as the
// transcript the Desk was redesigned around left it. Nothing here is read by
// the app; it is served to it by the story harness.
import type {
  AssistantArtifact,
  AssistantMessage,
  AssistantPlan,
  AssistantProposal,
  AssistantThread,
} from "@/types/assistant";

export const NOW = 1_791_000_000; // 2026-10-02T04:00:00Z-ish, Unix seconds
const H = 3600;
const D = 24 * H;

export const USER = {
  id: "usr_01M3Q2Y2Y67TNVTDWZC69MJC4Y",
  version: 1,
  createdAt: NOW - 90 * D,
  updatedAt: NOW,
  businessUnitId: "bu_1",
  currentOrganizationId: "org_1",
  status: "Active",
  name: "Avery Lane",
  username: "avery",
  emailAddress: "avery@trenova.app",
  profilePicUrl: "",
  thumbnailUrl: "",
  timezone: "America/Chicago",
  timeFormat: "12-hour",
  locale: "en-US",
  isLocked: false,
  mustChangePassword: false,
};

const starters = (...labels: string[]) => labels.map((label) => ({ label, prompt: label }));

export const AGENTS = [
  {
    id: "agdef_billing",
    name: "Billing Specialist",
    description: "Gets delivered shipments invoiced correctly and on time.",
    template: "BillingAssistant",
    icon: "receipt",
    accent: "teal",
    toolNames: ["list_billing_queue_items", "get_billing_queue_items", "approve_billing_queue_items", "post_invoices"],
    systemKey: "",
    starters: starters(
      "Which billing items are stuck the longest?",
      "Post the clean items in the queue",
      "Which invoices are past due?",
      "What is ready to transfer to billing?",
    ),
    delegates: [
      { id: "agdef_exceptions", name: "Billing exceptions", icon: "receipt", accent: "amber", template: "BillingException" },
    ],
  },
  {
    id: "agdef_dispatch",
    name: "Dispatch desk",
    description: "Covers open loads, assigns drivers and watches the day's moves.",
    template: "DispatchAssistant",
    icon: "truck",
    accent: "indigo",
    toolNames: ["search_shipments", "assign_move"],
    systemKey: "",
    starters: starters("Which carriers could cover an open load?", "Where is SEED-SHP-001?", "Quote a shipment"),
    delegates: [],
  },
  {
    id: "agdef_exceptions",
    name: "Billing exceptions",
    description: "Works held and excepted items so they can bill.",
    template: "BillingException",
    icon: "receipt",
    accent: "amber",
    toolNames: ["list_billing_queue_items"],
    systemKey: "",
    starters: starters("Blocked invoices this week"),
    delegates: [],
  },
  {
    id: "agdef_compliance",
    name: "Compliance desk",
    description: "Driver qualification, hours of service and expiring documents.",
    template: "ComplianceAssistant",
    icon: "shield",
    accent: "violet",
    toolNames: ["list_expiring_credentials"],
    systemKey: "",
    starters: starters("Who has a medical card expiring this month?"),
    delegates: [],
  },
];

function thread(id: string, agent: string, title: string, ago: number, extra: Partial<AssistantThread> = {}): AssistantThread {
  return {
    id,
    businessUnitId: "bu_1",
    organizationId: "org_1",
    userId: USER.id,
    agentDefinitionId: agent,
    title,
    status: "Active",
    lastMessageAt: NOW - ago,
    preferredProviderId: "",
    origin: "Desk",
    pinned: false,
    subjectType: "",
    subjectId: "",
    canContinue: true,
    version: 1,
    createdAt: NOW - ago - 600,
    updatedAt: NOW - ago,
    ...extra,
  } as AssistantThread;
}

export const THREAD_ID = "athr_01M3XD9H45PP13TE646125ZR1A";

export const THREADS: AssistantThread[] = [
  thread(THREAD_ID, "agdef_billing", "Check and see what shipments are eligible to be transferred", 12 * 60, { pinned: true }),
  thread("athr_2", "agdef_billing", "How many invoices have we posted this week?", 3 * H),
  thread("athr_3", "agdef_dispatch", "New load: Chicago IL to Columbus OH, 42,000 lb", 5 * H),
  thread("athr_4", "agdef_billing", "There are an assortment of invoices sitting in review", 26 * H),
  thread("athr_5", "agdef_dispatch", "Can you tell me which shipments are at risk today?", 30 * H),
  thread("athr_6", "agdef_exceptions", "Blocked invoices this week", 3 * D),
  thread("athr_7", "agdef_dispatch", "Where is SEED-SHP-001?", 4 * D),
  thread("athr_8", "agdef_compliance", "Who has a medical card expiring this month?", 6 * D),
  thread("athr_9", "agdef_billing", "Post the Acme Manufacturing drafts", 9 * D),
  thread("athr_10", "agdef_billing", "Explain the detention charge on SEED-DET-003", 12 * D),
  thread("athr_11", "agdef_dispatch", "Cover the Dallas to Memphis lane for next week", 20 * D),
  thread("athr_12", "agdef_billing", "Untitled conversation", 40 * D),
];

const queueRows = [
  ["bqi_1", "INV2610000015", "ReadyForReview", "SEED-DET-009", "Peak Distributing", "", "1,250.00"],
  ["bqi_2", "INV2610000012", "ReadyForReview", "SEED-PAY-010", "FreshHaul Foods", "", "5,750.00"],
  ["bqi_3", "INV2610000013", "ReadyForReview", "SEED-DET-003", "Peak Distributing", "", "2,950.00"],
  ["bqi_4", "INV2610000014", "ReadyForReview", "SEED-DET-007", "Range Logistics", "", "3,050.00"],
  ["bqi_5", "INV2610000016", "Approved", "SEED-DET-010", "Acme Manufacturing", "Avery Lane", "1,450.00"],
  ["bqi_6", "INV2610000006", "ReadyForReview", "SEED-PAY-004", "GlobalTrade Imports", "", "3,400.00"],
  ["bqi_7", "INV2610000007", "ReadyForReview", "SEED-PAY-005", "Acme Manufacturing", "", "5,200.00"],
  ["bqi_8", "INV2610000008", "ReadyForReview", "SEED-PAY-006", "Sunbelt Materials", "", "3,150.00"],
  ["bqi_9", "INV2610000009", "ReadyForReview", "SEED-PAY-007", "Range Logistics", "", "3,050.00"],
  ["bqi_10", "INV2610000010", "ReadyForReview", "SEED-PAY-008", "GlobalTrade Imports", "", "4,100.00"],
  ["bqi_11", "INV2610000011", "ReadyForReview", "SEED-PAY-009", "Peak Distributing", "", "2,600.00"],
  ["bqi_12", "INV2610000002", "ReadyForReview", "SEED-SHP-005", "Acme Manufacturing", "", "2,950.00"],
  ["bqi_13", "INV2610000003", "ReadyForReview", "SEED-PAY-001", "Acme Manufacturing", "", "1,450.00"],
  ["bqi_14", "INV2610000004", "ReadyForReview", "SEED-PAY-002", "FreshHaul Foods", "", "2,300.00"],
  ["bqi_15", "INV2610000005", "ReadyForReview", "SEED-PAY-003", "Peak Distributing", "", "1,980.00"],
];

function artifact(id: string, over: Partial<AssistantArtifact>): AssistantArtifact {
  return {
    id,
    threadId: THREAD_ID,
    messageId: "",
    runId: "",
    proposalId: "",
    planId: "",
    kind: "table_view",
    status: "Ready",
    title: "",
    payload: {},
    sourceToolCallId: "",
    pinned: false,
    createdAt: NOW - 20 * 60,
    updatedAt: NOW - 20 * 60,
    ...over,
  } as AssistantArtifact;
}

const queueTablePayload = (rows: typeof queueRows) => ({
  display: 1,
  tool: "list_billing_queue_items",
  entity: "billing queue items",
  recordEntity: "billing_queue_item",
  columns: ["number", "status", "proNumber", "billTo", "assignedBiller", "amount", "ageDays"],
  rows: rows.map(([id, number, status, proNumber, billTo, assignedBiller, amount]) => ({
    __recordId: id,
    number,
    status,
    proNumber,
    billTo,
    assignedBiller,
    amount,
    ageDays: 1,
  })),
  rowCount: rows.length,
  searchedFor: ["no filters: the most recent billing queue items"],
});

export const ARTIFACTS: AssistantArtifact[] = [
  artifact("art_queue", {
    title: "Billing queue items",
    sourceToolCallId: "call_list_1",
    messageId: "msg_a2",
    payload: queueTablePayload(queueRows),
    createdAt: NOW - 28 * 60,
    updatedAt: NOW - 28 * 60,
    pinned: true,
  }),
  artifact("art_bunch", {
    title: "Billing queue item (11)",
    sourceToolCallId: "call_get_1",
    messageId: "msg_a4",
    payload: {
      ...queueTablePayload(queueRows.slice(0, 11)),
      tool: "get_billing_queue_items",
      bunched: true,
      calls: Array.from({ length: 11 }, (_, i) => `call_get_${i + 1}`),
      columns: ["number", "status", "billTo", "canApprove", "approvalBlockedBy", "missingDocuments"],
      rows: queueRows.slice(0, 11).map(([id, number, status, , billTo]) => ({
        __recordId: id,
        number,
        status,
        billTo,
        canApprove: false,
        approvalBlockedBy: "an item in ReadyForReview cannot be approved; it must be in review",
        missingDocuments: "",
      })),
    },
    createdAt: NOW - 24 * 60,
    updatedAt: NOW - 24 * 60,
  }),
  artifact("art_card", {
    kind: "entity_card",
    title: "Shipment SEED-DET-003",
    sourceToolCallId: "call_ship_1",
    messageId: "msg_a4",
    payload: {
      display: 1,
      entity: "shipment",
      path: "/shipments/shp_1",
      record: {
        proNumber: "SEED-DET-003",
        bol: "BOL-2026-0203",
        status: "ReadyToInvoice",
        customer: "Peak Distributing",
        totalCharge: "2,950.00",
        actualShipDate: "2026-09-21",
        actualDeliveryDate: "2026-09-22",
        origin: "Peak Distributing DC, Fort Worth TX",
        destination: "Cold Storage, Memphis TN",
      },
    },
    createdAt: NOW - 22 * 60,
    updatedAt: NOW - 22 * 60,
  }),
  artifact("art_draft", {
    kind: "email_draft",
    title: "Invoice INV2610000016 to Acme Manufacturing",
    proposalId: "ap_draft",
    messageId: "msg_a6",
    status: "Pending",
    payload: {
      to: ["ap@acme-manufacturing.example"],
      subject: "Invoice INV2610000016 for PRO SEED-DET-010",
      body: "Hello,\n\nPlease find attached invoice INV2610000016 for the delivery completed on September 26.\n\nThank you,\nTrenova Billing",
    },
    createdAt: NOW - 16 * 60,
    updatedAt: NOW - 16 * 60,
  }),
  artifact("art_doc", {
    kind: "document",
    title: "What blocks the queue this morning",
    messageId: "msg_a6",
    payload: {
      markdown:
        "## Summary\n\nFifteen items are waiting. Eleven have no biller and cannot be approved until one is assigned.\n\n| Customer | Items | Blocked by |\n|---|---|---|\n| Peak Distributing | 4 | No biller |\n| Acme Manufacturing | 4 | No biller |\n| FreshHaul Foods | 2 | No biller |\n\n### Next step\n\nAssign a biller to every waiting item in one call, then approve the clean ones together.",
    },
    createdAt: NOW - 14 * 60,
    updatedAt: NOW - 14 * 60,
  }),
  ...Array.from({ length: 6 }, (_, i) =>
    artifact(`art_ship_${i}`, {
      title: "Shipments",
      sourceToolCallId: `call_search_${i}`,
      messageId: "msg_a8",
      payload: {
        display: 1,
        tool: "search_shipments",
        entity: "shipments",
        recordEntity: "shipment",
        columns: ["proNumber", "bol", "status", "customer", "totalCharge", "actualDeliveryDate"],
        rows: [
          {
            __recordId: `shp_${i}`,
            proNumber: `SEED-PAY-00${i + 1}`,
            bol: `BOL-2026-010${i + 1}`,
            status: "ReadyToInvoice",
            customer: ["Acme Manufacturing", "FreshHaul Foods", "Peak Distributing"][i % 3],
            totalCharge: ["1,450.00", "2,300.00", "2,950.00", "3,400.00", "5,200.00", "3,150.00"][i],
            actualDeliveryDate: "2026-09-2" + (1 + i),
          },
        ],
        rowCount: 1,
        searchedFor: [`text matching "SEED-PAY-00${i + 1}"`, "status ReadyToInvoice"],
      },
      createdAt: NOW - 10 * 60 + i * 5,
      updatedAt: NOW - 10 * 60 + i * 5,
    }),
  ),
];

let seq = 0;
function message(id: string, role: "User" | "Assistant" | "Tool", at: number, over: Partial<AssistantMessage> = {}): AssistantMessage {
  seq += 1;
  return {
    id,
    threadId: THREAD_ID,
    sequence: seq,
    role,
    kind: "Message",
    content: "",
    toolCallId: "",
    toolName: "",
    toolFailed: false,
    scopeStage: "",
    scopeCategory: "",
    scopeReason: "",
    refused: false,
    model: role === "Assistant" ? "gpt-6-luna" : "",
    inputTokens: role === "Assistant" ? 9340 : 0,
    outputTokens: role === "Assistant" ? 180 : 0,
    createdAt: at,
    ...over,
  } as AssistantMessage;
}

const T0 = NOW - 30 * 60;

export const MESSAGES: AssistantMessage[] = [
  message("msg_u1", "User", T0, { content: "Check and see what shipments are eligible to be transferred" }),
  message("msg_a1", "Assistant", T0 + 4, {
    toolCalls: [{ id: "call_cand_1", name: "list_billing_transfer_candidates", arguments: { status: "ReadyToInvoice", limit: 50 }, effect: "lookup" }],
    latencyMs: 3400,
  }),
  message("msg_t1", "Tool", T0 + 6, {
    toolCallId: "call_cand_1",
    toolName: "list_billing_transfer_candidates",
    effect: "lookup",
    summary: "0 shipments",
    content: JSON.stringify({ count: 0, items: [], note: "No shipments ready to transfer to billing matched status ReadyToInvoice." }),
  }),
  message("msg_a1b", "Assistant", T0 + 9, {
    content:
      "No eligible shipments were returned. I checked both **ReadyToInvoice** shipments and **Completed** shipments that could be marked ready before transfer; both searches returned zero candidates. Nothing was transferred.",
    latencyMs: 1300,
  }),
  message("msg_u2", "User", T0 + 150, { content: "Can you tell me what shipments are in the billing queue then?" }),
  message("msg_a2", "Assistant", T0 + 152, {
    toolCalls: [{ id: "call_list_1", name: "list_billing_queue_items", arguments: { limit: 50, sortBy: "createdAt" }, effect: "lookup" }],
    latencyMs: 2000,
  }),
  message("msg_t2", "Tool", T0 + 154, {
    toolCallId: "call_list_1",
    toolName: "list_billing_queue_items",
    effect: "lookup",
    summary: "15 billing queue items",
    content: JSON.stringify({ count: 15, columns: ["number", "status"], items: queueRows.map(([id, number, status]) => ({ id, number, status })) }),
  }),
  message("msg_a2b", "Assistant", T0 + 157, {
    content:
      "There are **15 items** in the billing queue: **14 ReadyForReview** and **1 Approved**, all queued yesterday. The numbers and bill-to customers are in the table beside this conversation; amounts were withheld at your data access.",
    latencyMs: 3000,
  }),
  message("msg_u3", "User", T0 + 400, { content: "Perfect, let's go ahead and post all of these" }),
  message("msg_a4", "Assistant", T0 + 404, {
    toolCalls: [{ id: "call_get_1", name: "get_billing_queue_items", arguments: { billingQueueItemIds: queueRows.slice(0, 11).map((r) => r[0]) }, effect: "lookup" }],
    latencyMs: 8400,
  }),
  message("msg_t4", "Tool", T0 + 410, {
    toolCallId: "call_get_1",
    toolName: "get_billing_queue_items",
    effect: "lookup",
    summary: "11 billing queue items",
    content: JSON.stringify({ count: 11, items: [] }),
  }),
  message("msg_a5", "Assistant", T0 + 412, {
    toolCalls: [{ id: "call_assign_1", name: "assign_billing_queue_billers", arguments: { billingQueueItemIds: queueRows.slice(0, 11).map((r) => r[0]) }, effect: "change" }],
    latencyMs: 2200,
  }),
  message("msg_t5", "Tool", T0 + 414, {
    toolCallId: "call_assign_1",
    toolName: "assign_billing_queue_billers",
    effect: "change",
    summary: "11 billing queue items",
    content: 'Recorded a proposal to run "assign_billing_queue_billers". It is awaiting a person\'s review at the Propose tier and has not run.',
  }),
  message("msg_a6", "Assistant", T0 + 418, {
    content:
      "Eleven of the fifteen have no biller yet, so they cannot be approved as they stand. I proposed assigning you as the biller on all eleven in one step, which moves them into review. Once you approve that, I'll approve the clean ones together and propose posting the drafts.",
    latencyMs: 4800,
  }),
  message("msg_d1", "User", T0 + 600, { kind: "DecisionNote", content: 'Approved the plan "Billing Specialist: 1 change", and the step ran.\nReport only what the note and the card hold.' }),
  message("msg_a8", "Assistant", T0 + 603, {
    toolCalls: [{ id: "call_search_0", name: "search_shipments", arguments: { query: "SEED-PAY-001", status: "ReadyToInvoice" }, effect: "lookup" }],
    latencyMs: 1800,
  }),
  message("msg_t8", "Tool", T0 + 605, {
    toolCallId: "call_search_0",
    toolName: "search_shipments",
    effect: "lookup",
    summary: "1 shipment",
    content: JSON.stringify({ count: 1 }),
  }),
  message("msg_a9", "Assistant", T0 + 610, {
    content:
      "All **11 items** are now **InReview** and assigned to you. None are posted yet; say the word and I'll propose approving the clean ones together.",
    latencyMs: 6000,
  }),
];

export const PROPOSALS: AssistantProposal[] = [
  {
    id: "ap_assign",
    runId: "run_1",
    toolName: "assign_billing_queue_billers",
    arguments: { billingQueueItemIds: queueRows.slice(0, 11).map((r) => r[0]) },
    rationale: 'Asked to assign billing queue billers in reply to: "Perfect, let\'s go ahead and post all of these"',
    autonomyTier: "Propose",
    status: "Executed",
    sourceMessageId: "msg_a5",
    confidence: 0.92,
    executedAt: T0 + 601,
    executionError: "",
    expiresAt: 0,
    planId: "",
    planStep: 0,
    fields: [],
    agentId: "agdef_billing",
    agentName: "Billing Specialist",
    createdAt: T0 + 414,
    decidedAt: T0 + 600,
    decidedByUserId: USER.id,
    decisionNote: "",
  } as AssistantProposal,
  {
    id: "ap_draft",
    runId: "run_1",
    toolName: "send_invoice",
    arguments: { invoiceId: "inv_16" },
    rationale: "The approved item made a draft; the customer takes invoices by email.",
    autonomyTier: "Propose",
    status: "Pending",
    sourceMessageId: "msg_a6",
    confidence: 0.8,
    executionError: "",
    expiresAt: NOW + 6 * D,
    planId: "",
    planStep: 0,
    fields: [],
    agentId: "agdef_billing",
    agentName: "Billing Specialist",
    createdAt: T0 + 418,
    decidedByUserId: "",
    decisionNote: "",
  } as AssistantProposal,
];

export const PLANS: AssistantPlan[] = [];

export const PENDING_SUMMARY = {
  total: 4,
  byAgent: [
    { agentDefinitionId: "agdef_billing", agentName: "Billing Specialist", count: 3 },
    { agentDefinitionId: "agdef_dispatch", agentName: "Dispatch desk", count: 1 },
  ],
  byTool: [
    { toolName: "approve_billing_queue_items", count: 2 },
    { toolName: "send_invoice", count: 1 },
    { toolName: "assign_move", count: 1 },
  ],
  oldestAt: NOW - 67 * 60,
};

export const ATTENTION = {
  billingQueue: 15,
  pendingApprovals: 2,
  reconciliationExceptions: 3,
  serviceFailures: 1,
  ediAttention: 0,
  agentDecisions: 4,
};

export const WATCHTOWER_COUNTS = {
  unresolved: 112,
  critical: 3,
  unseen: 112,
  unseenCritical: 3,
  seenAt: NOW - 2 * D,
  byKind: [
    { kind: "inbound_message", label: "Inbound messages", count: 48 },
    { kind: "edi", label: "EDI", count: 31 },
    { kind: "agent_run", label: "Agent runs", count: 20 },
    { kind: "weather", label: "Weather", count: 13 },
  ],
};

export const WATCHTOWER_ITEMS = [
  { id: "wt_1", sourceKind: "weather", sourceId: "w1", severity: "Critical", title: "Winter storm warning on I-80, Nebraska", summary: "Six loads routed through the corridor in the next 18 hours.", subjectType: "", subjectId: "", eventKind: "weather.alert", path: "/desk/watchtower", occurredAt: NOW - 50 * 60, resolvedAt: null, seen: false, kindLabel: "Weather" },
  { id: "wt_2", sourceKind: "edi", sourceId: "e1", severity: "Critical", title: "EDI 204 from Acme Manufacturing quarantined", summary: "The tender names a stop with no matching location.", subjectType: "edi_inbound_file", subjectId: "edi_1", eventKind: "edi.file_quarantined", path: "/edi/inbound/edi_1", occurredAt: NOW - 2 * H, resolvedAt: null, seen: false, kindLabel: "EDI" },
  { id: "wt_3", sourceKind: "agent_run", sourceId: "r1", severity: "Warning", title: "Billing exception agent flagged 3 items", summary: "Rate disagrees with the agreement on three Peak Distributing loads.", subjectType: "agent_run", subjectId: "run_9", eventKind: "agent.run_completed", path: "/desk/watchtower", occurredAt: NOW - 3 * H, resolvedAt: null, seen: false, kindLabel: "Agent runs" },
  { id: "wt_4", sourceKind: "inbound_message", sourceId: "m1", severity: "Info", title: "FreshHaul Foods asks about invoice INV2610000012", summary: "“Can you resend the POD with this one?”", subjectType: "inbound_message", subjectId: "im_1", eventKind: "inbound_message.classified", path: "/inbox/im_1", occurredAt: NOW - 4 * H, resolvedAt: null, seen: false, kindLabel: "Inbound messages" },
];

export const BRIEFING = {
  id: "brf_1",
  roleKey: "general",
  briefingDate: "2026-10-02",
  status: "Ready",
  headline: "Fifteen items wait on a biller and three loads sit under a storm warning.",
  narrated: false,
  readAt: null,
  sections: [
    { key: "billing", title: "Billing", summary: "15 items in the queue, 11 without a biller. Nothing posted since Tuesday.", body: "", read: false, path: "/billing/queue", items: [{ label: "Waiting on a biller", value: "11", path: "/billing/queue" }, { label: "Approved, not posted", value: "1", path: "/billing/queue" }] },
    { key: "dispatch", title: "Dispatch", summary: "Three loads route through the I-80 storm corridor tonight.", body: "", read: false, path: "/shipments", items: [{ label: "At risk", value: "3", path: "/shipments" }] },
    { key: "receivables", title: "Receivables", summary: "Acme Manufacturing is 14 days past due on two invoices.", body: "", read: false, path: "/ar", items: [{ label: "Past due", value: "$8,250", path: "/ar" }] },
  ],
};

export const LIVE_TURNS = [
  { turnId: "turn_live", threadId: "athr_3", threadTitle: "New load: Chicago IL to Columbus OH, 42,000 lb", origin: "Person", startedAt: NOW - 40 },
];

export const PROVIDERS = [
  { id: "prov_1", name: "OpenAI", model: "gpt-6-luna", kind: "openai" },
];
