const MAC = /Mac|iPhone|iPad/.test(navigator.platform || navigator.userAgent);
const MOD = MAC ? "⌘" : "Ctrl";
const ALT = MAC ? "⌥" : "Alt";

const ENT = {
  shipment: { icon: "truck", label: "Shipment", plural: "Shipments", dot: "var(--teal)" },
  customer: { icon: "building", label: "Customer", plural: "Customers", dot: "var(--violet)" },
  worker: { icon: "user", label: "Worker", plural: "Workers", dot: "var(--info)" },
  document: { icon: "file", label: "Document", plural: "Documents", dot: "var(--warn)" },
};

const SHIP_STATUS = {
  new: ["New", "muted"], assigned: ["Assigned", "info"], transit: ["In transit", "info"], delayed: ["Delayed", "warn"],
  completed: ["Completed", "success"], ready: ["Ready to invoice", "success"], invoiced: ["Invoiced", "muted"],
};

const SHIPMENTS = [
  { id: "S-240193", cust: "Acme Foods", code: "ACME", status: "transit", o: "Dallas, TX", d: "Atlanta, GA", pu: "Oct 11 · 08:24", del: "Oct 12 · 16:00", eta: "Today 17:40", late: "1h 40m late", tone: "warn", note: "Receiver open until 18:00 — still inside the window", miles: 781, done: 0.72, driver: "Marcus Bell", tractor: "T-118", trailer: "TR-5521", total: 3245.5, bol: "88213", ev: [["Departed Meridian, MS weigh station", "13:05"], ["Held at weigh station · 52 min", "12:13"], ["Picked up · Acme DC 4, Dallas", "Oct 11 08:24"]] },
  { id: "S-240201", cust: "Halvorsen Steel", code: "HALV", status: "delayed", o: "Gary, IN", d: "Nashville, TN", pu: "Oct 12 · 06:10", del: "Oct 12 · 19:00", eta: "Today 20:15", late: "At risk · 1h 15m", tone: "warn", note: "Leon Park has 2h 10m of drive time for 3h of road", miles: 472, done: 0.38, driver: "Leon Park", tractor: "T-204", trailer: "FB-0912", total: 2180, bol: "88240", ev: [["Fuel stop · Seymour, IN", "11:42"], ["Picked up · Halvorsen mill 2, Gary", "06:10"]] },
  { id: "S-240177", cust: "Northwind Grocers", code: "NWG", status: "ready", o: "Memphis, TN", d: "Little Rock, AR", pu: "Oct 10 · 09:00", del: "Oct 10 · 14:30", eta: "Delivered 14:12", late: "Delivered on time", tone: "success", note: "POD received and matched · ready to bill", miles: 137, done: 1, driver: "Dana Ortiz", tractor: "T-131", trailer: "TR-5508", total: 1120, bol: "88177", ev: [["Proof of delivery matched by Nova", "Oct 10 15:02"], ["Delivered · NWG Little Rock", "Oct 10 14:12"]] },
  { id: "S-240188", cust: "Blue Ridge Paper", code: "BRP", status: "assigned", o: "Charlotte, NC", d: "Richmond, VA", pu: "Oct 13 · 07:00", del: "Oct 13 · 15:00", eta: "Tomorrow 14:20", late: "Pickup tomorrow 07:00", tone: "", note: "", miles: 289, done: 0, driver: "Priya Raman", tractor: "T-142", trailer: "TR-5530", total: 1460, bol: "88188", ev: [["Assigned to Priya Raman", "09:31"]] },
  { id: "S-240205", cust: "Sunbelt Beverage", code: "SBEV", status: "new", o: "Houston, TX", d: "San Antonio, TX", pu: "Oct 13 · 10:00", del: "Oct 13 · 16:00", eta: "—", late: "No driver yet", tone: "", note: "", miles: 197, done: 0, driver: "", tractor: "", trailer: "", total: 980, bol: "88205", ev: [["Created from rate confirmation", "10:48"]] },
  { id: "S-240199", cust: "Pioneer Ag", code: "PION", status: "transit", o: "Omaha, NE", d: "Kansas City, MO", pu: "Oct 12 · 05:30", del: "Oct 12 · 12:30", eta: "Today 12:05", late: "On time", tone: "success", note: "", miles: 186, done: 0.86, driver: "Tom Haskell", tractor: "T-109", trailer: "RF-2210", total: 1340, bol: "88199", ev: [["Passed St. Joseph, MO", "10:58"], ["Picked up · Pioneer Ag, Omaha", "05:30"]] },
  { id: "S-240162", cust: "Acme Foods", code: "ACME", status: "invoiced", o: "Fort Worth, TX", d: "Oklahoma City, OK", pu: "Oct 7 · 08:00", del: "Oct 7 · 13:00", eta: "Delivered 12:41", late: "Delivered on time", tone: "success", note: "Invoiced on INV-10442", miles: 204, done: 1, driver: "Marcus Bell", tractor: "T-118", trailer: "TR-5521", total: 1185, bol: "88162", ev: [["Invoice INV-10442 sent", "Oct 8 09:10"]] },
  { id: "S-240210", cust: "Northwind Grocers", code: "NWG", status: "new", o: "Memphis, TN", d: "Jackson, MS", pu: "Oct 14 · 08:00", del: "Oct 14 · 13:00", eta: "—", late: "No driver yet", tone: "", note: "", miles: 211, done: 0, driver: "", tractor: "", trailer: "", total: 1050, bol: "88210", ev: [["Created by Nova from email", "11:20"]] },
];

