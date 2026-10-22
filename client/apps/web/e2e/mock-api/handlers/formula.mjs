// The Formula Studio's REST routes and GraphQL operations on Tariff: templates, versions,
// review, readiness, backtest and impact, the organisation's context, the preview with its bill
// and trace, "Save as test" and the standard templates. Every check, test, bill and scaffold is
// the rules module's (../rules.mjs, the rules.wasm the client loads), so the Studio, its tests
// panel, the backtest and Nova agree. Writes keep formulatemplateservice's rules: a template is
// born a draft, a change to its source or name takes a new version and returns an approved or
// in-review template to draft, submit is gated on readiness, and an approval takes a version.
import { rules, recordFrom, RulesError, stop } from "../rules.mjs";
import {
  backtestShipments,
  CODES,
  createFormulaLibraries,
  createFormulaTemplates,
  ORG,
  SETTINGS,
  SHIPMENTS,
  shipmentOverlay,
  STANDARDS,
  TEAM,
  USER,
  userRecord,
} from "../fixtures/formula.mjs";

const BASE = "/api/v1/formula-templates";
const ID = "fmt_[A-Za-z0-9_]+";
const DAY = 86400;
export const STALE_AFTER = 14 * DAY;
export const AGING_AFTER = 7 * DAY;
const MAX_REVIEWER_NOTE = 4000;
const MAX_NAME = 100;
const STATUSES = new Set(["Active", "Inactive", "Draft", "InReview"]);
const CONTENT_KEYS = ["name", "description", "source"];

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

/** The scenario's templates and libraries, built on first use so every scenario starts fresh. */
export function studio(state) {
  if (state.formulaStudio) return state.formulaStudio;
  state.formulaStudio = {
    templates: new Map(createFormulaTemplates(state.anchor).map((t) => [t.id, t])),
    libraries: new Map(createFormulaLibraries(state.anchor).map((l) => [l.id, l])),
    nextId: 1,
  };
  return state.formulaStudio;
}

export function nextId(state, prefix) {
  const s = studio(state);
  const id = `${prefix}${s.nextId}`;
  s.nextId += 1;
  return id;
}

const templateIdOf = (path) => new RegExp(`/formula-templates/(${ID})`).exec(path)?.[1];

export function templateOf(state, path) {
  return studio(state).templates.get(templateIdOf(path)) ?? null;
}

// --- The organisation's context and the module -----------------------------------------------

/** The newest approved version of a template or a library, or null when none was approved. */
export function approvedVersion(record) {
  return [...record.versions].reverse().find((version) => version.status === "Active") ?? null;
}

export const libraryPath = (library) => `org.${library.name}`;
export const libraryVersionId = (library, versionNumber) => `flv_${library.id.slice(3)}_${versionNumber}`;

/**
 * What every check is given: the approved version of each library that is not archived (a
 * library is left out of its own context), the accessorial codes and the settings.
 */
export function tariffContext(state, { except } = {}) {
  const libraries = [];
  for (const library of studio(state).libraries.values()) {
    if (library.id === except || library.status === "Inactive") continue;
    const approved = approvedVersion(library);
    if (!approved) continue;
    libraries.push({
      path: libraryPath(library),
      version: libraryVersionId(library, approved.versionNumber),
      source: approved.content.source,
    });
  }
  return { libraries, tables: [], codes: CODES, settings: SETTINGS };
}

const checks = new Map();

/** The module's check of a source, cached per source, kind and context. */
export function checkOf(source, kind, context) {
  const key = `${kind}\u0000${context.libraries.map((l) => l.version).join(",")}\u0000${source}`;
  let result = checks.get(key);
  if (!result) {
    result = rules("tariff.check", { source, kind, context });
    if (checks.size > 500) checks.clear();
    checks.set(key, result);
  }
  return result;
}

/** The module's run of a source's tests; a source that does not check runs none. */
export function testsOf(source, context) {
  return rules("tariff.test", { source, context });
}

/** Pins each `org.` import to the library and the approved version the check was given. */
export function pinsOf(state, check) {
  const byPath = new Map([...studio(state).libraries.values()].map((l) => [libraryPath(l), l]));
  return check.imports.flatMap((entry) => {
    const library = byPath.get(entry.path);
    const approved = library ? approvedVersion(library) : null;
    return library && approved
      ? [{ path: entry.path, libraryId: library.id, versionNumber: approved.versionNumber }]
      : [];
  });
}

/** What the server derives from a template's source when it saves it. */
function derived(state, source) {
  const check = checkOf(source, "tariff", tariffContext(state));
  return {
    inputs: check.inputs,
    imports: pinsOf(state, check),
    testCount: check.outline?.items.filter((item) => item.kind === "test").length ?? 0,
  };
}

const firstError = (check) => check.diagnostics.find((finding) => finding.severity === "error") ?? null;

/** A module message in the server's words (English), for a problem's detail. */
export function messageText(message) {
  if (typeof message === "string") return message;
  const args = message.args ?? [];
  return message.message.replace(/\{(\d+)\}/g, (_, n) => {
    const arg = args[Number(n)];
    return typeof arg === "object" && arg !== null ? messageText(arg) : String(arg ?? "");
  });
}

// --- Shapes ---------------------------------------------------------------------------------

