// Artifact catalogue for the demo — one of every kind Desk renders.
const KIND = {
  table: { label: "Table", icon: "a-table" },
  record: { label: "Record", icon: "a-record" },
  rate: { label: "Rate explanation", icon: "a-rate" },
  email: { label: "Email draft", icon: "a-mail" },
  plan: { label: "Plan", icon: "a-plan" },
  report: { label: "Report", icon: "a-report" },
  diff: { label: "Run diff", icon: "a-diff" },
  doc: { label: "Document", icon: "a-doc" },
  view: { label: "View", icon: "a-view" },
  decision: { label: "Decision", icon: "a-decision" },
  extract: { label: "Extraction", icon: "a-scan" },
  bill: { label: "Billing item", icon: "a-bill" },
};

const QUEUE_COLS = [
  { k: "id", l: "Item", t: "id" }, { k: "cust", l: "Customer" }, { k: "biller", l: "Biller", t: "opt", none: "No biller" },
  { k: "amt", l: "Amount", t: "money" }, { k: "status", l: "Status", t: "pill" },
];
const queueRows = f => ctx => QUEUE.filter(f).map(r => ({
  ...r, biller: r.biller || (ctx.assigned ? "Avery Lane" : null), status: ctx.posted ? "Posted" : r.status,
  _chg: { biller: !r.biller && ctx.assigned && !ctx.posted, status: ctx.posted },
}));

const BOARD = [
  ["SEED-SHP-001", "Chicago, IL → Columbus, OH", "Marcus Hill", "Today 4:10 PM", "In transit"],
  ["SEED-SHP-002", "Des Moines, IA → Omaha, NE", "Lena Park", "Today 6:45 PM", "Late"],
  ["SEED-SHP-003", "Joliet, IL → Indianapolis, IN", null, "Tomorrow 9:00 AM", "Needs driver"],
  ["SEED-SHP-004", "Gary, IN → Detroit, MI", null, "Tomorrow 11:30 AM", "Needs driver"],
  ["SEED-SHP-005", "Cedar Rapids, IA → Madison, WI", "Tom Reyes", "Today 7:20 PM", "Late"],
  ["SEED-SHP-006", "St. Louis, MO → Memphis, TN", "Ana Cruz", "Today 3:00 PM", "Delivered"],
  ["SEED-SHP-007", "Ames, IA → Minneapolis, MN", "Will Grant", "Today 8:05 PM", "Late"],
  ["SEED-SHP-008", "Chicago, IL → Columbus, OH", "Dana Ortiz", "Today 5:30 PM", "In transit"],
].map(([id, lane, driver, eta, st]) => ({ id, lane, driver, eta, st }));

