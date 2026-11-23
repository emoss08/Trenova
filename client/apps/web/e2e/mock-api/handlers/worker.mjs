// The worker record page (/hr/workers/<id>), the new worker form (/hr/workers/new) and the roster
// (/hr/workers): every GraphQL operation they send and the REST routes they call, from the files
// in fixtures/worker/. Writes change the scenario's state the way the server would, so a second
// read shows them. The variants (who reads, what the plan includes, whether telematics is
// connected) are scenario fields: `workerAccess`, `workerPlan`, `workerHos` (see README).
import { publishInvalidation } from "../realtime.mjs";
import { connection } from "./aicontrol.mjs";
import { TriageError } from "./triage-model.mjs";
import { permissionManifest } from "../fixtures/session.mjs";
import {
  BENEFIT_PLANS,
  CHECKLIST_TEMPLATES,
  COURSE_LIST,
  COURSES,
  CREDENTIAL_TYPE_LIST,
  CREDENTIAL_TYPES,
  DAY,
  FLEET_CODES,
  PAY_PROFILES,
  PEOPLE,
  POSITIONS,
  PTO_POLICIES,
  REVIEW_TEMPLATES,
  SHIFT_TEMPLATES,
  STATES,
  TENANT,
  WORKER_CUSTOM_FIELDS,
  daysUntil,
  pid,
  requiredCourses,
  userRef,
} from "../fixtures/worker/base.mjs";
import { createWorkerState, MARIA_ID } from "../fixtures/worker/index.mjs";
import { workerDocument } from "../fixtures/worker/maria.mjs";
import {
  credential as credentialRecord,
  employmentEvent,
  nextId,
  ptoRequest,
  trainingRecord,
  worker as workerRecord,
} from "../fixtures/worker/records.mjs";
import {
  NOW,
  buildClearance,
  buildStanding,
  checklistOut,
  courseOf,
  credentialOut,
  credentialSummary,
  credentialTypeOf,
  currentPtoPolicy,
  disciplinaryLadder,
  disciplinaryOut,
  dqfFile,
  drugAlcoholStanding,
  leaveEntitlement,
  ledgerEntries,
  openChecklist,
  ptoBalances,
  safetyEventOut,
  safetyScorecard,
  trainingOut,
  trainingSummary,
  worstTrainingHealth,
} from "./worker-derive.mjs";

// ---- State and variants -------------------------------------------------------------------

function ws(state) {
  state.workerPage ??= createWorkerState();
  return state.workerPage;
}

/**
 * The variant the scenario asks for. `workerAccess`: `full` | `dispatcher` (no Personal field
 * group, no credential read). `workerPlan`: `full` | `core` (off workforce.compliance and
 * workforce.time_off). `workerHos`: `reporting` | `notConnected`.
 */
export function workerVariant(state) {
  return {
    access: state.scenario.workerAccess ?? process.env.MOCK_WORKER_ACCESS ?? "full",
    plan: state.scenario.workerPlan ?? process.env.MOCK_WORKER_PLAN ?? "full",
    hos: state.scenario.workerHos ?? process.env.MOCK_WORKER_HOS ?? "reporting",
  };
}

const DISPATCHER_WITHHELD = new Set(["protected_personal", "worker_credential"]);
const COMPLIANCE_RESOURCES = new Set(["worker_credential", "qualification", "worker_dot_test"]);
const TIME_OFF_RESOURCES = new Set(["worker_pto", "worker_leave"]);

const UNAVAILABLE = {
  compliance: {
    feature: "workforce.compliance",
    featureName: "Driver compliance",
    addOn: "workforce",
    addOnName: "Workforce",
  },
  timeOff: {
    feature: "workforce.time_off",
    featureName: "Time off",
    addOn: "workforce",
    addOnName: "Workforce",
  },
};

/** Whether the variant's reader, on the variant's plan, can read a kind. */
function canRead(state, resource) {
  const variant = workerVariant(state);
  if (variant.access === "dispatcher" && DISPATCHER_WITHHELD.has(resource)) return false;
  if (
    variant.plan === "core" &&
    (COMPLIANCE_RESOURCES.has(resource) || TIME_OFF_RESOURCES.has(resource))
  )
    return false;
  return true;
}

/** A refusal presented the way the server's error presenter returns one (see graphql.mjs). */
function refuse(message) {
  const notFound = message.endsWith(" not found");
  const denied = message.startsWith("You do not have permission");
  throw new TriageError(message, {
    code: notFound ? "NOT_FOUND" : denied ? "FORBIDDEN" : "INVALID",
    type: notFound ? "not-found" : denied ? "forbidden" : "validation-error",
  });
}

function requireRead(state, resource) {
  if (!canRead(state, resource)) refuse(`You do not have permission to read ${resource}.`);
}

function isWorkerId(state, id) {
  return typeof id === "string" && ws(state).files.has(id);
}

function fileOf(state, id) {
  const file = ws(state).files.get(id);
  if (!file) refuse("Worker not found");
  return file;
}

/** The file holding a sub-record with this id, in one of its lists. */
function fileHolding(state, list, id) {
  for (const file of ws(state).files.values()) {
    const rows = list(file);
    const row = rows.find((entry) => entry.id === id);
    if (row) return { file, row, rows };
  }
  refuse("Record not found");
}

function bump(row, patch) {
  Object.assign(row, patch, { version: (row.version ?? 1) + 1, updatedAt: NOW });
  return row;
}

function touch(file, resource = "worker", recordId = null) {
  file.worker.updatedAt = NOW;
  publishInvalidation({ resource, recordId: recordId ?? file.worker.id });
}

/** An audit line on the worker's file, as the server writes it with the worker as subject. */
function audit(file, spec) {
  file.activity.unshift({
    id: nextId("aud"),
    timestamp: NOW,
    resource: spec.resource,
    resourceId: spec.resourceId ?? file.worker.id,
    operation: spec.operation,
    entityRef: spec.entityRef ?? null,
    comment: spec.comment ?? null,
    channel: "web",
    principalType: "user",
    principalId: PEOPLE.dana.id,
    actorName: PEOPLE.dana.name,
    changes: spec.changes ?? null,
    subjectResource: "worker",
    subjectId: file.worker.id,
    user: userRef(PEOPLE.dana),
  });
}

const DANA = () => userRef(PEOPLE.dana);

// ---- Records out --------------------------------------------------------------------------

/** The worker as WorkerRecord and the roster select it, with the server-kept roll-ups filled in. */
function workerOut(state, file) {
  const wrk = file.worker;
  const credentials = credentialSummary(file);
  const training = trainingSummary(file);
  const safety = safetyScorecard(file);
  const standing = drugAlcoholStanding(file);
  const nextCredential =
    credentials.items
      .map((item) => item.credential?.expiresAt)
      .filter((when) => when && when > NOW)
      .sort((a, b) => a - b)[0] ?? null;
  const nextTraining =
    training.items
      .map((item) => (item.record ? (item.record.dueAt ?? item.record.expiresAt) : null))
      .filter((when) => when && when > NOW)
      .sort((a, b) => a - b)[0] ?? null;
  const personal = canRead(state, "protected_personal");
  return {
    ...wrk,
    profile: {
      ...wrk.profile,
      dob: personal ? wrk.profile.dob : null,
      licenseNumber: personal ? wrk.profile.licenseNumber : null,
      twicCardNumber: personal ? wrk.profile.twicCardNumber : null,
      complianceStatus: credentials.complianceStatus,
      trainingHealth: worstTrainingHealth(training),
      safetyRating: safety.rating,
      safetyScore: safety.score,
      nextCredentialExpiry: nextCredential,
      nextTrainingDue: nextTraining,
      drugAlcoholStatus: standing.status,
      returnToDutyStatus: standing.returnToDuty,
      lastClearinghouseQueryAt: standing.lastClearinghouseQueryAt,
      nextClearinghouseQueryDue: standing.nextClearinghouseQueryDue,
    },
  };
}

function reviewDueAt(file) {
  const open = file.reviews.some(
    (review) => review.status === "Draft" || review.status === "Submitted",
  );
  if (open) return null;
  const last = file.reviews
    .filter((review) => review.status === "Closed" || review.status === "Acknowledged")
    .sort((a, b) => b.periodEnd - a.periodEnd)[0];
  return last?.nextReviewAt ?? null;
}

function hosLink(state, file) {
  return workerVariant(state).hos === "notConnected" ? "NotConnected" : file.hos.link;
}

/** `workeroverviewservice.Get`: one read for the header, tab marks, strips and file grid. */
function overviewOut(state, file) {
  const wrk = file.worker;
  const variant = workerVariant(state);
  const credentials = canRead(state, "worker_credential") ? credentialSummary(file) : null;
  const training = trainingSummary(file);
  const safety = safetyScorecard(file);
  const drugAlcohol = canRead(state, "worker_dot_test") ? drugAlcoholStanding(file) : null;
  const checklist = openChecklist(file);
  const dqf = canRead(state, "qualification") ? dqfFile(file) : null;
  const dueAt = reviewDueAt(file);
  const { standing, concerns } = buildStanding({
    worker: wrk,
    credentials,
    training,
    safety,
    safetyEvents: file.safety.events,
    checklist,
    reviewDueAt: dueAt,
    drugAlcohol,
    verifications: dqf ? file.dqf.verifications : [],
    safetyHistoryDueAt: file.dqf.safetyHistoryDueAt,
  });
  const clearance = buildClearance({
    worker: wrk,
    credentials,
    training,
    drugAlcohol,
    reviewDueAt: dueAt,
  });
  const reviews = [...file.reviews].sort((a, b) => b.periodEnd - a.periodEnd);
  const openReview =
    reviews.find((review) => review.status === "Draft" || review.status === "Submitted") ?? null;
  const lastReview =
    reviews.find((review) => review.status === "Closed" || review.status === "Acknowledged") ??
    null;
  const shift = file.schedule.assignments.find((row) => !row.effectiveTo || row.effectiveTo > NOW);
  const link = hosLink(state, file);
  const reporting = link === "Reporting";
  const payAssignment = effectivePayAssignment(file);
  const profile = payAssignment
    ? Object.values(PAY_PROFILES).find((p) => p.id === payAssignment.payProfileId)
    : null;
  const headline = profile ? payHeadline(profile, payAssignment) : null;
  const enrollments = file.benefits.enrollments;
  const pendingChanges = file.portal.profileChanges.filter(
    (row) => row.status === "Pending",
  ).length;
  const nextTimeOff = canRead(state, "worker_pto")
    ? ([...file.pto.requests]
        .filter(
          (row) =>
            (row.status === "Requested" || row.status === "Approved") && row.endDate >= NOW - DAY,
        )
        .sort((a, b) => a.startDate - b.startDate)[0] ?? null)
    : null;
  const hired = file.employmentEvents
    .filter((event) => event.kind === "Hired" || event.kind === "Rehired")
    .sort((a, b) => b.effectiveAt - a.effectiveAt)[0];
  return {
    worker: workerOut(state, file),
    asOf: NOW,
    standing,
    concerns: concerns.slice(0, 30),
    concernCount: concerns.length,
    clearance,
    credentials,
    qualificationFile: dqf,
    drugAlcohol: drugAlcohol
      ? {
          standing: drugAlcohol,
          nextQueryDaysUntil: drugAlcohol.nextClearinghouseQueryDue
            ? daysUntil(drugAlcohol.nextClearinghouseQueryDue, NOW)
            : null,
        }
      : null,
    training,
    safety,
    employment: {
      eventCount: file.employmentEvents.length,
      hiredAt: hired?.effectiveAt ?? null,
      openChecklistCount: file.checklists.filter((row) => row.status === "Open").length,
      closedChecklistCount: file.checklists.filter((row) => row.status !== "Open").length,
    },
    checklist,
    pto: canRead(state, "worker_pto") ? ptoBalances(file) : null,
    leave: canRead(state, "worker_leave") ? leaveEntitlement(file) : null,
    openReview,
    lastReview,
    nextReviewAt: dueAt,
    schedule: { currentShiftName: shift?.shiftTemplate?.name ?? null },
    hos: {
      link,
      dutyStatus: reporting ? file.hos.clocks.dutyStatus : null,
      dutyStatusSince: reporting ? file.hos.clocks.dutyStatusSince : null,
      driveRemainingMs: reporting ? file.hos.clocks.driveRemainingMs : null,
      driveLimitMs: reporting ? file.hos.clocks.driveLimitMs : null,
      shiftRemainingMs: reporting ? file.hos.clocks.shiftRemainingMs : null,
      shiftLimitMs: reporting ? file.hos.clocks.shiftLimitMs : null,
      cycleRemainingMs: reporting ? file.hos.clocks.cycleRemainingMs : null,
      cycleLimitMs: reporting ? file.hos.clocks.cycleLimitMs : null,
      breakRemainingMs: reporting ? file.hos.clocks.breakRemainingMs : null,
      breakLimitMs: reporting ? file.hos.clocks.breakLimitMs : null,
      tractorCode: reporting ? file.hos.tractorCode : null,
      recordedAt: reporting ? NOW - 60 : null,
    },
    pay:
      payAssignment && profile
        ? {
            profileName: profile.name,
            classification: profile.classification,
            effectiveFrom: payAssignment.effectiveFrom,
            headline,
          }
        : null,
    benefits: {
      coveredCount: enrollments.filter((row) => row.status === "Active").length,
      notStartedCount: enrollments.filter((row) => row.status === "Pending").length,
      employeeCostPerPeriodMinor: enrollments
        .filter((row) => row.status === "Active")
        .reduce((sum, row) => sum + row.employeeCostMinor, 0),
    },
    portal: {
      access: file.portal.linked
        ? "Linked"
        : file.portal.pendingInvitation
          ? "Invited"
          : "NotSetUp",
      pendingProfileChangeCount: pendingChanges,
      lastSignInAt: file.portal.linked ? (file.portal.portalUser?.lastLoginAt ?? null) : null,
    },
    nextTimeOff,
    documentCount: file.documents.length,
    agentMemoryCount: file.memories.filter((row) => row.status === "Active").length,
    // the variant's plan withholds these the same way a missing permission does
    ...(variant.plan === "core" ? { nextTimeOff: null } : {}),
  };
}

function effectivePayAssignment(file) {
  return (
    file.pay.assignments.find(
      (row) => row.effectiveFrom <= NOW && (!row.effectiveTo || row.effectiveTo >= NOW),
    ) ?? null
  );
}

function payHeadline(profile, assignment) {
  const active = (profile.components ?? []).filter((c) => c.isActive);
  if (!active.length) return null;
  const overrides = new Map(
    (assignment.rateOverrides ?? []).map((row) => [row.componentId, row.rate]),
  );
  const component = active.find((c) => overrides.has(c.id)) ?? active[0];
  return {
    kind: component.kind,
    method: component.method,
    rate: overrides.get(component.id) ?? component.rate,
    overridden: overrides.has(component.id),
  };
}

function payAssignmentOut(row, full = false) {
  const profile = Object.values(PAY_PROFILES).find((p) => p.id === row.payProfileId);
  return {
    ...row,
    payProfile: profile
      ? {
          id: profile.id,
          name: profile.name,
          classification: profile.classification,
          currencyCode: profile.currencyCode,
          guaranteedPeriodMinimumMinor: profile.guaranteedPeriodMinimumMinor,
          components: profile.components.map((c) => ({ ...c, bands: full ? c.bands : c.bands })),
        }
      : null,
  };
}

// ---- Roster -------------------------------------------------------------------------------

const encodeCursor = (offset) => Buffer.from(`offset:${offset}`).toString("base64url");
const decodeCursor = (cursor) => {
  if (!cursor) return 0;
  const match = /offset:(\d+)/.exec(Buffer.from(cursor, "base64url").toString("utf8"));
  return match ? Number(match[1]) : 0;
};

function rosterValue(row, field) {
  return field.split(".").reduce((value, key) => (value == null ? value : value[key]), row);
}

function matchesFilter(row, filter) {
  const value = rosterValue(row, filter.field);
  const list = Array.isArray(filter.value) ? filter.value.map(String) : [String(filter.value)];
  switch (filter.operator) {
    case "eq":
      return String(value) === list[0];
    case "ne":
      return String(value) !== list[0];
    case "in":
      return list.includes(String(value));
    case "notin":
      return !list.includes(String(value));
    case "contains":
      return String(value ?? "")
        .toLowerCase()
        .includes(list[0].toLowerCase());
    case "isnull":
      return value == null;
    case "isnotnull":
      return value != null;
    default:
      return true;
  }
}

function page(rows, input, includeTotalCount = true) {
  const offset = decodeCursor(input?.after);
  const first = input?.first ?? 25;
  const slice = rows.slice(offset, offset + first);
  return {
    edges: slice.map((node, index) => ({ node, cursor: encodeCursor(offset + index + 1) })),
    pageInfo: {
      hasNextPage: offset + first < rows.length,
      endCursor: slice.length ? encodeCursor(offset + slice.length) : null,
      hasPreviousPage: offset > 0,
      startCursor: slice.length ? encodeCursor(offset + 1) : null,
    },
    ...(includeTotalCount
      ? { totalCount: rows.length, totalCountExact: true }
      : { totalCountExact: true }),
  };
}

function sortRows(rows, sort) {
  const order = sort?.length ? sort : [{ field: "lastName", direction: "asc" }];
  return [...rows].sort((a, b) => {
    for (const { field, direction } of order) {
      const x = rosterValue(a, field);
      const y = rosterValue(b, field);
      if (x === y) continue;
      const cmp = x == null ? 1 : y == null ? -1 : x < y ? -1 : 1;
      return direction === "desc" ? -cmp : cmp;
    }
    return 0;
  });
}

