// What the server works out from a worker's stored records, ported rule for rule so the mock's
// overview agrees with the lists behind it: credential and training health and their summaries
// (workercredential, workertraining), the DQ file counts, the drug and alcohol standing, the
// safety scorecard, checklist progress, time off balances from the ledger, and the standing,
// concerns and clearance runway (workercompliance/standing.go, clearance.go).
import {
  COURSE_LIST,
  CREDENTIAL_TYPE_LIST,
  DAY,
  PEOPLE,
  PTO_POLICIES,
  WORKER_NOW,
  daysUntil,
  requiredCourses,
  requiredCredentialTypes,
} from "../fixtures/worker/base.mjs";

export const NOW = WORKER_NOW;
const CLEARINGHOUSE_DUE_SOON_DAYS = 30;
const REVIEW_DUE_SOON_DAYS = 30;
const TRAINING_DUE_SOON_DAYS = 7;

const typeById = new Map(CREDENTIAL_TYPE_LIST.map((type) => [type.id, type]));
const courseById = new Map(COURSE_LIST.map((entry) => [entry.id, entry]));

export function credentialTypeOf(id) {
  return typeById.get(id) ?? null;
}

export function courseOf(id) {
  return courseById.get(id) ?? null;
}

// ---- Credentials --------------------------------------------------------------------------

export function credentialHealth(expiresAt, renewalWindowDays) {
  if (!expiresAt || expiresAt <= 0) return "Valid";
  const days = daysUntil(expiresAt, NOW);
  if (days < 0) return "Expired";
  if (days <= renewalWindowDays) return "ExpiringSoon";
  return "Valid";
}

export function credentialOut(cred) {
  const type = credentialTypeOf(cred.credentialTypeId);
  return {
    ...cred,
    health: credentialHealth(cred.expiresAt, type?.renewalWindowDays ?? 30),
    daysUntilExpiry: cred.expiresAt ? daysUntil(cred.expiresAt, NOW) : null,
    credentialType: type ? { ...type } : null,
  };
}

export function credentialSummary(file) {
  const wrk = file.worker;
  const active = new Map();
  for (const cred of file.credentials) {
    if (cred.status !== "Active") continue;
    active.set(cred.credentialTypeId, cred);
  }
  const summary = {
    workerId: wrk.id,
    complianceStatus: "Compliant",
    requiredCount: 0,
    validCount: 0,
    expiringCount: 0,
    expiredCount: 0,
    missingCount: 0,
    items: [],
  };
  const required = new Set(requiredCredentialTypes(wrk.driverType).map((type) => type.id));
  const add = (item) => {
    summary.items.push(item);
    if (item.required) summary.requiredCount += 1;
    if (item.health === "Valid") summary.validCount += 1;
    if (item.health === "ExpiringSoon") summary.expiringCount += 1;
    if (item.health === "Expired") summary.expiredCount += 1;
    if (item.health === "Missing") summary.missingCount += 1;
    if (item.required && (item.health === "Expired" || item.health === "Missing")) {
      summary.complianceStatus = "NonCompliant";
    }
  };
  for (const type of CREDENTIAL_TYPE_LIST) {
    const isRequired = required.has(type.id);
    const cred = active.get(type.id);
    if (!isRequired && !cred) continue;
    const out = cred ? credentialOut(cred) : null;
    add({
      credentialType: { ...type },
      credential: out,
      required: isRequired,
      health: out ? out.health : "Missing",
      daysUntilExpiry: out ? out.daysUntilExpiry : null,
    });
  }
  summary.items.sort((a, b) =>
    a.required !== b.required
      ? a.required
        ? -1
        : 1
      : a.credentialType.sortOrder - b.credentialType.sortOrder,
  );
  return summary;
}

// ---- Training -----------------------------------------------------------------------------

const OPEN_TRAINING = new Set(["Assigned", "InProgress"]);

export function trainingHealth(record, renewalWindowDays) {
  if (!record) return "Missing";
  switch (record.status) {
    case "Assigned":
    case "InProgress": {
      if (!record.dueAt) return "Scheduled";
      const days = daysUntil(record.dueAt, NOW);
      if (days < 0) return "Overdue";
      if (days <= TRAINING_DUE_SOON_DAYS) return "DueSoon";
      return "Scheduled";
    }
    case "Completed": {
      if (!record.expiresAt) return "Current";
      const days = daysUntil(record.expiresAt, NOW);
      if (days < 0) return "Expired";
      if (days <= renewalWindowDays) return "ExpiringSoon";
      return "Current";
    }
    case "Waived":
      return "Current";
    case "Expired":
      return "Expired";
    case "Failed":
      return "Failed";
    default:
      return "Missing";
  }
}

