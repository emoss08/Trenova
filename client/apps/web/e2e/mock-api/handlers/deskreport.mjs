// A Desk conversation in which its agent, Billing Desk, ran the built-in
// revenue by customer report for the last 30 days, and previewed it before
// that. The `deskReport` scenario field picks how the run stands:
// "finished" (ranked, against last week's run), "trend" (revenue by month),
// "running", "failed", "empty" (finished, nothing matched), "capped" (hit its
// row cap), "expired", "scheduled" (a schedule's run, not the agent's),
// "withheld" (the person lost access to what it shows) and "older" (finished
// before runs kept a sample).
import { deskBudget } from "./desktable.mjs";

const THREAD = "athr_desk_report";
const RUN_ARTIFACT = "aart_revenue_run";
const PREVIEW_ARTIFACT = "aart_revenue_preview";
const RUN_SLUG = "revenue-by-customer";
const PREVIEW_SLUG = "revenue-by-customer-preview";
const AGENT = { id: "agd_mock_billing_desk", name: "Billing Desk", icon: "", accent: "violet" };
const QUESTION = "Run revenue by customer for the last 30 days.";
const STAGES = [
  "finished",
  "trend",
  "running",
  "failed",
  "empty",
  "capped",
  "expired",
  "scheduled",
  "withheld",
  "older",
];
const DAY = 86400;
const TIMEZONE = "America/Chicago";

const DISPLAY = {
  style: "text",
  decimals: 0,
  grouping: true,
  currency: "",
  negative: "minus",
  notation: "standard",
  prefix: "",
  suffix: "",
  dateStyle: "",
  boolStyle: "",
  durationUnit: "",
  durationStyle: "",
  nullText: "",
  rules: [],
  band: null,
  labels: null,
};
const money = { ...DISPLAY, style: "currency", currency: "USD", decimals: 2 };
const count = { ...DISPLAY, style: "number" };
const rate = { ...DISPLAY, style: "currency", currency: "USD", decimals: 2 };
const percent = { ...DISPLAY, style: "percent", decimals: 1 };

const CUSTOMERS = [
  ["Acme Foods", 162, "241880.50", "3.31", "0.07"],
  ["Harborline Logistics", 148, "198412.00", "2.96", "0.11"],
  ["Northwind Grocers", 121, "164209.75", "3.08", "0.05"],
  ["Blue Ridge Paper", 96, "128730.00", "2.71", "0.09"],
  ["Cascade Building Supply", 84, "112455.25", "3.44", "0.14"],
  ["Lone Star Beverage", 77, "96102.00", "2.88", "0.06"],
  ["Pioneer Plastics", 61, "74318.40", "3.12", "0.08"],
  ["Delta Farm Co-op", 52, "61988.00", "2.64", "0.04"],
  ["Granite State Mills", 44, "52101.10", "2.92", "0.07"],
  ["Sunbelt Produce", 40, "48000.00", "3.01", "0.05"],
];

const RANKED_COLUMNS = [
  { id: "customer", label: "Customer", type: "string", sensitivity: null, display: DISPLAY },
  { id: "shipments", label: "Shipments", type: "int", sensitivity: null, display: count },
  {
    id: "revenue",
    label: "Total revenue",
    type: "decimal",
    sensitivity: "confidential",
    display: money,
  },
  {
    id: "revenuePerMile",
    label: "Revenue per mile",
    type: "decimal",
    sensitivity: "confidential",
    display: rate,
  },
  {
    id: "accessorialShare",
    label: "Accessorial share",
    type: "decimal",
    sensitivity: null,
    display: percent,
  },
];

/** A run's sample as `ReportPreviewFields`: detail rows only, each at level 0. */
function preview({ columns, rows, totals, truncated }) {
  return {
    columns,
    rows,
    levels: rows.map(() => 0),
    totals,
    truncated,
    cursor: null,
    timezone: TIMEZONE,
    rollup: null,
  };
}

function stageOf(state) {
  const stage = state.scenario.deskReport ?? "finished";
  return STAGES.includes(stage) ? stage : "finished";
}

