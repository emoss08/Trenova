// A Desk conversation in which Nova listed Acme Foods' shipments in transit or
// delayed: 128 of them, of which it read the first page of 25 and then the
// next, so the table holds 50 and pages on through the rest with the
// person's own access. The `deskTable` scenario field picks how the list
// answers when the person pages it: "paged" (every page reads), "slow" (a
// page past Nova's is left open, so the rows shimmer), "failing" (pages past
// the first one read fail), "locked" (the person may no longer read
// shipments), "withheld" (revenue is withheld at their access), "cut" (the
// table was cut to fit and keeps no place to go on from), "whole" (the list
// was 30 rows, all held) and "short" (12 rows, one page).
import { TriageError } from "./triage-model.mjs";

const THREAD = "athr_desk_table";
const ARTIFACT_ID = "aart_acme_in_transit";
const SLUG = "acme-shipments-in-transit";
const TITLE = "Shipments";
const QUESTION = "Which Acme Foods shipments are in transit or delayed right now?";
const STAGES = ["paged", "slow", "failing", "locked", "withheld", "cut", "whole", "short"];
const NOVA_PAGE = 25;
const NOVA_PAGES = 2;

const CUSTOMERS = ["Acme Foods", "Acme Foods", "Acme Foods · Frozen", "Acme Foods"];
const LANES = [
  ["Dallas, TX", "Atlanta, GA"],
  ["Fort Worth, TX", "Memphis, TN"],
  ["Houston, TX", "Chicago, IL"],
  ["Dallas, TX", "Denver, CO"],
  ["San Antonio, TX", "Phoenix, AZ"],
  ["Laredo, TX", "Kansas City, MO"],
  ["Dallas, TX", "Nashville, TN"],
  ["Austin, TX", "Oklahoma City, OK"],
  ["El Paso, TX", "Albuquerque, NM"],
  ["Dallas, TX", "St. Louis, MO"],
];

export const DESK_TABLE_COLUMNS = [
  { key: "proNumber", label: "Pro number", type: "text" },
  { key: "customer", label: "Customer", type: "text" },
  { key: "lane", label: "Lane", type: "text" },
  { key: "pickupDate", label: "Picked up", type: "date" },
  { key: "deliveryDate", label: "Due", type: "date" },
  { key: "status", label: "Status", type: "status" },
  { key: "revenue", label: "Revenue", type: "money" },
];

function stageOf(state) {
  const stage = state.scenario.deskTable ?? "paged";
  return STAGES.includes(stage) ? stage : "paged";
}

function listSize(stage) {
  switch (stage) {
    case "whole":
      return 30;
    case "short":
      return 12;
    default:
      return 128;
  }
}

/** The shipments the list holds, newest pickup first, as the list tool reads them. */
function shipments(state) {
  const day = 86400;
  const today = Math.floor(state.anchor / day) * day;
  return Array.from({ length: listSize(stageOf(state)) }, (_, n) => {
    const lane = LANES[(n * 7) % LANES.length];
    const revenue = 1180 + ((n * 389) % 3400) + (n % 3) * 0.5;
    return {
      id: `shp_mock_${String(n + 1).padStart(3, "0")}`,
      proNumber: `S-${240318 + n * 13}`,
      customer: CUSTOMERS[n % CUSTOMERS.length],
      lane: `${lane[0]} → ${lane[1]}`,
      pickupDate: today - (n % 4) * day + 8 * 3600,
      deliveryDate: today + (1 + ((n * 5) % 4)) * day + 14 * 3600,
      status: n % 11 === 4 ? "Delayed" : "InTransit",
      revenue: revenue.toFixed(2),
    };
  });
}

function held(stage, rows) {
  if (stage === "whole" || stage === "short") {
    return { rows, hasMore: false };
  }
  const kept = rows.slice(0, NOVA_PAGE * NOVA_PAGES);
  if (stage === "cut") {
    return { rows: kept, hasMore: true, truncated: true };
  }
  return { rows: kept, hasMore: true, nextCursor: `c-${kept.length}` };
}