function roster(state, input, withoutCredential) {
  let rows = [...ws(state).files.values()].map((file) => workerOut(state, file));
  const query = String(input?.query ?? "")
    .trim()
    .toLowerCase();
  if (query) {
    rows = rows.filter((row) =>
      [row.wholeName, row.email, row.fleetCode?.code, row.city].some((text) =>
        String(text ?? "")
          .toLowerCase()
          .includes(query),
      ),
    );
  }
  for (const filter of input?.fieldFilters ?? [])
    rows = rows.filter((row) => matchesFilter(row, filter));
  for (const group of input?.filterGroups ?? []) {
    const filters = group.filters ?? [];
    if (filters.length)
      rows = rows.filter((row) => filters.some((filter) => matchesFilter(row, filter)));
  }
  if (withoutCredential) {
    rows = rows.filter(
      (row) =>
        !ws(state)
          .files.get(row.id)
          .credentials.some(
            (cred) => cred.status === "Active" && cred.credentialTypeId === withoutCredential,
          ),
    );
  }
  return sortRows(rows, input?.sort);
}

function rosterAttention(state) {
  const files = [...ws(state).files.values()];
  const active = files.filter((file) => file.worker.status === "Active");
  const overviews = active.map((file) => overviewOut(state, file));
  return {
    activeWorkers: active.length,
    nonCompliant: overviews.filter((o) => o.credentials?.complianceStatus === "NonCompliant")
      .length,
    trainingOverdue: overviews.filter((o) => o.training.overdueCount > 0).length,
    atRisk: overviews.filter((o) => o.standing === "AtRisk" || o.standing === "Blocked").length,
    expiringSoon: overviews.filter((o) => (o.credentials?.expiringCount ?? 0) > 0).length,
    reviewsAwaitingSignOff: files.reduce(
      (n, file) => n + file.reviews.filter((r) => r.status === "Submitted").length,
      0,
    ),
    ptoLiabilityDays: canRead(state, "worker_pto")
      ? String(
          active
            .flatMap((file) => ptoBalances(file))
            .filter((row) => row.ptoType === "Vacation")
            .reduce((sum, row) => sum + Number(row.availableDays), 0),
        )
      : null,
    leaveCertificationsOutstanding: canRead(state, "worker_leave")
      ? files.reduce(
          (n, file) =>
            n +
            file.leave.cases.filter(
              (c) => c.certificationStatus === "Requested" || c.certificationStatus === "Overdue",
            ).length,
          0,
        )
      : null,
  };
}

// ---- Employment events: options, plan, apply ----------------------------------------------

const REASON_REQUIRED = ["Terminated", "Suspended", "LeaveStarted", "RateChanged"];

function employmentPhase(file) {
  const wrk = file.worker;
  if (wrk.status !== "Active") return "Left";
  const events = [...file.employmentEvents].sort((a, b) => a.effectiveAt - b.effectiveAt);
  let phase = "Active";
  for (const event of events) {
    if (event.kind === "LeaveStarted") phase = "OnLeave";
    if (event.kind === "LeaveEnded") phase = "Active";
    if (event.kind === "Suspended") phase = "Suspended";
    if (event.kind === "Reinstated") phase = "Active";
  }
  return phase;
}

function recordableKinds(file) {
  switch (employmentPhase(file)) {
    case "Left":
      return ["Rehired"];
    case "OnLeave":
      return ["LeaveEnded", "RateChanged", "Terminated"];
    case "Suspended":
      return ["Reinstated", "Terminated"];
    default: {
      const kinds = [
        "Promoted",
        "Transferred",
        "RateChanged",
        "LeaveStarted",
        "Suspended",
        "Terminated",
      ];
      return file.employmentEvents.some((event) => event.kind === "ProbationEnded")
        ? kinds
        : ["ProbationEnded", ...kinds];
    }
  }
}

function fleetById(id) {
  return Object.values(FLEET_CODES).find((fleet) => fleet.id === id) ?? null;
}

function personById(id) {
  return Object.values(PEOPLE).find((person) => person.id === id) ?? null;
}

/** What recording the event changes on the worker and what else it sets off. */
function planEvent(file, input) {
  const wrk = file.worker;
  const from = {};
  const to = {};
  const effects = [];
  const effect = (kind, extra = {}) =>
    effects.push({
      kind,
      name: null,
      count: null,
      days: null,
      ptoType: null,
      startDate: null,
      endDate: null,
      failed: false,
      ...extra,
    });
  const openChecklists = file.checklists.filter((row) => row.status === "Open");
  switch (input.kind) {
    case "Terminated": {
      from.status = "Active";
      to.status = "Inactive";
      to.terminationDate = String(input.effectiveAt);
      const policy = currentPtoPolicy(file);
      if (policy) effect("PtoPolicyEnded", { name: policy.name });
      for (const balance of ptoBalances(file)) {
        const days = Number(balance.availableDays);
        if (days <= 0) continue;
        effect(balance.ptoType === "Vacation" ? "PtoPaidOut" : "PtoForfeited", {
          ptoType: balance.ptoType,
          days: balance.availableDays,
        });
      }
      for (const request of file.pto.requests.filter(
        (row) =>
          row.status === "Requested" ||
          (row.status === "Approved" && row.startDate > input.effectiveAt),
      )) {
        effect("TimeOffCancelled", {
          ptoType: request.type,
          startDate: request.startDate,
          endDate: request.endDate,
          days: request.days,
        });
      }
      const pay = effectivePayAssignment(file);
      if (pay)
        effect("PayAssignmentEnded", {
          name: Object.values(PAY_PROFILES).find((p) => p.id === pay.payProfileId)?.name ?? null,
        });
      if (file.portal.linked) effect("PortalAccessRevoked");
      effect("ChecklistStarted", { name: CHECKLIST_TEMPLATES.offboarding.name });
      if (openChecklists.length)
        effect("ChecklistClosed", {
          count: openChecklists.length,
          name: openChecklists.length === 1 ? openChecklists[0].name : null,
        });
      break;
    }
    case "Suspended":
      from.canBeAssigned = String(wrk.canBeAssigned);
      to.canBeAssigned = "false";
      effect("DispatchRemoved");
      break;
    case "LeaveStarted":
      from.canBeAssigned = String(wrk.canBeAssigned);
      to.canBeAssigned = "false";
      if (input.leaveType) to.leaveType = input.leaveType;
      effect("DispatchRemoved");
      break;
    case "LeaveEnded":
    case "Reinstated":
      from.canBeAssigned = String(wrk.canBeAssigned);
      to.canBeAssigned = "true";
      effect("DispatchRestored");
      break;
    case "Transferred": {
      const fleet = input.fleetCodeId ? fleetById(input.fleetCodeId) : null;
      const manager = input.managerId ? personById(input.managerId) : null;
      if (fleet) {
        from.fleetCode = wrk.fleetCode?.code ?? "";
        to.fleetCode = fleet.code;
      }
      if (manager) {
        from.manager = wrk.manager?.name ?? "";
        to.manager = manager.name;
      }
      break;
    }
    case "Promoted":
      if (input.driverType) {
        from.driverType = wrk.driverType;
        to.driverType = input.driverType;
      }
      if (input.workerType) {
        from.workerType = wrk.type;
        to.workerType = input.workerType;
      }
      break;
    case "RateChanged": {
      const last = [...file.employmentEvents]
        .filter((e) => e.kind === "RateChanged")
        .sort((a, b) => b.effectiveAt - a.effectiveAt)[0];
      const previous = last?.toValues.find((v) => v.key === "rate")?.value;
      if (previous) from.rate = previous;
      if (input.rate) to.rate = input.rate;
      if (input.rateUnit) to.rateUnit = input.rateUnit;
      break;
    }
    case "Rehired": {
      from.status = "Inactive";
      to.status = "Active";
      to.hireDate = String(input.effectiveAt);
      effect("DispatchRestored");
      effect("ChecklistStarted", { name: CHECKLIST_TEMPLATES.onboarding.name });
      effect("TrainingAssigned", { count: requiredCourses(wrk.driverType).length });
      effect("DefaultPtoPolicyApplied", {
        name: (wrk.driverType === "Local" ? PTO_POLICIES.local : PTO_POLICIES.otr).name,
      });
      break;
    }
    default:
      break;
  }
  const values = (pairs) => Object.entries(pairs).map(([key, value]) => ({ key, value }));
  return { fromValues: values(from), toValues: values(to), effects };
}

function validateEvent(file, input) {
  if (!recordableKinds(file).includes(input.kind))
    refuse(`A ${input.kind} event cannot be recorded for this worker now.`);
  if (REASON_REQUIRED.includes(input.kind) && !input.reason?.trim())
    refuse("Reason is required for this event.");
  if (input.kind === "Transferred" && !input.fleetCodeId && !input.managerId)
    refuse("A transfer needs a fleet code or a manager.");
  if (input.kind === "LeaveStarted" && !input.leaveType) refuse("Leave type is required.");
  if (input.kind === "RateChanged" && !input.rate) refuse("New rate is required.");
}

function applyEvent(state, file, input, plan) {
  const wrk = file.worker;
  switch (input.kind) {
    case "Terminated": {
      wrk.status = "Inactive";
      wrk.canBeAssigned = false;
      wrk.availableForDispatch = false;
      wrk.profile.terminationDate = input.effectiveAt;
      for (const row of file.pto.policyAssignments)
        if (!row.effectiveTo) row.effectiveTo = input.effectiveAt;
      for (const balance of ptoBalances(file)) {
        const days = Number(balance.availableDays);
        if (days > 0)
          file.pto.ledger.push({
            ptoType: balance.ptoType,
            entryType: "Expiry",
            amount: -days,
            effectiveAt: input.effectiveAt,
            note:
              balance.ptoType === "Vacation"
                ? "Paid out on termination"
                : "Forfeited on termination",
          });
      }
      for (const request of file.pto.requests) {
        if (
          request.status === "Requested" ||
          (request.status === "Approved" && request.startDate > input.effectiveAt)
        ) {
          bump(request, {
            status: "Cancelled",
            cancellationReason: "Employment ended",
            cancelledById: PEOPLE.dana.id,
            cancelledBy: DANA(),
          });
        }
      }
      const pay = effectivePayAssignment(file);
      if (pay) pay.effectiveTo = input.effectiveAt;
      if (file.portal.linked) {
        file.portal.linked = false;
        file.portal.portalUser = null;
      }
      for (const row of file.checklists.filter((c) => c.status === "Open")) {
        bump(row, { status: "Cancelled", cancelledAt: NOW, cancelReason: "Employment ended" });
      }
      file.checklists.unshift(startChecklist(file, CHECKLIST_TEMPLATES.offboarding, NOW));
      break;
    }
    case "Suspended":
    case "LeaveStarted":
      wrk.canBeAssigned = false;
      wrk.availableForDispatch = false;
      if (input.kind === "LeaveStarted") wrk.leaveType = input.leaveType;
      break;
    case "LeaveEnded":
    case "Reinstated":
      wrk.canBeAssigned = true;
      wrk.availableForDispatch = true;
      wrk.leaveType = null;
      break;
    case "Transferred": {
      const fleet = input.fleetCodeId ? fleetById(input.fleetCodeId) : null;
      const manager = input.managerId ? personById(input.managerId) : null;
      if (fleet) {
        wrk.fleetCodeId = fleet.id;
        wrk.fleetCode = { id: fleet.id, code: fleet.code, color: fleet.color };
      }
      if (manager) {
        wrk.managerId = manager.id;
        wrk.manager = userRef(manager);
      }
      break;
    }
    case "Promoted":
      if (input.driverType) wrk.driverType = input.driverType;
      if (input.workerType) wrk.type = input.workerType;
      break;
    case "Rehired":
      wrk.status = "Active";
      wrk.canBeAssigned = true;
      wrk.availableForDispatch = true;
      wrk.profile.terminationDate = null;
      file.checklists.unshift(startChecklist(file, CHECKLIST_TEMPLATES.onboarding, NOW));
      break;
    default:
      break;
  }
  const event = employmentEvent(wrk.id, 0, input.kind, input.effectiveAt, {
    reason: input.reason ?? null,
    notes: input.notes ?? null,
    recordedBy: "dana",
    recordedAt: NOW,
  });
  event.id = nextId("wee");
  event.fromValues = plan.fromValues;
  event.toValues = plan.toValues;
  file.employmentEvents.push(event);
  audit(file, {
    resource: "worker_employment_event",
    resourceId: event.id,
    operation: "create",
    entityRef: input.kind,
  });
  touch(file, "worker_employment_event", event.id);
  return event;
}

function startChecklist(file, template, startedAt) {
  const id = nextId("wck");
  return {
    id,
    workerId: file.worker.id,
    templateId: template.id,
    name:
      template.kind === "Custom"
        ? `${template.name} ${new Date(startedAt * 1000).getUTCFullYear()}`
        : template.name,
    kind: template.kind,
    status: "Open",
    startedAt,
    dueAt: startedAt + Math.max(...template.items.map((item) => item.dueOffsetDays)) * DAY,
    completedAt: null,
    cancelledAt: null,
    cancelReason: null,
    sourceEventId: null,
    startedById: PEOPLE.dana.id,
    version: 1,
    createdAt: startedAt,
    updatedAt: startedAt,
    startedBy: DANA(),
    items: template.items.map((item) => ({
      id: nextId("wci"),
      checklistId: id,
      label: item.label,
      description: item.description,
      kind: item.kind,
      required: item.required,
      owner: item.owner,
      dueAt: startedAt + item.dueOffsetDays * DAY,
      credentialTypeId: item.credentialTypeId,
      documentTypeId: null,
      status: "Pending",
      completedById: null,
      completedAt: null,
      autoCompleted: false,
      note: null,
      evidenceDocumentId: null,
      evidenceCredentialId: null,
      sortOrder: item.sortOrder,
      version: 1,
      completedBy: null,
      evidenceDocument: null,
      evidenceCredential: null,
      credentialType: item.credentialType,
    })),
  };
}

// ---- Hours of service ---------------------------------------------------------------------

/** Each day's duty segments in minutes from midnight, from the mockup; index 0 is today. */
const LOG_DAYS = [
  {
    mi: 212,
    segs: [
      ["offDuty", 0, 390, "Rest, Pilot travel center, Gary, IN"],
      ["onDuty", 390, 430, "Pre-trip, fuel"],
      ["driving", 430, 572, "I-94 west to Kenosha, WI"],
      ["onDuty", 572, 612, "Loading at the shipper, Kenosha, WI"],
      ["offDuty", 612, 810, "Waiting on the dock, counts as her break"],
      ["onDuty", 810, 822, "Paperwork, pulled out"],
      ["driving", 822, 968, "I-94 and I-294 toward Joliet, IL"],
    ],
  },
  {
    mi: 471,
    cert: 1,
    segs: [
      ["offDuty", 0, 300, "Rest, Des Moines, IA"],
      ["onDuty", 300, 340, "Pre-trip inspection"],
      ["driving", 340, 610, "I-80 east"],
      ["onDuty", 610, 670, "Loading at the shipper, Davenport, IA"],
      ["offDuty", 670, 700, "Break"],
      ["driving", 700, 940, "I-80 east to Gary, IN"],
      ["onDuty", 940, 970, "Post-trip, fuel at Pilot, Gary, IN"],
      ["offDuty", 970, 1440, "Off duty, Gary, IN"],
    ],
  },
  {
    mi: 446,
    cert: 1,
    segs: [
      ["offDuty", 0, 315, "Rest, Crawfordsville, IN"],
      ["onDuty", 315, 345, "Pre-trip inspection"],
      ["driving", 345, 630, "I-74 to Peoria, IL"],
      ["onDuty", 630, 690, "Unloading at the receiver"],
      ["offDuty", 690, 720, "Break"],
      ["driving", 720, 915, "I-74 and I-80 to Des Moines, IA"],
      ["onDuty", 915, 935, "Post-trip"],
      ["offDuty", 935, 1440, "Off duty, Des Moines, IA"],
    ],
  },
  {
    mi: 527,
    cert: 1,
    segs: [
      ["sleeperBed", 0, 300, "Sleeper berth, restart complete"],
      ["onDuty", 300, 330, "Pre-trip inspection"],
      ["driving", 330, 600, "I-65 north"],
      ["offDuty", 600, 630, "Break"],
      ["driving", 630, 840, "I-65 to Indianapolis, IN"],
      ["onDuty", 840, 930, "Loading at the shipper"],
      ["driving", 930, 1020, "I-74 west"],
      ["onDuty", 1020, 1040, "Fuel, Crawfordsville, IN"],
      ["offDuty", 1040, 1440, "Off duty, Crawfordsville, IN"],
    ],
  },
  {
    mi: 0,
    cert: 1,
    segs: [
      ["sleeperBed", 0, 450, "Sleeper berth"],
      ["offDuty", 450, 1290, "34-hour restart, Nashville, TN"],
      ["sleeperBed", 1290, 1440, "Sleeper berth"],
    ],
  },
  {
    mi: 381,
    cert: 2,
    segs: [
      ["offDuty", 0, 300, "Rest, Memphis, TN"],
      ["onDuty", 300, 330, "Pre-trip inspection"],
      ["driving", 330, 585, "I-40 east"],
      ["onDuty", 585, 645, "Loading at the shipper, Jackson, TN"],
      ["driving", 645, 810, "I-40 to Nashville, TN"],
      ["onDuty", 810, 830, "Post-trip"],
      ["offDuty", 830, 1440, "34-hour restart begins, Nashville, TN"],
    ],
  },
  {
    mi: 468,
    cert: 1,
    segs: [
      ["offDuty", 0, 330, "Rest, TA travel center, Effingham, IL"],
      ["onDuty", 330, 360, "Pre-trip inspection"],
      ["driving", 360, 615, "I-57 south toward Memphis"],
      ["onDuty", 615, 660, "Fuel and scale, Marion, IL"],
      ["offDuty", 660, 690, "Break"],
      ["driving", 690, 945, "I-55 into Memphis, TN"],
      ["onDuty", 945, 1020, "Unloading at the receiver"],
      ["offDuty", 1020, 1440, "Off duty, Memphis, TN"],
    ],
  },
];
const LIGHT_DAY = {
  mi: 236,
  cert: 1,
  segs: [
    ["offDuty", 0, 360, null],
    ["onDuty", 360, 390, "Pre-trip inspection"],
    ["driving", 390, 690, null],
    ["offDuty", 690, 720, "Break"],
    ["driving", 720, 900, null],
    ["onDuty", 900, 930, "Post-trip"],
    ["offDuty", 930, 1440, null],
  ],
};

