// Boards beyond dispatch, one conversation per approved mockup frame
// (.claude/tmp/desk-beyond-dispatch/project/*.dc.html), each a board placed in the
// conversation as the reply, filled with the frame's own numbers in the shape the server
// sends (`DeskBoardMetric`: a chart, split, grid, log, checks, table or figures). The last,
// "athr_desk_board_beyond", is the Measures frame: every form once.
//
// The `deskBoard` scenario field also reaches these: "locked" draws the Measures board as a
// reader without access (one exhibit not theirs to read, a ranked row withheld), "empty" its
// forms with nothing to draw (state "empty", as the server sets it when a read does not fit
// its form), "reading" holds them so each form draws its sketch.

const MIN = 60;
const HOUR = 3600;
const DAY = 86400;

export const BEYOND_AGENTS = {
  analyst: { name: "Report analyst" },
  receivables: { name: "Receivables" },
  fuel: { name: "Fuel and IFTA clerk" },
  detention: { name: "Detention desk" },
  coverage: { name: "Dispatch coverage agent" },
  settlements: { name: "Settlements clerk" },
  carriers: { name: "Carrier risk desk" },
  workforce: { name: "Workforce coordinator" },
  edi: { name: "EDI desk" },
  books: { name: "Books keeper" },
};

const value = (kind, number, currency = "") => ({
  kind,
  number: String(number),
  currency,
  text: "",
});
const usd = (amount) => value("money", amount, "USD");
const pct = (fraction) => value("percent", fraction);
const count = (n) => value("count", n);
const dur = (seconds) => value("duration", seconds);
const num = (n) => value("number", n);
const seg = (text, numeral = 0, tone = "") => ({
  text,
  numeral,
  tone,
  struck: false,
  reading: "",
});

function block(fields) {
  return {
    type: "metric",
    title: "",
    half: false,
    numeral: 0,
    caption: [],
    state: "ready",
    tone: "",
    measure: "",
    noteTone: "",
    source: { kind: "metrics", recordKind: "", count: 0, readable: 0, title: "" },
    figures: [],
    rows: [],
    compare: null,
    table: null,
    metric: null,
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

/** A metric as `DeskBoardMetric` carries it: every field its form does not fill left empty. */
function metric(fields) {
  return {
    name: "",
    form: "figures",
    title: "",
    source: "",
    figures: [],
    chart: null,
    split: null,
    grid: null,
    log: [],
    checks: [],
    table: null,
    ...fields,
  };
}

function chart(fields) {
  return {
    unit: "number",
    currency: "",
    ticks: [],
    series: [],
    marked: [],
    band: null,
    target: null,
    ref: null,
    ghost: false,
    rows: [],
    ...fields,
  };
}

const tick = (label, marked = false) => ({ at: null, label, marked });
const at = (moment) => ({ at: moment, label: "", marked: false });
const series = (name, role, points) => ({
  name,
  role,
  points: points.map((point) => (point === null ? null : String(point))),
});

function chartRow(fields) {
  return {
    label: "",
    record: null,
    value: null,
    marked: false,
    rest: false,
    spans: [],
    dots: [],
    ...fields,
  };
}

function record(kind, id, label, subtitle = "", status = "") {
  return {
    kind,
    id,
    readable: true,
    label,
    subtitle,
    status,
    statusTone: "",
    opens: null,
    measure: null,
  };
}

const figure = (name, label, figureValue, fields = {}) => ({
  name,
  label,
  value: figureValue,
  was: null,
  wasAt: null,
  tone: "",
  ...fields,
});

/** An exhibit that reads a metric, numbered, with its caption and where it was read from. */
function exhibit({ numeral, title, half = false, caption = "", read, state = "ready" }) {
  return block({
    type: "metric",
    title,
    numeral,
    half,
    state,
    caption: caption ? [seg(caption)] : [],
    source: { kind: "metrics", recordKind: "", count: 0, readable: 0, title: read.title },
    metric: state === "unreadable" ? null : read,
  });
}

const dayOf = (anchor) => anchor - (anchor % DAY);
const dayText = (moment) =>
  new Date(moment * 1000).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    timeZone: "UTC",
  });

// ------------------------------------------------------------------ the frames

function onTimeByWeek() {
  return metric({
    name: "onTimeByWeek",
    form: "trend",
    title: "On time by week, this quarter",
    source: "2,418 deliveries, 9 weeks",
    figures: [
      figure("average", "On time this quarter", pct(0.922)),
      figure("lowest", "Lowest week", pct(0.882)),
      figure("latest", "Last full week", pct(0.924)),
      figure("target", "Target", pct(0.92)),
    ],
    chart: chart({
      unit: "percent",
      ticks: [
        "Sep 28",
        "Oct 5",
        "Oct 12",
        "Oct 19",
        "Oct 26",
        "Nov 2",
        "Nov 9",
        "Nov 16",
        "Nov 23",
      ].map((label) => tick(label)),
      series: [
        series("On time", "now", [0.938, 0.941, 0.935, 0.944, 0.91, 0.882, 0.906, 0.924, 0.95]),
      ],
      marked: [5],
      band: { from: 5, to: 5, low: null, high: null },
      target: { value: "0.92", label: "Target" },
      ghost: true,
    }),
  });
}