/** Midnight in Chicago on the anchor's day, as the window's end is the midnight after it. */
function chicagoMidnight(anchor) {
  return Math.floor((anchor - 5 * 3600) / DAY) * DAY + 5 * 3600;
}

function windowParameter(state, days) {
  const end = chicagoMidnight(state.anchor) + DAY;
  return {
    name: "windowDays",
    label: "Window (days)",
    type: "int",
    value: days,
    isDefault: false,
    entity: null,
    windowStart: end - (days + 1) * DAY,
    windowEnd: end,
  };
}

const canceledParameter = {
  name: "includeCanceled",
  label: "Canceled shipments",
  type: "bool",
  value: false,
  isDefault: true,
  entity: null,
  windowStart: null,
  windowEnd: null,
};

function run(state, overrides = {}) {
  const queued = state.anchor - 14 * 60;
  return {
    __typename: "ReportRun",
    id: "rrun_mock_revenue",
    definitionId: null,
    revisionId: null,
    cannedKey: "revenue-by-customer",
    cannedVersion: "1.0.0",
    requestedById: "usr_mock",
    trigger: "manual",
    params: { windowDays: 30 },
    format: "xlsx",
    status: "succeeded",
    rowCount: 214,
    byteSize: 49152,
    durationMs: 3200,
    truncated: false,
    error: null,
    artifactExpiresAt: state.anchor + 7 * DAY,
    cacheHit: false,
    queuedAt: queued,
    startedAt: queued + 1,
    completedAt: queued + 4,
    version: 3,
    createdAt: queued,
    ...overrides,
  };
}

function rankedSample(rows = CUSTOMERS) {
  return preview({
    columns: RANKED_COLUMNS,
    rows: rows.map((row) => [...row]),
    totals: rows.length > 0 ? [null, 1142, "1284310.42", "3.04", "0.086"] : null,
    truncated: rows.length > 0,
  });
}

function trendSample(state) {
  const months = [0.86, 0.94, 0.78, 0.81, 0.92, 0.95, 1.02, 1.08, 1.04, 1.12, 1.19, 1.28];
  const ships = [0.82, 0.9, 0.74, 0.8, 0.88, 0.9, 0.97, 1.01, 0.99, 1.05, 1.1, 1.14];
  const start = new Date(state.anchor * 1000);
  return preview({
    columns: [
      {
        id: "month",
        label: "Month",
        type: "epoch",
        sensitivity: null,
        display: { ...DISPLAY, style: "date", dateStyle: "monthYear" },
      },
      { id: "shipments", label: "Shipments", type: "int", sensitivity: null, display: count },
      {
        id: "revenue",
        label: "Total revenue",
        type: "decimal",
        sensitivity: "confidential",
        display: money,
      },
      {
        id: "revenuePerMile",
        label: "Revenue per mile",
        type: "decimal",
        sensitivity: "confidential",
        display: rate,
      },
    ],
    rows: months.map((value, index) => {
      const month =
        Date.UTC(start.getUTCFullYear(), start.getUTCMonth() - (11 - index), 1, 6) / 1000;
      return [
        month,
        Math.round(ships[index] * 1000),
        (value * 1000000).toFixed(2),
        (2.7 + index * 0.03).toFixed(2),
      ];
    }),
    totals: [null, 11160, "11990000.00", "2.95"],
    truncated: false,
  });
}