const startOfUtcDay = (t) => t - (t % DAY);

/** The day pattern for a calendar day: today is the day the page is drawn, earlier days count back. */
function dayPattern(file, dayStart) {
  const back = Math.round((startOfUtcDay(NOW) - dayStart) / DAY);
  if (file.worker.id !== MARIA_ID)
    return back === 0
      ? { ...LIGHT_DAY, segs: LIGHT_DAY.segs.filter(([, from]) => from < 600) }
      : LIGHT_DAY;
  return LOG_DAYS[((back % 7) + 7) % 7];
}

function logEntries(file, startTime, endTime) {
  const out = [];
  for (let dayStart = startOfUtcDay(startTime); dayStart < endTime; dayStart += DAY) {
    const isToday = dayStart === startOfUtcDay(NOW);
    const pattern = dayPattern(file, dayStart);
    pattern.segs.forEach(([status, from, to, remark], index) => {
      const last = index === pattern.segs.length - 1;
      out.push({
        hosStatusType: status,
        logStartAt: dayStart + from * 60,
        logEndAt: isToday && last ? null : dayStart + to * 60,
        remark,
        vehicleId: status === "offDuty" || status === "sleeperBed" ? null : file.hos.vehicleId,
        vehicleName: status === "offDuty" || status === "sleeperBed" ? null : file.hos.tractorCode,
        latitude: null,
        longitude: null,
        codrivers: null,
      });
    });
  }
  return out.filter((entry) => entry.logStartAt < endTime && (entry.logEndAt ?? NOW) > startTime);
}

function dailyLogs(file, startDate, endDate) {
  const parse = (text) => Date.parse(`${text}T00:00:00Z`) / 1000;
  const out = [];
  for (let dayStart = parse(startDate); dayStart <= parse(endDate); dayStart += DAY) {
    const isToday = dayStart === startOfUtcDay(NOW);
    const pattern = dayPattern(file, dayStart);
    const totals = { offDuty: 0, sleeperBed: 0, driving: 0, onDuty: 0 };
    pattern.segs.forEach(([status, from, to]) => {
      totals[status] +=
        (isToday
          ? Math.min(to, (NOW - dayStart) / 60) - Math.min(from, (NOW - dayStart) / 60)
          : to - from) * 60_000;
    });
    out.push({
      startAt: dayStart,
      endAt: dayStart + DAY,
      driveDistanceMeters: Math.round(pattern.mi * 1609.344),
      activeDurationMs: totals.driving + totals.onDuty,
      driveDurationMs: totals.driving,
      onDutyDurationMs: totals.onDuty,
      offDutyDurationMs: totals.offDuty,
      sleeperBerthDurationMs: totals.sleeperBed,
      personalConveyanceDurationMs: 0,
      yardMoveDurationMs: 0,
      isCertified: !isToday && !!pattern.cert,
      certifiedAt: !isToday && pattern.cert ? dayStart + pattern.cert * DAY + 7 * 3600 : null,
      shippingDocs: pattern.mi ? "BOL on file" : null,
      vehicleNames: pattern.mi ? [file.hos.tractorCode] : [],
    });
  }
  return out;
}

function hosState(state, file) {
  if (hosLink(state, file) !== "Reporting") return null;
  const clocks = file.hos.clocks;
  return {
    workerId: file.worker.id,
    workerName: file.worker.wholeName,
    provider: file.hos.provider,
    providerDriverId: file.hos.providerDriverId,
    ...clocks,
    shiftDrivingViolationMs: 0,
    cycleViolationMs: 0,
    currentVehicleId:
      clocks.dutyStatus === "driving" || clocks.dutyStatus === "onDuty" ? file.hos.vehicleId : null,
    currentTractorId:
      clocks.dutyStatus === "driving" || clocks.dutyStatus === "onDuty" ? file.hos.tractorId : null,
    rulesetCycle: "USA 70 hour / 8 day",
    rulesetShift: "US Interstate Property",
    rulesetJurisdiction: "US",
    recordedAt: NOW - 60,
  };
}

// ---- Select options -----------------------------------------------------------------------

function option(id, label, description, meta) {
  return { id, label, description: description ?? null, meta: meta ?? null };
}

function workerSelectRows(state, resource) {
  switch (resource) {
    case "FLEET_CODE":
      return Object.values(FLEET_CODES).map((f) =>
        option(f.id, f.code, f.description, { code: f.code, color: f.color, name: f.description }),
      );
    case "JOB_POSITION":
      return Object.values(POSITIONS).map((p) => option(p.id, p.title, null, { title: p.title }));
    case "PERFORMANCE_REVIEW_TEMPLATE":
      return Object.values(REVIEW_TEMPLATES).map((t) =>
        option(t.id, t.name, t.description, {
          code: t.code,
          cadenceMonths: t.cadenceMonths,
          itemCount: t.items.length,
        }),
      );
    case "PTO_POLICY":
      return Object.values(PTO_POLICIES).map((p) =>
        option(p.id, p.name, p.code, { code: p.code, status: p.status }),
      );
    case "SHIFT_TEMPLATE":
      return Object.values(SHIFT_TEMPLATES).map((s) =>
        option(s.id, s.name, s.code, {
          code: s.code,
          color: s.color,
          daysOfWeek: s.daysOfWeek,
          startMinute: s.startMinute,
          durationMinutes: s.durationMinutes,
          cycleWeeks: s.cycleWeeks,
        }),
      );
    case "TRAINING_COURSE":
      return COURSE_LIST.map((c) =>
        option(c.id, c.name, c.description, {
          code: c.code,
          category: c.category,
          delivery: c.delivery,
          durationMinutes: c.durationMinutes,
          isRequired: c.isRequired,
          validityMonths: c.validityMonths,
        }),
      );
    case "WORKER_CREDENTIAL_TYPE":
      return CREDENTIAL_TYPE_LIST.map((t) =>
        option(t.id, t.name, null, {
          code: t.code,
          category: t.category,
          isRequired: t.isRequired,
          validityMonths: t.validityMonths,
          requiresNumber: t.requiresNumber,
        }),
      );
    case "PAY_PROFILE":
      return Object.values(PAY_PROFILES).map((p) =>
        option(p.id, p.name, null, {
          classification: p.classification,
          componentCount: p.components.length,
        }),
      );
    case "BENEFIT_PLAN":
      return Object.values(BENEFIT_PLANS).map((p) =>
        option(p.id, p.name, p.carrier, { code: p.code, planType: p.planType, carrier: p.carrier }),
      );
    case "USER":
      return Object.values(PEOPLE).map((p) =>
        option(p.id, p.name, p.emailAddress, { emailAddress: p.emailAddress }),
      );
    case "US_STATE":
      return Object.values(STATES).map((s) =>
        option(s.id, s.name, s.abbreviation, { abbreviation: s.abbreviation }),
      );
    case "WORKER":
      return [...ws(state).files.values()].map((file) =>
        option(file.worker.id, file.worker.wholeName, file.worker.fleetCode?.code ?? null, {
          fleetCode: file.worker.fleetCode?.code ?? "",
          status: file.worker.status,
        }),
      );
    default:
      return null;
  }
}

function selectConnection(rows, input) {
  let list = rows;
  if (input.ids?.length) list = list.filter((row) => input.ids.includes(row.id));
  const query = String(input.query ?? "").toLowerCase();
  if (query)
    list = list.filter((row) =>
      `${row.label} ${row.description ?? ""}`.toLowerCase().includes(query),
    );
  const offset = input.offset ?? 0;
  const first = input.first ?? 20;
  const slice = list.slice(offset, offset + first);
  return {
    selectOptions: {
      edges: slice.map((node, index) => ({ node, cursor: encodeCursor(offset + index + 1) })),
      pageInfo: {
        hasNextPage: offset + first < list.length,
        endCursor: slice.length ? encodeCursor(offset + slice.length) : null,
      },
      totalCount: list.length,
      totalCountExact: true,
    },
  };
}

// ---- Handlers -----------------------------------------------------------------------------

const byNewest = (field) => (a, b) => (b[field] ?? 0) - (a[field] ?? 0);

function profileChangesFor(state, filter = {}) {
  const files = filter.workerId ? [fileOf(state, filter.workerId)] : [...ws(state).files.values()];
  let rows = files.flatMap((file) =>
    file.portal.profileChanges.map((row) => ({
      ...row,
      worker: {
        id: file.worker.id,
        firstName: file.worker.firstName,
        lastName: file.worker.lastName,
      },
    })),
  );
  if (filter.statuses?.length) rows = rows.filter((row) => filter.statuses.includes(row.status));
  return rows.sort(byNewest("submittedAt")).slice(0, filter.limit ?? 50);
}

function memoryList(state, input) {
  const files = input.subjectId
    ? isWorkerId(state, input.subjectId)
      ? [fileOf(state, input.subjectId)]
      : []
    : [...ws(state).files.values()];
  let rows = files.flatMap((file) => file.memories).filter((row) => row.status !== "Retired");
  if (input.scope) rows = rows.filter((row) => row.scope === input.scope);
  const query = String(input.query ?? "").toLowerCase();
  if (query) rows = rows.filter((row) => row.content.toLowerCase().includes(query));
  const offset = decodeCursor(input.after);
  const first = input.first ?? 50;
  const slice = rows.slice(offset, offset + first);
  const scopes = ["Organization", "Role", "User", "Agent"];
  return {
    items: slice,
    next: offset + first < rows.length ? encodeCursor(offset + first) : null,
    all: rows.length,
    counts: scopes
      .map((scope) => ({
        scope,
        roleId: null,
        count: rows.filter((row) => row.scope === scope).length,
      }))
      .filter((row) => row.count > 0),
  };
}

function memoryById(state, id) {
  return fileHolding(state, (file) => file.memories, id);
}

function activityFor(state, v) {
  const file = fileOf(state, v.subjectId);
  if (!canRead(state, "audit_log")) refuse("You do not have permission to read the audit log.");
  let rows = [...file.activity].sort(byNewest("timestamp"));
  rows = rows.filter((row) => canRead(state, row.resource));
  for (const filter of v.input?.fieldFilters ?? [])
    rows = rows.filter((row) => matchesFilter(row, filter));
  const query = String(v.input?.query ?? "")
    .trim()
    .toLowerCase();
  if (query) {
    rows = rows.filter((row) =>
      [row.entityRef, row.comment, row.actorName, row.resource].some((text) =>
        String(text ?? "")
          .toLowerCase()
          .includes(query),
      ),
    );
  }
  const result = page(rows, v.input, v.includeTotalCount);
  return {
    edges: result.edges.map(({ node }) => ({ node })),
    pageInfo: { hasNextPage: result.pageInfo.hasNextPage, endCursor: result.pageInfo.endCursor },
    totalCount: v.includeTotalCount ? rows.length : null,
    totalCountExact: true,
  };
}

function ptoRows(state, input) {
  let files = input.workerId ? [fileOf(state, input.workerId)] : [...ws(state).files.values()];
  if (input.fleetCodeId)
    files = files.filter((file) => file.worker.fleetCodeId === input.fleetCodeId);
  let rows = files.flatMap((file) => file.pto.requests);
  if (input.status) rows = rows.filter((row) => row.status === input.status);
  if (input.type) rows = rows.filter((row) => row.type === input.type);
  if (input.startDate) rows = rows.filter((row) => row.endDate >= input.startDate);
  if (input.endDate) rows = rows.filter((row) => row.startDate <= input.endDate);
  for (const filter of input.fieldFilters ?? [])
    rows = rows.filter((row) => matchesFilter(row, filter));
  return sortRows(
    rows,
    input.sort?.length ? input.sort : [{ field: "startDate", direction: "desc" }],
  );
}

function findPto(state, id) {
  return fileHolding(state, (file) => file.pto.requests, id);
}

function decidePto(state, id, patch, operation) {
  const { file, row } = findPto(state, id);
  if (row.status !== "Requested" && operation !== "cancel")
    refuse("Only a requested time off can be decided.");
  bump(row, patch);
  if (row.status === "Approved") {
    file.pto.ledger.push({
      ptoType: row.type,
      entryType: "Usage",
      amount: -Number(row.days),
      effectiveAt: row.endDate,
      note: "Time off taken",
      sourcePtoId: row.id,
    });
    row.balanceAfterDays =
      ptoBalances(file).find((b) => b.ptoType === row.type)?.balanceDays ?? null;
  }
  audit(file, {
    resource: "worker_pto",
    resourceId: row.id,
    operation,
    entityRef: `${row.type}, ${row.days} days`,
  });
  touch(file, "worker_pto", row.id);
  return row;
}

function enrollmentOut(row) {
  return { ...row, benefitPlan: row.benefitPlan ? { ...row.benefitPlan } : null };
}

function totalCompensation(file, planYear) {
  const totals = file.benefits.totals;
  const year = planYear ?? new Date(NOW * 1000).getUTCFullYear();
  return {
    asOf: NOW,
    planYear: year,
    ...totals,
    totalCompensationMinor: totals.grossPayMinor + totals.employerBenefitMinor,
    enrollments: file.benefits.enrollments.map(enrollmentOut),
  };
}

function reviewOut(review) {
  return {
    ...review,
    ratings: review.ratings.map((r) => ({ ...r })),
    goals: review.goals.map((g) => ({ ...g })),
  };
}

function overallScore(ratings) {
  const scored = ratings.filter((r) => r.score != null);
  if (!scored.length) return null;
  const weight = scored.reduce((sum, r) => sum + r.weight, 0);
  return (
    Math.round((scored.reduce((sum, r) => sum + r.score * r.weight, 0) / weight) * 10) / 10
  ).toFixed(1);
}

function newSafetyEvent(file, input) {
  const id = nextId("wse");
  return {
    id,
    ...TENANT,
    workerId: file.worker.id,
    kind: input.kind,
    severity: input.severity,
    status: "Open",
    occurredAt: input.occurredAt,
    location: input.location ?? null,
    description: input.description,
    preventable: input.preventable ?? false,
    points: input.points ?? defaultPoints(input),
    pointsExpireAt: input.pointsExpireAt ?? input.occurredAt + 365 * DAY,
    referenceNumber: input.referenceNumber ?? null,
    shipmentId: input.shipmentId ?? null,
    inspectionLevel: input.inspectionLevel ?? null,
    inspectionResult: input.inspectionResult ?? null,
    outOfService: input.outOfService ?? false,
    fineAmount: input.fineAmount ?? null,
    costAmount: input.costAmount ?? null,
    documentId: input.documentId ?? null,
    document: input.documentId ? documentRefOf(file, input.documentId) : null,
    recordedById: PEOPLE.dana.id,
    recordedBy: DANA(),
    closedById: null,
    closedBy: null,
    closedAt: null,
    resolution: null,
    version: 1,
    createdAt: NOW,
    updatedAt: NOW,
    violations: [],
  };
}

function defaultPoints({ kind, severity, preventable, inspectionResult }) {
  const base = { Minor: 1, Moderate: 2, Major: 4, Critical: 6 }[severity] ?? 1;
  switch (kind) {
    case "Accident":
      return base + (preventable ? 3 : 0);
    case "Incident":
      return base + (preventable ? 2 : 0);
    case "Citation":
      return base + 1;
    case "Inspection":
      return inspectionResult === "OutOfService"
        ? base + 4
        : inspectionResult === "Fail"
          ? base + 2
          : 0;
    default:
      return 0;
  }
}

function documentRefOf(file, documentId) {
  const doc = file.documents.find((row) => row.id === documentId);
  return doc
    ? {
        id: doc.id,
        fileName: doc.fileName,
        originalName: doc.originalName,
        fileType: doc.fileType,
        fileSize: doc.fileSize,
        previewStatus: doc.previewStatus,
        createdAt: doc.createdAt,
      }
    : {
        id: documentId,
        fileName: "attachment.pdf",
        originalName: "attachment.pdf",
        fileType: "application/pdf",
        fileSize: 120_000,
        previewStatus: "Unsupported",
        createdAt: NOW,
      };
}

function checklistItemAction(state, input, status) {
  const { file, row: checklist } = fileHolding(
    state,
    (f) => f.checklists,
    findChecklistIdOfItem(state, input.id),
  );
  const item = checklist.items.find((entry) => entry.id === input.id);
  if (checklist.status !== "Open") refuse("The checklist is closed.");
  bump(item, {
    status,
    note: input.note ?? item.note,
    completedById: status === "Pending" ? null : PEOPLE.dana.id,
    completedBy: status === "Pending" ? null : DANA(),
    completedAt: status === "Pending" ? null : NOW,
    autoCompleted: false,
    evidenceDocumentId: input.evidenceDocumentId ?? item.evidenceDocumentId,
  });
  const out = checklistOut(checklist);
  if (out.progress.complete && checklist.items.every((entry) => entry.status !== "Pending")) {
    bump(checklist, { status: "Completed", completedAt: NOW });
  } else {
    bump(checklist, {});
  }
  audit(file, {
    resource: "worker_checklist",
    resourceId: checklist.id,
    operation: "update",
    entityRef: checklist.name,
    comment: `${status === "Pending" ? "Reopened" : status === "Done" ? "Completed" : status === "Skipped" ? "Skipped" : "Marked not applicable"} ${item.label}`,
  });
  touch(file, "worker_checklist", checklist.id);
  return checklistOut(checklist);
}