function lateByCustomer(locked = false) {
  return metric({
    name: "lateByCustomer",
    form: "bars",
    title: "Late in the week of Nov 2, by customer",
    source: "32 late deliveries, Nov 2 to Nov 8",
    figures: [
      figure("late", "Late that week", count(32)),
      figure("top", "Most at one customer", count(17)),
    ],
    chart: chart({
      unit: "count",
      rows: [
        chartRow({
          label: "Northline Grocers",
          record: record("customer", "cus_beyond_northline", "Northline Grocers"),
          value: count(17),
          marked: true,
        }),
        chartRow({
          label: "Acme Foods",
          record: record("customer", "cus_beyond_acme", "Acme Foods"),
          value: locked ? null : count(7),
        }),
        chartRow({ label: "Harbor Paper Co.", value: count(4) }),
        chartRow({ label: "Delta Tile Supply", value: count(2) }),
        chartRow({ label: "3 other customers", value: count(2) }),
      ],
    }),
  });
}

function agingCompare() {
  return metric({
    name: "agingCompare",
    form: "bars",
    title: "Aging, Nov 23 against Oct 23",
    source: "412 open invoices, as of each date",
    figures: [
      figure("totalNow", "Open now", usd("686810.00")),
      figure("totalThen", "Open Oct 23", usd("621650.00")),
    ],
    chart: chart({
      unit: "money",
      currency: "USD",
      ticks: ["Current", "1–30", "31–60", "61–90", "90+"].map((label, index) =>
        tick(label, index === 2),
      ),
      series: [
        series("Oct", "was", [398700, 151200, 41650, 18900, 11200]),
        series("Nov", "now", [412300, 168450, 74920, 21300, 9840]),
      ],
      marked: [2],
    }),
  });
}

function collecting(anchor) {
  const monthAgo = anchor - 31 * DAY;
  return metric({
    name: "collecting",
    form: "figures",
    title: "Collecting",
    source: "90 days of payments",
    figures: [
      figure("dso", "Days to collect", num(38.4), {
        was: num(34.9),
        wasAt: monthAgo,
        tone: "warn",
      }),
      figure("collected", "Collected of what was due", pct(0.81), {
        was: pct(0.86),
        wasAt: monthAgo,
      }),
    ],
  });
}

const FUEL_RUN = [
  0.613, 0.614, 0.612, 0.615, 0.617, 0.619, 0.618, 0.621, 0.624, 0.626, 0.628, 0.627, 0.63, 0.633,
  0.636, 0.637, 0.639, 0.641, 0.64, 0.643, 0.645, 0.646,
];

function fuelCostPerMile() {
  return metric({
    name: "fuelCostPerMile",
    form: "trend",
    title: "Fuel cost per mile, November",
    source: "1,204 fuel purchases and tractor miles",
    figures: [
      figure("monthToDate", "This month", usd("0.638")),
      figure("lastMonth", "October", usd("0.611")),
      figure("pricePerGallon", "Price per gallon", usd("3.92")),
    ],
    chart: chart({
      unit: "money",
      currency: "USD",
      ticks: FUEL_RUN.map((_, index) => tick(`Nov ${index + 1}`)),
      series: [series("Seven-day run", "now", FUEL_RUN)],
      ref: { value: "0.611", label: "October" },
    }),
  });
}

const FLEET = [
  0.628, 0.635, 0.645, 0.651, 0.583, 0.594, 0.606, 0.616, 0.625, 0.632, 0.644, 0.65, 0.657, 0.665,
  0.676, 0.687, 0.587, 0.6, 0.611, 0.621, 0.629, 0.647, 0.654, 0.661, 0.67, 0.681, 0.693, 0.591,
  0.604, 0.614, 0.623, 0.631, 0.645, 0.652, 0.659, 0.666, 0.674, 0.598,
];

function fuelCostByTractor() {
  const quiet = FLEET.map((cost, index) =>
    chartRow({ label: `T-${String(100 + index * 2).padStart(3, "0")}`, value: usd(cost) }),
  );
  const outlier = (label, cost) =>
    chartRow({
      label,
      record: record("tractor", `trc_beyond_${label.toLowerCase()}`, label),
      value: usd(cost),
      marked: true,
    });
  return metric({
    name: "fuelCostByTractor",
    form: "dots",
    title: "Each tractor this month",
    source: "42 tractors",
    figures: [
      figure("tractorsAboveRange", "Tractors running above the fleet's usual range", count(3)),
    ],
    chart: chart({
      unit: "money",
      currency: "USD",
      rows: [
        ...quiet,
        outlier("T-127", "0.719"),
        outlier("T-104", "0.742"),
        chartRow({
          label: "T-118",
          value: usd("0.781"),
          marked: true,
          dots: [{ at: null, value: "0.781", marked: true, ghost: false, label: "T-118 $0.781" }],
        }),
        outlier("T-131", "0.548"),
      ],
      band: { from: null, to: null, low: "0.574", high: "0.702" },
      ref: { value: "0.638", label: "Fleet" },
    }),
  });
}

function detentionByArrivalHour() {
  return metric({
    name: "detentionByArrivalHour",
    form: "columns",
    title: "Romeo, MI by arrival hour",
    source: "46 stops",
    figures: [figure("longest", "Longest window", dur(15600))],
    chart: chart({
      unit: "duration",
      ticks: ["4 AM", "6 AM", "8 AM", "10 AM", "12 PM"].map((label, index) =>
        tick(label, index === 2),
      ),
      series: [series("Median stop", "now", [7800, 10200, 15600, 14700, 11400])],
      marked: [2],
      ref: { value: "7200", label: "Free" },
    }),
  });
}

function detentionTotals() {
  return metric({
    name: "detentionTotals",
    form: "figures",
    title: "This quarter",
    source: "214 detained stops",
    figures: [
      figure("pastFree", "Past free time", dur(772800), { tone: "warn" }),
      figure("billed", "Billed", usd("9840.00")),
      figure("waived", "Waived", usd("2310.00")),
      figure("noticeOnTime", "Notice on time", pct(0.88)),
    ],
  });
}

