/* Trenova sidebar exploration v2 — shared data + helpers */
(function () {
  const stored = localStorage.getItem("tn-theme");
  document.documentElement.dataset.theme = stored || "dark";
})();

const TN = {};

TN.reducedMotion = window.matchMedia("(prefers-reduced-motion: reduce)").matches;

TN.toggleTheme = function () {
  const root = document.documentElement;
  const next = root.dataset.theme === "dark" ? "light" : "dark";
  root.dataset.theme = next;
  localStorage.setItem("tn-theme", next);
  document.querySelectorAll("[data-theme-icon]").forEach((el) => {
    el.innerHTML = TN.icon(next === "dark" ? "sun" : "moon");
  });
};

/* ---------- Icons (lucide, 24x24 stroke) ---------- */
const ICON_PATHS = {
  home: '<path d="m3 9 9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/><polyline points="9 22 9 12 15 12 15 22"/>',
  truck:
    '<path d="M14 18V6a2 2 0 0 0-2-2H4a2 2 0 0 0-2 2v11a1 1 0 0 0 1 1h2"/><path d="M15 18H9"/><path d="M19 18h2a1 1 0 0 0 1-1v-3.65a1 1 0 0 0-.22-.62l-3.48-4.35A1 1 0 0 0 17.52 8H14"/><circle cx="17" cy="18" r="2"/><circle cx="7" cy="18" r="2"/>',
  users:
    '<path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M22 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/>',
  container:
    '<path d="M22 7.7c0-.6-.4-1.2-.8-1.5l-6.3-3.9a1.72 1.72 0 0 0-1.7 0l-10.3 6c-.5.2-.9.8-.9 1.4v6.6c0 .5.4 1.2.8 1.5l6.3 3.9a1.72 1.72 0 0 0 1.7 0l10.3-6c.5-.3.9-1 .9-1.5Z"/><path d="M10 21.9V14L2.1 9.1"/><path d="m10 14 11.9-6.9"/><path d="M14 19.8v-8.1"/><path d="M18 17.5V9.4"/>',
  receipt:
    '<path d="M4 2v20l2-1 2 1 2-1 2 1 2-1 2 1 2-1 2 1V2l-2 1-2-1-2 1-2-1-2 1-2-1-2 1Z"/><path d="M14 8H8"/><path d="M16 12H8"/><path d="M13 16H8"/>',
  sliders:
    '<path d="M15 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7Z"/><path d="M14 2v4a2 2 0 0 0 2 2h4"/><path d="M8 12h8"/><path d="M10 11v2"/><path d="M8 17h8"/><path d="M14 16v2"/>',
  chart:
    '<path d="M3 3v18h18"/><path d="M18 17V9"/><path d="M13 17V5"/><path d="M8 17v-3"/>',
  calculator:
    '<rect width="16" height="20" x="4" y="2" rx="2"/><line x1="8" x2="16" y1="6" y2="6"/><line x1="16" x2="16" y1="14" y2="18"/><path d="M16 10h.01"/><path d="M12 10h.01"/><path d="M8 10h.01"/><path d="M12 14h.01"/><path d="M8 14h.01"/><path d="M12 18h.01"/><path d="M8 18h.01"/>',
  settings:
    '<path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/>',
  star: '<polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/>',
  search: '<circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/>',
  plus: '<path d="M5 12h14"/><path d="M12 5v14"/>',
  chevronDown: '<path d="m6 9 6 6 6-6"/>',
  chevronRight: '<path d="m9 18 6-6-6-6"/>',
  chevronsUpDown: '<path d="m7 15 5 5 5-5"/><path d="m7 9 5-5 5 5"/>',
  bell: '<path d="M6 8a6 6 0 0 1 12 0c0 7 3 9 3 9H3s3-2 3-9"/><path d="M10.3 21a1.94 1.94 0 0 0 3.4 0"/>',
  clock: '<circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/>',
  zap: '<polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/>',
  package:
    '<path d="m7.5 4.27 9 5.15"/><path d="M21 8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16Z"/><path d="M3.3 7 12 12l8.7-5"/><path d="M12 22V12"/>',
  inbox:
    '<polyline points="22 12 16 12 14 15 10 15 8 12 2 12"/><path d="M5.45 5.11 2 12v6a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-6l-3.45-6.89A2 2 0 0 0 16.76 4H7.24a2 2 0 0 0-1.79 1.11z"/>',
  sun: '<circle cx="12" cy="12" r="4"/><path d="M12 2v2"/><path d="M12 20v2"/><path d="m4.93 4.93 1.41 1.41"/><path d="m17.66 17.66 1.41 1.41"/><path d="M2 12h2"/><path d="M20 12h2"/><path d="m6.34 17.66-1.41 1.41"/><path d="m19.07 4.93-1.41 1.41"/>',
  moon: '<path d="M12 3a6 6 0 0 0 9 9 9 9 0 1 1-9-9Z"/>',
  arrowLeft: '<path d="m12 19-7-7 7-7"/><path d="M19 12H5"/>',
  panelLeft: '<rect width="18" height="18" x="3" y="3" rx="2"/><path d="M9 3v18"/>',
  check: '<path d="M20 6 9 17l-5-5"/>',
  mapPin:
    '<path d="M20 10c0 4.993-5.539 10.193-7.399 11.799a1 1 0 0 1-1.202 0C9.539 20.193 4 14.993 4 10a8 8 0 0 1 16 0"/><circle cx="12" cy="10" r="3"/>',
  activity: '<path d="M22 12h-2.48a2 2 0 0 0-1.93 1.46l-2.35 8.36a.25.25 0 0 1-.48 0L9.24 2.18a.25.25 0 0 0-.48 0l-2.35 8.36A2 2 0 0 1 4.49 12H2"/>',
};