function findChecklistIdOfItem(state, itemId) {
  for (const file of ws(state).files.values()) {
    for (const checklist of file.checklists) {
      if (checklist.items.some((item) => item.id === itemId)) return checklist.id;
    }
  }
  refuse("Checklist item not found");
}

function trainingWrite(state, id) {
  return fileHolding(state, (file) => file.training, id);
}

function assignTraining(file, courseId, dueAt, notes) {
  const course = courseOf(courseId);
  if (!course) refuse("Course not found");
  const existing = file.training.find(
    (row) =>
      row.courseId === courseId && (row.status === "Assigned" || row.status === "InProgress"),
  );
  if (existing) return { record: existing, skipped: true };
  const record = trainingRecord(
    file.worker.id,
    0,
    Object.keys(COURSES).find((key) => COURSES[key].id === courseId),
    {
      status: "Assigned",
      assignedAt: NOW,
      dueAt: dueAt ?? NOW + course.dueDaysAfterAssignment * DAY,
      notes: notes ?? null,
    },
  );
  record.id = nextId("wtr");
  file.training.push(record);
  file.trainingSlots[courseId] = record.id;
  audit(file, {
    resource: "worker_training",
    resourceId: record.id,
    operation: "assign",
    entityRef: course.name,
  });
  return { record, skipped: false };
}

function credentialProfileSync(file, type, cred) {
  if (!type.profileField || !cred.expiresAt) return;
  if (type.profileField in file.worker.profile)
    file.worker.profile[type.profileField] = cred.expiresAt;
}

/** The REST save merges what the form sends and keeps the ten fields the server derives. */
const SERVER_KEPT = [
  "complianceStatus",
  "nextCredentialExpiry",
  "trainingHealth",
  "nextTrainingDue",
  "safetyRating",
  "safetyScore",
  "drugAlcoholStatus",
  "returnToDutyStatus",
  "lastClearinghouseQueryAt",
  "nextClearinghouseQueryDue",
];
const PERSONAL = ["dob", "licenseNumber", "twicCardNumber"];

function saveWorker(state, file, body) {
  const wrk = file.worker;
  const {
    profile: profileIn,
    state: _state,
    fleetCode: _fleet,
    pto: _pto,
    position: _position,
    manager: _manager,
    ...rest
  } = body ?? {};
  const personal = canRead(state, "protected_personal");
  for (const key of [
    "firstName",
    "lastName",
    "gender",
    "type",
    "driverType",
    "addressLine1",
    "addressLine2",
    "city",
    "postalCode",
    "email",
    "phoneNumber",
    "emergencyContactName",
    "emergencyContactPhone",
    "canBeAssigned",
    "availableForDispatch",
    "homeLatitude",
    "homeLongitude",
    "homePlaceId",
    "lastHomeAt",
    "customFields",
    "positionId",
    "managerId",
    "fleetCodeId",
    "stateId",
  ]) {
    if (key in rest) wrk[key] = rest[key];
  }
  wrk.wholeName = `${wrk.firstName} ${wrk.lastName}`;
  const fleet = fleetById(wrk.fleetCodeId);
  wrk.fleetCode = fleet ? { id: fleet.id, code: fleet.code, color: fleet.color } : null;
  const position = Object.values(POSITIONS).find((p) => p.id === wrk.positionId);
  wrk.position = position ? { ...position } : null;
  const manager = personById(wrk.managerId);
  wrk.manager = manager ? userRef(manager) : null;
  const homeState = Object.values(STATES).find((s) => s.id === wrk.stateId);
  if (homeState) wrk.state = { ...homeState };
  if (profileIn) {
    for (const [key, value] of Object.entries(profileIn)) {
      if (
        SERVER_KEPT.includes(key) ||
        [
          "id",
          "workerId",
          "version",
          "createdAt",
          "updatedAt",
          "licenseState",
          "organizationId",
          "businessUnitId",
          "hireDate",
          "terminationDate",
        ].includes(key)
      )
        continue;
      if (PERSONAL.includes(key) && !personal) continue;
      wrk.profile[key] = value;
    }
    if (profileIn.hireDate && file.employmentEvents.length <= 1)
      wrk.profile.hireDate = profileIn.hireDate;
    const licenseState = Object.values(STATES).find((s) => s.id === wrk.profile.licenseStateId);
    if (licenseState) wrk.profile.licenseState = { ...licenseState };
    wrk.profile.version += 1;
    wrk.profile.updatedAt = NOW;
  }
  wrk.version += 1;
  wrk.updatedAt = NOW;
  audit(file, { resource: "worker", operation: "update", entityRef: wrk.wholeName });
  touch(file);
  return workerOut(state, file);
}

function createWorker(state, body) {
  const store = ws(state);
  store.created += 1;
  const id = pid("wrk", 500 + store.created);
  const hireDate = body?.profile?.hireDate ?? NOW;
  const blank = workerRecord({
    id,
    firstName: body?.firstName ?? "New",
    lastName: body?.lastName ?? "Worker",
    gender: body?.gender ?? "Female",
    dob: body?.profile?.dob ?? null,
    driverType: body?.driverType ?? "OTR",
    type: body?.type ?? "Employee",
    addressLine1: body?.addressLine1 ?? "",
    city: body?.city ?? "",
    postalCode: body?.postalCode ?? "",
    phoneNumber: body?.phoneNumber ?? "",
    licenseNumber: body?.profile?.licenseNumber ?? null,
    licenseExpiry: body?.profile?.licenseExpiry ?? NOW + 365 * DAY,
    hireDate,
    isQualified: false,
    canBeAssigned: false,
    availableForDispatch: false,
  });
  blank.version = 1;
  const file = {
    worker: blank,
    credentials: [],
    dqf: { hireDate, safetyHistoryDueAt: hireDate + 30 * DAY, items: [], verifications: [] },
    drugAlcohol: { tests: [], violations: [], queries: [], randomSelections: [] },
    training: [],
    trainingSlots: {},
    reviews: [],
    safety: {
      scorecard: {
        score: 100,
        rating: "Excellent",
        pointsWatchThreshold: 6,
        pointsAtRiskThreshold: 10,
        accidents: 0,
        preventableAccidents: 0,
        incidents: 0,
        nearMisses: 0,
        citations: 0,
        inspections: 0,
        inspectionsPassed: 0,
        inspectionsFailed: 0,
        outOfServiceOrders: 0,
        cleanInspectionRate: null,
      },
      events: [],
      actions: [],
      recognitions: [],
      injuries: [],
    },
    employmentEvents: [],
    checklists: [],
    leave: {
      entitlement: {
        method: "RollingBackward",
        totalHours: "480",
        usedHours: "0",
        remainingHours: "480",
        totalWeeks: "12",
        usedWeeks: "0",
        remainingWeeks: "12",
        exhausted: false,
        eligibleOnTenure: false,
        monthsEmployed: 0,
        militaryCaregiver: false,
        window: { from: NOW - 365 * DAY, through: NOW },
      },
      cases: [],
    },
    pto: { policyAssignments: [], balances: {}, requests: [], ledger: [] },
    schedule: { assignments: [], availability: [], swaps: [] },
    hos: {
      link: "NotLinked",
      provider: "Samsara",
      providerDriverId: "",
      tractorId: null,
      tractorCode: null,
      vehicleId: null,
      clocks: {},
      forms: [],
    },
    pay: {
      assignments: [],
      earnings: {
        accruedEventCount: 0,
        accruedGrossMinor: 0,
        outstandingAdvances: 0,
        escrowBalanceMinor: 0,
      },
    },
    benefits: {
      enrollments: [],
      totals: {
        grossPayMinor: 0,
        employerBenefitMinor: 0,
        employeeBenefitMinor: 0,
        accruedTimeOffDays: "0",
      },
    },
    portal: {
      linked: false,
      portalUser: null,
      pendingInvitation: null,
      invitations: [],
      acknowledgements: [],
      profileChanges: [],
    },
    documents: [],
    memories: [],
    activity: [],
  };
  store.files.set(id, file);
  saveWorker(state, file, body);
  // Creating a worker records Hired, which starts onboarding, assigns required training and
  // applies the default time off policy, as the server's create does.
  const hired = employmentEvent(id, 0, "Hired", hireDate, {
    reason: "New hire",
    recordedBy: "dana",
    to: { status: "Active", ...(blank.fleetCode ? { fleetCode: blank.fleetCode.code } : {}) },
  });
  hired.id = nextId("wee");
  file.employmentEvents.push(hired);
  file.checklists.push(startChecklist(file, CHECKLIST_TEMPLATES.onboarding, NOW));
  for (const course of requiredCourses(blank.driverType))
    assignTraining(file, course.id, null, null);
  const policy = blank.driverType === "Local" ? PTO_POLICIES.local : PTO_POLICIES.otr;
  file.pto.policyAssignments.push({
    id: nextId("wpa"),
    workerId: id,
    ptoPolicyId: policy.id,
    effectiveFrom: hireDate,
    effectiveTo: null,
    assignedById: PEOPLE.dana.id,
    note: null,
    version: 1,
    createdAt: NOW,
    updatedAt: NOW,
    ptoPolicy: { ...policy },
  });
  file.pto.balances = {
    Vacation: { maxBalanceDays: "15", carriedDays: "0", nextAccrual: null },
    Sick: { maxBalanceDays: "5", carriedDays: "0", nextAccrual: null },
  };
  file.activity = [];
  audit(file, { resource: "worker", operation: "create", entityRef: blank.wholeName });
  publishInvalidation({ resource: "worker", action: "created", recordId: id });
  return workerOut(state, file);
}

function packetSummary(file) {
  return {
    resourceId: file.worker.id,
    resourceType: "worker",
    status: file.documents.length ? "Complete" : "Incomplete",
    totalRules: 0,
    satisfiedRules: 0,
    missingRequired: 0,
    expiringSoon: 0,
    expired: 0,
    needsReview: 0,
    items: [],
  };
}

/** The permission manifest for the variant: the dispatcher lacks two kinds, the core plan two features. */
function manifest(state) {
  const base = permissionManifest();
  const variant = workerVariant(state);
  const permissions = { ...base.permissions };
  if (variant.access === "dispatcher")
    for (const resource of DISPATCHER_WITHHELD) permissions[resource] = 0;
  const unavailableFeatures =
    variant.plan === "core" ? [UNAVAILABLE.compliance, UNAVAILABLE.timeOff] : [];
  return {
    ...base,
    permissions,
    unavailableFeatures,
    checksum: `mock-${variant.access}-${variant.plan}`,
  };
}

/**
 * The worker page's operations, wrapping the handlers they share with other screens (`base`):
 * a worker id the files do not hold falls through to the board's and the Desk's own answers.
 */