const COVER_SLOTS = [
  ["8 AM", "Tue 24"],
  ["12 PM", "Tue 24"],
  ["4 PM", "Tue 24"],
  ["8 PM", "Tue 24"],
  ["12 AM", "Wed 25"],
  ["4 AM", "Wed 25"],
  ["8 AM", "Wed 25"],
  ["12 PM", "Wed 25"],
  ["4 PM", "Wed 25"],
  ["8 PM", "Wed 25"],
  ["12 AM", "Thu 26"],
  ["4 AM", "Thu 26"],
];

const cell = (have, need, state = "covered", needHigh = need) => ({ have, need, needHigh, state });

/** The freight forecast's slot balance: forecast areas by four-hour slot, the next 48 hours. */
function driverCover(rows = 4) {
  const all = [
    {
      label: "Los Angeles, CA",
      cells: [
        cell(9, 8),
        cell(7, 11, "deep"),
        cell(8, 10, "short"),
        cell(6, 6),
        cell(4, 2),
        cell(5, 3),
        cell(8, 7),
        cell(9, 9),
        cell(7, 8, "short"),
        cell(6, 5),
        cell(4, 2),
        cell(4, 1),
      ],
    },
    {
      label: "Omaha, NE",
      cells: [
        cell(6, 5),
        cell(5, 6, "short"),
        cell(6, 6),
        cell(5, 4),
        cell(3, 1),
        cell(3, 2),
        cell(5, 4),
        cell(5, 7, "short"),
        cell(6, 6),
        cell(5, 4),
        cell(3, 1),
        cell(3, 1),
      ],
    },
    {
      label: "Boston, MA",
      cells: [
        cell(4, 4),
        cell(5, 4),
        cell(4, 4, "maybe", 6),
        cell(4, 3),
        cell(2, 1),
        cell(2, 1),
        cell(4, 3),
        cell(4, 4),
        cell(4, 4),
        cell(3, 3),
        cell(2, 1),
        cell(2, 0),
      ],
    },
    {
      label: "Riverside, CA",
      cells: [
        cell(3, 3),
        cell(4, 3),
        cell(3, 4, "short"),
        cell(3, 3),
        cell(2, 1),
        cell(2, 1),
        cell(3, 3),
        cell(3, 3),
        cell(3, 3),
        cell(3, 2),
        cell(2, 1),
        cell(2, 1),
      ],
    },
  ];
  return metric({
    name: "driverCover",
    form: "grid",
    title: "Drivers against need by forecast area, the next 48 hours",
    source: "the freight forecast, 124 forecast areas, the next 48 hours",
    figures: [
      figure("short", "Loads short of a driver", count(11), { tone: "warn" }),
      figure("mayBeShort", "Slots that may be short", count(1)),
      figure("worst", "Short in the worst slot", count(4)),
      figure("spare", "Drivers spare after 48 hours", count(9)),
    ],
    grid: {
      columns: COVER_SLOTS.map(([label, sublabel]) => ({ label, sublabel })),
      rows: rows === 4 ? all : [all[0], all[2]],
    },
  });
}

function coverFigures() {
  const read = driverCover();
  return metric({
    ...read,
    form: "figures",
    grid: null,
    title: "The next 48 hours",
  });
}

function settlementBatch() {
  return metric({
    name: "settlementBatch",
    form: "split",
    title: "Pay period ending Nov 21, 2026",
    source: "183 settlements, paid Nov 27, 2026",
    figures: [
      figure("batchNet", "Net pay", usd("162461.06")),
      figure("batchGross", "Gross earnings", usd("182438.46")),
      figure("batchReimbursements", "Reimbursements", usd("5248.00")),
      figure("batchDeducted", "Deducted", usd("25225.40")),
      figure("batchSettlements", "Settlements", count(183)),
      figure("readyNet", "Ready to release", usd("161476.06")),
      figure("blockedSettlements", "Settlements that cannot be released as they stand", count(9)),
      figure("heldSettlements", "Settlements on hold", count(1)),
      figure("criticalExceptionSettlements", "Settlements with a critical exception", count(8)),
    ],
    split: {
      total: usd("187686.46"),
      parts: [
        { label: "Ready to release", value: usd("161476.06"), kind: "used" },
        { label: "On hold", value: usd("985.00"), kind: "over" },
        { label: "Has a critical exception", value: usd("0.00"), kind: "over" },
        { label: "Deducted", value: usd("25225.40"), kind: "credit" },
      ],
    },
  });
}

function settlementWeeks() {
  return metric({
    name: "settlementWeeks",
    form: "bars",
    title: "The last 5 pay batches",
    source: "6 pay batches",
    figures: [
      figure("largestBatchEarned", "Largest of the five", usd("201478.00")),
      figure("latestBatchEarned", "Newest batch earned", usd("187686.46")),
    ],
    chart: chart({
      unit: "money",
      currency: "USD",
      ticks: ["Oct 18", "Oct 25", "Nov 1", "Nov 8", "Nov 15"].map((label) => tick(label)),
      series: [
        series("Net pay", "stack", [160639.93, 169304.16, 157346.59, 178613.89, 162461.06]),
        series("Deducted", "stack", [25036.66, 24000.75, 27477.13, 23843.69, 25225.4]),
      ],
      marked: [4],
    }),
  });
}