TN.icon = function (name, cls) {
  return (
    '<svg class="tn-icon' +
    (cls ? " " + cls : "") +
    '" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">' +
    (ICON_PATHS[name] || "") +
    "</svg>"
  );
};

/* ---------- Navigation data (mirrors client/src/config/navigation.config.ts) ---------- */
TN.NAV = {
  groups: [
    { id: "operations", label: "Operations", modules: ["shipment", "dispatch", "fleet", "edi"] },
    { id: "financial", label: "Financial", modules: ["billing", "reports", "accounting"] },
    { id: "admin", label: "Administration", modules: ["admin"] },
  ],
  modules: {
    home: { label: "Home", short: "Home", icon: "home", desc: "Dashboard and overview", items: [] },
    shipment: {
      label: "Shipment Management",
      short: "Shipments",
      icon: "truck",
      desc: "Shipments and related configuration",
      quickAction: "New Shipment",
      items: [
        { label: "Shipments", count: 128 },
        { label: "Orders", count: 36 },
        { label: "Service Failures", count: 3, tone: "warning" },
        {
          group: "Configuration Files",
          items: ["Shipment Types", "Service Types", "Hazardous Materials", "Commodities"],
        },
      ],
    },
    dispatch: {
      label: "Dispatch Management",
      short: "Dispatch",
      icon: "users",
      desc: "Workers, drivers, and dispatch operations",
      quickAction: "New Worker",
      items: [
        { label: "Locations" },
        { label: "Workers" },
        { group: "Configuration Files", items: ["Location Categories", "Fleet Codes"] },
      ],
    },
    fleet: {
      label: "Equipment Management",
      short: "Equipment",
      icon: "container",
      desc: "Tractors, trailers, and equipment",
      quickAction: "New Tractor",
      items: [
        { label: "Tractors" },
        { label: "Trailers" },
        { group: "Configuration Files", items: ["Equipment Manufacturers", "Equipment Types"] },
      ],
    },
    billing: {
      label: "Billing Management",
      short: "Billing",
      icon: "receipt",
      desc: "Invoicing and financial management",
      quickAction: "New Invoice",
      items: [
        { label: "Billing Queue", count: 12, active: true },
        { label: "Invoices" },
        { label: "Pending Approvals", count: 4 },
        { label: "Reconciliation Exceptions", count: 2, tone: "destructive" },
        { label: "Batch Monitor" },
        {
          group: "Configuration Files",
          items: [
            "Accessorial Charges",
            "Charge Types",
            "Formula Templates",
            "Rate Tables",
            "Customers",
            "Document Types",
            "Packet Rules",
          ],
        },
      ],
    },
    edi: {
      label: "EDI",
      short: "EDI",
      icon: "sliders",
      desc: "Partner exchange and load tender workflow",
      items: [
        { label: "Overview", count: 5, tone: "warning" },
        { label: "Partners" },
        { label: "Communication Profiles" },
        { label: "Mapping Profiles" },
        { label: "Template Designer" },
        { label: "Inbound Transfers" },
        { label: "Outbound Transfers" },
        { label: "Messages" },
        { label: "Inbound Files" },
        { label: "Test Cases" },
      ],
    },
    reports: {
      label: "Reports",
      short: "Reports",
      icon: "chart",
      desc: "Analytics and reporting",
      items: [{ label: "Reports Dashboard" }],
    },
    accounting: {
      label: "Accounting Management",
      short: "Accounting",
      icon: "calculator",
      desc: "Journals, ledgers, and reconciliation",
      quickAction: "New Journal",
      items: [
        { label: "Accounting Dashboard" },
        { label: "Manual Journals" },
        { label: "Journal Reversals" },
        { group: "Reports", items: ["Trial Balance", "Income Statement", "Balance Sheet"] },
        { group: "Accounts Receivable", items: ["AR Aging", "Customer Ledger", "Open Items"] },
        {
          group: "Bank Reconciliation",
          items: ["Bank Receipts", "Work Queue", "Summary", "Import Batches"],
        },
        { group: "Configuration Files", items: ["Account Types", "Fiscal Years"] },
      ],
    },
    admin: {
      label: "Organization Settings",
      short: "Settings",
      icon: "settings",
      desc: "System administration",
      items: [],
    },
  },
  favorites: [
    { label: "Billing Queue", module: "Billing" },
    { label: "Shipments", module: "Shipments" },
    { label: "Workers", module: "Dispatch" },
    { label: "AR Aging", module: "Accounting" },
  ],
  attention: [
    { label: "Billing Queue", module: "Billing", count: 12 },
    { label: "Pending Approvals", module: "Billing", count: 4 },
    { label: "Reconciliation Exceptions", module: "Billing", count: 2, tone: "destructive" },
    { label: "Service Failures", module: "Shipments", count: 3, tone: "warning" },
    { label: "EDI Needs Attention", module: "EDI", count: 5, tone: "warning" },
  ],
};

