// A browser stand-in for the formula engine (rating/app/formula/engine): expr-lang syntax,
// shipment schema variables, the template's own inputs, then the charge policy.
const FS_KW = new Set(["true", "false", "nil", "in", "not", "and", "or", "let", "if", "else"]);
const FS_OPW = new Set(["startsWith", "endsWith", "contains", "matches"]);
const FS_TOK = /(\s+)|(\/\/[^\n]*|\/\*[\s\S]*?(?:\*\/|$))|(\d+(?:\.\d+)?)|("(?:\\.|[^"\\\n])*"?|'(?:\\.|[^'\\\n])*'?)|([A-Za-z_][\w]*(?:\.[A-Za-z_][\w]*)*)|(\?\?|==|!=|<=|>=|&&|\|\||\*\*|[-+*\/%<>!?:^])|([(),;=\[\]{}.])|(.)/g;

function fsTokenize(src) {
  const out = [];
  FS_TOK.lastIndex = 0;
  let m;
  while ((m = FS_TOK.exec(src))) {
    const t = m[1] ? "ws" : m[2] ? "com" : m[3] ? "num" : m[4] ? "str" : m[5] ? "id" : m[6] ? "op" : m[7] ? "pu" : "bad";
    out.push({ t, v: m[0], s: m.index, e: m.index + m[0].length });
  }
  return out;
}

function fsContext(customs) {
  const vars = new Map();
  FS_VARS.forEach((v) => vars.set(v.name, v));
  (customs || []).forEach((c) => c.name && vars.set(c.name, { ...c, category: "custom", custom: true, nullable: false }));
  const roots = new Set([...vars.keys()].map((k) => k.split(".")[0]));
  const funcs = new Set([...FS_FUNCS.map((f) => f.name), "len", "int", "float", "upper", "lower", "string"]);
  return { vars, roots, funcs };
}

