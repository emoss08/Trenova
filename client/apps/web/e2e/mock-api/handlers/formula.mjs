// The Formula Studio's REST routes and GraphQL operations: templates, versions, scenarios,
// readiness, review, backtest and impact, the preview with its statement trace, the schema and
// the standard library. Every amount comes from fixtures/formula-engine.mjs, so the preview, the
// scenarios, the backtest and Nova agree. Writes keep formulatemplateservice's rules: a template
// is born a draft, a material edit takes a new version and returns an approved or in-review
// template to draft, submit is gated on readiness, and an approval takes a version snapshot.
import {
  analyze,
  applyPolicy,
  bill,
  context,
  decimal,
  draftOf,
  run,
  SCHEMA_NAMES,
  statementsOf,
  evaluate,
  valuesFor,
} from "../fixtures/formula-engine.mjs";
import {
  backtestShipments,
  createFormulaTemplates,
  FIELD_SOURCE,
  FUNCTIONS,
  SHIPMENTS,
  STANDARDS,
  TEAM,
  USER,
  ORG,
  userRecord,
  VARIABLES,
} from "../fixtures/formula.mjs";

const BASE = "/api/v1/formula-templates";
const ID = "fmt_[A-Za-z0-9_]+";
const TC = "ftc_[A-Za-z0-9_]+";
const DAY = 86400;
const STALE_AFTER = 14 * DAY;
const AGING_AFTER = 7 * DAY;
const MAX_REVIEWER_NOTE = 4000;
const MAX_HELD_SCENARIOS = 50;
const ROUNDING_MODES = new Set(["HalfUp", "HalfEven", "Up", "Down", "None"]);
const TEMPLATE_TYPES = new Set(["FreightCharge", "AccessorialCharge"]);
const STATUSES = new Set(["Active", "Inactive", "Draft", "InReview"]);
const CONTENT_KEYS = [
  "name",
  "description",
  "type",
  "expression",
  "schemaId",
  "variableDefinitions",
  "breakdownDefinitions",
  "minCharge",
  "maxCharge",
  "roundingMode",
  "roundingPrecision",
];
const MATERIAL_KEYS = CONTENT_KEYS.filter((key) => key !== "name" && key !== "description");

// --- Problems, as the server's error presenter writes them --------------------------------

export function problem(status, type, title, detail, errors) {
  return { __status: status, body: { type, title, status, detail, ...(errors ? { errors } : {}) } };
}

export function invalid(field, message, code = "INVALID") {
  return problem(400, "validation-error", "Validation Failed", message, [{ field, code, message }]);
}

export function business(message, errors) {
  return problem(422, "business-rule-violation", "Business Rule Violation", message, errors);
}

export function notFound(what) {
  return problem(404, "resource-not-found", "Resource Not Found", `${what} not found within your organization`);
}

// --- State ----------------------------------------------------------------------------------

/** The scenario's templates, built on first use so every scenario starts from the fixture. */
export function studio(state) {
  if (state.formulaStudio) return state.formulaStudio;
  state.formulaStudio = {
    templates: new Map(createFormulaTemplates(state.anchor).map((t) => [t.id, t])),
    nextId: 1,
    nextTemplate: 1,
  };
  return state.formulaStudio;
}

function nextId(s, prefix) {
  const id = `${prefix}${s.nextId}`;
  s.nextId += 1;
  return id;
}

const templateIdOf = (path) => new RegExp(`/formula-templates/(${ID})`).exec(path)?.[1];
const testCaseIdOf = (path) => new RegExp(`/test-cases/(${TC})`).exec(path)?.[1];

export function templateOf(state, path) {
  return studio(state).templates.get(templateIdOf(path)) ?? null;
}

// --- Shapes ---------------------------------------------------------------------------------

function usageCount(t) {
  return t.usages.reduce((sum, usage) => sum + usage.count, 0);
}

export function publicTemplate(t) {
  const { versions: _v, usages: _u, testCases, reviews: _r, ...row } = t;
  return {
    organizationId: ORG.id,
    businessUnitId: USER.businessUnitId,
    ...row,
    usageCount: usageCount(t),
    scenarioCount: testCases.length,
  };
}

function personRef(id) {
  const user = userRecord(id);
  return user ? { id: user.id, name: user.name, username: user.username } : null;
}

export function publicTestCase(tc) {
  const { recordedContent: _c, ...row } = tc;
  return {
    organizationId: ORG.id,
    businessUnitId: USER.businessUnitId,
    ...row,
    recordedBy: tc.recordedById ? personRef(tc.recordedById) : null,
  };
}

function publicReview(review) {
  return {
    organizationId: ORG.id,
    businessUnitId: USER.businessUnitId,
    ...review,
    actor: review.actorId ? userRecord(review.actorId) : null,
    requestedReviewer: review.requestedReviewerId ? userRecord(review.requestedReviewerId) : null,
  };
}

/** The content a template or a version snapshot prices with. */
export function contentOf(source) {
  return Object.fromEntries(CONTENT_KEYS.map((key) => [key, structuredClone(source[key])]));
}

function publicVersion(t, version) {
  const previous = t.versions.find((v) => v.versionNumber === version.versionNumber - 1);
  return {
    id: `ftv_${t.id.slice(4)}_${version.versionNumber}`,
    templateId: t.id,
    organizationId: ORG.id,
    businessUnitId: USER.businessUnitId,
    versionNumber: version.versionNumber,
    ...contentOf(version.content),
    status: version.status,
    effectiveFrom: null,
    metadata: {},
    changeMessage: version.changeMessage ?? "",
    changeSummary: previous ? diffContent(previous.content, version.content) : null,
    tags: version.tags ?? [],
    createdById: version.createdById,
    createdAt: version.createdAt,
    createdBy: userRecord(version.createdById),
  };
}

/** The newest snapshot taken while the template was active: what production rates with. */
export function approvedVersion(t) {
  return [...t.versions].reverse().find((version) => version.status === "Active") ?? null;
}

// --- Diffs, as jsonutils.JSONDiff reports them: one entry per changed leaf ----------------

function fieldType(value) {
  if (value === null || value === undefined) return "null";
  if (Array.isArray(value)) return "array";
  return typeof value;
}

