// Nova, the formula assistant, on the Formula Studio and the library studio: the one-shot calls
// (explain the tariff statement by statement on a load, suggest tests, draft the reviewer's
// note) and each page's conversation, whose turns stream the way a real turn does and answer
// with a `formula_propose` (or `formula_library_propose`) draft edit carrying the proposed
// source, the module's check of it and its tests. Every amount in a reply is the rules module's
// (../rules.mjs). Each one-shot answers after about 700ms so its loading state can be shot. With
// the `aiProviders` scenario at `none`, the one-shots refuse as the server does with no provider.
import { ORG, SHIPMENTS, USER, shipmentOverlay } from "../fixtures/formula.mjs";
import { recordFrom, rules, stop } from "../rules.mjs";
import {
  approvedVersion,
  business,
  checkOf,
  impact,
  invalid,
  messageText,
  notFound,
  publicBacktest,
  reviewDiff,
  studio,
  tariffContext,
  templateOf,
  testsOf,
} from "./formula.mjs";

const NOVA_DELAY_MS = Number(process.env.MOCK_FORMULA_NOVA_MS ?? 700);
const THREAD_PREFIX = "athr_formula_";

const AGENT = {
  id: "agdef_formula_assistant",
  name: "Formula assistant",
  description:
    "Helps write, explain and test the tariffs and libraries that price freight, in Tariff, and proposes changes for a person to apply and save.",
  template: "FormulaAssistant",
  icon: "sparkle",
  accent: "violet",
  systemKey: "formula_assistant",
  toolNames: [
    "formula_check",
    "formula_run_tests",
    "formula_propose",
    "formula_list_templates",
    "formula_get_template",
    "formula_list_libraries",
    "formula_get_library",
    "formula_library_propose",
  ],
  starters: [
    { label: "Explain this tariff", prompt: "Explain this tariff, statement by statement" },
    { label: "Add a weekend fee", prompt: "Add a weekend pickup fee" },
    { label: "Suggest tests", prompt: "Suggest tests for the cases this tariff could get wrong" },
  ],
};

const noProvider = (state) => state.scenario.aiProviders === "none";
const refuseNoProvider = () => business("No AI provider is configured for the formula assistant");
const later = (payload, ms = NOVA_DELAY_MS) =>
  new Promise((resolve) => setTimeout(() => resolve(payload), ms));
const code = (name) => `\`${name}\``;

const joinPhrases = (list) =>
  list.length <= 1 ? (list[0] ?? "") : `${list.slice(0, -1).join(", ")} and ${list.at(-1)}`;

/** Dollars the way Nova writes them: $1,234.56. */
function money(x) {
  return `${x < 0 ? "−" : ""}$${Math.abs(x).toLocaleString("en-US", { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`;
}

/** A wire value in prose: money as dollars, quantities with their unit. */
function wireText(value) {
  if (value === null || value === undefined) return "none";
  if (typeof value === "object" && "$q" in value) {
    if (value.u === "$") return money(Number(value.$q));
    if (value.u === "%") return `${value.$q}%`;
    if (value.u.startsWith("$/")) return `${money(Number(value.$q))}${value.u.slice(1)}`;
    return `${Number(value.$q).toLocaleString("en-US")} ${value.u}`;
  }
  if (typeof value === "object" && "$n" in value) return value.$n;
  if (typeof value === "object" && "$date" in value) return value.$date;
  return String(value);
}

const PRICED_LOADS = ["shp_fs_240758", "shp_fs_240741", "shp_fs_240772"];

function recordOf(shipmentId) {
  const shipment = SHIPMENTS.find((s) => s.id === shipmentId);
  return shipment ? recordFrom(shipmentOverlay(shipment, stop)) : null;
}

function rate(state, source, record, inputs = {}) {
  const { results } = rules("tariff.evaluate", {
    context: tariffContext(state),
    record,
    programs: [{ id: "nova", source, inputs }],
    trace: false,
  });
  return results[0];
}

// --- One-shots ------------------------------------------------------------------------------