function coverAgainstLoads(anchor, rows = 4) {
  const day = dayOf(anchor);
  const end = day + 30 * DAY;
  const cover = (endsIn) => [{ from: day - 60 * DAY, to: day + endsIn * DAY, kind: "cover" }];
  const load = (inDays, marked = false, label = "") => ({
    at: day + inDays * DAY + 14 * HOUR,
    value: null,
    marked,
    ghost: false,
    label,
  });
  const all = [
    chartRow({
      label: "Ridgeline Freight",
      record: record("carrier", "car_beyond_ridgeline", "Ridgeline Freight"),
      spans: cover(6),
      dots: [
        load(1),
        load(2),
        load(4),
        load(8, true, `S26111040188306 delivers ${dayText(day + 8 * DAY)}`),
      ],
    }),
    chartRow({
      label: "Copperhead Transport",
      record: record("carrier", "car_beyond_copperhead", "Copperhead Transport"),
      spans: cover(10),
      dots: [load(7)],
    }),
    chartRow({
      label: "Blue Line Freight",
      record: record("carrier", "car_beyond_blueline", "Blue Line Freight"),
      spans: cover(18),
      dots: [load(1), load(3)],
    }),
    chartRow({
      label: "Sandoval Trucking",
      record: record("carrier", "car_beyond_sandoval", "Sandoval Trucking"),
      spans: cover(28),
    }),
  ];
  return metric({
    name: "coverAgainstLoads",
    form: "spans",
    title: "Their loads against the end of cover",
    source: "7 loads on carrier assignments",
    figures: [
      figure("carriers", "Carriers", count(4)),
      figure("pastCover", "Past cover", count(1)),
    ],
    chart: chart({
      unit: "count",
      ticks: [7, 14, 21].map((days) => at(day + days * DAY)).concat([at(end)]),
      rows: rows === 4 ? all : all.slice(0, 2),
    }),
  });
}

const h = (hours, minutes = 0) => dur(hours * HOUR + minutes * MIN);
const tc = (cellValue, mark = "") => ({ value: cellValue, text: "", mark });
const off = { value: null, text: "", mark: "off" };

function rotaWeek(rows = 6) {
  const people = [
    [
      "Hector Alvarez",
      [
        tc(h(11, 30), "hot"),
        tc(h(10)),
        off,
        tc(h(9)),
        tc(h(9)),
        off,
        tc(h(9), "w"),
        tc(h(48, 30), "w"),
      ],
    ],
    [
      "Janet Cole",
      [tc(h(10), "hot"), tc(h(10)), tc(h(8)), off, tc(h(8)), tc(h(8), "w"), off, tc(h(44), "w")],
    ],
    [
      "Dmitri Volkov",
      [
        tc(h(9, 45), "hot"),
        tc(h(8)),
        tc(h(8)),
        tc(h(8, 30)),
        off,
        tc(h(8), "w"),
        off,
        tc(h(42, 15), "w"),
      ],
    ],
    [
      "Sam Whitaker",
      [tc(h(8), "hot"), tc(h(8)), tc(h(8)), tc(h(8)), tc(h(8)), off, off, tc(h(40))],
    ],
    ["Rosa Lind", [off, off, tc(h(8)), tc(h(8)), off, off, off, tc(h(16))]],
    ["Kwame Boateng", [off, tc(h(8)), tc(h(8)), tc(h(8)), tc(h(8)), off, off, tc(h(32))]],
  ];
  return metric({
    name: "rotaWeek",
    form: "table",
    title: "The week, day by day",
    source: "6 workers' rota rows",
    figures: [figure("over", "Over the line", count(3)), figure("room", "With room", count(2))],
    table: {
      columns: [
        { label: "Person", kind: "text" },
        ...["Sun 22", "Mon 23", "Tue 24", "Wed 25", "Thu 26", "Fri 27", "Sat 28", "Week"].map(
          (label) => ({ label, kind: "value" }),
        ),
      ],
      rows: people
        .slice(0, rows)
        .map(([name, cells]) => [{ value: null, text: name, mark: "" }, ...cells]),
    },
  });
}

function iftaReturnChecks() {
  const shipment = (pro) => record("shipment", `shp_beyond_${pro.slice(-6)}`, pro);
  return metric({
    name: "iftaReturnChecks",
    form: "checks",
    title: "Before it can be filed",
    source: "the Q3 2026 return and its worksheet",
    figures: [figure("blocking", "Blocking", count(1))],
    checks: [
      {
        title: "Missing tax rate",
        tone: "bad",
        detail:
          "Oklahoma has no diesel rate for Q3 2026, so its line cannot be priced and the return cannot be finalized.",
        count: count(1),
        was: null,
        records: [record("ifta_jurisdiction", "ifj_beyond_ok", "Oklahoma")],
        hidden: 0,
      },
      {
        title: "Miles not attributed to a jurisdiction",
        tone: "warn",
        detail: "These moves ran with no route, so their miles sit outside every line.",
        count: { kind: "text", number: "0", currency: "", text: "412 mi" },
        was: null,
        records: [
          shipment("S26090038817402"),
          shipment("S26090039120533"),
          shipment("S26090039406118"),
        ],
        hidden: 0,
      },
      {
        title: "Past the filing date",
        tone: "warn",
        detail: "The return is still a draft; the quarter's filing date was Oct 31.",
        count: dur(23 * DAY),
        was: null,
        records: [],
        hidden: 0,
      },
      {
        title: "No miles without a tractor",
        tone: "ok",
        detail: "",
        count: count(0),
        was: null,
        records: [],
        hidden: 0,
      },
      {
        title: "Jurisdiction miles match the moves",
        tone: "ok",
        detail: "",
        count: count(186420),
        was: null,
        records: [],
        hidden: 0,
      },
    ],
  });
}