function diffInto(changes, from, to, path) {
  const container = (v) => v !== null && typeof v === "object";
  if (container(from) && container(to) && Array.isArray(from) === Array.isArray(to)) {
    const keys = new Set([...Object.keys(from), ...Object.keys(to)]);
    for (const key of keys) diffInto(changes, from[key], to[key], path ? `${path}.${key}` : key);
    return;
  }
  const same = JSON.stringify(from ?? null) === JSON.stringify(to ?? null);
  if (same) return;
  const type = from === undefined || from === null ? "created" : to === undefined || to === null ? "deleted" : "updated";
  changes[path] = { from: from ?? null, to: to ?? null, type, fieldType: fieldType(to ?? from), path };
}

export function diffContent(from, to) {
  const changes = {};
  for (const key of CONTENT_KEYS) diffInto(changes, from?.[key], to?.[key], key);
  return changes;
}

// --- Writes' input ---------------------------------------------------------------------------

function decimalOrNull(value) {
  if (value === null || value === undefined || value === "") return null;
  const n = Number(value);
  return Number.isFinite(n) ? decimal(n) : value;
}

/** A form's values (or a TestCaseCandidate) as the server stores them. */
export function normalizeContent(body, fallback = {}) {
  const pick = (key, value) => (body?.[key] !== undefined ? body[key] : (fallback[key] ?? value));
  return {
    name: String(pick("name", "")).trim(),
    description: String(pick("description", "") ?? ""),
    type: pick("type", "FreightCharge"),
    expression: String(pick("expression", "") ?? ""),
    schemaId: pick("schemaId", "shipment") || "shipment",
    variableDefinitions: (pick("variableDefinitions", []) ?? [])
      .filter((v) => v && String(v.name ?? "").trim())
      .map((v) => ({
        name: String(v.name).trim(),
        type: v.type ?? "Number",
        description: v.description ?? "",
        required: Boolean(v.required),
        ...(v.defaultValue !== undefined && v.defaultValue !== null && v.defaultValue !== "" ? { defaultValue: v.defaultValue } : {}),
        ...(v.source ? { source: v.source } : {}),
      })),
    breakdownDefinitions: (pick("breakdownDefinitions", []) ?? [])
      .filter((b) => b && (String(b.name ?? "").trim() || String(b.expression ?? "").trim()))
      .map((b) => ({ name: String(b.name ?? "").trim(), label: String(b.label ?? "").trim(), expression: String(b.expression ?? "") })),
    minCharge: decimalOrNull(pick("minCharge", null)),
    maxCharge: decimalOrNull(pick("maxCharge", null)),
    roundingMode: pick("roundingMode", "HalfUp") || "HalfUp",
    roundingPrecision: Number.isInteger(pick("roundingPrecision", 2)) ? pick("roundingPrecision", 2) : 2,
  };
}

/** validateTemplate: the field errors a save is refused with. */
function contentErrors(content) {
  const errors = [];
  const add = (field, message, code = "INVALID") => errors.push({ field, code, message });
  if (!content.name) add("name", "Name is required", "REQUIRED");
  else if ([...content.name].length > 100) add("name", "Name must be at most 100 characters");
  if (!TEMPLATE_TYPES.has(content.type)) add("type", "Type must be FreightCharge or AccessorialCharge");
  if (!content.expression.trim()) add("expression", "Expression is required", "REQUIRED");
  if (!ROUNDING_MODES.has(content.roundingMode)) add("roundingMode", "Must be one of HalfUp, HalfEven, Up, Down, or None");
  if (content.roundingPrecision < 0 || content.roundingPrecision > 4) add("roundingPrecision", "Must be between 0 and 4");
  const min = content.minCharge === null ? null : Number(content.minCharge);
  const max = content.maxCharge === null ? null : Number(content.maxCharge);
  if (min !== null && !(min >= 0)) add("minCharge", "Minimum charge cannot be negative");
  if (max !== null && !(max >= 0)) add("maxCharge", "Maximum charge cannot be negative");
  if (min !== null && max !== null && min > max) add("minCharge", "Minimum charge cannot exceed maximum charge");
  const seen = new Set();
  content.variableDefinitions.forEach((v, i) => {
    if (seen.has(v.name)) add(`variableDefinitions[${i}].name`, "Duplicate variable name");
    seen.add(v.name);
  });
  content.breakdownDefinitions.forEach((b, i) => {
    if (!/^[a-zA-Z][a-zA-Z0-9_]*$/.test(b.name)) add(`breakdownDefinitions[${i}].name`, "Must start with a letter and contain only letters, numbers, and underscores");
    if (!b.label) add(`breakdownDefinitions[${i}].label`, "Label is required", "REQUIRED");
    if (!b.expression.trim()) add(`breakdownDefinitions[${i}].expression`, "Expression is required", "REQUIRED");
  });
  if (content.expression.trim()) {
    const issue = analyze(content.expression, context(content.variableDefinitions)).issues.find((i) => i.sev === "error");
    if (issue) add("expression", issue.msg);
  }
  return errors;
}

function refusedContent(errors) {
  return problem(400, "validation-error", "Validation Failed", errors[0].message, errors);
}

/** Who recorded a scenario, on which version and from what content. */
function recording(t, candidate, now) {
  return {
    recordedOnVersion: approvedVersion(t) ? t.currentVersionNumber : 0,
    recordedById: USER.id,
    recordedAt: now,
    recordedContent: candidate ? normalizeContent(candidate, contentOf(t)) : contentOf(t),
  };
}

function testCaseErrors(body, index) {
  const at = (field) => (index === undefined ? field : `testCases[${index}].${field}`);
  const errors = [];
  const name = String(body?.name ?? "").trim();
  if (!name) errors.push({ field: at("name"), code: "REQUIRED", message: "Name is required" });
  else if ([...name].length > 100) errors.push({ field: at("name"), code: "INVALID", message: "Name must be at most 100 characters" });
  const expected = Number(body?.expectedAmount);
  if (body?.expectedAmount === undefined || body?.expectedAmount === null || !Number.isFinite(expected)) {
    errors.push({ field: at("expectedAmount"), code: "REQUIRED", message: "Expected amount is required" });
  } else if (expected < 0) {
    errors.push({ field: at("expectedAmount"), code: "INVALID", message: "Expected amount cannot be negative" });
  }
  if (body?.tolerance !== undefined && !(Number(body.tolerance) >= 0)) {
    errors.push({ field: at("tolerance"), code: "INVALID", message: "Tolerance cannot be negative" });
  }
  return errors;
}

function newTestCase(s, t, body, now) {
  return {
    id: nextId(s, "ftc_studio_"),
    templateId: t.id,
    name: String(body.name).trim(),
    description: body.description ?? "",
    variables: body.variables ?? {},
    expectedAmount: decimal(Number(body.expectedAmount)),
    tolerance: decimal(body.tolerance === undefined ? 0.01 : Number(body.tolerance)),
    version: 1,
    createdById: USER.id,
    ...recording(t, body.candidate, now),
    createdAt: now,
    updatedAt: now,
  };
}

