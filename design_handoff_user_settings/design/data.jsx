// Notification catalog mirrors components/notification-center/notification-registry.ts
// sev: crit = needs you now, act = something to do, fyi = news. n = sent to this user, last 30 days.
const NX_CATS = [
  { id: "mentions", name: "Mentions", icon: "at", tone: "brand", blurb: "When a teammate @mentions you", ev: [
    { k: "shipment_comment_mention", t: "Someone @mentions you on a shipment", sev: "act", n: 18, msg: "Dana Ortiz mentioned you on S-48213: “@maya can you confirm the reefer temp before pickup?”" },
  ]},
  { id: "assistant", name: "Assistant", icon: "spark", tone: "agent", blurb: "Replies you walked away from", ev: [
    { k: "assistant.reply_ready", t: "A reply you left is ready", sev: "fyi", n: 41, msg: "Your reply is ready: “Which Werner loads are at risk of detention today?”" },
  ]},
  { id: "dispatch", name: "Dispatch", icon: "truck", tone: "brand", blurb: "Tenders, waterfalls and rate confirmations", ev: [
    { k: "tender_accepted", t: "A carrier accepts a tender", sev: "fyi", n: 120, msg: "Werner Enterprises accepted S-48213 · Chicago, IL → Dallas, TX" },
    { k: "tender_needs_review", t: "A tender needs your review", sev: "act", n: 34, msg: "S-48219 needs review — Knight-Swift countered at $2,410 (+6%)" },
    { k: "tender_entries_skipped", t: "Carriers are skipped in a waterfall", sev: "act", n: 9, msg: "2 carriers skipped on S-48230 — insurance expired" },
    { k: "tender_waterfall_exhausted", t: "A waterfall runs out of carriers", sev: "crit", n: 6, msg: "No carrier left for S-48231 · pickup tomorrow 06:00" },
    { k: "rate_confirmation_issue_failed", t: "A rate confirmation fails to issue", sev: "crit", n: 3, msg: "Rate confirmation for S-48202 could not be generated" },
    { k: "tender_delivery_failed", t: "A tender can’t be delivered", sev: "crit", n: 2, msg: "Tender email to dispatch@hartcarriers.com bounced" },
  ]},
  { id: "billing", name: "Billing", icon: "receipt", tone: "warn", blurb: "Invoices and billing exceptions", ev: [
    { k: "invoice_shared", t: "Someone shares an invoice with you", sev: "act", n: 7, msg: "Priya Nair shared INV-20931 with you" },
    { k: "invoice_reconciliation_warning", t: "An invoice won’t reconcile", sev: "act", n: 12, msg: "INV-20944 is $184.00 off the shipment total" },
    { k: "billing_exception_recorded", t: "A billing exception is recorded", sev: "act", n: 15, msg: "Detention exception recorded on S-48188" },
  ]},
  { id: "accounting", name: "Accounting", icon: "bank", tone: "warn", blurb: "Bank receipt reconciliation", ev: [
    { k: "bank_receipt_reconciliation_exception", t: "A bank receipt doesn’t match", sev: "act", n: 5, msg: "Receipt #88120 ($12,400.00) has no matching invoice" },
  ]},
  { id: "carrier", name: "Carrier intelligence", icon: "radar", tone: "danger", blurb: "Monitoring, blocks and spend caps", ev: [
    { k: "carrier_intel_change", t: "A monitored carrier changes", sev: "act", n: 22, msg: "Hart Carriers — authority status changed to Pending" },
    { k: "carrier_intel_digest", t: "Weekly monitoring digest", sev: "fyi", n: 4, msg: "This week: 3 changes across 214 monitored carriers" },
    { k: "carrier_intel_block", t: "A carrier is blocked", sev: "crit", n: 2, msg: "Blue Line Freight blocked — insurance cancelled" },
    { k: "carrier_equipment_mismatch", t: "Equipment doesn’t match the load", sev: "crit", n: 3, msg: "S-48240 needs a reefer; assigned carrier has dry vans only" },
    { k: "carrier_intel_provider_paused", t: "A data provider pauses", sev: "crit", n: 1, msg: "Highway paused — lookups are on hold" },
    { k: "carrier_intel_spend_soft_cap", t: "Lookup spend nears its cap", sev: "act", n: 1, msg: "Carrier lookups are at 80% of this month’s budget" },
  ]},
  { id: "workers", name: "Workers", icon: "badge", tone: "brand", blurb: "PTO and driver credentials", ev: [
    { k: "dash.pto_requested", t: "A worker requests time off", sev: "act", n: 6, msg: "Luis Romero requested Oct 20–24 off" },
    { k: "credential_expiring", t: "A credential is about to expire", sev: "act", n: 11, msg: "Luis Romero’s medical card expires in 14 days" },
    { k: "credential_expired", t: "A credential expired", sev: "crit", n: 2, msg: "Ana Cole’s CDL expired today" },
    { k: "credential_document_uploaded", t: "A credential document is uploaded", sev: "fyi", n: 9, msg: "New HazMat endorsement uploaded for Ana Cole" },
  ]},
  { id: "reports", name: "Reports", icon: "file", tone: "success", blurb: "Runs and scheduled deliveries", ev: [
    { k: "report_run_completed", t: "A report finishes", sev: "fyi", n: 28, msg: "“Lane margin — weekly” is ready to download" },
    { k: "report_run_delivered", t: "A scheduled report is delivered", sev: "fyi", n: 14, msg: "“AR aging” was emailed to 4 people" },
    { k: "report_run_failed", t: "A report fails", sev: "crit", n: 2, msg: "“Fuel surcharge audit” failed — query timed out" },
    { k: "report_schedule_skipped", t: "A schedule is skipped", sev: "act", n: 2, msg: "Monday’s “Detention summary” run was skipped" },
    { k: "report_delivery_email_failed", t: "A report email bounces", sev: "act", n: 1, msg: "“AR aging” couldn’t reach finance@acme.co" },
  ]},
  { id: "edi", name: "EDI", icon: "swap", tone: "danger", blurb: "Messages and inbound files", ev: [
    { k: "edi.message.dead_lettered", t: "A message can’t be processed", sev: "crit", n: 4, msg: "214 from Walmart dead-lettered — segment N1 missing" },
    { k: "edi.inbound_file.quarantined", t: "An inbound file is quarantined", sev: "crit", n: 1, msg: "inbound/2026-10-11-0412.edi quarantined" },
  ]},
  { id: "ai", name: "AI Control", icon: "zap", tone: "agent", blurb: "Proposals, tool trust and audit exports", ev: [
    { k: "agent.proposals_pending", t: "Agent proposals wait for you", sev: "act", n: 10, msg: "Billing agent has 3 proposals waiting for approval" },
    { k: "agent.proposals_reminder", t: "Reminder about waiting proposals", sev: "act", n: 6, msg: "3 proposals have waited more than 4 hours" },
    { k: "agent.tool_promoted", t: "A tool earns more trust", sev: "fyi", n: 3, msg: "“Send rate confirmation” now runs without approval" },
    { k: "agent.tool_demoted", t: "A tool loses trust", sev: "fyi", n: 2, msg: "“Apply accessorial” is back to needing approval" },
    { k: "agent.quality_regression", t: "Agent quality drops", sev: "act", n: 1, msg: "Dispatch agent accuracy fell 9 points this week" },
    { k: "ai_audit.export_ready", t: "An audit export is ready", sev: "fyi", n: 1, msg: "AI audit export for September is ready" },
    { k: "ai_audit.chain_mismatch", t: "The audit chain doesn’t verify", sev: "crit", n: 0, lock: "Your organization requires this for admins", msg: "Audit chain mismatch detected at entry 18,204" },
  ]},
  { id: "data", name: "Data alerts", icon: "db", tone: "info", blurb: "Table change alerts you subscribed to", ev: [
    { k: "tca.*", t: "A table you watch changes", sev: "fyi", n: 26, msg: "Customer “Acme Foods” credit limit changed to $250,000" },
  ]},
  { id: "system", name: "System", icon: "info", tone: "muted", blurb: "Product and account notices", ev: [
    { k: "system", t: "Account and product notices", sev: "fyi", n: 3, lock: "Security notices always reach you in Trenova", lockCh: ["app"], msg: "New sign-in from Chrome on Windows · Dallas, TX" },
  ]},
];
const NX_CH = [["app", "In Trenova", "bell"], ["email", "Email", "mail"], ["desk", "Desktop", "monitor"]];
const NX_SEV = { crit: ["Urgent", "crit"], act: ["To do", "act"], fyi: ["FYI", "fyi"] };