function periodClose() {
  return metric({
    name: "periodClose",
    form: "checks",
    title: "What holds October",
    source: "the October 2026 period and its close checks",
    figures: [figure("blocking", "Blocking", count(2))],
    checks: [
      {
        title: "Accounting sources unposted",
        tone: "bad",
        detail:
          "The invoices and the settlement dated in October have entries drafted but not posted.",
        count: count(3),
        was: null,
        records: [],
        hidden: 0,
      },
      {
        title: "Manual journal requests waiting",
        tone: "bad",
        detail: "The October fuel accrual was submitted Nov 18 and has no approver yet.",
        count: count(1),
        was: null,
        records: [],
        hidden: 0,
      },
      {
        title: "No journal reversals waiting",
        tone: "ok",
        detail: "",
        count: count(0),
        was: null,
        records: [],
        hidden: 0,
      },
      {
        title: "Posted invoices reconcile",
        tone: "ok",
        detail: "",
        count: count(0),
        was: null,
        records: [],
        hidden: 0,
      },
    ],
  });
}

function partnerDay(anchor, entries = 6) {
  const day = dayOf(anchor);
  const shipment = (pro) =>
    record("shipment", `shp_beyond_${pro.slice(-6)}`, pro, "Acme Foods", "New");
  const all = [
    {
      at: day + 6 * HOUR + 2 * MIN,
      tone: "",
      text: "Tender 88214 became S26110041402218; our acceptance was sent and acknowledged",
      detail: "204 in, 990 out, 997 accepted",
      records: [shipment("S26110041402218")],
    },
    {
      at: day + 6 * HOUR + 2 * MIN,
      tone: "",
      text: "Tender 88215 became S26110041402226; our acceptance was sent and acknowledged",
      detail: "204 in, 990 out, 997 accepted",
      records: [shipment("S26110041402226")],
    },
    {
      at: day + 7 * HOUR + 40 * MIN,
      tone: "warn",
      text: "Tender 88231 was quarantined: ship-to 0412 is not a location we know",
      detail: "Inbound file quarantined, no shipment made",
      records: [],
    },
    {
      at: day + 8 * HOUR + 15 * MIN,
      tone: "",
      text: "Tender 88240 became S26110041402301; our acceptance was sent",
      detail: "204 in, 990 out, acknowledgment pending",
      records: [shipment("S26110041402301")],
    },
    {
      at: day + 8 * HOUR + 52 * MIN,
      tone: "",
      text: "Tender 88243 became S26110041402319; our acceptance was sent and acknowledged",
      detail: "204 in, 990 out, 997 accepted",
      records: [shipment("S26110041402319")],
    },
    {
      at: day + 9 * HOUR + 5 * MIN,
      tone: "warn",
      text: "Tender 88244 became S26110041402327, but our acceptance did not reach Acme",
      detail: "990 failed: SFTP login refused, retrying at 9:50 AM",
      records: [shipment("S26110041402327")],
    },
  ];
  return metric({
    name: "partnerDay",
    form: "log",
    title: "Today's tenders from Acme Foods",
    source: "6 inbound tenders and 5 responses",
    figures: [
      figure("tendersIn", "Tenders in", count(6)),
      figure("shipmentsMade", "Shipments made", count(5)),
      figure("answersDelivered", "Answers delivered", count(4)),
      figure("medianAnswer", "Median time to answer", dur(3 * MIN)),
    ],
    log: all.slice(0, entries),
  });
}

function partnerFigures(anchor) {
  return metric({
    ...partnerDay(anchor),
    form: "figures",
    log: [],
    title: "Acme Foods today",
    source: "11 EDI messages",
  });
}

// ------------------------------------------------------------------ the boards

function board({ key, agent, question, title, verdict, blocks, anchor, minutesAgo = 12 }) {
  return {
    artifactId: `aart_beyond_${key}`,
    threadId: `athr_desk_board_${key}`,
    title,
    author: agent.name,
    question,
    answeredAt: anchor - minutesAgo * MIN,
    readAt: anchor,
    place: "conversation",
    readings: [],
    markdown: `# ${title}\n\n${verdict.map((part) => part.text).join("")}\n`,
    verdict,
    blocks,
  };
}