/** What each statement does on the load, one sentence each, anchored to its range. */
function explainFormula(state, _path, body) {
  if (noProvider(state)) return refuseNoProvider();
  const source = String(body?.source ?? "");
  if (!source.trim()) return invalid("source", "There is nothing to explain yet", "REQUIRED");
  const context = tariffContext(state);
  const check = checkOf(source, "tariff", context);
  if (!check.ok) return business("Fix the errors first; Nova explains a tariff that checks");
  const record = body?.shipmentId ? recordOf(body.shipmentId) : null;
  if (body?.shipmentId && !record) return notFound("Shipment");
  const result = rate(state, source, record, body?.inputs ?? {});
  const bill = result.status === "rated" ? result.bill : null;
  const load = record ? "this shipment" : "the sample load";
  const lineFor = (label) => bill?.lines.find((line) => line.label === label);
  const items = (check.outline?.items ?? []).flatMap((item) => {
    if (item.kind === "input") {
      const input = check.inputs.find((declared) => declared.name === item.name);
      const fallback = input?.default
        ? `, ${wireText(input.default)} unless the agreement sets its own`
        : ", with no default: the agreement must set it";
      return [{ range: item.range, says: `${code(item.name)} is an input, ${input?.type ?? "a value"}${fallback}.` }];
    }
    if (item.kind === "charge") {
      const line = lineFor(item.name);
      return [
        {
          range: item.range,
          says: line
            ? `Bills ${item.name}${item.code ? ` (${item.code})` : ""}: ${wireText(line.amount)} on ${load}.`
            : `Bills ${item.name} only when its condition holds; it did not apply to ${load}.`,
        },
      ];
    }
    if (item.kind === "total") {
      const adjustment = bill?.lines.find((line) => line.kind === item.name);
      return [
        {
          range: item.range,
          says: adjustment
            ? `The ${item.name} lifted the bill by ${wireText(adjustment.amount)} on ${load}.`
            : `The ${item.name} did not change the bill on ${load}.`,
        },
      ];
    }
    if (item.kind === "test") return [{ range: item.range, says: `A test pins what the tariff bills: "${item.name}".` }];
    return [];
  });
  return later({ items, bill, generated: true });
}

/** Tests from real loads on the tariff's edges, each a `test` block the module wrote. */
function suggestedTests(state, source) {
  const context = tariffContext(state);
  return PRICED_LOADS.flatMap((id) => {
    const shipment = SHIPMENTS.find((s) => s.id === id);
    const name = `${shipment.lane} (${shipment.pro})`;
    const out = rules("tariff.scaffoldTest", { source, context, record: recordOf(id), inputs: {}, name });
    if (!out.test) return [];
    const why =
      id === "shp_fs_240758"
        ? "A short haul, where a minimum usually decides the bill."
        : id === "shp_fs_240741"
          ? "Five stops, which exercises every per-stop charge."
          : "Hazmat on a short lane, where handling fees stack up.";
    return [{ name, source: out.test, why }];
  });
}

function suggestTests(state, _path, body) {
  if (noProvider(state)) return refuseNoProvider();
  const source = String(body?.source ?? "");
  const check = checkOf(source, "tariff", tariffContext(state));
  if (!check.ok) return business("Fix the errors first; Nova suggests tests for a tariff that checks");
  return later({ tests: suggestedTests(state, source), generated: true });
}

/** The reviewer's note: what changed since the approved version, its effect and its tests. */
function reviewNote(state, path) {
  const t = templateOf(state, path);
  if (!t) return notFound("Formula template");
  if (noProvider(state)) return refuseNoProvider();
  const context = tariffContext(state);
  const run = checkOf(t.source, "tariff", context).ok ? testsOf(t.source, context) : { tests: [] };
  const failing = run.tests.filter((test) => test.status !== "passed").length;
  const testLine =
    run.tests.length === 0
      ? "It has no tests yet."
      : failing === 0
        ? `All ${run.tests.length} tests pass on this version.`
        : `${failing} of ${run.tests.length} tests fail on this version.`;
  const diff = reviewDiff(state, t);
  if (!diff.hasApprovedBase) {
    return later({ note: `This is the first version of ${t.name}, so nothing in production rates with it yet. ${testLine}`, generated: true, backtest: null });
  }
  const before = (approvedVersion(t)?.content.source ?? "").split("\n");
  const after = t.source.split("\n");
  const changed = after.filter((line) => !before.includes(line)).length + before.filter((line) => !after.includes(line)).length;
  const what = changed === 0 ? `The source matches approved v${diff.baseVersion}.` : `Changes ${changed} lines of the source since approved v${diff.baseVersion}.`;
  const summary = publicBacktest(impact(state, t)).summary;
  const delta = Number(summary.totalDelta);
  const pct = Number(summary.totalDeltaPct);
  const effect = `Against ${summary.shipmentCount} recent shipments, ${summary.changedCount} would rate differently (${delta >= 0 ? "+" : "−"}${money(Math.abs(delta))} in total, ${pct >= 0 ? "+" : ""}${pct.toFixed(1)}%).`;
  return later({ note: `${what} ${effect} ${testLine}`, generated: true, backtest: summary });
}

