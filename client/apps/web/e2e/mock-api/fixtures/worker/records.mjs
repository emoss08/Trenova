// Builders for each kind of record in a worker's file, shaped as the GraphQL types the worker page
// selects (client/packages/graphql/src/operations/**). Every builder fills the type's whole field
// set so a record can answer any operation that selects it; the derived parts (health, days until,
// progress) are computed by the handler from the stored dates, as the server does.
import {
  CREDENTIAL_TYPES,
  COURSES,
  DAY,
  FLEET_CODES,
  PEOPLE,
  POSITIONS,
  STATES,
  TENANT,
  pid,
  userRef,
} from "./base.mjs";

let sequence = 0;
/** A fresh id for a record the page creates. */
export function nextId(prefix) {
  sequence += 1;
  return pid(prefix, 900000 + sequence);
}

const stamped = (createdAt) => ({ version: 1, createdAt, updatedAt: createdAt });

export function documentRef(id, fileName, fileType, fileSize, createdAt, previewStatus) {
  return { id, fileName, originalName: fileName, fileType, fileSize, previewStatus, createdAt };
}

export function worker(spec) {
  const fleet = spec.fleet ? FLEET_CODES[spec.fleet] : null;
  const state = STATES[spec.state ?? "IL"];
  const licenseState = STATES[spec.licenseState ?? spec.state ?? "IL"];
  const position = spec.position ? POSITIONS[spec.position] : null;
  const manager = spec.manager ? PEOPLE[spec.manager] : null;
  const id = spec.id;
  return {
    id,
    ...TENANT,
    stateId: state.id,
    fleetCodeId: fleet?.id ?? null,
    managerId: manager?.id ?? null,
    positionId: position?.id ?? null,
    status: spec.status ?? "Active",
    type: spec.type ?? "Employee",
    driverType: spec.driverType ?? "OTR",
    leaveType: null,
    profilePicUrl: "",
    firstName: spec.firstName,
    lastName: spec.lastName,
    wholeName: `${spec.firstName} ${spec.lastName}`,
    addressLine1: spec.addressLine1,
    addressLine2: spec.addressLine2 ?? "",
    city: spec.city,
    postalCode: spec.postalCode,
    email: spec.email ?? null,
    phoneNumber: spec.phoneNumber,
    emergencyContactName: spec.emergencyContactName ?? "",
    emergencyContactPhone: spec.emergencyContactPhone ?? "",
    externalId: "",
    assignmentBlocked: spec.assignmentBlocked ?? "",
    gender: spec.gender,
    canBeAssigned: spec.canBeAssigned ?? true,
    availableForDispatch: spec.availableForDispatch ?? true,
    homeLatitude: spec.homeLatitude ?? null,
    homeLongitude: spec.homeLongitude ?? null,
    homePlaceId: null,
    lastHomeAt: spec.lastHomeAt ?? null,
    version: spec.version ?? 14,
    createdAt: spec.hireDate - 3 * DAY,
    updatedAt: spec.updatedAt ?? spec.hireDate,
    customFields: spec.customFields ?? {},
    fleetCode: fleet ? { id: fleet.id, code: fleet.code, color: fleet.color } : null,
    position: position ? { ...position } : null,
    manager: manager ? userRef(manager) : null,
    state: { ...state },
    profile: {
      id: pid("wpr", Number(id.slice(-4))),
      workerId: id,
      ...TENANT,
      licenseStateId: licenseState.id,
      dob: spec.dob,
      licenseNumber: spec.licenseNumber,
      cdlClass: spec.cdlClass ?? "A",
      cdlRestrictions: spec.cdlRestrictions ?? "",
      endorsement: spec.endorsement ?? "O",
      hazmatExpiry: spec.hazmatExpiry ?? null,
      licenseExpiry: spec.licenseExpiry,
      medicalCardExpiry: spec.medicalCardExpiry ?? null,
      medicalExaminerName: spec.medicalExaminerName ?? "",
      medicalExaminerNpi: spec.medicalExaminerNpi ?? "",
      twicCardNumber: spec.twicCardNumber ?? null,
      twicExpiry: spec.twicExpiry ?? null,
      hireDate: spec.hireDate,
      terminationDate: spec.terminationDate ?? null,
      physicalDueDate: spec.physicalDueDate ?? spec.medicalCardExpiry ?? null,
      mvrDueDate: spec.mvrDueDate ?? null,
      complianceStatus: "Compliant",
      trainingHealth: "Current",
      safetyRating: "Excellent",
      safetyScore: 100,
      nextCredentialExpiry: null,
      nextTrainingDue: null,
      drugAlcoholStatus: "Clear",
      returnToDutyStatus: "NotRequired",
      lastClearinghouseQueryAt: null,
      nextClearinghouseQueryDue: null,
      isQualified: spec.isQualified ?? true,
      disqualificationReason: spec.disqualificationReason ?? "",
      lastComplianceCheck: spec.updatedAt ?? spec.hireDate,
      lastMvrCheck: spec.lastMvrCheck ?? spec.hireDate,
      lastDrugTest: spec.lastDrugTest ?? spec.hireDate,
      eldExempt: spec.eldExempt ?? false,
      shortHaulExempt: spec.shortHaulExempt ?? false,
      version: 9,
      createdAt: spec.hireDate - 3 * DAY,
      updatedAt: spec.updatedAt ?? spec.hireDate,
      licenseState: { ...licenseState },
    },
  };
}