function artifact(state) {
  const stage = stageOf(state);
  const all = shipments(state);
  const { rows, ...paging } = held(stage, all);
  return {
    id: ARTIFACT_ID,
    threadId: THREAD,
    messageId: "amsg_table_a1",
    runId: "arun_acme_in_transit",
    proposalId: "",
    planId: "",
    kind: "table_view",
    status: "Ready",
    title: TITLE,
    payload: {
      display: 1,
      tool: "shipment_list",
      entity: "shipments",
      recordEntity: "shipment",
      columns: DESK_TABLE_COLUMNS,
      rows,
      rowCount: stage === "cut" ? NOVA_PAGE * NOVA_PAGES + 12 : rows.length,
      searchedFor: ["status is In transit or Delayed", "customer is Acme Foods"],
      pages: stage === "short" ? 1 : NOVA_PAGES,
      ...paging,
    },
    sourceToolCallId: "call_acme_in_transit",
    pinned: false,
    lineageId: "",
    lineageSeq: 1,
    slug: SLUG,
    turn: QUESTION,
    createdAt: state.anchor - 14 * 60,
    updatedAt: state.anchor - 13 * 60,
  };
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
  const stage = stageOf(state);
  const count = listSize(stage);
  const delayed = shipments(state).filter((row) => row.status === "Delayed").length;
  return [
    message("amsg_table_u1", 1, "User", QUESTION, state.anchor - 15 * 60),
    message(
      "amsg_table_a1",
      2,
      "Assistant",
      `${count} Acme Foods shipments are moving right now, ${delayed} of them delayed. [${TITLE}](artifact:${SLUG}) has every one of them, with the delayed ones marked.`,
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
    agentDefinitionId: "",
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
export function deskTableThreads(state) {
  return [thread(state)];
}

const SORTABLE = ["proNumber", "customer", "pickupDate", "deliveryDate", "revenue"];

function compare(key, direction) {
  return (a, b) => {
    const x = a[key];
    const y = b[key];
    const order =
      key === "revenue"
        ? Number(x) - Number(y)
        : typeof x === "number"
          ? x - y
          : String(x).localeCompare(String(y));
    return direction === "desc" ? -order : order;
  };
}

/**
 * One page of the list, as the rows endpoint reads it: searched across the
 * pro number, the customer and the lane, sorted by a sortable column, from
 * after a cursor, and counted when asked.
 */
function rows(state, query) {
  const stage = stageOf(state);
  if (stage === "locked") {
    return {
      __status: 403,
      body: {
        type: "https://trenova.app/problems/forbidden",
        title: "Forbidden",
        status: 403,
        detail: "You do not have read access to shipment",
      },
    };
  }
  const after = query.get("after") ?? "";
  if (stage === "failing" && after !== `c-${NOVA_PAGE * NOVA_PAGES}`) {
    return {
      __status: 504,
      body: {
        type: "https://trenova.app/problems/timeout",
        title: "The list took too long to answer",
        status: 504,
        detail: "The list took too long to answer. Try again in a moment.",
      },
    };
  }
  const start = after.startsWith("c-") ? Number(after.slice(2)) : Number(query.get("offset") ?? 0);
  if (stage === "slow" && start > NOVA_PAGE * NOVA_PAGES) {
    return { __hold: true };
  }
  const limit = Math.min(Math.max(Number(query.get("limit") ?? 25), 1), 100);
  const needle = (query.get("q") ?? "").trim().toLowerCase();
  const sortBy = query.get("sortBy") ?? "";
  if (sortBy !== "" && !SORTABLE.includes(sortBy)) {
    throw new TriageError("This table cannot be sorted by that column", { field: "sortBy" });
  }
  let all = shipments(state);
  if (needle !== "") {
    all = all.filter((row) =>
      [row.proNumber, row.customer, row.lane].some((value) => value.toLowerCase().includes(needle)),
    );
  }
  if (sortBy !== "") {
    all = [...all].sort(compare(sortBy, query.get("sortDirection") ?? "asc"));
  }
  const withheld = stage === "withheld" ? ["revenue"] : [];
  const page = all.slice(start, start + limit).map((row) => {
    if (withheld.length === 0) return row;
    const { revenue: _revenue, ...rest } = row;
    return rest;
  });
  const end = start + page.length;
  return {
    rows: page,
    hasMore: end < all.length,
    nextCursor: end < all.length ? `c-${end}` : "",
    total: query.get("total") === "true" ? all.length : undefined,
    withheld,
    searchable: true,
    sortable: SORTABLE,
  };
}

function budget(state) {
  const tomorrow = Math.floor(state.anchor / 86400) * 86400 + 86400;
  return {
    agentName: "Nova",
    spentUsd: "0.40",
    limitUsd: "100.00",
    share: 0.004,
    near: false,
    monthStart: Math.floor(state.anchor / 86400) * 86400 - 15 * 86400,
    resetsAt: tomorrow + 15 * 86400,
    runsToday: 1,
    dailyRunLimit: 100,
    budgetUsed: false,
    dailyUsed: false,
    dayResetsAt: tomorrow,
    disabledBy: "",
    disabledAt: 0,
    person: { used: 2, limit: 500, resetsAt: tomorrow },
  };
}

const route = (rest = "") => new RegExp(`^/api/v1/assistant/threads/${THREAD}${rest}/?$`);

export const DESK_TABLE_ROUTES = [
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
    (state) => ({
      results: [artifact(state)],
      total: 1,
      nextCursor: "",
      counts: { all: 1, pinned: 0, families: { tables: 1 } },
    }),
  ],
  [
    "GET",
    route("/artifacts/by-slug/([a-z0-9-]+)"),
    (state, path) => {
      const slug = path.match(/by-slug\/([a-z0-9-]+)/)?.[1] ?? "";
      return { results: slug === SLUG ? [artifact(state)] : [] };
    },
  ],
  [
    "GET",
    route("/artifacts/(aart_[a-z0-9_]+)/rows"),
    (state, path, _body, query) => {
      const id = path.match(/artifacts\/(aart_[a-z0-9_]+)/)?.[1] ?? "";
      if (id !== ARTIFACT_ID) {
        throw new TriageError("Artifact not found", { code: "NOT_FOUND", type: "not-found" });
      }
      return rows(state, query);
    },
  ],
  [
    "GET",
    route("/artifacts/(aart_[a-z0-9_]+)"),
    (state, path) => {
      const id = path.match(/artifacts\/(aart_[a-z0-9_]+)/)?.[1] ?? "";
      return { results: id === ARTIFACT_ID ? [artifact(state)] : [] };
    },
  ],
  ["GET", route("/proposals"), () => ({ results: [] })],
  ["GET", route("/plans"), () => ({ results: [] })],
  ["GET", route("/schedules"), () => ({ items: [], total: 0 })],
  ["GET", route("/waits"), () => ({ items: [] })],
  ["GET", route("/queue"), () => ({ items: [] })],
  ["GET", route("/budget"), budget],
  ["GET", route("/turns/active"), () => ({ turn: null })],
];
