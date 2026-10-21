// A stand-in for the formula engine (rating/app/formula/engine), ported from the design handoff's
// browser engine (design_handoff_formula_studio/design/engine.jsx): expr-lang syntax over the
// shipment schema and a template's own inputs, `let` statements, `??`, the ternary and the
// function library, then the charge policy (minimum, maximum, rounding). Test tooling only: it
// compiles the expression to JavaScript, so it prices exactly what the studio's preview,
// scenarios, backtest and Nova need, and no more. Offsets are UTF-16 code units, which is what a
// JavaScript string index counts in and what CodeMirror uses.
import { FIELD_SOURCE, FUNCTIONS, SAMPLE, TABLES, VARIABLES } from "./formula.mjs";

const KEYWORDS = new Set(["true", "false", "nil", "in", "not", "and", "or", "let", "if", "else"]);
const OPERATOR_WORDS = new Set(["startsWith", "endsWith", "contains", "matches"]);
const TOKEN =
  /(\s+)|(\/\/[^\n]*|\/\*[\s\S]*?(?:\*\/|$))|(\d+(?:\.\d+)?)|("(?:\\.|[^"\\\n])*"?|'(?:\\.|[^'\\\n])*'?)|([A-Za-z_][\w]*(?:\.[A-Za-z_][\w]*)*)|(\?\?|==|!=|<=|>=|&&|\|\||\*\*|[-+*/%<>!?:^])|([(),;=[\]{}.])|(.)/g;
const EXTRA_FUNCTIONS = ["len", "int", "float", "upper", "lower", "string"];

export const SCHEMA_NAMES = new Set(VARIABLES.map((v) => v.name));

export function tokenize(src) {
  const out = [];
  const re = new RegExp(TOKEN.source, "g");
  let m;
  while ((m = re.exec(src))) {
    const t = m[1] ? "ws" : m[2] ? "com" : m[3] ? "num" : m[4] ? "str" : m[5] ? "id" : m[6] ? "op" : m[7] ? "pu" : "bad";
    out.push({ t, v: m[0], s: m.index, e: m.index + m[0].length });
  }
  return out;
}

/** The names an expression may use: the schema, the template's inputs and the functions. */
export function context(customs) {
  const vars = new Map();
  VARIABLES.forEach((v) => vars.set(v.name, v));
  (customs || []).forEach((c) => c.name && vars.set(c.name, { ...c, category: "custom", custom: true, nullable: false }));
  const roots = new Set([...vars.keys()].map((k) => k.split(".")[0]));
  const funcs = new Set([...FUNCTIONS.map((f) => f.name), ...EXTRA_FUNCTIONS]);
  return { vars, roots, funcs };
}

const significant = (tk) => tk.t !== "ws" && tk.t !== "com";