const ART2 = {
  board: {
    kind: "table", title: "Today's board", at: "9:58 PM", src: "shipments.list", verb: "read", searched: ["pickup today", "delivery today"],
    cols: [{ k: "id", l: "Shipment", t: "id" }, { k: "lane", l: "Lane" }, { k: "driver", l: "Driver", t: "opt", none: "Unassigned" }, { k: "eta", l: "ETA", t: "mono" }, { k: "st", l: "Status", t: "pill" }],
    rows: () => BOARD, total: 42, slug: "todays-board",
  },
  shp: {
    kind: "record", title: "SEED-SHP-008", at: "10:04 PM", src: "shipments.get", verb: "opened", slug: "seed-shp-008",
    rec: {
      status: "In transit", customer: "Acme Manufacturing", progress: 0.62, eta: "5:30 PM", late: false,
      from: { city: "Chicago, IL", place: "Acme DC 4", time: "Picked up 9:12 AM" }, to: { city: "Columbus, OH", place: "Acme Plant 2", time: "Appt 5:30 PM" },
      fields: [["Driver", "Dana Ortiz"], ["Tractor", "TRK-2214"], ["Equipment", "53' Dry van"], ["Weight", "38,400 lb"], ["Miles", "356 loaded"], ["Rate", "$1,486.00"], ["Reference", "PO 77812"], ["Last ping", "I-70 E, 4 min ago"]],
    },
  },
  plan: {
    kind: "plan", title: "Cover the open loads", at: "10:06 PM", src: "planner.draft", verb: "drafted", slug: "cover-open-loads",
    summary: "Two loads still need a driver. Assign the closest drivers who have hours left, then tell both customers.",
    steps: [
      { tool: "drivers.hos", text: "Check which drivers have hours left today", st: "done" },
      { tool: "shipments.assign", text: "Assign Marcus Hill to SEED-SHP-003 (Joliet)", st: "done" },
      { tool: "shipments.assign", text: "Assign Dana Ortiz to SEED-SHP-004 (Gary)", st: "done" },
      { tool: "messages.send", text: "Send both customers their driver and ETA", st: "wait" },
      { tool: "watch.create", text: "Watch both loads for weather delays", st: "next" },
    ],
  },
  rate: {
    kind: "rate", title: "Rate for SEED-SHP-008", at: "10:14 PM", src: "rating.explain", verb: "explained", slug: "rate-seed-shp-008",
    winner: { code: "ACME-2026", name: "Acme contract 2026", rule: "Dry van · Midwest lanes" },
    tie: "Picked over the spot tariff because a contract rule matched first.",
    components: [["Linehaul", "356 mi × $2.85", 1014.6], ["Fuel surcharge", "34% of linehaul", 344.96], ["Detention", "1.5 h × $65", 97.5], ["Stop-off", "1 extra stop", 75]],
    guard: { kind: "Minimum charge", bound: 850, raw: 1532.06, result: "Kept — above minimum" },
    discount: ["Volume discount", "−3% contract", -46.06],
    rejected: [["SPOT-TARIFF", "Spot · all lanes", "Lower priority than a matching contract"], ["ACME-2025", "Dry van · Midwest", "Expired Sep 30"]],
    warnings: ["Detention is estimated — the driver hasn't checked out yet."],
  },
  email: {
    kind: "email", title: "Delay notice to Acme", at: "10:18 PM", src: "messages.draft", verb: "drafted", slug: "delay-notice-acme",
    to: ["jordan.pike@acme.com", "dock@acme.com"], subject: "SEED-SHP-002 running about 2 hours late",
    body: "Hi Jordan,\n\nSEED-SHP-002 is moving again after weather held it outside Des Moines. Our driver Lena Park now expects to reach Omaha around 6:45 PM, about two hours after the original appointment.\n\nWe've let your dock know and will confirm once she's 30 minutes out. Sorry for the shuffle.\n\nAvery Lane\nTrenova Logistics",
    why: "Leads with the new ETA, names the cause once, and promises a follow-up. That's what Acme asked for in their last two delay replies.",
  },
  report: {
    kind: "report", title: "Detention · last week", at: "10:20 PM", src: "reports.run", verb: "ran", slug: "detention-last-week",
    dataset: "Stops · Sep 22 – Sep 28", rowCount: 1204, measure: "Hours",
    rows: [["Acme Manufacturing", 41.5, 2697.5], ["Bluewater Retail", 28.0, 1820], ["Granite Building", 19.25, 1251.25], ["Harbor Supply Co.", 12.5, 812.5], ["Northline Foods", 9.0, 585], ["Peak Distributing", 4.75, 308.75]],
  },
  diff: {
    kind: "diff", title: "AR aging · this week vs last", at: "10:24 PM", src: "reports.compare", verb: "compared", slug: "ar-aging-diff",
    before: "Sep 25", after: "Oct 2", counts: { added: 3, removed: 1, changed: 6, unchanged: 41 },
    totals: [["Current", 182400, 196250], ["31–60 days", 48900, 41200], ["61–90 days", 12300, 15800], ["90+ days", 6400, 6400]],
    changes: [["added", "Peak Distributing", "Current", 0, 4210], ["changed", "Acme Manufacturing", "31–60 days", 18300, 9800], ["changed", "Harbor Supply Co.", "61–90 days", 2100, 5600], ["removed", "Lakeside Grocers", "90+ days", 1850, 0], ["changed", "Bluewater Retail", "Current", 22400, 27150]],
  },
  doc: {
    kind: "doc", title: "Storm impact summary", at: "10:26 PM", src: "notes.write", verb: "wrote", slug: "storm-impact",
  },
  view: {
    kind: "view", title: "Late loads in the Midwest", at: "10:28 PM", src: "views.compose", verb: "built", slug: "late-midwest", entity: "Shipments",
    explanation: ["Shipments that are ", { t: "running late" }, ", picking up or delivering in ", { t: "IA, IL, MN, NE or WI" }, ", with a ", { t: "delivery appointment today" }, "."],
    filters: 3, unresolved: [["storm-affected", "There's no weather field on shipments yet"]], count: 3,
  },
  queue: {
    kind: "table", title: "Billing queue", at: "10:32 PM", src: "billing.queue.list", verb: "read", searched: ["ready to invoice", "billing queue"],
    cols: QUEUE_COLS, rows: queueRows(() => true), slug: "billing-queue",
  },
  gaps: {
    kind: "table", title: "Missing a biller", at: "10:36 PM", src: "billing.queue.inspect", verb: "read", calls: 11, searched: ["biller is empty"],
    cols: QUEUE_COLS, rows: queueRows(r => !QUEUE.find(q => q.id === r.id).biller), slug: "missing-biller",
  },
  dec: { kind: "decision", title: "Assign biller · 11 items", at: "10:37 PM", src: "billing.assign_biller", verb: "proposed", slug: "assign-biller", d: "d1" },
  dec2: { kind: "decision", title: "Post 15 invoices", at: "10:41 PM", src: "billing.post_invoices", verb: "proposed", slug: "post-invoices", d: "d2" },
};