const FRAMES = {
  ontime: {
    agent: BEYOND_AGENTS.analyst,
    card: (state) =>
      board({
        key: "ontime",
        agent: BEYOND_AGENTS.analyst,
        anchor: state.anchor,
        question: "How has on-time delivery trended this quarter, and where did it slip?",
        title: "On time this quarter",
        verdict: [
          seg("On-time delivery has averaged 92.2% this quarter. "),
          seg("It slipped in the week of Nov 2", 1, "warn"),
          seg(", to 88.2%, and is still under target. "),
          seg("Northline Grocers held most of that week's late deliveries", 2),
          seg(", 17 of 32."),
        ],
        blocks: [
          exhibit({
            numeral: 1,
            title: "On time by week, this quarter",
            caption:
              "The dashed line is the on-time target; this week, still running, is the ghost.",
            read: onTimeByWeek(),
          }),
          exhibit({
            numeral: 2,
            title: "Late in the week of Nov 2, by customer",
            read: lateByCustomer(),
          }),
        ],
      }),
  },
  aging: {
    agent: BEYOND_AGENTS.receivables,
    card: (state) =>
      board({
        key: "aging",
        agent: BEYOND_AGENTS.receivables,
        anchor: state.anchor,
        question: "How does our receivable aging compare with last month?",
        title: "Aging against last month",
        verdict: [
          seg("Receivables grew to $686,810 from $621,650 a month ago, and "),
          seg("the 31 to 60 day bucket nearly doubled", 1, "warn"),
          seg(". "),
          seg("Days to collect rose to 38.4", 2),
          seg("."),
        ],
        blocks: [
          exhibit({
            numeral: 1,
            title: "Aging, Nov 23 against Oct 23",
            caption: "Each bucket's last month light and this month dark, by days past due.",
            read: agingCompare(),
          }),
          exhibit({ numeral: 2, title: "Collecting", half: true, read: collecting(state.anchor) }),
        ],
      }),
  },
  fuelcost: {
    agent: BEYOND_AGENTS.fuel,
    card: (state) =>
      board({
        key: "fuelcost",
        agent: BEYOND_AGENTS.fuel,
        anchor: state.anchor,
        question:
          "How has fuel cost per mile moved this month, and which tractors are off the fleet's line?",
        title: "Fuel cost per mile",
        verdict: [
          seg("Fuel cost per mile rose every week this month, "),
          seg("to $0.646 from October's $0.611", 1),
          seg(". "),
          seg("Three tractors run above the fleet's usual range", 2, "warn"),
          seg(", T-118 the furthest at $0.781."),
        ],
        blocks: [
          exhibit({
            numeral: 1,
            title: "Fuel cost per mile, November",
            caption: "A seven-day run of fuel bought over miles run; the dashed line is October's.",
            read: fuelCostPerMile(),
          }),
          exhibit({
            numeral: 2,
            title: "Each tractor this month",
            caption:
              "Each dot is a tractor; the shaded band is where the middle four fifths of the fleet runs.",
            read: fuelCostByTractor(),
          }),
        ],
      }),
  },
  facilities: {
    agent: BEYOND_AGENTS.detention,
    card: (state) =>
      board({
        key: "facilities",
        agent: BEYOND_AGENTS.detention,
        anchor: state.anchor,
        question: "Which facilities cost us the most waiting time this quarter?",
        title: "Waiting by facility",
        verdict: [
          seg("At Romeo, trucks that arrive "),
          seg("between 8 and 10 AM wait longest", 1, "warn"),
          seg(", a median of 4h 20m against 2h free. "),
          seg("This quarter's waiting past free time comes to 214h 40m", 2),
          seg("."),
        ],
        blocks: [
          exhibit({
            numeral: 1,
            title: "Romeo, MI by arrival hour",
            half: true,
            caption:
              "The median stop by the two hours the truck arrived in; above the dashed line is past free time.",
            read: detentionByArrivalHour(),
          }),
          exhibit({ numeral: 2, title: "This quarter", half: true, read: detentionTotals() }),
        ],
      }),
  },
  coverage: {
    agent: BEYOND_AGENTS.coverage,
    card: (state) =>
      board({
        key: "coverage",
        agent: BEYOND_AGENTS.coverage,
        anchor: state.anchor,
        question: "Where are we going to be short on drivers over the next couple of days?",
        title: "Drivers short, the next 48 hours",
        verdict: [
          seg("The next 48 hours are 11 loads short of a driver, "),
          seg("worst in Los Angeles on Tuesday from 12 PM, 4 short", 1, "warn"),
          seg(". Boston may be up to 2 short Tuesday afternoon if freight comes in high."),
        ],
        blocks: [
          exhibit({
            numeral: 1,
            title: "Drivers against need by forecast area, the next 48 hours",
            caption:
              "Each forecast area's drivers against the loads the forecast expects, in four-hour slots; hatched is short, outlined may be short.",
            read: driverCover(),
          }),
          exhibit({ numeral: 2, title: "The next 48 hours", half: true, read: coverFigures() }),
        ],
      }),
  },
  settlements: {
    agent: BEYOND_AGENTS.settlements,
    card: (state) =>
      board({
        key: "settlements",
        agent: BEYOND_AGENTS.settlements,
        anchor: state.anchor,
        question: "What does this week's settlement batch come to, and what is holding any of it?",
        title: "This week's settlement batch",
        verdict: [
          seg("The batch paid Nov 27 comes to "),
          seg("$162,461.06 net for 183 settlements", 1),
          seg(". "),
          seg("1 is on hold for a rate dispute", 1, "warn"),
          seg(" and "),
          seg("8 have a critical exception", 1, "warn"),
          seg(", so they cannot be released as they stand; the newest week is "),
          seg("below the largest of the last five", 2),
          seg("."),
        ],
        blocks: [
          exhibit({
            numeral: 1,
            title: "The newest batch",
            half: true,
            caption:
              "Earnings and reimbursements, split into what drivers take home and what is deducted.",
            read: settlementBatch(),
          }),
          exhibit({
            numeral: 2,
            title: "The last five batches",
            half: true,
            caption: "Earned and reimbursed each week: net below, deducted above.",
            read: settlementWeeks(),
          }),
        ],
      }),
  },
  carriers: {
    agent: BEYOND_AGENTS.carriers,
    card: (state) =>
      board({
        key: "carriers",
        agent: BEYOND_AGENTS.carriers,
        anchor: state.anchor,
        question: "Which carriers hauling for us have insurance about to lapse?",
        title: "Insurance lapsing",
        verdict: [
          seg("Four carriers' cover ends in the next 30 days. "),
          seg("Ridgeline Freight's ends first, and one of its loads delivers after it", 1, "warn"),
          seg("."),
        ],
        blocks: [
          exhibit({
            numeral: 1,
            title: "Their loads against the end of cover",
            caption: "Each load where it delivers; the line is the cover, the mark where it ends.",
            read: coverAgainstLoads(state.anchor),
          }),
        ],
      }),
  },
  overtime: {
    agent: BEYOND_AGENTS.workforce,
    card: (state) =>
      board({
        key: "overtime",
        agent: BEYOND_AGENTS.workforce,
        anchor: state.anchor,
        question: "Who is going to run into overtime this week?",
        title: "Overtime this week",
        verdict: [
          seg("Three people cross the 40-hour line this week on the rota as it stands; "),
          seg("Hector Alvarez the furthest, at 48h 30m", 1, "warn"),
          seg(". Rosa Lind and Kwame Boateng have room."),
        ],
        blocks: [
          exhibit({
            numeral: 1,
            title: "The week, day by day",
            caption:
              "Sunday is worked; the rest is the rota. The shift that crosses the line is marked, and the last two have room.",
            read: rotaWeek(),
          }),
        ],
      }),
  },
  ifta: {
    agent: BEYOND_AGENTS.fuel,
    card: (state) =>
      board({
        key: "ifta",
        agent: BEYOND_AGENTS.fuel,
        anchor: state.anchor,
        question: "Are we ready to file this quarter's IFTA?",
        title: "Ready to file?",
        verdict: [
          seg("Not yet: "),
          seg("Oklahoma has no diesel rate for the quarter", 1, "bad"),
          seg(", so its line cannot be priced."),
        ],
        blocks: [
          exhibit({ numeral: 1, title: "Before it can be filed", read: iftaReturnChecks() }),
        ],
      }),
  },
  edi: {
    agent: BEYOND_AGENTS.edi,
    card: (state) =>
      board({
        key: "edi",
        agent: BEYOND_AGENTS.edi,
        anchor: state.anchor,
        question: "Did all of today's tenders from Acme Foods go through?",
        title: "Today's tenders",
        verdict: [
          seg("Four of Acme's six tenders went through. "),
          seg("One was quarantined and one acceptance did not reach them", 1, "warn"),
          seg("."),
        ],
        blocks: [
          exhibit({
            numeral: 1,
            title: "Today's tenders from Acme Foods",
            read: partnerDay(state.anchor),
          }),
          exhibit({
            numeral: 2,
            title: "Acme Foods today",
            half: true,
            read: partnerFigures(state.anchor),
          }),
        ],
      }),
  },
  close: {
    agent: BEYOND_AGENTS.books,
    card: (state) =>
      board({
        key: "close",
        agent: BEYOND_AGENTS.books,
        anchor: state.anchor,
        question: "What is still blocking October close?",
        title: "What holds October",
        verdict: [
          seg("Two things hold October: "),
          seg("three sources are unposted and one journal request waits", 1, "bad"),
          seg("."),
        ],
        blocks: [exhibit({ numeral: 1, title: "What holds October", read: periodClose() })],
      }),
  },
};