function card(state) {
  const stage = stageOf(state);
  const base = {
    run: run(state),
    report: {
      name: "Revenue by customer",
      description:
        "Freight, accessorial and total charges per customer, with volume, miles and revenue per mile over a rolling window.",
      category: "Billing",
      definitionId: null,
      cannedKey: "revenue-by-customer",
    },
    parameters: [windowParameter(state, 30), canceledParameter],
    timezone: TIMEZONE,
    sample: rankedSample(),
    withheld: null,
    previous: {
      runId: "rrun_mock_revenue_before",
      completedAt: state.anchor - 7 * DAY,
      rowCount: 208,
      columnIds: ["customer", "shipments", "revenue", "revenuePerMile", "accessorialShare"],
      totals: [null, 1108, "1187000.00", "3.08", "0.081"],
    },
    schedule: null,
    chart: null,
    sort: [{ column: "revenue", desc: true }],
  };

  switch (stage) {
    case "trend":
      return {
        ...base,
        report: {
          ...base.report,
          name: "Revenue by month",
          description: "Month-over-month shipments, revenue, tonnage, miles and revenue per mile.",
        },
        run: run(state, { rowCount: 12, byteSize: 18432, durationMs: 5800 }),
        parameters: [windowParameter(state, 365)],
        sample: trendSample(state),
        previous: null,
        chart: { type: "line", x: "month", series: ["revenue", "shipments"] },
        sort: [{ column: "month", desc: false }],
      };
    case "running":
      return {
        ...base,
        run: run(state, {
          status: "running",
          rowCount: 0,
          byteSize: 0,
          durationMs: 0,
          completedAt: null,
          artifactExpiresAt: null,
        }),
        sample: null,
        previous: null,
      };
    case "failed":
      return {
        ...base,
        run: run(state, {
          status: "failed",
          rowCount: 0,
          byteSize: 0,
          durationMs: 30000,
          artifactExpiresAt: null,
          error: {
            code: "QUERY_TIMEOUT",
            message: "The report took longer than its 30 second limit.",
            detail: "A narrower window usually finishes.",
          },
        }),
        sample: null,
        previous: null,
      };
    case "empty":
      return {
        ...base,
        run: run(state, { rowCount: 0, byteSize: 6144, durationMs: 800 }),
        sample: rankedSample([]),
        previous: null,
      };
    case "capped":
      return {
        ...base,
        run: run(state, { rowCount: 50000, truncated: true, byteSize: 7340032, durationMs: 41600 }),
      };
    case "expired":
      return {
        ...base,
        run: run(state, {
          status: "expired",
          queuedAt: state.anchor - 15 * DAY,
          startedAt: state.anchor - 15 * DAY + 1,
          completedAt: state.anchor - 15 * DAY + 4,
          createdAt: state.anchor - 15 * DAY,
          artifactExpiresAt: state.anchor - 8 * DAY,
          rowCount: 198,
        }),
        sample: null,
        previous: null,
      };
    case "scheduled":
      return {
        ...base,
        run: run(state, { trigger: "schedule" }),
        schedule: {
          id: "rsch_mock_monday",
          cronExpression: "0 7 * * 1",
          timezone: TIMEZONE,
          nextRunAt: Math.floor(state.anchor / DAY) * DAY + 6 * DAY + 13 * 3600,
        },
      };
    case "withheld":
      return { ...base, sample: null, previous: null, withheld: "access" };
    case "older":
      return { ...base, sample: null, previous: null };
    default:
      return base;
  }
}

function runArtifact(state) {
  const stage = stageOf(state);
  return {
    id: RUN_ARTIFACT,
    threadId: THREAD,
    messageId: "amsg_report_a2",
    runId: "arun_report",
    proposalId: "",
    planId: "",
    kind: "report_run",
    status: stage === "running" ? "Pending" : stage === "failed" ? "Failed" : "Ready",
    title: stage === "trend" ? "Revenue by month" : "Revenue by customer",
    payload: {
      runId: "rrun_mock_revenue",
      reportKey: "revenue-by-customer",
      reportName: stage === "trend" ? "Revenue by month" : "Revenue by customer",
      status: card(state).run.status,
      finished: stage !== "running",
      format: "xlsx",
      ...(stage === "scheduled" ? {} : { readBy: AGENT }),
    },
    sourceToolCallId: "call_report_run",
    pinned: false,
    lineageId: "",
    lineageSeq: 1,
    slug: RUN_SLUG,
    turn: QUESTION,
    createdAt: state.anchor - 14 * 60,
    updatedAt: state.anchor - 13 * 60,
  };
}