/** Tokens, their classes, the issues a lint would show, the names used, and the statements. */
export function analyze(src, ctx) {
  const toks = tokenize(src || "");
  const cls = toks.map((t) => t.t);
  const issues = [];
  const used = { vars: new Set(), customs: new Set(), funcs: new Set() };
  const lets = [];
  const sigNext = (i) => {
    for (let j = i + 1; j < toks.length; j++) if (significant(toks[j])) return j;
    return -1;
  };
  const sigPrev = (i) => {
    for (let j = i - 1; j >= 0; j--) if (significant(toks[j])) return j;
    return -1;
  };
  const stack = [];
  const stmts = [];
  let cur = { s: 0, idx: [] };
  toks.forEach((tk, i) => {
    if (tk.t === "pu" && tk.v === ";" && stack.length === 0) {
      cur.e = tk.s;
      stmts.push(cur);
      cur = { s: tk.e, idx: [] };
      return;
    }
    cur.idx.push(i);
    if (tk.t === "pu" && (tk.v === "(" || tk.v === "[")) {
      const p = sigPrev(i);
      stack.push(p >= 0 && toks[p].t === "id" && tk.v === "(" ? toks[p].v : "");
    } else if (tk.t === "pu" && (tk.v === ")" || tk.v === "]")) {
      if (!stack.length) issues.push({ sev: "error", s: tk.s, e: tk.e, msg: `Unmatched "${tk.v}"` });
      else stack.pop();
    } else if (tk.t === "str" && (tk.v.length < 2 || tk.v[tk.v.length - 1] !== tk.v[0])) {
      issues.push({ sev: "error", s: tk.s, e: tk.e, msg: "This string is never closed" });
    } else if (tk.t === "bad") {
      issues.push({ sev: "error", s: tk.s, e: tk.e, msg: `Unexpected character "${tk.v}"` });
    } else if (tk.t === "id") {
      const p = sigPrev(i);
      const n = sigNext(i);
      const prevV = p >= 0 ? toks[p].v : "";
      if (KEYWORDS.has(tk.v) || OPERATOR_WORDS.has(tk.v)) {
        cls[i] = "kw";
        return;
      }
      if (prevV === "let") {
        cls[i] = "letdef";
        lets.push({ name: tk.v, s: tk.s, e: tk.e });
        return;
      }
      if (n >= 0 && toks[n].v === "(") {
        if (ctx.funcs.has(tk.v)) {
          cls[i] = "fn";
          used.funcs.add(tk.v);
        } else {
          cls[i] = "fnbad";
          issues.push({ sev: "error", s: tk.s, e: tk.e, msg: `"${tk.v}" is not a formula function`, kind: "fn", name: tk.v });
        }
        return;
      }
      if (lets.some((l) => l.name === tk.v)) {
        cls[i] = "let";
        return;
      }
      const doc = ctx.vars.get(tk.v);
      if (doc || ctx.roots.has(tk.v)) {
        cls[i] = doc && doc.custom ? "cvar" : "var";
        (doc && doc.custom ? used.customs : used.vars).add(tk.v);
        if (doc && doc.nullable) {
          const guarded = stack.includes("coalesce") || (n >= 0 && toks[n].v === "??");
          if (!guarded) {
            issues.push({ sev: "warn", s: tk.s, e: tk.e, msg: `${tk.v} is empty on some shipments, and the formula fails there`, kind: "nullable", name: tk.v, type: doc.type, fix: `(${tk.v} ?? 0)` });
          }
        }
        return;
      }
      cls[i] = "bad";
      issues.push({ sev: "error", s: tk.s, e: tk.e, msg: `Unknown variable "${tk.v}"`, kind: "var", name: tk.v });
    }
  });
  const source = src || "";
  if (stack.length) {
    issues.push({ sev: "error", s: source.length, e: source.length, msg: `Missing "${stack.length > 1 ? `) ×${stack.length}` : ")"}" at the end` });
  }
  cur.e = source.length;
  stmts.push(cur);
  const real = stmts.filter((st) => st.idx.some((i) => significant(toks[i])));
  real.forEach((st) => {
    const first = st.idx.find((i) => significant(toks[i]));
    const last = [...st.idx].reverse().find((i) => significant(toks[i]));
    st.s = toks[first].s;
    st.e = toks[last].e;
    st.code = source.slice(st.s, st.e);
    st.isLet = toks[first].v === "let";
    st.letName = st.isLet ? (lets.find((l) => l.s > st.s && l.s < st.e) || {}).name ?? null : null;
  });
  if (real.length && real[real.length - 1].isLet) {
    const last = real[real.length - 1];
    issues.push({ sev: "error", s: last.s, e: last.e, msg: "The formula ends on a let; add the expression it returns" });
  }
  return { src: source, toks, cls, issues, used, lets, stmts: real };
}

function tokenJs(an, i) {
  const tk = an.toks[i];
  const c = an.cls[i];
  if (c === "kw") {
    const map = { and: "&&", or: "||", not: "!", nil: "null", true: "true", false: "false" };
    if (tk.v in map) return map[tk.v];
    throw new Error(`"${tk.v}" isn't supported by the formula engine here`);
  }
  if (c === "fn") return `__f.${tk.v}`;
  if (c === "var" || c === "cvar") return `__v(${JSON.stringify(tk.v)})`;
  if (c === "let") return `L_${tk.v}`;
  if (c === "op" && tk.v === "^") return "**";
  if (c === "pu" && tk.v === "=") throw new Error('Use "==" to compare; "=" only follows let');
  return tk.v;
}

