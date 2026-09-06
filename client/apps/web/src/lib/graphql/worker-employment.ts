import {
  AmendWorkerEmploymentEventDocument,
  RecordWorkerEmploymentEventDocument,
  WorkerEmploymentEventsDocument,
  type AmendWorkerEmploymentEventInput,
  type AmendWorkerEmploymentEventMutation,
  type AmendWorkerEmploymentEventMutationVariables,
  type RecordWorkerEmploymentEventInput,
  type RecordWorkerEmploymentEventMutation,
  type RecordWorkerEmploymentEventMutationVariables,
  type WorkerEmploymentEventFieldsFragment,
  type WorkerEmploymentEventKind,
  type WorkerEmploymentEventsQuery,
  type WorkerEmploymentEventsQueryVariables,
} from "@trenova/graphql/generated/graphql";
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