function usageCount(t) {
  return t.usages.reduce((sum, usage) => sum + usage.count, 0);
}

export function publicTemplate(state, t) {
  const approved = approvedVersion(t);
  return {
    id: t.id,
    organizationId: ORG.id,
    businessUnitId: USER.businessUnitId,
    name: t.name,
    description: t.description,
    source: t.source,
    status: t.status,
    ...derived(state, t.source),
    submittedById: t.submittedById,
    submittedAt: t.submittedAt,
    approvedById: t.approvedById,
    approvedAt: t.approvedAt,
    reviewComment: t.reviewComment,
    version: t.version,
    sourceTemplateId: t.sourceTemplateId,
    sourceVersionNumber: t.sourceVersionNumber,
    currentVersionNumber: t.currentVersionNumber,
    billing: approved ? { versionNumber: approved.versionNumber, effectiveFrom: approved.createdAt } : null,
    createdAt: t.createdAt,
    updatedAt: t.updatedAt,
    usageCount: usageCount(t),
  };
}

export function publicReview(review, key) {
  return {
    organizationId: ORG.id,
    businessUnitId: USER.businessUnitId,
    ...review,
    [key]: review[key],
    actor: review.actorId ? userRecord(review.actorId) : null,
    requestedReviewer: review.requestedReviewerId ? userRecord(review.requestedReviewerId) : null,
  };
}

/** The content a version snapshot holds. */
export function contentOf(record) {
  return Object.fromEntries(CONTENT_KEYS.map((key) => [key, record[key]]));
}

function publicVersion(state, t, version) {
  const previous = t.versions.find((v) => v.versionNumber === version.versionNumber - 1);
  return {
    id: `ftv_${t.id.slice(4)}_${version.versionNumber}`,
    templateId: t.id,
    organizationId: ORG.id,
    businessUnitId: USER.businessUnitId,
    versionNumber: version.versionNumber,
    ...version.content,
    ...derived(state, version.content.source),
    status: version.status,
    effectiveFrom: version.status === "Active" ? version.createdAt : null,
    changeMessage: version.changeMessage ?? "",
    changeSummary: previous ? diffContent(previous.content, version.content) : null,
    tags: version.tags ?? [],
    createdById: version.createdById,
    createdAt: version.createdAt,
    createdBy: userRecord(version.createdById),
  };
}

// --- Diffs, as jsonutils.JSONDiff reports them ------------------------------------------------

function fieldType(value) {
  if (value === null || value === undefined) return "null";
  if (Array.isArray(value)) return "array";
  return typeof value;
}

/** One entry per changed content field. */
export function diffContent(from, to, keys = CONTENT_KEYS) {
  const changes = {};
  for (const key of keys) {
    const a = from?.[key] ?? null;
    const b = to?.[key] ?? null;
    if (JSON.stringify(a) === JSON.stringify(b)) continue;
    const type = a === null || a === "" ? "created" : b === null || b === "" ? "deleted" : "updated";
    changes[key] = { from: a, to: b, type, fieldType: fieldType(b ?? a), path: key };
  }
  return changes;
}

// --- Writes' input ---------------------------------------------------------------------------

/** A save's content, trimmed as the server stores it. */
function normalizeContent(body, fallback = {}) {
  const pick = (key) => (body?.[key] !== undefined ? body[key] : (fallback[key] ?? ""));
  return {
    name: String(pick("name") ?? "").trim(),
    description: String(pick("description") ?? ""),
    source: String(pick("source") ?? ""),
  };
}

/** The field errors a save is refused with; a draft may hold findings, review may not. */
function contentErrors(content) {
  const errors = [];
  if (!content.name) errors.push({ field: "name", code: "REQUIRED", message: "Name is required" });
  else if ([...content.name].length > MAX_NAME) {
    errors.push({ field: "name", code: "INVALID", message: `Name must be at most ${MAX_NAME} characters` });
  }
  if (!content.source.trim()) {
    errors.push({ field: "source", code: "REQUIRED", message: "The tariff's source is required" });
  }
  return errors;
}

const refusedContent = (errors) => problem(400, "validation-error", "Validation Failed", errors[0].message, errors);

// --- Rating -----------------------------------------------------------------------------------

function shipmentRecord(id) {
  const shipment = [...SHIPMENTS, ...backtestShipments()].find((s) => s.id === id);
  return shipment ? recordFrom(shipmentOverlay(shipment, stop)) : null;
}

/** A result's refusal, or why it could not be rated, as text for a backtest row. */
function reasonOf(result) {
  if (result.refusal) return messageText(result.refusal.message);
  if (result.error) return messageText(result.error.message);
  if (result.needs?.length) return `Needs ${result.needs.map((need) => need.name ?? need).join(", ")}`;
  return "Could not be rated";
}

