// The Lading studio's REST routes under /edi/lading/, and the partner read the studio's header
// uses, as edihandler serves them (docs/engineering/lading.md, "Versions, replay and shadow").
// Checks, tests, runs, replays and the shadow's comparisons are the EDI module's answers; a
// replay or a shadow reads the fixture's month of Acme tenders, its lookups answered from the
// organisation's mappings in the fixture.
import { edi, EdiError } from "../edi.mjs";
import {
  checkOf,
  createLadingState,
  digestOf,
  LADING_CONTEXT,
  LADING_LOOKUPS,
  LADING_PARTNERS,
} from "../fixtures/lading.mjs";
import { business, invalid, notFound } from "./formula.mjs";

const BASE = "/api/v1/edi/lading";
const ID = "[A-Za-z0-9_]+";
const DAY = 86_400;
const MAX_REPLAY_INPUTS = 500;
const MAX_REPLAY_ITEMS_KEPT = 500;
const MAX_REPLAY_DAYS = 90;
const MAX_SHADOW_DAYS = 30;
const MAX_NOTES = 2_000;

/** The scenario's projects, built on first use so every scenario starts fresh. */
function lading(state) {
  if (!state.lading) {
    state.lading = createLadingState(state.anchor, state.scenario.lading);
    seedReplays(state);
  }
  return state.lading;
}

const idAt = (path, pattern) => new RegExp(pattern).exec(path)?.[1];
const versionOf = (state, path) =>
  lading(state).versions.get(idAt(path, `/versions/(${ID})`)) ?? null;
const projectOf = (state, path) =>
  lading(state).projects.get(idAt(path, `/projects/(${ID})`)) ?? null;
const projectOfVersion = (state, version) => lading(state).projects.get(version.projectId);

function nextId(state, prefix) {
  const store = lading(state);
  store.nextId += 1;
  return `${prefix}${store.nextId}`;
}

function samplesOf(state, version) {
  const store = lading(state);
  return version.samples.map((ref) => ({ path: ref.path, content: store.samples.get(ref.digest) }));
}

function programOf(state, version) {
  return { files: version.files, samples: samplesOf(state, version) };
}

const editable = (version) => version.status === "Draft";

function readinessOf(report, now) {
  const { errors, failing, tests, uncoveredFlowClauses, ready } = report.readiness;
  return { errors, failing, tests, uncoveredFlowClauses, ready, checkedAt: now };
}

function touch(state, record) {
  record.version += 1;
  record.updatedAt = state.now();
  return record;
}

/** Every lookup the organisation has mapped, so a run or a replay never waits on one. */
const ALL_LOOKUPS = Object.entries(LADING_LOOKUPS).flatMap(([kind, codes]) =>
  Object.entries(codes).map(([code, found]) => ({ kind, code, found })),
);

/** A run's lookups answered as the TMS answers them: a code with no mapping stays unanswered. */
function runWithLookups(state, version, project, map, input) {
  const request = {
    project: programOf(state, version),
    context: LADING_CONTEXT,
    map,
    input,
    settings: project.settings,
    rules: true,
  };
  let result = edi("lading.run", request);
  if (result.needs.length > 0) {
    const lookups = result.needs.map((need) => {
      const found = LADING_LOOKUPS[need.kind]?.[need.code];
      return found ? { ...need, found } : need;
    });
    result = edi("lading.run", { ...request, lookups });
  }
  return result;
}

