const ME = "System Administrator";
const TEAM = [[ME, "You"], ["Dana Whitfield", "Billing"], ["Marcus Lee", "Dispatch"], ["Priya Shah", "Customer service"]];
const isNeed = m => m.status === "InReview" || m.status === "Quarantined";
const LANES = [
  ["waiting", "Waiting on you", m => isNeed(m) && !m.snooze && !m.await, "inbox"],
  ["them", "Waiting on them", m => isNeed(m) && !!m.await, "send"],
  ["mine", "Assigned to me", m => isNeed(m) && m.owner === ME, "user"],
  ["snoozed", "Snoozed", m => isNeed(m) && !!m.snooze, "clock"],
  ["handled", "Handled", m => m.status === "Actioned", "check"],
  ["ignored", "Ignored", m => m.status === "Ignored", "archive"],
  ["all", "Everything", () => true, "layers"],
];
const SNOOZES = [["Later today", "17:00", 200], ["Tomorrow morning", "Thu 08:00", 1100], ["Monday", "Mon 08:00", 7000]];
const RULES0 = [
  { id: "r1", kind: "ProofOfDelivery", from: "drivers.trenova.example", mb: "mb1", act: "File on the matched shipment", n: 142, on: true, by: "Dana Whitfield" },
  { id: "r2", kind: "Invoice", from: "any sender", mb: "mb2", act: "File and queue for carrier pay", n: 61, on: true, by: "Dana Whitfield" },
  { id: "r3", kind: "Other", from: "newsletters", mb: "mb1", act: "Ignore", n: 210, on: true, by: ME },
  { id: "r4", kind: "StatusRequest", from: "any customer", mb: "mb1", act: "Draft a reply for review", n: 0, on: false, by: ME },
];
const RULE_FOR = { attach: "File on the matched shipment", draft: "Create the load as a draft", ask: "Draft a reply for review", open: "Open the shipment", link: "Link by sender" };

const DOCREAD = {
  d1: { type: "Rate confirmation", conf: 0.96, pages: 2, fields: [["Load", "S-104251"], ["Rate", "$1,240.00 all-in"], ["Pickup", "Oct 7, 14:00 · Columbus OH"], ["Delivery", "Oct 8, 08:00 · Louisville KY"], ["Equipment", "53' dry van"], ["Signed", "Yes · Oct 7, 12:41"]] },
  d2: { type: "Proof of delivery", conf: 0.93, pages: 1, fields: [["BOL", "77120"], ["Received by", "R. Patel"], ["Delivered", "Oct 6, 15:12"], ["Signature", "Present"], ["Exceptions", "2 pallets short", "w"]] },
  d3: { type: "Tender", conf: 0.88, pages: 3, fields: [["Tender", "4471"], ["Loads", "3"], ["Week of", "Oct 12"], ["Equipment", "Dry van"], ["Respond by", "Thu Oct 8"]] },
  d9: { type: "Invoice", conf: 0.98, pages: 1, fields: [["Invoice", "INV-20931"], ["Linehaul", "$1,420.00"], ["Detention 2.5h", "$185.00"], ["Total", "$1,605.00"]] },
  d4: { type: "Invoice", conf: 0.97, pages: 1, fields: [["Invoice", "KF-5521"], ["Carrier", "Kestrel Freight"], ["Amount", "$2,180.00"], ["Terms", "Net 30"]] },
};
const RECX = {
  s1: { p: 1, drv: "Luis Ortega", eq: "Dry van 53'", cust: "Northwind Foods", eta: "Delivered Oct 6, 15:12", docs: ["BOL", "Rate con"] },
  s2: { p: 1, drv: "Ana Ruiz", eq: "Reefer 53'", cust: "Harborline Grocers", eta: "Delivered Oct 3, 13:22", docs: ["BOL", "POD", "Invoice"] },
  s3: { p: 0.35, drv: "Tom Becker", eq: "Dry van 53'", cust: "Blue Ridge Brokerage", eta: "ETA 16:40 today", docs: ["BOL"] },
  s4: { p: 0.62, drv: "Kim Tran", eq: "Dry van 53'", cust: "Crestline Retail", eta: "ETA 16:10 · 2h late", docs: ["BOL", "Rate con"], late: true },
};
const STOPS = { appt: "08:00 – 10:00", rows: [["Appointment window", "Oct 3, 08:00 – 10:00", ""], ["Arrived (ELD)", "Oct 3, 10:40", "d"], ["Checked in at gate", "Oct 3, 10:52", ""], ["Unloaded and out", "Oct 3, 13:22", ""], ["Billed detention", "2.5 h · $185.00", ""]], read: "The truck arrived 40 minutes after the window. Harborline's contract only pays detention on on-time arrivals, so the charge is unlikely to stand." };

