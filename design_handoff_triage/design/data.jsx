const KINDS = [
  ["Tender", "Tender", 295],
  ["RateConfirmation", "Rate confirmation", 182],
  ["ProofOfDelivery", "Proof of delivery", 150],
  ["Invoice", "Invoice", 75],
  ["StatusRequest", "Status request", 235],
  ["DetentionDispute", "Detention dispute", 18],
  ["Other", "Other", null],
];
const KIND = Object.fromEntries(KINDS.map(([k, l, h]) => [k, { k, l, h }]));
const STATUS = { InReview: ["Waiting on you", "w"], Quarantined: ["Held back", "d"], Actioned: ["Handled", "k"], Ignored: ["Ignored", "n"], Processing: ["Being read", "b"] };
const MAILBOXES = [
  { id: "mb1", name: "Intake", addr: "intake@dev.trenova.app", bar: 0.75, ok: true },
  { id: "mb2", name: "Billing", addr: "ap@dev.trenova.app", bar: 0.85, ok: true },
  { id: "mb3", name: "Claims", addr: "claims@dev.trenova.app", bar: null, ok: false },
];
const SHIPMENTS = [
  { id: "s1", pro: "S-104233", o: "Joliet, IL", d: "Indianapolis, IN", bol: "BOL 77120", po: "PO 4471", st: "Delivered Oct 6" },
  { id: "s2", pro: "SEED-PAY-002", o: "Savannah, GA", d: "Atlanta, GA", bol: "BOL 30218", po: "PO 99-210", st: "Delivered Oct 3" },
  { id: "s3", pro: "S-104251", o: "Columbus, OH", d: "Louisville, KY", bol: "BOL 77188", po: "", st: "In transit · ETA 16:40" },
  { id: "s4", pro: "S-104198", o: "Dallas, TX", d: "Memphis, TN", bol: "BOL 76902", po: "PO 88213", st: "In transit · 2h late" },
];
const SHP = Object.fromEntries(SHIPMENTS.map(s => [s.id, s]));
const CUSTOMERS = ["Northwind Foods", "Harborline Grocers", "Crestline Retail", "Acme Manufacturing", "Blue Ridge Brokerage"];