// --- Pricing ---------------------------------------------------------------------------------

/** Every scenario, run against `content`: the shape of a test-case run. */
export function runScenarios(t, content) {
  const draft = draftOf(content);
  const results = t.testCases.map((tc) => {
    const billed = bill(draft, tc.variables);
    const expected = Number(tc.expectedAmount);
    const tolerance = Number(tc.tolerance);
    const actual = billed.amount ?? 0;
    const passed = billed.error === null && Math.abs(actual - expected) <= tolerance + 1e-9;
    return {
      testCaseId: tc.id,
      name: tc.name,
      passed,
      expectedAmount: decimal(expected),
      actualAmount: decimal(actual),
      difference: decimal(actual - expected),
      tolerance: decimal(tolerance),
      ...(billed.error ? { error: billed.error } : {}),
    };
  });
  const passed = results.filter((r) => r.passed).length;
  return { results, total: results.length, passed, failed: results.length - passed };
}

/** The unguarded optional fields in the expression and every breakdown line. */
function nullableFields(content) {
  const ctx = context(content.variableDefinitions);
  const found = [];
  const scan = (scope, expression) =>
    analyze(expression, ctx)
      .issues.filter((i) => i.kind === "nullable")
      .forEach((i) => found.push({ scope, ...i }));
  scan("expression", content.expression);
  content.breakdownDefinitions.forEach((b, index) => scan(`breakdownDefinitions[${index}].expression`, b.expression));
  return found;
}

function warningsFor(found) {
  return found.map((item) => ({
    scope: item.scope,
    field: item.name,
    type: (item.type ?? "Number").toLowerCase(),
    message: `${item.name} can be empty on a shipment, and this expression would then fail to rate instead of pricing it.`,
    suggestion: item.fix,
  }));
}

function shipmentById(id) {
  return SHIPMENTS.find((s) => s.id === id) ?? null;
}

/** The preview (`POST /test`): the engine's answer for a draft on sample values or a shipment. */
function testExpression(state, _path, body) {
  const req = body ?? {};
  for (const field of ["minCharge", "maxCharge"]) {
    if (req[field] === undefined || req[field] === null || req[field] === "") continue;
    const n = Number(req[field]);
    if (!Number.isFinite(n)) return invalid(field, "Must be a valid decimal number");
    if (n < 0) return invalid(field, "Cannot be negative");
  }
  const min = req.minCharge === undefined || req.minCharge === "" || req.minCharge === null ? null : Number(req.minCharge);
  const max = req.maxCharge === undefined || req.maxCharge === "" || req.maxCharge === null ? null : Number(req.maxCharge);
  if (min !== null && max !== null && min > max) return invalid("minCharge", "Minimum charge cannot exceed maximum charge");
  if (req.roundingMode && !ROUNDING_MODES.has(req.roundingMode)) {
    return invalid("roundingMode", "Must be one of HalfUp, HalfEven, Up, Down, or None");
  }
  if (req.roundingPrecision !== undefined && req.roundingPrecision !== null && !(req.roundingPrecision >= 0 && req.roundingPrecision <= 4)) {
    return invalid("roundingPrecision", "Must be between 0 and 4");
  }
  return previewResponse({
    expression: req.expression ?? "",
    variables: req.variables ?? {},
    variableDefinitions: req.variableDefinitions,
    shipmentId: req.shipmentId || null,
    breakdowns: (req.breakdowns ?? []).filter((b) => b && b.name && b.expression),
    policy: {
      min,
      max,
      mode: req.roundingMode || "HalfUp",
      precision: Number.isInteger(req.roundingPrecision) ? req.roundingPrecision : 2,
    },
    trace: req.traceStatements === true,
  });
}

/**
 * Prices an expression the way TestExpression does. The request's `variables` are the sample
 * load's values with the template's input defaults over them; a name the schema does not know is
 * one of the template's inputs. With a shipment, its values stand in for the sample's.
 */
export function previewResponse({ expression, variables, variableDefinitions, shipmentId, breakdowns, policy, trace }) {
  const shipment = shipmentId ? shipmentById(shipmentId) : null;
  if (shipmentId && !shipment) {
    return { valid: false, error: `Shipment ${shipmentId} not found within your organization`, message: "Failed to load shipment" };
  }
  const customs = variableDefinitions?.length
    ? variableDefinitions.map((v) => ({ ...v, defaultValue: v.name in variables ? variables[v.name] : v.defaultValue }))
    : Object.entries(variables)
        .filter(([name]) => !SCHEMA_NAMES.has(name))
        .map(([name, value]) => ({ name, type: typeof value === "boolean" ? "Boolean" : typeof value === "string" ? "String" : "Number", defaultValue: value }));
  const sampleFields = Object.fromEntries(Object.entries(variables).filter(([name]) => SCHEMA_NAMES.has(name)));
  const base = shipment ? { ...shipment.values, ...sampleFields } : sampleFields;
  const ctx = context(customs);
  const values = valuesFor(base, customs, null);
  const an = analyze(expression, ctx);
  const res = evaluate(expression, ctx, values, an);
  const statements = trace ? statementsOf(res) : null;
  const withTrace = (out) => (statements ? { ...out, statements } : out);
  if (!res.ok) {
    return withTrace({
      valid: false,
      error: res.error,
      message: res.parseError || res.empty ? "Expression validation failed" : "Expression evaluation failed",
    });
  }
  if (typeof res.value !== "number") {
    return withTrace({ valid: false, error: "The expression evaluated to something other than an amount", message: "Expression evaluation failed" });
  }
  const applied = applyPolicy(res.value, policy);
  const source = (name) =>
    customs.some((c) => c.name === name) ? "default" : !shipment ? "sample" : FIELD_SOURCE.has(name) ? "field" : "computed";
  const response = {
    valid: true,
    result: decimal(applied.amount),
    message: shipment ? "Expression evaluated against shipment" : "Expression is valid",
    rounding: {
      mode: policy.mode,
      precision: policy.precision,
      applied: policy.mode !== "None" && applied.roundedChange,
      unroundedAmount: decimal(applied.clamped),
    },
    receipt: {
      variables: [...res.used.entries()].map(([name, value]) => ({ name, value, source: source(name) })),
      rawAmount: decimal(res.value),
      durationMicros: Math.max(1, res.micros ?? 1),
    },
  };
  if (policy.min !== null || policy.max !== null) {
    response.guardrail = {
      applied: applied.clamp !== null,
      ...(applied.clamp ? { bound: applied.clamp.bound } : {}),
      rawAmount: decimal(res.value),
      ...(policy.min !== null ? { minCharge: decimal(policy.min) } : {}),
      ...(policy.max !== null ? { maxCharge: decimal(policy.max) } : {}),
    };
  }
  if (breakdowns.length > 0) {
    response.breakdown = breakdowns.map((line) => {
      const r = evaluate(line.expression, ctx, values);
      const ok = r.ok && typeof r.value === "number";
      return {
        name: line.name,
        ...(line.label ? { label: line.label } : {}),
        amount: ok ? decimal(r.value) : "0",
        ...(ok ? {} : { error: r.ok ? "The line evaluated to something other than an amount" : r.error }),
      };
    });
  }
  if (shipment) response.resolvedVariables = values;
  const found = nullableFields({
    expression,
    variableDefinitions: customs,
    breakdownDefinitions: breakdowns,
  });
  if (found.length > 0) response.warnings = warningsFor(found);
  return withTrace(response);
}