// --- The page conversation ------------------------------------------------------------------

function nova(state) {
  state.formulaNova ??= { threads: new Map(), bySubject: new Map(), turns: new Map(), nextId: 1 };
  return state.formulaNova;
}

function nextId(n, prefix) {
  const id = `${prefix}${n.nextId}`;
  n.nextId += 1;
  return id;
}

/** A conversation about a template or a library (`kind`), saved (`recordId`) or not yet. */
function newThread(state, kind, recordId) {
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
      subjectType: recordId ? (kind === "library" ? "FormulaLibrary" : "FormulaTemplate") : "",
      subjectId: recordId ?? "",
      canContinue: true,
      version: 1,
      createdAt: now,
      updatedAt: now,
    },
    kind,
    recordId,
    messages: [],
    artifacts: [],
  };
  n.threads.set(id, entry);
  if (recordId) n.bySubject.set(`${kind}:${recordId}`, id);
  return entry;
}

/** The formula page's conversations, for the rail's list (which leaves them out, as it does for real). */
export function formulaThreads(state) {
  return [...(state.formulaNova?.threads.values() ?? [])].map((entry) => entry.thread);
}

/** Opens the conversation about a saved template or library, or a fresh one for one not yet saved. */
const openThread = (kind) => (state, _path, body) => {
  const raw = String((kind === "library" ? body?.libraryId : body?.templateId) ?? "").trim();
  if (raw) {
    const records = kind === "library" ? studio(state).libraries : studio(state).templates;
    if (!records.has(raw)) return notFound(kind === "library" ? "Formula library" : "Formula template");
    const known = nova(state).bySubject.get(`${kind}:${raw}`);
    const entry = known ? nova(state).threads.get(known) : newThread(state, kind, raw);
    return { thread: entry.thread, agent: AGENT };
  }
  return { thread: newThread(state, kind, null).thread, agent: AGENT };
};

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

/** The editor as the turn carries it (the page draft), or the saved record when it carries none. */
function editorOf(state, entry, body) {
  const draft = body?.context?.draft?.formula;
  const records = entry.kind === "library" ? studio(state).libraries : studio(state).templates;
  const saved = entry.recordId ? records.get(entry.recordId) : null;
  if (draft && typeof draft.source === "string") {
    return {
      name: draft.name ?? saved?.name ?? "",
      description: draft.description ?? saved?.description ?? "",
      source: draft.source,
    };
  }
  return { name: saved?.name ?? "", description: saved?.description ?? "", source: saved?.source ?? "" };
}

function contextOf(state, entry) {
  return tariffContext(state, entry.kind === "library" ? { except: entry.recordId } : {});
}

/** pagedraft.FormulaProposal: the proposed file, the module's check of it and its tests. */
function proposal(state, entry, editor, next, explanation) {
  const context = contextOf(state, entry);
  const check = checkOf(next.source, entry.kind, context);
  return {
    name: next.name ?? "",
    description: next.description ?? "",
    source: next.source,
    explanation,
    check: { ok: check.ok, diagnostics: check.diagnostics },
    tests: check.ok ? testsOf(next.source, context).tests : [],
  };
}

function tool(name, effect, args, summary, result) {
  return { name, effect, args, summary, content: JSON.stringify(result ?? { ok: true }) };
}

/** The first `$` amount or bare number in the question. */
function amountIn(q) {
  const match = /\$\s?(\d[\d,]*(?:\.\d+)?)/.exec(q) ?? /\b(\d[\d,]*(?:\.\d+)?)\b/.exec(q);
  return match ? Number(match[1].replaceAll(",", "")) : null;
}