export function workerHandlers(base) {
  const fallback = (name, state, v) => (base[name] ? base[name](state, v) : null);
  return {
    // ---- Roster, record, overview ----
    WorkerTable: (state, v) => ({
      workers: page(
        roster(state, v.input, v.withoutCredential),
        v.input,
        v.includeTotalCount !== false,
      ),
    }),
    WorkerRosterAttention: (state) => ({ workerRosterAttention: rosterAttention(state) }),
    WorkerRecord: (state, v) => ({
      worker: isWorkerId(state, v.id) ? workerOut(state, fileOf(state, v.id)) : null,
    }),
    WorkerOverview: (state, v) => ({
      workerOverview: overviewOut(state, fileOf(state, v.workerId)),
    }),
    PatchWorker: (state, v) => {
      const file = fileOf(state, v.id);
      Object.assign(
        file.worker,
        Object.fromEntries(Object.entries(v.input).filter(([, value]) => value != null)),
      );
      file.worker.version += 1;
      touch(file);
      return { patchWorker: workerOut(state, file) };
    },
    WorkerActivity: (state, v) => ({ auditEntriesBySubject: activityFor(state, v) }),
    CustomFieldFormFields: (_state, v) => ({
      customFieldFormFields: v.resourceType === "worker" ? WORKER_CUSTOM_FIELDS : [],
    }),
    CustomFieldResourceTypes: () => ({
      customFieldResourceTypes: ["worker", "customer", "shipment", "tractor", "trailer"],
    }),

    // ---- Credentials ----
    WorkerCredentials: (state, v) => {
      requireRead(state, "worker_credential");
      const file = fileOf(state, v.workerId);
      return {
        workerCredentials: file.credentials
          .filter((cred) => v.includeArchived || cred.status === "Active")
          .map(credentialOut),
      };
    },
    WorkerCredentialSummary: (state, v) => {
      requireRead(state, "worker_credential");
      return { workerCredentialSummary: credentialSummary(fileOf(state, v.workerId)) };
    },
    WorkerCredentialTypeChoices: () => ({
      activeWorkerCredentialTypes: CREDENTIAL_TYPE_LIST.map((t) => ({
        id: t.id,
        code: t.code,
        name: t.name,
        isRequired: t.isRequired,
      })),
    }),
    CreateWorkerCredential: (state, { input }) => {
      const file = fileOf(state, input.workerId);
      const type = credentialTypeOf(input.credentialTypeId);
      if (!type) refuse("Credential type not found");
      if (type.requiresNumber && !input.number?.trim())
        refuse("Number is required for this credential.");
      const current = file.credentials.find(
        (c) => c.status === "Active" && c.credentialTypeId === type.id,
      );
      if (current && !input.renew) refuse(`${type.name} is already on file; renew it instead.`);
      if (current)
        bump(current, {
          status: "Archived",
          archivedAt: NOW,
          archivedById: PEOPLE.dana.id,
          archiveReason: "Renewed",
        });
      const key = Object.keys(CREDENTIAL_TYPES).find((k) => CREDENTIAL_TYPES[k].id === type.id);
      const cred = credentialRecord(file.worker.id, 0, key, {
        number: input.number ?? null,
        issuingAuthority: input.issuingAuthority ?? null,
        issuedAt: input.issuedAt ?? null,
        expiresAt: input.expiresAt ?? null,
        notes: input.notes ?? null,
        document: input.documentId ? documentRefOf(file, input.documentId) : null,
        createdAt: NOW,
      });
      cred.id = nextId("wcr");
      cred.createdAt = NOW;
      cred.updatedAt = NOW;
      file.credentials.push(cred);
      credentialProfileSync(file, type, cred);
      audit(file, {
        resource: "worker_credential",
        resourceId: cred.id,
        operation: "create",
        entityRef: type.name,
      });
      touch(file, "worker_credential", cred.id);
      return { createWorkerCredential: credentialOut(cred) };
    },
    UpdateWorkerCredential: (state, { input }) => {
      const { file, row } = fileHolding(state, (f) => f.credentials, input.id);
      const { id: _id, version: _version, documentId, ...patch } = input;
      bump(row, {
        ...patch,
        ...(documentId !== undefined
          ? { documentId, document: documentId ? documentRefOf(file, documentId) : null }
          : {}),
      });
      credentialProfileSync(file, credentialTypeOf(row.credentialTypeId), row);
      audit(file, {
        resource: "worker_credential",
        resourceId: row.id,
        operation: "update",
        entityRef: credentialTypeOf(row.credentialTypeId).name,
      });
      touch(file, "worker_credential", row.id);
      return { updateWorkerCredential: credentialOut(row) };
    },
    ArchiveWorkerCredential: (state, { input }) => {
      const { file, row } = fileHolding(state, (f) => f.credentials, input.id);
      bump(row, {
        status: "Archived",
        archivedAt: NOW,
        archivedById: PEOPLE.dana.id,
        archiveReason: input.reason ?? null,
      });
      audit(file, {
        resource: "worker_credential",
        resourceId: row.id,
        operation: "archive",
        entityRef: credentialTypeOf(row.credentialTypeId).name,
      });
      touch(file, "worker_credential", row.id);
      return { archiveWorkerCredential: credentialOut(row) };
    },
    VerifyWorkerCredential: (state, v) => {
      const { file, row } = fileHolding(state, (f) => f.credentials, v.id);
      bump(row, { verifiedById: PEOPLE.dana.id, verifiedBy: DANA(), verifiedAt: NOW });
      audit(file, {
        resource: "worker_credential",
        resourceId: row.id,
        operation: "update",
        entityRef: credentialTypeOf(row.credentialTypeId).name,
        comment: "Verified",
      });
      touch(file, "worker_credential", row.id);
      return { verifyWorkerCredential: credentialOut(row) };
    },
    AttachWorkerCredentialDocument: (state, { input }) => {
      const { file, row } = fileHolding(state, (f) => f.credentials, input.id);
      bump(row, { documentId: input.documentId, document: documentRefOf(file, input.documentId) });
      touch(file, "worker_credential", row.id);
      return { attachWorkerCredentialDocument: credentialOut(row) };
    },

    // ---- DQ file ----
    DriverQualificationFile: (state, v) => {
      requireRead(state, "qualification");
      return { driverQualificationFile: dqfFile(fileOf(state, v.workerId)) };
    },
    RecordEmploymentVerification: (state, { input }) => {
      const file = fileOf(state, input.workerId);
      if (!input.employerName?.trim()) refuse("Employer name is required.");
      const row = {
        id: nextId("wev"),
        workerId: file.worker.id,
        employerName: input.employerName.trim(),
        employerDotNumber: input.employerDotNumber ?? null,
        employerMcNumber: input.employerMcNumber ?? null,
        contactName: input.contactName ?? null,
        contactPhone: input.contactPhone ?? null,
        contactEmail: input.contactEmail ?? null,
        employedFrom: input.employedFrom ?? null,
        employedTo: input.employedTo ?? null,
        wasDotRegulated: input.wasDotRegulated ?? true,
        status: input.status ?? (input.requestedAt ? "Requested" : "Pending"),
        method: input.method ?? "Email",
        requestedAt: input.requestedAt ?? null,
        responseReceivedAt: null,
        lastFollowUpAt: null,
        followUpCount: 0,
        drugAlcoholResponseReceivedAt: null,
        hadAccidents: false,
        accidentCount: 0,
        hadDrugAlcoholViolations: false,
        findings: null,
        notes: input.notes ?? null,
        documentId: null,
        version: 1,
      };
      file.dqf.verifications.push(row);
      audit(file, {
        resource: "qualification",
        resourceId: row.id,
        operation: "create",
        entityRef: row.employerName,
      });
      touch(file, "qualification", row.id);
      return { recordEmploymentVerification: row };
    },
    UpdateEmploymentVerification: (state, { input }) => {
      const { file, row } = fileHolding(state, (f) => f.dqf.verifications, input.verificationId);
      const { verificationId: _id, ...patch } = input;
      bump(
        row,
        Object.fromEntries(Object.entries(patch).filter(([, value]) => value !== undefined)),
      );
      touch(file, "qualification", row.id);
      return { updateEmploymentVerification: row };
    },
    DeleteEmploymentVerification: (state, v) => {
      const { file, rows, row } = fileHolding(state, (f) => f.dqf.verifications, v.id);
      rows.splice(rows.indexOf(row), 1);
      touch(file, "qualification", v.id);
      return { deleteEmploymentVerification: true };
    },
    MarkEmploymentVerificationRequested: (state, v) => {
      const { file, row } = fileHolding(state, (f) => f.dqf.verifications, v.id);
      bump(row, { status: "Requested", requestedAt: NOW });
      touch(file, "qualification", row.id);
      return { markEmploymentVerificationRequested: row };
    },
    RecordEmploymentVerificationFollowUp: (state, v) => {
      const { file, row } = fileHolding(state, (f) => f.dqf.verifications, v.id);
      bump(row, { followUpCount: row.followUpCount + 1, lastFollowUpAt: NOW });
      audit(file, {
        resource: "qualification",
        resourceId: row.id,
        operation: "update",
        entityRef: row.employerName,
        comment: "Chased the safety history request",
      });
      touch(file, "qualification", row.id);
      return { recordEmploymentVerificationFollowUp: row };
    },

    // ---- Drug and alcohol ----
    WorkerDrugAlcoholFile: (state, v) => {
      requireRead(state, "worker_dot_test");
      const file = fileOf(state, v.workerId);
      const da = file.drugAlcohol;
      return {
        workerDrugAlcoholStanding: drugAlcoholStanding(file),
        workerDotTests: [...da.tests].sort(byNewest("collectedAt")),
        workerDotViolations: da.violations,
        workerClearinghouseQueries: [...da.queries].sort(byNewest("requestedAt")),
        workerRandomSelections: da.randomSelections.map((row) => ({
          ...row,
          worker: {
            id: file.worker.id,
            firstName: file.worker.firstName,
            lastName: file.worker.lastName,
          },
        })),
      };
    },
    RecordDotTest: (state, { input }) => {
      const file = fileOf(state, input.workerId);
      const row = {
        id: nextId("wdt"),
        workerId: file.worker.id,
        testType: input.testType,
        substance: input.substance,
        status:
          input.status ??
          (input.result && input.result !== "Pending"
            ? "Completed"
            : input.collectedAt
              ? "AwaitingResult"
              : "Scheduled"),
        result: input.result ?? "Pending",
        isDot: input.isDot ?? true,
        reason: input.reason ?? null,
        scheduledAt: input.scheduledAt ?? null,
        collectedAt: input.collectedAt ?? null,
        resultAt: input.resultAt ?? null,
        collectionSite: input.collectionSite ?? null,
        collectorName: input.collectorName ?? null,
        specimenId: input.specimenId ?? null,
        labName: input.labName ?? null,
        mroName: input.mroName ?? null,
        mroVerifiedAt: input.mroVerifiedAt ?? null,
        alcoholConcentration: input.alcoholConcentration ?? null,
        safetyEventId: input.safetyEventId ?? null,
        drawEntryId: input.drawEntryId ?? null,
        documentId: input.documentId ?? null,
        notes: input.notes ?? null,
        version: 1,
        createdAt: NOW,
        updatedAt: NOW,
      };
      file.drugAlcohol.tests.push(row);
      audit(file, {
        resource: "worker_dot_test",
        resourceId: row.id,
        operation: "create",
        entityRef: `${row.testType} ${row.substance.toLowerCase()} test`,
      });
      touch(file, "worker_dot_test", row.id);
      return { recordDotTest: row };
    },
    RecordDotTestResult: (state, { input }) => {
      const { file, row } = fileHolding(state, (f) => f.drugAlcohol.tests, input.testId);
      const { testId: _id, ...patch } = input;
      bump(row, { ...patch, status: "Completed", resultAt: input.resultAt ?? NOW });
      touch(file, "worker_dot_test", row.id);
      return { recordDotTestResult: row };
    },
    CancelDotTest: (state, v) => {
      if (!v.reason?.trim()) refuse("A reason is required.");
      const { file, row } = fileHolding(state, (f) => f.drugAlcohol.tests, v.id);
      bump(row, { status: "Cancelled", result: "Cancelled", notes: v.reason });
      touch(file, "worker_dot_test", row.id);
      return { cancelDotTest: row };
    },
    UpdateDotViolation: (state, { input }) => {
      const { file, row } = fileHolding(state, (f) => f.drugAlcohol.violations, input.violationId);
      const { violationId: _id, ...patch } = input;
      bump(row, patch);
      touch(file, "worker_dot_test", row.id);
      return { updateDotViolation: row };
    },
    RecordClearinghouseQuery: (state, { input }) => {
      const file = fileOf(state, input.workerId);
      const requestedAt = input.requestedAt ?? NOW;
      const row = {
        id: nextId("wcq"),
        workerId: file.worker.id,
        queryType: input.queryType,
        result: input.result ?? "Pending",
        consentObtainedAt: input.consentObtainedAt ?? null,
        consentExpiresAt: input.consentExpiresAt ?? null,
        requestedAt,
        completedAt:
          input.result && input.result !== "Pending" ? (input.completedAt ?? requestedAt) : null,
        violationCount: input.violationCount ?? 0,
        reference: input.reference ?? null,
        documentId: input.documentId ?? null,
        notes: input.notes ?? null,
        version: 1,
        createdAt: NOW,
        updatedAt: NOW,
      };
      file.drugAlcohol.queries.push(row);
      if (row.completedAt) settleChecklistItem(file, "Clearinghouse annual query");
      audit(file, {
        resource: "worker_dot_test",
        resourceId: row.id,
        operation: "create",
        entityRef: "Clearinghouse query",
      });
      touch(file, "worker_dot_test", row.id);
      return { recordClearinghouseQuery: row };
    },
    CompleteClearinghouseQuery: (state, { input }) => {
      const { file, row } = fileHolding(state, (f) => f.drugAlcohol.queries, input.queryId);
      bump(row, {
        result: input.result,
        completedAt: input.completedAt ?? NOW,
        reference: input.reference ?? row.reference,
        violationCount: input.violationCount ?? 0,
        notes: input.notes ?? row.notes,
      });
      settleChecklistItem(file, "Clearinghouse annual query");
      touch(file, "worker_dot_test", row.id);
      return { completeClearinghouseQuery: row };
    },

    // ---- Training ----
    WorkerTrainingRecords: (state, v) => ({
      workerTrainingRecords: connection(
        fileOf(state, v.workerId)
          .training.filter(
            (row) => v.includeClosed || row.status === "Assigned" || row.status === "InProgress",
          )
          .map(trainingOut)
          .sort(byNewest("assignedAt")),
        { first: v.first, after: v.after },
      ),
    }),
    WorkerTrainingSummary: (state, v) => ({
      workerTrainingSummary: trainingSummary(fileOf(state, v.workerId)),
    }),
    ActiveTrainingCourses: (state) => ({
      activeTrainingCourses: COURSE_LIST.map((c) => ({
        ...c,
        openRecordCount: [...ws(state).files.values()].reduce(
          (n, file) =>
            n +
            file.training.filter(
              (r) => r.courseId === c.id && (r.status === "Assigned" || r.status === "InProgress"),
            ).length,
          0,
        ),
      })),
    }),
    AssignWorkerTraining: (state, { input }) => {
      const file = fileOf(state, input.workerId);
      const { record, skipped } = assignTraining(file, input.courseId, input.dueAt, input.notes);
      if (skipped) refuse("This course is already assigned and open.");
      touch(file, "worker_training", record.id);
      return { assignWorkerTraining: trainingOut(record) };
    },
    AssignRequiredWorkerTraining: (state, v) => {
      const file = fileOf(state, v.workerId);
      const summary = trainingSummary(file);
      const assigned = [];
      for (const item of summary.items) {
        if (
          !item.required ||
          ["Current", "Scheduled", "DueSoon", "ExpiringSoon"].includes(item.health)
        )
          continue;
        const { record, skipped } = assignTraining(file, item.course.id, null, null);
        if (!skipped) assigned.push(trainingOut(record));
      }
      touch(file, "worker_training");
      return { assignRequiredWorkerTraining: assigned };
    },
    CompleteWorkerTraining: (state, { input }) => {
      let file;
      let row;
      if (input.id) ({ file, row } = trainingWrite(state, input.id));
      else {
        file = fileOf(state, input.workerId);
        row = assignTraining(file, input.courseId, null, null).record;
      }
      const course = courseOf(row.courseId);
      const completedAt = input.completedAt ?? NOW;
      const score = input.score != null ? String(input.score) : null;
      const passed =
        score == null || !course.passingScore || Number(score) >= Number(course.passingScore);
      bump(row, {
        status: passed ? "Completed" : "Failed",
        completedAt,
        expiresAt:
          passed && course.validityMonths
            ? completedAt + Math.round(course.validityMonths * 30.44) * DAY
            : null,
        score,
        passed,
        notes: input.notes ?? row.notes,
        documentId: input.documentId ?? row.documentId,
        document: input.documentId ? documentRefOf(file, input.documentId) : row.document,
        recordedById: PEOPLE.dana.id,
        recordedBy: DANA(),
      });
      file.trainingSlots[row.courseId] = row.id;
      audit(file, {
        resource: "worker_training",
        resourceId: row.id,
        operation: "update",
        entityRef: course.name,
        changes: {
          status: { from: "Assigned", to: row.status, type: "updated", fieldType: "string" },
        },
      });
      touch(file, "worker_training", row.id);
      return { completeWorkerTraining: trainingOut(row) };
    },
    WaiveWorkerTraining: (state, { input }) => {
      if (!input.reason?.trim()) refuse("A reason is required.");
      const { file, row } = trainingWrite(state, input.id);
      bump(row, { status: "Waived", waivedReason: input.reason });
      touch(file, "worker_training", row.id);
      return { waiveWorkerTraining: trainingOut(row) };
    },
    CancelWorkerTraining: (state, { input }) => {
      const { file, row } = trainingWrite(state, input.id);
      bump(row, { status: "Cancelled", notes: input.reason ?? row.notes });
      if (file.trainingSlots[row.courseId] === row.id) file.trainingSlots[row.courseId] = null;
      touch(file, "worker_training", row.id);
      return { cancelWorkerTraining: trainingOut(row) };
    },
    AttachWorkerTrainingDocument: (state, { input }) => {
      const { file, row } = trainingWrite(state, input.id);
      bump(row, { documentId: input.documentId, document: documentRefOf(file, input.documentId) });
      touch(file, "worker_training", row.id);
      return { attachWorkerTrainingDocument: trainingOut(row) };
    },
    BulkAssignTraining: (state, { input }) => {
      const outcomes = [];
      for (const workerId of input.workerIds) {
        for (const courseId of input.courseIds) {
          if (!isWorkerId(state, workerId)) {
            outcomes.push({
              workerId,
              courseId,
              recordId: null,
              skipped: false,
              error: "Worker not found",
            });
            continue;
          }
          const file = fileOf(state, workerId);
          const { record, skipped } = assignTraining(file, courseId, input.dueAt, input.notes);
          outcomes.push({ workerId, courseId, recordId: record.id, skipped, error: "" });
        }
      }
      return {
        bulkAssignTraining: {
          assignedCount: outcomes.filter((o) => !o.skipped && !o.error).length,
          skippedCount: outcomes.filter((o) => o.skipped).length,
          failedCount: outcomes.filter((o) => o.error).length,
          outcomes,
        },
      };
    },

    // ---- Reviews ----
    WorkerPerformanceReviews: (state, v) => ({
      workerPerformanceReviews: [...fileOf(state, v.workerId).reviews]
        .sort(byNewest("periodEnd"))
        .map(reviewOut),
    }),
    ActivePerformanceReviewTemplates: () => ({
      activePerformanceReviewTemplates: Object.values(REVIEW_TEMPLATES),
    }),
    CreatePerformanceReview: (state, { input }) => {
      const file = fileOf(state, input.workerId);
      if (file.reviews.some((r) => r.status === "Draft" || r.status === "Submitted"))
        refuse("A review is already under way for this worker.");
      const template = Object.values(REVIEW_TEMPLATES).find((t) => t.id === input.templateId);
      if (!template) refuse("Review template not found");
      const review = {
        id: nextId("prv"),
        ...TENANT,
        workerId: file.worker.id,
        templateId: template.id,
        template: {
          id: template.id,
          code: template.code,
          name: template.name,
          cadenceMonths: template.cadenceMonths,
          items: template.items,
        },
        reviewerId: PEOPLE.dana.id,
        reviewer: DANA(),
        status: "Draft",
        title:
          input.title ??
          `${new Date(input.periodEnd * 1000).getUTCFullYear()} ${template.name.toLowerCase()} review`,
        periodStart: input.periodStart,
        periodEnd: input.periodEnd,
        ratings: template.items.map((item) => ({
          key: item.key,
          label: item.label,
          weight: item.weight,
          score: null,
          comment: null,
        })),
        overallScore: null,
        summary: null,
        strengths: null,
        improvements: null,
        goals: [],
        submittedAt: null,
        acknowledgedAt: null,
        workerComment: null,
        closedAt: null,
        closedById: null,
        nextReviewAt: null,
        version: 1,
        createdAt: NOW,
        updatedAt: NOW,
      };
      file.reviews.push(review);
      audit(file, {
        resource: "performance_review",
        resourceId: review.id,
        operation: "create",
        entityRef: review.title,
      });
      touch(file, "performance_review", review.id);
      return { createPerformanceReview: reviewOut(review) };
    },
    UpdatePerformanceReview: (state, { input }) => {
      const { file, row } = fileHolding(state, (f) => f.reviews, input.id);
      const ratings = row.ratings.map((rating) => {
        const next = input.ratings.find((r) => r.key === rating.key);
        return next
          ? { ...rating, score: next.score ?? null, comment: next.comment ?? null }
          : rating;
      });
      bump(row, {
        ratings,
        overallScore: overallScore(ratings),
        ...(input.title !== undefined ? { title: input.title } : {}),
        ...(input.summary !== undefined ? { summary: input.summary } : {}),
        ...(input.strengths !== undefined ? { strengths: input.strengths } : {}),
        ...(input.improvements !== undefined ? { improvements: input.improvements } : {}),
        ...(input.periodStart ? { periodStart: input.periodStart } : {}),
        ...(input.periodEnd ? { periodEnd: input.periodEnd } : {}),
        ...(input.goals
          ? {
              goals: input.goals.map((g, i) => ({
                id: g.id ?? `goal-${Date.now()}-${i}`,
                title: g.title,
                dueAt: g.dueAt ?? null,
                status: g.status ?? "Open",
              })),
            }
          : {}),
      });
      touch(file, "performance_review", row.id);
      return { updatePerformanceReview: reviewOut(row) };
    },
    SubmitPerformanceReview: (state, { input }) =>
      reviewStatus(state, input.id, "submitPerformanceReview", () => ({
        status: "Submitted",
        submittedAt: NOW,
      })),
    ClosePerformanceReview: (state, { input }) =>
      reviewStatus(state, input.id, "closePerformanceReview", (row) => ({
        status: "Closed",
        closedAt: NOW,
        closedById: PEOPLE.dana.id,
        nextReviewAt: row.template?.cadenceMonths
          ? row.periodEnd + row.template.cadenceMonths * 30 * DAY
          : null,
      })),
    ReopenPerformanceReview: (state, { input }) =>
      reviewStatus(state, input.id, "reopenPerformanceReview", () => ({
        status: "Draft",
        submittedAt: null,
        closedAt: null,
        closedById: null,
      })),
    DeletePerformanceReview: (state, v) => {
      const { file, rows, row } = fileHolding(state, (f) => f.reviews, v.id);
      if (row.status !== "Draft") refuse("Only a draft review can be deleted.");
      rows.splice(rows.indexOf(row), 1);
      touch(file, "performance_review", v.id);
      return { deletePerformanceReview: true };
    },

    // ---- Safety ----
    WorkerSafetyScorecard: (state, v) => ({
      workerSafetyScorecard: safetyScorecard(fileOf(state, v.workerId)),
    }),
    WorkerSafetyEvents: (state, v) => ({
      workerSafetyEvents: [...fileOf(state, v.workerId).safety.events]
        .sort(byNewest("occurredAt"))
        .map(safetyEventOut),
    }),
    WorkerDisciplinaryActions: (state, v) => ({
      workerDisciplinaryActions: connection(
        [...fileOf(state, v.workerId).safety.actions]
          .sort(byNewest("issuedAt"))
          .map(disciplinaryOut),
        { first: v.first, after: v.after },
      ),
    }),
    WorkerSafetyEvent: (state, v) => ({
      workerSafetyEvent: safetyEventOut(fileHolding(state, (f) => f.safety.events, v.id).row),
    }),
    WorkerDisciplinaryAction: (state, v) => ({
      workerDisciplinaryAction: disciplinaryOut(
        fileHolding(state, (f) => f.safety.actions, v.id).row,
      ),
    }),
    WorkerInjury: (state, v) => ({
      workerInjury: fileHolding(state, (f) => f.safety.injuries, v.id).row,
    }),
    WorkerTrainingRecord: (state, v) => ({
      workerTrainingRecord: trainingOut(fileHolding(state, (f) => f.training, v.id).row),
    }),
    WorkerEmploymentEvent: (state, v) => ({
      workerEmploymentEvent: fileHolding(state, (f) => f.employmentEvents, v.id).row,
    }),
    WorkerPto: (state, v) => ({
      workerPTO: fileHolding(state, (f) => f.pto.requests, v.id).row,
    }),
    WorkerDisciplinaryLadder: (state, v) => ({
      workerDisciplinaryLadder: disciplinaryLadder(fileOf(state, v.workerId)),
    }),
    WorkerRecognitions: (state, v) => ({
      workerRecognitions: [...fileOf(state, v.workerId).safety.recognitions].sort(
        byNewest("occurredAt"),
      ),
    }),
    WorkerInjuries: (state, v) => ({
      workerInjuries: connection(fileOf(state, v.workerId).safety.injuries, {
        first: v.first,
        after: v.after,
      }),
    }),
    DefaultSafetyPoints: (_state, v) => ({ defaultSafetyPoints: defaultPoints(v) }),
    CreateWorkerSafetyEvent: (state, { input }) => {
      const file = fileOf(state, input.workerId);
      const event = newSafetyEvent(file, input);
      file.safety.events.push(event);
      if (event.kind === "Inspection") {
        file.safety.scorecard.inspections += 1;
        if (event.inspectionResult === "Pass") file.safety.scorecard.inspectionsPassed += 1;
        else file.safety.scorecard.inspectionsFailed += 1;
      }
      audit(file, {
        resource: "worker_safety_event",
        resourceId: event.id,
        operation: "create",
        entityRef: event.kind,
      });
      touch(file, "worker_safety_event", event.id);
      return { createWorkerSafetyEvent: safetyEventOut(event) };
    },
    UpdateWorkerSafetyEvent: (state, { input }) => {
      const { file, row } = fileHolding(state, (f) => f.safety.events, input.id);
      const { id: _id, version: _version, documentId, ...patch } = input;
      bump(row, {
        ...patch,
        ...(documentId !== undefined
          ? { documentId, document: documentId ? documentRefOf(file, documentId) : null }
          : {}),
      });
      touch(file, "worker_safety_event", row.id);
      return { updateWorkerSafetyEvent: safetyEventOut(row) };
    },
    CloseWorkerSafetyEvent: (state, { input }) =>
      safetyStatus(state, input, "closeWorkerSafetyEvent", {
        status: "Closed",
        closedAt: NOW,
        closedById: PEOPLE.dana.id,
        closedBy: DANA(),
        ...(input.resolution ? { resolution: input.resolution } : {}),
      }),
    ReopenWorkerSafetyEvent: (state, { input }) =>
      safetyStatus(state, input, "reopenWorkerSafetyEvent", {
        status: "Open",
        closedAt: null,
        closedById: null,
        closedBy: null,
      }),
    ReviewWorkerSafetyEvent: (state, { input }) =>
      safetyStatus(state, input, "reviewWorkerSafetyEvent", { status: "UnderReview" }),
    DeleteWorkerSafetyEvent: (state, v) => {
      const { file, rows, row } = fileHolding(state, (f) => f.safety.events, v.id);
      rows.splice(rows.indexOf(row), 1);
      audit(file, {
        resource: "worker_safety_event",
        resourceId: v.id,
        operation: "delete",
        entityRef: row.kind,
      });
      touch(file, "worker_safety_event", v.id);
      return { deleteWorkerSafetyEvent: true };
    },
    IssueDisciplinaryAction: (state, { input }) => {
      const file = fileOf(state, input.workerId);
      if (!input.reason?.trim()) refuse("Reason is required.");
      const safetyEvent = input.safetyEventId
        ? file.safety.events.find((e) => e.id === input.safetyEventId)
        : null;
      const action = {
        id: nextId("wda"),
        ...TENANT,
        workerId: file.worker.id,
        level: input.level,
        status: "Active",
        reason: input.reason,
        details: input.details ?? null,
        occurredAt: input.occurredAt ?? null,
        issuedAt: NOW,
        expiresAt: input.expiresAt ?? NOW + 365 * DAY,
        suspensionDays: input.suspensionDays ?? null,
        safetyEventId: safetyEvent?.id ?? null,
        safetyEvent: safetyEvent
          ? { id: safetyEvent.id, kind: safetyEvent.kind, occurredAt: safetyEvent.occurredAt }
          : null,
        documentId: input.documentId ?? null,
        issuedById: PEOPLE.dana.id,
        issuedBy: DANA(),
        acknowledgedAt: null,
        workerComment: null,
        rescindedAt: null,
        rescindedById: null,
        rescindReason: null,
        version: 1,
        createdAt: NOW,
        updatedAt: NOW,
      };
      file.safety.actions.push(action);
      let employment = null;
      if (
        input.recordEmploymentEvent &&
        (input.level === "Suspension" || input.level === "Termination")
      ) {
        const kind = input.level === "Suspension" ? "Suspended" : "Terminated";
        const eventInput = {
          workerId: file.worker.id,
          kind,
          effectiveAt: NOW,
          reason: input.reason,
        };
        employment = { id: applyEvent(state, file, eventInput, planEvent(file, eventInput)).id };
      }
      audit(file, {
        resource: "worker_disciplinary_action",
        resourceId: action.id,
        operation: "create",
        entityRef: input.level,
      });
      touch(file, "worker_disciplinary_action", action.id);
      return {
        issueDisciplinaryAction: { action: disciplinaryOut(action), employmentEvent: employment },
      };
    },
    RescindDisciplinaryAction: (state, { input }) => {
      if (!input.reason?.trim()) refuse("Reason is required.");
      const { file, row } = fileHolding(state, (f) => f.safety.actions, input.id);
      bump(row, {
        status: "Rescinded",
        rescindedAt: NOW,
        rescindedById: PEOPLE.dana.id,
        rescindReason: input.reason,
      });
      touch(file, "worker_disciplinary_action", row.id);
      return { rescindDisciplinaryAction: disciplinaryOut(row) };
    },
    GiveWorkerRecognition: (state, { input }) => {
      const file = fileOf(state, input.workerId);
      const row = {
        id: nextId("wrc"),
        ...TENANT,
        workerId: file.worker.id,
        kind: input.kind,
        title: input.title,
        message: input.message ?? null,
        occurredAt: input.occurredAt ?? NOW,
        awardedById: PEOPLE.dana.id,
        awardedBy: DANA(),
        visibleToWorker: input.visibleToWorker ?? false,
        version: 1,
        createdAt: NOW,
        updatedAt: NOW,
      };
      file.safety.recognitions.push(row);
      audit(file, {
        resource: "worker_recognition",
        resourceId: row.id,
        operation: "create",
        entityRef: row.title,
      });
      touch(file, "worker_recognition", row.id);
      return { giveWorkerRecognition: row };
    },
    DeleteWorkerRecognition: (state, v) => {
      const { file, rows, row } = fileHolding(state, (f) => f.safety.recognitions, v.id);
      rows.splice(rows.indexOf(row), 1);
      touch(file, "worker_recognition", v.id);
      return { deleteWorkerRecognition: true };
    },
    RecordSafetyViolation: (state, { input }) => {
      const { file, row: event } = fileHolding(state, (f) => f.safety.events, input.safetyEventId);
      const violation = {
        id: nextId("wsv"),
        safetyEventId: event.id,
        workerId: file.worker.id,
        basic: input.basic ?? "UnsafeDriving",
        code: input.code ?? null,
        description: input.description,
        severityWeight: input.severityWeight ?? 1,
        outOfService: input.outOfService ?? false,
        version: 1,
      };
      event.violations.push(violation);
      touch(file, "worker_safety_event", event.id);
      return { recordSafetyViolation: violation };
    },
    UpdateSafetyViolation: (state, { input }) => {
      for (const file of ws(state).files.values()) {
        for (const event of file.safety.events) {
          const violation = event.violations.find((row) => row.id === input.id);
          if (!violation) continue;
          const { id: _id, version: _version, ...patch } = input;
          bump(violation, patch);
          touch(file, "worker_safety_event", event.id);
          return { updateSafetyViolation: violation };
        }
      }
      refuse("Violation not found");
    },
    DeleteSafetyViolation: (state, v) => {
      for (const file of ws(state).files.values()) {
        for (const event of file.safety.events) {
          const index = event.violations.findIndex((row) => row.id === v.id);
          if (index === -1) continue;
          event.violations.splice(index, 1);
          touch(file, "worker_safety_event", event.id);
          return { deleteSafetyViolation: true };
        }
      }
      refuse("Violation not found");
    },
    RecordWorkerInjury: (state, { input }) => {
      const file = fileOf(state, input.workerId);
      const year = new Date(input.occurredAt * 1000).getUTCFullYear();
      const caseNumber =
        [...ws(state).files.values()].reduce(
          (n, f) => n + f.safety.injuries.filter((i) => i.caseYear === year).length,
          0,
        ) + 1;
      const classification = input.classification ?? "FirstAidOnly";
      const row = {
        id: nextId("win"),
        workerId: file.worker.id,
        caseNumber,
        caseYear: year,
        classification,
        illnessType: input.illnessType ?? "Injury",
        treatment: input.treatment ?? "FirstAid",
        status: "Open",
        recordable: !["FirstAidOnly", "NotRecordable"].includes(classification),
        occurredAt: input.occurredAt,
        reportedAt: input.reportedAt ?? NOW,
        returnedToWorkAt: input.returnedToWorkAt ?? null,
        location: input.location ?? null,
        description: input.description,
        bodyPart: input.bodyPart ?? null,
        harmfulAgent: input.harmfulAgent ?? null,
        daysAway: input.daysAway ?? 0,
        daysRestricted: input.daysRestricted ?? 0,
        privacyCase: input.privacyCase ?? false,
        logName: input.privacyCase ? "Privacy case" : file.worker.wholeName,
        claimStatus: input.claimStatus ?? "NotFiled",
        claimNumber: input.claimNumber ?? null,
        claimCarrier: input.claimCarrier ?? null,
        claimFiledAt: input.claimFiledAt ?? null,
        claimClosedAt: null,
        safetyEventId: input.safetyEventId ?? null,
        documentId: input.documentId ?? null,
        notes: input.notes ?? null,
        version: 1,
      };
      file.safety.injuries.push(row);
      audit(file, {
        resource: "worker_injury",
        resourceId: row.id,
        operation: "create",
        entityRef: `Case ${year}-${caseNumber}`,
      });
      touch(file, "worker_injury", row.id);
      return { recordWorkerInjury: row };
    },
    UpdateWorkerInjury: (state, { input }) => {
      const { file, row } = fileHolding(state, (f) => f.safety.injuries, input.injuryId);
      const { injuryId: _id, ...patch } = input;
      bump(
        row,
        Object.fromEntries(Object.entries(patch).filter(([, value]) => value !== undefined)),
      );
      row.recordable = !["FirstAidOnly", "NotRecordable"].includes(row.classification);
      touch(file, "worker_injury", row.id);
      return { updateWorkerInjury: row };
    },
    DeleteWorkerInjury: (state, v) => {
      const { file, rows, row } = fileHolding(state, (f) => f.safety.injuries, v.id);
      rows.splice(rows.indexOf(row), 1);
      touch(file, "worker_injury", v.id);
      return { deleteWorkerInjury: true };
    },

    // ---- Employment ----
    WorkerEmploymentEvents: (state, v) => ({
      workerEmploymentEvents: connection(
        [...fileOf(state, v.workerId).employmentEvents]
          .filter((event) => !v.kinds?.length || v.kinds.includes(event.kind))
          .sort(byNewest("effectiveAt")),
        { first: v.first, after: v.after },
      ),
    }),
    WorkerEmploymentEventOptions: (state, v) => ({
      workerEmploymentEventOptions: {
        recordableKinds: recordableKinds(fileOf(state, v.workerId)),
        reasonRequiredKinds: REASON_REQUIRED,
      },
    }),
    PreviewWorkerEmploymentEvent: (state, { input }) => {
      const file = fileOf(state, input.workerId);
      const plan = planEvent(file, input);
      return {
        previewWorkerEmploymentEvent: { kind: input.kind, effectiveAt: input.effectiveAt, ...plan },
      };
    },
    RecordWorkerEmploymentEvent: (state, { input }) => {
      const file = fileOf(state, input.workerId);
      validateEvent(file, input);
      const plan = planEvent(file, input);
      const event = applyEvent(state, file, input, plan);
      return { recordWorkerEmploymentEvent: { event, effects: plan.effects } };
    },
    AmendWorkerEmploymentEvent: (state, { input }) => {
      if (!input.amendmentNote?.trim()) refuse("Say what the amendment corrects.");
      const { file, row } = fileHolding(state, (f) => f.employmentEvents, input.id);
      bump(row, {
        ...(input.effectiveAt ? { effectiveAt: input.effectiveAt } : {}),
        ...(input.reason !== undefined ? { reason: input.reason } : {}),
        ...(input.notes !== undefined ? { notes: input.notes } : {}),
        amendedById: PEOPLE.dana.id,
        amendedBy: DANA(),
        amendedAt: NOW,
        amendmentNote: input.amendmentNote,
      });
      audit(file, {
        resource: "worker_employment_event",
        resourceId: row.id,
        operation: "update",
        entityRef: row.kind,
        comment: input.amendmentNote,
      });
      touch(file, "worker_employment_event", row.id);
      return { amendWorkerEmploymentEvent: row };
    },

    // ---- Checklists ----
    WorkerChecklists: (state, v) => ({
      workerChecklists: fileOf(state, v.workerId)
        .checklists.filter((row) => v.includeClosed || row.status === "Open")
        .sort(byNewest("startedAt"))
        .map(checklistOut),
    }),
    ActiveWorkerChecklistTemplates: () => ({
      activeWorkerChecklistTemplates: Object.values(CHECKLIST_TEMPLATES),
    }),
    StartWorkerChecklist: (state, { input }) => {
      const file = fileOf(state, input.workerId);
      const template = Object.values(CHECKLIST_TEMPLATES).find((t) => t.id === input.templateId);
      if (!template) refuse("Checklist template not found");
      if (file.checklists.some((row) => row.status === "Open" && row.templateId === template.id))
        refuse(`${template.name} is already open for this worker.`);
      const checklist = startChecklist(file, template, input.startedAt ?? NOW);
      file.checklists.unshift(checklist);
      audit(file, {
        resource: "worker_checklist",
        resourceId: checklist.id,
        operation: "create",
        entityRef: checklist.name,
      });
      touch(file, "worker_checklist", checklist.id);
      return { startWorkerChecklist: checklistOut(checklist) };
    },
    CancelWorkerChecklist: (state, { input }) => {
      if (!input.reason?.trim()) refuse("A reason is required to cancel a checklist.");
      const { file, row } = fileHolding(state, (f) => f.checklists, input.id);
      bump(row, { status: "Cancelled", cancelledAt: NOW, cancelReason: input.reason.trim() });
      audit(file, {
        resource: "worker_checklist",
        resourceId: row.id,
        operation: "cancel",
        entityRef: row.name,
        comment: input.reason.trim(),
      });
      touch(file, "worker_checklist", row.id);
      return { cancelWorkerChecklist: checklistOut(row) };
    },
    CompleteWorkerChecklistItem: (state, { input }) => ({
      completeWorkerChecklistItem: checklistItemAction(state, input, "Done"),
    }),
    SkipWorkerChecklistItem: (state, { input }) => ({
      skipWorkerChecklistItem: checklistItemAction(state, input, "Skipped"),
    }),
    MarkWorkerChecklistItemNotApplicable: (state, { input }) => ({
      markWorkerChecklistItemNotApplicable: checklistItemAction(state, input, "NotApplicable"),
    }),
    ReopenWorkerChecklistItem: (state, v) => ({
      reopenWorkerChecklistItem: checklistItemAction(state, { id: v.id }, "Pending"),
    }),

    // ---- Leave ----
    WorkerLeaveFile: (state, v) => {
      requireRead(state, "worker_leave");
      const file = fileOf(state, v.workerId);
      return {
        workerLeaveEntitlement: leaveEntitlement(file),
        workerLeaveCases: [...file.leave.cases].sort(byNewest("startsAt")),
      };
    },
    OpenLeaveCase: (state, { input }) => {
      const file = fileOf(state, input.workerId);
      const row = {
        id: nextId("wlc"),
        workerId: file.worker.id,
        leaveType: input.leaveType ?? "FMLA",
        status: "Pending",
        frequency: input.frequency ?? "Continuous",
        reason: input.reason ?? null,
        fmlaDesignated: false,
        militaryCaregiver: input.militaryCaregiver ?? false,
        requestedAt: input.requestedAt ?? NOW,
        startsAt: input.startsAt,
        endsAt: input.endsAt ?? null,
        decidedAt: null,
        closedAt: null,
        certificationStatus: "NotRequired",
        certificationRequestedAt: null,
        certificationDueAt: null,
        certificationReceivedAt: null,
        recertificationDueAt: null,
        certificationLate: false,
        eligibilityHoursWorked: input.eligibilityHoursWorked ?? null,
        notes: input.notes ?? null,
        version: 1,
        entries: [],
      };
      file.leave.cases.push(row);
      audit(file, {
        resource: "worker_leave",
        resourceId: row.id,
        operation: "create",
        entityRef: `${row.leaveType} leave`,
      });
      touch(file, "worker_leave", row.id);
      return { openLeaveCase: row };
    },
    UpdateLeaveCase: (state, { input }) => {
      const { file, row } = fileHolding(state, (f) => f.leave.cases, input.caseId);
      const { caseId: _id, documentId: _doc, ...patch } = input;
      bump(
        row,
        Object.fromEntries(Object.entries(patch).filter(([, value]) => value !== undefined)),
      );
      touch(file, "worker_leave", row.id);
      return { updateLeaveCase: row };
    },
    DecideLeaveCase: (state, { input }) => {
      const { file, row } = fileHolding(state, (f) => f.leave.cases, input.caseId);
      bump(row, {
        status: input.approve ? "Approved" : "Denied",
        fmlaDesignated: input.approve && input.designate,
        decidedAt: NOW,
        notes: input.notes ?? row.notes,
      });
      touch(file, "worker_leave", row.id);
      return { decideLeaveCase: row };
    },
    CloseLeaveCase: (state, v) => {
      const { file, row } = fileHolding(state, (f) => f.leave.cases, v.id);
      bump(row, { status: "Closed", closedAt: NOW });
      touch(file, "worker_leave", row.id);
      return { closeLeaveCase: row };
    },
    RecordLeaveDay: (state, { input }) => {
      const { file, row } = fileHolding(state, (f) => f.leave.cases, input.caseId);
      const entry = {
        id: nextId("wle"),
        leaveCaseId: row.id,
        usedOn: input.usedOn,
        hours: String(input.hours),
        countsAgainstEntitlement: row.fmlaDesignated,
        ptoId: input.ptoId ?? null,
        notes: input.notes ?? null,
        version: 1,
      };
      row.entries.push(entry);
      if (entry.countsAgainstEntitlement && input.usedOn >= file.leave.entitlement.window.from) {
        const used = Number(file.leave.entitlement.usedHours) + Number(input.hours);
        const total = Number(file.leave.entitlement.totalHours);
        Object.assign(file.leave.entitlement, {
          usedHours: String(used),
          remainingHours: String(total - used),
          usedWeeks: String(Math.round((used / 40) * 100) / 100),
          remainingWeeks: String(Math.round(((total - used) / 40) * 100) / 100),
          exhausted: used >= total,
        });
      }
      touch(file, "worker_leave", row.id);
      return { recordLeaveDay: entry };
    },
    DeleteLeaveDay: (state, v) => {
      for (const file of ws(state).files.values()) {
        for (const leaveCase of file.leave.cases) {
          const index = leaveCase.entries.findIndex((entry) => entry.id === v.id);
          if (index === -1) continue;
          leaveCase.entries.splice(index, 1);
          touch(file, "worker_leave", leaveCase.id);
          return { deleteLeaveDay: true };
        }
      }
      refuse("Leave day not found");
    },
    RequestLeaveCertification: (state, v) => {
      const { file, row } = fileHolding(state, (f) => f.leave.cases, v.caseId);
      bump(row, {
        certificationStatus: "Requested",
        certificationRequestedAt: NOW,
        certificationDueAt: v.dueAt ?? NOW + 15 * DAY,
      });
      touch(file, "worker_leave", row.id);
      return { requestLeaveCertification: row };
    },
    RecordLeaveCertification: (state, { input }) => {
      const { file, row } = fileHolding(state, (f) => f.leave.cases, input.caseId);
      bump(row, {
        certificationStatus: input.status,
        certificationReceivedAt: input.receivedAt ?? NOW,
        recertificationDueAt: input.recertificationDueAt ?? row.recertificationDueAt,
      });
      touch(file, "worker_leave", row.id);
      return { recordLeaveCertification: row };
    },

    // ---- Time off ----
    WorkerPtoBalances: (state, v) => {
      requireRead(state, "worker_pto");
      return { workerPtoBalances: ptoBalances(fileOf(state, v.workerId)) };
    },
    WorkerPtoLedger: (state, { input }) => {
      requireRead(state, "worker_pto");
      let rows = ledgerEntries(fileOf(state, input.workerId)).reverse();
      if (input.ptoType) rows = rows.filter((row) => row.ptoType === input.ptoType);
      if (input.entryType) rows = rows.filter((row) => row.entryType === input.entryType);
      if (input.effectiveFrom) rows = rows.filter((row) => row.effectiveAt >= input.effectiveFrom);
      if (input.effectiveTo) rows = rows.filter((row) => row.effectiveAt <= input.effectiveTo);
      const result = page(rows, input, true);
      return {
        workerPtoLedger: {
          edges: result.edges.map(({ node }) => ({ node })),
          totalCount: rows.length,
          totalCountExact: true,
          pageInfo: {
            hasNextPage: result.pageInfo.hasNextPage,
            endCursor: result.pageInfo.endCursor,
          },
        },
      };
    },
    WorkerPtoPolicyAssignments: (state, v) => ({
      workerPtoPolicyAssignments: connection(
        [...fileOf(state, v.workerId).pto.policyAssignments].sort(byNewest("effectiveFrom")),
        { first: v.first, after: v.after },
      ),
    }),
    AssignWorkerPtoPolicy: (state, { input }) => {
      const policy = Object.values(PTO_POLICIES).find((p) => p.id === input.ptoPolicyId);
      if (!policy) refuse("Time off policy not found");
      const assignments = [];
      const failures = [];
      for (const workerId of input.workerIds) {
        if (!isWorkerId(state, workerId)) {
          failures.push({ workerId, error: "Worker not found" });
          continue;
        }
        const file = fileOf(state, workerId);
        for (const row of file.pto.policyAssignments)
          if (!row.effectiveTo) bump(row, { effectiveTo: input.effectiveFrom - DAY });
        const row = {
          id: nextId("wpa"),
          workerId,
          ptoPolicyId: policy.id,
          effectiveFrom: input.effectiveFrom,
          effectiveTo: null,
          assignedById: PEOPLE.dana.id,
          note: input.note ?? null,
          version: 1,
          createdAt: NOW,
          updatedAt: NOW,
          ptoPolicy: { ...policy },
        };
        file.pto.policyAssignments.push(row);
        for (const opening of input.openingBalances ?? []) {
          file.pto.ledger.push({
            ptoType: opening.ptoType,
            entryType: "OpeningBalance",
            amount: Number(opening.days),
            effectiveAt: input.effectiveFrom,
            note: "Opening balance",
          });
          file.pto.balances[opening.ptoType] ??= {
            maxBalanceDays: null,
            carriedDays: "0",
            nextAccrual: null,
          };
        }
        assignments.push(row);
        touch(file, "worker_pto");
      }
      return { assignWorkerPtoPolicy: { assignments, failures } };
    },
    AdjustWorkerPtoBalance: (state, { input }) => {
      if (!input.note?.trim()) refuse("Say why the balance changes.");
      const file = fileOf(state, input.workerId);
      file.pto.balances[input.ptoType] ??= {
        maxBalanceDays: null,
        carriedDays: "0",
        nextAccrual: null,
      };
      const id = nextId("ptl");
      file.pto.ledger.push({
        id,
        ptoType: input.ptoType,
        entryType: "Adjustment",
        amount: Number(input.amountDays),
        effectiveAt: input.effectiveAt ?? NOW,
        note: input.note,
        createdById: PEOPLE.dana.id,
      });
      audit(file, {
        resource: "worker_pto",
        operation: "update",
        entityRef: `${input.ptoType} balance`,
        comment: input.note,
      });
      touch(file, "worker_pto");
      return { adjustWorkerPtoBalance: ledgerEntries(file).find((row) => row.id === id) };
    },
    RunPtoAccrual: (state, { input }) => {
      const files = input.workerId
        ? [fileOf(state, input.workerId)]
        : [...ws(state).files.values()];
      return {
        runPtoAccrual: {
          queued: false,
          workflowId: null,
          workersProcessed: files.length,
          workersSkipped: 0,
          entriesPosted: 0,
          entriesCapped: 0,
          entriesSkipped: files.length,
        },
      };
    },
    WorkerPtoTable: (state, v) => {
      if (v.input?.workerId && !isWorkerId(state, v.input.workerId))
        return fallback("WorkerPtoTable", state, v);
      const rows = ptoRows(state, v.input ?? {});
      const result = page(rows, v.input, v.includeTotalCount !== false);
      return { workerPTOEntries: result };
    },
    UpcomingWorkerPto: (state, v) => {
      const rows = ptoRows(state, {
        ...v.input,
        sort: [{ field: "startDate", direction: "asc" }],
      }).filter((row) => row.endDate >= NOW - DAY);
      return { upcomingWorkerPTO: page(rows, v.input, true) };
    },
    CreateWorkerPto: (state, { input }) => {
      const file = fileOf(state, input.workerId);
      if (!input.reason?.trim()) refuse("Reason is required.");
      if (input.endDate < input.startDate) refuse("End date is before the start date.");
      let days = 0;
      for (let t = input.startDate; t <= input.endDate; t += DAY) {
        const weekday = new Date(t * 1000).getUTCDay();
        if (weekday !== 0 && weekday !== 6) days += 1;
      }
      const balance = ptoBalances(file).find((row) => row.ptoType === input.type);
      if (balance && Number(balance.availableDays) < days)
        refuse(`Only ${balance.availableDays} ${input.type.toLowerCase()} days are available.`);
      const row = ptoRequest(file.worker, 0, {
        status: "Requested",
        type: input.type,
        startDate: input.startDate,
        endDate: input.endDate,
        reason: input.reason,
        days: String(days),
        requestedAt: NOW,
      });
      row.id = nextId("wpt");
      file.pto.requests.unshift(row);
      audit(file, {
        resource: "worker_pto",
        resourceId: row.id,
        operation: "create",
        entityRef: `${input.type}, ${days} days`,
      });
      touch(file, "worker_pto", row.id);
      return { createWorkerPTO: row };
    },
    ApproveWorkerPto: (state, v) =>
      isWorkerPto(state, v.id)
        ? {
            approveWorkerPTO: decidePto(
              state,
              v.id,
              { status: "Approved", approverId: PEOPLE.dana.id, approver: DANA() },
              "approve",
            ),
          }
        : fallback("ApproveWorkerPto", state, v),
    RejectWorkerPto: (state, v) => {
      if (!v.reason?.trim()) refuse("Say why the request is turned down.");
      return {
        rejectWorkerPTO: decidePto(
          state,
          v.id,
          {
            status: "Rejected",
            rejectorId: PEOPLE.dana.id,
            rejector: DANA(),
            rejectionReason: v.reason,
          },
          "reject",
        ),
      };
    },
    CancelWorkerPto: (state, v) => ({
      cancelWorkerPTO: decidePto(
        state,
        v.id,
        {
          status: "Cancelled",
          cancelledById: PEOPLE.dana.id,
          cancelledBy: DANA(),
          cancellationReason: v.reason ?? null,
        },
        "cancel",
      ),
    }),

    // ---- Schedule ----
    WorkerShiftAssignments: (state, v) => ({
      workerShiftAssignments: fileOf(state, v.workerId).schedule.assignments.filter(
        (row) => !v.activeOnly || !row.effectiveTo || row.effectiveTo > NOW,
      ),
    }),
    WorkerAvailabilityPreferences: (state, v) => ({
      workerAvailabilityPreferences: fileOf(state, v.workerId).schedule.availability,
    }),
    ShiftSwapRequests: (state, v) => ({
      shiftSwapRequests:
        v.workerId && isWorkerId(state, v.workerId) ? fileOf(state, v.workerId).schedule.swaps : [],
    }),
    AssignWorkerShift: (state, { input }) => {
      const file = fileOf(state, input.workerId);
      const template = Object.values(SHIFT_TEMPLATES).find((t) => t.id === input.shiftTemplateId);
      if (!template) refuse("Shift template not found");
      for (const row of file.schedule.assignments)
        if (!row.effectiveTo) row.effectiveTo = input.effectiveFrom - DAY;
      const row = {
        id: nextId("wsa"),
        workerId: file.worker.id,
        shiftTemplateId: template.id,
        effectiveFrom: input.effectiveFrom,
        effectiveTo: null,
        cycleOffsetWeeks: input.cycleOffsetWeeks ?? 0,
        notes: input.notes ?? null,
        version: 1,
        shiftTemplate: { ...template },
      };
      file.schedule.assignments.push(row);
      audit(file, {
        resource: "worker_schedule",
        resourceId: row.id,
        operation: "assign",
        entityRef: template.name,
      });
      touch(file, "worker_schedule", row.id);
      return { assignWorkerShift: row };
    },
    EndWorkerShiftAssignment: (state, v) => {
      const { file, row } = fileHolding(state, (f) => f.schedule.assignments, v.id);
      bump(row, { effectiveTo: v.effectiveTo });
      touch(file, "worker_schedule", row.id);
      return { endWorkerShiftAssignment: row };
    },
    SetWorkerAvailabilityPreference: (state, { input }) => {
      const file = fileOf(state, input.workerId);
      let row = file.schedule.availability.find((entry) => entry.dayOfWeek === input.dayOfWeek);
      if (row) bump(row, { preference: input.preference, note: input.note ?? row.note });
      else {
        row = {
          id: nextId("wap"),
          workerId: file.worker.id,
          dayOfWeek: input.dayOfWeek,
          preference: input.preference,
          note: input.note ?? null,
          version: 1,
        };
        file.schedule.availability.push(row);
      }
      touch(file, "worker_schedule", row.id);
      return { setWorkerAvailabilityPreference: row };
    },

    // ---- Hours of service ----
    WorkerHosState: (state, v) =>
      isWorkerId(state, v.workerId)
        ? { workerHosState: hosState(state, fileOf(state, v.workerId)) }
        : fallback("WorkerHosState", state, v),
    WorkerHosClocks: (state, v) => ({ workerHosState: hosState(state, fileOf(state, v.workerId)) }),
    WorkerHosStates: (state, v) => {
      const mine = (v.workerIds ?? []).filter((id) => isWorkerId(state, id));
      if (!mine.length) return fallback("WorkerHosStates", state, v);
      return {
        workerHosStates: mine.map((id) => hosState(state, fileOf(state, id))).filter(Boolean),
      };
    },
    WorkerHosLogs: (state, v) => {
      const file = fileOf(state, v.workerId);
      return {
        workerHosLogs:
          hosLink(state, file) === "Reporting" ? logEntries(file, v.startTime, v.endTime) : [],
      };
    },
    WorkerHosDailyLogs: (state, v) => {
      const file = fileOf(state, v.workerId);
      return {
        workerHosDailyLogs:
          hosLink(state, file) === "Reporting" ? dailyLogs(file, v.startDate, v.endDate) : [],
      };
    },
    WorkerHosViolations: (state, v) =>
      v.workerId && isWorkerId(state, v.workerId)
        ? { workerHosViolations: [] }
        : (fallback("WorkerHosViolations", state, v) ?? { workerHosViolations: [] }),
    WorkerFormSubmissions: (state, v) => {
      const file = fileOf(state, v.workerId);
      const rows = file.hos.forms.map((form, index) => {
        const submittedAt =
          startOfUtcDay(NOW) - form.dayOffset * DAY + form.hour * 3600 + form.minute * 60;
        return {
          id: `${file.worker.id}-form-${index}`,
          templateId: `tpl-${index}`,
          templateName: form.templateName,
          submittedAt,
          fields: form.fields.map(([label, value]) => ({ label, value })),
        };
      });
      return {
        workerFormSubmissions: rows
          .filter((row) => row.submittedAt >= v.startTime && row.submittedAt <= v.endTime)
          .sort(byNewest("submittedAt")),
      };
    },
    TelematicsStatus: (state, v) => {
      const answer = fallback("TelematicsStatus", state, v);
      if (workerVariant(state).hos !== "notConnected") return answer;
      return {
        telematicsStatus: {
          ...answer.telematicsStatus,
          enabled: false,
          configured: false,
          webhookConfigured: false,
          lastPolledAt: null,
          lastSuccessAt: null,
          mappedTractors: 0,
          mappedWorkers: 0,
        },
      };
    },

    // ---- Pay ----
    EffectiveWorkerPayAssignment: (state, v) => {
      const row = effectivePayAssignment(fileOf(state, v.workerId));
      return { effectiveWorkerPayAssignment: row ? payAssignmentOut(row, true) : null };
    },
    WorkerPayAssignments: (state, v) => ({
      workerPayAssignments: connection(
        [...fileOf(state, v.workerId).pay.assignments]
          .sort(byNewest("effectiveFrom"))
          .map((row) => payAssignmentOut(row)),
        { first: v.first, after: v.after },
      ),
    }),
    WorkerEarningsSummary: (state, v) => ({
      workerEarningsSummary: { workerId: v.workerId, ...fileOf(state, v.workerId).pay.earnings },
    }),
    CurrentSettlementPeriod: () => ({
      currentSettlementPeriod: {
        periodStart: startOfUtcDay(NOW) - 6 * DAY,
        periodEnd: startOfUtcDay(NOW) + DAY - 1,
        payDate: startOfUtcDay(NOW) + 5 * DAY,
      },
    }),
    EndWorkerPayAssignment: (state, { input }) => {
      const { file, row } = fileHolding(state, (f) => f.pay.assignments, input.assignmentId);
      bump(row, { effectiveTo: input.endDate });
      touch(file, "driver_pay_profile", row.id);
      return { endWorkerPayAssignment: row };
    },
    AssignPayProfileToWorker: (state, { input }) => {
      const file = fileOf(state, input.workerId);
      const profile = Object.values(PAY_PROFILES).find((p) => p.id === input.payProfileId);
      if (!profile) refuse("Pay profile not found");
      for (const row of file.pay.assignments)
        if (!row.effectiveTo || row.effectiveTo >= input.effectiveFrom)
          row.effectiveTo = input.effectiveFrom - DAY;
      const row = {
        id: nextId("wpy"),
        workerId: file.worker.id,
        payProfileId: profile.id,
        effectiveFrom: input.effectiveFrom,
        effectiveTo: input.effectiveTo ?? null,
        splitPercent: String(input.splitPercent ?? "100"),
        rateOverrides: (input.rateOverrides ?? []).map((o) => ({
          componentId: o.componentId,
          rate: String(o.rate),
        })),
        notes: input.notes ?? "",
        version: 1,
        createdAt: NOW,
      };
      file.pay.assignments.push(row);
      audit(file, {
        resource: "driver_pay_profile",
        resourceId: row.id,
        operation: "assign",
        entityRef: profile.name,
      });
      touch(file, "driver_pay_profile", row.id);
      return { assignPayProfileToWorker: row };
    },
    PayProfileDetail: (_state, v) => {
      const profile = Object.values(PAY_PROFILES).find((p) => p.id === v.id);
      return {
        payProfile: profile
          ? { ...profile, components: profile.components.map((c) => ({ ...c })) }
          : null,
      };
    },

    // ---- Benefits ----
    WorkerBenefitEnrollments: (state, v) => ({
      workerBenefitEnrollments: connection(
        fileOf(state, v.workerId)
          .benefits.enrollments.filter(
            (row) => !v.openOnly || row.status === "Active" || row.status === "Pending",
          )
          .map(enrollmentOut),
        { first: v.first, after: v.after },
      ),
    }),
    WorkerTotalCompensation: (state, v) => ({
      workerTotalCompensation: totalCompensation(fileOf(state, v.workerId), v.planYear),
    }),
    EnrollBenefit: (state, { input }) => {
      const file = fileOf(state, input.workerId);
      const plan = Object.values(BENEFIT_PLANS).find((p) => p.id === input.benefitPlanId);
      if (!plan) refuse("Benefit plan not found");
      let row = file.benefits.enrollments.find(
        (entry) =>
          entry.benefitPlanId === plan.id &&
          (entry.status === "Pending" || entry.status === "Active"),
      );
      const patch = input.waive
        ? {
            status: "Waived",
            waivedReason: input.waivedReason ?? null,
            employeeCostMinor: 0,
            employerCostMinor: 0,
          }
        : {
            status: "Active",
            coverageTier: input.coverageTier ?? "Employee",
            effectiveFrom: input.effectiveFrom ?? NOW,
            employeeCostMinor: input.employeeCostMinor ?? 0,
            employerCostMinor: 0,
          };
      if (row) bump(row, { ...patch, notes: input.notes ?? row.notes });
      else {
        row = {
          id: nextId("wbe"),
          workerId: file.worker.id,
          benefitPlanId: plan.id,
          coverageTier: "Employee",
          effectiveFrom: NOW,
          effectiveTo: null,
          employeeCostMinor: 0,
          employerCostMinor: 0,
          recurringDeductionId: null,
          waivedReason: null,
          notes: input.notes ?? null,
          version: 1,
          benefitPlan: { ...plan },
          ...patch,
        };
        file.benefits.enrollments.push(row);
      }
      audit(file, {
        resource: "benefit_plan",
        resourceId: row.id,
        operation: "create",
        entityRef: plan.name,
      });
      touch(file, "benefit_plan", row.id);
      return { enrollBenefit: enrollmentOut(row) };
    },
    EndBenefitEnrollment: (state, { input }) => {
      const { file, row } = fileHolding(state, (f) => f.benefits.enrollments, input.id);
      bump(row, {
        status: "Ended",
        effectiveTo: input.effectiveTo ?? NOW,
        notes: input.notes ?? row.notes,
      });
      touch(file, "benefit_plan", row.id);
      return { endBenefitEnrollment: enrollmentOut(row) };
    },

    // ---- Driver portal ----
    WorkerPortalStatus: (state, v) => {
      const portal = fileOf(state, v.workerId).portal;
      return {
        workerPortalStatus: {
          linked: portal.linked,
          portalUser: portal.portalUser,
          pendingInvitation: portal.pendingInvitation,
          invitations: portal.invitations,
        },
      };
    },
    InviteWorkerToPortal: (state, { input }) => {
      const file = fileOf(state, input.workerId);
      const email = input.email ?? file.worker.email;
      if (!email) refuse("The worker has no email address.");
      if (file.portal.pendingInvitation && !input.replacePending)
        refuse("An invitation is already pending.");
      if (file.portal.pendingInvitation) file.portal.pendingInvitation.status = "Revoked";
      const invitation = {
        id: nextId("pin"),
        email,
        status: "Pending",
        expiresAt: NOW + 7 * DAY,
        acceptedAt: null,
        createdAt: NOW,
        invitedBy: DANA(),
      };
      file.portal.invitations.unshift(invitation);
      file.portal.pendingInvitation = {
        id: invitation.id,
        email,
        status: "Pending",
        expiresAt: invitation.expiresAt,
        createdAt: NOW,
      };
      touch(file, "driver_portal");
      return {
        inviteWorkerToPortal: {
          invitation: {
            id: invitation.id,
            email,
            status: "Pending",
            expiresAt: invitation.expiresAt,
          },
          inviteUrl: null,
          emailSent: true,
        },
      };
    },
    RevokeWorkerPortalAccess: (state, v) => {
      const file = fileOf(state, v.workerId);
      file.portal.linked = false;
      file.portal.portalUser = null;
      audit(file, {
        resource: "driver_portal",
        operation: "update",
        entityRef: "Dash access",
        comment: "Revoked Dash access",
      });
      touch(file, "driver_portal");
      return { revokeWorkerPortalAccess: true };
    },
    WorkerPolicyAcknowledgements: (state, v) => ({
      workerPolicyAcknowledgements: connection(fileOf(state, v.workerId).portal.acknowledgements, {
        first: v.first,
        after: v.after,
      }),
    }),
    ProfileChangeRequests: (state, v) => ({
      profileChangeRequests: profileChangesFor(state, v.filter ?? {}),
    }),
    DecideProfileChange: (state, { input }) => {
      const { file, row } = fileHolding(state, (f) => f.portal.profileChanges, input.id);
      if (row.status !== "Pending") refuse("This request has already been decided.");
      bump(row, {
        status: input.approve ? "Approved" : "Rejected",
        decidedAt: NOW,
        decisionNote: input.note ?? null,
      });
      if (input.approve) {
        for (const change of row.changes) {
          if (change.field in file.worker) file.worker[change.field] = change.to;
          else if (change.field in file.worker.profile)
            file.worker.profile[change.field] = change.to;
        }
        file.worker.version += 1;
      }
      audit(file, {
        resource: "profile_change_request",
        resourceId: row.id,
        operation: input.approve ? "approve" : "reject",
        entityRef: row.changes.map((c) => c.label).join(" and "),
      });
      touch(file, "profile_change_request", row.id);
      return { decideProfileChange: row };
    },

    // ---- Agent memory ----
    DeskMemories: (state, v) =>
      v.input?.subjectType === "Worker" || isWorkerId(state, v.input?.subjectId)
        ? { deskMemories: memoryList(state, v.input) }
        : (fallback("DeskMemories", state, v) ?? {
            deskMemories: { items: [], next: null, all: 0, counts: [] },
          }),
    CreateDeskMemory: (state, { input }) => {
      if (!input.subjectId || !isWorkerId(state, input.subjectId))
        refuse("Memory subject not found");
      const file = fileOf(state, input.subjectId);
      const row = {
        id: nextId("amm"),
        content: input.content,
        kind: input.kind ?? "Fact",
        scope: input.scope,
        roleId: input.roleId ?? null,
        roleName: "",
        subjectType: "Worker",
        subjectId: file.worker.id,
        subjectLabel: file.worker.wholeName,
        status: "Active",
        source: "User",
        sourceTitle: PEOPLE.dana.name,
        useCount: 0,
        lastUsedAt: null,
        createdAt: NOW,
        version: 1,
        editable: true,
        reason: "",
        quotes: [],
        replaces: null,
        replacedBy: null,
      };
      file.memories.unshift(row);
      audit(file, {
        resource: "agent_memory",
        resourceId: row.id,
        operation: "create",
        entityRef: row.content,
      });
      touch(file, "agent_memory", row.id);
      return { createDeskMemory: row };
    },
    ReviseDeskMemory: (state, v) => {
      const { file, row } = memoryById(state, v.id);
      bump(
        row,
        Object.fromEntries(
          Object.entries({
            content: v.input.content,
            scope: v.input.scope,
            roleId: v.input.roleId,
          }).filter(([, value]) => value !== undefined),
        ),
      );
      touch(file, "agent_memory", row.id);
      return { reviseDeskMemory: row };
    },
    SetDeskMemoryStatus: (state, v) => {
      const { file, row } = memoryById(state, v.id);
      bump(row, { status: v.status });
      audit(file, {
        resource: "agent_memory",
        resourceId: row.id,
        operation: v.status === "Retired" ? "delete" : "update",
        entityRef: row.content,
      });
      touch(file, "agent_memory", row.id);
      return { setDeskMemoryStatus: row };
    },

    // ---- Pickers ----
    SelectOptions: (state, v) => {
      const resource = v.input?.resource;
      const mine = workerSelectRows(state, resource);
      if (!mine) return fallback("SelectOptions", state, v);
      if (resource === "USER" || resource === "US_STATE" || resource === "WORKER") {
        const theirs = fallback("SelectOptions", state, v);
        const known = new Set((theirs?.selectOptions?.edges ?? []).map((edge) => edge.node.id));
        const merged = [
          ...mine.filter((row) => !known.has(row.id)),
          ...(theirs?.selectOptions?.edges ?? []).map((edge) => edge.node),
        ];
        return selectConnection(merged, { ...v.input, offset: v.input.offset ?? 0 });
      }
      return selectConnection(mine, v.input);
    },
  };

  function reviewStatus(state, id, key, patch) {
    const { file, row } = fileHolding(state, (f) => f.reviews, id);
    bump(row, patch(row));
    audit(file, {
      resource: "performance_review",
      resourceId: row.id,
      operation: key.startsWith("submit") ? "submit" : "update",
      entityRef: row.title,
    });
    touch(file, "performance_review", row.id);
    return { [key]: reviewOut(row) };
  }

  function safetyStatus(state, input, key, patch) {
    const { file, row } = fileHolding(state, (f) => f.safety.events, input.id);
    bump(row, patch);
    audit(file, {
      resource: "worker_safety_event",
      resourceId: row.id,
      operation: "update",
      entityRef: row.kind,
      changes: { status: { from: null, to: row.status, type: "updated", fieldType: "string" } },
    });
    touch(file, "worker_safety_event", row.id);
    return { [key]: safetyEventOut(row) };
  }
}