export function credential(workerId, n, typeKey, spec = {}) {
  const type = CREDENTIAL_TYPES[typeKey];
  const verifier = spec.verifiedBy ? PEOPLE[spec.verifiedBy] : null;
  return {
    id: pid("wcr", n),
    workerId,
    credentialTypeId: type.id,
    status: spec.archivedAt ? "Archived" : "Active",
    number: spec.number ?? null,
    issuingAuthority: spec.issuingAuthority ?? null,
    issuedAt: spec.issuedAt ?? null,
    expiresAt: spec.expiresAt ?? null,
    documentId: spec.document?.id ?? null,
    notes: spec.notes ?? null,
    verifiedById: verifier?.id ?? null,
    verifiedAt: verifier ? (spec.verifiedAt ?? spec.issuedAt ?? null) : null,
    archivedById: spec.archivedAt ? PEOPLE.dana.id : null,
    archivedAt: spec.archivedAt ?? null,
    archiveReason: spec.archiveReason ?? null,
    ...stamped(spec.issuedAt ?? spec.createdAt ?? 0),
    document: spec.document ?? null,
    verifiedBy: userRef(verifier),
  };
}

export function trainingRecord(workerId, n, courseKey, spec = {}) {
  const entry = COURSES[courseKey];
  const assignedBy = PEOPLE[spec.assignedBy ?? "dana"];
  const recordedBy = spec.recordedBy ? PEOPLE[spec.recordedBy] : null;
  return {
    id: pid("wtr", n),
    ...TENANT,
    workerId,
    courseId: entry.id,
    status: spec.status ?? "Completed",
    assignedAt: spec.assignedAt,
    dueAt: spec.dueAt ?? null,
    startedAt: spec.startedAt ?? null,
    completedAt: spec.completedAt ?? null,
    expiresAt: spec.expiresAt ?? null,
    score: spec.score ?? null,
    passed: spec.completedAt ? (spec.passed ?? true) : null,
    acknowledgedAt: spec.acknowledgedAt ?? null,
    documentId: spec.document?.id ?? null,
    document: spec.document ?? null,
    assignedById: assignedBy.id,
    assignedBy: userRef(assignedBy),
    recordedById: recordedBy?.id ?? null,
    recordedBy: userRef(recordedBy),
    notes: spec.notes ?? null,
    waivedReason: spec.waivedReason ?? null,
    ...stamped(spec.assignedAt),
  };
}

export function employmentEvent(workerId, n, kind, effectiveAt, spec = {}) {
  const recorder = PEOPLE[spec.recordedBy ?? "marcus"];
  const amender = spec.amendedBy ? PEOPLE[spec.amendedBy] : null;
  const values = (pairs) => Object.entries(pairs ?? {}).map(([key, value]) => ({ key, value }));
  return {
    id: pid("wee", n),
    workerId,
    kind,
    effectiveAt,
    reason: spec.reason ?? null,
    notes: spec.notes ?? null,
    fromValues: values(spec.from),
    toValues: values(spec.to),
    documentId: null,
    document: null,
    recordedById: recorder.id,
    recordedBy: userRef(recorder),
    amendedById: amender?.id ?? null,
    amendedBy: userRef(amender),
    amendedAt: spec.amendedAt ?? null,
    amendmentNote: spec.amendmentNote ?? null,
    ...stamped(spec.recordedAt ?? effectiveAt),
  };
}

