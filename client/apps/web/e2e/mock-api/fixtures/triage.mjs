// Triage demo data, ported from design_handoff_triage/design/{data,data2,briefing,app}.jsx.
// Every timestamp is an offset from the scenario anchor (the prototype's NOW, 13:39), so
// "today", "Xm left" and "Xh over" read the way the prototype draws them.
import { USER } from "./session.mjs";

const MIN = 60;
const HOUR = 3600;
const DAY = 86400;

export const ME_ID = USER.id;

export const KIND_ORDER = [
  "Tender",
  "RateConfirmation",
  "ProofOfDelivery",
  "Invoice",
  "StatusRequest",
  "DetentionDispute",
  "Other",
];

export const KIND_LABEL = {
  Tender: "Tender",
  RateConfirmation: "Rate confirmation",
  ProofOfDelivery: "Proof of delivery",
  Invoice: "Invoice",
  StatusRequest: "Status request",
  DetentionDispute: "Detention dispute",
  Other: "Other",
};

// data2.jsx SLA, which is the server's default response targets.
export const DEFAULT_TARGETS = {
  Tender: 120,
  RateConfirmation: 120,
  StatusRequest: 60,
  DetentionDispute: 1440,
  ProofOfDelivery: 480,
  Invoice: 1440,
  Other: 2880,
};

export const TEAM = [
  { id: ME_ID, name: "System Administrator", emailAddress: "admin@trenova.app", role: "You" },
  { id: "usr_dana", name: "Dana Whitfield", emailAddress: "dana.whitfield@trenova.app", role: "Billing" },
  { id: "usr_marcus", name: "Marcus Lee", emailAddress: "marcus.lee@trenova.app", role: "Dispatch" },
  { id: "usr_priya", name: "Priya Shah", emailAddress: "priya.shah@trenova.app", role: "Customer service" },
];
const USER_BY_NAME = Object.fromEntries(TEAM.map((u) => [u.name, u.id]));

export const DOCUMENT_TYPES = [
  { id: "dt_rate_confirmation", code: "RATECON", name: "Rate confirmation" },
  { id: "dt_proof_of_delivery", code: "POD", name: "Proof of delivery" },
  { id: "dt_bill_of_lading", code: "BOL", name: "Bill of lading" },
  { id: "dt_invoice", code: "INV", name: "Invoice" },
  { id: "dt_tender", code: "TENDER", name: "Tender" },
  { id: "dt_other", code: "OTHER", name: "Other" },
];

export const CUSTOMERS = [
  { id: "cus_northwind", code: "NORTHWIND", name: "Northwind Foods", sinceYear: 2023, domain: "northwindfoods.example" },
  { id: "cus_harborline", code: "HARBORLINE", name: "Harborline Grocers", sinceYear: 2020, domain: "harborline.example" },
  { id: "cus_crestline", code: "CRESTLINE", name: "Crestline Retail", sinceYear: 2021, domain: "crestline.example" },
  { id: "cus_acme", code: "ACME", name: "Acme Manufacturing", sinceYear: 2019, domain: "acme.example" },
  { id: "cus_blueridge", code: "BLUERIDGE", name: "Blue Ridge Brokerage", sinceYear: 2022, domain: "blueridgebrokerage.example", broker: true },
];
const CUSTOMER_BY_NAME = Object.fromEntries(CUSTOMERS.map((c) => [c.name, c.id]));

export const CARRIERS = [
  { id: "car_kestrel", name: "Kestrel Freight", domain: "kestrelfreight.example" },
  { id: "car_bridger", name: "Bridger Line Haul", domain: "bridgerline.example" },
  { id: "car_blackrock", name: "Blackrock Carriers", domain: "blackrockcarriers.example" },
];

// data.jsx MAILBOXES; MB_HEALTH says Claims stopped receiving on Oct 1, seven days back.
const MAILBOXES = [
  { key: "mb1", id: "imb_intake", name: "Intake", address: "intake@dev.trenova.app", purpose: "Tenders, PODs and status requests", bar: 0.75 },
  { key: "mb2", id: "imb_billing", name: "Billing", address: "ap@dev.trenova.app", purpose: "Carrier invoices", bar: 0.85 },
  { key: "mb3", id: "imb_claims", name: "Claims", address: "claims@dev.trenova.app", purpose: "Disputes and claims", bar: null, quietDays: 7 },
];
const MAILBOX_ID = Object.fromEntries(MAILBOXES.map((m) => [m.key, m.id]));

// data.jsx SHIPMENTS joined with data2.jsx RECX. Offsets are seconds from the anchor.
const SHIPMENTS = [
  { key: "s1", id: "shp_104233", pro: "S-104233", o: "Joliet, IL", d: "Indianapolis, IN", bol: "77120", po: "4471", status: "Completed", p: 1, drv: "Luis Ortega", eq: "Dry van 53'", cust: "Northwind Foods", etaText: "Delivered Oct 6, 15:12", delivered: -(22 * HOUR + 27 * MIN), docs: ["Bill of lading", "Rate confirmation"] },
  { key: "s2", id: "shp_seedpay002", pro: "SEED-PAY-002", o: "Savannah, GA", d: "Atlanta, GA", bol: "30218", po: "99-210", status: "Completed", p: 1, drv: "Ana Ruiz", eq: "Reefer 53'", cust: "Harborline Grocers", etaText: "Delivered Oct 3, 13:22", delivered: -(4 * DAY + 17 * MIN), docs: ["Bill of lading", "Proof of delivery", "Invoice"] },
  { key: "s3", id: "shp_104251", pro: "S-104251", o: "Columbus, OH", d: "Louisville, KY", bol: "77188", po: "", status: "InTransit", p: 0.35, drv: "Tom Becker", eq: "Dry van 53'", cust: "Blue Ridge Brokerage", etaText: "ETA 16:40 today", eta: 3 * HOUR + 1 * MIN, docs: ["Bill of lading"] },
  { key: "s4", id: "shp_104198", pro: "S-104198", o: "Dallas, TX", d: "Memphis, TN", bol: "76902", po: "88213", status: "Delayed", p: 0.62, drv: "Kim Tran", eq: "Dry van 53'", cust: "Crestline Retail", etaText: "ETA 16:10 · 2h late", eta: 2 * HOUR + 31 * MIN, late: true, docs: ["Bill of lading", "Rate confirmation"] },
  { key: "s190", id: "shp_104190", pro: "S-104190", o: "Toledo, OH", d: "Lansing, MI", bol: "77031", po: "4469", status: "New", p: 0, drv: "", eq: "Dry van 53'", cust: "Acme Manufacturing", etaText: "Picks up Oct 13", eta: 6 * DAY, docs: [] },
];