const CUSTOMERS = [
  { id: "ACME", name: "Acme Foods", status: "Active", city: "Dallas, TX", bal: 48210, aging: [31200, 11800, 4210, 1000], credit: 0.62, limit: "$80,000", open: 3, last: "S-240193 · today", contact: "Renee Walsh · AP", terms: "Net 30" },
  { id: "NWG", name: "Northwind Grocers", status: "Active", city: "Memphis, TN", bal: 12880, aging: [12880, 0, 0, 0], credit: 0.26, limit: "$50,000", open: 2, last: "S-240210 · today", contact: "Owen Price · Logistics", terms: "Net 15" },
  { id: "HALV", name: "Halvorsen Steel", status: "Active", city: "Gary, IN", bal: 61400, aging: [22100, 18300, 14000, 7000], credit: 0.88, limit: "$70,000", open: 1, last: "S-240201 · today", contact: "Ingrid Halvorsen · Owner", terms: "Net 45" },
  { id: "BRP", name: "Blue Ridge Paper", status: "Active", city: "Charlotte, NC", bal: 5420, aging: [5420, 0, 0, 0], credit: 0.14, limit: "$40,000", open: 1, last: "S-240188 · tomorrow", contact: "Sam Okafor · Shipping", terms: "Net 30" },
  { id: "SBEV", name: "Sunbelt Beverage", status: "On hold", city: "Houston, TX", bal: 9800, aging: [0, 2400, 3900, 3500], credit: 0.49, limit: "$20,000", open: 1, last: "S-240205 · tomorrow", contact: "Lucia Reyes · AP", terms: "Net 30" },
  { id: "PION", name: "Pioneer Ag", status: "Active", city: "Omaha, NE", bal: 7310, aging: [7310, 0, 0, 0], credit: 0.18, limit: "$40,000", open: 1, last: "S-240199 · today", contact: "Hank Duvall · Dispatch", terms: "Net 30" },
];

