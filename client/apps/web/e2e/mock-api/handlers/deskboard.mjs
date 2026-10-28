// A Desk conversation in which Dispatch Desk answered three questions as boards:
// six late loads on the Dallas lanes, three carriers head to head, and what still
// holds the October close. The `deskBoard` scenario field picks how they read:
// "live" (as answered), "moved" (read later: figures moved since the answer, a
// record and the chart's analysis no longer readable) and "reading" (held, so the
// board draws its skeleton).
import { deskBudget } from "./desktable.mjs";

const THREAD = "athr_desk_board";
const AGENT = { id: "agd_mock_dispatch_desk", name: "Dispatch Desk", icon: "", accent: "teal" };
const HOUR = 3600;
const MIN = 60;

const BOARDS = {
  late: {
    id: "aart_board_late",
    slug: "six-late-shipments",
    question: "Which Dallas loads are late, and why?",
    reply: "Here is [where the six late loads stand](artifact:six-late-shipments).",
  },
  carriers: {
    id: "aart_board_carriers",
    slug: "carriers-head-to-head",
    question: "Compare Acme, Blue Line and Corridor for the Dallas–Atlanta lanes.",
    reply: "I put [the three carriers side by side](artifact:carriers-head-to-head).",
  },
  close: {
    id: "aart_board_close",
    slug: "october-close",
    question: "What's blocking month-end close?",
    reply: "[What still holds October](artifact:october-close) is laid out with what I'd do.",
  },
};

const stageOf = (state) => state.scenario.deskBoard ?? "live";
const moved = (state) => stageOf(state) === "moved";

const segment = (text, numeral = 0, tone = "") => ({ text, numeral, tone, struck: false });
const count = (n) => ({ kind: "count", number: String(n), currency: "", text: "" });
const usd = (amount) => ({ kind: "money", number: amount, currency: "USD", text: "" });
const pct = (n) => ({ kind: "percent", number: String(n), currency: "", text: "" });
const dur = (seconds) => ({ kind: "duration", number: String(seconds), currency: "", text: "" });
const num = (n) => ({ kind: "number", number: String(n), currency: "", text: "" });

function block(fields) {
  return {
    type: "figures",
    title: "",
    half: false,
    numeral: 0,
    state: "ready",
    tone: "",
    measure: "",
    noteTone: "",
    source: { kind: "", resource: "", count: 0, readable: 0, title: "" },
    figures: [],
    rows: [],
    compare: null,
    table: null,
    chart: null,
    events: [],
    checks: [],
    meter: null,
    text: [],
    chips: [],
    ...fields,
  };
}

function record(resource, id, label, subtitle = "", status = "", measure = null) {
  return { resource, id, readable: true, label, subtitle, status, measure };
}

const gone = (resource) => ({
  resource,
  id: "",
  readable: false,
  label: "",
  subtitle: "",
  status: "",
  measure: null,
});

const LATE = [
  ["shp_board_0412", "S2610-0412", "Dallas, TX → Memphis, TN", "Delayed", 190, "late", 2 * HOUR],
  [
    "shp_board_0415",
    "S2610-0415",
    "Dallas, TX → Little Rock, AR",
    "Delayed",
    165,
    "late",
    2.5 * HOUR,
  ],
  ["shp_board_0419", "S2610-0419", "Dallas, TX → Jackson, MS", "Delayed", 140, "late", 0.5 * HOUR],
  ["shp_board_0421", "S2610-0421", "Irving, TX → Tyler, TX", "InTransit", 125, "held", 4 * HOUR],
  [
    "shp_board_0408",
    "S2610-0408",
    "Fort Worth, TX → Shreveport, LA",
    "InTransit",
    110,
    "held",
    5 * HOUR,
  ],
  ["shp_board_0423", "S2610-0423", "Dallas, TX → Texarkana, AR", "Delayed", 95, "late", HOUR],
];