// --- Backtest and impact ---------------------------------------------------------------------

function percent(delta, before) {
  return before === 0 ? 0 : Math.round((delta / before) * 100 * 100) / 100;
}

/**
 * Re-rates recent freight with the content production rates with (the newest active snapshot,
 * or the row itself on a template never approved) and with the candidate.
 */
export function backtest(t, candidate, limit) {
  const current = draftOf(approvedVersion(t)?.content ?? contentOf(t));
  const next = draftOf(candidate);
  const results = backtestShipments()
    .slice(0, limit)
    .map((shipment) => {
      const before = bill(current, null, shipment.values);
      const after = bill(next, null, shipment.values);
      const row = {
        shipmentId: shipment.id,
        proNumber: shipment.pro,
        currentAmount: decimal(before.amount ?? 0),
        candidateAmount: decimal(after.amount ?? 0),
        delta: "0",
        deltaPct: "0",
        guardrailApplied: Boolean(before.clamp || after.clamp),
        _before: before.amount,
        _after: after.amount,
      };
      if (before.error) row.currentError = before.error;
      if (after.error) row.candidateError = after.error;
      if (!before.error && !after.error) {
        row.delta = decimal(after.amount - before.amount);
        row.deltaPct = decimal(percent(after.amount - before.amount, before.amount));
      }
      return row;
    });
  return { results, summary: summarize(results) };
}

function summarize(results) {
  const s = {
    shipmentCount: 0,
    evaluatedCount: 0,
    changedCount: 0,
    increasedCount: 0,
    decreasedCount: 0,
    errorCount: 0,
    currentErrorCount: 0,
    candidateErrorCount: 0,
    guardrailCount: 0,
  };
  let before = 0;
  let after = 0;
  let maxIncrease = 0;
  let maxDecrease = 0;
  for (const row of results) {
    s.shipmentCount += 1;
    if (row.currentError) s.currentErrorCount += 1;
    if (row.candidateError) s.candidateErrorCount += 1;
    if (row.currentError || row.candidateError) {
      s.errorCount += 1;
      continue;
    }
    if (row.guardrailApplied) s.guardrailCount += 1;
    s.evaluatedCount += 1;
    before += row._before;
    after += row._after;
    const delta = Math.round((row._after - row._before) * 1e6) / 1e6;
    if (delta > 0) {
      s.changedCount += 1;
      s.increasedCount += 1;
      maxIncrease = Math.max(maxIncrease, delta);
    } else if (delta < 0) {
      s.changedCount += 1;
      s.decreasedCount += 1;
      maxDecrease = Math.min(maxDecrease, delta);
    }
  }
  return {
    ...s,
    currentTotal: decimal(before),
    candidateTotal: decimal(after),
    totalDelta: decimal(after - before),
    totalDeltaPct: decimal(percent(after - before, before)),
    maxIncrease: decimal(maxIncrease),
    maxDecrease: decimal(maxDecrease),
  };
}

export const publicBacktest = ({ results, summary }) => ({
  results: results.map(({ _before, _after, ...row }) => row),
  summary,
});

function runBacktest(state, path, body) {
  const t = templateOf(state, path);
  if (!t) return notFound("Formula template");
  const hasExpression = Boolean(body?.expression);
  const hasVersion = body?.versionNumber !== undefined && body?.versionNumber !== null;
  if (hasExpression === hasVersion) {
    const message = hasExpression
      ? "Provide either an expression or a version number, not both"
      : "Either an expression or a version number is required";
    const code = hasExpression ? "INVALID" : "REQUIRED";
    return problem(400, "validation-error", "Validation Failed", message, [
      { field: "expression", code, message },
      { field: "versionNumber", code, message },
    ]);
  }
  const limit = Number(body.limit) || 50;
  if (limit > 500) return invalid("limit", "Limit cannot exceed 500");
  let candidate = { ...contentOf(t), expression: body.expression };
  if (hasVersion) {
    const version = t.versions.find((v) => v.versionNumber === Number(body.versionNumber));
    if (!version) return notFound("Formula template version");
    candidate = contentOf(version.content);
  }
  return publicBacktest(backtest(t, candidate, limit));
}

/** What approving would change: the saved content against production, biggest moves first. */
export function impact(t, limit = 100) {
  const result = backtest(t, contentOf(t), limit);
  result.results.sort((a, b) => {
    const af = Boolean(a.currentError || a.candidateError);
    const bf = Boolean(b.currentError || b.candidateError);
    if (af !== bf) return af ? 1 : -1;
    return Math.abs(Number(b.delta)) - Math.abs(Number(a.delta));
  });
  return result;
}

function approvalImpact(state, path, body) {
  const t = templateOf(state, path);
  if (!t) return notFound("Formula template");
  const limit = Number(body?.limit) || 100;
  if (limit > 500) return invalid("limit", "Limit cannot exceed 500");
  return publicBacktest(impact(t, limit));
}

// --- Readiness -------------------------------------------------------------------------------

