import type {
  CloseWorkerSafetyEventMutation,
  CloseWorkerSafetyEventMutationVariables,
  CreateWorkerSafetyEventMutation,
  CreateWorkerSafetyEventMutationVariables,
  DefaultSafetyPointsQuery,
  DefaultSafetyPointsQueryVariables,
  DeleteWorkerRecognitionMutation,
  DeleteWorkerRecognitionMutationVariables,
  DeleteWorkerSafetyEventMutation,
  DeleteWorkerSafetyEventMutationVariables,
  GiveWorkerRecognitionMutation,
  GiveWorkerRecognitionMutationVariables,
  IssueDisciplinaryActionInput,
  IssueDisciplinaryActionMutation,
  IssueDisciplinaryActionMutationVariables,
  ReopenWorkerSafetyEventMutation,
  ReopenWorkerSafetyEventMutationVariables,
  RescindDisciplinaryActionInput,
  RescindDisciplinaryActionMutation,
  RescindDisciplinaryActionMutationVariables,
  ReviewWorkerSafetyEventMutation,
  ReviewWorkerSafetyEventMutationVariables,
  SafetyEventStatusInput,
  SafetyScorecardFieldsFragment,
  UpdateWorkerSafetyEventInput,
  UpdateWorkerSafetyEventMutation,
  UpdateWorkerSafetyEventMutationVariables,
  WorkerDisciplinaryActionFieldsFragment,
  WorkerDisciplinaryActionQuery,
  WorkerDisciplinaryActionQueryVariables,
  WorkerDisciplinaryActionsQuery,
  WorkerDisciplinaryActionsQueryVariables,
  WorkerDisciplinaryLadderQuery,
  WorkerDisciplinaryLadderQueryVariables,
  WorkerRecognitionFieldsFragment,
  WorkerRecognitionInput,
  WorkerRecognitionsQuery,
  WorkerRecognitionsQueryVariables,
  WorkerSafetyEventFieldsFragment,
  WorkerSafetyEventInput,
  WorkerSafetyEventQuery,
  WorkerSafetyEventQueryVariables,
  WorkerSafetyEventsQuery,
  WorkerSafetyEventsQueryVariables,
  WorkerSafetyScorecardQuery,
  WorkerSafetyScorecardQueryVariables,
} from "@trenova/graphql/generated/graphql";
import {
  CloseWorkerSafetyEventDocument,
  CreateWorkerSafetyEventDocument,
  DefaultSafetyPointsDocument,
  DeleteWorkerRecognitionDocument,
  DeleteWorkerSafetyEventDocument,
  GiveWorkerRecognitionDocument,
  IssueDisciplinaryActionDocument,
  ReopenWorkerSafetyEventDocument,
  RescindDisciplinaryActionDocument,
  ReviewWorkerSafetyEventDocument,
  UpdateWorkerSafetyEventDocument,
  WorkerDisciplinaryActionDocument,
  WorkerDisciplinaryActionsDocument,
  WorkerDisciplinaryLadderDocument,
  WorkerRecognitionsDocument,
  WorkerSafetyEventDocument,
  WorkerSafetyEventsDocument,
  WorkerSafetyScorecardDocument,
} from "@trenova/graphql/generated/documents/worker-safety/operations";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type { Connection } from "./connection";

export type WorkerSafetyEventRow = WorkerSafetyEventFieldsFragment;
export type WorkerDisciplinaryActionRow = WorkerDisciplinaryActionFieldsFragment;
export type WorkerRecognitionRow = WorkerRecognitionFieldsFragment;
export type SafetyScorecard = SafetyScorecardFieldsFragment;
export type DisciplinaryLadder = WorkerDisciplinaryLadderQuery["workerDisciplinaryLadder"];

export const WORKER_SAFETY_EVENTS_KEY = "worker-safety-events";
export const WORKER_SAFETY_SCORECARD_KEY = "worker-safety-scorecard";
export const WORKER_DISCIPLINARY_ACTIONS_KEY = "worker-disciplinary-actions";
export const WORKER_DISCIPLINARY_LADDER_KEY = "worker-disciplinary-ladder";
export const WORKER_RECOGNITIONS_KEY = "worker-recognitions";

/**
 * A worker's safety events, newest first: the whole record up to the server's cap (the newest
 * 100), read in one go because the Desk's tools read it whole too.
 */
export async function fetchWorkerSafetyEvents(
  workerId: string,
  filters: { openOnly?: boolean } = {},
  options?: { signal?: AbortSignal },
): Promise<WorkerSafetyEventRow[]> {
  const data = await requestGraphQL<WorkerSafetyEventsQuery, WorkerSafetyEventsQueryVariables>({
    document: WorkerSafetyEventsDocument,
    operationName: "WorkerSafetyEvents",
    variables: { workerId, openOnly: filters.openOnly || undefined },
    signal: options?.signal,
  });
  return data.workerSafetyEvents as WorkerSafetyEventRow[];
}