const WORKERS = [
  { id: "W-1042", name: "Marcus Bell", role: "Company driver", status: ["Driving", "info"], home: "Dallas, TX", load: "S-240193", hos: [["Drive", "4h 05m left", 0.37], ["Duty window", "6h 50m left", 0.49], ["Cycle", "38h left", 0.54]], cdl: "Mar 2028", med: "Nov 3, 2026", medWarn: true },
  { id: "W-1051", name: "Dana Ortiz", role: "Company driver", status: ["Off duty", "muted"], home: "Memphis, TN", load: "", hos: [["Drive", "11h left", 1], ["Duty window", "14h left", 1], ["Cycle", "52h left", 0.74]], cdl: "Aug 2029", med: "Jun 2027" },
  { id: "W-1063", name: "Leon Park", role: "Company driver", status: ["Driving", "warn"], home: "Gary, IN", load: "S-240201", hos: [["Drive", "2h 10m left", 0.2, 1], ["Duty window", "5h 05m left", 0.36], ["Cycle", "21h left", 0.3, 1]], cdl: "Jan 2027", med: "Oct 30, 2026", medWarn: true },
  { id: "W-1070", name: "Priya Raman", role: "Company driver", status: ["Off duty", "muted"], home: "Charlotte, NC", load: "S-240188", hos: [["Drive", "11h left", 1], ["Duty window", "14h left", 1], ["Cycle", "60h left", 0.86]], cdl: "May 2030", med: "Feb 2028" },
  { id: "W-1077", name: "Tom Haskell", role: "Owner-operator", status: ["Driving", "info"], home: "Omaha, NE", load: "S-240199", hos: [["Drive", "6h 20m left", 0.58], ["Duty window", "8h 35m left", 0.61], ["Cycle", "44h left", 0.63]], cdl: "Dec 2028", med: "Sep 2027" },
  { id: "W-1088", name: "Aisha Grant", role: "Owner-operator", status: ["Available", "success"], home: "Houston, TX", load: "", hos: [["Drive", "11h left", 1], ["Duty window", "14h left", 1], ["Cycle", "66h left", 0.94]], cdl: "Jul 2031", med: "Apr 2028" },
];

const DOCS = [
  { id: "D-9921", name: "BOL-88213.pdf", kind: "Bill of lading", on: "S-240193", by: "Capture · dispatch@acmefoods.com", when: "Oct 11 · 07:52", conf: 97, fields: [["Shipper", "Acme Foods DC 4"], ["Consignee", "Acme Foods Atlanta"], ["Pieces", "22 pallets"], ["Weight", "38,400 lb"]] },
  { id: "D-9934", name: "POD-S-240177.jpg", kind: "Proof of delivery", on: "S-240177", by: "Driver app · Dana Ortiz", when: "Oct 10 · 14:20", conf: 91, fields: [["Signed by", "J. Moreno"], ["Delivered", "Oct 10 · 14:12"], ["Exceptions", "None noted"]] },
  { id: "D-9940", name: "RateCon-Halvorsen-0412.pdf", kind: "Rate confirmation", on: "S-240201", by: "Triage · ingrid@halvorsensteel.com", when: "Oct 11 · 16:05", conf: 95, fields: [["Linehaul", "$1,940.00"], ["Fuel", "$240.00"], ["Pickup", "Oct 12 · 06:00"]] },
  { id: "D-9902", name: "INV-10442.pdf", kind: "Invoice", on: "S-240162", by: "Billing · auto-generated", when: "Oct 8 · 09:10", conf: 100, fields: [["Amount", "$1,185.00"], ["Due", "Nov 7, 2026"], ["Status", "Sent"]] },
  { id: "D-9877", name: "CDL-Dana-Ortiz.pdf", kind: "Commercial license", on: "W-1051", by: "HR · Kim Lee", when: "Aug 2 · 11:30", conf: 98, fields: [["Class", "A · Hazmat, Tanker"], ["Expires", "Aug 2029"], ["State", "TN"]] },
];

const money = (n) => "$" + n.toLocaleString("en-US", { minimumFractionDigits: n % 1 ? 2 : 0, maximumFractionDigits: 2 });