const LANES = [
  ["Joliet, IL", "Indianapolis, IN"], ["Columbus, OH", "Louisville, KY"], ["Dallas, TX", "Memphis, TN"],
  ["Savannah, GA", "Atlanta, GA"], ["Chicago, IL", "Columbus, OH"], ["Toledo, OH", "Fort Wayne, IN"],
  ["Gary, IN", "Milwaukee, WI"], ["Nashville, TN", "Birmingham, AL"], ["Kansas City, MO", "Omaha, NE"],
];

/* body: paragraphs; {{text|what Nova read it as}} marks a phrase the reading used */
const MESSAGES = [
  { id: "m1", from: ["Northwind Foods Dispatch", "dispatch@northwindfoods.example"], to: ["intake@dev.trenova.app"], subject: "New load: Chicago IL to Columbus OH, pickup Thursday 08:00", min: 1102, status: "InReview", kind: "Tender", conf: 0.68, mb: "mb1", cust: "Northwind Foods", reason: "Sender domain northwindfoods.example belongs to Northwind Foods, a customer since 2023.",
    body: ["Hi team —", "We have a {{dry van|Equipment}} load ready {{Thursday morning|Pickup · Thu Oct 8, 08:00}}. {{42,000 lbs|Weight}}, no appointment needed at origin, delivery is a {{2-hour window|Delivery window}}. Rate confirmation to follow once you accept.", "Origin is our {{Chicago DC, 2400 S Halsted|Origin}} and it delivers to {{Columbus OH|Destination}}.", "Thanks,\nMaya — Northwind dispatch"], atts: [], run: "run_8812" },
  { id: "m2", th: "t-harb", from: ["Harborline Grocers Claims", "claims@harborline.example"], to: ["claims@dev.trenova.app"], subject: "Disputing detention on SEED-PAY-002", min: 1222, status: "InReview", kind: "DetentionDispute", conf: 0.91, mb: "mb3", shp: "s2", cust: "Harborline Grocers", reason: "The PRO number SEED-PAY-002 is in the subject line.",
    body: ["Hello,", "We show the driver arriving {{40 minutes after the appointment window|Late arrival claim}} on {{Oct 3|Stop date}}, so we are disputing the {{$185.00 detention|Disputed amount}} on invoice {{INV-20931|Invoice}}.", "Please send the gate log or ELD record if you believe otherwise.", "Regards,\nHarborline Claims"], quoted: "On Oct 4, Trenova Billing wrote:\n> Attached is invoice INV-20931 for SEED-PAY-002, including 2.5h detention at the consignee.", atts: [{ n: "INV-20931.pdf", size: "88 KB", kind: "Invoice", doc: "d9" }], run: "run_8790" },
  { id: "m3", from: ["Partner Carrier EDI", "edi@partnercarrier.example"], to: ["intake@dev.trenova.app"], subject: "Automated: load status file 20260919-0441", min: 1462, status: "InReview", kind: "StatusRequest", conf: 0.41, mb: "mb1", fail: "The body was cut off by the sending system, so Nova could not read which load it is about.", failCode: "body_truncated", body: ["ST*214*0441~B10*PC0441**SCAC~LX*1~AT7*X3*NS***20260919*0412~MS1*COLUMBUS*OH~"],
    extraFacts: [{ key: "file", label: "File", value: "214 · 0441", quote: "ST*214*0441" }, { key: "city", label: "City", value: "Columbus OH", quote: "MS1*COLUMBUS*OH" }], atts: [], run: "run_8781" },
  { id: "m4", from: ["", "noreply@unknownsender.example"], to: ["intake@dev.trenova.app"], subject: "", min: 5302, status: "Quarantined", kind: "Other", conf: 0.22, mb: "mb1", spam: 0.92, fail: "Held back: the sender is not on this mailbox's allow list and the message carried a link to a sign-in page.", failCode: "suspected_phishing", body: ["Your shipping account requires verification. Sign in to keep receiving loads: hxxp://trenova-verify.example/login"], atts: [], run: "run_8702" },
  { id: "m5", from: ["Blue Ridge Brokerage", "loads@blueridgebrokerage.example"], to: ["intake@dev.trenova.app"], subject: "Rate con — S-104251 Columbus to Louisville", min: 54, status: "InReview", kind: "RateConfirmation", conf: 0.83, mb: "mb1", shp: "s3", cust: "Blue Ridge Brokerage", reason: "The PRO number S-104251 is in the subject, and the attached PDF names the same lane.",
    body: ["Please find the signed rate confirmation for {{S-104251|Shipment}} attached. {{$1,240 all-in|Rate}}, {{Columbus OH → Louisville KY|Lane}}.", "Sign and return within 2 hours to hold the truck."], atts: [{ n: "RateCon_S-104251_signed.pdf", size: "212 KB", kind: "RateConfirmation", doc: "d1" }], run: "run_8840" },
  { id: "m6", from: ["Luis Ortega", "luis.ortega@drivers.trenova.example"], to: ["pod@dev.trenova.app"], cc: ["dispatch@dev.trenova.app"], subject: "POD S-104233", min: 196, status: "InReview", kind: "ProofOfDelivery", conf: 0.94, mb: "mb1", shp: "s1", reason: "The driver is assigned to S-104233 and the subject names it.",
    body: ["Delivered, signed by receiving. {{2 pallets short-shipped|Exception · 2 pallets short}} noted on the BOL.", "Sent from my phone"], warn: ["exception"], atts: [{ n: "IMG_4410.jpg", size: "2.4 MB", kind: "ProofOfDelivery", doc: "d2" }, { n: "IMG_4411.HEIC", size: "31.8 MB", kind: "Unknown", fail: "Refused: larger than this mailbox's 25 MB limit per file." }], run: "run_8832" },
  { id: "m7", th: "t-crest", from: ["Crestline Retail", "transport@crestline.example"], to: ["intake@dev.trenova.app"], subject: "Where is PO 88213?", min: 22, status: "InReview", kind: "StatusRequest", conf: 0.88, mb: "mb1", shp: "s4", cust: "Crestline Retail", reason: "PO 88213 is on S-104198 for Crestline Retail.",
    body: ["Hi, can you give me an ETA on {{PO 88213|Purchase order}}? Our receiving closes at {{17:00|Receiving cutoff}} today."], atts: [], run: "run_8845" },
  { id: "m8", from: ["Acme Manufacturing", "logistics@acme.example"], to: ["intake@dev.trenova.app"], subject: "Tender 4471 — 3 loads week of Oct 12", min: 312, status: "InReview", kind: "Tender", conf: 0.72, mb: "mb1", cust: "Acme Manufacturing", reason: "Sender domain matches Acme Manufacturing.",
    body: ["Attached is tender 4471 for three dry van loads the week of Oct 12. Please confirm by end of day Thursday."], atts: [{ n: "tender_4471.pdf", size: "164 KB", kind: "Other", doc: "d3" }], run: "run_8829" },
  { id: "m9", from: ["Kestrel Freight AP", "ap@kestrelfreight.example"], to: ["ap@dev.trenova.app"], subject: "Invoice KF-5521 for S-104198", min: 410, status: "Actioned", kind: "Invoice", conf: 0.97, mb: "mb2", shp: "s4", carrier: "car_kestrel", reason: "Invoice references S-104198.", by: null, byMin: 409, done: "Filed KF-5521.pdf on S-104198 and queued it for carrier pay.", rule: "r2", ruleAction: "FileAndQueuePay",
    body: ["Invoice KF-5521 attached for S-104198. Net 30."], atts: [{ n: "KF-5521.pdf", size: "74 KB", kind: "Invoice", doc: "d4", filed: true }], run: "run_8820" },
  { id: "m10", th: "t-harb", from: ["Harborline Grocers", "ops@harborline.example"], to: ["intake@dev.trenova.app"], subject: "Re: appointment for Friday", min: 1640, status: "Actioned", kind: "StatusRequest", conf: 0.8, mb: "mb1", shp: "s2", cust: "Harborline Grocers", reason: "Thread continues one about SEED-PAY-002.", by: "Dana Whitfield", byMin: 1580, done: "Called receiving, moved the appointment to 10:00.",
    body: ["Can we move Friday's appointment to 10:00?"], atts: [], run: "run_8760" },
  { id: "m11", from: ["FreightWeekly", "news@freightweekly.example"], to: ["intake@dev.trenova.app"], subject: "This week in freight: spot rates soften", min: 2900, status: "Ignored", kind: "Other", conf: 0.9, mb: "mb1", by: "Dana Whitfield", byMin: 2800, done: "Newsletter.",
    body: ["Spot rates eased for a third week as capacity loosened across the Midwest…"], atts: [], run: "run_8711" },
  { id: "m12", th: "t-crest", from: ["Crestline Retail", "transport@crestline.example"], to: ["intake@dev.trenova.app"], subject: "PO 88213 — pickup confirmed?", min: 3020, status: "Actioned", kind: "StatusRequest", conf: 0.86, mb: "mb1", shp: "s4", cust: "Crestline Retail", reason: "PO 88213 is on S-104198.", by: "Priya Shah", byMin: 2990, done: "Confirmed pickup for Mon 09:00.", body: ["Just checking pickup for PO 88213 is still Monday morning?"], atts: [], run: "run_8700" },
];