function fsAnalyze(src, ctx) {
  const toks = fsTokenize(src || "");
  const cls = toks.map((t) => t.t);
  const issues = [];
  const used = { vars: new Set(), customs: new Set(), funcs: new Set() };
  const lets = [];
  const sigNext = (i) => { for (let j = i + 1; j < toks.length; j++) if (toks[j].t !== "ws" && toks[j].t !== "com") return j; return -1; };
  const sigPrev = (i) => { for (let j = i - 1; j >= 0; j--) if (toks[j].t !== "ws" && toks[j].t !== "com") return j; return -1; };
  const stack = [];
  const stmts = [];
  let cur = { s: 0, idx: [] };
  toks.forEach((tk, i) => {
    if (tk.t === "pu" && tk.v === ";" && stack.length === 0) { cur.e = tk.s; stmts.push(cur); cur = { s: tk.e, idx: [] }; return; }
    cur.idx.push(i);
    if (tk.t === "pu" && (tk.v === "(" || tk.v === "[")) {
      const p = sigPrev(i);
      stack.push(p >= 0 && toks[p].t === "id" && tk.v === "(" ? toks[p].v : "");
    } else if (tk.t === "pu" && (tk.v === ")" || tk.v === "]")) {
      if (!stack.length) issues.push({ sev: "error", s: tk.s, e: tk.e, msg: `Unmatched "${tk.v}"` });
      else stack.pop();
    } else if (tk.t === "str" && (tk.v.length < 2 || tk.v[tk.v.length - 1] !== tk.v[0])) {
      cls[i] = "str"; issues.push({ sev: "error", s: tk.s, e: tk.e, msg: "This string is never closed" });
    } else if (tk.t === "bad") {
      issues.push({ sev: "error", s: tk.s, e: tk.e, msg: `Unexpected character "${tk.v}"` });
    } else if (tk.t === "id") {
      const p = sigPrev(i), n = sigNext(i);
      const prevV = p >= 0 ? toks[p].v : "";
      if (FS_KW.has(tk.v) || FS_OPW.has(tk.v)) { cls[i] = "kw"; return; }
      if (prevV === "let") { cls[i] = "letdef"; lets.push({ name: tk.v, s: tk.s, e: tk.e }); return; }
      if (n >= 0 && toks[n].v === "(") {
        if (ctx.funcs.has(tk.v)) { cls[i] = "fn"; used.funcs.add(tk.v); }
        else { cls[i] = "fnbad"; issues.push({ sev: "error", s: tk.s, e: tk.e, msg: `"${tk.v}" is not a formula function`, kind: "fn", name: tk.v }); }
        return;
      }
      if (lets.some((l) => l.name === tk.v)) { cls[i] = "let"; return; }
      const doc = ctx.vars.get(tk.v);
      if (doc || ctx.roots.has(tk.v)) {
        cls[i] = doc && doc.custom ? "cvar" : "var";
        (doc && doc.custom ? used.customs : used.vars).add(tk.v);
        if (doc && doc.nullable) {
          const guarded = stack.includes("coalesce") || (n >= 0 && toks[n].v === "??");
          if (!guarded) issues.push({ sev: "warn", s: tk.s, e: tk.e, msg: `${tk.v} is empty on some shipments, and the formula fails there`, kind: "nullable", name: tk.v, fix: `(${tk.v} ?? 0)` });
        }
        return;
      }
      cls[i] = "bad";
      issues.push({ sev: "error", s: tk.s, e: tk.e, msg: `Unknown variable "${tk.v}"`, kind: "var", name: tk.v });
    }
  });
  if (stack.length) issues.push({ sev: "error", s: src.length, e: src.length, msg: `Missing "${stack.length > 1 ? ")" + " ×" + stack.length : ")"}" at the end` });
  cur.e = src.length;
  stmts.push(cur);
  const real = stmts.filter((st) => st.idx.some((i) => toks[i].t !== "ws" && toks[i].t !== "com"));
  real.forEach((st) => {
    const first = st.idx.find((i) => toks[i].t !== "ws" && toks[i].t !== "com");
    st.s = toks[first].s;
    const last = [...st.idx].reverse().find((i) => toks[i].t !== "ws" && toks[i].t !== "com");
    st.e = toks[last].e;
    st.code = src.slice(st.s, st.e);
    st.isLet = toks[first].v === "let";
    st.letName = st.isLet ? (lets.find((l) => l.s > st.s && l.s < st.e) || {}).name : null;
  });
  if (real.length && real[real.length - 1].isLet) issues.push({ sev: "error", s: real[real.length - 1].s, e: real[real.length - 1].e, msg: "The formula ends on a let; add the expression it returns" });
  return { src, toks, cls, issues, used, lets, stmts: real };
}

function fsTokJs(an, i) {
  const tk = an.toks[i], c = an.cls[i];
  if (c === "kw") {
    const map = { and: "&&", or: "||", not: "!", nil: "null", true: "true", false: "false" };
    if (tk.v in map) return map[tk.v];
    throw new Error(`"${tk.v}" isn't supported in the preview yet`);
  }
  if (c === "fn") return `__f.${tk.v}`;
  if (c === "var" || c === "cvar") return `__v(${JSON.stringify(tk.v)})`;
  if (c === "let") return `L_${tk.v}`;
  if (c === "op" && tk.v === "^") return "**";
  if (c === "pu" && tk.v === "=") throw new Error('Use "==" to compare; "=" only follows let');
  return tk.v;
}

function fsCompile(an) {
  const parts = [];
  an.stmts.forEach((st) => {
    const sig = st.idx.filter((i) => an.toks[i].t !== "ws" && an.toks[i].t !== "com");
    let k = 0;
    if (st.isLet) {
      if (an.cls[sig[1]] !== "letdef" || an.toks[sig[2]]?.v !== "=") throw new Error("let needs a name and =");
      k = 3;
    }
    const js = sig.slice(k).map((i) => fsTokJs(an, i)).join(" ");
    parts.push(st.isLet ? `const L_${an.toks[sig[1]].v} = (${js});` : `__r = (${js});`);
  });
  return new Function("__v", "__f", `"use strict"; let __r; ${parts.join("\n")}\nreturn __r;`);
}