/** One replay of `version` against `against` over the messages a map reads in the period. */
function replay(state, { version, against, map, from, to, createdAt }) {
  const store = lading(state);
  const project = projectOfVersion(state, version);
  const set = version.manifest.maps.find((item) => item.name === map)?.set;
  const messages = store.traffic.filter(
    (message) => message.set === set && message.generatedAt >= from && message.generatedAt < to,
  );
  const summary = {
    total: 0,
    same: 0,
    differ: 0,
    newlyRejected: 0,
    newlyHeld: 0,
    newlyDeclined: 0,
    failed: 0,
    fields: {},
  };
  const items = [];
  for (let start = 0; start < messages.length; start += MAX_REPLAY_INPUTS) {
    const batch = messages.slice(start, start + MAX_REPLAY_INPUTS);
    const report = edi("lading.replay", {
      before: programOf(state, against),
      after: programOf(state, version),
      context: LADING_CONTEXT,
      map,
      settings: project.settings,
      inputs: batch.map((message) => ({
        id: message.id,
        input: { x12: message.x12 },
        lookups: ALL_LOOKUPS,
        now: new Date(message.generatedAt * 1000).toISOString(),
      })),
    });
    for (const [key, value] of Object.entries(report.summary)) {
      if (key === "fields") {
        for (const [field, count] of Object.entries(value ?? {})) {
          summary.fields[field] = (summary.fields[field] ?? 0) + count;
        }
      } else {
        summary[key] += value;
      }
    }
    for (const item of report.items) {
      if (item.same || items.length >= MAX_REPLAY_ITEMS_KEPT) continue;
      items.push({
        messageId: item.id,
        same: false,
        differences: item.differences,
        newOutcomes: item.newOutcomes,
        goneOutcomes: item.goneOutcomes,
        error: item.afterError?.message ?? item.beforeError?.message ?? null,
      });
    }
  }
  return {
    id: nextId(state, "ldr_acme_"),
    versionId: version.id,
    againstVersionId: against.id,
    map,
    periodStart: from,
    periodEnd: to,
    status: "Completed",
    summary,
    items,
    error: null,
    finishedAt: createdAt + 40,
    createdAt,
  };
}

/** What the shadow read beside the active version: every message since it started, compared. */
function shadowResultsOf(state, version) {
  const store = lading(state);
  const cached = store.shadowResults.get(version.id);
  if (cached) return cached;
  const project = projectOfVersion(state, version);
  const active = store.versions.get(project.activeVersionId);
  const since = version.certifiedAt ?? version.updatedAt;
  const messages = store.traffic.filter((message) => message.generatedAt >= since);
  const report = edi("lading.replay", {
    before: programOf(state, active),
    after: programOf(state, version),
    context: LADING_CONTEXT,
    map: "tenderIn",
    settings: project.settings,
    inputs: messages.map((message) => ({
      id: message.id,
      input: { x12: message.x12 },
      lookups: ALL_LOOKUPS,
      now: new Date(message.generatedAt * 1000).toISOString(),
    })),
  });
  const byId = new Map(messages.map((message) => [message.id, message]));
  const results = report.items
    .map((item, index) => ({
      id: `lsr_acme_${index + 1}`,
      messageId: item.id,
      transactionSet: "204",
      same: item.same,
      differences: item.differences,
      newOutcomes: item.newOutcomes,
      goneOutcomes: item.goneOutcomes,
      error: item.afterError?.message ?? null,
      createdAt: byId.get(item.id).generatedAt + 2,
    }))
    .sort((a, b) => b.createdAt - a.createdAt);
  const shadow = {
    summary: {
      total: report.summary.total,
      same: report.summary.same,
      failed: report.summary.failed,
    },
    results,
  };
  store.shadowResults.set(version.id, shadow);
  return shadow;
}

/** Version 3's replay of tenderIn against version 2 over the last 30 days, run after its last save. */
function seedReplays(state) {
  const store = state.lading;
  const draft = store.versions.get("ldv_acme_3");
  const active = store.versions.get("ldv_acme_2");
  const createdAt = draft.updatedAt + 1_800;
  const seeded = replay(state, {
    version: draft,
    against: active,
    map: "tenderIn",
    from: createdAt - 30 * DAY,
    to: createdAt,
    createdAt,
  });
  store.replays.set(seeded.id, seeded);
}

const replaysOf = (state, versionId) =>
  [...lading(state).replays.values()]
    .filter((item) => item.versionId === versionId)
    .sort((a, b) => b.createdAt - a.createdAt);