export function trainingOut(record) {
  const entry = courseOf(record.courseId);
  return {
    ...record,
    course: entry ? { ...entry } : null,
    health: trainingHealth(record, entry?.renewalWindowDays ?? 30),
    daysUntilDue:
      record.dueAt && OPEN_TRAINING.has(record.status) ? daysUntil(record.dueAt, NOW) : null,
    daysUntilExpiry: record.expiresAt ? daysUntil(record.expiresAt, NOW) : null,
  };
}

/** The record standing for each course: the slot the file names, else the open or newest one. */
function trainingSlot(file, courseId) {
  const named = file.trainingSlots?.[courseId];
  const records = file.training.filter((record) => record.courseId === courseId);
  const byId = records.find((record) => record.id === named);
  if (byId && byId.status !== "Cancelled") return byId;
  const open = records.find((record) => OPEN_TRAINING.has(record.status));
  if (open) return open;
  return (
    records
      .filter((record) => record.status !== "Cancelled")
      .sort((a, b) => (b.completedAt ?? b.assignedAt) - (a.completedAt ?? a.assignedAt))[0] ?? null
  );
}

export function trainingSummary(file) {
  const required = new Set(requiredCourses(file.worker.driverType).map((entry) => entry.id));
  const summary = {
    workerId: file.worker.id,
    compliant: true,
    requiredCount: 0,
    currentCount: 0,
    dueCount: 0,
    overdueCount: 0,
    expiringCount: 0,
    expiredCount: 0,
    missingCount: 0,
    items: [],
  };
  for (const entry of COURSE_LIST) {
    const isRequired = required.has(entry.id);
    const record = trainingSlot(file, entry.id);
    if (!isRequired && !record) continue;
    const out = record ? trainingOut(record) : null;
    const health = out ? out.health : "Missing";
    summary.items.push({
      course: { ...entry },
      record: out,
      health,
      daysUntilDue: out?.daysUntilDue ?? null,
      daysUntilExpiry: out?.daysUntilExpiry ?? null,
      required: isRequired,
    });
    if (isRequired) summary.requiredCount += 1;
    if (health === "Current") summary.currentCount += 1;
    if (health === "Scheduled" || health === "DueSoon") summary.dueCount += 1;
    if (health === "Overdue") summary.overdueCount += 1;
    if (health === "ExpiringSoon") {
      summary.currentCount += 1;
      summary.expiringCount += 1;
    }
    if (health === "Expired") summary.expiredCount += 1;
    if (health === "Failed" || health === "Missing") summary.missingCount += 1;
    if (isRequired && ["Overdue", "Expired", "Failed", "Missing"].includes(health)) {
      summary.compliant = false;
    }
  }
  return summary;
}

const TRAINING_RANK = [
  "Expired",
  "Missing",
  "Failed",
  "Overdue",
  "DueSoon",
  "ExpiringSoon",
  "Scheduled",
  "Current",
];

export function worstTrainingHealth(summary) {
  return (
    summary.items
      .filter((item) => item.required)
      .map((item) => item.health)
      .sort((a, b) => TRAINING_RANK.indexOf(a) - TRAINING_RANK.indexOf(b))[0] ?? "Current"
  );
}

// ---- Driver qualification file ------------------------------------------------------------

const SETTLED_VERIFICATION = new Set(["Received", "NotApplicable", "NoResponse"]);

export function verificationSettled(verification) {
  return SETTLED_VERIFICATION.has(verification.status);
}