// data2.jsx DOCREAD: what the extraction read from each attached document.
export const DOCREAD = {
  d1: { type: "Rate confirmation", conf: 0.96, pages: 2, fields: [["Load", "S-104251"], ["Rate", "$1,240.00 all-in"], ["Pickup", "Oct 7, 14:00 · Columbus OH"], ["Delivery", "Oct 8, 08:00 · Louisville KY"], ["Equipment", "53' dry van"], ["Signed", "Yes · Oct 7, 12:41"]] },
  d2: { type: "Proof of delivery", conf: 0.93, pages: 1, fields: [["BOL", "77120"], ["Received by", "R. Patel"], ["Delivered", "Oct 6, 15:12"], ["Signature", "Present"], ["Exceptions", "2 pallets short", "w"]] },
  d3: { type: "Tender", conf: 0.88, pages: 3, fields: [["Tender", "4471"], ["Loads", "3"], ["Week of", "Oct 12"], ["Equipment", "Dry van"], ["Respond by", "Thu Oct 8"]] },
  d9: { type: "Invoice", conf: 0.98, pages: 1, fields: [["Invoice", "INV-20931"], ["Linehaul", "$1,420.00"], ["Detention 2.5h", "$185.00"], ["Total", "$1,605.00"]] },
  d4: { type: "Invoice", conf: 0.97, pages: 1, fields: [["Invoice", "KF-5521"], ["Carrier", "Kestrel Freight"], ["Amount", "$2,180.00"], ["Terms", "Net 30"]] },
  pod: { type: "Proof of delivery", conf: 0.95, pages: 1, fields: [["Received by", "Signed"], ["Signature", "Present"], ["Exceptions", "None"]] },
  inv: { type: "Invoice", conf: 0.97, pages: 1, fields: [["Terms", "Net 30"]] },
  rc: { type: "Rate confirmation", conf: 0.95, pages: 1, fields: [["Signed", "Yes"]] },
};

// data2.jsx PAGES: files the reading saw as several documents.
export const PAGES = {
  "RateCon_S-104251_signed.pdf": [["Rate confirmation", "Rate confirmation"], ["Signed BOL", "Bill of lading"]],
  "tender_4471.pdf": [["Cover letter", "Other"], ["Load sheet", "Tender"], ["BOL template", "Bill of lading"]],
};

// data2.jsx CONFLICTS, as the structured clashes the server returns.
const CONFLICTS = {
  m5: [{ id: "dup-ratecon", kind: "DuplicateDocument", attachment: 0, existingDocumentId: "doc_ratecon_104251_v1", existingShipment: "s3", existingFiledAt: -(31 * HOUR), existingAmount: "1180.00", newAmount: "1240.00", options: ["ReplaceExisting", "KeepBoth"] }],
  m8: [{ id: "overlap-load-2", kind: "OverlappingLoad", existingShipment: "s190", loadIndex: 2, options: ["SkipDuplicate", "ConfirmNew"] }],
};