/** The preview (`POST /test`): the check, the bill or why not, the trace and the tests. */
function preview(state, _path, body) {
  const source = String(body?.source ?? "");
  const context = tariffContext(state);
  const record = body?.shipmentId ? shipmentRecord(body.shipmentId) : null;
  if (body?.shipmentId && !record) return notFound("Shipment");
  const check = checkOf(source, "tariff", context);
  const { results } = rules("tariff.evaluate", {
    context,
    record,
    programs: [{ id: "preview", source, inputs: body?.inputs ?? {} }],
    trace: true,
  });
  const result = results[0];
  const needs =
    result.status === "needs"
      ? {
          code: "needs",
          message: { message: "The tariff needs a value for {0}", args: [(result.needs ?? []).map((need) => need.name ?? need).join(", ")] },
          range: null,
        }
      : null;
  return {
    diagnostics: check.diagnostics,
    bill: result.status === "rated" ? result.bill : null,
    refusal: result.status === "refused" ? result.refusal : null,
    error: result.status === "error" ? result.error : needs,
    trace: result.trace ?? [],
    lookups: result.lookups ?? [],
    tests: check.ok ? testsOf(source, context).tests : [],
  };
}

/** "Save as test": the module writes a test from a real shipment's facts and its bill. */
function scaffoldTest(state, _path, body) {
  const name = String(body?.name ?? "").trim();
  if (!name) return invalid("name", "Name the test", "REQUIRED");
  const record = body?.shipmentId ? shipmentRecord(body.shipmentId) : null;
  if (!record) return notFound("Shipment");
  const out = rules("tariff.scaffoldTest", {
    source: String(body?.source ?? ""),
    context: tariffContext(state),
    record,
    inputs: body?.inputs ?? {},
    name,
  });
  if (!out.test) return business(out.error ? messageText(out.error.message ?? out.error) : "This shipment could not be rated, so no test was written");
  return { test: out.test };
}

function percent(delta, before) {
  return before === 0 ? 0 : Math.round((delta / before) * 100 * 100) / 100;
}

const amountOf = (result) => (result.status === "rated" ? Number(result.bill.total.$q) : null);

/** Re-rates recent freight with what production rates with and with the candidate. */
export function backtest(state, t, candidateSource, limit) {
  const context = tariffContext(state);
  const approved = approvedVersion(t);
  const currentSource = approved?.content.source ?? t.source;
  const results = backtestShipments()
    .slice(0, limit)
    .map((shipment) => {
      const record = recordFrom(shipmentOverlay(shipment, stop));
      const { results: rated } = rules("tariff.evaluate", {
        context,
        record,
        programs: [
          { id: "current", source: currentSource, inputs: {} },
          { id: "candidate", source: candidateSource, inputs: {} },
        ],
        trace: false,
      });
      const [before, after] = rated;
      const b = amountOf(before);
      const a = amountOf(after);
      const row = {
        shipmentId: shipment.id,
        proNumber: shipment.pro,
        currentSource: "Recomputed",
        currentAmount: (b ?? 0).toFixed(2),
        candidateAmount: (a ?? 0).toFixed(2),
        delta: "0",
        deltaPct: "0",
        _before: b,
        _after: a,
      };
      if (b === null) row.currentError = reasonOf(before);
      if (after.status === "refused") row.candidateRefusal = reasonOf(after);
      else if (a === null) row.candidateError = reasonOf(after);
      if (b !== null && a !== null) {
        row.delta = (a - b).toFixed(2);
        row.deltaPct = percent(a - b, b).toFixed(2);
      }
      return row;
    });
  return { results, summary: summarize(results), truncated: false };
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
    refusedCount: 0,
  };
  let before = 0;
  let after = 0;
  let maxIncrease = 0;
  let maxDecrease = 0;
  for (const row of results) {
    s.shipmentCount += 1;
    if (row.currentError) s.currentErrorCount += 1;
    if (row.candidateError) s.candidateErrorCount += 1;
    if (row.candidateRefusal) s.refusedCount += 1;
    if (row.currentError || row.candidateError || row.candidateRefusal) {
      if (row.currentError || row.candidateError) s.errorCount += 1;
      continue;
    }
    s.evaluatedCount += 1;
    before += row._before;
    after += row._after;
    const delta = Math.round((row._after - row._before) * 100) / 100;
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
    currentTotal: before.toFixed(2),
    candidateTotal: after.toFixed(2),
    totalDelta: (after - before).toFixed(2),
    totalDeltaPct: percent(after - before, before).toFixed(2),
    maxIncrease: maxIncrease.toFixed(2),
    maxDecrease: maxDecrease.toFixed(2),
  };
}

export const publicBacktest = ({ results, summary, truncated }) => ({
  results: results.map(({ _before, _after, ...row }) => row),
  summary,
  truncated,
});

function runBacktest(state, path, body) {
  const t = templateOf(state, path);
  if (!t) return notFound("Formula template");
  const hasSource = Boolean(body?.source);
  const hasVersion = body?.versionNumber !== undefined && body?.versionNumber !== null;
  if (hasSource === hasVersion) {
    const message = "Provide exactly one of a source or a version number";
    return problem(400, "validation-error", "Validation Failed", message, [
      { field: "source", code: hasSource ? "INVALID" : "REQUIRED", message },
    ]);
  }
  const limit = Number(body.limit) || 50;
  if (limit > 500) return invalid("limit", "Limit cannot exceed 500");
  let candidate = body.source;
  if (hasVersion) {
    const version = t.versions.find((v) => v.versionNumber === Number(body.versionNumber));
    if (!version) return notFound("Formula template version");
    candidate = version.content.source;
  }
  return publicBacktest(backtest(state, t, candidate, limit));
}

