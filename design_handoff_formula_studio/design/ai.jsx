const { useState: aS, useRef: aR, useEffect: aE, useMemo: aM } = React;

const NOVA_SYSTEM = `You are Nova, the formula assistant inside Trenova, a transportation management system. Organizations write rating methods ("formula templates") in expr-lang that turn shipment facts into a charge in dollars.
Syntax: arithmetic + - * / % **, comparisons, and/or/not, ternary a ? b : c, nil, x ?? fallback for empty values, statements "let name = expr;" followed by one final expression that is the charge. No assignment besides let.
Functions: ${FS_FUNCS.map((f) => f.signature).join(", ")}.
Shipment variables: ${FS_VARS.map((v) => `${v.name} (${v.type}${v.nullable ? ", can be empty" : ""})`).join(", ")}.
A template may declare its own inputs (Number, String or Boolean with a default). Minimum, maximum and rounding are applied after the formula by the charge policy, so never round inside the formula unless asked.
Always answer with ONLY a JSON object, no prose around it:
{"reply": string, under 90 words, plain sentences, wrap identifiers in backticks,
 "proposal": null or {"name": string, "description": string, "type": "FreightCharge" or "AccessorialCharge", "expression": string, "variables": [{"name": string, "type": "Number"|"String"|"Boolean", "defaultValue": number|string|boolean, "description": string}], "breakdown": [{"name": camelCase string, "label": string, "expression": string}], "explanation": string under 50 words},
 "scenarios": null or [{"name": string, "inputs": {variableName: value}}]}
Only include a proposal when the person asks you to write or change the formula; include every input the expression needs that is not a shipment variable. Only include scenarios when asked for tests or edge cases (3 to 5, inputs only, never expected amounts).`;

function novaJson(text) {
  const a = text.indexOf("{"), b = text.lastIndexOf("}");
  if (a < 0 || b < a) throw new Error("No JSON in reply");
  return JSON.parse(text.slice(a, b + 1));
}
async function novaCall(prompt, system = NOVA_SYSTEM, max = 1400) {
  if (!window.claude || !window.claude.complete) throw new Error("offline");
  const text = await window.claude.complete({ system, max_tokens: max, messages: [{ role: "user", content: prompt }] });
  return novaJson(text);
}
function novaDraftBrief(d) {
  return `Template "${d.name || "untitled"}" (${d.type}). Description: ${d.description || "none"}.
Expression:
${d.expression || "(empty)"}
Inputs: ${d.variables.filter((v) => v.name).map((v) => `${v.name}:${v.type}=${v.defaultValue}`).join(", ") || "none"}.
Breakdown lines: ${d.breakdown.map((b) => `${b.name}=${b.expression}`).join("; ") || "none"}.
Charge policy: minimum ${d.policy.min ?? "none"}, maximum ${d.policy.max ?? "none"}, rounding ${d.policy.mode} to ${d.policy.precision} places.`;
}