/** The review gate, computed ahead of time (contract §4): what Submit and Approve would refuse. */
export function readiness(state, t) {
  const now = state.now();
  const content = contentOf(t);
  const checks = [];

  const details = { key: "details", label: "Name and description" };
  if (!content.name.trim()) Object.assign(details, { status: "fail", detail: "Name is missing" });
  else if (!content.description.trim()) Object.assign(details, { status: "fail", detail: "Description is missing" });
  else Object.assign(details, { status: "pass", detail: "Present" });
  checks.push(details);

  const expression = { key: "expression", label: "Expression is valid" };
  const draft = draftOf(content);
  const sample = run(draft, null, null);
  const parseIssue = analyze(content.expression, context(content.variableDefinitions)).issues.find((i) => i.sev === "error");
  if (!content.expression.trim()) Object.assign(expression, { status: "fail", detail: "Expression is required" });
  else if (parseIssue) Object.assign(expression, { status: "fail", detail: parseIssue.msg });
  else if (!sample.ok && /Rate table/.test(sample.error ?? "")) Object.assign(expression, { status: "fail", detail: sample.error });
  else Object.assign(expression, { status: "pass", detail: "Compiles, and every rate table it names exists" });
  checks.push(expression);

  const nullable = { key: "nullableFields", label: "Safe on every shipment" };
  const names = [...new Set(nullableFields(content).map((item) => item.name))];
  if (names.length > 0) {
    Object.assign(nullable, {
      status: "warn",
      detail: `${names.length} ${names.length === 1 ? "value" : "values"} can be empty: ${names.join(", ")}`,
    });
  } else {
    Object.assign(nullable, { status: "pass", detail: "Every optional field is guarded" });
  }
  checks.push(nullable);

  const scenarioCheck = { key: "scenarios", label: "Scenarios pass" };
  const scenarioRun = runScenarios(t, content);
  const failing = scenarioRun.results.filter((r) => !r.passed).map((r) => r.name);
  if (scenarioRun.total === 0) Object.assign(scenarioCheck, { status: "fail", detail: "Add at least one" });
  else if (failing.length > 0) Object.assign(scenarioCheck, { status: "fail", detail: `${scenarioRun.passed} of ${scenarioRun.total}` });
  else Object.assign(scenarioCheck, { status: "pass", detail: `${scenarioRun.passed} of ${scenarioRun.total}` });
  checks.push(scenarioCheck);

  if (content.breakdownDefinitions.length > 0) {
    const breakdown = { key: "breakdown", label: "Breakdown reconciles" };
    if (!sample.ok) {
      Object.assign(breakdown, { status: "warn", detail: `The sample load does not price: ${sample.error}` });
    } else if (!sample.reconcile.balanced) {
      Object.assign(breakdown, {
        status: "warn",
        detail: sample.reconcile.failed
          ? `${sample.reconcile.failed} of ${sample.lines.length} lines fail on the sample load`
          : `The lines add up to ${decimal(sample.reconcile.sum)}, not the formula's ${decimal(sample.value)} on the sample load`,
      });
    } else if (sample.policy.clamp) {
      Object.assign(breakdown, {
        status: "warn",
        detail: `The ${sample.policy.clamp.bound === "min" ? "minimum" : "maximum"} moved the total to ${decimal(sample.policy.amount)}; the lines add up to ${decimal(sample.reconcile.sum)}`,
      });
    } else {
      Object.assign(breakdown, { status: "pass", detail: "The lines add up to the charge on the sample load" });
    }
    checks.push(breakdown);
  }

  const status = { key: "status", label: "Review state" };
  if (t.status === "Draft" || t.status === "Inactive") Object.assign(status, { status: "pass", detail: "Ready to submit for review" });
  else if (t.status === "InReview") Object.assign(status, { status: "pass", detail: "In review; awaiting a decision" });
  else Object.assign(status, { status: "pass", detail: "Active; editing the content will return it to draft" });
  checks.push(status);

  const reviewer = { key: "reviewer", label: "Independent reviewer" };
  if (t.status !== "InReview") Object.assign(reviewer, { status: "pass", detail: "Applies once the template is in review" });
  else if (t.submittedById === USER.id) Object.assign(reviewer, { status: "fail", detail: "You submitted this template; someone else must approve it" });
  else Object.assign(reviewer, { status: "pass", detail: "You did not submit this template" });
  checks.push(reviewer);

  const age = { key: "submissionAge", label: "Submission age" };
  if (t.status !== "InReview" || !t.submittedAt) {
    Object.assign(age, { status: "pass", detail: "Applies once the template is in review" });
  } else {
    const waited = now - t.submittedAt;
    const days = Math.floor(waited / DAY);
    if (waited > STALE_AFTER) Object.assign(age, { status: "fail", detail: `Submitted ${days} days ago; older than the 14-day limit, so it must be resubmitted` });
    else if (waited > AGING_AFTER) Object.assign(age, { status: "warn", detail: `Submitted ${days} days ago; decide soon or it expires at 14` });
    else Object.assign(age, { status: "pass", detail: "Submitted recently" });
  }
  checks.push(age);

  const blocking = [details, expression, scenarioCheck].some((check) => check.status === "fail");
  return {
    canSubmit: !blocking && (t.status === "Draft" || t.status === "Inactive"),
    canApprove: !blocking && t.status === "InReview" && reviewer.status !== "fail" && age.status !== "fail",
    checks,
    scenarioTotal: scenarioRun.total,
    scenarioPassed: scenarioRun.passed,
    ...(failing.length > 0 ? { scenarioFailing: failing } : {}),
  };
}

// --- Templates -------------------------------------------------------------------------------

function listTemplates(state, _path, _body, query) {
  const limit = Math.min(Number(query?.get("limit")) || 20, 100);
  const offset = Math.max(Number(query?.get("offset")) || 0, 0);
  const type = query?.get("type");
  const status = query?.get("status");
  const q = (query?.get("query") ?? query?.get("search") ?? "").toLowerCase();
  const rows = [...studio(state).templates.values()]
    .filter((t) => (!type || t.type === type) && (!status || t.status === status))
    .filter((t) => !q || `${t.name} ${t.description}`.toLowerCase().includes(q))
    .sort((a, b) => b.updatedAt - a.updatedAt || (a.id < b.id ? -1 : 1));
  const results = rows.slice(offset, offset + limit).map(publicTemplate);
  const next = offset + limit < rows.length ? `${BASE}/?limit=${limit}&offset=${offset + limit}${type ? `&type=${type}` : ""}` : null;
  const prev = offset > 0 ? `${BASE}/?limit=${limit}&offset=${Math.max(offset - limit, 0)}${type ? `&type=${type}` : ""}` : null;
  return { results, count: rows.length, next, prev };
}

function getTemplate(state, path) {
  const t = templateOf(state, path);
  return t ? publicTemplate(t) : notFound("Formula template");
}