const fsRound = (x, d = 0) => { const f = 10 ** d; return Math.sign(x) * Math.round(Math.abs(x) * f + 1e-9) / f; };
const fsHalfEven = (x, d = 0) => { const f = 10 ** d, n = x * f, r = Math.round(n); return (Math.abs(n % 1) === 0.5 ? 2 * Math.round(n / 2) : r) / f; };
const FS_FN = {
  min: Math.min, max: Math.max, pow: Math.pow, abs: Math.abs, ceil: Math.ceil, floor: Math.floor,
  clamp: (v, a, b) => Math.min(Math.max(v, a), b),
  round: fsRound, roundHalfEven: fsHalfEven,
  roundUp: (x, d = 0) => Math.ceil(x * 10 ** d - 1e-9) / 10 ** d,
  roundDown: (x, d = 0) => Math.floor(x * 10 ** d + 1e-9) / 10 ** d,
  roundTo: (x, inc) => Math.round(x / inc) * inc,
  sum: (...a) => a.reduce((s, x) => s + x, 0), avg: (...a) => a.reduce((s, x) => s + x, 0) / a.length,
  coalesce: (...a) => a.find((x) => x !== null && x !== undefined) ?? null,
  len: (x) => (x == null ? 0 : x.length), int: (x) => Math.trunc(x), float: (x) => Number(x), string: (x) => String(x), upper: (s) => String(s).toUpperCase(), lower: (s) => String(s).toLowerCase(),
  lookup: (t, k) => { const tb = FS_TABLES[t]; if (!tb) throw new Error(`Rate table "${t}" was not found`); if (!(k in tb)) throw new Error(`No row in "${t}" for ${k}`); return tb[k]; },
  lookupOr: (t, k, f) => { const tb = FS_TABLES[t]; return tb && k in tb ? tb[k] : f; },
  lookup2: (t) => { throw new Error(`Rate table "${t}" resolves on the server; preview with a real shipment`); },
  lookupInterp: (t) => { throw new Error(`Rate table "${t}" resolves on the server; preview with a real shipment`); },
  deficitWeight: (t) => { throw new Error(`Rate table "${t}" resolves on the server; preview with a real shipment`); },
};

// values: flat map of every variable's value for this run. sources: name -> where it came from.
function fsEvaluate(src, ctx, values, an) {
  an = an || fsAnalyze(src, ctx);
  if (!src || !src.trim()) return { ok: false, empty: true, an };
  const err = an.issues.find((i) => i.sev === "error");
  if (err) return { ok: false, error: err.msg, an };
  const t0 = performance.now();
  const used = new Map();
  try {
    const fn = fsCompile(an);
    const risky = new Set(an.issues.filter((i) => i.kind === "nullable").map((i) => i.name));
    const value = fn((n) => {
      const v = values[n];
      if (v === undefined) throw new Error(`No value for "${n}" on this load`);
      used.set(n, v);
      if (v === null && risky.has(n)) throw new Error(`${n} is empty on this load. Guard it with (${n} ?? 0) or coalesce(${n}, 0)`);
      return v;
    }, FS_FN);
    if (typeof value === "number" && !Number.isFinite(value)) throw new Error("The result is not a finite number; check for division by zero");
    return { ok: true, value, used, ms: performance.now() - t0, an };
  } catch (e) {
    return { ok: false, error: e instanceof SyntaxError ? "The expression is incomplete or malformed" : e.message, an, used };
  }
}

function fsRoundMode(x, mode, p) {
  const f = 10 ** p;
  switch (mode) {
    case "HalfEven": return fsHalfEven(x, p);
    case "Up": return Math.ceil(x * f - 1e-9) / f;
    case "Down": return Math.floor(x * f + 1e-9) / f;
    case "None": return x;
    default: return fsRound(x, p);
  }
}
function fsApplyPolicy(raw, policy) {
  let amount = raw, clamp = null;
  if (policy.min != null && amount < policy.min) { clamp = { bound: "minimum", limit: policy.min }; amount = policy.min; }
  if (policy.max != null && amount > policy.max) { clamp = { bound: "maximum", limit: policy.max }; amount = policy.max; }
  const rounded = fsRoundMode(amount, policy.mode, Number(policy.precision) || 0);
  return { raw, clamped: amount, amount: rounded, clamp, roundedChange: Math.abs(rounded - amount) > 1e-9 };
}