const SEED = { m2: { owner: "Dana Whitfield" }, m6: { owner: ME } };

/* what the email has to work with, as artifacts: what came in, what's linked, what Nova made */
function artsOf(m) {
  const out = [];
  if (m.made && m.made.reply) out.push({ id: "reply", kind: "reply", title: m.made.reply.sent ? "Reply sent" : "Reply draft", sub: m.made.reply.desk ? "Drafted by Nova" : "Written by you" });
  if (m.made && m.made.load) out.push({ id: "load", kind: "load", title: m.made.load.pro ? m.made.load.pro + " created" : "Load draft", sub: m.made.load.src || "Drafted from the email" });
  if (m.id === "m2") out.push({ id: "stops", kind: "stops", title: "Stop times · SEED-PAY-002", sub: "From ELD and gate log" });
  m.atts.filter(a => !a.fail).forEach(a => out.push({ id: "doc:" + a.n, kind: "doc", title: a.n, sub: (DOCREAD[a.doc] || {}).type || "Attachment", a }));
  if (m.shp) out.push({ id: "rec", kind: "record", title: SHP[m.shp].pro, sub: SHP[m.shp].o.split(",")[0] + " → " + SHP[m.shp].d.split(",")[0] });
  out.push({ id: "about", kind: "about", title: "About this email", sub: "Reading, sender, history" });
  return out;
}
const ART_IC = { reply: "send", load: "box", stops: "clock", doc: "file", record: "truck", about: "mark" };

const SLA = { Tender: 120, RateConfirmation: 120, StatusRequest: 60, DetentionDispute: 1440, ProofOfDelivery: 480, Invoice: 1440, Other: 2880 };
const dueOf = m => SLA[m.kind] - m.min;
const short = n => n < 60 ? n + "m" : n < 1440 ? Math.floor(n / 60) + "h" : Math.floor(n / 1440) + "d";
const dueTxt = d => d < 0 ? short(-d) + " over" : short(d) + " left";
const VIEWS0 = [{ id: "v1", name: "Acme Manufacturing", q: "acme" }, { id: "v2", name: "Has a PRO", q: "S-10" }];
const CONFLICTS = {
  m5: { t: "A rate con for S-104251 is already on file", d: "Filed Oct 6 at $1,180. This one says $1,240 — $60 more.", acts: [["Replace the old one", "Replaced the rate con on S-104251"], ["Keep both", "Kept both rate cons"]] },
  m8: { t: "Load 2 may already exist", d: "S-104190 is Toledo → Lansing on Oct 13 for Acme.", acts: [["Skip load 2", "Skipped load 2 — it's S-104190"], ["It's a new load", "Load 2 is new"]] },
};
const PAGES = { "RateCon_S-104251_signed.pdf": [["Rate confirmation", "Rate confirmation"], ["Signed BOL", "Bill of lading"]], "tender_4471.pdf": [["Cover letter", "Other"], ["Load sheet", "Tender"], ["BOL template", "Bill of lading"]] };
const MB_HEALTH = { mb3: { t: "Claims hasn't received mail in 7 days", d: "Harborline's forward from claims@harborline.example stopped on Oct 1 — their server now gets a 550 back from our address.", addr: "claims@dev.trenova.app" } };
function profileOf(m, msgs) {
  const dom = m.from[1].split("@")[1], xs = msgs.filter(x => x.from[1].split("@")[1] === dom);
  const kinds = {}; xs.forEach(x => kinds[x.kind] = (kinds[x.kind] || 0) + 1);
  const h = [...dom].reduce((s, c) => s + c.charCodeAt(0), 0);
  return { dom, n: xs.length + (h % 30) + 4, kinds: Object.entries(kinds).sort((a, b) => b[1] - a[1]), resp: 12 + (h % 70), role: m.cust ? (m.cust === "Blue Ridge Brokerage" ? "Broker" : "Customer") : /drivers\./.test(dom) ? "Driver" : /edi|carrier/.test(dom) ? "Carrier" : "Unknown sender", since: m.cust ? "Since " + (2019 + (h % 5)) : null, recent: xs.filter(x => x.id !== m.id).slice(0, 3) };
}
function previewRule(r) { const h = [...(r.kind + r.from + r.mb + r.act)].reduce((s, c) => s + c.charCodeAt(0), 0); return { n: 8 + (h % 46), wrong: h % 4 === 0 ? 0 : h % 3 }; }

