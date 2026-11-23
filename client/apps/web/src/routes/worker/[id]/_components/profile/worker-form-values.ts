import type { WorkerRecord } from "@/lib/graphql/worker-record";
import type { Worker, WorkerProfile } from "@trenova/shared/types/worker";

/** A new worker before anything is typed: the defaults the server expects of a hire. */
export const NEW_WORKER_VALUES: Worker = {
  status: "Active",
  type: "Employee",
  driverType: "OTR",
  gender: "Male",
  firstName: "",
  lastName: "",
  addressLine1: "",
  addressLine2: "",
  city: "",
  postalCode: "",
  stateId: "",
  fleetCodeId: "",
  managerId: "",
  positionId: "",
  email: "",
  phoneNumber: "",
  emergencyContactName: "",
  emergencyContactPhone: "",
  externalId: "",
  profilePicUrl: "",
  canBeAssigned: false,
  availableForDispatch: true,
  homeLatitude: null,
  homeLongitude: null,
  homePlaceId: "",
  lastHomeAt: null,
  assignmentBlocked: "",
  profile: {
    dob: undefined,
    hireDate: undefined as unknown as number,
    licenseNumber: "",
    licenseStateId: "",
    licenseExpiry: undefined as unknown as number,
    cdlClass: "A",
    cdlRestrictions: "",
    endorsement: "O",
    hazmatExpiry: undefined,
    medicalCardExpiry: undefined,
    medicalExaminerName: "",
    medicalExaminerNpi: "",
    twicCardNumber: "",
    twicExpiry: undefined,
    terminationDate: undefined,
    physicalDueDate: undefined,
    mvrDueDate: undefined,
    complianceStatus: "Pending",
    isQualified: false,
    disqualificationReason: "",
    lastComplianceCheck: 0,
    lastMvrCheck: 0,
    lastDrugTest: 0,
    eldExempt: false,
    shortHaulExempt: false,
  } as WorkerProfile,
  id: undefined,
  version: undefined,
  createdAt: undefined,
  updatedAt: undefined,
  state: undefined,
  fleetCode: undefined,
  pto: undefined,
};

const WORKER_TEXT = [
  "fleetCodeId",
  "managerId",
  "positionId",
  "addressLine2",
  "email",
  "emergencyContactName",
  "emergencyContactPhone",
  "externalId",
  "profilePicUrl",
  "homePlaceId",
  "assignmentBlocked",
] as const;

const PROFILE_TEXT = [
  "licenseStateId",
  "licenseNumber",
  "cdlRestrictions",
  "medicalExaminerName",
  "medicalExaminerNpi",
  "twicCardNumber",
  "disqualificationReason",
] as const;

/**
 * Profile fields the server derives and keeps on every update, never taking them from the request:
 * compliance from credentials, training and safety standing, drug and alcohol standing.
 */
export const SERVER_OWNED_PROFILE_FIELDS = [
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
] as const;

/**
 * The worker as the Profile form saves it: without the profile fields the server owns, so a value
 * read before another tab's write is never sent back as though the person had changed it.
 */
export function workerSaveValues(values: Worker): Worker {
  if (!values.profile) return values;
  const owned = new Set<string>(SERVER_OWNED_PROFILE_FIELDS);
  const profile = Object.fromEntries(
    Object.entries(values.profile).filter(([key]) => !owned.has(key)),
  );
  return { ...values, profile: profile as Worker["profile"] };
}

/**
 * The stored worker as the Profile form holds it: optional text as empty strings, so a field
 * cleared and a field never filled compare equal, and without the relations the form only reads
 * (position, manager), so the full-record save sends exactly the worker's own fields.
 */
export function workerFormValues(record: WorkerRecord): Worker {
  const { profile } = record;
  const values = { ...record } as Record<string, unknown>;
  delete values.position;
  delete values.manager;
  for (const key of WORKER_TEXT) values[key] = record[key] ?? "";
  let formProfile: Record<string, unknown> | null = null;
  if (profile) {
    formProfile = { ...profile };
    for (const key of PROFILE_TEXT) formProfile[key] = profile[key] ?? "";
  }
  values.profile = formProfile;
  values.customFields = record.customFields ?? {};
  return values as unknown as Worker;
}