/** Every inbound X12 map has a clean replay since the version's last save, as Certify asks. */
function replayGaps(state, version) {
  const latest = new Map();
  for (const item of replaysOf(state, version.id)) {
    if (item.status !== "Completed" || item.createdAt < version.updatedAt) continue;
    if (!latest.has(item.map)) latest.set(item.map, item);
  }
  return version.manifest.maps
    .filter((map) => map.direction === "inbound" && map.source === "x12")
    .map((map) => map.name)
    .sort()
    .flatMap((name) => {
      const item = latest.get(name);
      if (!item) {
        return [
          {
            field: "replay",
            code: "REQUIRED",
            message: `Replay ${name} against the active version before certifying`,
          },
        ];
      }
      if ((item.summary?.newlyRejected ?? 0) > 0) {
        return [
          {
            field: "replay",
            code: "INVALID",
            message: `The replay of ${name} rejects ${item.summary.newlyRejected} messages the active version accepts`,
          },
        ];
      }
      return [];
    });
}

function notReadyErrors(readiness) {
  const errors = [];
  if (readiness.errors > 0)
    errors.push({
      field: "files",
      code: "INVALID",
      message: `${readiness.errors} errors remain in the files`,
    });
  if (readiness.failing > 0)
    errors.push({ field: "tests", code: "INVALID", message: `${readiness.failing} tests fail` });
  if (readiness.tests === 0)
    errors.push({ field: "tests", code: "REQUIRED", message: "The project has no tests" });
  if (readiness.uncoveredFlowClauses > 0) {
    errors.push({
      field: "tests",
      code: "INVALID",
      message: `${readiness.uncoveredFlowClauses} flow clauses are not reached by any test`,
    });
  }
  return errors;
}

const validation = (errors) => ({
  __status: 400,
  body: {
    type: "validation-error",
    title: "Validation Failed",
    status: 400,
    detail: errors[0]?.message,
    errors,
  },
});

function versionsOf(state, projectId) {
  return [...lading(state).versions.values()]
    .filter((version) => version.projectId === projectId)
    .sort((a, b) => b.number - a.number);
}

function newVersion(state, project, from, { status, files, samples, notes }) {
  const manifest = checkOf(files, samplesOf(state, { samples }));
  const now = state.now();
  const version = {
    id: nextId(state, "ldv_acme_new_"),
    projectId: project.id,
    number: project.nextVersionNumber,
    status,
    parentVersionId: from?.id ?? null,
    files,
    samples,
    manifest,
    readiness: null,
    notes,
    errorCount: manifest.errors,
    warningCount: manifest.warnings,
    shadowUntil: null,
    certifiedAt: null,
    activatedAt: null,
    supersededAt: null,
    archivedAt: null,
    version: 1,
    createdAt: now,
    updatedAt: now,
  };
  project.nextVersionNumber += 1;
  lading(state).versions.set(version.id, version);
  return version;
}

function storeSamples(state, samples) {
  const store = lading(state);
  return samples.map((sample) => {
    const digest = digestOf(sample.content);
    store.samples.set(digest, sample.content);
    return { path: sample.path, digest, size: sample.content.length };
  });
}

function createProject(state, _path, body) {
  const partner = LADING_PARTNERS.find((item) => item.id === body?.partnerId);
  if (!partner) return notFound("EDI partner");
  if (![...lading(state).projects.values()].some((item) => item.ediPartnerId === partner.id)) {
    if (!["carrier", "broker"].includes(body.role))
      return invalid("role", "Pick carrier or broker");
    if (!Array.isArray(body.sets) || body.sets.length === 0) {
      return invalid("sets", "Pick at least one transaction set");
    }
    const starter = edi("lading.starter", {
      name: partner.code.toLowerCase(),
      label: partner.name,
      role: body.role,
      them: body.them,
      us: body.us,
      zone: partner.timezone,
      group: body.group,
      usage: body.usage,
      acknowledge: body.acknowledge,
      ackWithin: body.ackWithin,
      sets: body.sets,
    });
    const now = state.now();
    const project = {
      id: nextId(state, "ldp_"),
      ediPartnerId: partner.id,
      activeVersionId: null,
      shadowVersionId: null,
      settings: {},
      nextVersionNumber: 1,
      version: 1,
      createdAt: now,
      updatedAt: now,
      partner: { id: partner.id, code: partner.code, name: partner.name },
    };
    lading(state).projects.set(project.id, project);
    newVersion(state, project, null, {
      status: "Draft",
      files: starter.files,
      samples: storeSamples(state, starter.samples),
      notes: `Started from Trenova's ${body.role} starter`,
    });
    return { __status: 201, body: project };
  }
  return business(`${partner.name} already has a Lading project`);
}