function isWorkerPto(state, id) {
  for (const file of ws(state).files.values())
    if (file.pto.requests.some((row) => row.id === id)) return true;
  return false;
}

function settleChecklistItem(file, label) {
  for (const checklist of file.checklists) {
    if (checklist.status !== "Open") continue;
    const item = checklist.items.find(
      (entry) => entry.label === label && entry.status === "Pending",
    );
    if (item) bump(item, { status: "Done", autoCompleted: true, completedAt: NOW });
  }
}

// ---- REST ---------------------------------------------------------------------------------

const WORKER_PATH = /^\/api\/v1\/workers\/(wrk_[A-Z0-9]+)\/?$/;
const DOCUMENTS_PATH = /^\/api\/v1\/documents\/resource\/worker\/(wrk_[A-Z0-9]+)\/?$/;
const PACKET_PATH = /^\/api\/v1\/documents\/resource\/worker\/(wrk_[A-Z0-9]+)\/packet-summary\/?$/;

function restWorker(state, path) {
  const id = path.match(WORKER_PATH)?.[1];
  return id && isWorkerId(state, id) ? fileOf(state, id) : null;
}

// ---- Document uploads (the single-part session protocol `useDocumentUpload` speaks) ---------

const UPLOAD_PATH = /^\/api\/v1\/documents\/uploads\/(dus_01JF8Q2M3V[0-9]+)\/?$/;
const UPLOAD_ACTION =
  /^\/api\/v1\/documents\/uploads\/(dus_01JF8Q2M3V[0-9]+)\/(parts|complete|cancel|content)\/?$/;