// data2.jsx STOPS: the stop times a detention dispute turns on (Oct 3, four days back).
export const STOPS = {
  m2: { stopName: "Harborline Grocers DC · Atlanta, GA", day: -4, appointmentStart: 8 * HOUR, appointmentEnd: 10 * HOUR, arrivedAt: 10 * HOUR + 40 * MIN, checkedInAt: 10 * HOUR + 52 * MIN, departedAt: 13 * HOUR + 22 * MIN, lateMinutes: 40, billedDetentionMinutes: 150, billedDetentionAmount: "185.00", detentionPaysWhenLate: false, arrivalSource: "ELD" },
};

// data2.jsx SEED: who owns what when the scenario starts.
const SEED = { m2: { owner: "Dana Whitfield" }, m6: { owner: "System Administrator" } };

// data2.jsx RULES0, in the server's terms. `n` is this month's count.
export const RULES0 = [
  { id: "trule_r1", key: "r1", classification: "ProofOfDelivery", senderScope: "Domain", sender: "drivers.trenova.example", mb: "mb1", action: "FileOnShipment", n: 142, total: 1260, enabled: true, by: "Dana Whitfield", ageDays: 210 },
  { id: "trule_r2", key: "r2", classification: "Invoice", senderScope: "Anyone", sender: "", mb: "mb2", action: "FileAndQueuePay", n: 61, total: 540, enabled: true, by: "Dana Whitfield", ageDays: 180 },
  { id: "trule_r3", key: "r3", classification: "Other", senderScope: "Anyone", sender: "", mb: "mb1", action: "Ignore", n: 210, total: 1890, enabled: true, by: "System Administrator", ageDays: 300 },
  { id: "trule_r4", key: "r4", classification: "StatusRequest", senderScope: "AnyCustomer", sender: "", mb: "mb1", action: "DraftReply", n: 0, total: 0, enabled: false, by: "System Administrator", ageDays: 12 },
];

// data2.jsx TEMPLATES0.
export const TEMPLATES0 = [
  { id: "ttpl_t1", name: "ETA update", body: "Hi {first} — {pro} is {status}. Current ETA is {eta}. We'll tell you if that moves.\n\nThanks,\nTrenova", by: "Priya Shah", ageDays: 90 },
  { id: "ttpl_t2", name: "Detention waiver", body: "Hi {first} — you're right. We've removed the detention on {pro}; a corrected invoice follows today.\n\nThanks,\nTrenova Billing", by: "Dana Whitfield", ageDays: 60 },
  { id: "ttpl_t3", name: "Ask for the POD", body: "Hi {first} — could you send the signed POD for {pro}? A clear photo of each page works.\n\nThanks,\nTrenova", by: "Dana Whitfield", ageDays: 45 },
  { id: "ttpl_t4", name: "Accept the tender", body: "Hi {first} — we'll take it. Confirming {lane}. Rate confirmation to follow.\n\nThanks,\nTrenova Dispatch", by: "Marcus Lee", ageDays: 30 },
];

// data2.jsx VIEWS0.
export const VIEWS0 = [
  { id: "tview_v1", name: "Acme Manufacturing", query: "acme" },
  { id: "tview_v2", name: "Has a PRO", query: "S-10" },
];

// data2.jsx VIEWING: who else has a message open when the scenario starts.
export const VIEWING = [
  { messageId: "imsg_m2", userId: "usr_dana", connectionId: "conn_dana", name: "Dana Whitfield", activity: "viewing" },
  { messageId: "imsg_m7", userId: "usr_priya", connectionId: "conn_priya", name: "Priya Shah", activity: "writing" },
];

// data2.jsx LIVE_POOL: what arrives every 26 seconds in the live scenario.
export const LIVE_POOL = [
  { from: ["Kestrel Freight", "dispatch@kestrelfreight.example"], subject: "Truck available Columbus Thu AM", kind: "Other", conf: 0.71, mb: "mb1", carrier: "car_kestrel", reason: "Sender domain belongs to Kestrel Freight, a carrier.", body: ["We have a 53' dry van empty in Columbus Thursday morning if you have anything heading south."], atts: [] },
  { from: ["Crestline Retail", "transport@crestline.example"], subject: "PO 88240 — can you pick up early?", kind: "StatusRequest", conf: 0.9, mb: "mb1", shp: "s4", cust: "Crestline Retail", reason: "PO 88240 is the next Crestline load on this lane.", body: ["Any chance the {{PO 88240|Purchase order}} pickup can move to {{07:00 tomorrow|Requested pickup}}? Dock is free early."], atts: [] },
  { from: ["Luis Ortega", "luis.ortega@drivers.trenova.example"], subject: "IMG_4411 again (smaller)", kind: "ProofOfDelivery", conf: 0.95, mb: "mb1", shp: "s1", reason: "Same driver and PRO as this morning's POD.", body: ["Resending the second page smaller."], atts: [{ n: "IMG_4411_small.jpg", size: "1.1 MB", kind: "ProofOfDelivery", doc: "d2" }] },
];

