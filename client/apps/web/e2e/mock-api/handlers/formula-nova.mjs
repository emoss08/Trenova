// Nova, the formula assistant, on the Formula Studio: the one-shot calls (explain the formula,
// describe its inputs, suggest scenarios, explain why a scenario broke, draft the reviewer's
// note) and the studio's page conversation, whose turns stream the way a real turn does and
// answer with a `propose_formula` draft edit. The words are the handoff's offline answers
// (design_handoff_formula_studio/design/ai.jsx); every amount in them is the mock engine's, and
// each one-shot answers after about 700ms so its loading state can be shot. With the `aiProviders`
// scenario at `none`, the one-shots refuse as the server does with no provider configured, and
// explaining a scenario falls back to the deterministic cause with `generated: false`.
import {
  analyze,
  applyPolicy,
  bill,
  context,
  decimal,
  diagnose,
  draftOf,
  evaluate,
  fmt,
  money,
  SCHEMA_NAMES,
  statementsOf,
  valuesFor,
} from "../fixtures/formula-engine.mjs";
import { FUNCTIONS, ORG, SHIPMENTS, TEAM, USER, VARIABLES } from "../fixtures/formula.mjs";
import {
  approvedVersion,
  business,
  contentOf,
  impact,
  invalid,
  normalizeContent,
  notFound,
  publicBacktest,
  reviewDiff,
  runScenarios,
  studio,
  templateOf,
} from "./formula.mjs";

const NOVA_DELAY_MS = Number(process.env.MOCK_FORMULA_NOVA_MS ?? 700);
const MAX_DESCRIPTION = 120;
const THREAD_PREFIX = "athr_formula_";

const AGENT = {
  id: "agdef_formula_assistant",
  name: "Formula assistant",
  description:
    "Helps write and explain the rating formulas that price freight, drafts and revises rate agreements, and proposes rate changes, rate sheets and fuel index prices for a person to approve.",
  template: "FormulaAssistant",
  icon: "sparkle",
  accent: "violet",
  systemKey: "formula_assistant",
  toolNames: [
    "formula_describe_schema",
    "formula_test_expression",
    "propose_formula",
    "formula_list_templates",
    "formula_get_template",
  ],
  starters: [
    { label: "Explain this formula", prompt: "Explain this formula, term by term" },
    { label: "Add a weekend fee", prompt: "Add a weekend pickup fee" },
    { label: "Suggest edge cases", prompt: "Suggest edge cases to test" },
    { label: "Why the minimum?", prompt: "Why did the short haul bill the minimum?" },
  ],
};

const noProvider = (state) => state.scenario.aiProviders === "none";
const refuseNoProvider = () => business("No AI provider is configured for the formula assistant");
const later = (payload, ms = NOVA_DELAY_MS) =>
  new Promise((resolve) => setTimeout(() => resolve(payload), ms));
const code = (name) => `\`${name}\``;

/** The load a one-shot prices on: the preview's sample values, or a saved shipment over the sample. */
function loadFor(body, draft) {
  const sample = body?.sampleValues ?? {};
  const shipment = body?.shipmentId ? SHIPMENTS.find((s) => s.id === body.shipmentId) : null;
  const schemaValues = Object.fromEntries(
    Object.entries(sample).filter(([name]) => SCHEMA_NAMES.has(name)),
  );
  const customValues = Object.fromEntries(
    Object.entries(sample).filter(([name]) => draft.variables.some((v) => v.name === name)),
  );
  return {
    base: shipment ? { ...shipment.values } : schemaValues,
    overrides: customValues,
    shipment,
  };
}

// --- Explain --------------------------------------------------------------------------------

/** The handoff's canned sentences, written with the draft's own defaults. */
function cannedExplanation(key, defaults) {
  const value = (name, fallback) => defaults[name] ?? fallback;
  switch (key) {
    case "let linehaul = baseRate * totalDistance":
      return "Linehaul is the lane's per-mile rate times the miles on the shipment.";
    case "let fuel = linehaul * (fuelSurchargePercent / 100)":
      return `Fuel adds a percentage of linehaul, ${fmt(value("fuelSurchargePercent", 18))}% unless the lane sets its own.`;
    case "let stops = max(totalStops - includedStops, 0) * stopFee":
      return `Each stop past the ${fmt(value("includedStops", 2))} included ones costs ${money(Number(value("stopFee", 50)))}. max(…, 0) keeps a load with fewer stops from going negative.`;
    case "linehaul + fuel + stops + (hasHazmat ? hazmatFee : 0)":
      return `The charge is the three parts plus the ${money(Number(value("hazmatFee", 175)))} hazmat fee when any commodity is hazardous. The minimum and rounding happen after this, in Charge policy.`;
    case "linehaul + fuel + stops":
      return "The charge is the three parts added together. The minimum and rounding happen after this, in Charge policy.";
    default:
      return null;
  }
}

