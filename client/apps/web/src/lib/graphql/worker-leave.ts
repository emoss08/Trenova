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
  type CloseLeaveCaseMutation,
  type CloseLeaveCaseMutationVariables,
  type DecideLeaveCaseInput,
  type DecideLeaveCaseMutation,
  type DecideLeaveCaseMutationVariables,
  type DeleteLeaveDayMutation,
  type DeleteLeaveDayMutationVariables,
  type LeaveControlQuery,
  type LeaveControlQueryVariables,
  type OpenLeaveCaseInput,
  type OpenLeaveCaseMutation,
  type OpenLeaveCaseMutationVariables,
  type RecordLeaveCertificationInput,
  type RecordLeaveCertificationMutation,
  type RecordLeaveCertificationMutationVariables,
  type RecordLeaveDayInput,
  type RecordLeaveDayMutation,
  type RecordLeaveDayMutationVariables,
  type RequestLeaveCertificationMutation,
  type RequestLeaveCertificationMutationVariables,
  type UpdateLeaveCaseInput,
  type UpdateLeaveCaseMutation,
  type UpdateLeaveCaseMutationVariables,
  type UpdateLeaveControlInput,
  type UpdateLeaveControlMutation,
  type UpdateLeaveControlMutationVariables,
  type WorkerLeaveFileQuery,
  type WorkerLeaveFileQueryVariables,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type LeaveEntitlement = WorkerLeaveFileQuery["workerLeaveEntitlement"];
export type LeaveCaseRow = WorkerLeaveFileQuery["workerLeaveCases"][number];
export type LeaveEntryRow = LeaveCaseRow["entries"][number];
export type LeaveControlRow = LeaveControlQuery["leaveControl"];

/**
 * A worker's whole leave picture. The balance and the cases travel together
 * because the balance is only meaningful beside the days it was drawn down by.
 */
export type WorkerLeaveFile = {
  entitlement: LeaveEntitlement;
  cases: LeaveCaseRow[];
};

export const WORKER_LEAVE_KEY = "worker-leave";
export const LEAVE_CONTROL_KEY = "leave-control";

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

export async function fetchLeaveControl(options?: {
  signal?: AbortSignal;
}): Promise<LeaveControlRow> {
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

export async function updateLeaveControl(input: UpdateLeaveControlInput) {
  const data = await requestGraphQL<
    UpdateLeaveControlMutation,
    UpdateLeaveControlMutationVariables
  >({
    document: UpdateLeaveControlDocument,
    operationName: "UpdateLeaveControl",
    variables: { input },
  });
  return data.updateLeaveControl;
}