function lateRows(state) {
  const rows = LATE.map(([id, label, lane, status, minutes, standing, dueAgo]) =>
    record("shipment", id, label, lane, status, {
      measure: "lateness",
      value: dur((moved(state) ? Math.max(0, minutes - 90) : minutes) * MIN),
      state: moved(state) && minutes <= 110 ? "delivered" : standing,
      at: state.anchor - dueAgo,
      tone: moved(state) && minutes <= 110 ? "" : "warn",
    }),
  );
  return moved(state) ? [...rows.slice(0, 4), gone("shipment"), rows[5]] : rows;
}

function lateCard(state) {
  const answeredAt = state.anchor - 40 * MIN;
  const late = moved(state) ? 2 : 6;
  return {
    artifactId: BOARDS.late.id,
    threadId: THREAD,
    title: "Six late shipments on the Dallas lanes",
    author: AGENT.name,
    question: BOARDS.late.question,
    answeredAt,
    readAt: state.anchor,
    markdown:
      "# Six late shipments on the Dallas lanes\n\nFour of the six are held behind one driver's hours.\n",
    verdict: [
      segment("Four of the six are held behind "),
      segment("one driver’s hours", 3, "warn"),
      segment(
        ": Marcus Bell ran out of driving hours at the Shreveport fuel stop, and the loads relayed to him wait on his break. ",
      ),
      segment("The other two are late at the dock", 2),
      segment(", not on the road."),
    ],
    blocks: [
      block({
        type: "figures",
        title: "Where they stand",
        numeral: 1,
        source: { kind: "records", resource: "shipment", count: 6, readable: 6, title: "" },
        figures: [
          {
            label: "Late now",
            value: count(late),
            was: moved(state) ? count(6) : null,
            tone: "bad",
          },
          {
            label: "Furthest behind",
            value: dur((moved(state) ? 100 : 190) * MIN),
            was: moved(state) ? dur(190 * MIN) : null,
            tone: "warn",
          },
          {
            label: "Revenue on them",
            value: usd(moved(state) ? "6145.00" : "18420.00"),
            was: null,
            tone: "",
          },
          { label: "Due by 6 PM", value: count(moved(state) ? 0 : 4), was: null, tone: "" },
        ],
      }),
      block({
        type: "records",
        title: "The six, furthest behind first",
        numeral: 2,
        measure: "lateness",
        source: {
          kind: "records",
          resource: "shipment",
          count: 6,
          readable: moved(state) ? 5 : 6,
          title: "",
        },
        rows: lateRows(state),
      }),
      block({
        type: "timeline",
        title: "S2610-0412, hour by hour",
        numeral: 3,
        half: true,
        source: { kind: "activity", resource: "shipment", count: 1, readable: 1, title: "" },
        events: [
          event(state, 6 * HOUR + 50 * MIN, "create", "Picked up at Dallas, TX", "Marcus Bell"),
          event(state, 3 * HOUR, "update", "Stopped at the Shreveport fuel stop", "", "api_key"),
          event(state, 2 * HOUR + 25 * MIN, "update", "Ran out of driving hours", "", "api_key"),
          event(state, 55 * MIN, "update", "Missed the Memphis appointment", ""),
          event(state, 50 * MIN, "update", "ETA moved to 5:10 PM", "Dana Ruiz"),
        ],
      }),
      block({
        type: "note",
        title: "Re-tender the two furthest behind to a team",
        half: true,
        noteTone: "info",
        text: [
          segment(
            "S2610-0412 and S2610-0415 can still make their receivers before they close. The other four clear on their own once Marcus is back on duty.",
          ),
        ],
        chips: [
          record("shipment", "shp_board_0412", "S2610-0412"),
          record("shipment", "shp_board_0415", "S2610-0415"),
          record("worker", "wrk_board_bell", "Marcus Bell"),
        ],
      }),
    ],
  };
}

function event(state, ago, operation, comment, actor, principalType = "") {
  return {
    at: state.anchor - ago,
    operation,
    actor,
    principalType: actor ? "session_user" : principalType,
    comment,
    changes: null,
  };
}

const CARRIERS = [
  ["car_board_acme", "Acme Freight", "MC 884120", "Qualified"],
  ["car_board_blue", "Blue Line", "MC 610233", "Qualified"],
  ["car_board_corr", "Corridor", "MC 457781", "Pending"],
];