/* ---------- Presence + live activity ---------- */
TN.PEOPLE = [
  { initials: "MC", name: "Maria Chen" },
  { initials: "DJ", name: "Devon James" },
  { initials: "KP", name: "Kira Patel" },
  { initials: "TW", name: "Tom Wells" },
];

TN.ACTIVITY = [
  { p: 0, text: "approved <strong>INV-2026-0847</strong>", t: "2m" },
  { p: 1, text: "assigned <strong>S-104829</strong> to R. Alvarez", t: "4m" },
  { p: 2, text: "cleared recon exception on <strong>INV-2026-0791</strong>", t: "7m" },
  { p: 3, text: "updated rate table <strong>Dry Van 2026</strong>", t: "12m" },
  { p: 1, text: "accepted load tender from <strong>Sysco Foods</strong>", t: "15m" },
  { p: 0, text: "transferred 6 shipments to <strong>Billing Queue</strong>", t: "19m" },
  { p: 2, text: "flagged <strong>S-104798</strong> — missing BOL", t: "24m" },
  { p: 3, text: "added worker <strong>J. Okafor</strong> to Fleet 12", t: "31m" },
];

const AV_GRADIENTS = [
  "linear-gradient(135deg, oklch(0.65 0.17 300), oklch(0.5 0.18 320))",
  "linear-gradient(135deg, oklch(0.68 0.15 180), oklch(0.52 0.14 200))",
  "linear-gradient(135deg, oklch(0.7 0.15 60), oklch(0.55 0.16 40))",
  "linear-gradient(135deg, oklch(0.62 0.2 263), oklch(0.46 0.19 285))",
];

TN.presenceRow = function (label) {
  return (
    '<div class="presence"><span class="stack">' +
    TN.PEOPLE.map(
      (p) => '<span class="p-av" title="' + p.name + '">' + p.initials + "</span>",
    ).join("") +
    '</span><span class="p-label"><span class="dot success"></span>' +
    (label || "4 dispatchers online") +
    "</span></div>"
  );
};

TN.feedRowHtml = function (entry, fresh) {
  const person = TN.PEOPLE[entry.p];
  return (
    '<div class="feed-row' + (fresh ? " fresh" : "") + '">' +
    '<span class="f-av" style="background:' + AV_GRADIENTS[entry.p] + '">' +
    person.initials + "</span>" +
    '<span class="f-body"><strong>' + person.name.split(" ")[0] + "</strong> " +
    entry.text + '</span><span class="f-time">' + entry.t + "</span></div>"
  );
};

TN.mountActivity = function (el, rows) {
  const max = rows || 3;
  let cursor = max;
  el.classList.add("feed");
  el.innerHTML = TN.ACTIVITY.slice(0, max).map((e) => TN.feedRowHtml(e)).join("");
  if (TN.reducedMotion) return;
  setInterval(() => {
    const entry = { ...TN.ACTIVITY[cursor % TN.ACTIVITY.length], t: "now" };
    cursor++;
    el.insertAdjacentHTML("afterbegin", TN.feedRowHtml(entry, true));
    while (el.children.length > max) el.lastElementChild.remove();
  }, 7000);
};

/* ---------- Choreography helpers ---------- */
TN.stagger = function (container, selector) {
  const els = selector ? container.querySelectorAll(selector) : container.children;
  let i = 0;
  for (const el of els) {
    el.style.setProperty("--i", i++);
    el.classList.add("stagger-in");
  }
};