/** One safety event by id: a deep link to an event outside the newest 100. */
export async function fetchWorkerSafetyEvent(
  id: string,
  options?: { signal?: AbortSignal },
): Promise<WorkerSafetyEventRow> {
  const data = await requestGraphQL<WorkerSafetyEventQuery, WorkerSafetyEventQueryVariables>({
    document: WorkerSafetyEventDocument,
    operationName: "WorkerSafetyEvent",
    variables: { id },
    signal: options?.signal,
  });
  return data.workerSafetyEvent as WorkerSafetyEventRow;
}

export async function fetchWorkerSafetyScorecard(
  workerId: string,
  options?: { signal?: AbortSignal },
): Promise<SafetyScorecard> {
  const data = await requestGraphQL<
    WorkerSafetyScorecardQuery,
    WorkerSafetyScorecardQueryVariables
  >({
    document: WorkerSafetyScorecardDocument,
    operationName: "WorkerSafetyScorecard",
    variables: { workerId },
    signal: options?.signal,
  });
  return data.workerSafetyScorecard as SafetyScorecard;
}

/** A page of a worker's disciplinary actions, newest first, as the server's connection. */
export async function fetchWorkerDisciplinaryActions(
  variables: WorkerDisciplinaryActionsQueryVariables,
  options?: { signal?: AbortSignal },
): Promise<Connection<WorkerDisciplinaryActionRow>> {
  const data = await requestGraphQL<
    WorkerDisciplinaryActionsQuery,
    WorkerDisciplinaryActionsQueryVariables
  >({
    document: WorkerDisciplinaryActionsDocument,
    operationName: "WorkerDisciplinaryActions",
    variables: { ...variables, includeTotalCount: !variables.after },
    signal: options?.signal,
  });
  return data.workerDisciplinaryActions as Connection<WorkerDisciplinaryActionRow>;
}

/** One disciplinary action by id: a deep link to an action outside the pages already read. */
export async function fetchWorkerDisciplinaryAction(
  id: string,
  options?: { signal?: AbortSignal },
): Promise<WorkerDisciplinaryActionRow> {
  const data = await requestGraphQL<
    WorkerDisciplinaryActionQuery,
    WorkerDisciplinaryActionQueryVariables
  >({
    document: WorkerDisciplinaryActionDocument,
    operationName: "WorkerDisciplinaryAction",
    variables: { id },
    signal: options?.signal,
  });
  return data.workerDisciplinaryAction as WorkerDisciplinaryActionRow;
}

export async function fetchWorkerDisciplinaryLadder(
  workerId: string,
  options?: { signal?: AbortSignal },
): Promise<DisciplinaryLadder> {
  const data = await requestGraphQL<
    WorkerDisciplinaryLadderQuery,
    WorkerDisciplinaryLadderQueryVariables
  >({
    document: WorkerDisciplinaryLadderDocument,
    operationName: "WorkerDisciplinaryLadder",
    variables: { workerId },
    signal: options?.signal,
  });
  return data.workerDisciplinaryLadder;
}

/** A worker's recognitions, newest first, up to the server's cap (the newest 100). */
export async function fetchWorkerRecognitions(
  workerId: string,
  options?: { signal?: AbortSignal },
): Promise<WorkerRecognitionRow[]> {
  const data = await requestGraphQL<WorkerRecognitionsQuery, WorkerRecognitionsQueryVariables>({
    document: WorkerRecognitionsDocument,
    operationName: "WorkerRecognitions",
    variables: { workerId },
    signal: options?.signal,
  });
  return data.workerRecognitions as WorkerRecognitionRow[];
}

export async function fetchDefaultSafetyPoints(
  variables: DefaultSafetyPointsQueryVariables,
  options?: { signal?: AbortSignal },
): Promise<number> {
  const data = await requestGraphQL<DefaultSafetyPointsQuery, DefaultSafetyPointsQueryVariables>({
    document: DefaultSafetyPointsDocument,
    operationName: "DefaultSafetyPoints",
    variables,
    signal: options?.signal,
  });
  return data.defaultSafetyPoints;
}

export async function createWorkerSafetyEvent(
  input: WorkerSafetyEventInput,
): Promise<WorkerSafetyEventRow> {
  const data = await requestGraphQL<
    CreateWorkerSafetyEventMutation,
    CreateWorkerSafetyEventMutationVariables
  >({
    document: CreateWorkerSafetyEventDocument,
    operationName: "CreateWorkerSafetyEvent",
    variables: { input },
  });
  return data.createWorkerSafetyEvent as WorkerSafetyEventRow;
}