/** Each statement as JavaScript: `L_name = (…)` for a let, the bare expression for the result. */
function statementJs(an) {
  return an.stmts.map((st) => {
    const sig = st.idx.filter((i) => significant(an.toks[i]));
    let k = 0;
    if (st.isLet) {
      if (an.cls[sig[1]] !== "letdef" || an.toks[sig[2]]?.v !== "=") throw new Error("let needs a name and =");
      k = 3;
    }
    const js = sig.slice(k).map((i) => tokenJs(an, i)).join(" ");
    return { name: st.isLet ? an.toks[sig[1]].v : null, js };
  });
}

/**
 * Compiles the expression to a function that runs each statement in order, recording what each
 * came to, and stops at the first one that fails. The last record is the result.
 */
function compileTrace(an) {
  const parts = statementJs(an);
  const declared = parts.filter((p) => p.name).map((p) => `L_${p.name}`);
  const body = parts
    .map((p) => {
      const target = p.name ? `L_${p.name}` : "__r";
      return `try { ${target} = (${p.js}); __out.push({ value: ${target} }); } catch (e) { __out.push({ error: e }); return __out; }`;
    })
    .join("\n");
  return new Function(
    "__v",
    "__f",
    `"use strict"; const __out = []; let __r; ${declared.length ? `let ${declared.join(", ")};` : ""}\n${body}\nreturn __out;`,
  );
}

const roundHalfUp = (x, d = 0) => {
  const f = 10 ** d;
  return (Math.sign(x) * Math.round(Math.abs(x) * f + 1e-9)) / f;
};
const roundHalfEven = (x, d = 0) => {
  const f = 10 ** d;
  const n = x * f;
  const r = Math.round(n);
  return (Math.abs(n % 1) === 0.5 ? 2 * Math.round(n / 2) : r) / f;
};
const serverTable = (t) => {
  throw new Error(`Rate table "${t}" was not found`);
};
const LIBRARY = {
  min: Math.min,
  max: Math.max,
  pow: Math.pow,
  abs: Math.abs,
  ceil: Math.ceil,
  floor: Math.floor,
  sqrt: Math.sqrt,
  clamp: (v, a, b) => Math.min(Math.max(v, a), b),
  round: roundHalfUp,
  roundHalfEven,
  roundUp: (x, d = 0) => Math.ceil(x * 10 ** d - 1e-9) / 10 ** d,
  roundDown: (x, d = 0) => Math.floor(x * 10 ** d + 1e-9) / 10 ** d,
  roundTo: (x, inc) => Math.round(x / inc) * inc,
  sum: (...a) => a.reduce((s, x) => s + x, 0),
  avg: (...a) => a.reduce((s, x) => s + x, 0) / a.length,
  coalesce: (...a) => a.find((x) => x !== null && x !== undefined) ?? null,
  len: (x) => (x == null ? 0 : x.length),
  int: (x) => Math.trunc(x),
  float: (x) => Number(x),
  string: (x) => String(x),
  upper: (s) => String(s).toUpperCase(),
  lower: (s) => String(s).toLowerCase(),
  lookup: (t, k) => {
    const tb = TABLES[t];
    if (!tb) throw new Error(`Rate table "${t}" was not found`);
    if (!(k in tb)) throw new Error(`No row in "${t}" for ${k}`);
    return tb[k];
  },
  lookupOr: (t, k, f) => {
    const tb = TABLES[t];
    if (!tb) throw new Error(`Rate table "${t}" was not found`);
    return k in tb ? tb[k] : f;
  },
  lookup2: serverTable,
  lookup2Or: serverTable,
  lookupInterp: serverTable,
  deficitWeight: serverTable,
};