export function dqfFile(file) {
  const wrk = file.worker;
  const verifications = file.dqf.verifications;
  const late =
    verifications.some((row) => !verificationSettled(row)) && NOW > file.dqf.safetyHistoryDueAt;
  const items = file.dqf.items.map((item) => {
    if (item.section !== "SafetyHistory" || verifications.length === 0) return { ...item };
    const open = verifications.filter((row) => !verificationSettled(row)).length;
    return {
      ...item,
      status: open ? "Outstanding" : "Satisfied",
      detail: open
        ? `${open} of ${verifications.length} employers ${open === 1 ? "has" : "have"} not answered`
        : `All ${verifications.length} employers answered`,
    };
  });
  const count = (status) => items.filter((item) => item.status === status).length;
  const terminated = wrk.profile.terminationDate;
  return {
    workerId: wrk.id,
    complete: count("Missing") + count("Expired") + count("Outstanding") === 0,
    missingRequired: count("Missing"),
    expiringSoon: count("ExpiringSoon"),
    expired: count("Expired"),
    outstanding: count("Outstanding"),
    hireDate: file.dqf.hireDate,
    terminationDate: terminated,
    safetyHistoryDueAt: file.dqf.safetyHistoryDueAt,
    safetyHistoryLate: late,
    retentionExpiresAt: terminated ? terminated + 3 * 365 * DAY : null,
    purgeEligible: false,
    items,
    verifications: verifications.map((row) => ({ ...row })),
  };
}

// ---- Drug and alcohol ---------------------------------------------------------------------

export function drugAlcoholStanding(file) {
  const { tests, violations, queries } = file.drugAlcohol;
  const completed = queries
    .filter((query) => query.completedAt && query.result !== "Pending")
    .sort((a, b) => b.completedAt - a.completedAt);
  const last = completed[0] ?? null;
  const open = violations.filter((violation) => violation.status !== "Resolved");
  const hasPreTest = tests.some(
    (test) => test.testType === "PreEmployment" && test.result === "Negative",
  );
  const hasPreQuery = queries.some(
    (query) => query.queryType === "PreEmploymentFull" && query.result === "NoViolations",
  );
  return {
    status: open.length
      ? "Prohibited"
      : hasPreTest
        ? "Clear"
        : tests.length
          ? "Pending"
          : "Unknown",
    returnToDuty: open.length ? "SAPEvaluation" : "NotRequired",
    lastClearinghouseQueryAt: last?.requestedAt ?? null,
    nextClearinghouseQueryDue: last ? last.requestedAt + 365 * DAY : null,
    hasPreEmploymentTest: hasPreTest,
    hasPreEmploymentQuery: hasPreQuery,
    openTestCount: tests.filter((test) => !["Completed", "Cancelled"].includes(test.status)).length,
  };
}

// ---- Safety -------------------------------------------------------------------------------

export function safetyEventOut(event) {
  return {
    ...event,
    activePoints: event.pointsExpireAt && event.pointsExpireAt > NOW ? event.points : 0,
  };
}

export function disciplinaryOut(action) {
  const active = action.status === "Active" && (!action.expiresAt || action.expiresAt > NOW);
  return {
    ...action,
    status: action.status === "Active" && !active ? "Expired" : action.status,
    active,
  };
}

const LEVELS = [
  "Coaching",
  "VerbalWarning",
  "WrittenWarning",
  "FinalWarning",
  "Suspension",
  "Termination",
];

export function disciplinaryLadder(file) {
  const active = file.safety.actions.map(disciplinaryOut).filter((action) => action.active);
  const highest =
    active.map((action) => action.level).sort((a, b) => LEVELS.indexOf(b) - LEVELS.indexOf(a))[0] ??
    null;
  const next = highest
    ? LEVELS[Math.min(LEVELS.indexOf(highest) + 1, LEVELS.length - 1)]
    : "Coaching";
  return {
    highestLevel: highest,
    suggestedLevel: next,
    atFinalStep: highest === "Termination",
    activeActions: active.map((action) => ({
      id: action.id,
      level: action.level,
      status: action.status,
      reason: action.reason,
      issuedAt: action.issuedAt,
      expiresAt: action.expiresAt,
      active: action.active,
    })),
  };
}

export function safetyScorecard(file) {
  const card = file.safety.scorecard;
  const events = file.safety.events.map(safetyEventOut);
  const activePoints = events.reduce((sum, event) => sum + event.activePoints, 0);
  const ladder = disciplinaryLadder(file);
  const last = events.map((event) => event.occurredAt).sort((a, b) => b - a)[0] ?? null;
  const rating =
    activePoints >= card.pointsAtRiskThreshold
      ? "AtRisk"
      : activePoints >= card.pointsWatchThreshold
        ? "Watch"
        : card.rating === "Watch" || card.rating === "AtRisk"
          ? "Good"
          : card.rating;
  return {
    workerId: file.worker.id,
    asOf: NOW,
    ...card,
    rating,
    activePoints,
    openEvents: events.filter((event) => event.status !== "Closed").length,
    activeDiscipline: ladder.activeActions.length,
    highestDiscipline: ladder.highestLevel,
    daysSinceLastEvent: last ? daysUntil(NOW, last) : null,
    lastEventAt: last,
    recognitions: file.safety.recognitions.length,
  };
}

