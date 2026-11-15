import type {
  AmendWorkerEmploymentEventInput,
  AmendWorkerEmploymentEventMutation,
  AmendWorkerEmploymentEventMutationVariables,
  RecordWorkerEmploymentEventInput,
  RecordWorkerEmploymentEventMutation,
  RecordWorkerEmploymentEventMutationVariables,
  WorkerEmploymentEventFieldsFragment,
  WorkerEmploymentEventKind,
  WorkerEmploymentEventsQuery,
  WorkerEmploymentEventsQueryVariables,
} from "@trenova/graphql/generated/graphql";
import {
  AmendWorkerEmploymentEventDocument,
  RecordWorkerEmploymentEventDocument,
  WorkerEmploymentEventsDocument,
} from "@trenova/graphql/generated/documents/worker-employment/operations";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type WorkerEmploymentEventRow = WorkerEmploymentEventFieldsFragment;
export type EmploymentCascade =
  RecordWorkerEmploymentEventMutation["recordWorkerEmploymentEvent"]["cascade"];
export type RecordEmploymentEventResult = {
  event: WorkerEmploymentEventRow;
  cascade: EmploymentCascade;
};

export const WORKER_EMPLOYMENT_EVENTS_KEY = "worker-employment-events";

export async function fetchWorkerEmploymentEvents(
  workerId: string,
  kinds?: WorkerEmploymentEventKind[],
  options?: { signal?: AbortSignal },
): Promise<WorkerEmploymentEventRow[]> {
  const data = await requestGraphQL<
    WorkerEmploymentEventsQuery,
    WorkerEmploymentEventsQueryVariables
  >({
    document: WorkerEmploymentEventsDocument,
    operationName: "WorkerEmploymentEvents",
    variables: { workerId, kinds: kinds && kinds.length > 0 ? kinds : undefined },
    signal: options?.signal,
  });
  return data.workerEmploymentEvents as WorkerEmploymentEventRow[];
}

export async function recordWorkerEmploymentEvent(
  input: RecordWorkerEmploymentEventInput,
): Promise<RecordEmploymentEventResult> {
  const data = await requestGraphQL<
    RecordWorkerEmploymentEventMutation,
    RecordWorkerEmploymentEventMutationVariables
  >({
    document: RecordWorkerEmploymentEventDocument,
    operationName: "RecordWorkerEmploymentEvent",
    variables: { input },
  });
  return {
    event: data.recordWorkerEmploymentEvent.event as WorkerEmploymentEventRow,
    cascade: data.recordWorkerEmploymentEvent.cascade,
  };
}

export async function amendWorkerEmploymentEvent(
  input: AmendWorkerEmploymentEventInput,
): Promise<WorkerEmploymentEventRow> {
  const data = await requestGraphQL<
    AmendWorkerEmploymentEventMutation,
    AmendWorkerEmploymentEventMutationVariables
  >({
    document: AmendWorkerEmploymentEventDocument,
    operationName: "AmendWorkerEmploymentEvent",
    variables: { input },
  });
  return data.amendWorkerEmploymentEvent as WorkerEmploymentEventRow;
}