function createTemplate(state, _path, body) {
  const s = studio(state);
  const content = normalizeContent(body ?? {});
  const errors = contentErrors(content);
  const held = Array.isArray(body?.testCases) ? body.testCases : [];
  if (held.length > MAX_HELD_SCENARIOS) {
    errors.push({ field: "testCases", code: "INVALID", message: `A template can be created with at most ${MAX_HELD_SCENARIOS} scenarios` });
  }
  held.forEach((tc, index) => errors.push(...testCaseErrors(tc, index)));
  if (errors.length > 0) return refusedContent(errors);
  const now = state.now();
  const id = `fmt_studio_new_${s.nextTemplate}`;
  s.nextTemplate += 1;
  const t = {
    id,
    status: "Draft",
    ...content,
    submittedById: null,
    submittedAt: null,
    approvedById: null,
    approvedAt: null,
    reviewComment: "",
    metadata: body?.metadata ?? {},
    version: 1,
    sourceTemplateId: body?.sourceTemplateId ?? null,
    sourceVersionNumber: body?.sourceVersionNumber ?? null,
    currentVersionNumber: 1,
    createdAt: now,
    updatedAt: now,
    versions: [{ versionNumber: 1, status: "Draft", createdById: USER.id, createdAt: now, changeMessage: "Initial version", tags: [], content: contentOf(content) }],
    usages: [],
    testCases: [],
    reviews: [],
  };
  t.testCases = held.map((tc) => newTestCase(s, t, tc, now));
  s.templates.set(id, t);
  return { __status: 201, body: publicTemplate(t) };
}

/** Update: a material change takes a new version and returns an approved template to draft. */
function updateTemplate(state, path, body, { merge = false } = {}) {
  const t = templateOf(state, path);
  if (!t) return notFound("Formula template");
  const original = contentOf(t);
  const content = normalizeContent(body ?? {}, merge ? original : {});
  const errors = contentErrors(content);
  if (errors.length > 0) return refusedContent(errors);
  if (body?.status && STATUSES.has(body.status) && body.status !== t.status) {
    return invalid(
      "status",
      `Status cannot be changed from ${t.status} to ${body.status} here; submit, approve, reject, or archive the template instead`,
    );
  }
  const changes = diffContent(original, content);
  const material = Object.keys(changes).some((key) => MATERIAL_KEYS.some((m) => key === m || key.startsWith(`${m}.`)));
  const now = state.now();
  Object.assign(t, content, { version: t.version + 1, updatedAt: now });
  if (material) {
    if (t.status === "Active" || t.status === "InReview") {
      t.status = "Draft";
      t.submittedById = null;
      t.submittedAt = null;
      t.approvedById = null;
      t.approvedAt = null;
    }
    t.currentVersionNumber += 1;
    t.versions.push({ versionNumber: t.currentVersionNumber, status: t.status, createdById: USER.id, createdAt: now, changeMessage: "", tags: [], content: contentOf(t) });
  }
  return publicTemplate(t);
}

// --- Scenarios -------------------------------------------------------------------------------

function listTestCases(state, path) {
  const t = templateOf(state, path);
  return t ? t.testCases.map(publicTestCase) : notFound("Formula template");
}

function createTestCase(state, path, body) {
  const t = templateOf(state, path);
  if (!t) return notFound("Formula template");
  const errors = testCaseErrors(body);
  if (errors.length > 0) return refusedContent(errors);
  const tc = newTestCase(studio(state), t, body, state.now());
  t.testCases.push(tc);
  return { __status: 201, body: publicTestCase(tc) };
}

function updateTestCase(state, path, body) {
  const t = templateOf(state, path);
  if (!t) return notFound("Formula template");
  const tc = t.testCases.find((c) => c.id === testCaseIdOf(path));
  if (!tc) return notFound("Formula test case");
  const errors = testCaseErrors(body);
  if (errors.length > 0) return refusedContent(errors);
  if (body.version !== undefined && Number(body.version) !== tc.version) {
    return problem(409, "resource-conflict", "Resource Conflict", "The scenario changed since you opened it. Reload and try again.");
  }
  const now = state.now();
  Object.assign(tc, {
    name: String(body.name).trim(),
    description: body.description ?? "",
    variables: body.variables ?? {},
    expectedAmount: decimal(Number(body.expectedAmount)),
    tolerance: decimal(body.tolerance === undefined ? Number(tc.tolerance) : Number(body.tolerance)),
    version: tc.version + 1,
    updatedAt: now,
    ...(body.candidate ? recording(t, body.candidate, now) : {}),
  });
  return publicTestCase(tc);
}

function deleteTestCase(state, path) {
  const t = templateOf(state, path);
  if (!t) return notFound("Formula template");
  const index = t.testCases.findIndex((c) => c.id === testCaseIdOf(path));
  if (index === -1) return notFound("Formula test case");
  t.testCases.splice(index, 1);
  return { __status: 204, body: undefined };
}

function runTestCases(state, path, body) {
  const t = templateOf(state, path);
  if (!t) return notFound("Formula template");
  const content = body?.candidate ? normalizeContent(body.candidate, contentOf(t)) : contentOf(t);
  return runScenarios(t, content);
}

// --- Review ----------------------------------------------------------------------------------

/** Active teammates who hold formula template approve, without the person asking. */
export function reviewers() {
  return TEAM.filter((person) => person.id !== USER.id).map((person) => {
    const user = userRecord(person.id);
    return { id: user.id, name: user.name, username: user.username, emailAddress: user.emailAddress, profilePicUrl: user.profilePicUrl };
  });
}

/** recordReview: a submission opens or continues a round; a decision lands in the open one. */
function recordReview(t, decision, comment, now, extra = {}) {
  const latest = t.reviews.reduce((best, r) => (!best || r.createdAt >= best.createdAt ? r : best), null);
  const closed = latest && ["Approved", "Rejected", "Expired"].includes(latest.decision);
  const round = !latest ? 1 : decision === "Submitted" && closed ? latest.round + 1 : latest.round;
  t.reviews.push({
    id: `ftr_${t.id.slice(4)}_${t.reviews.length + 1}`,
    templateId: t.id,
    round,
    decision,
    actorId: USER.id,
    comment: comment ?? "",
    baseVersionNumber: decision === "Submitted" ? (approvedVersion(t)?.versionNumber ?? 0) : (latest?.baseVersionNumber ?? 0),
    createdAt: now,
    requestedReviewerId: extra.reviewerId ?? null,
    reviewerNote: extra.reviewerNote ?? "",
  });
}