/** The Measures frame: every form once, at real size. */
function measuresCard(state) {
  const stage = state.scenario.deskBoard ?? "live";
  const locked = stage === "locked";
  const empty = stage === "empty";
  const drained = (read) =>
    empty
      ? {
          ...read,
          chart: read.chart ? { ...read.chart, series: [], rows: [] } : null,
          split: read.split ? { ...read.split, parts: [] } : null,
          grid: read.grid ? { ...read.grid, rows: [] } : null,
          log: [],
          checks: [],
          table: read.table ? { ...read.table, rows: [] } : null,
        }
      : read;
  let numeral = 0;
  const next = (fields) => {
    numeral += 1;
    const state = empty && fields.read.form !== "figures" ? "empty" : (fields.state ?? "ready");
    return exhibit({ numeral, ...fields, state, read: drained(fields.read) });
  };
  return board({
    key: "beyond",
    agent: BEYOND_AGENTS.analyst,
    anchor: state.anchor,
    question: "Show me every form a board can draw.",
    title: "Measures and charts",
    verdict: [seg("Every form a metric is drawn in, once, with the numbers of its frame.")],
    blocks: [
      next({ title: "Trend with the slip marked", read: onTimeByWeek() }),
      next({ title: "Period against period", read: agingCompare() }),
      next({ title: "Ranked bars", read: lateByCustomer(locked) }),
      next({
        title: "Spread across units",
        read: fuelCostByTractor(),
        state: locked ? "unreadable" : "ready",
      }),
      next({ title: "Stacked periods", half: true, read: settlementWeeks() }),
      next({ title: "Columns over an allowance", half: true, read: detentionByArrivalHour() }),
      next({ title: "Events against a bound", read: coverAgainstLoads(state.anchor, 2) }),
      next({ title: "Grid of cover", read: driverCover(2) }),
      next({ title: "Parts of a whole", half: true, read: settlementBatch() }),
      next({ title: "A readout", half: true, read: detentionTotals() }),
      next({ title: "A log", read: partnerDay(state.anchor, 3) }),
      next({ title: "A table", read: rotaWeek(3) }),
      next({ title: "Checks", read: periodClose() }),
    ],
  });
}

/**
 * Real-shaped data at half width, as the bench's boards drew it: two exhibits side by side
 * (about 349px of content each). Sixty tractors with many marked, six carriers with long names
 * and a load past cover, a fuel run whose latest point sits on the line, ten customers with long
 * names and the catch-all row, and settlements whose second fact is a sentence.
 */