function previewArtifact(state) {
  const columns = RANKED_COLUMNS.map(({ id, label, type, display }) => ({
    id,
    label,
    type,
    format: display.style,
  }));
  const byLabel = (row) =>
    Object.fromEntries(columns.map((column, index) => [column.label, row[index]]));
  return {
    id: PREVIEW_ARTIFACT,
    threadId: THREAD,
    messageId: "amsg_report_a1",
    runId: "arun_report",
    proposalId: "",
    planId: "",
    kind: "report_preview",
    status: "Ready",
    title: "Revenue by customer",
    payload: {
      name: "Revenue by customer",
      dataset: "invoice",
      columns,
      rows: CUSTOMERS.slice(0, 6).map(byLabel),
      totals: byLabel([null, 1142, "1284310.42", "3.04", "0.086"]),
      rowCount: 214,
      truncated: false,
      tool: "report_preview",
      reportKey: "revenue-by-customer",
      readWith: { reportKey: "revenue-by-customer", parameters: { windowDays: 30 } },
      readBy: AGENT,
    },
    sourceToolCallId: "call_report_preview",
    pinned: false,
    lineageId: "",
    lineageSeq: 1,
    slug: PREVIEW_SLUG,
    turn: QUESTION,
    createdAt: state.anchor - 16 * 60,
    updatedAt: state.anchor - 16 * 60,
  };
}

function artifacts(state) {
  return [runArtifact(state), previewArtifact(state)];
}

function message(id, sequence, role, content, createdAt) {
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
  };
}

function messages(state) {
  return [
    message("amsg_report_u1", 1, "User", QUESTION, state.anchor - 17 * 60),
    message(
      "amsg_report_a1",
      2,
      "Assistant",
      `A first look is in [the preview](artifact:${PREVIEW_SLUG}); I started the full run.`,
      state.anchor - 16 * 60,
    ),
    message(
      "amsg_report_a2",
      3,
      "Assistant",
      `[Revenue by customer](artifact:${RUN_SLUG}) has the whole result and the file.`,
      state.anchor - 13 * 60,
    ),
  ];
}

function thread(state) {
  const list = messages(state);
  return {
    id: THREAD,
    businessUnitId: "bu_mock",
    organizationId: "org_mock",
    userId: "usr_mock",
    agentDefinitionId: AGENT.id,
    title: QUESTION,
    status: "Active",
    lastMessageAt: list.at(-1).createdAt,
    lastReadAt: state.anchor,
    origin: "Desk",
    pinned: false,
    canContinue: true,
    version: 2,
    createdAt: list[0].createdAt,
    updatedAt: list.at(-1).createdAt,
  };
}

/** The conversation this file serves, for the rail's list and shelf counts. */
export function deskReportThreads(state) {
  return [thread(state)];
}

export const DESK_REPORT_HANDLERS = {
  DeskReportRunCard: (state) => ({ reportRunDeskCard: card(state) }),
};

const route = (rest = "") => new RegExp(`^/api/v1/assistant/threads/${THREAD}${rest}/?$`);

export const DESK_REPORT_ROUTES = [
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
        counts: { all: list.length, pinned: 0, families: { report: list.length } },
      };
    },
  ],
  [
    "GET",
    route("/artifacts/by-slug/([a-z0-9-]+)"),
    (state, path) => {
      const slug = path.match(/by-slug\/([a-z0-9-]+)/)?.[1] ?? "";
      return { results: artifacts(state).filter((artifact) => artifact.slug === slug) };
    },
  ],
  [
    "POST",
    route("/artifacts/(aart_[a-z0-9_]+)/run"),
    (state) => ({
      ...runArtifact(state),
      id: "aart_revenue_run_v2",
      lineageId: RUN_ARTIFACT,
      lineageSeq: 2,
      payload: {
        ...runArtifact(state).payload,
        ranByPerson: true,
        status: "queued",
        finished: false,
      },
    }),
  ],
  [
    "GET",
    route("/artifacts/(aart_[a-z0-9_]+)"),
    (state, path) => {
      const id = path.match(/artifacts\/(aart_[a-z0-9_]+)/)?.[1] ?? "";
      return { results: artifacts(state).filter((artifact) => artifact.id === id) };
    },
  ],
  ["GET", route("/proposals"), () => ({ results: [] })],
  ["GET", route("/plans"), () => ({ results: [] })],
  ["GET", route("/schedules"), () => ({ items: [], total: 0 })],
  ["GET", route("/waits"), () => ({ items: [] })],
  ["GET", route("/queue"), () => ({ items: [] })],
  ["GET", route("/budget"), deskBudget],
  ["GET", route("/turns/active"), () => ({ turn: null })],
];
