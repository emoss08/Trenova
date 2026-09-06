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
  WorkerDisciplinaryActionsDocument,
  WorkerDisciplinaryLadderDocument,
  WorkerRecognitionsDocument,
  WorkerSafetyEventsDocument,
  WorkerSafetyScorecardDocument,
  type CloseWorkerSafetyEventMutation,
  type CloseWorkerSafetyEventMutationVariables,
  type CreateWorkerSafetyEventMutation,
  type CreateWorkerSafetyEventMutationVariables,
  type DefaultSafetyPointsQuery,
  type DefaultSafetyPointsQueryVariables,
  type DeleteWorkerRecognitionMutation,
  type DeleteWorkerRecognitionMutationVariables,
  type DeleteWorkerSafetyEventMutation,
  type DeleteWorkerSafetyEventMutationVariables,
  type GiveWorkerRecognitionMutation,
  type GiveWorkerRecognitionMutationVariables,
  type IssueDisciplinaryActionInput,
  type IssueDisciplinaryActionMutation,
  type IssueDisciplinaryActionMutationVariables,
  type ReopenWorkerSafetyEventMutation,
  type ReopenWorkerSafetyEventMutationVariables,
  type RescindDisciplinaryActionInput,
  type RescindDisciplinaryActionMutation,
  type RescindDisciplinaryActionMutationVariables,
  type ReviewWorkerSafetyEventMutation,
  type ReviewWorkerSafetyEventMutationVariables,
  type SafetyEventStatusInput,
  type SafetyScorecardFieldsFragment,
  type UpdateWorkerSafetyEventInput,
  type UpdateWorkerSafetyEventMutation,
  type UpdateWorkerSafetyEventMutationVariables,
  type WorkerDisciplinaryActionFieldsFragment,
  type WorkerDisciplinaryActionsQuery,
  type WorkerDisciplinaryActionsQueryVariables,
  type WorkerDisciplinaryLadderQuery,
  type WorkerDisciplinaryLadderQueryVariables,
  type WorkerRecognitionFieldsFragment,
  type WorkerRecognitionInput,
  type WorkerRecognitionsQuery,
  type WorkerRecognitionsQueryVariables,
  type WorkerSafetyEventFieldsFragment,
  type WorkerSafetyEventInput,
  type WorkerSafetyEventsQuery,
  type WorkerSafetyEventsQueryVariables,
  type WorkerSafetyScorecardQuery,
  type WorkerSafetyScorecardQueryVariables,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

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

export async function fetchWorkerSafetyEvents(
  workerId: string,
  options?: { signal?: AbortSignal },
): Promise<WorkerSafetyEventRow[]> {
  const data = await requestGraphQL<WorkerSafetyEventsQuery, WorkerSafetyEventsQueryVariables>({
    document: WorkerSafetyEventsDocument,
    operationName: "WorkerSafetyEvents",
    variables: { workerId },
    signal: options?.signal,
  });
  return data.workerSafetyEvents as WorkerSafetyEventRow[];
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

export async function fetchWorkerDisciplinaryActions(
  workerId: string,
  options?: { signal?: AbortSignal },
): Promise<WorkerDisciplinaryActionRow[]> {
  const data = await requestGraphQL<
    WorkerDisciplinaryActionsQuery,
    WorkerDisciplinaryActionsQueryVariables
  >({
    document: WorkerDisciplinaryActionsDocument,
    operationName: "WorkerDisciplinaryActions",
    variables: { workerId },
    signal: options?.signal,
  });
  return data.workerDisciplinaryActions as WorkerDisciplinaryActionRow[];
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