function errorText(error) {
  return error instanceof SyntaxError ? "The expression is incomplete or malformed" : String(error?.message ?? error);
}

/**
 * Runs every statement of the expression on one set of values. `steps` holds what each statement
 * came to (or why it failed); `value` is the result when every statement ran.
 */
function execute(src, ctx, values, an) {
  an = an || analyze(src, ctx);
  if (!src || !src.trim()) return { ok: false, empty: true, error: "Expression is required", an, steps: [], used: new Map() };
  const err = an.issues.find((i) => i.sev === "error");
  if (err) return { ok: false, error: err.msg, an, steps: [], used: new Map(), parseError: true };
  const started = process.hrtime.bigint();
  const used = new Map();
  let fn;
  try {
    fn = compileTrace(an);
  } catch (e) {
    return { ok: false, error: errorText(e), an, steps: [], used, parseError: true };
  }
  const risky = new Set(an.issues.filter((i) => i.kind === "nullable").map((i) => i.name));
  const steps = fn((n) => {
    const v = values[n];
    if (v === undefined) throw new Error(`No value for "${n}" on this load`);
    used.set(n, v);
    if (v === null && risky.has(n)) {
      throw new Error(`${n} is empty on this load. Guard it with (${n} ?? 0) or coalesce(${n}, 0)`);
    }
    return v;
  }, LIBRARY).map((step) => (step.error ? { error: errorText(step.error) } : step));
  const micros = Number((process.hrtime.bigint() - started) / 1000n);
  const failed = steps.find((step) => step.error);
  if (failed) return { ok: false, error: failed.error, an, steps, used, micros };
  const value = steps.at(-1)?.value;
  if (typeof value === "number" && !Number.isFinite(value)) {
    steps[steps.length - 1] = { error: "The result is not a finite number; check for division by zero" };
    return { ok: false, error: steps.at(-1).error, an, steps, used, micros };
  }
  return { ok: true, value, an, steps, used, micros };
}

export function evaluate(src, ctx, values, an) {
  return execute(src, ctx, values, an);
}

export function roundMode(x, mode, p) {
  const f = 10 ** p;
  switch (mode) {
    case "HalfEven":
      return roundHalfEven(x, p);
    case "Up":
      return Math.ceil(x * f - 1e-9) / f;
    case "Down":
      return Math.floor(x * f + 1e-9) / f;
    case "None":
      return x;
    default:
      return roundHalfUp(x, p);
  }
}

/** The charge policy: clamp to the minimum and maximum, then round. */
export function applyPolicy(raw, policy) {
  let amount = raw;
  let clamp = null;
  if (policy.min != null && amount < policy.min) {
    clamp = { bound: "min", limit: policy.min };
    amount = policy.min;
  }
  if (policy.max != null && amount > policy.max) {
    clamp = { bound: "max", limit: policy.max };
    amount = policy.max;
  }
  const precision = Number.isInteger(policy.precision) ? policy.precision : 2;
  const rounded = roundMode(amount, policy.mode || "HalfUp", precision);
  return { raw, clamped: amount, amount: rounded, clamp, roundedChange: Math.abs(rounded - amount) > 1e-9 };
}

/** Every value a run reads: the sample load (or a shipment over it), the inputs' defaults, then overrides. */
export function valuesFor(base, customs, overrides) {
  const out = { ...SAMPLE, ...base };
  (customs || []).forEach((c) => {
    if (c.name) out[c.name] = c.defaultValue === "" || c.defaultValue === undefined ? null : c.defaultValue;
  });
  return { ...out, ...overrides };
}

/** Where a value in the receipt came from, in the server's ValueSource words. */
export function sourceOf(name, customs, overrides, shipment) {
  if ((customs || []).some((c) => c.name === name)) {
    return overrides && name in overrides ? "input" : "default";
  }
  if (overrides && name in overrides) return "input";
  if (!shipment) return "sample";
  return FIELD_SOURCE.has(name) ? "field" : "computed";
}