/** What approving would change: the saved source against production, biggest moves first. */
export function impact(state, t, limit = 64) {
  const result = backtest(state, t, t.source, limit);
  result.results.sort((a, b) => {
    const af = Boolean(a.currentError || a.candidateError || a.candidateRefusal);
    const bf = Boolean(b.currentError || b.candidateError || b.candidateRefusal);
    if (af !== bf) return af ? 1 : -1;
    return Math.abs(Number(b.delta)) - Math.abs(Number(a.delta));
  });
  return result;
}

function approvalImpact(state, path, body) {
  const t = templateOf(state, path);
  if (!t) return notFound("Formula template");
  const limit = Number(body?.limit) || 64;
  if (limit > 500) return invalid("limit", "Limit cannot exceed 500");
  return publicBacktest(impact(state, t, limit));
}

// --- Readiness -------------------------------------------------------------------------------

/**
 * The review gate, computed ahead of time: a description, a source that checks, the tests
 * (required for a library, a warning for a template without), the review state, an
 * independent reviewer and the submission's age. Shared by templates and libraries.
 */
export function readinessOf(state, record, { kind, context, label }) {
  const now = state.now();
  const checks = [];
  const details = { key: "details", label: "Name and description" };
  if (!record.name.trim()) Object.assign(details, { status: "fail", detail: "Name is missing" });
  else if (!record.description.trim()) Object.assign(details, { status: "fail", detail: "Description is missing" });
  else Object.assign(details, { status: "pass", detail: "Present" });
  checks.push(details);

  const check = checkOf(record.source, kind, context);
  const error = firstError(check);
  const source = { key: "source", label: "Source checks clean" };
  if (!record.source.trim()) Object.assign(source, { status: "fail", detail: "Empty" });
  else if (error) Object.assign(source, { status: "fail", detail: messageText(error.message) });
  else Object.assign(source, { status: "pass", detail: "No errors" });
  checks.push(source);

  const warnings = check.diagnostics.filter((finding) => finding.severity === "warning").length;
  if (warnings > 0) {
    checks.push({ key: "warnings", label: "No warnings", status: "warn", detail: `${warnings} ${warnings === 1 ? "warning" : "warnings"}` });
  }

  const run = check.ok ? testsOf(record.source, context) : { tests: [], error: null };
  const passed = run.tests.filter((test) => test.status === "passed").length;
  const failing = run.tests.filter((test) => test.status !== "passed").map((test) => test.name);
  const tests = { key: "tests", label: "Tests pass" };
  if (!check.ok) Object.assign(tests, { status: "fail", detail: "The tests run once the source checks" });
  else if (run.tests.length === 0) {
    Object.assign(tests, kind === "library" ? { status: "fail", detail: "Add at least one test" } : { status: "warn", detail: "Add a test so a reviewer can see what it bills" });
  } else Object.assign(tests, { status: failing.length > 0 || run.error ? "fail" : "pass", detail: `${passed} of ${run.tests.length} pass` });
  checks.push(tests);

  const status = { key: "status", label: "Review state", status: "pass" };
  if (record.status === "Draft" || record.status === "Inactive") status.detail = "Ready to submit for review";
  else if (record.status === "InReview") status.detail = "In review; awaiting a decision";
  else status.detail = `Active; editing the source returns the ${label} to draft`;
  checks.push(status);

  const reviewer = { key: "reviewer", label: "Independent reviewer" };
  if (record.status !== "InReview") Object.assign(reviewer, { status: "pass", detail: `Applies once the ${label} is in review` });
  else if (record.submittedById === USER.id) Object.assign(reviewer, { status: "fail", detail: `You submitted this ${label}; someone else must approve it` });
  else Object.assign(reviewer, { status: "pass", detail: `You did not submit this ${label}` });
  checks.push(reviewer);

  const age = { key: "submissionAge", label: "Submission age" };
  if (record.status !== "InReview" || !record.submittedAt) Object.assign(age, { status: "pass", detail: `Applies once the ${label} is in review` });
  else {
    const waited = now - record.submittedAt;
    const days = Math.floor(waited / DAY);
    if (waited > STALE_AFTER) Object.assign(age, { status: "fail", detail: `Submitted ${days} days ago; older than the 14-day limit, so it must be resubmitted` });
    else if (waited > AGING_AFTER) Object.assign(age, { status: "warn", detail: `Submitted ${days} days ago; decide soon or it expires at 14` });
    else Object.assign(age, { status: "pass", detail: "Submitted recently" });
  }
  checks.push(age);

  const blocking = [details, source, tests].some((item) => item.status === "fail");
  return {
    canSubmit: !blocking && (record.status === "Draft" || record.status === "Inactive"),
    canApprove: !blocking && record.status === "InReview" && reviewer.status !== "fail" && age.status !== "fail",
    checks,
    testTotal: run.tests.length,
    testPassed: passed,
    testFailing: failing,
    pendingSchedules: [],
  };
}

function readiness(state, t) {
  return readinessOf(state, t, { kind: "tariff", context: tariffContext(state), label: "template" });
}

// --- Templates -------------------------------------------------------------------------------