// app.jsx DRAFTS: Nova's load drafts and reply drafts.
export const LOAD_DRAFTS = {
  imsg_m1: { source: "", fields: [["Customer", "Northwind Foods"], ["Origin", "Chicago DC, 2400 S Halsted"], ["Destination", ""], ["Pickup", "Thu Oct 8, 08:00"], ["Delivery", ""], ["Equipment", "Dry van"], ["Weight", "42,000 lb"], ["Rate", ""]] },
  imsg_m8: { source: "tender_4471.pdf", fields: [["Customer", "Acme Manufacturing"], ["Loads", "3 × dry van"], ["WeekOf", "Oct 12"], ["Origin", "Acme Plant 2, Toledo OH"], ["Destination", "Fort Wayne, Lansing, Dayton"], ["RespondBy", "Thu Oct 8"]] },
};
export const REPLY_DRAFTS = {
  imsg_m7: "Hi — PO 88213 is on S-104198, Dallas to Memphis. The truck is running about two hours behind; current ETA is 16:10, ahead of your 17:00 receiving cutoff. We'll tell you if that moves.\n\nThanks,\nTrenova",
};
export const DEFAULT_REPLY = "Hi — thanks for this. We're on it and will confirm shortly.\n\nThanks,\nTrenova";
// actions.jsx StopsX: the reply each stop-times decision drafts.
export const STOPS_REPLIES = {
  Waive: "Hi — you're right. The ELD shows arrival at 10:40, 40 minutes after the 08:00–10:00 window, so we've removed the $185.00 detention from INV-20931. A corrected invoice follows today.\n\nThanks,\nTrenova Billing",
  StandBy: "Hi — attached is the ELD record and gate log for SEED-PAY-002 on Oct 3. Happy to talk it through.\n\nThanks,\nTrenova Billing",
  StandByNoRecord: "Hi — the ELD shows the truck arriving within the appointment window on Oct 3, so the detention stands. We can send that record if you'd like it.\n\nThanks,\nTrenova Billing",
};
// app.jsx sendReply: the EDI sender refuses replies.
export const REFUSING_SENDERS = {
  "edi@partnercarrier.example": {
    failureText: "edi@partnercarrier.example doesn't accept replies — their server returned 550.",
    suggestedAddress: "ops@partnercarrier.example",
  },
};
// app.jsx wait(): what a sender says back when they answer a thread.
export const THEY_REPLY = {
  StatusRequest: "Perfect, thanks — we'll hold the dock until 17:30 just in case.",
  DetentionDispute: "Thanks for looking into it. Corrected invoice received.",
  default: "Thanks — got it.",
};
// app.jsx PRO_N: created loads are numbered from S-104260.
export const FIRST_CREATED_PRO = 60;

// briefing.jsx NOVA_T: when Nova settled the day's mail, minutes before now.
const NOVA_T = [[460, 3], [430, 5], [395, 6], [350, 4], [300, 5], [250, 3], [170, 2], [80, 1], [35, 1]];
// list.jsx SAMPLE: the lines a rule's last match and Nova's receipts quote.
const POD_SAMPLES = [["IMG_4402.jpg", "S-104219"], ["POD_77114.jpg", "S-104207"], ["IMG_4398.jpg", "S-104201"]];
const INVOICE_SAMPLES = [["KF-5518", "car_kestrel"], ["INV-8841", "car_bridger"], ["BR-2210", "car_blackrock"], ["KF-5512", "car_kestrel"], ["INV-8830", "car_bridger"]];
const NEWSLETTERS = [
  ["FreightWeekly", "news@freightweekly.example", "This week in freight: tender rejections tick up"],
  ["DAT Trendlines", "trendlines@dat.example", "Trendlines: van rates hold steady"],
  ["Logistics Events", "events@logisticsevents.example", "Webinar invite: peak season planning"],
  ["Truckstop", "news@truckstop.example", "Load board update: Midwest volume up 4%"],
  ["FreightWeekly", "news@freightweekly.example", "Fuel watch: diesel down 3 cents"],
  ["DAT Trendlines", "trendlines@dat.example", "Spot market recap for the week"],
  ["Logistics Events", "events@logisticsevents.example", "Last call: carrier summit registration"],
  ["Truckstop", "news@truckstop.example", "New feature: saved lane alerts"],
  ["FreightWeekly", "news@freightweekly.example", "Capacity check: reefer tightens in the Southeast"],
];
const DRIVERS = [
  ["Luis Ortega", "luis.ortega@drivers.trenova.example"],
  ["Tom Becker", "tom.becker@drivers.trenova.example"],
  ["Kim Tran", "kim.tran@drivers.trenova.example"],
  ["Ana Ruiz", "ana.ruiz@drivers.trenova.example"],
];
const CARRIER_SENDERS = {
  car_kestrel: ["Kestrel Freight AP", "ap@kestrelfreight.example"],
  car_bridger: ["Bridger Line Haul", "billing@bridgerline.example"],
  car_blackrock: ["Blackrock Carriers", "ar@blackrockcarriers.example"],
};
// What people closed by hand today (briefing.jsx "you closed … + 6").
const CLOSED_TODAY = [
  { from: ["Crestline Retail", "transport@crestline.example"], subject: "PO 88190 — ETA?", kind: "StatusRequest", cust: "Crestline Retail", min: 380, byMin: 368, note: "Sent the ETA from S-104226." },
  { from: ["Northwind Foods Dispatch", "dispatch@northwindfoods.example"], subject: "PO 88174 status", kind: "StatusRequest", cust: "Northwind Foods", min: 300, byMin: 290, note: "Replied with the delivery window." },
  { from: ["Harborline Grocers", "ops@harborline.example"], subject: "Dock change for Thursday", kind: "StatusRequest", cust: "Harborline Grocers", min: 240, byMin: 226, note: "Moved the appointment to dock 4." },
  { from: ["Acme Manufacturing", "logistics@acme.example"], subject: "Pickup number for 4466", kind: "Other", cust: "Acme Manufacturing", min: 190, byMin: 181, note: "Sent pickup number 4466-B." },
  { from: ["Summit Logistics", "rates@summitlogistics.example"], subject: "Rate con — S-104228", kind: "RateConfirmation", min: 140, byMin: 131, note: "Filed and signed." },
  { from: ["Kestrel Freight", "dispatch@kestrelfreight.example"], subject: "Driver info for S-104226", kind: "StatusRequest", carrier: "car_kestrel", min: 75, byMin: 66, note: "Sent the driver's name and cell." },
];

const ACTIVITY_BASE = { until: null, fromClassification: null, toClassification: null, recordId: null, subject: "", detail: "" };

function parseSize(size) {
  const [value, unit] = size.split(" ");
  const n = Number(value);
  return Math.round(unit === "MB" ? n * 1048576 : unit === "KB" ? n * 1024 : n);
}

function contentTypeOf(fileName) {
  const ext = fileName.split(".").pop().toLowerCase();
  return { pdf: "application/pdf", jpg: "image/jpeg", jpeg: "image/jpeg", png: "image/png", heic: "image/heic" }[ext] ?? "application/octet-stream";
}

