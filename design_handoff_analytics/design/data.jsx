const MOD = /Mac|iPhone|iPad/.test(navigator.platform || navigator.userAgent) ? "⌘" : "Ctrl";
const TASKS = [
  { k: "AssistantChat", l: "Assistant chat", fb: "The assistant can't answer" },
  { k: "BillingDiagnosis", l: "Billing diagnosis", trust: true, fb: "Billing holds wait for a person" },
  { k: "DocumentExtraction", l: "Document extraction", fb: "Documents wait for manual entry" },
  { k: "DocumentClassification", l: "Document classification", fb: "Documents are filed by hand" },
  { k: "ScopeClassification", l: "Scope classification", fb: "Built-in rules check scope" },
  { k: "OperationalInsights", l: "Operational insights", fb: "Insights show without narration" },
  { k: "DailyBriefing", l: "Daily briefing", fb: "No morning briefing" },
  { k: "FormulaAssistant", l: "Formula assistant", fb: "Formulas are written by hand" },
  { k: "Embedding", l: "Embedding", fb: "Search uses words only" },
  { k: "General", l: "General", fb: "Unrouted work is refused" },
];
const TASK_L = Object.fromEntries(TASKS.map(t => [t.k, t.l]));
const PROVIDERS0 = [
  { id: "anthropic", n: "Anthropic", kind: "Anthropic Messages", model: "claude-sonnet-5", m: "A", h: 45, c: 0.09, on: false, key: false, trusted: true, priv: false, base: "", tasks: ["BillingDiagnosis", "AssistantChat"], last: null, wk: null },
  { id: "ollama", n: "Local Ollama", kind: "Ollama", model: "qwen2.5:14b", m: "Ol", h: 0, c: 0, on: true, local: true, key: false, trusted: false, priv: true, base: "http://localhost:11434", tasks: ["ScopeClassification", "DocumentClassification", "OperationalInsights", "DailyBriefing", "General"], last: { ok: true, msg: "Connected · 386 ms", when: "Today 5:40 AM" }, wk: { calls: 672, f: 5, ms: 410, tok: "0.9M" } },
  { id: "vllm", n: "Workstation vLLM", kind: "OpenAI-compatible", model: "meta-llama/Llama-3.3-70B-Instruct", m: "WV", h: 258, c: 0.1, on: true, local: true, key: false, trusted: false, priv: true, base: "http://localhost:8000/v1", tasks: ["AssistantChat", "DocumentExtraction", "FormulaAssistant", "OperationalInsights", "General"], last: { ok: false, msg: "Could not connect", when: "Today 5:21 AM" }, wk: { calls: 612, f: 24, ms: 1420, tok: "2.2M" } },
];
const PRESETS = [
  { k: "anthropic", n: "Anthropic", kind: "Anthropic Messages", model: "claude-sonnet-5", m: "A", h: 45, c: 0.09, trusted: true, ph: "sk-ant-…" },
  { k: "openai", n: "OpenAI", kind: "OpenAI Responses", model: "gpt-5.1", m: "O", h: 165, c: 0.06, ph: "sk-…" },
  { k: "azure", n: "Azure OpenAI", kind: "Azure OpenAI", model: "gpt-5.1", m: "Az", h: 235, c: 0.1, ph: "Azure key" },
  { k: "gemini", n: "Google Gemini", kind: "Gemini", model: "gemini-3-pro", m: "G", h: 265, c: 0.12, ph: "AIza…" },
  { k: "ollama", n: "Ollama", kind: "Ollama", model: "qwen2.5:14b", m: "Ol", h: 0, c: 0, local: true, base: "http://localhost:11434" },
  { k: "compat", n: "OpenAI-compatible", kind: "OpenAI-compatible", model: "your-model", m: "{}", h: 300, c: 0.08, local: true, base: "http://localhost:8000/v1" },
];
const seed = s => { let x = s; return () => (x = (x * 9301 + 49297) % 233280) / 233280; };
const runsOf = (s, lvl) => { const r = seed(s); return Array.from({ length: 14 }, (_, i) => lvl ? Math.round(r() * lvl * (0.4 + i / 20)) : 0); };
const AGENTS0 = [
  { id: "billing", n: "Billing exceptions", trig: "Chat", ic: "receipt", h: 75, d: "Works blocked billing queue items and proposes what to do about them.", tools: 56, tiers: [38, 16, 2], pend: 3, open: 1, last: "12 min ago", runs: runsOf(3, 9), rec: { ap: 41, ch: 6, rj: 2, fl: 1, streak: 7, tool: "Release billing hold" }, can: "Report analyst, Receivables" },
  { id: "compliance", n: "Compliance desk", trig: "Chat", ic: "shield", h: 160, d: "Answers questions about driver qualification and what is expiring.", tools: 55, tiers: [49, 6, 0], last: "1 h ago", runs: runsOf(7, 4) },
  { id: "dispatch", n: "Dispatch desk", trig: "Chat", ic: "truck", h: 258, d: "Looks up shipments and drivers for the dispatch team.", tools: 56, tiers: [40, 14, 2], pend: 4, open: 4, last: "3 min ago", runs: runsOf(11, 14), rec: { ap: 88, ch: 11, rj: 4, fl: 0, streak: 9, tool: "Assign driver" }, can: "Dispatch coverage agent" },
  { id: "fuel", n: "Fuel and IFTA clerk", trig: "Chat", ic: "fuel", h: 85, d: "Keeps the fuel tax record: fuel purchases, card statements, state miles, and the quarter's IFTA return for a person to file.", tools: 33, tiers: [21, 12, 0], last: "Yesterday", runs: runsOf(5, 2) },
  { id: "help", n: "Help", trig: "Chat", ic: "help", h: 0, d: "Explains how to do things in Trenova. Cannot read or change records.", tools: 0, tiers: [0, 0, 0], last: "8 min ago", runs: runsOf(13, 6) },
  { id: "mds", n: "Master data steward", trig: "Chat", ic: "database", h: 0, d: "Keeps carriers, customers, commodities, hazmat, locations and equipment right; files scanned paperwork and clears the attention feed.", tools: 55, tiers: [33, 22, 0], last: null, runs: runsOf(1, 0) },
  { id: "recv", n: "Receivables", trig: "Chat", ic: "dollar", h: 200, d: "Works what customers owe once an invoice is out: payments, credit, disputes, late charges, and who to chase first.", tools: 27, tiers: [17, 10, 0], last: "2 h ago", runs: runsOf(17, 3) },
  { id: "report", n: "Report analyst", trig: "Chat", ic: "chart", h: 285, d: "Builds, runs and explains reports and dashboards, and proposes scheduled emails for a person to approve.", tools: 23, tiers: [19, 4, 0], last: "Yesterday", runs: runsOf(19, 2), rec: { ap: 12, ch: 3, rj: 0, fl: 0, streak: 4, tool: "Schedule report" } },
  { id: "settle", n: "Settlements clerk", trig: "Chat", ic: "wallet", h: 145, d: "Runs driver and carrier settlements: drafts the period's pay, matches carrier invoices, and puts every payment in front of a person.", tools: 56, tiers: [36, 20, 0], last: "Mon", runs: runsOf(23, 2) },
  { id: "workforce", n: "Workforce coordinator", trig: "Chat", ic: "users", h: 310, d: "Handles time off, leave, injuries, reviews, safety records, random testing and permits, and proposes each change.", tools: 55, tiers: [30, 25, 0], last: null, runs: runsOf(2, 0), access: "3 roles" },
  { id: "cs", n: "Customer service", trig: "Chat", ic: "headset", h: 0, d: "Shipment status for the customer-facing team. Off until reviewed.", tools: 53, tiers: [50, 3, 0], off: true, last: null, runs: runsOf(4, 0) },
  { id: "digest", n: "Morning operations digest", trig: "Scheduled", ic: "sun", h: 60, d: "A weekday summary of what is stuck, late or unassigned, ready before the desk opens.", tools: 5, tiers: [5, 0, 0], cron: "0 6 * * 1-5", tz: "America/Los_Angeles", next: "Thu, Oct 8 · 6:00 AM", last: "Today 6:00 AM", runs: [1, 1, 1, 1, 1, 0, 0, 1, 1, 1, 1, 1, 0, 1] },
  { id: "billev", n: "Billing exception agent", trig: "Event", ic: "receipt", h: 75, d: "Diagnoses blocked billing items as they appear and proposes how to clear them.", tools: 14, tiers: [10, 4, 0], shadow: true, sys: true, events: ["Billing hold placed", "Invoice rejected", "Rate mismatch"], last: "40 min ago", runs: runsOf(29, 5) },
  { id: "coverage", n: "Dispatch coverage agent", trig: "Event", ic: "route", h: 258, d: "Reviews uncovered moves and proposes driver assignments.", tools: 20, tiers: [14, 6, 0], shadow: true, sys: true, events: ["Shipment uncovered", "Driver unassigned"], last: "6 min ago", runs: runsOf(31, 8) },
  { id: "inbox", n: "Inbox desk", trig: "Event", ic: "inbox", h: 230, d: "Works the inbox: files what arrives against the right load, attaches paperwork, answers status questions and puts tenders in front of a person.", tools: 25, tiers: [15, 10, 0], shadow: true, events: ["Email received", "EDI 204 received"], last: "1 min ago", runs: runsOf(37, 11) },
].map(a => ({ pend: 0, open: 0, access: "Everyone", ...a, on: !a.off }));
const TIERS = ["Read", "Propose", "Act"];
const PROPOSALS = {
  billing: [{ k: "Release hold", t: "INV-20931 · POD is now attached", ago: "12m" }, { k: "Re-rate", t: "SHP-48211 to the contract lane rate", ago: "31m" }, { k: "Write off", t: "$42.10 short-pay from Acme Foods", ago: "1h" }],
  dispatch: [{ k: "Assign", t: "Marcus Reed to SHP-48302 · 14 mi out", ago: "3m" }, { k: "Tender", t: "SHP-48315 to Ridgeline Freight at $2.41/mi", ago: "9m" }, { k: "Reschedule", t: "SHP-48190 pickup to 14:00", ago: "22m" }, { k: "Swap trailer", t: "SHP-48277 · reefer unit 5531", ago: "40m" }],
};
const WEEK = [["Thu", 168, 0], ["Fri", 191, 1], ["Sat", 74, 0], ["Sun", 61, 0], ["Mon", 244, 2], ["Tue", 268, 18], ["Wed", 249, 8]];
const FEATURES = [["Agents and assistant", 612, 24, "2.2M", 1420], ["Scope classification", 273, 5, "0.2M", 210], ["Document classification", 204, 0, "0.3M", 380], ["Operational insights", 188, 0, "0.4M", 920], ["Daily briefing", 7, 0, "0.04M", 2610]];
const FAILURES = [
  { p: "vllm", err: "Could not connect", n: 24, span: "Oct 6, 10:33 PM – 5:21 AM", detail: 'execute provider request: Post "http://localhost:8000/v1/chat/completions": dial tcp 127.0.0.1:8000: connect: connection refused' },
  { p: "ollama", err: "Could not connect", n: 3, span: "Oct 7, 5:21 AM", detail: 'execute provider request: Post "http://localhost:11434/api/chat": dial tcp 127.0.0.1:11434: connect: connection refused' },
  { p: "ollama", err: "Timed out", n: 2, span: "Oct 6, 2:14 PM", detail: "no response within 60s · scope classification · prompt 3.1k tokens" },
];
const POLICIES = [
  { k: "earned", t: "Earned autonomy", ic: "award", s: "A tool moves up a tier after a run of clean approvals — never past the agent's ceiling.", more: "When a tool's proposals are approved unchanged this many times in a row, it moves up one tier on that agent. A rejection or a failed run takes an earned tier back. Each change is audited and announced.", opt: { k: "threshold", l: "Clean approvals in a row", v: [5, 10, 25, 50] } },
  { k: "learn", t: "Learn from their work", ic: "brain", s: "When a conversation settles, the agent keeps the lesson as memory.", more: "If something went wrong, took several tries or a person corrected it, the agent keeps a preference, a fact or the steps that worked. Lessons shared beyond one person wait for approval, and anything drawn from outside content is only ever offered. Each agent also has its own switch." },
  { k: "allowance", t: "Monthly allowance per person", ic: "gauge", s: "How many questions each person may ask the agents in a calendar month.", more: "Desk warns people as they get close and says when it refreshes. Each agent's own budget and daily limit still apply.", opt: { k: "allowance", l: "Questions per person", v: [0, 100, 250, 500, 1000] }, noSwitch: true },
  { k: "share", t: "Share corrections for model training", ic: "database", s: "Anonymized document corrections help improve extraction for every customer.", more: "Trenova keeps what was read from a document beside what a person confirmed, so accuracy can be measured. Turning this on lets those corrections be anonymized and used for training. Turning it off keeps them out of any training after the change.", foot: "No training export has included this organization's corrections." },
];
function routeOf(t, ps) {
  const as = ps.filter(p => p.tasks.includes(t.k));
  const ok = as.filter(p => p.on && (!t.trust || p.trusted));
  let why = "No provider assigned", fix = null;
  if (as.length && !ok.length) {
    const off = as.find(p => !p.on);
    if (off) { why = off.n + (off.key || off.local ? " is off" : " needs a key"); fix = { p: off, l: off.key || off.local ? "Turn on" : "Add key" }; }
    else { why = "Needs a trusted provider"; fix = { p: as[0], l: "Trust " + as[0].n }; }
  }
  return { first: ok[0] || null, next: ok[1] || null, as, why, fix };
}
const PROVIDERS_MORE = [
  { id: "openai", n: "OpenAI", kind: "OpenAI Responses", model: "gpt-5.1-mini", m: "O", h: 165, c: 0.06, on: true, key: true, trusted: true, priv: false, base: "", tasks: ["DocumentClassification", "ScopeClassification"], last: { ok: true, msg: "Connected · 290 ms", when: "Today 8:02 AM" }, wk: { calls: 1840, f: 2, ms: 310, tok: "1.4M" } },
  { id: "groq", n: "Groq", kind: "OpenAI-compatible", model: "llama-3.3-70b-versatile", m: "Gq", h: 25, c: 0.14, on: true, key: true, trusted: false, priv: false, base: "https://api.groq.com/openai/v1", tasks: ["OperationalInsights", "General"], last: { ok: true, msg: "Connected · 120 ms", when: "Today 7:40 AM" }, wk: { calls: 960, f: 0, ms: 140, tok: "0.8M" } },
  { id: "azure", n: "Azure OpenAI", kind: "Azure OpenAI", model: "gpt-5.1", m: "Az", h: 235, c: 0.1, on: true, key: true, trusted: true, priv: false, base: "https://trenova.openai.azure.com", tasks: ["DocumentExtraction", "BillingDiagnosis"], last: { ok: true, msg: "Connected · 520 ms", when: "Yesterday" }, wk: { calls: 412, f: 1, ms: 980, tok: "1.1M" } },
  { id: "gemini", n: "Google Gemini", kind: "Gemini", model: "gemini-3-flash", m: "G", h: 265, c: 0.12, on: false, key: true, trusted: false, priv: false, base: "", tasks: ["DailyBriefing"], last: { ok: true, msg: "Connected · 340 ms", when: "Oct 3" }, wk: null },
  { id: "mistral", n: "Mistral", kind: "Mistral", model: "mistral-large-3", m: "Mi", h: 40, c: 0.15, on: false, key: false, trusted: false, priv: false, base: "", tasks: [], last: null, wk: null },
  { id: "bedrock", n: "AWS Bedrock", kind: "Bedrock", model: "anthropic.claude-sonnet-5", m: "Br", h: 60, c: 0.13, on: false, key: true, trusted: true, priv: false, base: "us-east-1", tasks: ["AssistantChat"], last: { ok: false, msg: "Access denied", when: "Oct 2" }, wk: null },
];
Object.assign(window, { PROVIDERS_MORE, MOD, TASKS, TASK_L, PROVIDERS0, PRESETS, AGENTS0, TIERS, PROPOSALS, WEEK, FEATURES, FAILURES, POLICIES, routeOf });