const VIEWING = { m2: ["Dana Whitfield", "viewing"], m7: ["Priya Shah", "writing a reply"] };
const TEMPLATES0 = [
  { id: "t1", n: "ETA update", b: "Hi {first} — {pro} is {status}. Current ETA is {eta}. We'll tell you if that moves.\n\nThanks,\nTrenova" },
  { id: "t2", n: "Detention waiver", b: "Hi {first} — you're right. We've removed the detention on {pro}; a corrected invoice follows today.\n\nThanks,\nTrenova Billing" },
  { id: "t3", n: "Ask for the POD", b: "Hi {first} — could you send the signed POD for {pro}? A clear photo of each page works.\n\nThanks,\nTrenova" },
  { id: "t4", n: "Accept the tender", b: "Hi {first} — we'll take it. Confirming {lane}. Rate confirmation to follow.\n\nThanks,\nTrenova Dispatch" },
];
function fillT(t, m) { const s = m.shp && SHP[m.shp], x = m.shp && RECX[m.shp]; const f = { first: (m.from[0] || "there").split(" ")[0], pro: s ? s.pro : "your load", status: x ? (x.p >= 1 ? "delivered" : x.late ? "running about two hours behind" : "on schedule") : "moving", eta: x ? x.eta.replace("ETA ", "") : "to follow", lane: s ? s.o.split(",")[0] + " → " + s.d.split(",")[0] : "the load" }; return t.replace(/\{(\w+)\}/g, (a, k) => f[k] || a); }
const EVENT_SNOOZE = m => { const s = m.shp && SHP[m.shp], out = []; if (s && RECX[m.shp].p < 1) out.push(["Until " + s.pro + " delivers", "event"]); if (m.kind === "Tender") out.push(["Until the rate con arrives", "event"]); if (m.kind === "ProofOfDelivery" || m.kind === "DetentionDispute") out.push(["Until the invoice is paid", "event"]); return out; };
const LIVE_POOL = [
  { from: ["Kestrel Freight", "dispatch@kestrelfreight.example"], subject: "Truck available Columbus Thu AM", kind: "Other", conf: 0.71, mb: "mb1", body: ["We have a 53' dry van empty in Columbus Thursday morning if you have anything heading south."], note: "" },
  { from: ["Crestline Retail", "transport@crestline.example"], subject: "PO 88240 — can you pick up early?", kind: "StatusRequest", conf: 0.9, mb: "mb1", shp: "s4", cust: "Crestline Retail", reason: "PO 88240 is the next Crestline load on this lane.", body: ["Any chance the PO 88240 pickup can move to 07:00 tomorrow? Dock is free early."] },
  { from: ["Luis Ortega", "luis.ortega@drivers.trenova.example"], subject: "IMG_4411 again (smaller)", kind: "ProofOfDelivery", conf: 0.95, mb: "mb1", shp: "s1", reason: "Same driver and PRO as this morning's POD.", body: ["Resending the second page smaller."], atts: [{ n: "IMG_4411_small.jpg", size: "1.1 MB", kind: "ProofOfDelivery", doc: "d2" }] },
];

Object.assign(window, { VIEWING, TEMPLATES0, fillT, EVENT_SNOOZE, LIVE_POOL });
Object.assign(window, { SLA, dueOf, dueTxt, VIEWS0, CONFLICTS, PAGES, MB_HEALTH, profileOf, previewRule });
Object.assign(window, { ME, TEAM, isNeed, LANES, SNOOZES, RULES0, RULE_FOR, DOCREAD, RECX, STOPS, SEED, artsOf, ART_IC });