// ---- Checklists ---------------------------------------------------------------------------

export function checklistOut(checklist) {
  const items = checklist.items.map((item) => ({
    ...item,
    overdue: item.status === "Pending" && !!item.dueAt && item.dueAt < NOW,
  }));
  const required = items.filter((item) => item.required);
  const requiredDone = required.filter((item) => item.status !== "Pending").length;
  const settled = items.filter((item) => item.status !== "Pending").length;
  const overdue = items.filter((item) => item.overdue).length;
  return {
    ...checklist,
    items,
    progress: {
      total: items.length,
      settled,
      requiredTotal: required.length,
      requiredDone,
      overdue,
      percent: required.length ? Math.round((requiredDone / required.length) * 100) : 100,
      complete: requiredDone === required.length,
    },
  };
}

export function openChecklist(file) {
  return (
    file.checklists
      .filter((checklist) => checklist.status === "Open")
      .sort((a, b) => b.startedAt - a.startedAt)
      .map(checklistOut)[0] ?? null
  );
}

// ---- Time off -----------------------------------------------------------------------------

const YEAR_START = Date.UTC(new Date(NOW * 1000).getUTCFullYear(), 0, 1) / 1000;

const decimal = (value) => {
  const rounded = Math.round(value * 100) / 100;
  return String(rounded);
};

/** Ledger entries with their running balance and sequence, oldest first per type. */
export function ledgerEntries(file) {
  const assignment = file.pto.policyAssignments.find((row) => !row.effectiveTo);
  const sorted = file.pto.ledger
    .map((entry, index) => ({ entry, index }))
    .sort((a, b) => a.entry.effectiveAt - b.entry.effectiveAt || a.index - b.index);
  const balances = {};
  return sorted.map(({ entry, index }, sequence) => {
    balances[entry.ptoType] = (balances[entry.ptoType] ?? 0) + entry.amount;
    const system =
      entry.entryType === "Accrual" ||
      entry.entryType === "Carryover" ||
      entry.entryType === "Expiry";
    return {
      id: entry.id ?? `ptl_${file.worker.id.slice(-6)}_${index}`,
      workerId: file.worker.id,
      ptoType: entry.ptoType,
      entryType: entry.entryType,
      amountDays: decimal(entry.amount),
      balanceAfterDays: decimal(balances[entry.ptoType]),
      sequence: sequence + 1,
      effectiveAt: entry.effectiveAt,
      periodKey: entry.periodKey ?? null,
      sourcePtoId: entry.sourcePtoId ?? null,
      assignmentId: assignment?.id ?? null,
      ptoPolicyId: assignment?.ptoPolicyId ?? null,
      actorType: system ? "System" : "User",
      createdById: system ? null : (entry.createdById ?? PEOPLE.priya.id),
      note: entry.note ?? null,
      createdAt: entry.effectiveAt,
    };
  });
}

export function ptoBalances(file) {
  return Object.entries(file.pto.balances).map(([ptoType, config]) => {
    const entries = file.pto.ledger.filter((entry) => entry.ptoType === ptoType);
    const balance = entries.reduce((sum, entry) => sum + entry.amount, 0);
    const pending = file.pto.requests
      .filter((request) => request.type === ptoType && request.status === "Requested")
      .reduce((sum, request) => sum + Number(request.days), 0);
    const accrued = entries
      .filter(
        (entry) =>
          entry.effectiveAt >= YEAR_START &&
          (entry.entryType === "Accrual" || (entry.entryType === "Adjustment" && entry.amount > 0)),
      )
      .reduce((sum, entry) => sum + entry.amount, 0);
    const used = entries
      .filter((entry) => entry.effectiveAt >= YEAR_START && entry.entryType === "Usage")
      .reduce((sum, entry) => sum - entry.amount, 0);
    return {
      ptoType,
      tracked: true,
      enforced: true,
      balanceDays: decimal(balance),
      pendingDays: decimal(pending),
      availableDays: decimal(balance - pending),
      accruedYtdDays: decimal(accrued),
      usedYtdDays: decimal(used),
      carriedDays: config.carriedDays ?? "0",
      maxBalanceDays: config.maxBalanceDays ?? null,
      nextAccrual: config.nextAccrual ?? null,
    };
  });
}