const toNumber = (value) => {
  if (value === null || value === undefined || value === "") return null;
  const n = Number(value);
  return Number.isFinite(n) ? n : null;
};

/** A template's content (or a TestCaseCandidate) in the shape the engine runs. */
export function draftOf(content) {
  return {
    expression: content?.expression ?? "",
    variables: (content?.variableDefinitions ?? []).filter((v) => v && v.name).map((v) => ({ ...v })),
    breakdown: (content?.breakdownDefinitions ?? []).filter((b) => b && (b.name || b.expression)),
    policy: {
      min: toNumber(content?.minCharge),
      max: toNumber(content?.maxCharge),
      mode: content?.roundingMode || "HalfUp",
      precision: Number.isInteger(content?.roundingPrecision) ? content.roundingPrecision : 2,
    },
  };
}

/** Runs the whole template against one set of values: result, policy, breakdown and receipt. */
export function run(draft, base, overrides, shipment) {
  const ctx = context(draft.variables);
  const values = valuesFor(base, draft.variables, overrides);
  const res = execute(draft.expression, ctx, values);
  if (!res.ok) return { ...res, values, ctx };
  if (typeof res.value !== "number") {
    return { ...res, ok: false, error: "The expression evaluated to something other than an amount", values, ctx };
  }
  const policy = applyPolicy(res.value, draft.policy);
  const lines = (draft.breakdown || []).map((b) => {
    const r = execute(b.expression, ctx, values);
    const ok = r.ok && typeof r.value === "number";
    return { ...b, ok, amount: ok ? r.value : null, error: ok ? null : r.ok ? "Not a number" : r.error || "Empty" };
  });
  const sum = lines.reduce((s, l) => s + (l.ok ? l.amount : 0), 0);
  const failedLines = lines.filter((l) => !l.ok).length;
  const reconcile = lines.length
    ? { sum, residual: res.value - sum, balanced: Math.abs(res.value - sum) < 0.005 && !failedLines, failed: failedLines }
    : null;
  const receipt = [...res.used.entries()].map(([name, value]) => ({ name, value, source: sourceOf(name, draft.variables, overrides, shipment) }));
  return { ...res, values, ctx, policy, lines, reconcile, receipt };
}

/** The billed amount for a scenario's inputs, or the reason it fails. */
export function bill(draft, inputs, base) {
  const r = run(draft, base ?? null, inputs);
  return r.ok ? { amount: r.policy.amount, raw: r.value, clamp: r.policy.clamp, error: null } : { amount: null, raw: null, clamp: null, error: r.error };
}

/** A number as the server writes a decimal: plain digits, trailing zeros dropped. */
export function decimal(x) {
  if (x === null || x === undefined || Number.isNaN(x)) return null;
  const r = Math.round(Number(x) * 1e6) / 1e6;
  return Object.is(r, -0) ? "0" : String(r);
}

/** A value as a trace row writes it: decimals as decimal strings, everything else as text. */
export function display(value) {
  if (value === null || value === undefined) return null;
  if (typeof value === "number") return decimal(value);
  return String(value);
}

/**
 * The statements of a parsing expression with what each came to on `values`, in the contract's
 * shape: every let, then the result, with UTF-16 offsets into the expression.
 */
export function statementsOf(res) {
  if (!res.an || res.parseError || res.empty) return null;
  return res.an.stmts.map((st, index) => {
    const step = res.steps[index];
    const out = {
      index,
      kind: st.isLet ? "let" : "result",
      ...(st.isLet && st.letName ? { name: st.letName } : {}),
      code: st.code.trim(),
      start: st.s,
      end: st.e,
    };
    if (!step) return { ...out, error: "Not evaluated: an earlier statement failed" };
    if (step.error) return { ...out, error: step.error };
    return { ...out, value: typeof step.value === "number" ? decimal(step.value) : step.value };
  });
}