// When each of this conversation's artifacts was made, as the turn that made it.
const TURN_OF = { board: "9:58 PM · Read today's loads", shp: "10:04 PM · Checked driver hours", plan: "10:06 PM · Drafted a plan", rate: "10:14 PM · Explained a rate", email: "10:18 PM · Drafted a delay notice", report: "10:20 PM · Ran the detention report", diff: "10:24 PM · Compared AR aging", doc: "10:26 PM · Wrote a storm summary", view: "10:28 PM · Built a view", queue: "10:32 PM · Read the billing queue", gaps: "10:36 PM · Checked the billers", dec: "10:37 PM · Proposed a change", dec2: "10:41 PM · Proposed a change" };
Object.keys(TURN_OF).forEach(k => { ART2[k].day = "Today"; ART2[k].turn = TURN_OF[k]; });
Object.assign(ART2.shp, { preview: "In transit · Acme Manufacturing" }); Object.assign(ART2.plan, { preview: "5 steps · 3 done" }); Object.assign(ART2.rate, { preview: "$1,486.00 · Acme contract" });
Object.assign(ART2.email, { preview: "2 recipients" }); Object.assign(ART2.diff, { preview: "6 changed · 3 added" }); Object.assign(ART2.doc, { preview: "Write-up · 3 loads" });
Object.assign(ART2.view, { preview: "3 filters · 3 results" }); Object.assign(ART2.dec, { preview: "11 changes" }); Object.assign(ART2.dec2, { preview: "15 changes" });
ART2.queue.versions = [
  { v: 1, at: "10:32 PM", note: "First read · 15 rows" },
  { v: 2, at: "10:36 PM", note: "Re-read before checking billers" },
  { v: 3, at: "10:40 PM", note: "After you approved Assign biller", needs: "assigned" },
  { v: 4, at: "10:42 PM", note: "After posting", needs: "posted" },
];