function fsValues(base, customs, overrides) {
  const out = { ...FS_SAMPLE, ...(base || {}) };
  (customs || []).forEach((c) => { if (c.name) out[c.name] = c.defaultValue === "" || c.defaultValue === undefined ? null : c.defaultValue; });
  return { ...out, ...(overrides || {}) };
}
function fsSource(name, customs, overrides, shipment) {
  if (overrides && name in overrides) return "input";
  if ((customs || []).some((c) => c.name === name)) return "default";
  if (!shipment) return "sample";
  return FS_FIELD_SOURCE.has(name) ? "shipment" : "computed";
}

// Run the whole template against one set of values.
function fsRun(draft, base, overrides, shipment) {
  const ctx = fsContext(draft.variables);
  const values = fsValues(base, draft.variables, overrides);
  const res = fsEvaluate(draft.expression, ctx, values);
  if (!res.ok || typeof res.value !== "number") return { ...res, values, ctx };
  const policy = fsApplyPolicy(res.value, draft.policy);
  const lines = (draft.breakdown || []).filter((b) => b.name || b.expression).map((b) => {
    const r = fsEvaluate(b.expression, ctx, values);
    return { ...b, ok: r.ok && typeof r.value === "number", amount: r.ok ? r.value : null, error: r.ok ? (typeof r.value === "number" ? null : "Not a number") : r.error || "Empty" };
  });
  const sum = lines.reduce((s, l) => s + (l.ok ? l.amount : 0), 0);
  const failed = lines.filter((l) => !l.ok).length;
  const reconcile = lines.length ? { sum, residual: res.value - sum, balanced: Math.abs(res.value - sum) < 0.005 && !failed, failed, clampMismatch: !!policy.clamp && Math.abs(sum - res.value) < 0.005 } : null;
  const receipt = [...res.used.entries()].map(([name, value]) => ({ name, value, source: fsSource(name, draft.variables, overrides, shipment) }));
  return { ...res, values, ctx, policy, lines, reconcile, receipt };
}

function fsScenarioRun(draft, sc) {
  const r = fsRun(draft, null, sc.inputs);
  const actual = r.ok && r.policy ? r.policy.amount : null;
  return { ...sc, actual, error: r.ok ? null : r.error, pass: actual !== null && Math.abs(actual - sc.expected) < 0.005 };
}

// Re-rate recent freight with the saved and the edited formula (formulatemplateservice/backtest.go).
function fsBacktest(saved, draft, n = 128) {
  let seed = 7;
  const rnd = () => ((seed = (seed * 16807) % 2147483647) / 2147483647);
  const rows = [];
  for (let i = 0; i < n; i++) {
    const v = { baseRate: +(2.4 + rnd() * 1.2).toFixed(2), totalDistance: Math.round(40 + rnd() * 1300), totalStops: 2 + Math.floor(rnd() * rnd() * 5), hasHazmat: rnd() < 0.08, isWeekendPickup: rnd() < 0.16, requiresTemperatureControl: rnd() < 0.2, totalWeight: Math.round(8000 + rnd() * 36000), totalHours: +(4 + rnd() * 30).toFixed(1) };
    v.pickupDayOfWeek = v.isWeekendPickup ? 6 : 1 + Math.floor(rnd() * 5);
    const a = fsRun(saved, null, v), b = fsRun(draft, null, v);
    rows.push({ id: `S-${239400 + i * 7}`, before: a.ok && a.policy ? a.policy.amount : null, after: b.ok && b.policy ? b.policy.amount : null, v });
  }
  const ok = rows.filter((r) => r.before !== null && r.after !== null);
  const changed = ok.filter((r) => Math.abs(r.after - r.before) >= 0.005);
  const failed = rows.length - ok.length;
  const totalBefore = ok.reduce((s, r) => s + r.before, 0), totalAfter = ok.reduce((s, r) => s + r.after, 0);
  const movers = [...changed].sort((x, y) => Math.abs(y.after - y.before) - Math.abs(x.after - x.before)).slice(0, 3);
  return { n, changed: changed.length, failed, delta: totalAfter - totalBefore, pct: totalBefore ? ((totalAfter - totalBefore) / totalBefore) * 100 : 0, avg: changed.length ? changed.reduce((s, r) => s + (r.after - r.before), 0) / changed.length : 0, movers };
}