function listTemplates(state, _path, _body, query) {
  const limit = Math.min(Number(query?.get("limit")) || 20, 100);
  const offset = Math.max(Number(query?.get("offset")) || 0, 0);
  const status = query?.get("status");
  const q = (query?.get("query") ?? query?.get("search") ?? "").toLowerCase();
  const rows = [...studio(state).templates.values()]
    .filter((t) => !status || t.status === status)
    .filter((t) => !q || `${t.name} ${t.description}`.toLowerCase().includes(q))
    .sort((a, b) => b.updatedAt - a.updatedAt || (a.id < b.id ? -1 : 1));
  const results = rows.slice(offset, offset + limit).map((t) => publicTemplate(state, t));
  const next = offset + limit < rows.length ? `${BASE}/?limit=${limit}&offset=${offset + limit}` : null;
  const prev = offset > 0 ? `${BASE}/?limit=${limit}&offset=${Math.max(offset - limit, 0)}` : null;
  return { results, count: rows.length, next, prev };
}

function getTemplate(state, path) {
  const t = templateOf(state, path);
  return t ? publicTemplate(state, t) : notFound("Formula template");
}

/** A new draft template with its first version. */
export function newTemplate(state, content, extra = {}) {
  const now = state.now();
  const t = {
    id: nextId(state, "fmt_studio_new_"),
    ...content,
    status: "Draft",
    submittedById: null,
    submittedAt: null,
    approvedById: null,
    approvedAt: null,
    reviewComment: "",
    version: 1,
    sourceTemplateId: extra.sourceTemplateId ?? null,
    sourceVersionNumber: extra.sourceVersionNumber ?? null,
    currentVersionNumber: 1,
    createdAt: now,
    updatedAt: now,
    versions: [
      { versionNumber: 1, status: "Draft", createdById: USER.id, createdAt: now, changeMessage: extra.changeMessage ?? "Initial version", tags: [], content: { ...content } },
    ],
    usages: [],
    reviews: [],
  };
  studio(state).templates.set(t.id, t);
  return t;
}

function freeName(state, name) {
  const taken = new Set([...studio(state).templates.values()].map((t) => t.name));
  if (!taken.has(name)) return name;
  for (let n = 2; ; n += 1) if (!taken.has(`${name} (${n})`)) return `${name} (${n})`;
}

function createTemplate(state, _path, body) {
  const content = normalizeContent(body ?? {});
  const errors = contentErrors(content);
  if (errors.length > 0) return refusedContent(errors);
  return { __status: 201, body: publicTemplate(state, newTemplate(state, content)) };
}

/** A change to the source or the name takes a new version and returns the template to draft. */
export function applyEdit(state, record, content) {
  const material = record.source !== content.source || record.name !== content.name;
  const now = state.now();
  Object.assign(record, content, { version: record.version + 1, updatedAt: now });
  if (!material) return;
  if (record.status === "Active" || record.status === "InReview") {
    Object.assign(record, { status: "Draft", submittedById: null, submittedAt: null, approvedById: null, approvedAt: null });
  }
  record.currentVersionNumber += 1;
  record.versions.push({
    versionNumber: record.currentVersionNumber,
    status: record.status,
    createdById: USER.id,
    createdAt: now,
    changeMessage: "",
    tags: [],
    content: contentOf(record),
  });
}

export function conflict(what) {
  return problem(409, "resource-conflict", "Resource Conflict", `The ${what} changed since you opened it. Reload and try again.`);
}

function updateTemplate(state, path, body) {
  const t = templateOf(state, path);
  if (!t) return notFound("Formula template");
  if (body?.version !== undefined && Number(body.version) !== t.version) return conflict("template");
  const content = normalizeContent(body ?? {});
  const errors = contentErrors(content);
  if (errors.length > 0) return refusedContent(errors);
  if (body?.status && STATUSES.has(body.status) && body.status !== t.status) {
    return invalid("status", `Status cannot be changed from ${t.status} to ${body.status} here; submit, approve, reject, or archive the template instead`);
  }
  applyEdit(state, t, content);
  return publicTemplate(state, t);
}

function duplicateTemplates(state, _path, body) {
  const ids = Array.isArray(body?.templateIds) ? body.templateIds : [];
  if (ids.length === 0) return invalid("templateIds", "Template Ids are required", "REQUIRED");
  const copies = [];
  for (const id of ids) {
    const t = studio(state).templates.get(id);
    if (!t) return notFound("Formula template");
    copies.push(newTemplate(state, { ...contentOf(t), name: freeName(state, `${t.name} (copy)`) }));
  }
  return copies.map((t) => publicTemplate(state, t));
}

function bulkUpdateStatus(state, _path, body) {
  const ids = Array.isArray(body?.templateIds) ? body.templateIds : [];
  const status = body?.status;
  if (!STATUSES.has(status)) return invalid("status", "Unknown status");
  const templates = ids.map((id) => studio(state).templates.get(id));
  if (templates.some((t) => !t)) return notFound("Formula template");
  if (status === "Inactive" && !body?.confirmInUse) {
    const used = templates.filter((t) => usageCount(t) > 0);
    if (used.length > 0) {
      const message = `Archiving stops these templates rating anything: ${used.map((t) => `${t.name} (${t.usages.map((u) => `${u.count} ${u.type.replaceAll("_", " ")}`).join(", ")})`).join("; ")}. Confirm to archive them anyway`;
      return problem(422, "validation-error", "Validation Failed", message, [{ field: "confirmInUse", code: "INVALID", message }]);
    }
  }
  const now = state.now();
  for (const t of templates) Object.assign(t, { status, version: t.version + 1, updatedAt: now });
  return templates.map((t) => publicTemplate(state, t));
}