/* body: paragraphs; {{text|what Nova read it as}} marks a phrase the reading used */
const MESSAGES = [
  { id: "m1", from: ["Northwind Foods Dispatch", "dispatch@northwindfoods.example"], to: ["intake@dev.trenova.app"], subject: "New load: Chicago IL to Columbus OH, pickup Thursday 08:00", min: 1102, status: "InReview", kind: "Tender", conf: 0.68, mb: "mb1", cust: "Northwind Foods", reason: "Sender domain northwindfoods.example belongs to Northwind Foods, a customer since 2023.", note: "Creating a load is not something Nova does on its own.",
    body: ["Hi team —", "We have a {{dry van|Equipment}} load ready {{Thursday morning|Pickup · Thu Oct 8, 08:00}}. {{42,000 lbs|Weight}}, no appointment needed at origin, delivery is a {{2-hour window|Delivery window}}. Rate confirmation to follow once you accept.", "Origin is our {{Chicago DC, 2400 S Halsted|Origin}} and it delivers to {{Columbus OH|Destination}}.", "Thanks,\nMaya — Northwind dispatch"], atts: [], run: "run_8812" },
  { id: "m2", th: "t-harb", from: ["Harborline Grocers Claims", "claims@harborline.example"], to: ["claims@dev.trenova.app"], subject: "Disputing detention on SEED-PAY-002", min: 1222, status: "InReview", kind: "DetentionDispute", conf: 0.91, mb: "mb3", shp: "s2", cust: "Harborline Grocers", reason: "The PRO number SEED-PAY-002 is in the subject line.",
    body: ["Hello,", "We show the driver arriving {{40 minutes after the appointment window|Late arrival claim}} on {{Oct 3|Stop date}}, so we are disputing the {{$185.00 detention|Disputed amount}} on invoice {{INV-20931|Invoice}}.", "Please send the gate log or ELD record if you believe otherwise.", "Regards,\nHarborline Claims"], quoted: "On Oct 4, Trenova Billing wrote:\n> Attached is invoice INV-20931 for SEED-PAY-002, including 2.5h detention at the consignee.", atts: [{ n: "INV-20931.pdf", size: "88 KB", kind: "Invoice", doc: "d9" }], run: "run_8790" },
  { id: "m3", from: ["Partner Carrier EDI", "edi@partnercarrier.example"], to: ["intake@dev.trenova.app"], subject: "Automated: load status file 20260919-0441", min: 1462, status: "InReview", kind: "StatusRequest", conf: 0.41, mb: "mb1", fail: "The body was cut off by the sending system, so Nova could not read which load it is about.", body: ["ST*214*0441~B10*PC0441**SCAC~LX*1~AT7*X3*NS***20260919*0412~MS1*COLUMBUS*OH~"], atts: [], run: "run_8781" },
  { id: "m4", from: ["", "noreply@unknownsender.example"], to: ["intake@dev.trenova.app"], subject: "", min: 5302, status: "Quarantined", kind: "Other", conf: 0.22, mb: "mb1", fail: "Held back: the sender is not on this mailbox's allow list and the message carried a link to a sign-in page.", body: ["Your shipping account requires verification. Sign in to keep receiving loads: hxxp://trenova-verify.example/login"], atts: [], run: "run_8702" },
  { id: "m5", from: ["Blue Ridge Brokerage", "loads@blueridgebrokerage.example"], to: ["intake@dev.trenova.app"], subject: "Rate con — S-104251 Columbus to Louisville", min: 54, status: "InReview", kind: "RateConfirmation", conf: 0.83, mb: "mb1", shp: "s3", cust: "Blue Ridge Brokerage", reason: "The PRO number S-104251 is in the subject, and the attached PDF names the same lane.", note: "Filing a document on a shipment waits for a person on this mailbox.",
    body: ["Please find the signed rate confirmation for {{S-104251|Shipment}} attached. {{$1,240 all-in|Rate}}, {{Columbus OH → Louisville KY|Lane}}.", "Sign and return within 2 hours to hold the truck."], atts: [{ n: "RateCon_S-104251_signed.pdf", size: "212 KB", kind: "RateConfirmation", doc: "d1" }], run: "run_8840" },
  { id: "m6", from: ["Luis Ortega", "luis.ortega@drivers.trenova.example"], to: ["pod@dev.trenova.app"], cc: ["dispatch@dev.trenova.app"], subject: "POD S-104233", min: 196, status: "InReview", kind: "ProofOfDelivery", conf: 0.94, mb: "mb1", shp: "s1", reason: "The driver is assigned to S-104233 and the subject names it.", note: "Filing a document on a shipment waits for a person on this mailbox.",
    body: ["Delivered, signed by receiving. 2 pallets short-shipped noted on the BOL.", "Sent from my phone"], atts: [{ n: "IMG_4410.jpg", size: "2.4 MB", kind: "ProofOfDelivery", doc: "d2" }, { n: "IMG_4411.HEIC", size: "31.8 MB", kind: "Unknown", fail: "Refused: larger than this mailbox's 25 MB limit per file." }], run: "run_8832" },
  { id: "m7", th: "t-crest", from: ["Crestline Retail", "transport@crestline.example"], to: ["intake@dev.trenova.app"], subject: "Where is PO 88213?", min: 22, status: "InReview", kind: "StatusRequest", conf: 0.88, mb: "mb1", shp: "s4", cust: "Crestline Retail", reason: "PO 88213 is on S-104198 for Crestline Retail.",
    body: ["Hi, can you give me an ETA on {{PO 88213|Purchase order}}? Our receiving closes at {{17:00|Receiving cutoff}} today."], atts: [], run: "run_8845" },
  { id: "m8", from: ["Acme Manufacturing", "logistics@acme.example"], to: ["intake@dev.trenova.app"], subject: "Tender 4471 — 3 loads week of Oct 12", min: 312, status: "InReview", kind: "Tender", conf: 0.72, mb: "mb1", cust: "Acme Manufacturing", reason: "Sender domain matches Acme Manufacturing.", note: "Creating a load is not something Nova does on its own.",
    body: ["Attached is tender 4471 for three dry van loads the week of Oct 12. Please confirm by end of day Thursday."], atts: [{ n: "tender_4471.pdf", size: "164 KB", kind: "Other", doc: "d3" }], run: "run_8829" },
  { id: "m9", from: ["Kestrel Freight AP", "ap@kestrelfreight.example"], to: ["ap@dev.trenova.app"], subject: "Invoice KF-5521 for S-104198", min: 410, status: "Actioned", kind: "Invoice", conf: 0.97, mb: "mb2", shp: "s4", reason: "Invoice references S-104198.", by: null, byMin: 409, done: "Filed KF-5521.pdf on S-104198 and queued it for carrier pay.",
    body: ["Invoice KF-5521 attached for S-104198. Net 30."], atts: [{ n: "KF-5521.pdf", size: "74 KB", kind: "Invoice", doc: "d4", filed: true }], run: "run_8820" },
  { id: "m10", th: "t-harb", from: ["Harborline Grocers", "ops@harborline.example"], to: ["intake@dev.trenova.app"], subject: "Re: appointment for Friday", min: 1640, status: "Actioned", kind: "StatusRequest", conf: 0.8, mb: "mb1", shp: "s2", cust: "Harborline Grocers", reason: "Thread continues one about SEED-PAY-002.", by: "Dana Whitfield", byMin: 1580, done: "Called receiving, moved the appointment to 10:00.",
    body: ["Can we move Friday's appointment to 10:00?"], atts: [], run: "run_8760" },
  { id: "m11", from: ["FreightWeekly", "news@freightweekly.example"], to: ["intake@dev.trenova.app"], subject: "This week in freight: spot rates soften", min: 2900, status: "Ignored", kind: "Other", conf: 0.9, mb: "mb1", by: "Dana Whitfield", byMin: 2800, done: "Newsletter.",
    body: ["Spot rates eased for a third week as capacity loosened across the Midwest…"], atts: [], run: "run_8711" },
  { id: "m12", th: "t-crest", from: ["Crestline Retail", "transport@crestline.example"], to: ["intake@dev.trenova.app"], subject: "PO 88213 — pickup confirmed?", min: 3020, status: "Actioned", kind: "StatusRequest", conf: 0.86, mb: "mb1", shp: "s4", cust: "Crestline Retail", reason: "PO 88213 is on S-104198.", by: "Priya Shah", byMin: 2990, done: "Confirmed pickup for Mon 09:00.", body: ["Just checking pickup for PO 88213 is still Monday morning?"], atts: [], run: "run_8700" },
];

