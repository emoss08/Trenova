const anH = s => { let h = 2166136261; for (const ch of s) h = Math.imul(h ^ ch.charCodeAt(0), 16777619); return ((h >>> 0) % 10000) / 10000; };
const AN_PROV = Object.fromEntries([...PROVIDERS0, ...PROVIDERS_MORE].map(p => [p.id, p]));
const AN_MODEL = { "gpt-5.1": "GPT-5.1", "gpt-5.1-mini": "GPT-5.1 mini", "llama-3.3-70b": "Llama 3.3 70B", "qwen2.5:14b": "Qwen 2.5 14B", "claude-sonnet-5": "Claude Sonnet 5", "gemini-3-flash": "Gemini 3 Flash" };
const AN_OUT = [["c", "Completed"], ["w", "Handed to a person"], ["s", "Stopped at a limit"], ["f", "Failed"]];
const AN_WHY = [["args", "Wrong arguments", "m"], ["unk", "Called a tool it doesn't hold", "m"], ["loop", "Repeated the same call", "m"], ["err", "Tool returned an error", "s"], ["deny", "Blocked by a tool rule", "s"], ["time", "Tool timed out", "s"]];
const AN_PERR = [["rate", "Rate limited"], ["time", "Timed out"], ["conn", "Couldn't connect"], ["auth", "Access denied"], ["srv", "Server error"]];
const AN_TOOLS = [
  ["get_shipment", "Get shipment", 16, { args: 0.4, loop: 2 }],
  ["search_drivers", "Search drivers", 10, { args: 0.9 }],
  ["get_hos_status", "Check hours of service", 8, { args: 0.7, err: 1.6 }],
  ["assign_driver", "Assign driver", 5, { args: 2.6, deny: 2.2, unk: 0.6 }],
  ["release_billing_hold", "Release billing hold", 3, { args: 1.2, deny: 4.5 }],
  ["lookup_rate", "Look up rate", 7, { args: 1.4, err: 1.3 }],
  ["extract_document_fields", "Extract document fields", 9, { args: 1.3, time: 3.2 }],
  ["run_report", "Run report", 4, { args: 2.1, time: 2.6, loop: 1.4 }],
  ["update_load_status", "Update load status", 6, { args: 1.1, deny: 1.6 }],
  ["create_invoice_note", "Add invoice note", 4, { args: 0.8, unk: 1.8 }],
];
// 30-day baselines per model-on-provider. out = [completed, handed to a person, stopped, failed]. dur = p25/p50/p75/p95 seconds.
const AN_DEPS = [
  { id: "az", pid: "azure", model: "gpt-5.1", runs: 1240, out: [0.93, 0.035, 0.01, 0.025], dur: { c: [6, 11, 19, 38], w: [5, 9, 15, 28], s: [40, 62, 88, 120], f: [9, 21, 44, 90] }, tc: 7.4, tf: { args: 0.004, unk: 0.0005, loop: 0.001, err: 0.006, deny: 0.004, time: 0.002 }, prop: [0.78, 0.15, 0.07], cpr: 0.041, ttft: 820, rated: 0.9, perr: { rate: 3, time: 2, srv: 4 }, fb: 6, tasks: { DocumentExtraction: 0.95, BillingDiagnosis: 0.93 } },
  { id: "oai", pid: "openai", model: "gpt-5.1-mini", runs: 3100, out: [0.91, 0.045, 0.015, 0.03], dur: { c: [2, 4, 7, 14], w: [2, 4, 6, 11], s: [18, 30, 44, 70], f: [3, 8, 16, 35] }, tc: 3.1, tf: { args: 0.009, unk: 0.001, loop: 0.003, err: 0.006, deny: 0.003, time: 0.002 }, prop: [0.7, 0.18, 0.12], cpr: 0.006, ttft: 310, rated: 0.84, perr: { rate: 12, time: 4, srv: 6 }, fb: 9, tasks: { DocumentClassification: 0.96, ScopeClassification: 0.97 }, tools: [0, 1, 6, 9] },
  { id: "gq", pid: "groq", model: "llama-3.3-70b", runs: 2050, out: [0.86, 0.05, 0.02, 0.07], dur: { c: [1.5, 3, 6, 12], w: [1.5, 3, 5, 9], s: [14, 22, 34, 55], f: [2, 5, 11, 24] }, tc: 4.2, tf: { args: 0.031, unk: 0.006, loop: 0.008, err: 0.006, deny: 0.004, time: 0.001 }, prop: [0.61, 0.2, 0.19], cpr: 0.004, ttft: 140, rated: 0.79, perr: { rate: 41, time: 1, srv: 3 }, fb: 22, tasks: { OperationalInsights: 0.88, General: 0.85 } },
  { id: "wv", pid: "vllm", model: "llama-3.3-70b", runs: 1420, out: [0.74, 0.05, 0.02, 0.19], dur: { c: [5, 12, 24, 52], w: [4, 10, 19, 40], s: [48, 70, 96, 140], f: [31, 58, 61, 118] }, tc: 4.0, tf: { args: 0.029, unk: 0.005, loop: 0.009, err: 0.006, deny: 0.004, time: 0.038 }, prop: [0.6, 0.21, 0.19], cpr: 0, ttft: 1420, rated: 0.78, perr: { conn: 212, time: 96, srv: 11 }, fb: 241, tasks: { AssistantChat: 0.72, DocumentExtraction: 0.7, FormulaAssistant: 0.81, OperationalInsights: 0.77, General: 0.74 } },
  { id: "ol", pid: "ollama", model: "qwen2.5:14b", runs: 1580, out: [0.82, 0.04, 0.03, 0.11], dur: { c: [3, 6, 11, 22], w: [3, 5, 9, 17], s: [26, 40, 58, 90], f: [4, 9, 18, 40] }, tc: 2.6, tf: { args: 0.052, unk: 0.021, loop: 0.017, err: 0.005, deny: 0.006, time: 0.004 }, prop: [0.52, 0.22, 0.26], cpr: 0, ttft: 410, rated: 0.71, perr: { conn: 3, time: 8 }, fb: 7, tasks: { ScopeClassification: 0.93, DocumentClassification: 0.9, OperationalInsights: 0.74, DailyBriefing: 0.81, General: 0.7 } },
  { id: "br", pid: "bedrock", model: "claude-sonnet-5", runs: 610, until: 7, out: [0.95, 0.025, 0.005, 0.02], dur: { c: [7, 13, 22, 41], w: [6, 11, 18, 30], s: [44, 66, 90, 126], f: [8, 18, 36, 70] }, tc: 8.1, tf: { args: 0.002, unk: 0, loop: 0.0005, err: 0.006, deny: 0.004, time: 0.002 }, prop: [0.84, 0.12, 0.04], cpr: 0.052, ttft: 690, rated: 0.93, perr: { auth: 318, rate: 2 }, fb: 318, tasks: { AssistantChat: 0.96, BillingDiagnosis: 0.95 } },
  { id: "gm", pid: "gemini", model: "gemini-3-flash", runs: 380, until: 6, out: [0.9, 0.04, 0.02, 0.04], dur: { c: [2, 4, 8, 15], w: [2, 4, 7, 12], s: [16, 26, 38, 60], f: [3, 7, 14, 30] }, tc: 3.4, tf: { args: 0.012, unk: 0.002, loop: 0.003, err: 0.006, deny: 0.003, time: 0.002 }, prop: [0.69, 0.19, 0.12], cpr: 0.005, ttft: 340, rated: 0.82, perr: { rate: 5 }, fb: 2, tasks: { DailyBriefing: 0.92 }, tools: [0, 7, 8] },
];
const AN_SAMPLE = {
  args: (t) => [t, `"driver_id" must be a string — got 1042`],
  unk: () => ["update_rate_table", "Not one of this agent's tools"],
  loop: (t) => [t, "Same arguments 4 times in a row · stopped by the loop guard"],
  err: (t) => [t, "SHP-48190 is locked by another edit"],
  deny: (t) => [t, "Asks first for this agent · proposal created instead"],
  time: (t) => [t, "No result after 30 s"],
};
function anDep(d, rg, ag, tk) {
  const days = rg === "d1" ? 1 : rg === "d7" ? 7 : 30;
  let f = Math.max(0, days - (d.until || 0)) / 30;
  let j = 1;
  if (ag !== "all") { f *= 0.04 + anH(d.id + ag) * 0.22; j = 0.93 + anH(ag + d.id) * 0.1; }
  let cR = d.out[0];
  if (tk !== "all") { if (!(tk in d.tasks)) return null; f /= Object.keys(d.tasks).length; cR = d.tasks[tk]; }
  cR = Math.min(0.99, cR * j);
  const runs = Math.round(d.runs * f); if (!runs) return null;
  const sc = (1 - cR) / (1 - d.out[0]);
  const w = Math.round(runs * d.out[1] * sc), s = Math.round(runs * d.out[2] * sc), fl = Math.round(runs * d.out[3] * sc);
  const out = { c: runs - w - s - fl, w, s, f: fl };
  const dur = {}; for (const [o] of AN_OUT) dur[o] = d.dur[o].map(v => v * (0.94 + anH(d.id + rg + o + ag) * 0.12));
  const calls = Math.round(runs * d.tc);
  const used = d.tools || AN_TOOLS.map((_, i) => i), wsum = used.reduce((t, i) => t + AN_TOOLS[i][2], 0);
  const tools = AN_TOOLS.map((tl, i) => { if (!used.includes(i)) return null; const n = Math.round(calls * tl[2] / wsum); const fx = {}; for (const [r] of AN_WHY) fx[r] = Math.round(n * d.tf[r] * (tl[3][r] || 1) * (0.6 + anH(d.id + i + r) * 0.8) / j); return { n, f: fx }; });
  const tf = {}; for (const [r] of AN_WHY) tf[r] = tools.reduce((t, x) => t + (x ? x.f[r] : 0), 0);
  const np = Math.round(runs * 0.35), pa = Math.round(np * Math.min(0.97, d.prop[0] * j)), pr = Math.round(np * d.prop[2] / j);
  const perr = {}; for (const [k] of AN_PERR) perr[k] = Math.round((d.perr[k] || 0) * f * (0.85 + anH(d.id + k + rg) * 0.3));
  const tasks = {}; for (const [k, r] of Object.entries(d.tasks)) { if (tk !== "all" && k !== tk) continue; const n = tk !== "all" ? runs : Math.round(runs / Object.keys(d.tasks).length); tasks[k] = { n, c: n * Math.min(0.99, r * j) }; }
  return { runs, out, dur, calls, tools, tf, prop: { a: pa, r: pr, m: Math.max(0, np - pa - pr) }, cost: runs * d.cpr, local: d.cpr === 0, mc: Math.round(runs * (d.tc * 0.7 + 1.4)), perr, fb: Math.round(d.fb * f), ttft: d.ttft * (0.95 + anH(d.id + rg) * 0.1), rated: Math.min(0.98, d.rated * j), tasks };
}
function anAgg(ms) {
  const S = k => ms.reduce((t, m) => t + m[k], 0), runs = S("runs");
  const out = {}, dur = {}; for (const [o] of AN_OUT) { out[o] = ms.reduce((t, m) => t + m.out[o], 0); dur[o] = [0, 1, 2, 3].map(i => out[o] ? ms.reduce((t, m) => t + m.dur[o][i] * m.out[o], 0) / out[o] : 0); }
  const tools = AN_TOOLS.map((_, i) => { const xs = ms.map(m => m.tools[i]).filter(Boolean); if (!xs.length) return null; const f = {}; for (const [r] of AN_WHY) f[r] = xs.reduce((t, x) => t + x.f[r], 0); return { n: xs.reduce((t, x) => t + x.n, 0), f }; });
  const tf = {}, perr = {}, prop = {}; for (const [r] of AN_WHY) tf[r] = ms.reduce((t, m) => t + m.tf[r], 0); for (const [k] of AN_PERR) perr[k] = ms.reduce((t, m) => t + m.perr[k], 0); for (const k of ["a", "m", "r"]) prop[k] = ms.reduce((t, m) => t + m.prop[k], 0);
  const tasks = {}; ms.forEach(m => Object.entries(m.tasks).forEach(([k, v]) => { tasks[k] = tasks[k] || { n: 0, c: 0 }; tasks[k].n += v.n; tasks[k].c += v.c; }));
  const mc = S("mc");
  return { runs, out, dur, tools, tf, perr, prop, tasks, calls: S("calls"), cost: S("cost"), local: ms.every(m => m.local), mc, fb: S("fb"), ttft: ms.reduce((t, m) => t + m.ttft * m.mc, 0) / (mc || 1), rated: ms.reduce((t, m) => t + m.rated * m.runs, 0) / (runs || 1) };
}
const anTF = m => Object.values(m.tf).reduce((t, v) => t + v, 0);
const anTFm = m => AN_WHY.filter(w => w[2] === "m").reduce((t, [r]) => t + m.tf[r], 0);
const anPE = m => Object.values(m.perr).reduce((t, v) => t + v, 0);
function anGroups(rg, ag, tk, by) {
  const deps = AN_DEPS.map(d => ({ d, m: anDep(d, rg, ag, tk) })).filter(x => x.m);
  const keyOf = x => by === "model" ? x.d.model : x.d.pid;
  const keys = [...new Set(deps.map(keyOf))];
  const kid = x => ({ key: x.d.id, label: AN_PROV[x.d.pid].n, short: AN_SHORT.p[x.d.pid], col: anColor(AN_HUE.p[x.d.pid]), model: x.d.model, sub: AN_MODEL[x.d.model], provs: [AN_PROV[x.d.pid]], pids: [x.d.pid], deps: [x.d], m: x.m, until: x.d.until });
  const groups = keys.map(k => { const xs = deps.filter(x => keyOf(x) === k), d0 = xs[0].d; const provs = xs.map(x => AN_PROV[x.d.pid]);
    return { key: k, short: by === "model" ? AN_SHORT.m[d0.model] : AN_SHORT.p[d0.pid], col: anColor(by === "model" ? AN_HUE.m[d0.model] : AN_HUE.p[d0.pid]), label: by === "model" ? AN_MODEL[d0.model] : AN_PROV[d0.pid].n, sub: by === "model" ? provs.map(p => p.n).join(" · ") : AN_MODEL[d0.model], provs, pids: xs.map(x => x.d.pid), deps: xs.map(x => x.d), m: anAgg(xs.map(x => x.m)), until: xs.every(x => x.d.until) ? Math.min(...xs.map(x => x.d.until)) : 0, kids: xs.length > 1 ? xs.map(kid) : null }; });
  groups.sort((a, b) => b.m.runs - a.m.runs);
  return { groups, deps: deps.map(kid).sort((a, b) => b.m.runs - a.m.runs), all: deps.length ? anAgg(deps.map(x => x.m)) : null };
}
const AN_HUE = { m: { "gpt-5.1": 258, "gpt-5.1-mini": 200, "llama-3.3-70b": 28, "qwen2.5:14b": 80, "claude-sonnet-5": 300, "gemini-3-flash": 150 }, p: { azure: 258, openai: 200, groq: 28, vllm: 350, ollama: 80, bedrock: 300, gemini: 150 } };
const AN_SHORT = { m: { "gpt-5.1": "GPT-5.1", "gpt-5.1-mini": "5.1 mini", "llama-3.3-70b": "Llama 70B", "qwen2.5:14b": "Qwen 14B", "claude-sonnet-5": "Sonnet 5", "gemini-3-flash": "Gem. Flash" }, p: { azure: "Azure", openai: "OpenAI", groq: "Groq", vllm: "vLLM", ollama: "Ollama", bedrock: "Bedrock", gemini: "Gemini" } };
const anColor = h => `oklch(0.72 0.15 ${h})`;
const AN_EVENTS = [{ ago: 12.4, pid: "vllm", l: "vLLM restarted" }, { ago: 7, pid: "bedrock", l: "Bedrock lost access" }, { ago: 6, pid: "gemini", l: "Gemini turned off" }, { ago: 0.45, pid: "vllm", l: "vLLM stopped answering" }];
const AN_BUCKETS = { d1: [24, 1 / 24], d7: [28, 0.25], d30: [30, 1] };
function anDepSeries(d, rg, ag) {
  const [n, span] = AN_BUCKETS[rg], j = ag !== "all" ? 0.93 + anH(ag + d.id) * 0.1 : 1, tf0 = Object.values(d.tf).reduce((a, b) => a + b, 0);
  return Array.from({ length: n }, (_, i) => {
    const ago = (n - 1 - i) * span; if (ago < (d.until || 0) - 0.01) return null;
    const nz = k => anH(d.id + rg + k + i) - 0.5;
    let c = d.out[0] * j * (1 + nz("c") * 0.05), tf = tf0 * (1 + nz("t") * 0.3), med = d.dur.c[1] * (1 + nz("m") * 0.14), w = 1;
    if (d.id === "wv" && ago < 0.9) { c *= 0.55; tf *= 2.6; med *= 2.1; }
    if (d.id === "wv" && ago > 12.4 && ago < 14) { c *= 0.7; tf *= 1.8; med *= 1.6; }
    if (d.id === "gq" && ago > 17 && ago < 19) { c *= 0.9; tf *= 1.4; }
    if (d.id === "ol") c *= 0.96 + i / n * 0.06;
    return { c: Math.min(0.995, c), tf, med, w: d.runs * w };
  });
}
function anSeries(g, rg, ag) {
  const ss = g.deps.map(d => anDepSeries(d, rg, ag)), n = ss[0].length;
  return Array.from({ length: n }, (_, i) => { const xs = ss.map(s => s[i]).filter(Boolean); if (!xs.length) return null; const W = xs.reduce((t, x) => t + x.w, 0); return { c: xs.reduce((t, x) => t + x.c * x.w, 0) / W, tf: xs.reduce((t, x) => t + x.tf * x.w, 0) / W, med: xs.reduce((t, x) => t + x.med * x.w, 0) / W }; });
}
const anMs = s => s < 1 ? Math.round(s * 1000) + "ms" : s.toFixed(2) + "s";
const anNP = m => m.prop.a + m.prop.m + m.prop.r;
const AN_MET = {
  c: { l: "Completed", d: "Runs that finished without handing off or failing", v: m => m.out.c / m.runs, f: x => anP(x), hi: true, pct: true },
  med: { l: "Median run time", d: "Completed runs, from the first token to the last tool call", v: m => m.dur.c[1], f: x => anS(x), hi: false },
  tf: { l: "Tool calls that failed", d: "Out of every tool call the model made", v: m => anTF(m) / (m.calls || 1), f: x => anP(x, 1), hi: false, pct: true },
  cost: { l: "Cost per completed run", d: "What the provider charged, divided by runs that finished", v: m => m.local ? 0 : m.cost / (m.out.c || 1), f: x => x === 0 ? "Local" : "$" + x.toFixed(3), hi: false },
  ap: { l: "Approved as proposed", d: "Proposals a person accepted without changing", v: m => anNP(m) ? m.prop.a / anNP(m) : null, f: x => anP(x), hi: true, pct: true },
  ttft: { l: "Time to first token", d: "How long before the model starts answering", v: m => m.ttft / 1000, f: x => anMs(x), hi: false },
  rt: { l: "Rated helpful", d: "Thumbs up from the people who rated an answer", v: m => m.rated, f: x => anP(x), hi: true, pct: true },
};
const anS = s => s >= 60 ? Math.floor(s / 60) + "m " + String(Math.round(s % 60)).padStart(2, "0") + "s" : s < 10 ? s.toFixed(1) + "s" : Math.round(s) + "s";
const anP = (x, dp = 0) => (x * 100).toFixed(dp) + "%";
const anCol = p => `oklch(0.68 ${p.c ? 0.14 : 0} ${p.h})`;
const anUntil = u => u ? "last ran " + ["", "", "", "", "", "", "Oct 3", "Oct 2"][u] : null;
Object.assign(window, { AN_SHORT, AN_MET, AN_EVENTS, AN_BUCKETS, anSeries, anMs, anNP, AN_PROV, AN_MODEL, AN_OUT, AN_WHY, AN_PERR, AN_TOOLS, AN_DEPS, AN_SAMPLE, anGroups, anTF, anTFm, anPE, anS, anP, anCol, anUntil, anH });