function importTemplates(state, _path, body) {
  if (body?.exportVersion !== "2.0") return invalid("exportVersion", "Only version 2.0 exports can be imported");
  const list = Array.isArray(body?.templates) ? body.templates : [];
  if (list.length === 0) return invalid("templates", "The file holds no templates", "REQUIRED");
  const renamed = {};
  const created = [];
  for (const [index, payload] of list.entries()) {
    const content = normalizeContent(payload ?? {});
    const errors = contentErrors(content).map((error) => ({ ...error, field: `templates[${index}].${error.field}` }));
    if (errors.length > 0) return refusedContent(errors);
    const name = freeName(state, content.name);
    if (name !== content.name) {
      if (body.onConflict !== "rename") return business(`A template named ${content.name} already exists`, [{ field: `templates[${index}].name`, code: "INVALID", message: `A template named ${content.name} already exists` }]);
      renamed[content.name] = name;
    }
    created.push(newTemplate(state, { ...content, name }, { changeMessage: "Imported" }));
  }
  return { created: created.map((t) => publicTemplate(state, t)), renamed };
}

function listStandards() {
  return STANDARDS.map((standard) => ({ ...standard }));
}

function installStandards(state) {
  const names = new Set([...studio(state).templates.values()].map((t) => t.name));
  const installed = [];
  const skipped = [];
  for (const standard of STANDARDS) {
    if (names.has(standard.name)) {
      skipped.push(standard.name);
      continue;
    }
    installed.push(newTemplate(state, { ...standard }, { changeMessage: "Installed from the standard templates" }));
  }
  return { installed: installed.map((t) => publicTemplate(state, t)), skipped };
}

// --- Review ----------------------------------------------------------------------------------

/** Active teammates who hold approve, without the person asking. */
export function reviewers() {
  return TEAM.filter((person) => person.id !== USER.id).map((person) => {
    const user = userRecord(person.id);
    return { id: user.id, name: user.name, username: user.username, profilePicUrl: user.profilePicUrl };
  });
}

/** recordReview: a submission opens or continues a round; a decision lands in the open one. */
export function recordReview(record, { idPrefix, key }, decision, comment, now, extra = {}) {
  const latest = record.reviews.reduce((best, r) => (!best || r.createdAt >= best.createdAt ? r : best), null);
  const closed = latest && ["Approved", "Rejected", "Expired"].includes(latest.decision);
  const round = !latest ? 1 : decision === "Submitted" && closed ? latest.round + 1 : latest.round;
  record.reviews.push({
    id: `${idPrefix}${record.id.split("_").slice(1).join("_")}_${record.reviews.length + 1}`,
    [key]: record.id,
    round,
    decision,
    actorId: USER.id,
    comment: comment ?? "",
    baseVersionNumber: decision === "Submitted" ? (approvedVersion(record)?.versionNumber ?? 0) : (latest?.baseVersionNumber ?? 0),
    createdAt: now,
    requestedReviewerId: extra.reviewerId ?? null,
    reviewerNote: extra.reviewerNote ?? "",
  });
}

/**
 * The review flow, shared by templates and libraries: submit is gated on readiness; approve
 * takes a version snapshot; reject archives; request changes returns it to draft.
 */