const RECORDS = [
  ...SHIPMENTS.map((s) => ({ kind: "record", type: "shipment", id: s.id, key: "r:shipment:" + s.id, title: s.id, d: s })),
  ...CUSTOMERS.map((c) => ({ kind: "record", type: "customer", id: c.id, key: "r:customer:" + c.id, title: c.name, d: c })),
  ...WORKERS.map((w) => ({ kind: "record", type: "worker", id: w.id, key: "r:worker:" + w.id, title: w.name, d: w })),
  ...DOCS.map((x) => ({ kind: "record", type: "document", id: x.id, key: "r:document:" + x.id, title: x.name, d: x })),
];
const RECORD_BY_KEY = Object.fromEntries(RECORDS.map((r) => [r.key, r]));
const recordById = (id) => RECORDS.find((r) => r.id === id);

const P = (id, title, trail, href, icon, kw, glance, extra) => ({ kind: "page", key: "p:" + id, id, title, trail, href, icon, kw, glance, ...extra });
const PAGES = [
  P("shipments", "Shipments", ["Shipment management"], "/shipments", "truck", ["loads", "orders", "dispatch", "board"], [["In transit", "42"], ["Delayed", "3", "warn"], ["Unassigned", "7"]], { related: ["new-shipment"], types: "shipment", visited: "8 min ago" }),
  P("service-failures", "Service failures", ["Shipment management"], "/shipments/service-failures", "alert", ["late", "missed", "exceptions", "otd"], [["This week", "5", "warn"], ["Open", "2"]], { visited: "Yesterday" }),
  P("customers", "Customers", ["Billing", "Configuration files"], "/billing/configurations/customers", "building", ["accounts", "shippers", "clients"], [["Active", "128"], ["On hold", "3", "warn"]], { related: ["new-customer"], types: "customer", visited: "25 min ago" }),
  P("billing-queue", "Billing queue", ["Billing"], "/billing/queue", "dollar", ["invoice", "ready to bill", "bill"], [["Ready", "6", "success"], ["On hold", "2", "warn"], ["Value", "$18.4k"]], { visited: "2 h ago" }),
  P("invoices", "Invoices", ["Billing"], "/billing/invoices", "file", ["bills", "statements", "receivables"], [["Sent this week", "31"], ["Overdue", "4", "warn"]]),
  P("formula-templates", "Formula templates", ["Billing", "Configuration files"], "/billing/configurations/formula-templates", "formula", ["rating", "rates", "pricing", "formula studio"], [["Published", "14"], ["Drafts", "2"]], { visited: "3 h ago" }),
  P("customer-ledger", "Customer ledger", ["Accounting", "Accounts receivable"], "/accounting/ar/customer-ledger", "book", ["ar", "aging", "balance", "payments"], [["Open AR", "$145k"], ["Over 60 days", "$29k", "warn"]]),
  P("workers", "Workers", ["Human resource management"], "/dispatch/configurations/workers", "users", ["drivers", "employees", "staff"], [["Active", "64"], ["Available now", "9", "success"]], { related: ["new-worker"], types: "worker", visited: "1 h ago" }),
  P("hos", "Hours of service", ["Human resource management"], "/dispatch/hos", "clock", ["hos", "eld", "logs", "drive time"], [["Driving", "21"], ["Under 2h left", "2", "warn"]]),
  P("training", "Training courses", ["Human resource management", "Configuration files"], "/hr/configurations/training-courses", "book", ["courses", "onboarding", "certification"], [["Courses", "18"], ["Overdue", "5", "warn"]]),
  P("tractors", "Tractors", ["Equipment management"], "/equipment/tractors", "truck", ["power units", "trucks", "fleet"], [["In service", "58"], ["In shop", "4"]]),
  P("trailers", "Trailers", ["Equipment management"], "/equipment/trailers", "trailer", ["vans", "reefers", "flatbeds", "fleet"], [["In service", "112"], ["Inspection due", "6", "warn"]]),
  P("desk", "Desk", [], "/desk", "desk", ["nova", "assistant", "ai", "chat", "conversations"], [["Open cases", "12"], ["Decisions waiting", "3", "warn"]], { visited: "Today" }),
  P("triage", "Triage", [], "/triage", "inbox", ["inbox", "email", "messages", "mail"], [["Unread", "23"], ["Needs a link", "4", "warn"]], { visited: "12 min ago" }),
  P("analytics", "Operational insights", ["Analytics"], "/analytics/insights", "chart", ["reports", "kpi", "dashboard", "findings"], [["New findings", "4"]]),
  P("ai-control", "AI control", ["Administration", "AI & Automation"], "/admin/ai-control", "spark", ["agents", "providers", "models", "ai", "automation"], [["Agents running", "4"], ["Provider degraded", "1", "warn"]]),
  P("doc-intel", "Document intelligence", ["Administration", "AI & Automation"], "/admin/document-intelligence", "eye", ["ocr", "extraction", "capture", "ai", "scanning"], [["Extracted today", "86"], ["Needs review", "3", "warn"]]),
  P("email-logs", "Email logs", ["Administration", "Email & SMS"], "/admin/email-logs", "mail", ["smtp", "delivery", "sent mail"], [["Sent today", "412"], ["Bounced", "2", "warn"]]),
  P("sf-reasons", "Service failure reasons", ["Administration", "Organization"], "/admin/service-failure-reasons", "alert", ["codes", "reasons", "late codes"], [["Reasons", "11"]]),
  P("users", "Users & roles", ["Administration", "Organization"], "/admin/users", "shield", ["permissions", "access", "team", "invite"], [["Users", "37"], ["Pending invites", "2"]]),
  P("integrations", "Integrations", ["Administration"], "/admin/integrations", "plug", ["eld", "samsara", "quickbooks", "connect", "api"], [["Connected", "6"], ["Needs attention", "1", "warn"]]),
];
const PAGE_BY_ID = Object.fromEntries(PAGES.map((p) => [p.id, p]));