export function currentPtoPolicy(file) {
  const assignment = file.pto.policyAssignments.find(
    (row) => !row.effectiveTo || row.effectiveTo > NOW,
  );
  return assignment
    ? Object.values(PTO_POLICIES).find((policy) => policy.id === assignment.ptoPolicyId)
    : null;
}

// ---- Leave --------------------------------------------------------------------------------

export function leaveEntitlement(file) {
  return {
    ...file.leave.entitlement,
    openCaseCount: file.leave.cases.filter(
      (row) => row.status === "Pending" || row.status === "Approved",
    ).length,
  };
}

// ---- Standing, concerns and clearance -----------------------------------------------------

function dated(dueAt, concern) {
  if (!dueAt || dueAt <= 0) return { ...concern, dueAt: null, daysUntil: null };
  return { ...concern, dueAt, daysUntil: daysUntil(dueAt, NOW) };
}

function concern(spec) {
  return {
    subject: null,
    occurredAt: null,
    dueAt: null,
    daysUntil: null,
    stopsAssignment: false,
    ...spec,
  };
}

const at = (tab, view = null, record = null, recordId = null) => ({ tab, view, record, recordId });

function credentialDestination(item) {
  return item.credential
    ? at("Qualification", "Credentials", "Credential", item.credential.id)
    : at("Qualification", "Credentials");
}

function trainingDestination(item) {
  return item.record
    ? at("Development", "Training", "Training", item.record.id)
    : at("Development", "Training");
}

function trainingDueAt(item) {
  if (!item.record) return null;
  if (OPEN_TRAINING.has(item.record.status)) return item.record.dueAt || null;
  return item.record.expiresAt || null;
}

const SEVERITY_RANK = { Critical: 0, Warning: 1, Info: 2 };

/**
 * `workercompliance.BuildStanding`. Sections the reader may not see arrive as null and count as
 * neither a problem nor evidence. No dispatch compliance verdicts are modelled, so only the
 * required-credential rule stops assignment.
 */