function line(label, cells, best, better = "higher") {
  return { label, better, cells, wins: cells.map((_, index) => best.includes(index)) };
}

function carriersCard(state) {
  const lines = [
    line("Loads hauled", [count(142), count(118), count(64)], [0]),
    line("On-time pickup", [pct(0.951), pct(0.983), pct(0.969)], [1]),
    line("On-time delivery", [pct(0.918), pct(0.974), pct(0.931)], [1]),
    line("Cost per mile", [usd("2.31"), usd("2.48"), usd("2.39")], [0], "lower"),
    line("Tenders accepted", [pct(0.82), pct(0.94), pct(0.77)], [1]),
    line("Detention per load (h)", [num(1.4), num(0.6), num(1.1)], [1], "lower"),
    line("Open claims", [count(0), count(0), count(2)], [0, 1], "lower"),
  ];
  const weeks = [
    "Aug 4",
    "Aug 11",
    "Aug 18",
    "Aug 25",
    "Sep 1",
    "Sep 8",
    "Sep 15",
    "Sep 22",
    "Sep 29",
    "Oct 6",
    "Oct 13",
    "Oct 20",
  ];
  return {
    artifactId: BOARDS.carriers.id,
    threadId: THREAD,
    title: "Acme, Blue Line and Corridor, head to head",
    author: AGENT.name,
    question: BOARDS.carriers.question,
    answeredAt: state.anchor - 25 * MIN,
    readAt: state.anchor,
    markdown:
      "# Acme, Blue Line and Corridor, head to head\n\nBlue Line wins five of the seven measures.\n",
    verdict: [
      segment("Blue Line wins five of the seven measures", 1),
      segment(
        ", and it is the one you can count on to deliver on time. Acme is the cheapest per mile but ",
      ),
      segment("has slipped almost every week since September", 2, "warn"),
      segment(", and Corridor still has two claims open."),
    ],
    blocks: [
      block({
        type: "compare",
        title: "Head to head, last 90 days",
        numeral: 1,
        source: { kind: "mixed", resource: "", count: 3, readable: 3, title: "" },
        compare: {
          columns: CARRIERS.map(([id, label, mc, status]) =>
            record("carrier", id, label, mc, status),
          ),
          lines,
          wins: [3, 5, 0],
        },
      }),
      block({
        type: "chart",
        title: "On-time delivery, week by week",
        numeral: 2,
        state: moved(state) ? "unreadable" : "ready",
        source: {
          kind: "analysis",
          resource: "",
          count: 0,
          readable: 0,
          title: "Carrier scorecard",
        },
        chart: moved(state)
          ? null
          : {
              kind: "line",
              title: "On-time delivery",
              x: { label: "Week", kind: "text", values: weeks },
              series: [
                {
                  name: "Blue Line",
                  values: [0.96, 0.97, 0.97, 0.98, 0.96, 0.98, 0.97, 0.98, 0.97, 0.98, 0.97, 0.98],
                },
                {
                  name: "Corridor",
                  values: [0.9, 0.92, 0.93, 0.92, 0.94, 0.93, 0.95, 0.93, 0.94, 0.92, 0.95, 0.94],
                },
                {
                  name: "Acme",
                  values: [0.94, 0.95, 0.93, 0.92, 0.94, 0.91, 0.9, 0.92, 0.89, 0.9, 0.88, 0.89],
                },
              ],
              valueKind: "percent",
              currency: "",
              stacked: false,
              horizontal: false,
            },
      }),
      block({
        type: "note",
        title: "Give Blue Line the Dallas–Atlanta lanes",
        noteTone: "ok",
        text: [
          segment(
            "It costs a few cents more a mile and pays that back in fewer missed appointments and less detention. Keep Acme for the overflow, and leave Corridor on probation until both claims are closed.",
          ),
        ],
        chips: CARRIERS.map(([id, label]) => record("carrier", id, label)),
      }),
    ],
  };
}