function submit(state, path, body) {
  const t = templateOf(state, path);
  if (!t) return notFound("Formula template");
  if (t.status !== "Draft" && t.status !== "Inactive") {
    return business(`Only a draft or inactive template can be submitted; this one is ${t.status}`);
  }
  const note = String(body?.reviewerNote ?? "");
  if ([...note].length > MAX_REVIEWER_NOTE) {
    return problem(422, "validation-error", "Validation Failed", `The note can be at most ${MAX_REVIEWER_NOTE} characters`, [
      { field: "reviewerNote", code: "INVALID", message: `The note can be at most ${MAX_REVIEWER_NOTE} characters` },
    ]);
  }
  const reviewerId = body?.reviewerId ? String(body.reviewerId) : null;
  if (reviewerId && !reviewers().some((r) => r.id === reviewerId)) {
    return problem(422, "validation-error", "Validation Failed", "That person cannot approve formula templates", [
      { field: "reviewerId", code: "INVALID", message: "That person cannot approve formula templates" },
    ]);
  }
  const ready = readiness(state, t);
  const blocking = ready.checks.filter((check) => ["details", "expression", "scenarios"].includes(check.key) && check.status === "fail");
  if (blocking.length > 0) {
    return problem(
      422,
      "business-rule-violation",
      "Business Rule Violation",
      "This template is not ready for review",
      blocking.map((check) => ({ field: check.key, code: "INVALID", message: `${check.label}: ${check.detail}` })),
    );
  }
  const now = state.now();
  Object.assign(t, { status: "InReview", submittedById: USER.id, submittedAt: now, reviewComment: body?.comment ?? "", version: t.version + 1, updatedAt: now });
  recordReview(t, "Submitted", body?.comment, now, { reviewerId, reviewerNote: note });
  return publicTemplate(t);
}

function approve(state, path, body) {
  const t = templateOf(state, path);
  if (!t) return notFound("Formula template");
  if (t.status !== "InReview") return business(`Only a template in review can be approved; this one is ${t.status}`);
  if (t.submittedById === USER.id) return invalid("approvedById", "A template cannot be approved by its submitter");
  const now = state.now();
  if (t.submittedAt && now - t.submittedAt > STALE_AFTER) {
    return business("This submission is older than 14 days; the author must resubmit it");
  }
  const errors = contentErrors(contentOf(t));
  if (errors.length > 0) return refusedContent(errors);
  const scenarioRun = runScenarios(t, contentOf(t));
  if (scenarioRun.total === 0) return business("Add at least one scenario before approving");
  if (scenarioRun.failed > 0) {
    const names = scenarioRun.results.filter((r) => !r.passed).map((r) => r.name);
    return business(`Every scenario must pass before approval; failing: ${names.join(", ")}`);
  }
  t.currentVersionNumber += 1;
  Object.assign(t, { status: "Active", approvedById: USER.id, approvedAt: now, reviewComment: body?.comment ?? "", version: t.version + 1, updatedAt: now });
  t.versions.push({ versionNumber: t.currentVersionNumber, status: "Active", createdById: USER.id, createdAt: now, changeMessage: "Approved", tags: [], content: contentOf(t) });
  recordReview(t, "Approved", body?.comment, now);
  return publicTemplate(t);
}

function decide(decision) {
  return (state, path, body) => {
    const t = templateOf(state, path);
    if (!t) return notFound("Formula template");
    if (t.status !== "InReview") return business(`Only a template in review can be decided; this one is ${t.status}`);
    const comment = String(body?.comment ?? "").trim();
    if (!comment) {
      return invalid("comment", decision === "Rejected" ? "A comment is required when rejecting a formula template" : "A comment is required when requesting changes", "REQUIRED");
    }
    if (t.submittedById === USER.id) {
      return invalid("approvedById", "A template's submitter cannot review it; ask another reviewer or archive it");
    }
    const now = state.now();
    if (decision === "Rejected") Object.assign(t, { status: "Inactive", submittedById: null, submittedAt: null });
    else t.status = "Draft";
    Object.assign(t, { reviewComment: comment, version: t.version + 1, updatedAt: now });
    recordReview(t, decision, comment, now);
    return publicTemplate(t);
  };
}

function listReviews(state, path) {
  const t = templateOf(state, path);
  if (!t) return notFound("Formula template");
  return [...t.reviews].sort((a, b) => b.createdAt - a.createdAt || b.id.localeCompare(a.id)).map(publicReview);
}

export function reviewDiff(t) {
  const base = approvedVersion(t);
  const changes = diffContent(base ? base.content : {}, contentOf(t));
  return {
    hasApprovedBase: Boolean(base),
    baseVersion: base?.versionNumber ?? 0,
    currentVersion: t.currentVersionNumber,
    baseExpression: base?.content.expression ?? "",
    currentExpression: t.expression,
    changes,
    changeCount: Object.keys(changes).length,
  };
}

// --- Versions --------------------------------------------------------------------------------

function listVersions(state, path, _body, query) {
  const t = templateOf(state, path);
  if (!t) return notFound("Formula template");
  const limit = Number(query?.get("limit")) || 20;
  const offset = Number(query?.get("offset")) || 0;
  const rows = [...t.versions].reverse();
  return {
    results: rows.slice(offset, offset + limit).map((v) => publicVersion(t, v)),
    count: rows.length,
    next: offset + limit < rows.length ? `${BASE}/${t.id}/versions?limit=${limit}&offset=${offset + limit}` : null,
    prev: offset > 0 ? `${BASE}/${t.id}/versions?limit=${limit}&offset=${Math.max(offset - limit, 0)}` : null,
  };
}

function getVersion(state, path) {
  const t = templateOf(state, path);
  if (!t) return notFound("Formula template");
  const n = Number(/\/versions\/(\d+)/.exec(path)[1]);
  const version = t.versions.find((v) => v.versionNumber === n);
  return version ? publicVersion(t, version) : notFound("Formula template version");
}

function compareVersions(state, path, _body, query) {
  const t = templateOf(state, path);
  if (!t) return notFound("Formula template");
  const from = t.versions.find((v) => v.versionNumber === Number(query?.get("from")));
  const to = t.versions.find((v) => v.versionNumber === Number(query?.get("to")));
  if (!from || !to) return notFound("Formula template version");
  const changes = diffContent(from.content, to.content);
  return { fromVersion: from.versionNumber, toVersion: to.versionNumber, changes, changeCount: Object.keys(changes).length };
}