export function safetyEvent(workerId, n, spec) {
  const recorder = PEOPLE[spec.recordedBy ?? "dana"];
  const closer = spec.closedBy ? PEOPLE[spec.closedBy] : null;
  const id = pid("wse", n);
  return {
    id,
    ...TENANT,
    workerId,
    kind: spec.kind,
    severity: spec.severity ?? "Minor",
    status: spec.status ?? "Open",
    occurredAt: spec.occurredAt,
    location: spec.location ?? null,
    description: spec.description,
    preventable: spec.preventable ?? false,
    points: spec.points ?? 0,
    pointsExpireAt: spec.pointsExpireAt ?? null,
    referenceNumber: spec.referenceNumber ?? null,
    shipmentId: null,
    inspectionLevel: spec.inspectionLevel ?? null,
    inspectionResult: spec.inspectionResult ?? null,
    outOfService: spec.outOfService ?? false,
    fineAmount: spec.fineAmount ?? null,
    costAmount: spec.costAmount ?? null,
    documentId: spec.document?.id ?? null,
    document: spec.document ?? null,
    recordedById: recorder.id,
    recordedBy: userRef(recorder),
    closedById: closer?.id ?? null,
    closedBy: userRef(closer),
    closedAt: spec.closedAt ?? null,
    resolution: spec.resolution ?? null,
    ...stamped(spec.occurredAt + DAY),
    violations: (spec.violations ?? []).map((violation, index) => ({
      id: pid("wsv", n * 10 + index),
      safetyEventId: id,
      workerId,
      code: null,
      outOfService: false,
      version: 1,
      ...violation,
    })),
  };
}

export function disciplinaryAction(workerId, n, spec) {
  const issuer = PEOPLE[spec.issuedBy ?? "dana"];
  return {
    id: pid("wda", n),
    ...TENANT,
    workerId,
    level: spec.level,
    status: spec.status ?? "Active",
    reason: spec.reason,
    details: spec.details ?? null,
    occurredAt: spec.occurredAt ?? null,
    issuedAt: spec.issuedAt,
    expiresAt: spec.expiresAt ?? null,
    suspensionDays: spec.suspensionDays ?? null,
    safetyEventId: spec.safetyEvent?.id ?? null,
    safetyEvent: spec.safetyEvent
      ? {
          id: spec.safetyEvent.id,
          kind: spec.safetyEvent.kind,
          occurredAt: spec.safetyEvent.occurredAt,
        }
      : null,
    documentId: null,
    issuedById: issuer.id,
    issuedBy: userRef(issuer),
    acknowledgedAt: spec.acknowledgedAt ?? null,
    workerComment: spec.workerComment ?? null,
    rescindedAt: null,
    rescindedById: null,
    rescindReason: null,
    ...stamped(spec.issuedAt),
  };
}

export function recognition(workerId, n, spec) {
  const awarder = PEOPLE[spec.awardedBy ?? "dana"];
  return {
    id: pid("wrc", n),
    ...TENANT,
    workerId,
    kind: spec.kind,
    title: spec.title,
    message: spec.message ?? null,
    occurredAt: spec.occurredAt,
    awardedById: awarder.id,
    awardedBy: userRef(awarder),
    visibleToWorker: spec.visibleToWorker ?? false,
    ...stamped(spec.occurredAt),
  };
}

export function ptoRequest(worker, n, spec) {
  const approver = spec.approvedBy ? PEOPLE[spec.approvedBy] : null;
  return {
    id: pid("wpt", n),
    workerId: worker.id,
    ...TENANT,
    approverId: approver?.id ?? null,
    rejectorId: null,
    cancelledById: spec.cancelledByWorker ? worker.id : null,
    status: spec.status,
    type: spec.type,
    startDate: spec.startDate,
    endDate: spec.endDate,
    reason: spec.reason,
    rejectionReason: null,
    cancellationReason: spec.cancellationReason ?? null,
    days: spec.days,
    balanceAfterDays: spec.balanceAfterDays ?? null,
    autoApproved: false,
    ...stamped(spec.requestedAt),
    worker: {
      id: worker.id,
      firstName: worker.firstName,
      lastName: worker.lastName,
      wholeName: worker.wholeName,
      profilePicUrl: "",
    },
    approver: userRef(approver),
    rejector: null,
    cancelledBy: spec.cancelledByWorker ? { id: worker.id, name: worker.wholeName } : null,
  };
}

export function hiredEvent(workerId, n, hireDate, fleet) {
  return employmentEvent(workerId, n, "Hired", hireDate, {
    reason: "New hire",
    to: { status: "Active", ...(fleet ? { fleetCode: fleet } : {}) },
  });
}