export function buildStanding(input) {
  const wrk = input.worker;
  if (wrk.status !== "Active") {
    return {
      standing: "Blocked",
      concerns: [
        concern({
          severity: "Info",
          code: "NotEmployed",
          count: 1,
          stopsAssignment: true,
          destination: at("Employment", "History"),
        }),
      ],
    };
  }
  const concerns = [];
  let blocked = !wrk.canBeAssigned;
  if (blocked) {
    concerns.push(
      concern({
        severity: "Critical",
        code: "NotAssignable",
        count: 1,
        stopsAssignment: true,
        destination: at("Profile"),
      }),
    );
  }
  if (input.credentials) {
    for (const item of input.credentials.items) {
      const codes = {
        Expired: ["Critical", "CredentialExpired"],
        Missing: ["Critical", "CredentialMissing"],
        ExpiringSoon: ["Warning", "CredentialExpiring"],
      }[item.health];
      if (!codes) continue;
      let entry = concern({
        severity: codes[0],
        code: codes[1],
        subject: item.credentialType.name,
        count: 1,
        stopsAssignment: item.required,
        destination: credentialDestination(item),
      });
      if (item.credential) {
        entry.occurredAt = item.credential.issuedAt;
        entry = dated(item.credential.expiresAt, entry);
      }
      concerns.push(entry);
    }
    if (input.credentials.complianceStatus === "NonCompliant") blocked = true;
  }
  if (input.training) {
    for (const item of input.training.items) {
      const codes = {
        Expired: ["Critical", "TrainingExpired"],
        Missing: ["Critical", "TrainingMissing"],
        Failed: ["Critical", "TrainingMissing"],
        Overdue: ["Critical", "TrainingOverdue"],
        Scheduled: ["Warning", "TrainingDue"],
        DueSoon: ["Warning", "TrainingDue"],
        ExpiringSoon: ["Warning", "TrainingDue"],
      }[item.health];
      if (!codes) continue;
      let entry = concern({
        severity: codes[0],
        code: codes[1],
        subject: item.course.name,
        count: 1,
        destination: trainingDestination(item),
      });
      if (item.record) {
        entry.occurredAt = item.record.assignedAt || null;
        entry = dated(trainingDueAt(item), entry);
      }
      concerns.push(entry);
    }
  }
  if (input.safety) {
    const card = input.safety;
    if (card.rating === "AtRisk" || card.rating === "Watch") {
      concerns.push(
        concern({
          severity: card.rating === "AtRisk" ? "Critical" : "Warning",
          code: "SafetyRating",
          count: card.activePoints,
          destination: at("Safety"),
        }),
      );
    }
    for (const event of input.safetyEvents) {
      const dest = at("Safety", null, "Event", event.id);
      if (event.outOfService && event.occurredAt >= NOW - 365 * DAY) {
        concerns.push(
          concern({
            severity: "Critical",
            code: "OutOfService",
            count: 1,
            occurredAt: event.occurredAt,
            destination: dest,
          }),
        );
      }
      if (event.status !== "Closed") {
        concerns.push(
          concern({
            severity: "Info",
            code: "OpenSafetyEvent",
            count: 1,
            occurredAt: event.occurredAt,
            destination: dest,
          }),
        );
      }
    }
    if (card.activeDiscipline > 0) {
      concerns.push(
        concern({
          severity: "Warning",
          code: "ActiveDiscipline",
          count: card.activeDiscipline,
          destination: at("Safety"),
        }),
      );
    }
  }
  if (input.checklist && input.checklist.status === "Open") {
    const list = input.checklist;
    let entry = dated(
      list.dueAt,
      concern({
        severity: "Info",
        code: "ChecklistOpen",
        subject: list.name,
        count: 1,
        occurredAt: list.startedAt,
        destination: at("Employment", "Checklists", "Checklist", list.id),
      }),
    );
    if (list.progress.overdue > 0) {
      entry = {
        ...entry,
        severity: "Warning",
        code: "ChecklistOverdue",
        count: list.progress.overdue,
      };
    }
    concerns.push(entry);
  }
  if (input.reviewDueAt) {
    const due = input.reviewDueAt;
    const dest = at("Development", "Reviews");
    if (due < NOW)
      concerns.push(
        dated(
          due,
          concern({ severity: "Warning", code: "ReviewOverdue", count: 1, destination: dest }),
        ),
      );
    else if (due <= NOW + REVIEW_DUE_SOON_DAYS * DAY)
      concerns.push(
        dated(due, concern({ severity: "Info", code: "ReviewDue", count: 1, destination: dest })),
      );
  }
  if (input.drugAlcohol) {
    const standing = input.drugAlcohol;
    const dest = at("Qualification", "DrugAndAlcohol");
    if (standing.status === "Prohibited") {
      concerns.push(
        concern({
          severity: "Critical",
          code: "DrugAlcoholProhibited",
          count: 1,
          destination: dest,
        }),
      );
    }
    const due = standing.nextClearinghouseQueryDue;
    if (due && due < NOW) {
      concerns.push(
        dated(
          due,
          concern({
            severity: "Warning",
            code: "ClearinghouseQueryOverdue",
            count: 1,
            occurredAt: standing.lastClearinghouseQueryAt,
            destination: dest,
          }),
        ),
      );
    } else if (due && due <= NOW + CLEARINGHOUSE_DUE_SOON_DAYS * DAY) {
      concerns.push(
        dated(
          due,
          concern({
            severity: "Info",
            code: "ClearinghouseQueryDue",
            count: 1,
            occurredAt: standing.lastClearinghouseQueryAt,
            destination: dest,
          }),
        ),
      );
    }
  }
  for (const verification of input.verifications ?? []) {
    if (verificationSettled(verification)) continue;
    let entry = concern({
      severity: "Info",
      code: "EmploymentVerificationOutstanding",
      subject: verification.employerName,
      count: 1,
      occurredAt: verification.requestedAt,
      destination: at("Qualification", "DqFile", "Employer", verification.id),
    });
    if (input.safetyHistoryDueAt) {
      entry = dated(input.safetyHistoryDueAt, entry);
      if (input.safetyHistoryDueAt < NOW) entry.severity = "Warning";
    }
    concerns.push(entry);
  }

  const indexed = concerns.map((entry, index) => ({ entry, index }));
  indexed.sort((a, b) => {
    const x = a.entry;
    const y = b.entry;
    if (SEVERITY_RANK[x.severity] !== SEVERITY_RANK[y.severity])
      return SEVERITY_RANK[x.severity] - SEVERITY_RANK[y.severity];
    if (x.stopsAssignment !== y.stopsAssignment) return x.stopsAssignment ? -1 : 1;
    if (x.dueAt != null && y.dueAt != null && x.dueAt !== y.dueAt) return x.dueAt - y.dueAt;
    if (x.dueAt != null && y.dueAt == null) return -1;
    if (x.dueAt == null && y.dueAt != null) return 1;
    return a.index - b.index;
  });
  const ranked = indexed.map(({ entry }) => entry);
  const standing = blocked
    ? "Blocked"
    : ranked.some((entry) => entry.severity === "Critical")
      ? "AtRisk"
      : ranked.some((entry) => entry.severity === "Warning")
        ? "Watch"
        : "Good";
  return { standing, concerns: ranked };
}