const C = (id, label, desc, group, icon, kw, extra) => ({ kind: "command", key: "c:" + id, id, label, desc, group, icon, kw, ...extra });
const CMD_GROUPS = { create: "Create", nova: "Nova", navigation: "Go to", notifications: "Notifications", appearance: "Appearance", account: "Account" };
function buildCommands(theme) {
  return [
    C("new-shipment", "New shipment", "Start a load from scratch, or drop a rate confirmation", "create", "plus", ["create", "add", "load", "order"], { keys: [MOD, "⇧", "N"], to: "shipments" }),
    C("new-customer", "New customer", "Add a shipper, bill-to or consignee", "create", "plus", ["create", "add", "account"], { to: "customers" }),
    C("new-worker", "New worker", "Onboard a driver or employee", "create", "plus", ["create", "add", "driver", "hire"], { to: "workers" }),
    C("upload", "Upload a document", "Nova reads it and files it to the right record", "create", "upload", ["scan", "capture", "bol", "pod", "attach"], {}),
    C("nova-new", "New conversation", "Start a fresh thread with Nova in the Desk", "nova", "desk", ["assistant", "chat", "ai", "thread"], { keys: [MOD, "J"], to: "desk" }),
    C("insights", "Operational insights", "Findings computed from your own records", "nova", "chart", ["analytics", "reports", "ai"], { to: "analytics" }),
    C("copy-page", "Copy link to this page", "Share exactly what you're looking at, filters included", "navigation", "link", ["share", "url"], {}),
    C("sidebar", "Toggle sidebar", "Show or hide the navigation sidebar", "navigation", "layout", ["collapse", "expand", "hide"], { keys: [MOD, "B"] }),
    C("shortcuts", "Keyboard shortcuts", "Every shortcut that works across Trenova", "navigation", "keyboard", ["hotkeys", "keys", "help"], { keys: [MOD, "/"] }),
    C("notifications", "Open notifications", "4 unread notifications", "notifications", "bell", ["alerts", "inbox", "unread"], {}),
    C("read-all", "Mark all notifications as read", "Clear the unread count without opening each one", "notifications", "checks", ["clear", "dismiss"], {}),
    C("theme-light", "Use light theme", "Change how Trenova looks on this device", "appearance", "sun", ["theme", "appearance", "mode", "light"], { theme: "light", checked: theme === "light" }),
    C("theme-dark", "Use dark theme", "Change how Trenova looks on this device", "appearance", "moon", ["theme", "appearance", "mode", "dark"], { theme: "dark", checked: theme === "dark" }),
    C("theme-system", "Match system theme", "Follow your operating system", "appearance", "monitor", ["theme", "appearance", "auto"], { theme: "system", checked: theme === "system" }),
    C("profile", "Your profile", "Your details, sessions and security", "account", "user2", ["account", "me"], {}),
    C("settings", "Settings", "Preferences, time zone and language", "account", "gear", ["preferences", "timezone", "language"], { keys: [MOD, "⇧", "S"] }),
    C("logout", "Log out", "End this session on this device", "account", "logout", ["sign out", "exit"], { danger: true }),
  ];
}