TN.attachIndicator = function (scroller) {
  if (getComputedStyle(scroller).position === "static") scroller.style.position = "relative";
  const ind = document.createElement("div");
  ind.className = "active-ind";
  scroller.appendChild(ind);
  const move = () => {
    const active = scroller.querySelector(".ni.active");
    if (!active) {
      ind.style.opacity = "0";
      return;
    }
    let top = 0;
    let node = active;
    while (node && node !== scroller) {
      top += node.offsetTop;
      node = node.offsetParent;
    }
    ind.style.opacity = "1";
    ind.style.transform = "translateY(" + top + "px)";
    ind.style.height = active.offsetHeight + "px";
  };
  requestAnimationFrame(move);
  return move;
};

/* ---------- Shared chrome ---------- */
TN.headerRight = function () {
  return (
    '<div class="header-right">' +
    '<button type="button" class="search-pill">' +
    TN.icon("search") +
    "<span>Search or jump to…</span><kbd>⌘K</kbd></button>" +
    '<button type="button" class="icon-btn" title="Notifications">' +
    TN.icon("bell") +
    '<span class="notif-dot"></span></button>' +
    '<button type="button" class="avatar" title="Eric Moss">EM</button>' +
    "</div>"
  );
};

TN.header = function (crumbs) {
  const parts = crumbs
    .map(
      (c, i) =>
        '<span class="' + (i === crumbs.length - 1 ? "current" : "") + '">' + c + "</span>",
    )
    .join('<span class="sep">/</span>');
  return '<header class="app-header"><div class="crumbs">' + parts + "</div>" + TN.headerRight() + "</header>";
};

const ROWS = [
  ["S-104829", "Archer Daniels Midland", "Chicago, IL → Dallas, TX", "ready", "Ready to Bill", "2,845.00"],
  ["S-104811", "Sysco Foods", "Atlanta, GA → Miami, FL", "ready", "Ready to Bill", "1,932.50"],
  ["S-104798", "Walmart Transportation", "Bentonville, AR → Tulsa, OK", "docs", "Missing Docs", "1,204.00"],
  ["S-104790", "Nestlé USA", "Fort Wayne, IN → Columbus, OH", "ready", "Ready to Bill", "987.25"],
  ["S-104782", "Home Depot Supply", "Savannah, GA → Charlotte, NC", "hold", "On Hold", "2,310.75"],
  ["S-104779", "Kroger Logistics", "Cincinnati, OH → Louisville, KY", "ready", "Ready to Bill", "645.00"],
  ["S-104771", "PepsiCo Beverages", "Plano, TX → Oklahoma City, OK", "ready", "Ready to Bill", "1,518.40"],
  ["S-104765", "Costco Wholesale", "Tracy, CA → Reno, NV", "docs", "Missing Docs", "1,876.90"],
];

TN.pageMock = function () {
  const rows = ROWS.map(
    (r, i) =>
      "<tr class='stagger-in' style='--i:" + (i + 3) + "'><td class='mono'>" +
      r[0] +
      "</td><td>" +
      r[1] +
      "</td><td class='mono'>" +
      r[2] +
      "</td><td><span class='status-pill " +
      r[3] +
      "'>" +
      r[4] +
      "</span></td><td class='num'>$" +
      r[5] +
      "</td></tr>",
  ).join("");
  return (
    '<div class="pm-head stagger-in" style="--i:0"><div><h1>Billing Queue</h1>' +
    "<p>Shipments that are ready to be transferred to an invoice.</p></div>" +
    '<div class="pm-actions"><button type="button" class="btn">Filter</button>' +
    '<button type="button" class="btn">Export</button>' +
    '<button type="button" class="btn primary">' +
    TN.icon("plus") +
    "Transfer to Invoice</button></div></div>" +
    '<div class="pm-table stagger-in" style="--i:2"><table><thead><tr>' +
    "<th>Pro Number</th><th>Customer</th><th>Lane</th><th>Status</th><th style='text-align:right'>Charges</th>" +
    "</tr></thead><tbody>" +
    rows +
    "</tbody></table></div>"
  );
};

TN.mountPill = function (name) {
  const el = document.createElement("div");
  el.className = "variant-pill";
  el.innerHTML =
    '<span class="vp-name"><strong>' +
    name +
    "</strong></span>" +
    '<a href="index.html">' +
    TN.icon("arrowLeft") +
    "All variants</a>" +
    '<button type="button" data-theme-icon onclick="TN.toggleTheme()" title="Toggle theme">' +
    TN.icon(document.documentElement.dataset.theme === "dark" ? "sun" : "moon") +
    "</button>";
  document.body.appendChild(el);
};

TN.badge = function (item) {
  if (item.count == null) return "";
  return '<span class="count' + (item.tone ? " " + item.tone : "") + '">' + item.count + "</span>";
};