function percentIn(q) {
  const match = /(\d+(?:\.\d+)?)\s?%/.exec(q);
  return match ? Number(match[1]) : null;
}

const moneyLiteral = (n) => `$${Number.isInteger(n) ? n : n.toFixed(2)}`;

/** Inserts a statement before the total clauses and tests, or at the end, after a blank line. */
function insertBeforeTests(source, block) {
  const at = source.search(/^(?:minimum|maximum|round)\b|^test "/m);
  if (at === -1) return `${source.replace(/\s+$/, "")}\n\n${block.trim()}\n`;
  return `${source.slice(0, at).replace(/\s+$/, "")}\n\n${block.trim()}\n\n${source.slice(at)}`;
}

/** Adds an import after the header (or after the other `use` lines). */
function withImport(source, line) {
  if (source.includes(line)) return source;
  const uses = [...source.matchAll(/^use .*$/gm)];
  if (uses.length > 0) {
    const last = uses.at(-1);
    const end = last.index + last[0].length;
    return `${source.slice(0, end)}\n${line}${source.slice(end)}`;
  }
  const header = /^(tariff|library) "[^"]*"\n/m.exec(source);
  if (!header) return `${line}\n${source}`;
  const end = header.index + header[0].length;
  return `${source.slice(0, end)}\n${line}\n${source.slice(end)}`;
}

/** A tariff drafted from a description: per mile, a fuel percent and a minimum. */
function draftTariff(q) {
  const fuel = percentIn(q) ?? 18;
  const minimum = amountIn(q.replace(/\d+(?:\.\d+)?\s?%/, "")) ?? 250;
  const rateMatch = /\$\s?(\d+(?:\.\d+)?)\s?(?:\/|per )\s?mi/.exec(q);
  const ratePerMile = rateMatch ? Number(rateMatch[1]) : 2.85;
  return `tariff "Per mile + fuel"

/// Linehaul by the mile with a fuel surcharge on it, and a minimum for short hauls.

/// The rate per loaded mile.
input rate: Money/mi = ${moneyLiteral(ratePerMile)}/mi
/// The fuel surcharge, as a share of the linehaul.
input fuelPercent: Percent = ${fuel}%
/// The least a shipment pays.
input minimumCharge: Money = ${moneyLiteral(minimum)}

let linehaul = rate * shipment.distance

charge "Linehaul" (quantity: shipment.distance, rate: rate)
charge "Fuel surcharge" as FSC = linehaul * fuelPercent

minimum minimumCharge

test "a short haul bills the minimum" {
  given shipment.distance = 10 mi
  expect total == ${moneyLiteral(minimum)}
}
`;
}

function draftLibrary(name) {
  const path = `org.${/^[a-z][A-Za-z0-9]*$/.test(name) ? name : "shared"}`;
  return `library "${path}"

/// Accessorial fees the organisation's tariffs share.

/// The hazmat handling fee, per shipment.
const HAZMAT_FEE: Money = $175

/// The residential delivery fee.
const RESIDENTIAL_FEE: Money = $95

/// The hazmat fee when the shipment carries hazardous goods.
fn hazmat(carries: Bool) -> Money {
  if carries {
    return HAZMAT_FEE
  }
  return $0
}

test "hazmat bills only hazardous loads" {
  expect hazmat(true) == $175
  expect hazmat(false) == $0
}
`;
}