function camel(label) {
  const words = label.replace(/[^A-Za-z0-9 ]+/g, " ").trim().split(/\s+/);
  return words.map((w, i) => (i === 0 ? w.toLowerCase() : w[0].toUpperCase() + w.slice(1).toLowerCase())).join("");
}

/** Turns the prototype's marked paragraphs into the stored body and the facts Nova read from it. */
export function readBody(paragraphs, { quoted = "", warn = [], extraFacts = [] } = {}) {
  const facts = [];
  const plain = paragraphs.map((p) =>
    p.replace(/\{\{([^|}]+)\|([^}]+)\}\}/g, (_all, text, label) => {
      const [name, value] = label.split(" · ");
      let key = camel(name);
      while (facts.some((f) => f.key === key)) key += "2";
      facts.push({ key, label: name, value: value ?? text, quote: text, tone: warn.includes(key) ? "Warning" : "Neutral", flagged: false });
      return text;
    }),
  );
  for (const fact of extraFacts) facts.push({ tone: "Neutral", flagged: false, ...fact });
  const text = plain.join("\n\n") + (quoted ? `\n\n${quoted}` : "");
  return { text, facts };
}

function splitPlace(place) {
  const [city, state] = place.split(", ");
  return { city, state: state ?? "" };
}

/** Builds the scenario's triage world: mailboxes, shipments, customers, messages, rules and the rest. */
export function buildTriageWorld({ anchor, scenario }) {
  const at = (offsetSeconds) => anchor + offsetSeconds;
  const dayStart = anchor - (anchor % DAY);
  const ids = { activity: 0, attachment: 0, document: 0, reply: 0 };
  const nextId = (kind, prefix) => `${prefix}_${String(++ids[kind]).padStart(4, "0")}`;

  const shipments = new Map();
  const addShipment = (s) => {
    const origin = splitPlace(s.o);
    const destination = splitPlace(s.d);
    const record = {
      id: s.id,
      proNumber: s.pro,
      bol: s.bol ?? "",
      poNumber: s.po ?? "",
      originCity: origin.city,
      originState: origin.state,
      destinationCity: destination.city,
      destinationState: destination.state,
      status: s.status,
      progress: s.p,
      late: !!s.late,
      driverName: s.drv ?? "",
      equipment: s.eq ?? "Dry van 53'",
      customerId: CUSTOMER_BY_NAME[s.cust] ?? null,
      etaText: s.etaText ?? "",
      eta: s.eta != null ? at(s.eta) : null,
      deliveredAt: s.delivered != null ? at(s.delivered) : null,
      docs: [...(s.docs ?? [])],
      createdAt: at(-(s.ageDays ?? 6) * DAY),
    };
    shipments.set(record.id, record);
    return record;
  };
  for (const s of SHIPMENTS) addShipment(s);
  const shipmentByKey = Object.fromEntries(SHIPMENTS.map((s) => [s.key, s.id]));
  const historyShipment = (pro, cust, lane, deliveredOffset) =>
    shipments.get(`shp_${pro.replace(/\D/g, "")}`) ??
    addShipment({
      id: `shp_${pro.replace(/\D/g, "")}`,
      pro,
      o: lane[0],
      d: lane[1],
      bol: String(70000 + (Number(pro.replace(/\D/g, "")) % 9000)),
      po: String(88000 + (Number(pro.replace(/\D/g, "")) % 500)),
      status: "Completed",
      p: 1,
      drv: DRIVERS[Number(pro.slice(-1)) % DRIVERS.length][0],
      cust,
      etaText: "Delivered",
      delivered: deliveredOffset,
      docs: ["Bill of lading", "Rate confirmation"],
      ageDays: Math.ceil(-deliveredOffset / DAY) + 2,
    });

  const users = new Map(TEAM.map((u) => [u.id, { ...u }]));
  const customers = new Map(
    CUSTOMERS.map((c) => [c.id, { ...c, since: Date.UTC(c.sinceYear, 2, 14) / 1000 }]),
  );
  const carriers = new Map(CARRIERS.map((c) => [c.id, { ...c }]));

  const activity = (kind, actorKind, actorId, createdAt, extra = {}) => ({
    id: nextId("activity", "iact"),
    kind,
    actorKind,
    actorId: actorId ?? null,
    ...ACTIVITY_BASE,
    ...extra,
    createdAt,
  });

  const attachmentsFor = (messageId, atts, receivedAt, filedShipmentId, filedAt) =>
    atts.map((a) => {
      const reading = a.doc ? DOCREAD[a.doc] : null;
      const pageNames = PAGES[a.n];
      const pageCount = pageNames ? pageNames.length : reading ? reading.pages : a.fail ? 0 : 1;
      const pages = pageNames
        ? pageNames.map(([label, type], i) => ({ number: i + 1, label, suggestedType: type }))
        : Array.from({ length: pageCount }, (_, i) => ({ number: i + 1, label: `Page ${i + 1}`, suggestedType: reading?.type ?? "Other" }));
      const filed = a.filed && filedShipmentId;
      return {
        id: nextId("attachment", `imsga_${messageId.replace("imsg_", "")}`),
        fileName: a.n,
        contentType: contentTypeOf(a.n),
        byteSize: parseSize(a.size),
        kind: a.kind,
        documentId: a.fail || !a.doc ? null : nextId("document", "doc"),
        failureText: a.fail ?? "",
        filedShipmentId: filed ? filedShipmentId : null,
        filedAs: filed ? reading?.type ?? "Other" : null,
        filedAt: filed ? filedAt : null,
        filedBy: null,
        pageCount,
        pages,
        splitDocuments: [],
        reading: reading
          ? {
              documentType: reading.type,
              confidence: reading.conf,
              pageCount: reading.pages,
              fields: reading.fields.map(([label, value, tone]) => ({ label, value, tone: tone === "w" ? "Warning" : "Neutral" })),
            }
          : null,
        createdAt: receivedAt,
      };
    });

  const fresh = scenario.triage === "firstDay";
  const mailboxes = MAILBOXES.map((m, index) => ({
    id: m.id,
    key: m.key,
    name: m.name,
    address: m.address,
    provider: "Postmark",
    purpose: m.purpose,
    reviewPolicy: m.bar == null ? "AlwaysReview" : "ReviewBelowConfidence",
    minConfidence: m.bar ?? 0.9,
    status: "Active",
    hasSigningSecret: true,
    hasApiKey: false,
    lastReceivedAt: fresh ? null : m.quietDays ? at(-m.quietDays * DAY) : null,
    testSentAt: null,
    testReceivedAt: null,
    blockedSenders: [],
    tokenVersion: 1,
    version: fresh ? 1 : 3,
    createdAt: fresh ? at(-(12 - index) * MIN) : at(-(420 - index * 40) * DAY),
    updatedAt: fresh ? at(-(12 - index) * MIN) : at(-(30 - index) * DAY),
  }));

  /** One message, from the prototype's shape to the server's. */
  const makeMessage = (m, { id = `imsg_${m.id}`, receivedAt = at(-m.min * MIN), status = m.status } = {}) => {
    const { text, facts } = readBody(m.body, { quoted: m.quoted, warn: m.warn, extraFacts: m.extraFacts });
    const shipmentId = m.shp ? shipmentByKey[m.shp] ?? m.shp : null;
    const reviewedAt = m.byMin != null ? at(-m.byMin * MIN) : null;
    const reviewer = m.by ? USER_BY_NAME[m.by] : null;
    const settled = status === "Actioned" || status === "Ignored";
    const owner = SEED[m.id]?.owner ? USER_BY_NAME[SEED[m.id].owner] : null;
    const message = {
      id,
      mailboxId: MAILBOX_ID[m.mb] ?? m.mb,
      fromAddress: m.from[1],
      fromName: m.from[0],
      toAddresses: m.to ?? ["intake@dev.trenova.app"],
      ccAddresses: m.cc ?? [],
      subject: m.subject,
      textBody: text,
      receivedAt,
      spamScore: m.spam ?? 0.02,
      classification: m.kind,
      confidence: m.conf,
      status,
      matchedCustomerId: m.cust ? CUSTOMER_BY_NAME[m.cust] : null,
      matchedCarrierId: m.carrier ?? null,
      matchedShipmentId: shipmentId,
      matchReason: m.reason ?? "",
      runId: m.run ?? `run_${id}`,
      reviewedBy: settled ? reviewer : null,
      reviewedAt: settled ? reviewedAt : null,
      reviewNote: settled ? m.done ?? "" : "",
      failureCode: m.failCode ?? "",
      failureText: m.fail ?? "",
      threadKey: m.th ?? `<${id.replace("imsg_", "")}@${m.from[1].split("@")[1]}>`,
      senderDomain: m.from[1].split("@")[1],
      ownerId: owner,
      dueAt: null,
      parked: "",
      snoozedUntil: null,
      snoozeEvent: null,
      snoozeWatchId: null,
      snoozeDocumentTypeId: null,
      snoozedAt: null,
      snoozedBy: null,
      awaitingSince: null,
      awaitingUntil: null,
      novaRead: true,
      handledByNova: settled && !reviewer,
      handledByRuleId: m.rule ? RULES0.find((r) => r.key === m.rule).id : null,
      handledAction: m.ruleAction ?? null,
      classificationCorrected: false,
      facts,
      conflicts: [],
      actionState: {},
      loadDraft: null,
      attachments: [],
      activity: [],
      replies: [],
      version: 1,
      createdAt: receivedAt,
      updatedAt: reviewedAt ?? receivedAt,
    };
    message.attachments = attachmentsFor(id, m.atts ?? [], receivedAt, shipmentId, reviewedAt);
    message.conflicts = (CONFLICTS[m.id] ?? []).map((c) => ({
      id: c.id,
      kind: c.kind,
      attachmentId: c.attachment != null ? message.attachments[c.attachment].id : null,
      existingDocumentId: c.existingDocumentId ?? null,
      existingShipmentId: c.existingShipment ? shipmentByKey[c.existingShipment] : null,
      existingFiledAt: c.existingFiledAt != null ? at(c.existingFiledAt) : null,
      existingAmount: c.existingAmount ?? null,
      newAmount: c.newAmount ?? null,
      loadIndex: c.loadIndex ?? null,
      resolution: null,
      resolvedAt: null,
      resolvedBy: null,
      options: c.options,
    }));
    message.activity.push(activity("NovaRead", "Nova", null, receivedAt + 40, { subject: KIND_LABEL[m.kind] }));
    if (owner) {
      message.activity.push(activity("Assigned", "Person", owner === ME_ID ? ME_ID : "usr_dana", receivedAt + 20 * MIN, { subject: users.get(owner).name, recordId: owner }));
    }
    if (settled) {
      if (message.handledByNova && message.handledAction) {
        message.activity.push(activity("RuleApplied", "Nova", null, reviewedAt, { subject: message.handledAction, recordId: message.handledByRuleId }));
      }
      for (const a of message.attachments.filter((x) => x.filedShipmentId)) {
        message.activity.push(activity("Filed", message.handledByNova ? "Nova" : "Person", reviewer, reviewedAt, { subject: a.fileName, detail: shipments.get(a.filedShipmentId).proNumber, recordId: a.filedShipmentId }));
      }
      message.activity.push(activity(status === "Actioned" ? "Handled" : "Ignored", message.handledByNova ? "Nova" : "Person", reviewer, reviewedAt, { detail: message.reviewNote }));
    }
    return message;
  };

  const messages = [];
  if (scenario.triage === "live" || scenario.triage === "deskOff") {
    messages.push(...MESSAGES.map((m) => makeMessage(m)));
    messages.push(...historyMessages({ at, makeMessage, historyShipment }));
  }

  const rules = RULES0.map((r) => ({
    id: r.id,
    classification: r.classification,
    senderScope: r.senderScope,
    sender: r.sender,
    mailboxId: MAILBOX_ID[r.mb],
    action: r.action,
    assigneeId: null,
    enabled: r.enabled,
    createdById: USER_BY_NAME[r.by],
    baseThisMonth: r.n,
    baseTotal: r.total,
    version: 1,
    createdAt: at(-r.ageDays * DAY),
  }));

  const templates = TEMPLATES0.map((t) => ({
    id: t.id,
    name: t.name,
    body: t.body,
    createdById: USER_BY_NAME[t.by],
    version: 1,
    createdAt: at(-t.ageDays * DAY),
  }));

  const views = VIEWS0.map((v, i) => ({ ...v, position: i + 1, createdAt: at(-(20 - i) * DAY) }));

  return {
    dayStart,
    users,
    customers,
    carriers,
    shipments,
    mailboxes: scenario.triage === "noMailbox" ? [] : mailboxes,
    messages,
    rules,
    templates,
    views,
    ids,
    nextId,
    activity,
    makeMessage,
    attachmentsFor,
    addShipment,
    shipmentByKey,
    mailboxByKey: MAILBOX_ID,
  };
}