function saveDraft(state, path, body) {
  const version = versionOf(state, path);
  if (!version) return notFound("Lading version");
  if (!editable(version)) {
    return business(`Version ${version.number} is ${version.status}; only a draft can change`);
  }
  if (!Array.isArray(body?.files) || body.files.length === 0) {
    return invalid("files", "A project needs at least its partner file", "REQUIRED");
  }
  if ((body.notes ?? "").length > MAX_NOTES) {
    return invalid("notes", "Notes cannot be longer than 2,000 characters");
  }
  if (Number(body.version) !== version.version) {
    return {
      __status: 409,
      body: {
        type: "resource-conflict",
        title: "Resource Conflict",
        status: 409,
        detail: "The draft changed since you opened it. Reload and try again.",
      },
    };
  }
  const manifest = checkOf(body.files, body.samples ?? []);
  Object.assign(version, {
    files: body.files,
    samples: storeSamples(state, body.samples ?? []),
    manifest,
    errorCount: manifest.errors,
    warningCount: manifest.warnings,
    readiness: null,
    notes: body.notes ?? "",
  });
  return touch(state, version);
}

function test(state, path) {
  const version = versionOf(state, path);
  if (!version) return notFound("Lading version");
  const report = edi("lading.test", {
    project: programOf(state, version),
    context: LADING_CONTEXT,
  });
  if (editable(version)) {
    version.readiness = readinessOf(report, state.now());
    version.version += 1;
  }
  return report;
}

function certify(state, path) {
  const version = versionOf(state, path);
  if (!version) return notFound("Lading version");
  if (!editable(version)) {
    return business(`Version ${version.number} is ${version.status}; only a draft is certified`);
  }
  const report = edi("lading.test", {
    project: programOf(state, version),
    context: LADING_CONTEXT,
  });
  version.readiness = readinessOf(report, state.now());
  if (!version.readiness.ready) return validation(notReadyErrors(version.readiness));
  const project = projectOfVersion(state, version);
  if (project.activeVersionId) {
    const gaps = replayGaps(state, version);
    if (gaps.length > 0) return validation(gaps);
  }
  version.status = "Certified";
  version.certifiedAt = state.now();
  version.version += 1;
  return version;
}

function activate(state, path) {
  const version = versionOf(state, path);
  if (!version) return notFound("Lading version");
  if (version.status !== "Certified" && version.status !== "Shadow") {
    return business(
      `Version ${version.number} is ${version.status}; only a certified version goes live`,
    );
  }
  const project = projectOfVersion(state, version);
  const previous = lading(state).versions.get(project.activeVersionId);
  const now = state.now();
  if (previous) {
    previous.status = "Superseded";
    previous.supersededAt = now;
    previous.version += 1;
  }
  version.status = "Active";
  version.activatedAt = now;
  version.version += 1;
  project.activeVersionId = version.id;
  if (project.shadowVersionId === version.id) project.shadowVersionId = null;
  touch(state, project);
  return version;
}

function archive(state, path) {
  const version = versionOf(state, path);
  if (!version) return notFound("Lading version");
  if (version.status === "Active" || version.status === "Shadow") {
    return business(`Version ${version.number} is ${version.status}; stop or replace it first`);
  }
  if (version.status !== "Archived") {
    version.status = "Archived";
    version.archivedAt = state.now();
    version.version += 1;
  }
  return version;
}

function stopShadowOf(state, version) {
  const project = projectOfVersion(state, version);
  if (version.status === "Shadow") {
    version.status = "Certified";
    version.shadowUntil = null;
    version.version += 1;
  }
  if (project.shadowVersionId === version.id) {
    project.shadowVersionId = null;
    touch(state, project);
  }
  lading(state).shadowResults.delete(version.id);
}