export async function updateWorkerSafetyEvent(
  input: UpdateWorkerSafetyEventInput,
): Promise<WorkerSafetyEventRow> {
  const data = await requestGraphQL<
    UpdateWorkerSafetyEventMutation,
    UpdateWorkerSafetyEventMutationVariables
  >({
    document: UpdateWorkerSafetyEventDocument,
    operationName: "UpdateWorkerSafetyEvent",
    variables: { input },
  });
  return data.updateWorkerSafetyEvent as WorkerSafetyEventRow;
}

export async function closeWorkerSafetyEvent(
  input: SafetyEventStatusInput,
): Promise<WorkerSafetyEventRow> {
  const data = await requestGraphQL<
    CloseWorkerSafetyEventMutation,
    CloseWorkerSafetyEventMutationVariables
  >({
    document: CloseWorkerSafetyEventDocument,
    operationName: "CloseWorkerSafetyEvent",
    variables: { input },
  });
  return data.closeWorkerSafetyEvent as WorkerSafetyEventRow;
}

export async function reviewWorkerSafetyEvent(
  input: SafetyEventStatusInput,
): Promise<WorkerSafetyEventRow> {
  const data = await requestGraphQL<
    ReviewWorkerSafetyEventMutation,
    ReviewWorkerSafetyEventMutationVariables
  >({
    document: ReviewWorkerSafetyEventDocument,
    operationName: "ReviewWorkerSafetyEvent",
    variables: { input },
  });
  return data.reviewWorkerSafetyEvent as WorkerSafetyEventRow;
}

export async function reopenWorkerSafetyEvent(
  input: SafetyEventStatusInput,
): Promise<WorkerSafetyEventRow> {
  const data = await requestGraphQL<
    ReopenWorkerSafetyEventMutation,
    ReopenWorkerSafetyEventMutationVariables
  >({
    document: ReopenWorkerSafetyEventDocument,
    operationName: "ReopenWorkerSafetyEvent",
    variables: { input },
  });
  return data.reopenWorkerSafetyEvent as WorkerSafetyEventRow;
}

export async function deleteWorkerSafetyEvent(id: string): Promise<boolean> {
  const data = await requestGraphQL<
    DeleteWorkerSafetyEventMutation,
    DeleteWorkerSafetyEventMutationVariables
  >({
    document: DeleteWorkerSafetyEventDocument,
    operationName: "DeleteWorkerSafetyEvent",
    variables: { id },
  });
  return data.deleteWorkerSafetyEvent;
}

export type IssueActionResult = IssueDisciplinaryActionMutation["issueDisciplinaryAction"];

export async function issueDisciplinaryAction(
  input: IssueDisciplinaryActionInput,
): Promise<IssueActionResult> {
  const data = await requestGraphQL<
    IssueDisciplinaryActionMutation,
    IssueDisciplinaryActionMutationVariables
  >({
    document: IssueDisciplinaryActionDocument,
    operationName: "IssueDisciplinaryAction",
    variables: { input },
  });
  return data.issueDisciplinaryAction;
}

export async function rescindDisciplinaryAction(
  input: RescindDisciplinaryActionInput,
): Promise<WorkerDisciplinaryActionRow> {
  const data = await requestGraphQL<
    RescindDisciplinaryActionMutation,
    RescindDisciplinaryActionMutationVariables
  >({
    document: RescindDisciplinaryActionDocument,
    operationName: "RescindDisciplinaryAction",
    variables: { input },
  });
  return data.rescindDisciplinaryAction as WorkerDisciplinaryActionRow;
}

export async function giveWorkerRecognition(
  input: WorkerRecognitionInput,
): Promise<WorkerRecognitionRow> {
  const data = await requestGraphQL<
    GiveWorkerRecognitionMutation,
    GiveWorkerRecognitionMutationVariables
  >({
    document: GiveWorkerRecognitionDocument,
    operationName: "GiveWorkerRecognition",
    variables: { input },
  });
  return data.giveWorkerRecognition as WorkerRecognitionRow;
}

export async function deleteWorkerRecognition(id: string): Promise<boolean> {
  const data = await requestGraphQL<
    DeleteWorkerRecognitionMutation,
    DeleteWorkerRecognitionMutationVariables
  >({
    document: DeleteWorkerRecognitionDocument,
    operationName: "DeleteWorkerRecognition",
    variables: { id },
  });
  return data.deleteWorkerRecognition;
}