export function reviewFlow({ find, present, readinessFor, label, review }) {
  const submitOne = (state, path, body) => {
    const record = find(state, path);
    if (!record) return notFound(`Formula ${label}`);
    if (record.status !== "Draft" && record.status !== "Inactive") {
      return business(`Only a draft or inactive ${label} can be submitted; this one is ${record.status}`);
    }
    const note = String(body?.reviewerNote ?? "");
    if ([...note].length > MAX_REVIEWER_NOTE) {
      return problem(422, "validation-error", "Validation Failed", `The note can be at most ${MAX_REVIEWER_NOTE} characters`, [
        { field: "reviewerNote", code: "INVALID", message: `The note can be at most ${MAX_REVIEWER_NOTE} characters` },
      ]);
    }
    const reviewerId = body?.reviewerId ? String(body.reviewerId) : null;
    if (reviewerId && !reviewers().some((r) => r.id === reviewerId)) {
      return problem(422, "validation-error", "Validation Failed", `That person cannot approve formula ${label}s`, [
        { field: "reviewerId", code: "INVALID", message: `That person cannot approve formula ${label}s` },
      ]);
    }
    const ready = readinessFor(state, record);
    const blocking = ready.checks.filter((check) => ["details", "source", "tests"].includes(check.key) && check.status === "fail");
    if (blocking.length > 0) {
      return business(
        `This ${label} is not ready for review`,
        blocking.map((check) => ({ field: check.key, code: "INVALID", message: `${check.label}: ${check.detail}` })),
      );
    }
    const now = state.now();
    Object.assign(record, { status: "InReview", submittedById: USER.id, submittedAt: now, reviewComment: body?.comment ?? "", version: record.version + 1, updatedAt: now });
    recordReview(record, review, "Submitted", body?.comment, now, { reviewerId, reviewerNote: note });
    return present(state, record);
  };

  const approveOne = (state, path, body) => {
    const record = find(state, path);
    if (!record) return notFound(`Formula ${label}`);
    if (record.status !== "InReview") return business(`Only a ${label} in review can be approved; this one is ${record.status}`);
    if (record.submittedById === USER.id) return invalid("approvedById", `A ${label} cannot be approved by its submitter`);
    const now = state.now();
    if (record.submittedAt && now - record.submittedAt > STALE_AFTER) {
      return business("This submission is older than 14 days; the author must resubmit it");
    }
    const ready = readinessFor(state, record);
    if (!ready.canApprove) {
      const failing = ready.checks.filter((check) => check.status === "fail");
      return business(`This ${label} cannot be approved: ${failing.map((check) => `${check.label} (${check.detail})`).join("; ")}`);
    }
    record.currentVersionNumber += 1;
    Object.assign(record, { status: "Active", approvedById: USER.id, approvedAt: now, reviewComment: body?.comment ?? "", version: record.version + 1, updatedAt: now });
    record.versions.push({ versionNumber: record.currentVersionNumber, status: "Active", createdById: USER.id, createdAt: now, changeMessage: "Approved", tags: [], content: contentOf(record) });
    recordReview(record, review, "Approved", body?.comment, now);
    return present(state, record);
  };

  const decide = (decision) => (state, path, body) => {
    const record = find(state, path);
    if (!record) return notFound(`Formula ${label}`);
    if (record.status !== "InReview") return business(`Only a ${label} in review can be decided; this one is ${record.status}`);
    const comment = String(body?.comment ?? "").trim();
    if (!comment) {
      return invalid("comment", decision === "Rejected" ? `A comment is required when rejecting a formula ${label}` : "A comment is required when requesting changes", "REQUIRED");
    }
    if (record.submittedById === USER.id) {
      return invalid("approvedById", `A ${label}'s submitter cannot review it; ask another reviewer or archive it`);
    }
    const now = state.now();
    if (decision === "Rejected") Object.assign(record, { status: "Inactive", submittedById: null, submittedAt: null });
    else record.status = "Draft";
    Object.assign(record, { reviewComment: comment, version: record.version + 1, updatedAt: now });
    recordReview(record, review, decision, comment, now);
    return present(state, record);
  };

  return { submit: submitOne, approve: approveOne, reject: decide("Rejected"), requestChanges: decide("ChangesRequested") };
}

/** What review compares: the approved version's source and pins against the saved ones. */
export function reviewDiffOf(state, record, kind, context) {
  const base = approvedVersion(record);
  const pins = (source) => pinsOf(state, checkOf(source, kind, context));
  const before = base ? { ...base.content, imports: pins(base.content.source) } : {};
  const after = { ...contentOf(record), imports: pins(record.source) };
  const changes = diffContent(before, after, [...CONTENT_KEYS, "imports"]);
  return {
    hasApprovedBase: Boolean(base),
    baseVersion: base?.versionNumber ?? 0,
    currentVersion: record.currentVersionNumber,
    baseSource: base?.content.source ?? "",
    currentSource: record.source,
    baseImports: before.imports ?? [],
    currentImports: after.imports,
    changes,
    changeCount: Object.keys(changes).length,
    pendingSchedules: [],
  };
}

export const reviewDiff = (state, t) => reviewDiffOf(state, t, "tariff", tariffContext(state));

const templateReview = reviewFlow({
  find: templateOf,
  present: publicTemplate,
  readinessFor: readiness,
  label: "template",
  review: { idPrefix: "ftr_", key: "templateId" },
});

function listReviews(state, path) {
  const t = templateOf(state, path);
  if (!t) return notFound("Formula template");
  return [...t.reviews].sort((a, b) => b.createdAt - a.createdAt || b.id.localeCompare(a.id)).map((review) => publicReview(review, "templateId"));
}

// --- Versions --------------------------------------------------------------------------------

