import type {
  CloseLeaveCaseMutation,
  CloseLeaveCaseMutationVariables,
  DecideLeaveCaseInput,
  DecideLeaveCaseMutation,
  DecideLeaveCaseMutationVariables,
  DeleteLeaveDayMutation,
  DeleteLeaveDayMutationVariables,
  LeaveControlQuery,
  LeaveControlQueryVariables,
  OpenLeaveCaseInput,
  OpenLeaveCaseMutation,
  OpenLeaveCaseMutationVariables,
  RecordLeaveCertificationInput,
  RecordLeaveCertificationMutation,
  RecordLeaveCertificationMutationVariables,
  RecordLeaveDayInput,
  RecordLeaveDayMutation,
  RecordLeaveDayMutationVariables,
  RequestLeaveCertificationMutation,
  RequestLeaveCertificationMutationVariables,
  UpdateLeaveCaseInput,
  UpdateLeaveCaseMutation,
  UpdateLeaveCaseMutationVariables,
  UpdateLeaveControlMutation,
  UpdateLeaveControlMutationVariables,
  WorkerLeaveFileQuery,
  WorkerLeaveFileQueryVariables,
} from "@trenova/graphql/generated/graphql";
import {
  CloseLeaveCaseDocument,
  DecideLeaveCaseDocument,
  DeleteLeaveDayDocument,
  LeaveControlDocument,
  OpenLeaveCaseDocument,
  RecordLeaveCertificationDocument,
  RecordLeaveDayDocument,
  RequestLeaveCertificationDocument,
  UpdateLeaveCaseDocument,
  UpdateLeaveControlDocument,
  WorkerLeaveFileDocument,
} from "@trenova/graphql/generated/documents/worker-leave/operations";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type { LeaveControl } from "@trenova/shared/types/worker-leave";

export type LeaveEntitlement = WorkerLeaveFileQuery["workerLeaveEntitlement"];
export type LeaveCaseRow = WorkerLeaveFileQuery["workerLeaveCases"][number];
export type LeaveEntryRow = LeaveCaseRow["entries"][number];

/**
 * A worker's whole leave picture. The balance and the cases travel together
 * because the balance is only meaningful beside the days it was drawn down by.
 */
export type WorkerLeaveFile = {
  entitlement: LeaveEntitlement;
  cases: LeaveCaseRow[];
};

export const WORKER_LEAVE_KEY = "worker-leave";

export async function fetchWorkerLeaveFile(
  workerId: string,
  options?: { signal?: AbortSignal },
): Promise<WorkerLeaveFile> {
  const data = await requestGraphQL<WorkerLeaveFileQuery, WorkerLeaveFileQueryVariables>({
    document: WorkerLeaveFileDocument,
    operationName: "WorkerLeaveFile",
    variables: { workerId },
    signal: options?.signal,
  });
  return { entitlement: data.workerLeaveEntitlement, cases: data.workerLeaveCases };
}

export async function fetchLeaveControl(options?: { signal?: AbortSignal }): Promise<LeaveControl> {
  const data = await requestGraphQL<LeaveControlQuery, LeaveControlQueryVariables>({
    document: LeaveControlDocument,
    operationName: "LeaveControl",
    variables: {},
    signal: options?.signal,
  });
  return data.leaveControl;
}

export async function openLeaveCase(input: OpenLeaveCaseInput) {
  const data = await requestGraphQL<OpenLeaveCaseMutation, OpenLeaveCaseMutationVariables>({
    document: OpenLeaveCaseDocument,
    operationName: "OpenLeaveCase",
    variables: { input },
  });
  return data.openLeaveCase;
}

export async function updateLeaveCase(input: UpdateLeaveCaseInput) {
  const data = await requestGraphQL<UpdateLeaveCaseMutation, UpdateLeaveCaseMutationVariables>({
    document: UpdateLeaveCaseDocument,
    operationName: "UpdateLeaveCase",
    variables: { input },
  });
  return data.updateLeaveCase;
}

export async function decideLeaveCase(input: DecideLeaveCaseInput) {
  const data = await requestGraphQL<DecideLeaveCaseMutation, DecideLeaveCaseMutationVariables>({
    document: DecideLeaveCaseDocument,
    operationName: "DecideLeaveCase",
    variables: { input },
  });
  return data.decideLeaveCase;
}

export async function closeLeaveCase(id: string) {
  const data = await requestGraphQL<CloseLeaveCaseMutation, CloseLeaveCaseMutationVariables>({
    document: CloseLeaveCaseDocument,
    operationName: "CloseLeaveCase",
    variables: { id },
  });
  return data.closeLeaveCase;
}

export async function requestLeaveCertification(caseId: string) {
  const data = await requestGraphQL<
    RequestLeaveCertificationMutation,
    RequestLeaveCertificationMutationVariables
  >({
    document: RequestLeaveCertificationDocument,
    operationName: "RequestLeaveCertification",
    variables: { caseId, dueAt: null },
  });
  return data.requestLeaveCertification;
}

export async function recordLeaveCertification(input: RecordLeaveCertificationInput) {
  const data = await requestGraphQL<
    RecordLeaveCertificationMutation,
    RecordLeaveCertificationMutationVariables
  >({
    document: RecordLeaveCertificationDocument,
    operationName: "RecordLeaveCertification",
    variables: { input },
  });
  return data.recordLeaveCertification;
}

export async function recordLeaveDay(input: RecordLeaveDayInput) {
  const data = await requestGraphQL<RecordLeaveDayMutation, RecordLeaveDayMutationVariables>({
    document: RecordLeaveDayDocument,
    operationName: "RecordLeaveDay",
    variables: { input },
  });
  return data.recordLeaveDay;
}

export async function deleteLeaveDay(id: string): Promise<boolean> {
  const data = await requestGraphQL<DeleteLeaveDayMutation, DeleteLeaveDayMutationVariables>({
    document: DeleteLeaveDayDocument,
    operationName: "DeleteLeaveDay",
    variables: { id },
  });
  return data.deleteLeaveDay;
}

/**
 * Saves the organisation's leave settings and returns them as saved. It sends the version the
 * form read, so a save over someone else's newer one is refused as a conflict instead of
 * silently overwriting it; the server stores the settings under a new version.
 */
export async function updateLeaveControl(record: LeaveControl): Promise<LeaveControl> {
  const data = await requestGraphQL<
    UpdateLeaveControlMutation,
    UpdateLeaveControlMutationVariables
  >({
    document: UpdateLeaveControlDocument,
    operationName: "UpdateLeaveControl",
    variables: {
      input: {
        version: record.version,
        measurementMethod: record.measurementMethod,
        entitlementWeeks: record.entitlementWeeks,
        militaryCaregiverWeeks: record.militaryCaregiverWeeks,
        workweekHours: record.workweekHours,
        eligibilityMonths: record.eligibilityMonths,
        eligibilityHours: record.eligibilityHours,
        certificationDueDays: record.certificationDueDays,
      },
    },
  });
  return { ...record, version: data.updateLeaveControl.version };
}