/** The mail around the prototype's twelve: what Nova settled today, what people closed today, and Blue Ridge's history. */
function historyMessages({ at, makeMessage, historyShipment }) {
  const out = [];
  const slots = NOVA_T.flatMap(([min, k]) => Array.from({ length: k }, (_, i) => min - i * 9))
    .sort((a, b) => a - b)
    .slice(0, 28);
  const queues = { pod: 14, invoice: 5, newsletter: 9 };
  const totals = { ...queues };
  const used = { pod: 0, invoice: 0, newsletter: 0 };
  slots.forEach((min, index) => {
    const kind = Object.keys(queues)
      .filter((k) => used[k] < totals[k])
      .sort((a, b) => (totals[b] - used[b]) / totals[b] - (totals[a] - used[a]) / totals[a] || a.localeCompare(b))[0];
    const n = used[kind]++;
    if (kind === "pod") {
      const [file, pro] = POD_SAMPLES[n] ?? [`IMG_${4396 - n * 3}.jpg`, `S-1042${String(18 - n).padStart(2, "0")}`];
      const [name, address] = DRIVERS[n % DRIVERS.length];
      const lane = LANES[n % LANES.length];
      const shipment = historyShipment(pro, ["Northwind Foods", "Crestline Retail", "Harborline Grocers", "Acme Manufacturing"][n % 4], lane, -(min + 30) * MIN);
      out.push(makeMessage(
        { id: `h_pod_${n + 1}`, from: [name, address], subject: `POD ${pro}`, min, status: "Actioned", kind: "ProofOfDelivery", conf: 0.95, mb: "mb1", shp: shipment.id, reason: `The driver is assigned to ${pro} and the subject names it.`, byMin: min - 1, done: `Filed ${file} on ${pro}.`, rule: "r1", ruleAction: "FileOnShipment", body: ["Delivered, signed by receiving."], atts: [{ n: file, size: "1.8 MB", kind: "ProofOfDelivery", doc: "pod", filed: true }], run: `run_77${String(index).padStart(2, "0")}` },
      ));
    } else if (kind === "invoice") {
      const [number, carrierId] = INVOICE_SAMPLES[n];
      const [name, address] = CARRIER_SENDERS[carrierId];
      const pro = `S-1042${String(5 + n * 2).padStart(2, "0")}`;
      const shipment = historyShipment(pro, "Crestline Retail", LANES[(n + 3) % LANES.length], -(2 * DAY));
      out.push(makeMessage(
        { id: `h_inv_${n + 1}`, from: [name, address], to: ["ap@dev.trenova.app"], subject: `Invoice ${number} for ${pro}`, min, status: "Actioned", kind: "Invoice", conf: 0.97, mb: "mb2", shp: shipment.id, carrier: carrierId, reason: `Invoice references ${pro}.`, byMin: min - 1, done: `Filed ${number}.pdf on ${pro} and queued it for carrier pay.`, rule: "r2", ruleAction: "FileAndQueuePay", body: [`Invoice ${number} attached for ${pro}. Net 30.`], atts: [{ n: `${number}.pdf`, size: "71 KB", kind: "Invoice", doc: "inv", filed: true }], run: `run_78${String(index).padStart(2, "0")}` },
      ));
    } else {
      const [name, address, subject] = NEWSLETTERS[n];
      out.push(makeMessage(
        { id: `h_news_${n + 1}`, from: [name, address], subject, min, status: "Ignored", kind: "Other", conf: 0.93, mb: "mb1", byMin: min - 1, done: "Newsletter.", rule: "r3", ruleAction: "Ignore", body: [`${subject}. Read the full issue online.`], atts: [], run: `run_79${String(index).padStart(2, "0")}` },
      ));
    }
  });

  CLOSED_TODAY.forEach((c, n) => {
    out.push(makeMessage(
      { id: `h_mine_${n + 1}`, from: c.from, subject: c.subject, min: c.min, status: "Actioned", kind: c.kind, conf: 0.9, mb: "mb1", cust: c.cust, carrier: c.carrier, reason: c.cust ? `Sender domain belongs to ${c.cust}.` : "", by: "System Administrator", byMin: c.byMin, done: c.note, body: [`${c.subject}.`], atts: [], run: `run_80${n}` },
    ));
  });

  // Blue Ridge's rate cons over the last month, every one filed by hand: the
  // briefing's "11 of 11 rate cons from blueridgebrokerage.example by hand".
  for (let n = 0; n < 11; n++) {
    const pro = `S-1041${String(20 + n).padStart(2, "0")}`;
    const days = 2 + n * 2.5;
    const shipment = historyShipment(pro, "Blue Ridge Brokerage", LANES[(n + 1) % LANES.length], -(days - 1) * 86400);
    const file = `RateCon_${pro}_signed.pdf`;
    const min = Math.round(days * 1440);
    out.push(makeMessage(
      { id: `h_brb_${n + 1}`, from: ["Blue Ridge Brokerage", "loads@blueridgebrokerage.example"], subject: `Rate con — ${pro}`, min, status: "Actioned", kind: "RateConfirmation", conf: 0.86, mb: "mb1", shp: shipment.id, cust: "Blue Ridge Brokerage", reason: `The PRO number ${pro} is in the subject.`, by: n % 3 === 0 ? "Dana Whitfield" : "System Administrator", byMin: min - 35, done: `Filed ${file} on ${pro}.`, body: [`Signed rate confirmation for ${pro} attached.`], atts: [{ n: file, size: "198 KB", kind: "RateConfirmation", doc: "rc", filed: true }], run: `run_81${String(n).padStart(2, "0")}` },
    ));
  }
  return out;
}
