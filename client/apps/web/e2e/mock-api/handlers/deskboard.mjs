// A Desk conversation in which Dispatch Desk answered three questions as boards:
// six late loads on the Dallas lanes, three carriers head to head, and what still
// holds the October close. The `deskBoard` scenario field picks how they read:
// "live" (as answered), "moved" (read later: figures moved since the answer, a
// record and the chart's analysis no longer readable) and "reading" (held, so the
// board draws its skeleton).
//
// A second conversation, "athr_desk_board_inline", holds boards placed in the
// conversation as the reply itself: eight late loads (a list cut to six), Acme
// Foods' credit (a figure beside a meter, then a list and an action) and a
// carrier comparison asked once and then revised, so the first reply goes back
// to its words and the revision's reply draws the board. The same field also
// takes "locked" (a reader who may not read two of the loads, so a reading has
// no value and two rows are counted, not named) and "unreadable" (the boards
// cannot be read, so each reply draws its words with Try again).
import { deskBudget } from "./desktable.mjs";

const THREAD = "athr_desk_board";
const AGENT = { id: "agd_dispatch", name: "Dispatch desk", icon: "", accent: "" };
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

const segment = (text, numeral = 0, tone = "") => ({
  text,
  numeral,
  tone,
  struck: false,
  reading: "",
});
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
    caption: [],
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
    work: [],
    actions: [],
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
  [
    "shp_board_0423",
    "S2610-0423",
    "Dallas, TX → Texarkana, AR",
    "InTransit",
    25,
    "projected",
    -HOUR,
  ],
];