const fsMoney = (x, d = 2) => x == null || Number.isNaN(x) ? "—" : (x < 0 ? "−" : "") + "$" + Math.abs(x).toLocaleString("en-US", { minimumFractionDigits: d, maximumFractionDigits: d });
const fsFmt = (v) => v === null || v === undefined ? "empty" : typeof v === "number" ? (Number.isInteger(v) ? v.toLocaleString("en-US") : String(+v.toFixed(4))) : typeof v === "boolean" ? String(v) : `"${v}"`;

Object.assign(window, { fsTokenize, fsContext, fsAnalyze, fsEvaluate, fsApplyPolicy, fsValues, fsRun, fsScenarioRun, fsBacktest, fsMoney, fsFmt, FS_KW });

// Why a scenario no longer matches: evaluate every let, the formula and the policy on the scenario's
// inputs, once with the template the expectation was recorded on and once with the editor.
function fsTerms(draft, inputs) {
  const ctx = fsContext(draft.variables);
  const values = fsValues(null, draft.variables, inputs);
  const an = fsAnalyze(draft.expression, ctx);
  const out = new Map();
  an.stmts.forEach((st) => {
    if (!st.isLet || !st.letName) return;
    const src = draft.expression.slice(0, st.e) + (st.code.trim().endsWith(";") ? "" : ";") + "\n" + st.letName;
    const r = fsEvaluate(src, ctx, values);
    out.set(st.letName, r.ok ? r.value : null);
  });
  const run = fsRun(draft, null, inputs);
  return { terms: out, run, values };
}
function fsRecorded(sc, saved) {
  if (!sc.recordedOn) return { base: saved, label: "the saved version" };
  const vars = saved.variables.map((v) => (v.name in (sc.recordedOn.variables || {}) ? { ...v, defaultValue: sc.recordedOn.variables[v.name] } : v));
  return { base: { ...saved, variables: vars, expression: sc.recordedOn.expression || saved.expression, policy: sc.recordedOn.policy || saved.policy }, label: "v" + sc.recordedOn.version };
}
function fsDiagnose(draft, saved, sc) {
  const { base, label } = fsRecorded(sc, saved);
  const A = fsTerms(base, sc.inputs), B = fsTerms(draft, sc.inputs);
  const rows = [];
  const near = (x, y) => x === y || (typeof x === "number" && typeof y === "number" && Math.abs(x - y) < 0.005);
  draft.variables.forEach((v) => {
    if (!v.name || v.name in sc.inputs) return;
    const was = base.variables.find((x) => x.name === v.name);
    const b = was ? was.defaultValue : undefined, a = v.defaultValue;
    if (!near(b, a)) rows.push({ kind: "input", name: v.name, before: b === undefined ? null : b, after: a, added: !was });
  });
  base.variables.forEach((v) => { if (v.name && !(v.name in sc.inputs) && !draft.variables.some((x) => x.name === v.name)) rows.push({ kind: "input", name: v.name, before: v.defaultValue, after: null, removed: true }); });
  const names = new Set([...A.terms.keys(), ...B.terms.keys()]);
  names.forEach((n) => { const b = A.terms.has(n) ? A.terms.get(n) : null, a = B.terms.has(n) ? B.terms.get(n) : null; if (!near(a, b)) rows.push({ kind: "term", name: n, before: b, after: a }); });
  const rb = A.run.ok && A.run.policy ? A.run.policy : null, ra = B.run.ok && B.run.policy ? B.run.policy : null;
  const exprChanged = base.expression.trim() !== draft.expression.trim();
  const policyChanged = JSON.stringify(base.policy) !== JSON.stringify(draft.policy);
  return {
    label, recordedOn: sc.recordedOn || null, rows, exprChanged, policyChanged,
    raw: { before: rb ? rb.raw : null, after: ra ? ra.raw : null },
    clamp: { before: rb && rb.clamp, after: ra && ra.clamp },
    billed: { before: rb ? rb.amount : null, after: ra ? ra.amount : null },
    expected: sc.expected, error: B.run.ok ? null : B.run.error,
    reproduces: rb && Math.abs(rb.amount - sc.expected) < 0.005,
  };
}