const ATTENTION = [
  { kind: "attention", key: "a:delayed", label: "Shipments running late", module: "Shipment management", count: 2, tone: "warn", page: "shipments", sample: ["S-240193", "S-240201"], why: "ETAs past the delivery window, from the last ELD ping." },
  { kind: "attention", key: "a:unassigned", label: "Unassigned for tomorrow", module: "Dispatch", count: 2, tone: "info", page: "shipments", sample: ["S-240205", "S-240210"], why: "Pickups in the next 24 hours with no driver." },
  { kind: "attention", key: "a:ready", label: "Ready to invoice", module: "Billing", count: 6, tone: "success", page: "billing-queue", sample: ["S-240177"], why: "Delivered, POD matched, charges complete." },
  { kind: "attention", key: "a:med", label: "Medical cards expiring", module: "Compliance", count: 2, tone: "warn", page: "workers", sample: ["W-1063", "W-1042"], why: "Expiring in the next 30 days." },
];

const INITIAL_RECENTS = [["r:shipment:S-240193", "8m"], ["r:customer:ACME", "25m"], ["r:worker:W-1042", "1h"], ["r:document:D-9921", "2h"], ["r:shipment:S-240201", "3h"]];
const INITIAL_PINNED = ["shipments", "billing-queue", "triage"];

const SCOPES = [
  { id: "all", label: "All" },
  { id: "shipment", label: "Shipments", m: "shipment" },
  { id: "customer", label: "Customers", m: "customer" },
  { id: "worker", label: "Workers", m: "worker" },
  { id: "document", label: "Documents", m: "document" },
  { id: "page", label: "Pages", m: "page" },
  { id: "command", label: "Commands", m: "command" },
];

const SUBJECT_QUESTIONS = {
  shipment: ["Why is it running late?", "Draft an update for the customer", "What's left before we can bill it?"],
  customer: ["What's overdue and why?", "How has their volume changed this quarter?", "Draft a payment reminder"],
  worker: ["Can they take another load today?", "What's expiring soon?", "Summarize this week's hours"],
  document: ["What did Nova pull from this?", "Does it match the shipment?", "Who sent it?"],
};

Object.assign(window, { MAC, MOD, ALT, ENT, SHIP_STATUS, SHIPMENTS, CUSTOMERS, WORKERS, DOCS, RECORDS, RECORD_BY_KEY, recordById, PAGES, PAGE_BY_ID, buildCommands, CMD_GROUPS, ATTENTION, INITIAL_RECENTS, INITIAL_PINNED, SCOPES, SUBJECT_QUESTIONS, money });