/** Statement values for a draft on a scenario's inputs, keyed by let name. */
export function terms(draft, inputs, base) {
  const ctx = context(draft.variables);
  const values = valuesFor(base ?? null, draft.variables, inputs);
  const res = execute(draft.expression, ctx, values);
  const out = new Map();
  (res.an?.stmts ?? []).forEach((st, i) => {
    if (!st.isLet || !st.letName) return;
    const step = res.steps[i];
    out.set(st.letName, step && !step.error ? step.value : null);
  });
  return { terms: out, run: run(draft, base ?? null, inputs) };
}

const near = (x, y) => x === y || (typeof x === "number" && typeof y === "number" && Math.abs(x - y) < 0.005);

/**
 * Why a scenario no longer matches: every input default, let, the formula, the policy and the
 * billed amount, once with the content the expectation was recorded on and once with the draft.
 */
export function diagnose(draft, recorded, scenario) {
  const inputs = scenario.variables ?? {};
  const A = terms(recorded, inputs);
  const B = terms(draft, inputs);
  const rows = [];
  draft.variables.forEach((v) => {
    if (!v.name || v.name in inputs) return;
    const was = recorded.variables.find((x) => x.name === v.name);
    const before = was ? was.defaultValue : undefined;
    const after = v.defaultValue;
    if (!near(before, after)) rows.push({ kind: "input", name: v.name, before: display(before), after: display(after) });
  });
  recorded.variables.forEach((v) => {
    if (v.name && !(v.name in inputs) && !draft.variables.some((x) => x.name === v.name)) {
      rows.push({ kind: "input", name: v.name, before: display(v.defaultValue), after: null });
    }
  });
  const names = [...new Set([...A.terms.keys(), ...B.terms.keys()])];
  const termRows = [];
  names.forEach((n) => {
    const before = A.terms.has(n) ? A.terms.get(n) : null;
    const after = B.terms.has(n) ? B.terms.get(n) : null;
    if (!near(after, before)) termRows.push({ kind: "term", name: n, before, after });
  });
  rows.push(...termRows.map((r) => ({ ...r, before: display(r.before), after: display(r.after) })));
  const rb = A.run.ok ? A.run.policy : null;
  const ra = B.run.ok ? B.run.policy : null;
  const rawBefore = rb ? rb.raw : null;
  const rawAfter = ra ? ra.raw : null;
  if (!near(rawBefore, rawAfter)) rows.push({ kind: "formula", name: "formula", before: display(rawBefore), after: display(rawAfter) });
  [
    ["min", "minimum"],
    ["max", "maximum"],
  ].forEach(([key, name]) => {
    if (!near(recorded.policy[key], draft.policy[key])) {
      rows.push({ kind: "policy", name, before: display(recorded.policy[key]), after: display(draft.policy[key]) });
    }
  });
  const billedBefore = rb ? rb.amount : null;
  const billedAfter = ra ? ra.amount : null;
  if (!near(billedBefore, billedAfter)) rows.push({ kind: "billed", name: "billed", before: display(billedBefore), after: display(billedAfter) });
  return {
    rows,
    termRows,
    exprChanged: recorded.expression.trim() !== draft.expression.trim(),
    policyChanged: JSON.stringify(recorded.policy) !== JSON.stringify(draft.policy),
    raw: { before: rawBefore, after: rawAfter },
    clamp: { before: rb && rb.clamp, after: ra && ra.clamp },
    billed: { before: billedBefore, after: billedAfter },
    error: B.run.ok ? null : B.run.error,
  };
}

/** Dollars the way the prototype writes them: $1,234.56. */
export function money(x, d = 2) {
  if (x == null || Number.isNaN(x)) return "—";
  return `${x < 0 ? "−" : ""}$${Math.abs(x).toLocaleString("en-US", { minimumFractionDigits: d, maximumFractionDigits: d })}`;
}

/** A value in prose: numbers plain, text quoted, empty as "empty". */
export function fmt(v) {
  if (v === null || v === undefined) return "empty";
  if (typeof v === "number") return Number.isInteger(v) ? v.toLocaleString("en-US") : String(+v.toFixed(4));
  if (typeof v === "boolean") return String(v);
  return `"${v}"`;
}