// Older history: everything earlier turns in this conversation made.
(function seedHistory() {
  const CITIES = ["Chicago", "Columbus", "Des Moines", "Omaha", "Joliet", "Gary", "Detroit", "Madison", "Memphis", "St. Louis", "Indianapolis", "Minneapolis"];
  const CUST = ["Acme Manufacturing", "Bluewater Retail", "Granite Building", "Harbor Supply Co.", "Northline Foods", "Peak Distributing", "Lakeside Grocers"];
  const DRV = ["Marcus Hill", "Dana Ortiz", "Lena Park", "Tom Reyes", "Ana Cruz", "Will Grant"];
  const T = {
    table: [i => ["Loads delivering in " + CITIES[i % 12], "shipments.list", (8 + i % 30) + " rows"], i => ["Invoices over 60 days", "invoices.list", (4 + i % 9) + " rows"], i => ["Drivers near " + CITIES[(i + 3) % 12], "drivers.nearby", (3 + i % 6) + " rows"], i => ["Carriers with expiring insurance", "carriers.list", (2 + i % 5) + " rows"], i => ["Detention by lane", "stops.aggregate", (10 + i % 12) + " rows"]],
    record: [i => ["SEED-SHP-0" + (10 + i % 89), "shipments.get", "In transit · " + CUST[i % 7]], i => ["Customer · " + CUST[i % 7], "customers.get", "Net 30 · " + (3 + i % 9) + " open loads"], i => ["Driver · " + DRV[i % 6], "drivers.get", (4 + i % 7) + "h left today"]],
    rate: [i => ["Rate for SEED-SHP-0" + (10 + i % 89), "rating.explain", "$" + (900 + (i * 137) % 1800).toLocaleString() + " · contract"]],
    email: [i => ["Delay notice to " + CUST[i % 7].split(" ")[0], "messages.draft", "2 recipients"], i => ["Rate confirmation to carrier", "messages.draft", "1 recipient"]],
    plan: [i => ["Cover " + (2 + i % 3) + " open loads", "planner.draft", (3 + i % 3) + " steps"]],
    report: [i => ["Detention · week " + (30 + i % 9), "reports.run", (800 + (i * 53) % 900) + " rows"], i => ["On-time delivery by customer", "reports.run", (40 + i % 60) + " rows"]],
    diff: [i => ["AR aging · week " + (30 + i % 9) + " vs " + (29 + i % 9), "reports.compare", (3 + i % 8) + " changed"]],
    doc: [i => ["Weekly ops summary", "notes.write", "Write-up"], i => ["Lane notes · " + CITIES[i % 12], "notes.write", "Write-up"]],
    view: [i => ["Late loads into " + CITIES[i % 12], "views.compose", (2 + i % 4) + " filters"]],
    decision: [i => ["Assign driver · " + (1 + i % 4) + " loads", "shipments.assign", "Approved"], i => ["Post " + (3 + i % 12) + " invoices", "billing.post_invoices", "Approved"]],
  };
  const BASE = { table: "board", record: "shp", rate: "rate", email: "email", plan: "plan", report: "report", diff: "diff", doc: "doc", view: "view", decision: "dec" };
  const TURNS_OLD = ["Read the board", "Checked detention", "Explained a rate", "Drafted customer notes", "Planned coverage", "Compared AR", "Ran reports", "Checked billers", "Looked up a customer", "Proposed driver assignments"];
  const KINDS = ["table", "table", "table", "record", "record", "rate", "email", "plan", "report", "diff", "doc", "view", "decision", "table", "record"];
  const DAYS = ["Yesterday", "Wed, Sep 30", "Tue, Sep 29", "Mon, Sep 28", "Fri, Sep 25", "Thu, Sep 24", "Wed, Sep 23"];
  let n = 0, seq = 0;
  const HIST = [];
  DAYS.forEach((day, di) => {
    const turns = 6 - (di % 2);
    for (let t = 0; t < turns; t++) {
      const hr = 5 - Math.floor(t * 0.8), mn = (t * 17 + di * 7) % 60;
      const label = (hr > 0 ? hr : 11) + ":" + String(mn).padStart(2, "0") + " PM · " + TURNS_OLD[(t + di * 3) % TURNS_OLD.length];
      const count = 3 + ((t * 7 + di * 5) % 4);
      for (let k = 0; k < count && n < 201; k++) {
        const kind = KINDS[(seq++) % KINDS.length];
        const tpl = T[kind][(k + t + di) % T[kind].length];
        const [title, src, preview] = tpl(seq);
        const id = "s" + (n++);
        ART2[id] = { ...ART2[BASE[kind]], kind, title, src, preview, day, turn: label, at: label.split(" · ")[0], old: true, slug: id, versions: undefined };
        HIST.push(id);
      }
    }
  });
  window.ART_HISTORY = HIST;
})();

const BULK_COLS = [{ k: "id", l: "Item", t: "id" }, { k: "name", l: "Customer" }, { k: "reason", l: "Why", t: "warnText" }];
ART2.failpost = { kind: "table", title: "Didn't post · 47 invoices", at: "10:43 PM", src: "billing.post_invoices", verb: "couldn't post", cols: BULK_COLS, rows: () => window.FAILED_POST || [], slug: "didnt-post", day: "Today", turn: "10:43 PM · Posted invoices" };
ART2.wouldfail = { kind: "table", title: "Would be refused · 38 items", at: "10:37 PM", src: "billing.assign_biller", verb: "previewed", cols: BULK_COLS, rows: () => window.WOULD_FAIL || [], slug: "would-be-refused", day: "Today", turn: "10:37 PM · Previewed a change" };

ART2.extract = { kind: "extract", title: "Rate con · Acme PO 77812", at: "10:44 PM", src: "documents.extract", verb: "read", file: "rate-con-acme-77812.pdf", pages: 3, slug: "rate-con-77812", day: "Today", turn: "10:44 PM · Read a document", preview: "12 fields · 2 need a look" };

ART2.bqi = { kind: "bill", title: "BQ-24101 · Acme", at: "10:38 PM", src: "billing.queue.get", verb: "opened", slug: "bq-24101", day: "Today", turn: "10:38 PM · Opened a queue item", preview: "$3,150.00 · 2 need you" };

function artMeta(id) {
  const a = ART2[id];
  if (!a) return { title: id, count: "", kind: "table" };
  if (a.old || (a.preview && a.kind !== "table" && a.kind !== "report")) return { ...a, id, count: a.preview };
  const count = a.kind === "table" ? (a.total || a.rows({}).length) + " rows" : a.kind === "report" ? a.rowCount.toLocaleString() + " rows" : KIND[a.kind].label;
  return { ...a, id, count };
}

Object.assign(window, { KIND, ART2, artMeta });