/** `workercompliance.BuildClearance`, without dispatch compliance verdicts. */
export function buildClearance(input) {
  const wrk = input.worker;
  if (!input.credentials) return null;
  const blocks = [];
  if (wrk.status !== "Active") blocks.push("NotEmployed");
  if (!wrk.canBeAssigned || !wrk.availableForDispatch || wrk.assignmentBlocked)
    blocks.push("DispatchHold");
  const required = input.credentials.items.filter((item) => item.required);
  if (required.some((item) => item.health === "Expired")) blocks.push("RequiredCredentialLapsed");
  if (required.some((item) => item.health === "Missing")) blocks.push("RequiredCredentialMissing");

  const lane = (category) =>
    category === "Medical"
      ? "Medical"
      : category === "Background"
        ? "ReviewsAndQueries"
        : "License";
  const items = [];
  for (const item of input.credentials.items) {
    if (!item.credential?.expiresAt) continue;
    const due = item.credential.expiresAt;
    items.push({
      key: `credential:${item.credentialType.id}`,
      lane: lane(item.credentialType.category),
      source: "Credential",
      name: item.credentialType.name,
      dueAt: due,
      daysUntil: daysUntil(due, NOW),
      effect: item.required ? "StopsAssignment" : "Due",
      rule: item.required ? "RequiredCredential" : null,
      hazmatOnly: false,
      regulation: null,
      required: item.required,
      destination: credentialDestination(item),
    });
  }
  for (const item of input.training?.items ?? []) {
    const due = trainingDueAt(item);
    if (!due) continue;
    items.push({
      key: `training:${item.course.id}`,
      lane: "Training",
      source: "Training",
      name: item.course.name,
      dueAt: due,
      daysUntil: daysUntil(due, NOW),
      effect: "Due",
      rule: null,
      hazmatOnly: false,
      regulation: null,
      required: item.required,
      destination: trainingDestination(item),
    });
  }
  const queryDue = input.drugAlcohol?.nextClearinghouseQueryDue;
  if (queryDue) {
    items.push({
      key: "clearinghouse",
      lane: "ReviewsAndQueries",
      source: "ClearinghouseQuery",
      name: "Clearinghouse query",
      dueAt: queryDue,
      daysUntil: daysUntil(queryDue, NOW),
      effect: "Due",
      rule: null,
      hazmatOnly: false,
      regulation: null,
      required: true,
      destination: at("Qualification", "DrugAndAlcohol"),
    });
  }
  if (input.reviewDueAt) {
    items.push({
      key: "review",
      lane: "ReviewsAndQueries",
      source: "PerformanceReview",
      name: "Performance review",
      dueAt: input.reviewDueAt,
      daysUntil: daysUntil(input.reviewDueAt, NOW),
      effect: "Due",
      rule: null,
      hazmatOnly: false,
      regulation: null,
      required: false,
      destination: at("Development", "Reviews"),
    });
  }
  items.sort((a, b) => a.dueAt - b.dueAt || (a.key < b.key ? -1 : 1));

  const cleared = blocks.length === 0;
  let firstStop = null;
  let clearedThrough = null;
  let notClearedSince = null;
  for (const item of items) {
    if (item.effect !== "StopsAssignment" || item.hazmatOnly) continue;
    firstStop ??= item;
    if (item.dueAt < NOW) {
      if (!cleared && notClearedSince == null) notClearedSince = item.dueAt;
      continue;
    }
    if (cleared && clearedThrough == null) clearedThrough = item.dueAt;
  }
  return {
    cleared,
    clearedThrough,
    notClearedSince,
    blocks,
    firstStop,
    items: items.slice(0, 60),
    itemsTruncated: items.length > 60,
  };
}