// Offline answers, built from the standard library and the engine, so the studio still helps without a model.
function novaFallback(q, d, run) {
  const t = q.toLowerCase();
  const prop = (base, extra = {}) => ({ name: base.name, description: base.description, type: base.type, expression: base.expression, variables: base.variables, breakdown: [], explanation: base.description, ...extra });
  if (/edge|scenario|test/.test(t)) return { reply: "Here are loads that sit on this formula's edges. Expected amounts come from the engine, from the formula as it is in the editor.", scenarios: [
    { name: "Local move, 0 miles", inputs: { totalDistance: 0, totalStops: 2 } },
    { name: "Five stops with hazmat", inputs: { totalStops: 5, hasHazmat: true } },
    { name: "Long haul, 1,400 mi", inputs: { totalDistance: 1400, totalStops: 2 } },
    { name: "Single-stop drop", inputs: { totalStops: 1 } },
  ] };
  if (/weekend/.test(t)) return { reply: "I added a weekend fee as its own input so lanes can override it. It applies when the first pickup is on a Saturday or Sunday.", proposal: { ...novaExtend(d, "weekendFee", 125, "Fee when the first pickup is on a weekend", "(isWeekendPickup ? weekendFee : 0)"), breakdownAdd: { name: "weekend", label: "Weekend pickup", expression: "isWeekendPickup ? weekendFee : 0" }, explanation: "Adds weekendFee when isWeekendPickup is true." } };
  if (/reefer|temperature/.test(t)) return { reply: "I added a reefer fee that applies only when a commodity needs temperature control.", proposal: { ...novaExtend(d, "reeferFee", 225, "Temperature control fee", "(requiresTemperatureControl ? reeferFee : 0)"), breakdownAdd: { name: "reefer", label: "Temperature control", expression: "requiresTemperatureControl ? reeferFee : 0" }, explanation: "Adds reeferFee when requiresTemperatureControl is true." } };
  if (/minimum|250|floor|why/.test(t) && run && run.policy) return { reply: run.policy.clamp ? `The formula produced ${fsMoney(run.policy.raw)} on this load, under the ${fsMoney(run.policy.clamp.limit)} ${run.policy.clamp.bound} in Charge policy, so the charge was raised to it. Breakdown lines still add up to the raw amount.` : `On this load the formula produced ${fsMoney(run.policy.raw)}, inside the charge policy, so it billed ${fsMoney(run.policy.amount)} after rounding.` };
  if (/explain|what does|how does/.test(t)) return { reply: novaLocalExplain(d).map((x) => x.says).join(" ") };
  if (d.expression.trim() && !/write|draft|create|build|price|charge|per /.test(t)) return { reply: "I couldn't reach the model just now. Try again, or ask me to explain the formula, suggest edge cases, or add a fee." };
  const pick = /cwt|hundredweight|weight|pound/.test(t) ? "Per CWT" : /stop/.test(t) && !/mile/.test(t) ? "Additional Stop Fee" : /hour/.test(t) ? "Per Hour" : /hazmat/.test(t) && !/mile/.test(t) ? "Hazmat Surcharge" : /fuel/.test(t) ? "Per Mile with Fuel Surcharge" : /minimum/.test(t) ? "Per Mile with Minimum" : "Per Mile";
  const base = FS_STANDARDS.find((s) => s.name === pick);
  return { reply: `This draft starts from the standard “${base.name}” method; adjust the inputs to match your lanes.`, proposal: prop(base) };
}
function novaExtend(d, name, def, desc, term) {
  const has = d.variables.some((v) => v.name === name);
  const lines = d.expression.split("\n");
  lines[lines.length - 1] = lines[lines.length - 1].replace(/\s*$/, "") + " + " + term;
  return { name: d.name, description: d.description, type: d.type, expression: d.expression.trim() ? lines.join("\n") : `baseRate * totalDistance + ${term}`, variables: has ? [] : [{ name, type: "Number", defaultValue: def, description: desc }] };
}