function startShadow(state, path, body) {
  const days = Number(body?.days);
  if (!Number.isInteger(days) || days < 1 || days > MAX_SHADOW_DAYS) {
    return invalid("days", "A shadow runs for 1 to 30 days");
  }
  const version = versionOf(state, path);
  if (!version) return notFound("Lading version");
  if (version.status !== "Certified") {
    return business(
      `Version ${version.number} is ${version.status}; only a certified version shadows`,
    );
  }
  const project = projectOfVersion(state, version);
  if (!project.activeVersionId) {
    return business("The project has no active version to shadow; activate this one instead");
  }
  const running = lading(state).versions.get(project.shadowVersionId);
  if (running && running.id !== version.id) stopShadowOf(state, running);
  version.status = "Shadow";
  version.shadowUntil = state.now() + days * DAY;
  version.version += 1;
  project.shadowVersionId = version.id;
  touch(state, project);
  return version;
}

function startReplay(state, path, body) {
  const from = Number(body?.from);
  const to = Number(body?.to);
  if (!(from > 0) || !(to > from)) return invalid("to", "The period ends after it starts");
  if (to - from > MAX_REPLAY_DAYS * DAY) return invalid("from", "A replay covers at most 90 days");
  const version = versionOf(state, path);
  if (!version) return notFound("Lading version");
  if (!version.manifest.maps.some((map) => map.name === body.map)) {
    return invalid("map", `The version has no map ${body.map}`);
  }
  if (version.errorCount > 0) return business("Fix the version's errors before replaying it");
  const project = projectOfVersion(state, version);
  const againstId = body.againstVersionId ?? project.activeVersionId;
  const against = lading(state).versions.get(againstId);
  if (!against) return business("The project has no active version to replay against");
  const created = replay(state, {
    version,
    against,
    map: body.map,
    from,
    to,
    createdAt: state.now(),
  });
  lading(state).replays.set(created.id, created);
  return { __status: 201, body: created };
}

function run(state, path, body) {
  const version = versionOf(state, path);
  if (!version) return notFound("Lading version");
  if (!body?.map) return invalid("map", "Pick a map to run", "REQUIRED");
  const input = body.input ?? {};
  if (Object.values(input).filter((value) => value !== undefined && value !== null).length !== 1) {
    return invalid("input", "Give the run exactly one input");
  }
  return runWithLookups(state, version, projectOfVersion(state, version), body.map, input);
}

const guard = (handler) => (state, path, body, query) => {
  try {
    return handler(state, path, body, query);
  } catch (error) {
    if (error instanceof EdiError) return invalid("project", error.detail);
    throw error;
  }
};

