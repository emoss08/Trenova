const CUSTOMERS = ["Acme Manufacturing", "Peak Distributing", "Northline Foods", "Harbor Supply Co.", "Granite Building", "Bluewater Retail"];
const AMOUNTS = [3150, 2875.5, 4120, 1980, 3640, 2210, 5300, 2745, 3890, 1650, 4475, 2990, 3335, 2560, 3289.5];
const QUEUE = AMOUNTS.map((amt, i) => ({
  id: "BQ-" + (24101 + i),
  shp: "SEED-SHP-" + String(118 + i * 3).padStart(4, "0"),
  cust: CUSTOMERS[(i * 5) % 6],
  amt,
  status: i === 6 ? "Approved" : "ReadyForReview",
  biller: [1, 4, 6, 10].includes(i) ? "Jordan Pike" : null,
}));
const TOTAL = AMOUNTS.reduce((a, b) => a + b, 0);
const money = n => n.toLocaleString("en-US", { minimumFractionDigits: 2, maximumFractionDigits: 2 });

const ARTIFACTS = {
  queue: { id: "queue", kind: "Table", title: "Billing queue", sub: "15 rows · from billing.queue.list", slug: "billing-queue", filter: () => true },
  gaps: { id: "gaps", kind: "Table", title: "Missing a biller", sub: "11 rows · from billing.queue.inspect", slug: "missing-biller", filter: r => !QUEUE.find(q => q.id === r.id).biller },
};

// Segments: string | {r:n, t} reference to trace step n (1-based) | {b} bold
const TURNS = {
  t1: {
    dur: "9s", time: "10:30 PM",
    trace: [
      { pose: "lookup", live: "Searching shipments ready to invoice", done: "Searched ready-to-invoice shipments", k: "shipments.search", v: "status = ReadyToInvoice", r: "0 rows", t: "3.8s", ms: 900 },
      { pose: "lookup", live: "Checking completed shipments", done: "Checked completed, unbilled shipments", k: "shipments.search", v: "status = Completed · unbilled", r: "0 rows", t: "4.1s", ms: 900 },
    ],
    text: [["No shipments are eligible to transfer right now. I checked ", { r: 1, t: "ready-to-invoice shipments" }, " and ", { r: 2, t: "completed shipments" }, " that could be marked ready first — both came back empty, so nothing was moved."]],
  },
  t2: {
    dur: "7s", time: "10:32 PM", art: "queue",
    trace: [
      { pose: "lookup", live: "Reading the billing queue", done: "Read the billing queue", k: "billing.queue.list", v: "all open items", r: "15 rows", t: "2.1s", ms: 900, art: "queue" },
      { pose: "think", live: "Grouping by status", done: "Grouped by status", k: "group_by", v: "status", r: "2 groups", t: "0.4s", ms: 700 },
    ],
    text: [["There are ", { r: 1, t: "15 items in the billing queue" }, ": ", { b: "14 ready for review" }, " and ", { b: "1 approved" }, ", all queued yesterday. Here's the full list with bill-to customers and amounts: ", { a: "queue" }]],
  },
  t3: {
    dur: "6s", time: "10:36 PM", art: "gaps", decision: "d1",
    trace: [
      { pose: "lookup", live: "Re-reading the billing queue", done: "Re-read the billing queue", k: "billing.queue.list", v: "all open items", r: "15 rows", t: "1.4s", ms: 1400, art: "queue" },
      { pose: "lookup", live: "Checking a biller on each item", done: "Checked the biller on each item", k: "billing.queue.inspect", v: "biller is empty", r: "11 rows", t: "1.8s", ms: 1700, art: "gaps" },
      { pose: "think", live: "Working out what can post", done: "Worked out what can post", k: "policy.check", v: "post requires approved item", r: "blocked", t: "1.3s", ms: 1400 },
      { pose: "change", live: "Drafting a change for you to approve", done: "Drafted one change for approval", k: "billing.assign_biller", v: "11 items → Avery Lane", r: "draft", t: "1.1s", ms: 1300 },
    ],
    text: [
      ["Not all of them can post yet. ", { r: 2, t: "Eleven of the fifteen" }, " have no biller, and an item can't be approved without one: ", { a: "gaps" }],
      ["I've drafted ", { r: 4, t: "one change" }, " that assigns you as biller on all eleven. Once you approve it, they move into review and I'll queue the posting."],
    ],
  },
  t4: {
    dur: "3s", time: "10:40 PM", art: "queue", decision: "d2",
    trace: [
      { pose: "change", live: "Assigning you on 11 items", done: "Assigned Avery Lane on 11 items", k: "billing.assign_biller", v: "11 items", r: "11 updated", t: "1.6s", ms: 1500, art: "gaps" },
      { pose: "lookup", live: "Re-checking the queue", done: "Re-checked the queue", k: "billing.queue.list", v: "all open items", r: "15 ready", t: "0.9s", ms: 1000, art: "queue" },
    ],
    text: [["Done — you're the biller on ", { r: 1, t: "all eleven" }, ". Every item is ready now, so I've drafted the posting: ", { r: 2, t: "15 invoices" }, " totaling ", { b: "$" + money(TOTAL) }, ". The updated queue is here: ", { a: "queue" }, "."]],
  },
};

const DECISIONS = {
  d1: { id: "d1", title: "Assign biller", scope: "11 billing queue items", field: "Biller", from: "None", to: "Avery Lane", meta: ["Reversible", "11 records", "billing.assign_biller"], art: "gaps" },
  d2: { id: "d2", title: "Post invoices", scope: "15 invoice drafts", field: "Status", from: "Ready for review", to: "Posted", meta: ["$" + money(TOTAL), "6 customers", "Can't be undone"], art: "queue", hard: true },
};

const CONVOS = [
  { g: "Pinned", items: [{ t: "Check and see what shipments are eligible to be transferred", a: "Billing Specialist", w: "now", on: true }] },
  { g: "Today", items: [{ t: "How many invoices have we posted this week?", a: "Billing Specialist", w: "2h" }, { t: "New load: Chicago IL to Columbus OH", a: "Dispatch desk", k: "dispatch", w: "now", s: "work" }] },
  { g: "Yesterday", items: [{ t: "There are an assortment of invoices sitting in draft", a: "Billing Specialist", w: "12h", s: "wait" }, { t: "Which shipments are at risk for the storm?", a: "Dispatch desk", k: "dispatch", w: "16h" }] },
  { g: "Previous 7 days", items: [{ t: "Blocked invoices this week", a: "Billing exceptions", w: "2d", s: "error" }, { t: "Where is SEED-SHP-001?", a: "Dispatch desk", k: "dispatch", w: "3d", s: "new" }, { t: "Who has a medical card expiring this month?", a: "Compliance desk", k: "dispatch", w: "5d" }] },
];

Object.assign(window, { QUEUE, TOTAL, money, ARTIFACTS, TURNS, DECISIONS, CONVOS });