const PAPER = new Set(["ProofOfDelivery", "RateConfirmation", "Invoice"]);
function nextStep(m) {
  if (m.status !== "InReview" && m.status !== "Quarantined") return { k: "none" };
  if (m.status === "Quarantined" || m.fail) return { k: "link" };
  const s = m.shp ? SHP[m.shp] : null, doc = (m.atts || []).find(a => !a.fail && a.doc);
  if (m.kind === "Tender") return doc ? { k: "draft", doc } : { k: "ask", why: "build" };
  if (PAPER.has(m.kind)) return !s ? { k: "link" } : doc ? { k: "attach", doc, s } : { k: "open", s };
  if (m.kind === "StatusRequest") return { k: "ask", why: "answer", s };
  if (m.kind === "DetentionDispute") return s ? { k: "open", s } : { k: "link" };
  return { k: "none" };
}
/* the step, as one line for a list row and as a sentence + button for the reader */
function stepCopy(st, m) {
  switch (st.k) {
    case "draft": return { short: "Review the drafted load", say: <>Nova drafted a load from <b>{st.doc.n}</b>. Check it before it goes on the board.</>, btn: "Review the draft", ic: "scan" };
    case "attach": return { short: "File " + st.doc.n.replace(/_signed|\.pdf|\.jpg/g, "") + " on " + st.s.pro, say: <>File <b>{st.doc.n}</b> on <b>{st.s.pro}</b>, where billing will look for it.</>, btn: "Attach to " + st.s.pro, ic: "clip" };
    case "open": return { short: "Answer it from " + st.s.pro, say: <>The answer is on <b>{st.s.pro}</b>{m.kind === "DetentionDispute" ? " — the stop times and gate log are there." : "."}</>, btn: "Open " + st.s.pro, ic: "ext" };
    case "ask": return st.why === "build"
      ? { short: "Have Nova build the load", say: <>Nothing readable came attached. Nova can build the load from the text it underlined below.</>, btn: "Have Nova build it", ic: "mark", desk: true }
      : { short: "Have Nova draft a reply", say: <>Nova can look up {st.s ? <b>{st.s.pro}</b> : "the load"} and draft the answer for you to send.</>, btn: "Draft the answer", ic: "mark", desk: true };
    case "link": return { short: m.status === "Quarantined" ? "Check it is safe, then link or ignore" : "Say which record it is about", say: m.status === "Quarantined" ? <>Nothing about this looks like freight. Ignore it unless you know the sender.</> : <>Nova could not tell what this is about. Say which shipment or customer it belongs to.</>, btn: "Link by hand", ic: "link" };
    default: return null;
  }
}

const NOW = new Date(2026, 9, 7, 13, 39);
const at = min => new Date(NOW.getTime() - min * 60000);
const hhmm = d => d.toLocaleTimeString("en-US", { hour: "numeric", minute: "2-digit" });
const ago = min => min < 60 ? min + "m" : min < 1440 ? Math.floor(min / 60) + "h " + (min % 60) + "m" : Math.floor(min / 1440) + "d " + Math.floor((min % 1440) / 60) + "h";
function dayOf(min) { const d = at(min), t = new Date(NOW); t.setHours(0, 0, 0, 0); const diff = Math.round((t - new Date(d.getFullYear(), d.getMonth(), d.getDate())) / 864e5); return diff <= 0 ? "Today" : diff === 1 ? "Yesterday" : d.toLocaleDateString("en-US", { weekday: "long" }); }
const stamp = min => { const d = at(min); return d.toLocaleDateString("en-US", { month: "short", day: "numeric" }) + ", " + hhmm(d); };
const senderOf = m => m.from[0] || m.from[1];
const initials = s => { const w = s.replace(/@.*/, "").split(/[\s._-]+/).filter(Boolean); return ((w[0] || "?")[0] + (w[1] ? w[1][0] : "")).toUpperCase(); };

Object.assign(window, { KINDS, KIND, STATUS, MAILBOXES, SHIPMENTS, SHP, CUSTOMERS, MESSAGES, nextStep, stepCopy, NOW, at, hhmm, ago, dayOf, stamp, senderOf, initials });