function halfWidthCard(state) {
  const day = dayOf(state.anchor);
  const costs = Array.from({ length: 60 }, (_, index) =>
    Number((0.48 + ((index * 37) % 61) / 1000 + (index % 7) / 400).toFixed(3)),
  );
  const tractors = chart({
    unit: "money",
    currency: "USD",
    rows: costs.map((cost, index) => {
      const unit = String([1092, 10051, 1211, 1040, 1177, 1205][index % 6] + index * 3);
      const marked = index % 4 === 0;
      return chartRow({
        label: unit,
        value: usd(cost),
        marked,
        dots: [{ at: null, value: String(cost), marked, ghost: index === 59, label: unit }],
      });
    }),
    band: { from: null, to: null, low: "0.489", high: "0.597" },
    ref: { value: "0.543", label: "Fleet" },
  });
  const carrierNames = [
    "Ridgeline Freight Systems",
    "Copperhead Transport Logistics",
    "Blue Line Freight",
    "Sandoval Trucking Company",
    "Great Plains Refrigerated",
    "Northwoods Carrier Services",
  ];
  const loads = chart({
    unit: "count",
    ticks: [2, 6, 10, 14, 18, 22, 26, 29, 31].map((days, index) => ({
      at: day + days * DAY,
      label: "",
      marked: index === 7,
    })),
    rows: carrierNames.map((label, index) =>
      chartRow({
        label,
        spans: [{ from: day - 60 * DAY, to: day + (6 + index * 4) * DAY, kind: "cover" }],
        dots:
          index === 0
            ? [
                { at: day + 2 * DAY, value: null, marked: false, ghost: false, label: "" },
                {
                  at: day + 9 * DAY,
                  value: null,
                  marked: true,
                  ghost: false,
                  label: `SHP-CAR-000123 ${dayText(day + 9 * DAY)}`,
                },
              ]
            : [
                {
                  at: day + (3 + index * 3) * DAY,
                  value: null,
                  marked: false,
                  ghost: false,
                  label: "",
                },
              ],
      }),
    ),
  });
  const run = Array.from({ length: 22 }, (_, index) => 0.47 + index * 0.001 + (index % 3) / 500);
  const fuel = chart({
    unit: "money",
    currency: "USD",
    ticks: run.map((_, index) => tick(`Nov ${index + 1}`)),
    series: [series("Seven-day run", "now", [...run.slice(0, 21), 0.492])],
    ref: { value: "0.488", label: "October" },
  });
  const customers = [
    ["Rodriguez & Evans Distribution", 14],
    ["Northline Grocers Cooperative", 11],
    ["Acme Foods", 9],
    ["Harbor Paper Company of Louisiana", 7],
    ["Delta Tile Supply", 6],
    ["Summit Peak Building Materials", 5],
    ["Gulfport Cold Storage", 4],
    ["Pine Ridge Mills", 3],
    ["Kessler Farms", 3],
  ];
  const late = chart({
    unit: "count",
    rows: [
      ...customers.map(([label, late], index) =>
        chartRow({ label, value: count(late), marked: index === 0 }),
      ),
      chartRow({ label: "38 other customers", value: count(71), rest: true }),
    ],
  });
  const fact = (text) => ({
    text: "{0}",
    value: { kind: "text", number: "0", currency: "", text },
    at: null,
    day: false,
    place: "",
  });
  const settlement = (id, number, driver, net, sentence) => ({
    kind: "driver_settlement",
    id,
    readable: true,
    label: number,
    subtitle: driver,
    status: "On hold",
    statusTone: "warn",
    opens: null,
    measure: {
      measure: "netPay",
      value: usd(net),
      standing: { text: "{0}", value: usd(net), at: null, day: false, place: "" },
      fact: fact(sentence),
      tone: "",
      bar: null,
    },
  });
  const moment = (id, number, driver, net) => ({
    ...settlement(id, number, driver, net, ""),
    status: "Approved",
    statusTone: "ok",
    measure: {
      ...settlement(id, number, driver, net, "").measure,
      fact: {
        text: "Approved {0}",
        value: null,
        at: state.anchor - 2 * HOUR,
        day: false,
        place: "",
      },
    },
  });
  const rows = [
    settlement(
      "dstl_half_1",
      "STL2611000412",
      "Omar Haddad-Fitzgerald",
      "1844.20",
      "On hold: Rate dispute, Bench: waiting on the broker's corrected rate confirmation",
    ),
    settlement(
      "dstl_half_2",
      "STL2611000418",
      "Brian O'Callaghan",
      "-3338.70",
      "Has a critical exception: Deductions exceed earnings; 3338.79 USD carries forward",
    ),
    {
      ...settlement(
        "dstl_half_4",
        "STL2611000430",
        "MC 690441",
        "980.00",
        "Auto liability ended Oct 20",
      ),
    },
    moment("dstl_half_3", "STL2611000421", "Ana Lucia Ferreira", "2210.55"),
  ];
  const records = block({
    type: "records",
    title: "Held settlements",
    numeral: 5,
    half: true,
    measure: "netPay",
    source: {
      kind: "records",
      recordKind: "driver_settlement",
      count: rows.length,
      readable: rows.length,
      title: "",
    },
    metric: null,
    rows,
  });
  return board({
    key: "halfwidth",
    agent: BEYOND_AGENTS.fuel,
    anchor: state.anchor,
    question: "Show me the half-width boards from the bench.",
    title: "Half width, real-shaped",
    verdict: [seg("Every crowded form at half width, with the bench's shapes of data.")],
    blocks: [
      exhibit({
        numeral: 1,
        title: "Each tractor this month",
        half: true,
        read: metric({
          name: "fuelCostByTractor",
          form: "dots",
          title: "Each tractor",
          chart: tractors,
        }),
      }),
      exhibit({
        numeral: 2,
        title: "Their loads against the end of cover",
        half: true,
        read: metric({ name: "coverAgainstLoads", form: "spans", title: "Cover", chart: loads }),
      }),
      exhibit({
        numeral: 3,
        title: "Fuel cost per mile",
        half: true,
        read: metric({
          name: "fuelCostPerMile",
          form: "trend",
          title: "Fuel cost per mile",
          chart: fuel,
        }),
      }),
      exhibit({
        numeral: 4,
        title: "Late last week, by customer",
        half: true,
        read: metric({
          name: "lateByCustomer",
          form: "bars",
          title: "Late by customer",
          chart: late,
        }),
      }),
      records,
      exhibit({
        numeral: 6,
        title: "A wide grid",
        half: true,
        read: driverCover(),
      }),
    ],
  });
}

/** Every beyond conversation, its agent and its board, keyed by the thread's suffix. */
export const BEYOND_FRAMES = {
  ...FRAMES,
  beyond: { agent: BEYOND_AGENTS.analyst, card: measuresCard },
  halfwidth: { agent: BEYOND_AGENTS.fuel, card: halfWidthCard },
};