const DOCUMENT_PATH = /^\/api\/v1\/documents\/(doc_01JF8Q2M3V[0-9]+)\/?$/;
const PUBLIC_URL =
  process.env.MOCK_API_PUBLIC_URL ?? `http://localhost:${process.env.MOCK_API_PORT ?? 8080}`;

function uploads(state) {
  ws(state).uploads ??= new Map();
  return ws(state).uploads;
}

function sessionOut(session) {
  return { ...session, version: session.version, updatedAt: session.updatedAt };
}

function createUpload(state, body) {
  const file = fileOf(state, body.resourceId);
  const id = nextId("dus");
  const session = {
    id,
    ...TENANT,
    documentId: null,
    lineageId: body.lineageId ?? null,
    resourceId: file.worker.id,
    resourceType: "worker",
    processingProfile: body.processingProfile ?? "none",
    documentTypeId: body.documentTypeId ?? null,
    originalName: body.fileName,
    contentType: body.contentType,
    fileSize: body.fileSize,
    storagePath: `mock/workers/${file.worker.id}/${body.fileName}`,
    storageProviderUploadId: "",
    strategy: "single",
    status: "Initiated",
    description: body.description ?? null,
    tags: body.tags ?? [],
    uploadedParts: [],
    partSize: body.fileSize,
    failureCode: null,
    failureMessage: null,
    malwareScan: false,
    threatName: null,
    expiresAt: Math.floor(Date.now() / 1000) + DAY,
    lastActivityAt: Math.floor(Date.now() / 1000),
    version: 1,
    createdAt: Math.floor(Date.now() / 1000),
    updatedAt: Math.floor(Date.now() / 1000),
  };
  uploads(state).set(id, session);
  return session;
}