function plainWords(statement) {
  const words = statement.code
    .replace(/^let\s+(\w+)\s*=\s*/, "")
    .replace(/\*/g, "×")
    .replace(/\//g, "÷")
    .replace(/\?\?/g, "or else")
    .replace(/\s+\?\s+/g, " → if so ")
    .replace(/\s+:\s+/g, ", otherwise ");
  return statement.kind === "let" ? `${statement.name} is ${words}.` : `The charge is ${words}.`;
}

/** One plain sentence per statement (novaLocalExplain). */
function explainStatements(statements, draft) {
  const defaults = Object.fromEntries(draft.variables.map((v) => [v.name, v.defaultValue]));
  return statements.map((statement) => ({
    statementIndex: statement.index,
    says:
      cannedExplanation(statement.code.replace(/\s+/g, " ").trim(), defaults) ??
      plainWords(statement),
  }));
}

function explainFormula(state, _path, body) {
  if (noProvider(state)) return refuseNoProvider();
  if (!String(body?.expression ?? "").trim())
    return invalid("expression", "Expression is required", "REQUIRED");
  const content = normalizeContent(body);
  const draft = draftOf(content);
  const { base, overrides } = loadFor(body, draft);
  const ctx = context(draft.variables);
  const values = valuesFor(base, draft.variables, overrides);
  const res = evaluate(content.expression, ctx, values);
  const statements = statementsOf(res) ?? [];
  const raw = res.ok && typeof res.value === "number" ? res.value : null;
  const applied = raw === null ? null : applyPolicy(raw, draft.policy);
  const bounded = draft.policy.min !== null || draft.policy.max !== null;
  return later({
    statements,
    items: explainStatements(statements, draft),
    rawAmount: decimal(raw),
    billedAmount: applied ? decimal(applied.amount) : null,
    guardrail:
      applied && bounded
        ? {
            applied: applied.clamp !== null,
            ...(applied.clamp ? { bound: applied.clamp.bound } : {}),
            rawAmount: decimal(raw),
            ...(draft.policy.min !== null ? { minCharge: decimal(draft.policy.min) } : {}),
            ...(draft.policy.max !== null ? { maxCharge: decimal(draft.policy.max) } : {}),
          }
        : null,
  });
}

// --- Describe inputs ------------------------------------------------------------------------

const INPUT_DESCRIPTIONS = {
  hazmatFee: "Fee added when any commodity on the load is hazardous, in dollars",
  weekendFee: "Fee added when the first pickup is on a Saturday or Sunday, in dollars",
  reeferFee: "Fee added when a commodity needs temperature control, in dollars",
  fuelSurchargePercent: "Fuel surcharge as a percent of linehaul",
  includedStops: "Stops the linehaul already covers",
  stopFee: "Fee for each stop past the included ones, in dollars",
  minimumCharge: "Least this template charges for a load, in dollars",
};

/** novaDescribe's offline answer: the name in words, and "in dollars" for a fee. */
function describeName(name) {
  if (INPUT_DESCRIPTIONS[name]) return INPUT_DESCRIPTIONS[name];
  const words = name
    .replace(/([A-Z])/g, " $1")
    .replace(/^./, (c) => c.toUpperCase())
    .replace(/ Fee$/, " fee")
    .trim();
  return (words + (/fee|charge|rate/i.test(name) ? ", in dollars" : "")).slice(0, MAX_DESCRIPTION);
}

function describeInputs(state, _path, body) {
  if (noProvider(state)) return refuseNoProvider();
  const content = normalizeContent(body);
  const descriptions = Object.fromEntries(
    content.variableDefinitions
      .filter((v) => !String(v.description ?? "").trim())
      .map((v) => [v.name, describeName(v.name)]),
  );
  return later({ descriptions });
}

// --- Suggest scenarios ----------------------------------------------------------------------

function edgePool(expression) {
  const pool = [
    {
      name: "Local move, 0 miles",
      description: "Nothing to rate per mile, so the minimum decides the charge.",
      variables: { totalDistance: 0, totalStops: 2 },
    },
    {
      name: "Five stops with hazmat",
      description: "Three stops past the included two, with the hazmat fee on top.",
      variables: { totalStops: 5, hasHazmat: true },
    },
    {
      name: "Long haul, 1,400 mi",
      description: "A long lane, where the per-mile terms carry the charge.",
      variables: { totalDistance: 1400, totalStops: 2 },
    },
    {
      name: "Single-stop drop",
      description: "Fewer stops than the formula includes; the stop charge must not go negative.",
      variables: { totalStops: 1 },
    },
  ];
  if (/isWeekendPickup|pickupDayOfWeek/.test(expression)) {
    pool.push({
      name: "Saturday pickup",
      description: "The first pickup falls on a weekend.",
      variables: { isWeekendPickup: true, pickupDayOfWeek: 6 },
    });
  }
  if (/requiresTemperatureControl|temperature/.test(expression)) {
    pool.push({
      name: "Reefer load",
      description: "A commodity needs temperature control.",
      variables: {
        requiresTemperatureControl: true,
        temperatureMin: 34,
        temperatureMax: 38,
        temperatureDifferential: 4,
      },
    });
  }
  pool.push({
    name: "Heavy load, 44,000 lb",
    description: "A load at the legal weight limit.",
    variables: { totalWeight: 44000, weight: 44000 },
  });
  return pool;
}

/** A scenario priced as a scenario run prices it: the billed amount, policy applied. */
function priced(draft, scenario) {
  const billed = bill(draft, scenario.variables);
  return {
    name: scenario.name,
    description: scenario.description,
    variables: scenario.variables,
    amount: billed.amount === null ? "" : decimal(billed.amount),
    valid: billed.error === null,
    error: billed.error ?? "",
  };
}

function suggestScenarios(state, _path, body) {
  if (noProvider(state)) return refuseNoProvider();
  if (!String(body?.expression ?? "").trim())
    return invalid("expression", "Expression is required", "REQUIRED");
  const content = normalizeContent(body);
  const draft = draftOf(content);
  const existing = Array.isArray(body?.existing) ? body.existing : [];
  const takenNames = new Set(
    existing.map((e) =>
      String(e.name ?? "")
        .trim()
        .toLowerCase(),
    ),
  );
  const takenInputs = new Set(existing.map((e) => JSON.stringify(e.variables ?? {})));
  const fresh = edgePool(content.expression).filter(
    (s) => !takenNames.has(s.name.toLowerCase()) && !takenInputs.has(JSON.stringify(s.variables)),
  );
  return later({ scenarios: fresh.slice(0, 5).map((s) => priced(draft, s)) });
}

// --- Explain a failing scenario -------------------------------------------------------------

function valueText(value) {
  if (value === null || value === undefined) return "empty";
  const n = Number(value);
  return Number.isNaN(n) ? `"${value}"` : fmt(n);
}

/** novaWhyLocal: the cause, from the engine's trace, in at most two sentences. */
function whyLocal(dg, label, expected) {
  const input = dg.rows.find((r) => r.kind === "input");
  const term = [...dg.termRows]
    .filter((r) => typeof r.before === "number" && typeof r.after === "number")
    .sort((x, y) => Math.abs(y.after - y.before) - Math.abs(x.after - x.before))[0];
  if (dg.error) return { why: `The formula fails on this load now: ${dg.error}`, verdict: "bug" };
  if (input && !dg.exprChanged) {
    return {
      why: `${code(input.name)}'s default changed from ${valueText(input.before)} to ${valueText(input.after)} after this scenario was recorded on ${label}${term ? `, which moves ${code(term.name)} from ${money(term.before)} to ${money(term.after)}` : ""}. The formula itself didn't change, so this looks like a rate update rather than a bug.`,
      verdict: "intended",
    };
  }
  const bound = (clamp) => (clamp.bound === "min" ? "minimum" : "maximum");
  if (dg.clamp.before && !dg.clamp.after) {
    return {
      why: `This load used to land on the ${bound(dg.clamp.before)}; it no longer does, so it bills the formula's ${money(dg.raw.after)}.`,
      verdict: "unclear",
    };
  }
  if (dg.clamp.after && !dg.clamp.before) {
    return {
      why: `This load now hits the ${bound(dg.clamp.after)} of ${money(dg.clamp.after.limit)}, so the formula's ${money(dg.raw.after)} is replaced.`,
      verdict: "unclear",
    };
  }
  if (term) {
    return {
      why: `${code(term.name)} went from ${money(term.before)} to ${money(term.after)} on this load${dg.exprChanged ? ", from an edit to the expression" : ""}.`,
      verdict: dg.exprChanged ? "unclear" : "intended",
    };
  }
  return {
    why: `Nothing in the template changed for this load since ${label}; the expected ${money(expected)} doesn't match what that version billed either, so the expectation itself is likely off.`,
    verdict: "intended",
  };
}

function explainScenario(state, path, body) {
  const t = templateOf(state, path);
  if (!t) return notFound("Formula template");
  const tc = t.testCases.find((c) => c.id === /\/test-cases\/([^/]+)\/explain/.exec(path)[1]);
  if (!tc) return notFound("Formula test case");
  if (!body?.candidate || typeof body.candidate.expression !== "string") {
    return invalid("candidate", "The editor's content is required", "REQUIRED");
  }
  const draft = draftOf(normalizeContent(body.candidate, contentOf(t)));
  const recorded = draftOf(tc.recordedContent);
  const dg = diagnose(draft, recorded, tc);
  const label = tc.recordedOnVersion > 0 ? `v${tc.recordedOnVersion}` : "the recorded draft";
  const expected = Number(tc.expectedAmount);
  const { why, verdict } = whyLocal(dg, label, expected);
  const actual = dg.billed.after;
  const recordedBy = tc.recordedById
    ? (() => {
        const person = [{ id: USER.id, name: USER.name }, ...TEAM].find(
          (p) => p.id === tc.recordedById,
        );
        return person ? { id: person.id, name: person.name } : null;
      })()
    : null;
  return later({
    verdict,
    why,
    generated: !noProvider(state),
    expected: decimal(expected),
    actual: actual === null ? null : decimal(actual),
    delta: actual === null ? null : decimal(actual - expected),
    baselineLabel: label,
    recordedOnVersion: tc.recordedOnVersion,
    recordedBy,
    ...(dg.error ? { error: dg.error } : {}),
    rows: dg.rows,
  });
}

// --- The reviewer's note --------------------------------------------------------------------

function changePhrases(diff) {
  const phrases = [];
  const entries = Object.entries(diff.changes);
  for (const [path, change] of entries) {
    const match = /^variableDefinitions\.(\d+)\.defaultValue$/.exec(path);
    if (match && typeof change.from === "number" && typeof change.to === "number") {
      const name = diff._current.variableDefinitions[Number(match[1])]?.name ?? "an input";
      phrases.push(
        `${change.to > change.from ? "raises" : "lowers"} ${code(name)} from ${fmt(change.from)} to ${fmt(change.to)}`,
      );
    }
  }
  const before = diff._base?.breakdownDefinitions ?? [];
  const after = diff._current.breakdownDefinitions;
  if (before.length === 0 && after.length > 0) {
    phrases.push(
      `itemizes the charge into ${joinPhrases(after.map((b) => b.label.toLowerCase()))} lines on the invoice`,
    );
  } else if (JSON.stringify(before) !== JSON.stringify(after)) {
    phrases.push("changes the breakdown lines");
  }
  if (diff.baseExpression && diff.baseExpression.trim() !== diff.currentExpression.trim())
    phrases.push("changes the expression");
  if (entries.some(([p]) => p === "minCharge"))
    phrases.push(
      `moves the minimum charge to ${diff._current.minCharge ? money(Number(diff._current.minCharge)) : "none"}`,
    );
  if (entries.some(([p]) => p === "maxCharge"))
    phrases.push(
      `moves the maximum charge to ${diff._current.maxCharge ? money(Number(diff._current.maxCharge)) : "none"}`,
    );
  return phrases;
}

const joinPhrases = (list) =>
  list.length <= 1 ? (list[0] ?? "") : `${list.slice(0, -1).join(", ")} and ${list.at(-1)}`;

function reviewNote(state, path) {
  const t = templateOf(state, path);
  if (!t) return notFound("Formula template");
  if (noProvider(state)) return refuseNoProvider();
  const diff = {
    ...reviewDiff(t),
    _current: contentOf(t),
    _base: approvedVersion(t)?.content ?? null,
  };
  const scenarios = runScenarios(t, contentOf(t));
  const scenarioLine =
    scenarios.total === 0
      ? "It has no scenarios yet."
      : scenarios.failed === 0
        ? `All ${scenarios.total} scenarios pass against this version.`
        : `${scenarios.failed} of ${scenarios.total} scenarios fail against this version.`;
  if (!diff.hasApprovedBase) {
    return later({
      note: `This is the first version of ${t.name}, so nothing in production rates with it yet. ${scenarioLine}`,
      generated: true,
      backtest: null,
    });
  }
  const summary = publicBacktest(impact(t)).summary;
  const phrases = changePhrases(diff);
  const what = phrases.length
    ? `${joinPhrases(phrases).replace(/^./, (c) => c.toUpperCase())}.`
    : `Nothing that prices a load changed since v${diff.baseVersion}.`;
  const delta = Number(summary.totalDelta);
  const pct = Number(summary.totalDeltaPct);
  const effect = `Against ${summary.shipmentCount} recent shipments, ${summary.changedCount} would rate differently (${delta >= 0 ? "+" : "−"}${money(Math.abs(delta), 0)} in total, ${pct >= 0 ? "+" : ""}${pct.toFixed(1)}%).`;
  return later({ note: `${what} ${effect} ${scenarioLine}`, generated: true, backtest: summary });
}

// --- The page conversation ------------------------------------------------------------------

function nova(state) {
  state.formulaNova ??= { threads: new Map(), byTemplate: new Map(), turns: new Map(), nextId: 1 };
  return state.formulaNova;
}

function nextId(n, prefix) {
  const id = `${prefix}${n.nextId}`;
  n.nextId += 1;
  return id;
}

function newThread(state, templateId) {
  const n = nova(state);
  const now = state.now();
  const id = nextId(n, THREAD_PREFIX);
  const entry = {
    thread: {
      id,
      businessUnitId: USER.businessUnitId,
      organizationId: ORG.id,
      userId: USER.id,
      agentDefinitionId: AGENT.id,
      title: "",
      status: "Active",
      lastMessageAt: 0,
      lastReadAt: now,
      origin: "Formula",
      pinned: false,
      pinnedFacts: [],
      subjectType: templateId ? "FormulaTemplate" : "",
      subjectId: templateId ?? "",
      canContinue: true,
      version: 1,
      createdAt: now,
      updatedAt: now,
    },
    templateId,
    messages: [],
    artifacts: [],
  };
  n.threads.set(id, entry);
  if (templateId) n.byTemplate.set(templateId, id);
  return entry;
}

/** The formula page's conversations, for the rail's list (which leaves them out, as it does for real). */
export function formulaThreads(state) {
  return [...(state.formulaNova?.threads.values() ?? [])].map((entry) => entry.thread);
}

/** Opens the conversation about a saved template, or a fresh one for a template not yet saved. */
function openThread(state, _path, body) {
  const raw = String(body?.templateId ?? "").trim();
  if (raw) {
    if (!studio(state).templates.has(raw)) return notFound("Formula template");
    const known = nova(state).byTemplate.get(raw);
    const entry = known ? nova(state).threads.get(known) : newThread(state, raw);
    return { thread: entry.thread, agent: AGENT };
  }
  return { thread: newThread(state, null).thread, agent: AGENT };
}

const threadIdOf = (path) => /\/threads\/(athr_formula_\d+)/.exec(path)?.[1];

function entryOf(state, path) {
  return nova(state).threads.get(threadIdOf(path)) ?? null;
}

function message(entry, role, content, createdAt, extra = {}) {
  return {
    id: `amsg_${entry.thread.id.slice(5)}_${entry.messages.length + 1}`,
    threadId: entry.thread.id,
    sequence: entry.messages.length + 1,
    role,
    kind: "Message",
    content,
    toolFailed: false,
    refused: false,
    inputTokens: 0,
    outputTokens: 0,
    model: role === "Assistant" ? "mock" : "",
    createdAt,
    ...extra,
  };
}

/** The editor as the turn carries it (the page draft), or the saved template when it carries none. */
function editorOf(state, entry, body) {
  const formula = body?.context?.draft?.formula;
  const t = entry.templateId ? studio(state).templates.get(entry.templateId) : null;
  const saved = t ? draftOf(contentOf(t)) : null;
  if (formula && typeof formula.expression === "string") {
    const policy = formula.policy
      ? {
          min: formula.policy.minCharge == null ? null : Number(formula.policy.minCharge),
          max: formula.policy.maxCharge == null ? null : Number(formula.policy.maxCharge),
          mode: formula.policy.roundingMode || "HalfUp",
          precision: Number.isInteger(formula.policy.roundingPrecision)
            ? formula.policy.roundingPrecision
            : 2,
        }
      : (saved?.policy ?? DEFAULT_POLICY);
    return {
      schemaId: formula.schemaId || "shipment",
      name: formula.name ?? t?.name ?? "",
      description: formula.description ?? t?.description ?? "",
      expression: formula.expression,
      variables: (formula.variables ?? []).map((v) => ({
        name: v.name,
        type: v.type ?? "Number",
        description: v.description ?? "",
        defaultValue: v.defaultValue ?? null,
      })),
      breakdown: (formula.breakdown ?? saved?.breakdown ?? []).map((b) => ({
        name: b.name,
        label: b.label ?? "",
        expression: b.expression,
      })),
      policy,
    };
  }
  return {
    schemaId: "shipment",
    name: t?.name ?? "",
    description: t?.description ?? "",
    expression: t?.expression ?? "",
    variables: (t?.variableDefinitions ?? []).map((v) => ({
      name: v.name,
      type: v.type,
      description: v.description ?? "",
      defaultValue: v.defaultValue ?? null,
    })),
    breakdown: (saved?.breakdown ?? []).map((b) => ({ ...b })),
    policy: saved?.policy ?? DEFAULT_POLICY,
  };
}

const DEFAULT_POLICY = { min: null, max: null, mode: "HalfUp", precision: 2 };

function engineDraft(editor, policy = editor.policy ?? DEFAULT_POLICY) {
  return { expression: editor.expression, variables: editor.variables, breakdown: [], policy };
}

/** pagedraft.FormulaPolicy from the engine's shape: the bounds as decimal strings. */
function policyOut(policy) {
  return {
    minCharge: policy.min === null ? null : decimal(policy.min),
    maxCharge: policy.max === null ? null : decimal(policy.max),
    roundingMode: policy.mode,
    roundingPrecision: policy.precision,
  };
}

/** pagedraft.FormulaProposal: the formula, its inputs, the policy, the engine's check and the loads it priced. */
function proposal(editor, next, explanation, extra = {}) {
  const expression = next.expression ?? editor.expression;
  const variables = next.variables ?? editor.variables;
  const policy = next.policy ?? editor.policy ?? DEFAULT_POLICY;
  const draft = engineDraft({ expression, variables }, policy);
  const defaults = Object.fromEntries(
    variables
      .filter((v) => v.defaultValue !== null && v.defaultValue !== undefined)
      .map((v) => [v.name, v.defaultValue]),
  );
  const sample = bill(draft, {});
  const loads = [
    { name: "Sample load", description: "742 mi, 3 stops, a Tuesday pickup", variables: {} },
    ...["shp_fs_240811", "shp_fs_240794", "shp_fs_240741"].map((id) => {
      const s = SHIPMENTS.find((x) => x.id === id);
      return { name: s.pro, description: `${s.lane} · ${s.meta}`, variables: s.values };
    }),
  ];
  return {
    schemaId: editor.schemaId,
    ...extra,
    expression,
    variables,
    policy: policyOut(policy),
    explanation,
    check: sample.error
      ? { valid: false, error: sample.error }
      : { valid: true, result: decimal(sample.amount) },
    scenarios: loads.map((load) => {
      const billed = bill(draft, load.variables);
      return {
        name: load.name,
        description: load.description,
        variables: { ...defaults, ...load.variables },
        ...(billed.error ? {} : { amount: decimal(billed.amount) }),
        valid: billed.error === null,
        ...(billed.error ? { error: billed.error } : {}),
      };
    }),
  };
}

/**
 * The editor's formula with a new input and the charge it pays for bound on
 * its own `let` line, the way Nova writes a formula, added to the sum on the
 * result line. A one-line formula becomes `let linehaul = …;` first.
 */
function extend(editor, name, defaultValue, description, charge) {
  const variables = editor.variables.some((v) => v.name === name)
    ? editor.variables
    : [...editor.variables, { name, type: "Number", description, defaultValue }];
  const bind = `let ${charge.name} = ${charge.expression};`;
  if (!editor.expression.trim()) {
    return {
      expression: `let linehaul = baseRate * totalDistance;\n${bind}\nlinehaul + ${charge.name}`,
      variables,
    };
  }
  const lines = editor.expression.replace(/\s+$/, "").split("\n");
  const result = lines.pop();
  if (lines.length === 0 && !/\blet\b/.test(result) && !/[+-]/.test(result)) {
    return {
      expression: `let linehaul = ${result.trim()};\n${bind}\nlinehaul + ${charge.name}`,
      variables,
    };
  }
  return { expression: [...lines, bind, `${result} + ${charge.name}`].join("\n"), variables };
}

/** The editor's formula without every term that reads a name, and without the input itself. */
function removeTerm(editor, name) {
  let expression = editor.expression;
  // A `let` that reads the name goes, and so does the binding's own name
  // wherever the sum adds it.
  const bound = new RegExp(String.raw`\blet\s+(\w+)\s*=[^;]*\b${name}\b[^;]*;\s*\n?`, "g");
  for (const match of expression.matchAll(bound)) {
    expression = expression.replace(new RegExp(String.raw`\s*\+\s*\b${match[1]}\b`, "g"), "");
  }
  expression = expression.replace(bound, "");
  const pattern = new RegExp(String.raw`\s*\+\s*\(?[^+\n;]*\b${name}\b[^+\n;]*\)?`, "g");
  expression = expression.replace(pattern, "").replace(/[ \t]*\+[ \t]*/g, " + ");
  const variables = editor.variables.filter((v) => v.name !== name);
  return { expression, variables };
}

function levenshtein(a, b) {
  const row = Array.from({ length: b.length + 1 }, (_, j) => j);
  for (let i = 1; i <= a.length; i++) {
    let prev = row[0];
    row[0] = i;
    for (let j = 1; j <= b.length; j++) {
      const tmp = row[j];
      row[j] = Math.min(row[j] + 1, row[j - 1] + 1, prev + (a[i - 1] === b[j - 1] ? 0 : 1));
      prev = tmp;
    }
  }
  return row[b.length];
}

function closest(name, candidates) {
  let best = null;
  for (const candidate of candidates) {
    const d = levenshtein(name.toLowerCase(), candidate.toLowerCase());
    if (d <= 3 && (!best || d < best.d)) best = { name: candidate, d };
  }
  return best?.name ?? null;
}

/** Fix with Nova: guard each unguarded optional field and correct each misspelt name. */
function fixOf(editor) {
  const an = analyze(editor.expression, context(editor.variables));
  const edits = [];
  const notes = [];
  for (const issue of an.issues) {
    if (issue.kind === "nullable") {
      edits.push({ s: issue.s, e: issue.e, text: issue.fix });
      notes.push(
        `${code(issue.name)} can be empty on a shipment, so I guarded it with ${code("?? 0")}`,
      );
    } else if (issue.kind === "fn") {
      const fix = closest(
        issue.name,
        FUNCTIONS.map((f) => f.name),
      );
      if (fix) {
        edits.push({ s: issue.s, e: issue.e, text: fix });
        notes.push(`${code(issue.name)} isn't a formula function; I used ${code(fix)}`);
      }
    } else if (issue.kind === "var") {
      const fix = closest(issue.name, [
        ...VARIABLES.map((v) => v.name),
        ...editor.variables.map((v) => v.name),
      ]);
      if (fix) {
        edits.push({ s: issue.s, e: issue.e, text: fix });
        notes.push(
          `${code(issue.name)} isn't a shipment variable or one of the inputs; I used ${code(fix)}`,
        );
      }
    }
  }
  if (edits.length === 0) return null;
  let expression = editor.expression;
  for (const edit of [...edits].sort((a, b) => b.s - a.s)) {
    expression = expression.slice(0, edit.s) + edit.text + expression.slice(edit.e);
  }
  return { expression, notes };
}

function tool(name, effect, args, summary, result) {
  return { name, effect, args, summary, content: JSON.stringify(result ?? { summary }) };
}

/** A dollar figure or a bare number in the question, such as "$250" or "300 dollars". */
function amountIn(q) {
  const m =
    /\$\s?(\d[\d,]*(?:\.\d+)?)|(\d[\d,]*(?:\.\d+)?)\s*(?:dollars|usd)\b|\b(\d[\d,]*(?:\.\d+)?)\b/.exec(
      q,
    );
  const raw = m?.[1] ?? m?.[2] ?? m?.[3];
  return raw === undefined ? null : Number(raw.replace(/,/g, ""));
}

/** The input the question names, by its identifier or the words in it ("the hazmat fee" → hazmatFee). */
function inputIn(q, editor) {
  for (const v of editor.variables) {
    if (q.includes(v.name.toLowerCase())) return v;
  }
  for (const v of editor.variables) {
    const words = v.name
      .replace(/([A-Z])/g, " $1")
      .toLowerCase()
      .trim()
      .split(/\s+/);
    if (words.length > 1 && words.every((w) => q.includes(w))) return v;
  }
  return null;
}

const FEES = [
  {
    key: /weekend/,
    name: "weekendFee",
    def: 125,
    desc: "Fee when the first pickup is on a weekend",
    charge: { name: "weekendCharge", expression: "isWeekendPickup ? weekendFee : 0" },
    line: {
      name: "weekend",
      label: "Weekend pickup",
      expression: "isWeekendPickup ? weekendFee : 0",
    },
    says: "a weekend fee as its own input so lanes can override it. It applies when the first pickup is on a Saturday or Sunday",
  },
  {
    key: /reefer|temperature|refrigerat/,
    name: "reeferFee",
    def: 225,
    desc: "Fee when a commodity needs temperature control",
    charge: { name: "reeferCharge", expression: "requiresTemperatureControl ? reeferFee : 0" },
    line: {
      name: "reefer",
      label: "Temperature control",
      expression: "requiresTemperatureControl ? reeferFee : 0",
    },
    says: "a reefer fee that applies only when a commodity needs temperature control",
  },
  {
    key: /hazmat|hazardous/,
    name: "hazmatFee",
    def: 175,
    desc: "Fee when any commodity on the load is hazardous",
    charge: { name: "hazmatCharge", expression: "hasHazmat ? hazmatFee : 0" },
    line: { name: "hazmat", label: "Hazmat", expression: "hasHazmat ? hazmatFee : 0" },
    says: "a hazmat fee that applies when any commodity on the load is hazardous",
  },
  {
    key: /liftgate|lift gate/,
    name: "liftgateFee",
    def: 85,
    desc: "Fee when a liftgate is needed at a stop",
    charge: { name: "liftgateCharge", expression: "requiresLiftgate ? liftgateFee : 0" },
    line: { name: "liftgate", label: "Liftgate", expression: "requiresLiftgate ? liftgateFee : 0" },
    says: "a liftgate fee as a yes/no input lanes can set",
    extra: {
      name: "requiresLiftgate",
      type: "Boolean",
      def: false,
      desc: "Whether a liftgate is needed at a stop; no shipment field marks it",
    },
  },
  {
    key: /fuel/,
    name: "fuelSurchargePercent",
    def: 18,
    desc: "Fuel surcharge as a percent of linehaul",
    charge: {
      name: "fuelSurcharge",
      expression: "baseRate * totalDistance * (fuelSurchargePercent / 100)",
    },
    line: {
      name: "fuel",
      label: "Fuel surcharge",
      expression: "baseRate * totalDistance * (fuelSurchargePercent / 100)",
    },
    says: "a fuel surcharge as a percentage of linehaul, with the percent as an input",
  },
  {
    key: /stop/,
    name: "stopFee",
    def: 50,
    desc: "Fee for each stop past the included ones",
    charge: { name: "stopCharge", expression: "max(totalStops - includedStops, 0) * stopFee" },
    line: {
      name: "stops",
      label: "Additional stops",
      expression: "max(totalStops - includedStops, 0) * stopFee",
    },
    says: "a per-stop fee for every stop past the included ones, both as inputs",
    extra: { name: "includedStops", def: 2, desc: "Stops the linehaul already covers" },
  },
];

/** What the turn will say and do, decided from the question and the editor. */
function planTurn(state, entry, body) {
  const question = String(body.content ?? "");
  const q = question.toLowerCase();
  const editor = editorOf(state, entry, body);
  const lines = editor.expression.split("\n").filter((l) => l.trim()).length;
  const reading = lines
    ? `Reading the editor: ${lines} lines, ${editor.variables.length} inputs.`
    : "Reading the editor: it is empty.";
  const schemaStep = tool(
    "formula_describe_schema",
    "lookup",
    { schemaId: editor.schemaId },
    `${VARIABLES.length} shipment variables, ${FUNCTIONS.length} functions`,
  );
  const priceStep = (n) =>
    tool("formula_test_expression", "lookup", { scenarios: n }, `Priced ${n} loads`);
  const propose = (next, explanation, reply, extra = {}) => {
    const formula = proposal(editor, next, explanation, extra);
    const sample = formula.check.valid
      ? ` The engine prices the sample load at ${money(Number(formula.check.result))}.`
      : ` The engine can't run it yet: ${formula.check.error}.`;
    return {
      reasoning: reading,
      tools: [schemaStep, priceStep(formula.scenarios.length)],
      proposal: formula,
      reply: reply + sample,
    };
  };
  const policy = editor.policy ?? DEFAULT_POLICY;
  const draft = engineDraft(editor, policy);
  const has = (re) => re.test(q);

  if (has(/edge|scenario|test case/)) {
    const loads = edgePool(editor.expression)
      .slice(0, 4)
      .map((s) => ({ ...s, billed: bill(draft, s.variables) }));
    const parts = loads.map(
      (l) =>
        `${l.name.charAt(0).toLowerCase()}${l.name.slice(1)} ${l.billed.error ? `fails (${l.billed.error})` : `bills ${money(l.billed.amount)}`}`,
    );
    return {
      reasoning: reading,
      tools: [priceStep(loads.length)],
      reply: `Four loads sit on this formula's edges, priced by the formula engine: ${joinPhrases(parts)}. Add the ones you want to keep from Suggest edge cases under Scenarios, and they'll run on every edit.`,
    };
  }

  if (has(/\bfix\b|error|broken|fails|doesn't work|does not work/)) {
    const fixed = fixOf(editor);
    if (!fixed) {
      return {
        reasoning: reading,
        tools: [schemaStep],
        reply:
          "I don't see anything to fix: the formula compiles and every optional field is guarded.",
      };
    }
    return propose(
      { expression: fixed.expression },
      `${joinPhrases(fixed.notes)}.`,
      `${joinPhrases(fixed.notes).replace(/^./, (c) => c.toUpperCase())}.`,
    );
  }

  // The charge policy: a minimum, a maximum, or the rounding.
  if (
    has(/minimum|floor|at least|no less than/) &&
    has(/set|add|make|change|raise|lower|put|use|remove|drop|clear|\$|\d/)
  ) {
    if (has(/remove|drop|clear|no minimum/)) {
      return propose(
        { policy: { ...policy, min: null } },
        "Removes the minimum charge; the formula's own amount bills.",
        "I removed the minimum from the charge policy, so loads bill what the formula comes to.",
      );
    }
    const amount = amountIn(q);
    if (amount !== null) {
      return propose(
        { policy: { ...policy, min: amount } },
        `Raises any load under ${money(amount)} to ${money(amount)}, in the charge policy rather than the expression.`,
        `I set the minimum charge to ${money(amount)} in the charge policy, so the expression stays a pure rate and the floor shows on the rate trace.`,
      );
    }
  }
  if (
    has(/maximum|cap\b|ceiling|at most|no more than/) &&
    has(/set|add|make|change|raise|lower|put|use|remove|drop|clear|\$|\d/)
  ) {
    if (has(/remove|drop|clear|no maximum|no cap/)) {
      return propose(
        { policy: { ...policy, max: null } },
        "Removes the maximum charge.",
        "I removed the maximum from the charge policy.",
      );
    }
    const amount = amountIn(q);
    if (amount !== null) {
      return propose(
        { policy: { ...policy, max: amount } },
        `Caps the charge at ${money(amount)} in the charge policy.`,
        `I capped the charge at ${money(amount)} in the charge policy; anything the formula comes to above that bills ${money(amount)}.`,
      );
    }
  }
  if (
    has(/round/) &&
    has(/up|down|even|nearest|dollar|cent|whole|place|decimal|no rounding|don't round|do not round/)
  ) {
    const mode = has(/no rounding|don't round|do not round/)
      ? "None"
      : has(/\bup\b/)
        ? "Up"
        : has(/\bdown\b/)
          ? "Down"
          : has(/even|banker/)
            ? "HalfEven"
            : "HalfUp";
    const precision =
      has(/whole|dollar/) && !has(/cent/)
        ? 0
        : /(\d)\s*(?:decimal|place)/.exec(q)?.[1]
          ? Number(/(\d)\s*(?:decimal|place)/.exec(q)[1])
          : policy.precision;
    const words = {
      None: "not rounded",
      Up: "rounded up",
      Down: "rounded down",
      HalfEven: "rounded half even",
      HalfUp: "rounded half up",
    }[mode];
    return propose(
      { policy: { ...policy, mode, precision } },
      `The billed amount is ${words}${mode === "None" ? "" : ` to ${precision} ${precision === 1 ? "place" : "places"}`}, in the charge policy.`,
      `I set rounding to ${words}${mode === "None" ? "" : ` to ${precision} ${precision === 1 ? "decimal place" : "decimal places"}`} in the charge policy; the expression itself never rounds.`,
    );
  }

  // An input's default: "set the hazmat fee to 200", "change fuelSurchargePercent to 22".
  if (has(/\b(set|change|make|raise|lower|update|bump|default)\b/) && editor.variables.length > 0) {
    const input = inputIn(q, editor);
    const amount = amountIn(q.replace(/\b(the|to|from|at)\b/g, " "));
    if (input && amount !== null && !has(/minimum|maximum|cap\b/)) {
      const variables = editor.variables.map((v) =>
        v.name === input.name
          ? { ...v, defaultValue: input.type === "Boolean" ? amount !== 0 : amount }
          : v,
      );
      return propose(
        { variables },
        `Sets ${code(input.name)}'s default to ${fmt(amount)}; lanes and rate agreements can still override it.`,
        `I changed ${code(input.name)}'s default from ${fmt(input.defaultValue ?? 0)} to ${fmt(amount)}.`,
      );
    }
  }

  // Taking a term out: "remove the hazmat fee", "drop stopFee".
  if (has(/\b(remove|drop|delete|take out|get rid of|without)\b/) && editor.variables.length > 0) {
    const input =
      inputIn(q, editor) ??
      FEES.find((fee) => fee.key.test(q) && editor.variables.some((v) => v.name === fee.name));
    const name = input?.name;
    if (name) {
      const next = removeTerm(editor, name);
      const breakdown = editor.breakdown.filter(
        (b) => !new RegExp(String.raw`\b${name}\b`).test(b.expression),
      );
      return propose(
        { ...next },
        `Removes ${code(name)} and the term that read it; the input is dropped too.`,
        `I took ${code(name)} out of the formula and removed the input.`,
        breakdown.length !== editor.breakdown.length ? { breakdown } : {},
      );
    }
  }

  // Renaming the template.
  if (has(/\b(rename|call it|name it|title)\b/)) {
    const m =
      /(?:rename(?: it| this| the template)?(?: to)?|call it|name it|title(?: it)?)\s+["“]?([^"”]+?)["”]?\s*$/.exec(
        question.trim(),
      );
    if (m) {
      const name = m[1].trim().slice(0, 100);
      return propose({}, `Names the template ${name}.`, `I renamed the template to “${name}”.`, {
        name,
        description: editor.description,
      });
    }
  }

  // Itemizing: "add a breakdown", "itemize the charge".
  if (has(/breakdown|itemi[sz]e|line items|invoice lines/)) {
    const an = analyze(editor.expression, context(editor.variables));
    const lets = an.stmts.filter((st) => st.isLet && st.letName);
    const lines = lets.length
      ? lets.map((st) => ({
          name: st.letName,
          label: st.letName.replace(/([A-Z])/g, " $1").replace(/^./, (c) => c.toUpperCase()),
          expression: st.code.replace(/^let\s+\w+\s*=\s*/, ""),
        }))
      : [
          {
            name: "linehaul",
            label: "Linehaul",
            expression: editor.expression.trim() || "baseRate * totalDistance",
          },
        ];
    return propose(
      {},
      `Itemizes the charge into ${joinPhrases(lines.map((l) => l.label.toLowerCase()))}, one invoice line each.`,
      `I itemized the charge into ${lines.length} invoice ${lines.length === 1 ? "line" : "lines"}: ${joinPhrases(lines.map((l) => l.label.toLowerCase()))}. The lines add up to the formula's result.`,
      { breakdown: lines },
    );
  }

  // Adding a fee or a surcharge, or drafting from scratch.
  const fee = FEES.find((f) => f.key.test(q));
  if (
    fee &&
    has(/add|include|charge|with/) &&
    editor.variables.some((v) => v.name === fee.name) &&
    new RegExp(String.raw`\b${fee.name}\b`).test(editor.expression)
  ) {
    return {
      reasoning: reading,
      tools: [schemaStep],
      reply: `The formula already charges ${code(fee.name)}${fee.charge.expression.includes("?") ? ` when ${code(fee.charge.expression.match(/^(\w+) \?/)[1])} is true` : ""}, with a default of ${money(fee.def)}. Say what to change about it — the amount, when it applies, or its own invoice line — and I'll make the change.`,
    };
  }
  if (
    fee &&
    (has(/add|write|draft|create|build|include|charge|with/) || !editor.expression.trim())
  ) {
    let next = extend(editor, fee.name, fee.def, fee.desc, fee.charge);
    if (fee.extra && !next.variables.some((v) => v.name === fee.extra.name)) {
      next = {
        ...next,
        variables: [
          ...next.variables,
          {
            name: fee.extra.name,
            type: fee.extra.type ?? "Number",
            description: fee.extra.desc,
            defaultValue: fee.extra.def,
          },
        ],
      };
    }
    const extra = editor.expression.trim()
      ? editor.breakdown.length && !editor.breakdown.some((b) => b.name === fee.line.name)
        ? { breakdown: [...editor.breakdown, fee.line] }
        : {}
      : {
          name: `Per mile with ${fee.line.label.toLowerCase()}`,
          description: `Per-mile linehaul with ${fee.says.split(" that ")[0].split(" as ")[0]}.`,
          templateType: "FreightCharge",
          breakdown: [
            { name: "linehaul", label: "Linehaul", expression: "baseRate * totalDistance" },
            fee.line,
          ],
        };
    return propose(
      next,
      `Adds ${code(fee.charge.name)}${fee.charge.expression.includes("?") ? ` when ${code(fee.charge.expression.match(/^(\w+) \?/)[1])} is true` : ""}, on its own line.`,
      `I added ${fee.says}.`,
      extra,
    );
  }

  if (
    has(/per mile|per-mile|mileage/) &&
    (has(/write|draft|create|build|start|make/) || !editor.expression.trim())
  ) {
    const minimum = has(/minimum/) ? (amountIn(q.replace(/\d+\s*%/, "")) ?? 250) : null;
    const fuel = has(/fuel/);
    const expression = fuel
      ? "let linehaul = baseRate * totalDistance;\nlet fuel = linehaul * (fuelSurchargePercent / 100);\nlinehaul + fuel"
      : "baseRate * totalDistance";
    const variables = fuel
      ? [
          {
            name: "fuelSurchargePercent",
            type: "Number",
            description: "Fuel surcharge as a percent of linehaul",
            defaultValue: Number(/(\d+)\s*%/.exec(q)?.[1] ?? 18),
          },
        ]
      : [];
    return propose(
      { expression, variables, policy: { ...policy, min: minimum } },
      `Linehaul is the lane's per-mile rate times the miles${fuel ? ", plus a fuel surcharge as a percent of it" : ""}.${minimum ? ` Loads under ${money(minimum)} bill the minimum.` : ""}`,
      `I drafted a per-mile method${fuel ? " with a fuel surcharge" : ""}${minimum ? ` and a ${money(minimum)} minimum in the charge policy` : ""}.`,
      {
        name: fuel ? "Per mile with fuel surcharge" : "Per mile",
        description: "Per-mile linehaul on the lane's base rate.",
        templateType: "FreightCharge",
      },
    );
  }
  if (
    has(/cwt|hundredweight|per hundred/) &&
    (has(/write|draft|create|build|start|make/) || !editor.expression.trim())
  ) {
    return propose(
      {
        expression: "let cwt = roundUp(totalWeight / 100, 0);\ncwt * ratePerCwt",
        variables: [
          {
            name: "ratePerCwt",
            type: "Number",
            description: "Rate per hundredweight",
            defaultValue: 12.5,
          },
        ],
        policy,
      },
      "Weight is rounded up to the next hundredweight, then multiplied by the rate per CWT.",
      "I drafted a per-hundredweight method: the weight rounds up to the next 100 lb and multiplies by the rate.",
      {
        name: "Per CWT",
        description: "Per hundredweight on the shipment's total weight.",
        templateType: "FreightCharge",
      },
    );
  }

  if (has(/minimum|floor|why|250/) && editor.expression.trim()) {
    const short = bill(draft, { totalDistance: 60, totalStops: 2 });
    const sample = bill(draft, {});
    if (short.error)
      return {
        reasoning: reading,
        tools: [],
        reply: `The formula fails on a 60-mile load: ${short.error}`,
      };
    const clampWord = short.clamp ? (short.clamp.bound === "min" ? "minimum" : "maximum") : null;
    return {
      reasoning: reading,
      tools: [tool("formula_test_expression", "lookup", { totalDistance: 60 }, "Priced 1 load")],
      reply: short.clamp
        ? `On a 60-mile load the formula produces ${money(short.raw)}, under the ${money(short.clamp.limit)} ${clampWord} in Charge policy, so the charge is raised to it. Breakdown lines still add up to the raw amount.`
        : `On a 60-mile load the formula produces ${money(short.raw)}, inside the charge policy, so it bills ${money(short.amount)}. The sample load bills ${sample.error ? "nothing" : money(sample.amount)}.`,
    };
  }
  if (has(/explain|what does|how does|term|walk me/) && editor.expression.trim()) {
    const res = evaluate(
      editor.expression,
      context(editor.variables),
      valuesFor(null, editor.variables, null),
    );
    const statements = statementsOf(res) ?? [];
    return {
      reasoning: reading,
      tools: [],
      reply: explainStatements(statements, draft)
        .map((item) => item.says)
        .join(" "),
    };
  }
  if (has(/what would|how much|charge for|price for|cost for/) && editor.expression.trim()) {
    const miles = amountIn(q);
    const load = miles !== null ? { totalDistance: miles } : {};
    const billed = bill(draft, load);
    return {
      reasoning: reading,
      tools: [priceStep(1)],
      reply: billed.error
        ? `The engine can't price that load: ${billed.error}`
        : `For ${miles !== null ? `a ${fmt(miles)}-mile load` : "the sample load"} the formula comes to ${money(billed.raw)}${billed.clamp ? `, and the ${billed.clamp.bound === "min" ? "minimum" : "maximum"} makes it ${money(billed.amount)}` : `, billed as ${money(billed.amount)} after rounding`}.`,
    };
  }
  return {
    reasoning: reading,
    tools: [],
    reply:
      "I can change the formula, its inputs or the charge policy as you describe it — add a fee, set a minimum, change a default, itemize the invoice — and explain or test what is there. Every amount I quote comes from the formula engine.",
  };
}

function startTurn(state, path, body) {
  const entry = entryOf(state, path);
  if (!entry) return notFound("Conversation");
  const content = String(body?.content ?? "").trim();
  if (!content) return invalid("content", "Message cannot be empty", "REQUIRED");
  const n = nova(state);
  const now = state.now();
  const user = message(
    entry,
    "User",
    content,
    now,
    body?.context ? { pageContext: body.context } : {},
  );
  entry.messages.push(user);
  if (!entry.thread.title) entry.thread.title = content.slice(0, 80);
  entry.thread.lastMessageAt = now;
  entry.thread.updatedAt = now;
  const turnId = nextId(n, "aturn_formula_");
  n.turns.set(turnId, {
    id: turnId,
    threadId: entry.thread.id,
    user,
    plan: planTurn(state, entry, { ...body, content }),
    done: false,
    startedAt: now,
  });
  return {
    turnId,
    threadId: entry.thread.id,
    streamUrl: `/assistant/turns/${turnId}/stream/`,
    status: "Running",
    thread: entry.thread,
  };
}

/** Writes what the turn did into the conversation, once: the tool steps, the reply and the artifact. */
function settle(state, turn) {
  if (turn.settled) return turn.settled;
  const entry = nova(state).threads.get(turn.threadId);
  const at = turn.startedAt + 3;
  const calls = turn.plan.tools.map((step, index) => ({
    ...step,
    id: `call_${turn.id.slice(6)}_${index + 1}`,
  }));
  let proposeCall = null;
  if (turn.plan.proposal) {
    proposeCall = {
      id: `call_${turn.id.slice(6)}_${calls.length + 1}`,
      name: "propose_formula",
      effect: "present",
      args: {
        expression: turn.plan.proposal.expression,
        explanation: turn.plan.proposal.explanation,
      },
      summary: "Proposed formula",
    };
    const edit = { surface: "formula", action: "propose_formula", formula: turn.plan.proposal };
    proposeCall.content = JSON.stringify({
      draft: edit,
      note: "The formula is in front of the person with its sample loads.",
    });
    proposeCall.edit = edit;
    calls.push(proposeCall);
  }
  const written = [];
  if (calls.length > 0) {
    const asking = message(entry, "Assistant", "", at, {
      toolCalls: calls.map((c) => ({
        id: c.id,
        name: c.name,
        arguments: c.args,
        effect: c.effect,
      })),
    });
    entry.messages.push(asking);
    written.push(asking);
    for (const c of calls) {
      const result = message(entry, "Tool", c.content, at, {
        toolCallId: c.id,
        toolName: c.name,
        effect: c.effect,
        summary: c.summary,
      });
      entry.messages.push(result);
      written.push(result);
    }
  }
  const reply = message(entry, "Assistant", turn.plan.reply, at + 1);
  entry.messages.push(reply);
  written.push(reply);
  let artifact = null;
  if (proposeCall) {
    const id = `aart_formula_${turn.id.split("_").at(-1)}`;
    artifact = {
      id,
      threadId: entry.thread.id,
      messageId: reply.id,
      kind: "draft_edit",
      status: "Ready",
      title: "Proposed formula",
      payload: proposeCall.edit,
      sourceToolCallId: proposeCall.id,
      pinned: false,
      lineageSeq: 1,
      slug: `proposed-formula-${entry.artifacts.length + 1}`,
      turn: turn.user.content,
      createdAt: at + 1,
      updatedAt: at + 1,
    };
    entry.artifacts.push(artifact);
  }
  entry.thread.lastMessageAt = reply.createdAt;
  entry.thread.updatedAt = reply.createdAt;
  entry.thread.version += 1;
  turn.done = true;
  turn.settled = { calls, written, reply, artifact };
  return turn.settled;
}

/** The turn's stream, framed as the server frames it (turn-stream.ts reads these). */
function followTurn(state, path) {
  const turn = nova(state).turns.get(/\/turns\/([^/]+)\/stream/.exec(path)[1]);
  if (!turn) return notFound("Turn");
  const entry = nova(state).threads.get(turn.threadId);
  const { calls, written, reply, artifact } = settle(state, turn);
  const frames = [{ event: "accepted", data: { content: turn.user.content }, delay: 300 }];
  if (turn.plan.reasoning)
    frames.push({ event: "reasoning", data: { text: turn.plan.reasoning }, delay: 350 });
  for (const c of calls) {
    frames.push({
      event: "tool_started",
      data: { callId: c.id, name: c.name, arguments: c.args, effect: c.effect },
      delay: 450,
    });
    frames.push({
      event: "tool_finished",
      data: {
        callId: c.id,
        name: c.name,
        content: c.content,
        effect: c.effect,
        summary: c.summary,
      },
      delay: 650,
    });
  }
  if (artifact) {
    frames.push({
      event: "artifact",
      data: {
        id: artifact.id,
        kind: artifact.kind,
        status: artifact.status,
        title: artifact.title,
        sourceToolCallId: artifact.sourceToolCallId,
        draft: artifact.payload,
      },
      delay: 120,
    });
  }
  const words = reply.content.split(/(?<= )/);
  for (let i = 0; i < words.length; i += 3) {
    frames.push({
      event: "delta",
      data: { text: words.slice(i, i + 3).join("") },
      delay: i === 0 ? 400 : 50,
    });
  }
  frames.push({ event: "message", data: { content: reply.content, model: "mock" }, delay: 40 });
  frames.push({
    event: "done",
    data: {
      thread: entry.thread,
      messages: [turn.user, ...written],
      reply: reply.content,
      refused: false,
      artifacts: artifact ? [artifact] : [],
    },
    delay: 40,
  });
  return { __sse: frames };
}

function listMessages(state, path, _body, query) {
  const entry = entryOf(state, path);
  if (!entry) return notFound("Conversation");
  const limit = Number(query?.get("limit")) || 50;
  const before = Number(query?.get("before")) || 0;
  const rows = entry.messages.filter((m) => !before || m.sequence < before);
  const results = rows.slice(-limit);
  return {
    results,
    hasMore: rows.length > results.length,
    total: entry.messages.length,
    limit: 200,
  };
}

/** The conversation's artifacts by lineage; `summary` cuts each payload to what names it. */
function listArtifacts(state, path, _body, query) {
  const entry = entryOf(state, path);
  if (!entry) return notFound("Conversation");
  const kind = query?.get("kind");
  const pinned = query?.get("pinned") === "true";
  const q = (query?.get("q") ?? "").toLowerCase();
  const summary = query?.get("summary") === "true";
  const rows = [...entry.artifacts]
    .reverse()
    .filter(
      (a) =>
        (!kind || a.kind === kind) &&
        (!pinned || a.pinned) &&
        (!q || `${a.title} ${a.turn}`.toLowerCase().includes(q)),
    );
  return {
    results: rows.map((a) =>
      summary ? { ...a, payload: { surface: a.payload.surface, action: a.payload.action } } : a,
    ),
    total: rows.length,
    nextCursor: "",
    counts: {
      all: entry.artifacts.length,
      pinned: entry.artifacts.filter((a) => a.pinned).length,
      families: entry.artifacts.length ? { doc: entry.artifacts.length } : {},
    },
  };
}

function artifactLineage(state, path) {
  const entry = entryOf(state, path);
  if (!entry) return notFound("Conversation");
  const id = /\/artifacts\/(?:by-slug\/)?([^/]+)/.exec(path)[1];
  const artifact = entry.artifacts.find((a) => a.id === id || a.slug === id);
  return artifact ? { results: [artifact] } : notFound("Artifact");
}

function pinArtifact(state, path, body) {
  const entry = entryOf(state, path);
  if (!entry) return notFound("Conversation");
  const artifact = entry.artifacts.find((a) => a.id === /\/artifacts\/([^/]+)\/pin/.exec(path)[1]);
  if (!artifact) return notFound("Artifact");
  artifact.pinned = Boolean(body?.pinned);
  return artifact;
}

function budget(state) {
  const tomorrow = Math.floor(state.anchor / 86400) * 86400 + 86400;
  const monthStart = Math.floor(new Date(state.anchor * 1000).setUTCDate(1) / 1000 / 86400) * 86400;
  const nextMonth = new Date(monthStart * 1000);
  nextMonth.setUTCMonth(nextMonth.getUTCMonth() + 1);
  return {
    agentName: AGENT.name,
    spentUsd: "3.12",
    limitUsd: "100.00",
    share: 0.0312,
    near: false,
    monthStart,
    resetsAt: Math.floor(nextMonth.getTime() / 1000),
    runsToday: 6,
    dailyRunLimit: 200,
    budgetUsed: false,
    dailyUsed: false,
    dayResetsAt: tomorrow,
    disabledBy: "",
    disabledAt: 0,
    person: { used: 23, limit: 500, resetsAt: tomorrow },
  };
}

function activeTurn(state, path) {
  const threadId = threadIdOf(path);
  const turn = [...nova(state).turns.values()].find((t) => t.threadId === threadId && !t.done);
  return {
    turn: turn
      ? { id: turn.id, threadId, status: "Running", origin: "Person", input: turn.user.content }
      : null,
  };
}

function patchThread(state, path, body) {
  const entry = entryOf(state, path);
  if (!entry) return notFound("Conversation");
  if (typeof body?.title === "string") entry.thread.title = body.title.trim().slice(0, 200);
  if (typeof body?.pinned === "boolean") entry.thread.pinned = body.pinned;
  entry.thread.version += 1;
  entry.thread.updatedAt = state.now();
  return entry.thread;
}

const FT = "/api/v1/formula-templates";
const THREAD = `athr_formula_\\d+`;
const thread = (rest = "") => new RegExp(`^/api/v1/assistant/threads/${THREAD}${rest}/?$`);

export const FORMULA_NOVA_ROUTES = [
  ["POST", new RegExp(`^${FT}/ai/thread/?$`), openThread],
  ["POST", new RegExp(`^${FT}/ai/explain/?$`), explainFormula],
  ["POST", new RegExp(`^${FT}/ai/describe-inputs/?$`), describeInputs],
  ["POST", new RegExp(`^${FT}/ai/suggest-scenarios/?$`), suggestScenarios],
  [
    "POST",
    new RegExp(`^${FT}/fmt_[A-Za-z0-9_]+/test-cases/ftc_[A-Za-z0-9_]+/explain/?$`),
    explainScenario,
  ],
  ["POST", new RegExp(`^${FT}/fmt_[A-Za-z0-9_]+/review-note/?$`), reviewNote],
  ["GET", thread(), (s, p) => entryOf(s, p)?.thread ?? notFound("Conversation")],
  ["PATCH", thread(), patchThread],
  [
    "POST",
    thread("/read"),
    (s, p) => (entryOf(s, p) ? { __status: 204, body: undefined } : notFound("Conversation")),
  ],
  ["GET", thread("/messages"), listMessages],
  [
    "POST",
    thread("/turns"),
    (s, p, b) => {
      const started = startTurn(s, p, b);
      return started.__status ? started : { __status: 202, body: started };
    },
  ],
  ["GET", thread("/queue"), () => ({ items: [] })],
  [
    "POST",
    thread("/queue"),
    (s, p, b) => {
      const started = startTurn(s, p, b);
      return started.__status
        ? started
        : { __status: 202, body: { steering: false, turn: started } };
    },
  ],
  ["GET", thread("/turns/active"), activeTurn],
  ["GET", thread("/artifacts"), listArtifacts],
  ["POST", thread("/artifacts/[^/]+/pin"), pinArtifact],
  ["GET", thread("/artifacts/by-slug/[^/]+"), artifactLineage],
  ["GET", thread("/artifacts/[^/]+"), artifactLineage],
  ["GET", thread("/proposals"), () => ({ results: [] })],
  ["GET", thread("/plans"), () => ({ results: [] })],
  ["GET", thread("/schedules"), () => ({ items: [], total: 0 })],
  ["GET", thread("/waits"), () => ({ items: [] })],
  ["GET", thread("/budget"), budget],
  ["GET", thread("/case"), () => notFound("Case")],
  ["GET", /^\/api\/v1\/assistant\/turns\/aturn_formula_\d+\/stream\/?$/, followTurn],
  [
    "POST",
    /^\/api\/v1\/assistant\/turns\/aturn_formula_\d+\/stop\/?$/,
    () => ({ __status: 204, body: undefined }),
  ],
];