const NOVA_CANNED = {
  "let linehaul = baseRate * totalDistance": "Linehaul is the lane's per-mile rate times the miles on the shipment.",
  "let fuel = linehaul * (fuelSurchargePercent / 100)": "Fuel adds a percentage of linehaul, 18% unless the lane sets its own.",
  "let stops = max(totalStops - includedStops, 0) * stopFee": "Each stop past the two included ones costs $50. max(…, 0) keeps a one- or two-stop load from going negative.",
  "linehaul + fuel + stops + (hasHazmat ? hazmatFee : 0)": "The charge is the three parts plus the $175 hazmat fee when any commodity is hazardous. The minimum and rounding happen after this, in Charge policy.",
};
function novaLocalExplain(d) {
  const an = fsAnalyze(d.expression, fsContext(d.variables));
  return an.stmts.map((st) => {
    const key = st.code.replace(/\s+/g, " ").trim();
    if (NOVA_CANNED[key]) return { s: st.s, e: st.e, code: st.code, says: NOVA_CANNED[key] };
    const words = st.code.replace(/^let\s+(\w+)\s*=\s*/, "").replace(/\*/g, "×").replace(/\//g, "÷").replace(/\?\?/g, "or else").replace(/\s+\?\s+/g, " → if so ").replace(/\s+:\s+/g, ", otherwise ");
    return { s: st.s, e: st.e, code: st.code, says: st.isLet ? `${st.letName} is ${words}.` : `The charge is ${words}.` };
  });
}
async function novaExplain(d) {
  const an = fsAnalyze(d.expression, fsContext(d.variables));
  try {
    const r = await novaCall(`${novaDraftBrief(d)}\n\nExplain this formula to a pricing analyst, one plain sentence per statement, using the real default values. Statements:\n${an.stmts.map((s, i) => `${i}: ${s.code}`).join("\n")}\nAnswer ONLY JSON {"items":[{"i":number,"says":string}]}.`, "You explain freight rating formulas plainly. Answer only JSON.", 900);
    return an.stmts.map((st, i) => ({ s: st.s, e: st.e, code: st.code, says: (r.items.find((x) => x.i === i) || {}).says || novaLocalExplain(d)[i].says }));
  } catch { return novaLocalExplain(d); }
}
async function novaDescribe(d) {
  const missing = d.variables.filter((v) => v.name && !v.description);
  try {
    const r = await novaCall(`${novaDraftBrief(d)}\nWrite a short description (under 12 words) for each of these inputs: ${missing.map((v) => v.name).join(", ")}. Answer ONLY JSON {"descriptions":{name:string}}.`, "You document freight rating formulas. Answer only JSON.", 500);
    return r.descriptions || {};
  } catch {
    return Object.fromEntries(missing.map((v) => [v.name, v.name.replace(/([A-Z])/g, " $1").replace(/^./, (c) => c.toUpperCase()).replace(/ Fee$/, " fee").trim() + (v.name.toLowerCase().includes("fee") ? ", in dollars" : "")]));
  }
}
async function novaSummary(saved, d, bt) {
  const fallback = `Breaks the charge into linehaul, fuel and stop lines and adds a hazmat fee. Against ${bt.n} recent shipments, ${bt.changed} would rate differently (${bt.delta >= 0 ? "+" : "−"}${fsMoney(Math.abs(bt.delta), 0)} in total, ${bt.pct >= 0 ? "+" : ""}${bt.pct.toFixed(1)}%). Every scenario was re-run against this version.`;
  try {
    const r = await novaCall(`Saved version:\n${novaDraftBrief(saved)}\n\nVersion being submitted:\n${novaDraftBrief(d)}\n\nBacktest on ${bt.n} recent shipments: ${bt.changed} change, total delta ${bt.delta.toFixed(2)} dollars (${bt.pct.toFixed(1)}%), ${bt.failed} fail to rate.\nWrite the note a reviewer reads before approving: what changed and what it does to charges, 2 to 3 sentences, no headings. Answer ONLY JSON {"summary":string}.`, "You write concise change notes for pricing reviewers. Answer only JSON.", 400);
    return r.summary || fallback;
  } catch { return fallback; }
}
function novaWhyLocal(dg, sc) {
  const inp = dg.rows.find((r) => r.kind === "input");
  const term = dg.rows.filter((r) => r.kind === "term" && r.before != null && r.after != null).sort((x, y) => Math.abs(y.after - y.before) - Math.abs(x.after - x.before))[0];
  if (dg.error) return { why: `The formula fails on this load now: ${dg.error}`, verdict: "bug" };
  if (inp && !dg.exprChanged) return { why: `${inp.name}'s default changed from ${fsFmt(inp.before)} to ${fsFmt(inp.after)} after this scenario was recorded on ${dg.label}${term ? `, which moves ${term.name} from ${fsMoney(term.before)} to ${fsMoney(term.after)}` : ""}. The formula itself didn't change, so this looks like a rate update rather than a bug.`, verdict: "intended" };
  if (dg.clamp.before && !dg.clamp.after) return { why: `This load used to land on the ${dg.clamp.before.bound}; it no longer does, so it bills the formula's ${fsMoney(dg.raw.after)}.`, verdict: "unclear" };
  if (dg.clamp.after && !dg.clamp.before) return { why: `This load now hits the ${dg.clamp.after.bound} of ${fsMoney(dg.clamp.after.limit)}, so the formula's ${fsMoney(dg.raw.after)} is replaced.`, verdict: "unclear" };
  if (term) return { why: `${term.name} went from ${fsMoney(term.before)} to ${fsMoney(term.after)} on this load${dg.exprChanged ? ", from an edit to the expression" : ""}.`, verdict: dg.exprChanged ? "unclear" : "intended" };
  return { why: `Nothing in the template changed for this load since ${dg.label}; the expected ${fsMoney(sc.expected)} doesn't match what that version billed either, so the expectation itself is likely off.`, verdict: "intended" };
}
async function novaWhy(dg, sc, d) {
  const local = novaWhyLocal(dg, sc);
  try {
    const rows = dg.rows.map((r) => `${r.kind} ${r.name}: ${r.before ?? "none"} -> ${r.after ?? "none"}`).join("\n") || "no differences";
    const r = await novaCall(`A test scenario on the formula template "${d.name}" is failing.
Scenario: "${sc.name}", inputs ${JSON.stringify(sc.inputs)}. Expected ${sc.expected}, recorded on ${dg.label}${dg.recordedOn ? " by " + dg.recordedOn.by : ""}.
The formula engine evaluated both versions on these inputs (these numbers are authoritative; do not recompute):
${rows}
formula raw: ${dg.raw.before} -> ${dg.raw.after}
policy clamp: ${dg.clamp.before ? dg.clamp.before.bound : "none"} -> ${dg.clamp.after ? dg.clamp.after.bound : "none"}
billed: ${dg.billed.before} -> ${dg.billed.after}
expression changed: ${dg.exprChanged}; policy changed: ${dg.policyChanged}${dg.error ? "; runtime error: " + dg.error : ""}
Current expression:
${d.expression}

Explain in at most 2 short sentences, plain English for a pricing analyst, the root cause of the difference. Name the specific input or term in backticks and quote amounts only from the numbers above, formatted as dollars like $1,234.56 (inputs that are not money stay plain numbers). Then judge whether this looks like an intended rate change (update the expectation) or a regression (fix the formula).
Reply with only JSON: {"why": string, "verdict": "intended" | "bug" | "unclear"}`, undefined, 400);
    return r && r.why ? { why: r.why, verdict: r.verdict || "unclear", live: true } : local;
  } catch { return local; }
}
async function novaAsk(q, d, run) {
  try {
    const r = await novaCall(`${novaDraftBrief(d)}\nLast preview: ${run && run.ok ? `formula ${fsMoney(run.value)}, billed ${fsMoney(run.policy?.amount)}` : run?.error || "no result"}.\n\nThe person asks: ${q}`);
    if (!r || typeof r.reply !== "string") throw new Error("bad");
    return { ...r, live: true };
  } catch { return { ...novaFallback(q, d, run), live: false }; }
}

function lineDiff(a, b) {
  const A = (a || "").split("\n"), B = (b || "").split("\n");
  const m = A.length, n = B.length, L = Array.from({ length: m + 1 }, () => Array(n + 1).fill(0));
  for (let i = m - 1; i >= 0; i--) for (let j = n - 1; j >= 0; j--) L[i][j] = A[i] === B[j] ? L[i + 1][j + 1] + 1 : Math.max(L[i + 1][j], L[i][j + 1]);
  const out = []; let i = 0, j = 0;
  while (i < m && j < n) { if (A[i] === B[j]) { out.push({ k: " ", t: A[i] }); i++; j++; } else if (L[i + 1][j] >= L[i][j + 1]) out.push({ k: "-", t: A[i++] }); else out.push({ k: "+", t: B[j++] }); }
  while (i < m) out.push({ k: "-", t: A[i++] });
  while (j < n) out.push({ k: "+", t: B[j++] });
  return out.filter((l) => !(l.k === "-" && !l.t.trim() && !a.trim()));
}

// What the proposal becomes once inserted: the editor's draft with Nova's changes on top.
function novaApplied(d, p) {
  const names = new Set(d.variables.map((v) => v.name));
  const vars = [...d.variables, ...(p.variables || []).filter((v) => v.name && !names.has(v.name)).map((v) => ({ id: fsId("v"), required: false, ...v, type: v.type || "Number", description: v.description || "" }))];
  let bd = d.breakdown;
  if (p.breakdown && p.breakdown.length) bd = p.breakdown.map((b) => ({ id: fsId("b"), ...b }));
  if (p.breakdownAdd && bd.length && !bd.some((b) => b.name === p.breakdownAdd.name)) bd = [...bd, { id: fsId("b"), ...p.breakdownAdd }];
  return { ...d, expression: p.expression, variables: vars, breakdown: bd, name: d.name || p.name || "", description: d.description || p.description || "", type: d.name ? d.type : p.type || d.type };
}

function ProposalCard({ p, draft, state, onInsert, onUndo, onDismiss }) {
  const next = aM(() => novaApplied(draft, p), [p, state === "inserted" ? null : draft]);
  const an = aM(() => fsAnalyze(next.expression, fsContext(next.variables)), [next]);
  const errs = an.issues.filter((i) => i.sev === "error"), warns = an.issues.filter((i) => i.sev === "warn");
  const priced = aM(() => [{ id: "Sample load", meta: "742 mi · 3 stops", base: null }, ...FS_SHIPMENTS.map((s) => ({ id: s.id, meta: s.meta, base: s.values }))].map((r) => {
    const x = fsRun(next, r.base); return { ...r, amount: x.ok && x.policy ? x.policy.amount : null, err: x.ok ? null : x.error };
  }), [next]);
  const diff = aM(() => lineDiff(state === "inserted" ? p.before || "" : draft.expression, p.expression), [p, draft.expression, state]);
  const newVars = (p.variables || []).filter((v) => !draft.variables.some((x) => x.name === v.name) || state === "inserted");
  return (
    <div className={"pc" + (state === "inserted" ? " done" : "") + (state === "dismissed" ? " gone" : "")}>
      <div className="pc-h"><FI n="code" s={13} /><b>{draft.expression.trim() || p.before ? "Proposed change" : "Proposed formula"}</b>{p.name && !draft.name && <span className="pc-nm">{p.name}</span>}<span className="sp"></span>
        {state === "inserted" ? <span className="bdg success"><FI n="check" s={11} />In the editor</span> : null}</div>
      <div className="pc-diff mono">{diff.map((l, i) => <div key={i} className={"dl " + (l.k === "+" ? "add" : l.k === "-" ? "del" : "")}><i>{l.k === " " ? "" : l.k}</i><span>{l.t || " "}</span></div>)}</div>
      {newVars.length > 0 && <div className="pc-vars">{newVars.map((v) => <span key={v.name} className="pc-var"><b className="mono">{v.name}</b><em>{v.type || "Number"} = {String(v.defaultValue ?? "—")}</em><i>new input</i></span>)}</div>}
      {p.explanation && <p className="pc-x">{p.explanation}</p>}
      <div className={"pc-chk " + (errs.length ? "bad" : warns.length ? "warn" : "ok")}>
        <FI n={errs.length ? "xc" : warns.length ? "alert" : "okc"} s={13} />
        {errs.length ? errs[0].msg : warns.length ? warns[0].msg : "Checked against the shipment schema. Every name resolves."}
      </div>
      <div className="pc-t">
        <div className="pc-th"><span>Priced by the formula engine</span><span>Billed</span></div>
        {priced.map((r) => <div key={r.id} className="pc-tr"><span><b className={r.base ? "mono" : ""}>{r.id}</b><em>{r.meta}</em></span>{r.amount !== null ? <b className="mono">{fsMoney(r.amount)}</b> : <span className="pc-err">{r.err}</span>}</div>)}
      </div>
      {state === "inserted" ? (
        <div className="pc-a"><span className="pc-note">Review and test it, then save. Nothing is saved yet.</span><button className="tb tb-ghost tb-s-xs" onClick={onUndo}><FI n="undo" s={12} />Undo</button></div>
      ) : state === "dismissed" ? null : (
        <div className="pc-a"><button className="tb tb-default tb-s-sm" onClick={() => onInsert(next)} disabled={errs.length > 0}>Insert into editor</button><button className="tb tb-quiet tb-s-sm" onClick={onDismiss}>Dismiss</button></div>
      )}
    </div>
  );
}

function ScenarioPicks({ list, draft, onAdd, added, canSave }) {
  const priced = aM(() => list.map((s) => { const r = fsRun(draft, null, s.inputs); return { ...s, expected: r.ok && r.policy ? r.policy.amount : null, err: r.ok ? null : r.error }; }), [list, draft]);
  const pending = priced.filter((s) => s.expected !== null && !added.has(s.name));
  return (
    <div className="pc">
      <div className="pc-h"><FI n="beaker" s={13} /><b>Proposed scenarios</b><span className="sp"></span>{pending.length > 1 && <button className="tb tb-ghost tb-s-xs" onClick={() => onAdd(pending)}><FI n="plus" s={12} />Add all</button>}</div>
      <div className="pc-t">
        {priced.map((s) => (
          <div key={s.name} className="pc-tr sc">
            <span><b>{s.name}</b><em className="mono">{Object.entries(s.inputs).map(([k, v]) => `${k} ${fsFmt(v)}`).join(" · ")}</em></span>
            {s.expected !== null ? <b className="mono">{fsMoney(s.expected)}</b> : <span className="pc-err">{s.err}</span>}
            <button className={"tb tb-s-xs " + (added.has(s.name) ? "tb-ghost" : "tb-outline")} disabled={s.expected === null || added.has(s.name)} onClick={() => onAdd([s])}>{added.has(s.name) ? <><FI n="check" s={11} />Added</> : <><FI n="plus" s={11} />Add</>}</button>
          </div>
        ))}
      </div>
      <p className="pc-note">Expected amounts are the engine's, from the formula in the editor. {canSave ? "Added scenarios keep running as you edit." : "They are saved with the template when you create it."}</p>
    </div>
  );
}

const NOVA_STARTERS = {
  empty: ["Per mile with an 18% fuel surcharge and a $250 minimum", "Per CWT, weight rounded up to the next 100 lb", "$75 per stop after the first two"],
  full: ["Explain this formula, term by term", "Add a weekend pickup fee", "Suggest edge cases to test", "Why did the short haul bill the minimum?"],
};

function useTypewriter(list, on) {
  const [st, setSt] = aS({ i: 0, n: 0 });
  aE(() => {
    if (!on) return;
    const full = list[st.i % list.length];
    const t = setTimeout(() => setSt((s) => (s.n < full.length ? { ...s, n: s.n + 1 } : { i: s.i + 1, n: 0 })), st.n < full.length ? 28 : 2600);
    return () => clearTimeout(t);
  }, [st, on, list]);
  const full = list[st.i % list.length];
  return { text: full.slice(0, st.n), full };
}
function Elapsed({ from }) {
  const [, tick] = aS(0);
  aE(() => { const t = setInterval(() => tick((x) => x + 1), 100); return () => clearInterval(t); }, []);
  return <span className="nl-el mono">{((Date.now() - from) / 1000).toFixed(1)}s</span>;
}

function NovaComposer({ draft, busy, status, onSend, inputs, starters, hasReplies }) {
  const [q, setQ] = aS("");
  const [model, setModel] = aS("");
  const empty = !q;
  const tw = useTypewriter(starters, empty && !busy);
  const send = (t) => { const v = (t ?? q).trim(); if (!v || busy) return; setQ(""); onSend(v); };
  const lines = draft.expression.split("\n").filter(Boolean).length;
  return (
    <div className={"cmp" + (busy ? " busy" : "")}>
      <span className="cmp-ring" aria-hidden="true"></span>
      {busy && status && <div className="cmp-st" key={status}><span className="cmp-st-d"></span><span className="shim">{status}</span></div>}
      <div className="cmp-ta">
        <textarea rows={2} value={q} aria-label="Ask Nova" onChange={(e) => setQ(e.target.value)}
          onKeyDown={(e) => { if (e.key === "Enter" && !e.shiftKey) { e.preventDefault(); send(); } if (e.key === "Tab" && empty && tw.text) { e.preventDefault(); setQ(tw.full); } }}></textarea>
        {empty && !busy && <div className="tw" aria-hidden="true"><span>{tw.text}</span><i className="tw-c"></i>{tw.text === tw.full && <span className="tw-tab"><span className="kbd">Tab</span></span>}</div>}
      </div>
      <div className="cmp-b">
        <span className="pgc" title="Nova reads the editor as it is, saved or not"><FI n="code" s={13} /><span className="pgc-t">{draft.name || "Formula editor"}</span><em>{lines ? `${lines} lines · ${inputs} inputs` : "empty"}</em></span>
        <span className="sp"></span>
        <ModelPicker value={model} onChange={setModel} hasReplies={hasReplies} />
        {busy ? <button className="send stop" title="Stop" aria-label="Stop"><span className="sq"></span></button>
          : <button className="send" disabled={!q.trim()} onClick={() => send()} aria-label="Send"><FI n="arrowUp" s={15} w={2.2} /></button>}
      </div>
    </div>
  );
}

function NovaPane({ thread, busy, draft, onAsk, onInsert, onUndo, onDismiss, onAddScenarios, addedScenarios, canSave, inputs }) {
  const end = aR();
  aE(() => { const el = end.current && end.current.parentElement; if (el) el.scrollTo({ top: el.scrollHeight, behavior: "smooth" }); }, [thread.length, busy]);
  const has = !!draft.expression.trim();
  const starters = has ? NOVA_STARTERS.full : NOVA_STARTERS.empty;
  const live = thread.find((m) => m.role === "nova" && m.steps.some((s) => !s.done));
  const status = live ? (live.steps.find((s) => !s.done) || {}).text + "…" : null;
  const qSize = (t) => (t.length > 90 ? " long" : t.length > 48 ? " mid" : "");
  return (
    <div className="nv">
      <ScrollArea className="nv-sa" viewportClassName="nv-flow" maskVariant="background">
        {thread.length === 0 && (
          <div className="nv-home">
            <NovaMark s={22} />
            <h3>{has ? "What should we change?" : "How should this price a load?"}</h3>
            <p>Nova reads the editor as it is, saved or not, and prices every draft with the formula engine. Nothing lands in the editor until you insert it.</p>
            <div className="sugg">{starters.map((s) => <button key={s} onClick={() => onAsk(s)}><span>{s}</span><FI n="arrowR" s={12} /></button>)}</div>
          </div>
        )}
        {thread.map((m) => m.role === "user" ? (
          <div key={m.id} className={"q" + qSize(m.text)}>{m.text}</div>
        ) : (
          <div key={m.id} className="ans">
            <div className="nar">{m.steps.map((s, i) => (
              <div key={i} className={"nl " + (s.done ? "past" : "cur")}>
                <span className="ck">{s.done ? <FI n="check" s={12} w={2.2} /> : <span className="cmp-st-d"></span>}</span>
                <span className="tx">{s.text}</span>
                {s.meta ? <span className="nl-el mono">{s.meta}</span> : !s.done && m.at ? <Elapsed from={m.at} /> : null}
              </div>))}
            </div>
            {m.reply && <div className="prose"><p>{m.reply.split(/(`[^`]+`)/).map((part, i) => part.startsWith("`") ? <code key={i}>{part.slice(1, -1)}</code> : part)}</p></div>}
            {m.offline && <p className="nv-off"><FI n="info" s={12} />Answered offline from the standard library and the engine.</p>}
            {m.proposal && <div className="art"><ProposalCard p={m.proposal} draft={draft} state={m.state} onInsert={(next) => onInsert(m.id, next)} onUndo={() => onUndo(m.id)} onDismiss={() => onDismiss(m.id)} /></div>}
            {m.scenarios && <div className="art"><ScenarioPicks list={m.scenarios} draft={draft} onAdd={onAddScenarios} added={addedScenarios} canSave={canSave} /></div>}
          </div>
        ))}
        <div ref={end}></div>
      </ScrollArea>
      <div className="nv-dock">
        <NovaComposer draft={draft} busy={busy} status={status} onSend={onAsk} inputs={inputs} starters={starters} hasReplies={thread.length > 0} />
        <p className="nv-hint">Nova can be wrong. Every amount it shows is the formula engine's.</p>
      </div>
    </div>
  );
}

Object.assign(window, { novaAsk, novaExplain, novaDescribe, novaSummary, novaLocalExplain, NovaPane, ProposalCard, ScenarioPicks, lineDiff });