function lineage(state, path) {
  const t = templateOf(state, path);
  if (!t) return notFound("Formula template");
  const all = [...studio(state).templates.values()];
  const root = t.sourceTemplateId ? (all.find((x) => x.id === t.sourceTemplateId) ?? t) : t;
  const node = (x) => ({
    templateId: x.id,
    templateName: x.name,
    sourceTemplateId: x.sourceTemplateId,
    sourceVersion: x.sourceVersionNumber,
    forkedTemplates: all.filter((child) => child.sourceTemplateId === x.id).map(node),
  });
  return node(root);
}

// --- Schema ----------------------------------------------------------------------------------

const VARIABLE_CATEGORY = {
  charges: () => "computed",
  computed: () => "computed",
  shipment: () => "shipment",
  customer: () => "customer",
  equipment: (name) => name.split(".")[0],
  location: (name) => name.split(".")[0],
};
const FUNCTION_CATEGORY = { Math: "math", Rounding: "rounding", Aggregate: "aggregate", "Rate tables": "rateTable" };

/** DescribeSchema: the shipment schema and the function library, as the server lists them. */
function describeSchema(_state, _path, _body, query) {
  const schemaId = query?.get("schemaId") || "shipment";
  if (schemaId !== "shipment") return invalid("schemaId", `Unknown formula schema: ${schemaId}`);
  const variables = VARIABLES.map((v) => ({
    name: v.name,
    type: v.type.toLowerCase(),
    description: v.description,
    category: VARIABLE_CATEGORY[v.category](v.name),
    nullable: Boolean(v.nullable),
    computed: v.category === "computed" || v.category === "charges",
  })).sort((a, b) => a.category.localeCompare(b.category) || a.name.localeCompare(b.name));
  const functions = FUNCTIONS.map((f) => ({
    name: f.name,
    signature: f.signature,
    description: f.description,
    example: f.example,
    category: f.name === "coalesce" ? "conditional" : FUNCTION_CATEGORY[f.category],
    operator: false,
  }));
  return { schemaId, variables, functions };
}

function listStandards() {
  return STANDARDS.map((standard) => ({
    name: standard.name,
    description: standard.description,
    type: standard.type,
    expression: standard.expression,
    schemaId: "shipment",
    variableDefinitions: standard.variables.map((v) => ({ ...v })),
  }));
}

// --- GraphQL ---------------------------------------------------------------------------------

/** FormulaTemplateTable: the list page, newest first. */
function formulaTemplateTable(state, v) {
  const input = v.input ?? {};
  const rows = [...studio(state).templates.values()].sort((a, b) => b.updatedAt - a.updatedAt);
  const offset = input.after ? Number(Buffer.from(input.after, "base64url").toString()) : 0;
  const page = rows.slice(offset, offset + (input.first ?? 20));
  const end = offset + page.length;
  return {
    formulaTemplates: {
      edges: page.map((t) => ({ node: { __typename: "FormulaTemplate", ...publicTemplate(t) } })),
      ...(v.includeTotalCount === false ? {} : { totalCount: rows.length }),
      pageInfo: {
        hasNextPage: end < rows.length,
        endCursor: page.length ? Buffer.from(String(end)).toString("base64url") : null,
        hasPreviousPage: offset > 0,
        startCursor: null,
      },
    },
  };
}

/** Select options the studio asks for: the "copy from" template picker and the shipment picker. */
export function formulaSelectRows(state, resource) {
  if (resource === "FORMULA_TEMPLATE") {
    return [...studio(state).templates.values()]
      .sort((a, b) => b.updatedAt - a.updatedAt)
      .map((t) => ({
        id: t.id,
        label: t.name,
        description: `${t.status === "InReview" ? "In review" : t.status} · v${t.currentVersionNumber}`,
        meta: { type: t.type, status: t.status, currentVersionNumber: t.currentVersionNumber },
      }));
  }
  if (resource === "SHIPMENT") {
    return SHIPMENTS.map((s) => ({ id: s.id, label: s.pro, description: s.lane, meta: { customer: s.customer, summary: s.meta } }));
  }
  return [];
}

export const FORMULA_HANDLERS = {
  FormulaTemplateTable: formulaTemplateTable,
};

// --- Routes ----------------------------------------------------------------------------------

const route = (rest) => new RegExp(`^${BASE}${rest}/?$`);
const one = (rest = "") => route(`/${ID}${rest}`);

export const FORMULA_ROUTES = [
  ["GET", route(""), listTemplates],
  ["POST", route(""), createTemplate],
  ["GET", route("/schema"), describeSchema],
  ["GET", route("/standards"), listStandards],
  ["POST", route("/test"), testExpression],
  ["GET", one(), getTemplate],
  ["PUT", one(), (s, p, b) => updateTemplate(s, p, b)],
  ["PATCH", one(), (s, p, b) => updateTemplate(s, p, b, { merge: true })],
  [
    "GET",
    one("/usage"),
    (s, p) => {
      const t = templateOf(s, p);
      return t ? { inUse: t.usages.length > 0, usages: t.usages } : notFound("Formula template");
    },
  ],
  ["GET", one("/versions"), listVersions],
  ["GET", one("/versions/scheduled"), (s, p) => (templateOf(s, p) ? [] : notFound("Formula template"))],
  ["GET", one("/versions/\\d+"), getVersion],
  ["GET", one("/compare"), compareVersions],
  ["GET", one("/lineage"), lineage],
  ["GET", one("/test-cases"), listTestCases],
  ["POST", one("/test-cases"), createTestCase],
  ["POST", one("/test-cases/run"), runTestCases],
  ["PUT", one(`/test-cases/${TC}`), updateTestCase],
  ["DELETE", one(`/test-cases/${TC}`), deleteTestCase],
  [
    "GET",
    one("/readiness"),
    (s, p) => {
      const t = templateOf(s, p);
      return t ? readiness(s, t) : notFound("Formula template");
    },
  ],
  ["GET", one("/reviews"), listReviews],
  [
    "GET",
    one("/review-diff"),
    (s, p) => {
      const t = templateOf(s, p);
      return t ? reviewDiff(t) : notFound("Formula template");
    },
  ],
  ["GET", one("/reviewers"), (s, p) => (templateOf(s, p) ? reviewers() : notFound("Formula template"))],
  ["POST", one("/backtest"), runBacktest],
  ["POST", one("/impact"), approvalImpact],
  ["POST", one("/submit"), submit],
  ["POST", one("/approve"), approve],
  ["POST", one("/reject"), decide("Rejected")],
  ["POST", one("/request-changes"), decide("ChangesRequested")],
];