function listVersions(state, path, _body, query) {
  const t = templateOf(state, path);
  if (!t) return notFound("Formula template");
  const limit = Number(query?.get("limit")) || 20;
  const offset = Number(query?.get("offset")) || 0;
  const rows = [...t.versions].reverse();
  return {
    results: rows.slice(offset, offset + limit).map((v) => publicVersion(state, t, v)),
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
  return version ? publicVersion(state, t, version) : notFound("Formula template version");
}

function createVersion(state, path, body) {
  const t = templateOf(state, path);
  if (!t) return notFound("Formula template");
  const now = state.now();
  t.currentVersionNumber += 1;
  const version = { versionNumber: t.currentVersionNumber, status: t.status, createdById: USER.id, createdAt: now, changeMessage: body?.changeMessage ?? "", tags: [], content: contentOf(t) };
  t.versions.push(version);
  Object.assign(t, { version: t.version + 1, updatedAt: now });
  return publicVersion(state, t, version);
}

/** Restores a version's content as a new draft version. */
export function rollbackRecord(state, record, body, what) {
  const target = record.versions.find((v) => v.versionNumber === Number(body?.targetVersion));
  if (!target) return notFound(`${what} version`);
  if (target.versionNumber === record.currentVersionNumber) return business("That version is already the current one");
  applyEdit(state, record, { ...target.content });
  const latest = record.versions.at(-1);
  if (latest) latest.changeMessage = body?.changeMessage || `Restored v${target.versionNumber}`;
  return null;
}

function rollback(state, path, body) {
  const t = templateOf(state, path);
  if (!t) return notFound("Formula template");
  return rollbackRecord(state, t, body, "Formula template") ?? publicTemplate(state, t);
}

function fork(state, path, body) {
  const t = templateOf(state, path);
  if (!t) return notFound("Formula template");
  const newName = String(body?.newName ?? "").trim();
  if (!newName) return invalid("newName", "Name is required", "REQUIRED");
  const source = body?.sourceVersion ? t.versions.find((v) => v.versionNumber === Number(body.sourceVersion)) : null;
  if (body?.sourceVersion && !source) return notFound("Formula template version");
  const content = source ? { ...source.content } : contentOf(t);
  const copy = newTemplate(state, { ...content, name: freeName(state, newName) }, {
    sourceTemplateId: t.id,
    sourceVersionNumber: source?.versionNumber ?? t.currentVersionNumber,
    changeMessage: body?.changeMessage || `Forked from ${t.name} v${source?.versionNumber ?? t.currentVersionNumber}`,
  });
  return publicTemplate(state, copy);
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

// --- GraphQL ---------------------------------------------------------------------------------

/** A connection page in the server's shape, newest first. */
export function connection(rows, v, node) {
  const input = v.input ?? {};
  const offset = input.after ? Number(Buffer.from(input.after, "base64url").toString()) : 0;
  const page = rows.slice(offset, offset + (input.first ?? 20));
  const end = offset + page.length;
  return {
    edges: page.map((row) => ({ node: node(row), cursor: Buffer.from(String(offset)).toString("base64url") })),
    ...(v.includeTotalCount === false ? {} : { totalCount: rows.length }),
    pageInfo: {
      hasNextPage: end < rows.length,
      endCursor: page.length ? Buffer.from(String(end)).toString("base64url") : null,
      hasPreviousPage: offset > 0,
      startCursor: null,
    },
  };
}

function formulaTemplateTable(state, v) {
  const rows = [...studio(state).templates.values()].sort((a, b) => b.updatedAt - a.updatedAt);
  return {
    formulaTemplates: connection(rows, v, (t) => ({ __typename: "FormulaTemplate", ...publicTemplate(state, t) })),
  };
}

/** Select options the studio asks for: the template picker and the shipment picker. */
export function formulaSelectRows(state, resource) {
  if (resource === "FORMULA_TEMPLATE") {
    return [...studio(state).templates.values()]
      .sort((a, b) => b.updatedAt - a.updatedAt)
      .map((t) => ({
        id: t.id,
        label: t.name,
        description: `${t.status === "InReview" ? "In review" : t.status} · v${t.currentVersionNumber}`,
        meta: { status: t.status, currentVersionNumber: t.currentVersionNumber },
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

/** A route whose handler answers a module refusal as the server's 422. */
const guarded = (handler) => (state, path, body, query) => {
  try {
    return handler(state, path, body, query);
  } catch (error) {
    if (error instanceof RulesError) return business(`The rules module refused the request: ${error.detail}`);
    throw error;
  }
};

const route = (rest) => new RegExp(`^${BASE}${rest}/?$`);
const one = (rest = "") => route(`/${ID}${rest}`);
const withTemplate = (fn) => (s, p) => {
  const t = templateOf(s, p);
  return t ? fn(s, t) : notFound("Formula template");
};

export const FORMULA_ROUTES = [
  ["GET", route(""), listTemplates],
  ["POST", route(""), createTemplate],
  ["GET", route("/context"), (s) => tariffContext(s)],
  ["GET", route("/standards"), listStandards],
  ["POST", route("/install-standards"), installStandards],
  ["POST", route("/test"), preview],
  ["POST", route("/scaffold-test"), scaffoldTest],
  ["POST", route("/duplicate"), duplicateTemplates],
  ["POST", route("/bulk-update-status"), bulkUpdateStatus],
  ["POST", route("/import"), importTemplates],
  ["GET", one(), getTemplate],
  ["PUT", one(), updateTemplate],
  ["GET", one("/usage"), withTemplate((_s, t) => ({ inUse: t.usages.length > 0, usages: t.usages }))],
  ["GET", one("/versions"), listVersions],
  ["POST", one("/versions"), createVersion],
  ["GET", one("/versions/scheduled"), withTemplate(() => [])],
  ["GET", one("/versions/\\d+"), getVersion],
  ["POST", one("/rollback"), rollback],
  ["POST", one("/fork"), fork],
  ["GET", one("/compare"), compareVersions],
  ["GET", one("/lineage"), lineage],
  ["GET", one("/readiness"), withTemplate(readiness)],
  ["GET", one("/reviews"), listReviews],
  ["GET", one("/review-diff"), withTemplate(reviewDiff)],
  ["GET", one("/reviewers"), withTemplate(() => reviewers())],
  ["POST", one("/backtest"), runBacktest],
  ["POST", one("/impact"), approvalImpact],
  ["POST", one("/submit"), templateReview.submit],
  ["POST", one("/approve"), templateReview.approve],
  ["POST", one("/reject"), templateReview.reject],
  ["POST", one("/request-changes"), templateReview.requestChanges],
].map(([method, pattern, handler]) => [method, pattern, guarded(handler)]);
