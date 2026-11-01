import {
  CreateWorkerDocument,
  UpdateWorkerDocument,
  type WorkerInput,
  type WorkerProfileInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type { Worker, WorkerProfile } from "@trenova/shared/types/worker";

function toWorkerProfileInput(profile: WorkerProfile): WorkerProfileInput {
  return {
    licenseStateId: profile.licenseStateId ?? null,
    dob: profile.dob ?? null,
    licenseNumber: profile.licenseNumber || null,
    cdlClass: profile.cdlClass,
    cdlRestrictions: profile.cdlRestrictions ?? null,
    endorsement: profile.endorsement,
    hazmatExpiry: profile.hazmatExpiry ?? null,
    licenseExpiry: profile.licenseExpiry,
    medicalCardExpiry: profile.medicalCardExpiry ?? null,
    medicalExaminerName: profile.medicalExaminerName ?? null,
    medicalExaminerNpi: profile.medicalExaminerNpi ?? null,
    twicCardNumber: profile.twicCardNumber ?? null,
    twicExpiry: profile.twicExpiry ?? null,
    hireDate: profile.hireDate,
    terminationDate: profile.terminationDate ?? null,
    physicalDueDate: profile.physicalDueDate ?? null,
    mvrDueDate: profile.mvrDueDate ?? null,
    complianceStatus: profile.complianceStatus,
    isQualified: profile.isQualified,
    disqualificationReason: profile.disqualificationReason ?? null,
    eldExempt: profile.eldExempt,
    shortHaulExempt: profile.shortHaulExempt,
  };
}

export function toWorkerInput(data: Worker): WorkerInput {
  return {
    stateId: data.stateId,
    fleetCodeId: data.fleetCodeId ?? null,
    managerId: data.managerId ?? null,
    positionId: data.positionId ?? null,
    status: data.status,
    type: data.type,
    driverType: data.driverType,
    gender: data.gender,
    firstName: data.firstName,
    lastName: data.lastName,
    addressLine1: data.addressLine1,
    addressLine2: data.addressLine2 ?? null,
    city: data.city,
    postalCode: data.postalCode,
    email: data.email ?? null,
    phoneNumber: data.phoneNumber || null,
    emergencyContactName: data.emergencyContactName ?? null,
    emergencyContactPhone: data.emergencyContactPhone || null,
    externalId: data.externalId ?? null,
    canBeAssigned: data.canBeAssigned,
    availableForDispatch: data.availableForDispatch,
    customFields: data.customFields ?? null,
    version: data.version,
    profile: data.profile ? toWorkerProfileInput(data.profile) : null,
  };
}

export class WorkerService {
  public async create(data: Worker) {
    const response = await requestGraphQL({
      document: CreateWorkerDocument,
      operationName: "CreateWorker",
      variables: { input: toWorkerInput(data) },
    });

    return response.createWorker;
  }

  public async update(id: Worker["id"], data: Worker) {
    if (!id) {
      throw new Error("No worker selected");
    }
    const response = await requestGraphQL({
      document: UpdateWorkerDocument,
      operationName: "UpdateWorker",
      variables: { id, input: toWorkerInput(data) },
    });

    return response.updateWorker;
  }
}