const ENTRIES = [
  ["je_board_0118", "JE-2610-0118", "Accrue October fuel, Pilot and Love’s", 3],
  ["je_board_0121", "JE-2610-0121", "Reclass detention revenue to accessorials", 2],
  ["je_board_0124", "JE-2610-0124", "Driver advance write-off, M. Ortiz", 1],
];
const INVOICES = [
  ["inv_board_0042", "INV-2610-0042", "Acme Foods", "2930.50", 12],
  ["inv_board_0057", "INV-2610-0057", "Riverside Grocers", "4118.00", 9],
  ["inv_board_0063", "INV-2610-0063", "Tyson Foods", "11402.25", 6],
  ["inv_board_0066", "INV-2610-0066", "Northwind Grocers", "3960.00", 5],
];

function closeCard(state) {
  const day = 86400;
  const entries = ENTRIES.map(([id, label, memo, days]) =>
    record("journal_entry", id, label, memo, "Pending", {
      measure: "pendingFor",
      value: dur(days * day),
      state: "pending",
      at: state.anchor - days * day,
      tone: "warn",
    }),
  );
  const invoices = INVOICES.map(([id, label, customer, amount, days]) =>
    record("invoice", id, label, customer, "Posted", {
      measure: "amountDue",
      value: usd(amount),
      state: "open",
      at: state.anchor + days * day,
      tone: "",
    }),
  );
  const check = (title, tone, n, detail, records = [], hidden = 0) => ({
    title,
    tone,
    detail,
    count: count(n),
    was: null,
    records,
    hidden,
  });
  const tones = [...Array(19).fill("ok"), "info", ...Array(4).fill("warn"), "bad", "bad"];
  return {
    artifactId: BOARDS.close.id,
    threadId: THREAD,
    title: "October close, and what still holds it",
    author: AGENT.name,
    question: BOARDS.close.question,
    answeredAt: state.anchor - 6 * MIN,
    readAt: state.anchor,
    markdown: "# October close, and what still holds it\n\nTwo things hold October open.\n",
    verdict: [
      segment("Two things hold October open: "),
      segment("three journal entries waiting for approval", 2),
      segment(" and "),
      segment("four invoices that never reached QuickBooks", 2),
      segment(". The rest is "),
      segment("close to done", 1),
      segment(", and nothing else blocks the close."),
    ],
    blocks: [
      block({
        type: "meter",
        title: "How far the close is",
        numeral: 1,
        source: { kind: "checks", resource: "", count: 26, readable: 26, title: "" },
        meter: { clear: 19, total: 26, tones, percent: null },
      }),
      block({
        type: "checks",
        title: "What is still open",
        numeral: 2,
        source: { kind: "checks", resource: "", count: 5, readable: 5, title: "" },
        checks: [
          check(
            "Journal entries waiting for approval",
            "bad",
            3,
            "Posting holds until an approver signs them off.",
            entries,
          ),
          check(
            "Invoices that did not reach QuickBooks",
            "bad",
            4,
            "Each stopped on a customer with no QuickBooks match.",
            invoices.slice(0, 3),
            1,
          ),
          check(
            "Receipts not applied",
            "warn",
            2,
            "Both paid more than one invoice; the remittance says which.",
          ),
          check(
            "Bank lines not matched",
            "warn",
            12,
            "All from the last three days; most are carrier payments.",
          ),
          check(
            "Fuel purchases still importing",
            "info",
            1,
            "Last week’s Comdata file posts on its own when it finishes.",
          ),
        ],
      }),
      block({
        type: "records",
        title: "The entries to approve",
        numeral: 3,
        half: true,
        measure: "pendingFor",
        source: { kind: "records", resource: "journal_entry", count: 3, readable: 3, title: "" },
        rows: entries,
      }),
      block({
        type: "records",
        title: "The invoices to send again",
        numeral: 4,
        half: true,
        measure: "amountDue",
        source: { kind: "records", resource: "invoice", count: 4, readable: 4, title: "" },
        rows: invoices,
      }),
      block({
        type: "note",
        title: "The close cannot finish before Friday as things stand",
        noteTone: "bad",
        text: [
          segment(
            "The three entries wait on Mia Chen, who is out until Thursday. Omar Haddad can approve journal entries too; asking him today keeps Friday open.",
          ),
        ],
      }),
    ],
  };
}