export const LADING_ROUTES = [
  [
    "GET",
    new RegExp(`^/api/v1/edi/partners/(${ID})/?$`),
    (_state, path) =>
      LADING_PARTNERS.find((partner) => path.includes(`/partners/${partner.id}`)) ??
      notFound("EDI partner"),
  ],
  ["GET", new RegExp(`^${BASE}/context/?$`), () => LADING_CONTEXT],
  ["GET", new RegExp(`^${BASE}/catalog/?$`), () => edi("lading.catalog", {})],
  ["POST", new RegExp(`^${BASE}/projects/?$`), guard(createProject)],
  [
    "GET",
    new RegExp(`^${BASE}/projects/by-partner/(${ID})/?$`),
    (state, path) => {
      const partnerId = idAt(path, `/by-partner/(${ID})`);
      return (
        [...lading(state).projects.values()].find((item) => item.ediPartnerId === partnerId) ??
        notFound("Lading project")
      );
    },
  ],
  [
    "GET",
    new RegExp(`^${BASE}/projects/(${ID})/versions/?$`),
    (state, path) => {
      const project = projectOf(state, path);
      return project ? { items: versionsOf(state, project.id) } : notFound("Lading project");
    },
  ],
  [
    "POST",
    new RegExp(`^${BASE}/projects/(${ID})/drafts/?$`),
    guard((state, path, body) => {
      const project = projectOf(state, path);
      if (!project) return notFound("Lading project");
      const from = lading(state).versions.get(body?.fromVersionId);
      if (!from) return notFound("Lading version");
      if (from.projectId !== project.id) {
        return invalid("fromVersionId", "The version belongs to another project");
      }
      const created = newVersion(state, project, from, {
        status: "Draft",
        files: from.files,
        samples: from.samples,
        notes: `Started from version ${from.number}`,
      });
      touch(state, project);
      return { __status: 201, body: created };
    }),
  ],
  [
    "PUT",
    new RegExp(`^${BASE}/projects/(${ID})/settings/?$`),
    (state, path, body) => {
      const project = projectOf(state, path);
      if (!project) return notFound("Lading project");
      if (Number(body?.version) !== project.version) {
        return business("The settings changed since you opened them. Reload and try again.");
      }
      const active = lading(state).versions.get(project.activeVersionId);
      const declared = new Set((active?.manifest.partner?.settings ?? []).map((item) => item.name));
      const unknown = Object.keys(body.settings ?? {}).find((name) => !declared.has(name));
      if (unknown)
        return invalid(`settings.${unknown}`, `The partner file declares no setting ${unknown}`);
      project.settings = Object.fromEntries(
        Object.entries(body.settings ?? {}).filter(([, value]) => value !== null),
      );
      return touch(state, project);
    },
  ],
  [
    "GET",
    new RegExp(`^${BASE}/projects/(${ID})/?$`),
    (state, path) => projectOf(state, path) ?? notFound("Lading project"),
  ],
  ["POST", new RegExp(`^${BASE}/versions/(${ID})/run/?$`), guard(run)],
  ["POST", new RegExp(`^${BASE}/versions/(${ID})/test/?$`), guard(test)],
  ["POST", new RegExp(`^${BASE}/versions/(${ID})/certify/?$`), guard(certify)],
  ["POST", new RegExp(`^${BASE}/versions/(${ID})/activate/?$`), activate],
  ["POST", new RegExp(`^${BASE}/versions/(${ID})/archive/?$`), archive],
  ["POST", new RegExp(`^${BASE}/versions/(${ID})/shadow/?$`), startShadow],
  [
    "DELETE",
    new RegExp(`^${BASE}/versions/(${ID})/shadow/?$`),
    (state, path) => {
      const version = versionOf(state, path);
      if (!version) return notFound("Lading version");
      stopShadowOf(state, version);
      return version;
    },
  ],
  [
    "GET",
    new RegExp(`^${BASE}/versions/(${ID})/shadow-results/?$`),
    guard((state, path, _body, query) => {
      const version = versionOf(state, path);
      if (!version) return notFound("Lading version");
      if (version.status !== "Shadow" && version.shadowUntil == null) {
        return { summary: { total: 0, same: 0, failed: 0 }, results: [] };
      }
      const shadow = shadowResultsOf(state, version);
      const onlyDiffering = query?.get?.("onlyDiffering") === "true";
      return {
        summary: shadow.summary,
        results: onlyDiffering ? shadow.results.filter((item) => !item.same) : shadow.results,
      };
    }),
  ],
  ["POST", new RegExp(`^${BASE}/versions/(${ID})/replays/?$`), guard(startReplay)],
  [
    "GET",
    new RegExp(`^${BASE}/versions/(${ID})/replays/?$`),
    (state, path) => {
      const version = versionOf(state, path);
      return version ? { items: replaysOf(state, version.id) } : notFound("Lading version");
    },
  ],
  [
    "GET",
    new RegExp(`^${BASE}/versions/(${ID})/?$`),
    (state, path) => {
      const version = versionOf(state, path);
      return version ? { version, samples: samplesOf(state, version) } : notFound("Lading version");
    },
  ],
  ["PUT", new RegExp(`^${BASE}/versions/(${ID})/?$`), guard(saveDraft)],
  [
    "GET",
    new RegExp(`^${BASE}/replays/(${ID})/?$`),
    (state, path) =>
      lading(state).replays.get(idAt(path, `/replays/(${ID})`)) ?? notFound("Lading replay"),
  ],
];