/** Inserts a total clause before the first test, or at the end. */
function insertBeforeFirstTest(source, block) {
  const at = source.search(/^test "/m);
  if (at === -1) return `${source.replace(/\s+$/, "")}\n\n${block.trim()}\n`;
  return `${source.slice(0, at).replace(/\s+$/, "")}\n\n${block.trim()}\n\n${source.slice(at)}`;
}

const WEEKEND_BLOCK = (fee) => `if let start = shipment.pickup?.window?.start {
  if weekend(start) {
    charge "Weekend pickup" as WKND = ${moneyLiteral(fee)}
  }
}`;

/** What the turn will say and do, decided from the question and the editor. */
function planTurn(state, entry, body) {
  const question = String(body.content ?? "");
  const q = question.toLowerCase();
  const editor = editorOf(state, entry, body);
  const context = contextOf(state, entry);
  const check = checkOf(editor.source, entry.kind, context);
  const errors = check.diagnostics.filter((finding) => finding.severity === "error");
  const lines = editor.source.split("\n").filter((line) => line.trim()).length;
  const reading = lines
    ? `Reading the editor: ${lines} lines, ${errors.length === 0 ? "it checks" : `${errors.length} errors`}.`
    : "Reading the editor: it is empty.";
  const checkStep = tool(
    "formula_check",
    "lookup",
    { kind: entry.kind },
    errors.length === 0 ? "Checks clean" : `${errors.length} errors`,
  );
  const propose = (next, explanation, reply) => {
    const formula = proposal(state, entry, editor, next, explanation);
    const passed = formula.tests.filter((test) => test.status === "passed").length;
    const testStep = tool(
      "formula_run_tests",
      "lookup",
      {},
      formula.tests.length ? `${passed} of ${formula.tests.length} tests pass` : "No tests",
    );
    const verdict = !formula.check.ok
      ? ` The checker refuses it: ${messageText(formula.check.diagnostics.find((d) => d.severity === "error")?.message ?? "")}.`
      : formula.tests.length === 0
        ? " It checks clean; it has no tests yet."
        : passed === formula.tests.length
          ? formula.tests.length === 1
            ? " It checks clean and its test passes."
            : ` It checks clean and all ${formula.tests.length} tests pass.`
          : ` It checks clean, but ${formula.tests.length - passed} of ${formula.tests.length} tests fail.`;
    return { reasoning: reading, tools: [checkStep, testStep], proposal: formula, reply: reply + verdict };
  };
  const has = (re) => re.test(q);
  const library = entry.kind === "library";

  if (!editor.source.trim()) {
    if (library) {
      return propose(
        { name: editor.name || "shared", description: "Accessorial fees the organisation's tariffs share.", source: draftLibrary(editor.name) },
        "A library with the hazmat and residential fees as constants, a function that bills hazmat only on hazardous loads, and a test for it.",
        "I drafted a library of shared accessorial fees, with a test.",
      );
    }
    return propose(
      { name: "Per mile + fuel", description: "Linehaul by the mile, a fuel surcharge on it and a minimum.", source: draftTariff(q) },
      "A per-mile linehaul, a fuel surcharge on the linehaul and a minimum, with a test that a short haul bills the minimum.",
      "I drafted the tariff from your description.",
    );
  }

  if (has(/\bfix\b|error|broken|fails|doesn't check|does not check/)) {
    const fixable = check.diagnostics.find((finding) => finding.fixes.length > 0);
    if (!fixable) {
      return {
        reasoning: reading,
        tools: [checkStep],
        reply: errors.length === 0 ? "There is nothing to fix: the file checks clean." : `I can't fix this one by rule: ${messageText(errors[0].message)}. Tell me what it should do and I'll rewrite it.`,
      };
    }
    let next = editor.source;
    for (const change of [...fixable.fixes[0].changes].sort((a, b) => b.range.from - a.range.from)) {
      next = next.slice(0, change.range.from) + change.insert + next.slice(change.range.to);
    }
    return propose(
      { source: next },
      `${messageText(fixable.fixes[0].label)}: ${messageText(fixable.message)}`,
      `I applied the checker's fix: ${messageText(fixable.fixes[0].label).toLowerCase()}.`,
    );
  }

  if (!library && has(/weekend/)) {
    const fee = amountIn(q) ?? 150;
    const next = withImport(insertBeforeTests(editor.source, WEEKEND_BLOCK(fee)), "use std.time.{ weekend }");
    return propose(
      { source: next },
      `Bills ${moneyLiteral(fee)} as WKND when the first pickup's window starts on a Saturday or Sunday, using std.time's weekend.`,
      `I added a ${moneyLiteral(fee)} weekend pickup charge.`,
    );
  }

  if (!library && has(/minimum/) && amountIn(q) !== null) {
    const amount = amountIn(q);
    const next = /^minimum .*$/m.test(editor.source)
      ? editor.source.replace(/^minimum .*$/m, `minimum ${moneyLiteral(amount)}`)
      : insertBeforeFirstTest(editor.source, `minimum ${moneyLiteral(amount)}`);
    return propose(
      { source: next },
      `Lifts any bill under ${moneyLiteral(amount)} to ${moneyLiteral(amount)}.`,
      `I set the minimum to ${moneyLiteral(amount)}.`,
    );
  }

  if (has(/test|edge/)) {
    if (errors.length > 0) {
      return { reasoning: reading, tools: [checkStep], reply: `Fix the errors first; tests run on a file that checks. The first: ${messageText(errors[0].message)}.` };
    }
    if (library) {
      const fn = check.outline?.items.find((item) => item.kind === "function");
      const block = `test "${fn ? fn.name : "the library"} on its edges" {\n  expect 1 == 1\n}`;
      return propose(
        { source: insertBeforeFirstTest(editor.source, block) },
        "A test to fill in with the edge cases of each function.",
        "I added a test block to fill in with each function's edge cases.",
      );
    }
    const suggested = suggestedTests(state, editor.source);
    const next = suggested.reduce((source, test) => `${source.replace(/\s+$/, "")}\n\n${test.source.trim()}\n`, editor.source);
    return propose(
      { source: next },
      suggested.map((test) => `${test.name}: ${test.why}`).join(" "),
      `I added ${suggested.length} tests from real loads on this tariff's edges.`,
    );
  }

  if (has(/explain|what does|how does/)) {
    const items = check.outline?.items ?? [];
    const charges = items.filter((item) => item.kind === "charge").map((item) => item.name);
    const inputs = check.inputs.map((input) => code(input.name));
    const functions = items.filter((item) => item.kind === "function").map((item) => code(item.name));
    return {
      reasoning: reading,
      tools: [checkStep],
      reply: library
        ? `This library declares ${functions.length ? joinPhrases(functions) : "no functions"}, and ${items.filter((item) => item.kind === "test").length} tests pin what they return.`
        : `It takes ${inputs.length ? joinPhrases(inputs) : "no inputs"}, bills ${charges.length ? joinPhrases(charges) : "no charges"}${items.some((item) => item.kind === "total") ? ", then applies its total rules" : ""}. Explain under the source walks it statement by statement on a load.`,
    };
  }

  return {
    reasoning: reading,
    tools: [checkStep],
    reply: library
      ? "I can add functions and constants, write tests for them, fix what the checker refuses, or explain what is there. Every change I propose is checked and tested before you see it."
      : "I can change the tariff as you describe it (add a fee, set a minimum, change a rate), write tests from real loads, fix what the checker refuses, or explain what is there. Every amount I quote comes from the rules module.",
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
      name: entry.kind === "library" ? "formula_library_propose" : "formula_propose",
      effect: "present",
      args: { source: turn.plan.proposal.source, explanation: turn.plan.proposal.explanation },
      summary: entry.kind === "library" ? "Proposed library" : "Proposed tariff",
    };
    const edit =
      entry.kind === "library"
        ? { surface: "formula_library", action: "formula_library_propose", formula: turn.plan.proposal }
        : { surface: "formula", action: "formula_propose", formula: turn.plan.proposal };
    proposeCall.content = JSON.stringify({
      draft: edit,
      note: "The change is in front of the person with its check and its tests.",
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
      title: entry.kind === "library" ? "Proposed library" : "Proposed tariff",
      payload: proposeCall.edit,
      sourceToolCallId: proposeCall.id,
      pinned: false,
      lineageSeq: 1,
      slug: `proposed-${entry.kind}-${entry.artifacts.length + 1}`,
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
const FL = "/api/v1/formula-libraries";
const THREAD = `athr_formula_\\d+`;
const thread = (rest = "") => new RegExp(`^/api/v1/assistant/threads/${THREAD}${rest}/?$`);

export const FORMULA_NOVA_ROUTES = [
  ["POST", new RegExp(`^${FT}/ai/thread/?$`), openThread("tariff")],
  ["POST", new RegExp(`^${FL}/ai/thread/?$`), openThread("library")],
  ["POST", new RegExp(`^${FT}/ai/explain/?$`), explainFormula],
  ["POST", new RegExp(`^${FT}/ai/suggest-tests/?$`), suggestTests],
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