const CARDS = { late: lateCard, carriers: carriersCard, close: closeCard };

function layout(card) {
  return {
    title: card.title,
    verdict: card.verdict
      .map((part) => (part.numeral ? `[${part.text}][${part.numeral}]` : part.text))
      .join(""),
    blocks: card.blocks.map((filled) => ({
      type: filled.type,
      title: filled.title,
      ...(filled.half ? { half: true } : {}),
      ...(filled.figures.length ? { figures: filled.figures.map(({ label }) => ({ label })) } : {}),
      ...(filled.rows.length
        ? { records: { resource: filled.rows[0].resource, ids: filled.rows.map((row) => row.id) } }
        : {}),
      ...(filled.checks.length
        ? { checks: filled.checks.map(({ title, tone }) => ({ title, tone })) }
        : {}),
    })),
  };
}

function artifact(state, key, index) {
  const board = BOARDS[key];
  const card = CARDS[key](state);
  return {
    id: board.id,
    threadId: THREAD,
    messageId: `amsg_board_a${index + 1}`,
    runId: "arun_board",
    proposalId: "",
    planId: "",
    kind: "board",
    status: "Ready",
    title: card.title,
    payload: { board: layout(card), author: AGENT.name },
    sourceToolCallId: `call_board_${key}`,
    pinned: false,
    lineageId: "",
    lineageSeq: 1,
    slug: board.slug,
    turn: board.question,
    createdAt: card.answeredAt,
    updatedAt: card.answeredAt,
  };
}

function artifacts(state) {
  return Object.keys(BOARDS).map((key, index) => artifact(state, key, index));
}

function messages(state) {
  return Object.keys(BOARDS).flatMap((key, index) => {
    const answeredAt = CARDS[key](state).answeredAt;
    const base = {
      threadId: THREAD,
      kind: "Message",
      toolFailed: false,
      refused: false,
    };
    return [
      {
        ...base,
        id: `amsg_board_u${index + 1}`,
        sequence: index * 2 + 1,
        role: "User",
        content: BOARDS[key].question,
        createdAt: answeredAt - MIN,
      },
      {
        ...base,
        id: `amsg_board_a${index + 1}`,
        sequence: index * 2 + 2,
        role: "Assistant",
        content: BOARDS[key].reply,
        createdAt: answeredAt,
      },
    ];
  });
}

function thread(state) {
  const list = messages(state);
  return {
    id: THREAD,
    businessUnitId: "bu_mock",
    organizationId: "org_mock",
    userId: "usr_mock",
    agentDefinitionId: AGENT.id,
    title: BOARDS.late.question,
    status: "Active",
    lastMessageAt: list.at(-1).createdAt,
    lastReadAt: state.anchor,
    origin: "Desk",
    pinned: false,
    canContinue: true,
    version: 3,
    createdAt: list[0].createdAt,
    updatedAt: list.at(-1).createdAt,
  };
}

/** The conversation this file serves, for the rail's list and shelf counts. */
export function deskBoardThreads(state) {
  return [thread(state)];
}

export const DESK_BOARD_HANDLERS = {
  DeskBoardCard: (state, { artifactId }) => {
    if (stageOf(state) === "reading") return { __hold: true };
    const key = Object.keys(BOARDS).find((name) => BOARDS[name].id === artifactId);
    if (!key) throw new Error(`board ${artifactId} not found`);
    return { deskBoardCard: CARDS[key](state) };
  },
};

const route = (rest = "") => new RegExp(`^/api/v1/assistant/threads/${THREAD}${rest}/?$`);

export const DESK_BOARD_ROUTES = [
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
      return { results: artifacts(state).filter((item) => item.slug === slug) };
    },
  ],
  [
    "GET",
    route("/artifacts/(aart_[a-z0-9_]+)"),
    (state, path) => {
      const id = path.match(/artifacts\/(aart_[a-z0-9_]+)/)?.[1] ?? "";
      return { results: artifacts(state).filter((item) => item.id === id) };
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