function completeUpload(state, session) {
  const file = fileOf(state, session.resourceId);
  const doc = workerDocument(
    file.worker.id,
    0,
    session.originalName,
    session.contentType,
    session.fileSize,
    NOW,
  );
  doc.id = nextId("doc");
  doc.lineageId = session.lineageId ?? doc.id;
  doc.uploadedById = PEOPLE.dana.id;
  doc.previewStatus = "Pending";
  file.documents.unshift(doc);
  Object.assign(session, {
    status: "Available",
    documentId: doc.id,
    uploadedParts: [{ partNumber: 1, etag: "mock", size: session.fileSize }],
    version: session.version + 1,
  });
  audit(file, {
    resource: "document",
    resourceId: doc.id,
    operation: "create",
    entityRef: doc.fileName,
  });
  touch(file, "document", doc.id);
  return session;
}

function documentById(state, id) {
  for (const file of ws(state).files.values()) {
    const doc = file.documents.find((row) => row.id === id);
    if (doc) return doc;
  }
  return null;
}

const notFound = {
  __status: 404,
  body: {
    type: "https://api.trenova.app/problems/not-found",
    title: "Not found",
    status: 404,
    detail: "Worker not found",
  },
};

export const WORKER_ROUTES = [
  [
    "POST",
    /^\/api\/v1\/documents\/uploads\/?$/,
    (state, _path, body) =>
      body?.resourceType === "worker" && isWorkerId(state, body.resourceId)
        ? createUpload(state, body)
        : {
            __status: 422,
            body: { title: "Uploads are mocked for the worker page only", status: 422 },
          },
  ],
  [
    "GET",
    UPLOAD_PATH,
    (state, path) => {
      const session = uploads(state).get(path.match(UPLOAD_PATH)[1]);
      return session ? { session: sessionOut(session), parts: session.uploadedParts } : notFound;
    },
  ],
  [
    "POST",
    UPLOAD_ACTION,
    (state, path) => {
      const [, id, action] = path.match(UPLOAD_ACTION);
      const session = uploads(state).get(id);
      if (!session) return notFound;
      if (action === "parts")
        return {
          parts: [{ partNumber: 1, url: `${PUBLIC_URL}/api/v1/documents/uploads/${id}/content/` }],
        };
      if (action === "cancel") {
        session.status = "Canceled";
        return {};
      }
      return sessionOut(completeUpload(state, session));
    },
  ],
  [
    "PUT",
    UPLOAD_ACTION,
    (state, path) => {
      const session = uploads(state).get(path.match(UPLOAD_ACTION)[1]);
      if (!session) return notFound;
      session.status = "Uploaded";
      return {};
    },
  ],
  [
    "GET",
    DOCUMENT_PATH,
    (state, path) => documentById(state, path.match(DOCUMENT_PATH)[1]) ?? notFound,
  ],
  ["GET", /^\/api\/v1\/me\/permissions\/?$/, (state) => manifest(state)],
  [
    "GET",
    /^\/api\/v1\/me\/permissions\/version\/?$/,
    (state) => ({
      checksum: manifest(state).checksum,
      expiresAt: Math.floor(Date.now() / 1000) + 86400,
    }),
  ],
  [
    "PUT",
    WORKER_PATH,
    (state, path, body) => {
      const file = restWorker(state, path);
      return file ? saveWorker(state, file, body) : notFound;
    },
  ],
  [
    "PATCH",
    WORKER_PATH,
    (state, path, body) => {
      const file = restWorker(state, path);
      return file ? saveWorker(state, file, body) : notFound;
    },
  ],
  [
    "GET",
    WORKER_PATH,
    (state, path) => {
      const file = restWorker(state, path);
      return file ? workerOut(state, file) : notFound;
    },
  ],
  [
    "POST",
    /^\/api\/v1\/workers\/?$/,
    (state, _path, body) => ({ __status: 201, body: createWorker(state, body) }),
  ],
  [
    "GET",
    PACKET_PATH,
    (state, path) => {
      const id = path.match(PACKET_PATH)[1];
      return isWorkerId(state, id) ? packetSummary(fileOf(state, id)) : notFound;
    },
  ],
  [
    "GET",
    DOCUMENTS_PATH,
    (state, path, _body, query) => {
      const id = path.match(DOCUMENTS_PATH)[1];
      if (!isWorkerId(state, id)) return [];
      const text = String(query?.get("query") ?? "").toLowerCase();
      return fileOf(state, id).documents.filter(
        (doc) => !text || doc.fileName.toLowerCase().includes(text),
      );
    },
  ],
];