function lateRows(state) {
  const rows = LATE.map(([id, label, lane, status, minutes, standing, dueAgo]) =>
    record("shipment", id, label, lane, status, {
      measure: "lateness",
      value: dur((moved(state) ? Math.max(0, minutes - 90) : minutes) * MIN),
      state: moved(state) && minutes <= 110 ? "delivered" : standing,
      at: state.anchor - dueAgo,
      place: lane.split(" → ").at(-1),
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
    place: "beside",
    readings: [],
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
    place: "beside",
    readings: [],
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
      place: "",
      tone: "warn",
    }),
  );
  const invoices = INVOICES.map(([id, label, customer, amount, days]) =>
    record("invoice", id, label, customer, "Posted", {
      measure: "amountDue",
      value: usd(amount),
      state: "open",
      at: state.anchor + days * day,
      place: "",
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
    place: "beside",
    readings: [],
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
    payload: { board: { ...layout(card), place: "beside" }, place: "beside", author: AGENT.name },
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

function thread(state, id = THREAD, list = messages(state), title = BOARDS.late.question) {
  return {
    id,
    businessUnitId: "bu_mock",
    organizationId: "org_mock",
    userId: "usr_mock",
    agentDefinitionId: AGENT.id,
    title,
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

// ------------------------------------------------- boards in the conversation

const INLINE = "athr_desk_board_inline";
const locked = (state) => stageOf(state) === "locked";
const reading = (key, text) => ({ text, numeral: 0, tone: "", struck: false, reading: key });

function readingOf(key, label, value, was, source, measure = "", aggregate = "") {
  return { key, label, value, was, source, measure, aggregate };
}

const INLINE_LATE = [
  ["shp_inline_0412", "S2610-0412", "Dallas, TX → Memphis, TN", 190, "held"],
  ["shp_inline_0415", "S2610-0415", "Dallas, TX → Little Rock, AR", 165, "held"],
  ["shp_inline_0398", "S2610-0398", "Dallas, TX → Jackson, MS", 140, "late"],
  ["shp_inline_0421", "S2610-0421", "Dallas, TX → Monroe, LA", 110, "late"],
  ["shp_inline_0433", "S2610-0433", "Dallas, TX → Birmingham, AL", 75, "late"],
  ["shp_inline_0440", "S2610-0440", "Dallas, TX → Atlanta, GA", 40, "late"],
  ["shp_inline_0447", "S2610-0447", "Dallas, TX → Mobile, AL", 30, "late"],
  ["shp_inline_0452", "S2610-0452", "Dallas, TX → Pensacola, FL", 20, "late"],
];

function inlineLateRows(state) {
  return INLINE_LATE.map(([id, label, lane, minutes, standing], index) => {
    if (locked(state) && (index === 2 || index === 5)) return gone("shipment");
    const delivered = moved(state) && index < 2;
    return record("shipment", id, label, lane, delivered ? "Delivered" : "Delayed", {
      measure: "lateness",
      value: dur((moved(state) && !delivered ? minutes + 45 : minutes) * MIN),
      state: delivered ? "delivered" : standing,
      at: state.anchor - (minutes + 40) * MIN,
      place: lane.split(" → ").at(-1),
      tone: delivered ? "" : "warn",
    });
  });
}

function inlineLateCard(state) {
  const loads = {
    kind: "records",
    resource: "shipment",
    count: 8,
    readable: locked(state) ? 6 : 8,
    title: "",
  };
  return {
    artifactId: "aart_inline_late",
    threadId: INLINE,
    title: "Eight late shipments on the Dallas lanes",
    author: AGENT.name,
    question: "Which Dallas loads are late, and why?",
    answeredAt: state.anchor - 50 * MIN,
    readAt: state.anchor,
    place: "conversation",
    readings: [
      readingOf(
        "late",
        "Late now",
        count(moved(state) ? 6 : 8),
        moved(state) ? count(8) : null,
        loads,
        "lateness",
        "count",
      ),
      readingOf(
        "furthest",
        "Furthest behind",
        dur((moved(state) ? 185 : 190) * MIN),
        moved(state) ? dur(190 * MIN) : null,
        loads,
        "lateness",
        "max",
      ),
      readingOf(
        "owed",
        "Still owed on them",
        locked(state) ? null : usd(moved(state) ? "12180.00" : "18420.00"),
        moved(state) ? usd("18420.00") : null,
        loads,
        "amountDue",
        "sum",
      ),
    ],
    markdown: "# Eight late shipments on the Dallas lanes\n\nFour wait on one driver's hours.\n",
    verdict: [
      segment("There are "),
      reading("late", "8"),
      segment(" late loads on the Dallas lanes, the furthest "),
      reading("furthest", "3h 10m"),
      segment(" behind, with "),
      reading("owed", locked(state) ? "" : "$18,420.00"),
      segment(" still owed on them. "),
      segment("Four wait on one driver’s hours", 1, "warn"),
      segment(" and "),
      segment("two are late at the dock", 1),
      segment("."),
    ],
    blocks: [
      block({
        type: "records",
        title: "The eight, furthest behind first",
        numeral: 1,
        measure: "lateness",
        source: loads,
        rows: inlineLateRows(state),
      }),
      block({
        type: "actions",
        actions: [
          {
            label: "Ask the Memphis dock for an unload time",
            ask: "Ask the Memphis dock for an unload time",
            records: [record("shipment", "shp_inline_0412", "S2610-0412")],
          },
        ],
      }),
    ],
  };
}

const CREDIT_INVOICES = [
  ["inv_inline_20418", "INV-20418", "Sent Sep 30, net 30", "18400.00", 23],
  ["inv_inline_20431", "INV-20431", "Sent Oct 7, net 30", "12650.00", 16],
  ["inv_inline_20455", "INV-20455", "Sent Oct 14, net 30", "10200.00", 9],
];

const ANSWER_WAITING = ["shp_inline_0398", "shp_inline_0421", "shp_inline_0433", "shp_inline_0440"];
const ANSWER_DOCK = ["shp_inline_0412", "shp_inline_0415"];

/**
 * The same late loads answered with show_answer, the small tool: the answer
 * and two groups of records, side by side, each with a note under its tab. The
 * figures are its fixed vocabulary ({count}, {largest}, {count_1}); the server
 * declares a reading for each one used.
 */
function inlineAnswerCard(state) {
  const rows = inlineLateRows(state);
  const pick = (ids) => rows.filter((row, index) => ids.includes(INLINE_LATE[index][0]));
  const loads = (count) => ({
    kind: "records",
    resource: "shipment",
    count,
    readable: locked(state) ? count - 1 : count,
    title: "",
  });
  return {
    artifactId: "aart_inline_answer",
    threadId: INLINE,
    title: "Who the late loads wait on",
    author: AGENT.name,
    question: "Which of them wait on Marcus Bell, and which are at the dock?",
    answeredAt: state.anchor - 40 * MIN,
    readAt: state.anchor,
    place: "conversation",
    readings: [
      readingOf("count", "Late loads", count(6), null, loads(6), "lateness", "count"),
      readingOf(
        "largest",
        "Furthest behind",
        dur((moved(state) ? 185 : 190) * MIN),
        moved(state) ? dur(190 * MIN) : null,
        loads(6),
        "lateness",
        "max",
      ),
      readingOf(
        "count_1",
        "Waiting on his hours",
        locked(state) ? null : count(4),
        null,
        loads(4),
        "lateness",
        "count",
      ),
    ],
    markdown: "# Who the late loads wait on\n\nFour wait on one driver's hours.\n",
    verdict: [
      segment("There are "),
      reading("count", "6"),
      segment(" late loads, the furthest "),
      reading("largest", "3h 10m"),
      segment(" behind. Four wait on one driver’s hours and two are late at the dock."),
    ],
    blocks: [
      block({
        type: "records",
        title: "Waiting on one driver’s hours",
        numeral: 1,
        half: true,
        measure: "lateness",
        caption: [
          segment("Marcus Bell is out of hours until 9:15 AM, with "),
          reading("count_1", locked(state) ? "" : "4"),
          segment(" loads waiting on him."),
        ],
        source: loads(4),
        rows: pick(ANSWER_WAITING),
      }),
      block({
        type: "records",
        title: "Late at the dock",
        numeral: 2,
        half: true,
        measure: "lateness",
        caption: [segment("Both trucks are at the stop, waiting to unload.")],
        source: loads(2),
        rows: pick(ANSWER_DOCK),
      }),
      block({
        type: "actions",
        actions: [
          {
            label: "Ask the Memphis dock for an unload time",
            ask: "Ask the Memphis dock for an unload time",
            records: [],
          },
        ],
      }),
    ],
  };
}

/** The show_answer call that made the answer board, in the shape that tool takes. */
function answerArgs() {
  return {
    title: "Who the late loads wait on",
    answer:
      "There are {count} late loads, the furthest {largest} behind. Four wait on one driver’s hours and two are late at the dock.",
    measure: "lateness",
    groups: [
      {
        title: "Waiting on one driver’s hours",
        records: ANSWER_WAITING,
        note: "Marcus Bell is out of hours until 9:15 AM, with {count_1} loads waiting on him.",
      },
      {
        title: "Late at the dock",
        records: ANSWER_DOCK,
        note: "Both trucks are at the stop, waiting to unload.",
      },
    ],
    actions: [
      {
        label: "Ask the Memphis dock for an unload time",
        ask: "Ask the Memphis dock for an unload time",
      },
    ],
  };
}

function inlineCreditCard(state) {
  const day = 86400;
  const invoices = {
    kind: "records",
    resource: "invoice",
    count: 3,
    readable: 3,
    title: "",
  };
  const metrics = { kind: "metrics", resource: "", count: 0, readable: 0, title: "" };
  return {
    artifactId: "aart_inline_credit",
    threadId: INLINE,
    title: "Acme Foods' credit",
    author: AGENT.name,
    question: "How much room does Acme Foods have on its credit?",
    answeredAt: state.anchor - 30 * MIN,
    readAt: state.anchor,
    place: "conversation",
    readings: [
      readingOf(
        "open",
        "Credit still open",
        usd(moved(state) ? "58200.00" : "65700.00"),
        moved(state) ? usd("65700.00") : null,
        metrics,
      ),
      readingOf("past_due", "Past due", usd("41250.00"), null, invoices, "amountDue", "sum"),
    ],
    markdown: "# Acme Foods' credit\n\nPast due on three invoices.\n",
    verdict: [
      segment("Acme Foods has "),
      reading("open", "$65,700.00"),
      segment(" of credit open, but "),
      reading("past_due", "$41,250.00"),
      segment(" of what it owes is "),
      segment("past due", 3, "warn"),
      segment("."),
    ],
    blocks: [
      block({
        type: "figures",
        title: "What is open",
        numeral: 1,
        half: true,
        source: metrics,
        figures: [
          {
            label: "Open",
            value: usd(moved(state) ? "58200.00" : "65700.00"),
            was: moved(state) ? usd("65700.00") : null,
            tone: "",
          },
          { label: "Limit", value: usd("250000.00"), was: null, tone: "" },
        ],
      }),
      block({
        type: "meter",
        title: "Credit used, against the limit",
        numeral: 2,
        half: true,
        source: metrics,
        meter: { clear: 0, total: 0, tones: [], percent: pct(moved(state) ? 0.767 : 0.737) },
      }),
      block({
        type: "records",
        title: "Past due, oldest first",
        numeral: 3,
        measure: "amountDue",
        source: invoices,
        rows: CREDIT_INVOICES.map(([id, label, sent, amount, days]) =>
          record("invoice", id, label, sent, "Posted", {
            measure: "amountDue",
            value: usd(amount),
            state: "open",
            at: state.anchor - days * day,
            place: "",
            tone: "warn",
          }),
        ),
      }),
      block({
        type: "actions",
        actions: [
          {
            label: "Draft a reminder to Acme Foods",
            ask: "Draft a payment reminder to Acme Foods' payables contact for the three past-due invoices",
            records: [],
          },
        ],
      }),
    ],
  };
}

const INLINE_CARRIERS = [
  ["car_inline_acme", "Acme Logistics", "MC 884120", "Qualified"],
  ["car_inline_blue", "Blue Line Freight", "MC 610233", "Qualified"],
  ["car_inline_corr", "Corridor Transport", "MC 457781", "Qualified"],
];

function inlineCarriersCard(state, revised) {
  const columns = revised ? INLINE_CARRIERS.slice(0, 2) : INLINE_CARRIERS;
  const pick = (cells) => (revised ? cells.slice(0, 2) : cells);
  const lines = [
    line("Rate per mile", pick([usd("2.41"), usd("2.28"), usd("2.36")]), [1], "lower"),
    line("On time", pick([pct(0.91), pct(0.96), pct(0.88)]), [1]),
    line("Loads covered", pick([count(42), count(11), count(27)]), [0]),
  ];
  const metrics = { kind: "metrics", resource: "", count: 0, readable: 0, title: "" };
  return {
    artifactId: revised ? "aart_inline_carriers_v2" : "aart_inline_carriers",
    threadId: INLINE,
    title: revised
      ? "Acme against Blue Line, Dallas to Atlanta"
      : "Three carriers, Dallas to Atlanta",
    author: AGENT.name,
    question: revised
      ? "Leave Corridor out: just Acme and Blue Line."
      : "Compare Acme, Blue Line and Corridor for the Dallas–Atlanta lanes.",
    answeredAt: state.anchor - (revised ? 10 : 20) * MIN,
    readAt: state.anchor,
    place: "conversation",
    readings: [
      readingOf("blue_loads", "Blue Line's loads", count(11), null, metrics),
      readingOf("acme_loads", "Acme's loads", count(42), null, metrics),
    ],
    markdown: "# Carriers, Dallas to Atlanta\n\nBlue Line is the cheapest and the most punctual.\n",
    verdict: [
      segment("Blue Line is "),
      segment(
        revised
          ? "cheaper and more punctual than Acme"
          : "the cheapest and the most punctual of the three",
        1,
      ),
      segment(", but it has covered "),
      reading("blue_loads", "11"),
      segment(" loads against Acme’s "),
      reading("acme_loads", "42"),
      segment(", so its numbers rest on far less."),
    ],
    blocks: [
      block({
        type: "compare",
        title: revised ? "Acme against Blue Line, last 90 days" : "Three carriers, last 90 days",
        numeral: 1,
        source: {
          kind: "mixed",
          resource: "",
          count: columns.length,
          readable: columns.length,
          title: "",
        },
        compare: {
          columns: columns.map(([id, label, mc, status]) =>
            record("carrier", id, label, mc, status),
          ),
          lines,
          wins: revised ? [1, 2] : [1, 2, 0],
        },
      }),
      block({
        type: "prose",
        text: [segment("Either tender is drafted for your approval before anything is sent.")],
      }),
      block({
        type: "actions",
        actions: [
          {
            label: "Tender to Blue Line",
            ask: "Tender the Atlanta load to Blue Line Freight",
            records: [],
          },
          {
            label: "Tender to Acme",
            ask: "Tender the Atlanta load to Acme Logistics",
            records: [],
          },
        ],
      }),
    ],
  };
}

const INLINE_TURNS = [
  { key: "late", call: "call_inline_late", card: (state) => inlineLateCard(state) },
  {
    key: "answer",
    call: "call_inline_answer",
    tool: "show_answer",
    card: (state) => inlineAnswerCard(state),
    args: answerArgs,
  },
  { key: "credit", call: "call_inline_credit", card: (state) => inlineCreditCard(state) },
  {
    key: "carriers",
    call: "call_inline_carriers",
    card: (state) => inlineCarriersCard(state, false),
  },
  { key: "revised", call: "call_inline_revised", card: (state) => inlineCarriersCard(state, true) },
];

function inlineVerdict(card) {
  return card.verdict
    .map((part) =>
      part.reading
        ? `{${part.reading}}`
        : part.numeral
          ? `[${part.text}][${part.numeral}]`
          : part.text,
    )
    .join("");
}

/** The board as the agent asked show_board for it: the words with their readings and cites. */
function inlineSpec(card, lineage) {
  return {
    ...layout(card),
    verdict: inlineVerdict(card),
    place: "conversation",
    readings: card.readings.map((entry) => ({
      key: entry.key,
      label: entry.label,
      ...(entry.source.kind === "records"
        ? {
            records: { resource: entry.source.resource, ids: [] },
            measure: entry.measure,
            aggregate: entry.aggregate,
          }
        : { metric: entry.key }),
    })),
    blocks: card.blocks.map((filled) => ({
      ...layout({ ...card, blocks: [filled] }).blocks[0],
      ...(filled.actions.length
        ? { actions: filled.actions.map(({ label, ask }) => ({ label, ask })) }
        : {}),
      ...(filled.text.length ? { text: filled.text.map((part) => part.text).join("") } : {}),
      ...(filled.caption.length
        ? {
            caption: filled.caption
              .map((part) => (part.reading ? `{${part.reading}}` : part.text))
              .join(""),
          }
        : {}),
    })),
    ...(lineage ? { boardId: lineage } : {}),
  };
}

/**
 * The reply the server records for a board in the conversation: the verdict
 * with each reading's value written in, cite marks removed, and the board's
 * link joined in parentheses before the closing punctuation.
 */
function inlineReply(card) {
  const words = card.verdict.map((part) => part.text || "—").join("");
  const link = ` ([${card.title}](artifact:${card.artifactId}))`;
  return words.endsWith(".") ? `${words.slice(0, -1)}${link}.` : `${words}${link}`;
}

function inlineArtifacts(state) {
  return INLINE_TURNS.map((turn, index) => {
    const card = turn.card(state);
    const revision = turn.key === "revised";
    return {
      id: card.artifactId,
      threadId: INLINE,
      messageId: `amsg_inline_a${index + 1}`,
      runId: "arun_inline",
      proposalId: "",
      planId: "",
      kind: "board",
      status: "Ready",
      title: card.title,
      payload: {
        board: inlineSpec(card, revision ? "aart_inline_carriers" : ""),
        place: "conversation",
        author: AGENT.name,
      },
      sourceToolCallId: turn.call,
      pinned: false,
      lineageId: revision ? "aart_inline_carriers" : "",
      lineageSeq: revision ? 2 : 1,
      slug: revision || turn.key === "carriers" ? "carriers-dallas-atlanta" : `inline-${turn.key}`,
      turn: card.question,
      createdAt: card.answeredAt,
      updatedAt: card.answeredAt,
    };
  });
}

/**
 * A board tool's result as the runtime saves it (`runtimetools.BoardContent`
 * fenced by `toolresult.Fence`): the JSON line and the board's brief inside the
 * untrusted-data fence, then the note after it. The client never parses it.
 */
function fencedResult(tool, card) {
  const json = JSON.stringify({
    artifactId: card.artifactId,
    title: card.title,
    status: "shown",
    place: "conversation",
  });
  return `Result from ${tool}:\n<untrusted_data>\n${json}\n\n${card.markdown.trim()}\n</untrusted_data>\n\n[The board was drawn in the conversation as your reply, and the turn ended with it.]`;
}

function inlineMessages(state) {
  const base = { threadId: INLINE, kind: "Message", toolFailed: false, refused: false };
  return INLINE_TURNS.flatMap((turn, index) => {
    const card = turn.card(state);
    const at = card.answeredAt;
    const sequence = index * 4;
    return [
      {
        ...base,
        id: `amsg_inline_u${index + 1}`,
        sequence: sequence + 1,
        role: "User",
        content: card.question,
        createdAt: at - MIN,
      },
      {
        ...base,
        id: `amsg_inline_c${index + 1}`,
        sequence: sequence + 2,
        role: "Assistant",
        content: "",
        createdAt: at - 20,
        toolCalls: [
          {
            id: turn.call,
            name: turn.tool ?? "show_board",
            arguments: turn.args
              ? turn.args()
              : inlineSpec(card, turn.key === "revised" ? "aart_inline_carriers" : ""),
            effect: "present",
          },
        ],
      },
      {
        ...base,
        id: `amsg_inline_r${index + 1}`,
        sequence: sequence + 3,
        role: "Tool",
        content: fencedResult(turn.tool ?? "show_board", card),
        createdAt: at - 5,
        toolCallId: turn.call,
        toolName: turn.tool ?? "show_board",
        effect: "present",
        toolVerdict: "ran",
        summary: "Showed a board",
      },
      {
        ...base,
        id: `amsg_inline_a${index + 1}`,
        sequence: sequence + 4,
        role: "Assistant",
        content: inlineReply(
          turn.card({ ...state, scenario: { ...state.scenario, deskBoard: "live" } }),
        ),
        createdAt: at,
      },
    ];
  });
}

function inlineCard(state, artifactId) {
  const turn = INLINE_TURNS.find((entry) => entry.card(state).artifactId === artifactId);
  return turn ? turn.card(state) : null;
}

/** The conversations this file serves, for the rail's list and shelf counts. */
export function deskBoardThreads(state) {
  return [thread(state), inlineThread(state)];
}

function inlineThread(state) {
  return thread(state, INLINE, inlineMessages(state), "Late Dallas loads, and the Atlanta lane");
}

export const DESK_BOARD_HANDLERS = {
  DeskBoardCard: (state, { artifactId }) => {
    if (stageOf(state) === "reading") return { __hold: true };
    const inline = inlineCard(state, artifactId);
    if (inline) {
      if (stageOf(state) === "unreadable") throw new Error("The board could not be read");
      return { deskBoardCard: inline };
    }
    const key = Object.keys(BOARDS).find((name) => BOARDS[name].id === artifactId);
    if (!key) throw new Error(`board ${artifactId} not found`);
    return { deskBoardCard: CARDS[key](state) };
  },
};

const routeOf =
  (id) =>
  (rest = "") =>
    new RegExp(`^/api/v1/assistant/threads/${id}${rest}/?$`);

function threadRoutes(id, { read, list, artifactList }) {
  const route = routeOf(id);
  return [
    ["GET", route(), read],
    [
      "GET",
      route("/messages"),
      (state) => {
        const all = list(state);
        return { results: all, hasMore: false, total: all.length, limit: 50 };
      },
    ],
    [
      "GET",
      route("/artifacts"),
      (state) => {
        const all = artifactList(state);
        return {
          results: all,
          total: all.length,
          nextCursor: "",
          counts: { all: all.length, pinned: 0, families: { report: all.length } },
        };
      },
    ],
    [
      "GET",
      route("/artifacts/by-slug/([a-z0-9-]+)"),
      (state, path) => {
        const slug = path.match(/by-slug\/([a-z0-9-]+)/)?.[1] ?? "";
        return { results: artifactList(state).filter((item) => item.slug === slug) };
      },
    ],
    [
      "GET",
      route("/artifacts/(aart_[a-z0-9_]+)"),
      (state, path) => {
        const artifactId = path.match(/artifacts\/(aart_[a-z0-9_]+)/)?.[1] ?? "";
        const named = artifactList(state).find((item) => item.id === artifactId);
        const lineage = named ? named.lineageId || named.id : "";
        return {
          results: artifactList(state).filter(
            (item) => item.id === lineage || item.lineageId === lineage,
          ),
        };
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
}

export const DESK_BOARD_ROUTES = [
  ...threadRoutes(THREAD, {
    read: (state) => thread(state),
    list: messages,
    artifactList: artifacts,
  }),
  ...threadRoutes(INLINE, {
    read: inlineThread,
    list: inlineMessages,
    artifactList: inlineArtifacts,
  }),
];