function nxDefaults() {
  const m = {}; NX_CATS.forEach(c => c.ev.forEach(e => { m[e.k] = true; })); return m;
}

const US_TEAM = [["jordan", "Jordan Pike", "Dispatch lead"], ["dana", "Dana Ortiz", "Dispatcher"], ["priya", "Priya Nair", "Billing"], ["sam", "Sam Okafor", "Operations"]];

const US_SESSIONS = [
  { id: "s1", dev: "laptop", t: "Chrome on macOS", where: "Chicago, IL", ip: "73.12.88.4", last: "Active now", me: true },
  { id: "s2", dev: "phone", t: "Safari on iPhone", where: "Chicago, IL", ip: "172.56.3.19", last: "2 hours ago" },
  { id: "s3", dev: "laptop", t: "Edge on Windows", where: "Dallas, TX", ip: "98.4.211.60", last: "Yesterday, 4:12 PM" },
  { id: "s4", dev: "monitor", t: "Firefox on Ubuntu", where: "Naperville, IL", ip: "24.1.90.12", last: "Sep 28" },
];
const US_SIGNINS = [
  ["ok", "Signed in with password + code", "Chrome on macOS · Chicago, IL", "Today, 8:02 AM"],
  ["ok", "Signed in with password + code", "Safari on iPhone · Chicago, IL", "Today, 6:40 AM"],
  ["bad", "Wrong verification code", "Edge on Windows · Dallas, TX", "Yesterday, 4:10 PM"],
  ["ok", "Signed in with password + code", "Edge on Windows · Dallas, TX", "Yesterday, 4:11 PM"],
  ["ok", "Recovery code used", "Firefox on Ubuntu · Naperville, IL", "Sep 28, 9:15 AM"],
];
const US_TZ = [["America/New_York", "Eastern Time", "New York", "UTC−4", "United States"], ["America/Chicago", "Central Time", "Chicago", "UTC−5", "United States"], ["America/Denver", "Mountain Time", "Denver", "UTC−6", "United States"], ["America/Phoenix", "Arizona", "Phoenix", "UTC−7", "United States"], ["America/Los_Angeles", "Pacific Time", "Los Angeles", "UTC−7", "United States"], ["America/Mexico_City", "Mexico City", "Mexico City", "UTC−6", "Americas"], ["America/Toronto", "Toronto", "Toronto", "UTC−4", "Americas"], ["Europe/London", "London", "London", "UTC+1", "Europe"], ["UTC", "UTC", "UTC", "UTC±0", "Other"]];
const US_LANG = [["en-US", "English (US)"], ["es-MX", "Español (México)"], ["fr-CA", "Français (Canada)"], ["pt-BR", "Português (Brasil)"]];

Object.assign(window, { NX_CATS, NX_CH, NX_SEV, nxDefaults, US_TEAM, US_SESSIONS, US_SIGNINS, US_TZ, US_LANG });
