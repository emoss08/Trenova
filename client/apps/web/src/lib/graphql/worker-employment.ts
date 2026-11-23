import type {
  AmendWorkerEmploymentEventInput,
  AmendWorkerEmploymentEventMutation,
  AmendWorkerEmploymentEventMutationVariables,
  PreviewWorkerEmploymentEventQuery,
  PreviewWorkerEmploymentEventQueryVariables,
  RecordWorkerEmploymentEventInput,
  RecordWorkerEmploymentEventMutation,
  RecordWorkerEmploymentEventMutationVariables,
  WorkerEmploymentEffectFieldsFragment,
  WorkerEmploymentEventFieldsFragment,
  WorkerEmploymentEventKind,
  WorkerEmploymentEventOptionsQuery,
  WorkerEmploymentEventOptionsQueryVariables,
  WorkerEmploymentEventQuery,
  WorkerEmploymentEventQueryVariables,
  WorkerEmploymentEventsQuery,
  WorkerEmploymentEventsQueryVariables,
} from "@trenova/graphql/generated/graphql";
import {
  AmendWorkerEmploymentEventDocument,
  PreviewWorkerEmploymentEventDocument,
  RecordWorkerEmploymentEventDocument,
  WorkerEmploymentEventDocument,
  WorkerEmploymentEventFieldsFragmentDoc,
  WorkerEmploymentEventOptionsDocument,
  WorkerEmploymentEventsDocument,
} from "@trenova/graphql/generated/documents/worker-employment/operations";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { unmaskConnection, type Connection } from "./connection";

export type WorkerEmploymentEventRow = WorkerEmploymentEventFieldsFragment;
/** One consequence of an event beyond the event itself, as previewed or as carried out. */
export type EmploymentEffect = WorkerEmploymentEffectFieldsFragment;
export type RecordEmploymentEventResult = {
  event: WorkerEmploymentEventRow;
  effects: EmploymentEffect[];
};
export type EmploymentEventOptions =
  WorkerEmploymentEventOptionsQuery["workerEmploymentEventOptions"];
type RawEmploymentEventPreview = PreviewWorkerEmploymentEventQuery["previewWorkerEmploymentEvent"];
export type EmploymentEventPreview = Omit<RawEmploymentEventPreview, "effects"> & {
  effects: EmploymentEffect[];
};

export const WORKER_EMPLOYMENT_EVENTS_KEY = "worker-employment-events";
export const WORKER_EMPLOYMENT_OPTIONS_KEY = "worker-employment-event-options";
export const WORKER_EMPLOYMENT_PREVIEW_KEY = "worker-employment-event-preview";

/** A worker's employment events, newest first, a page at a time; `kinds` narrows them. */
export async function fetchWorkerEmploymentEvents(
  input: {
    workerId: string;
    kinds?: readonly WorkerEmploymentEventKind[];
    first: number;
    after?: string | null;
  },
  options?: { signal?: AbortSignal },
): Promise<Connection<WorkerEmploymentEventRow>> {
  const data = await requestGraphQL<
    WorkerEmploymentEventsQuery,
    WorkerEmploymentEventsQueryVariables
  >({
    document: WorkerEmploymentEventsDocument,
    operationName: "WorkerEmploymentEvents",
    variables: {
      workerId: input.workerId,
      kinds: input.kinds && input.kinds.length > 0 ? [...input.kinds] : undefined,
      first: input.first,
      after: input.after ?? undefined,
      includeTotalCount: !input.after,
    },
    signal: options?.signal,
  });
  return unmaskConnection(WorkerEmploymentEventFieldsFragmentDoc, data.workerEmploymentEvents);
}

/** One employment event by id: a deep link to an event outside the pages already read. */
export async function fetchWorkerEmploymentEvent(
  id: string,
  options?: { signal?: AbortSignal },
): Promise<WorkerEmploymentEventRow> {
  const data = await requestGraphQL<
    WorkerEmploymentEventQuery,
    WorkerEmploymentEventQueryVariables
  >({
    document: WorkerEmploymentEventDocument,
    operationName: "WorkerEmploymentEvent",
    variables: { id },
    signal: options?.signal,
  });
  return data.workerEmploymentEvent as WorkerEmploymentEventRow;
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
    effects: data.recordWorkerEmploymentEvent.effects as EmploymentEffect[],
  };
}

/** The event kinds the worker may move through now, and which of them need a reason. */
export async function fetchWorkerEmploymentEventOptions(
  workerId: string,
  options?: { signal?: AbortSignal },
): Promise<EmploymentEventOptions> {
  const data = await requestGraphQL<
    WorkerEmploymentEventOptionsQuery,
    WorkerEmploymentEventOptionsQueryVariables
  >({
    document: WorkerEmploymentEventOptionsDocument,
    operationName: "WorkerEmploymentEventOptions",
    variables: { workerId },
    signal: options?.signal,
  });
  return data.workerEmploymentEventOptions;
}

/** What recording the event would do, from the plan the write runs; writes nothing. */
export async function previewWorkerEmploymentEvent(
  input: RecordWorkerEmploymentEventInput,
  options?: { signal?: AbortSignal },
): Promise<EmploymentEventPreview> {
  const data = await requestGraphQL<
    PreviewWorkerEmploymentEventQuery,
    PreviewWorkerEmploymentEventQueryVariables
  >({
    document: PreviewWorkerEmploymentEventDocument,
    operationName: "PreviewWorkerEmploymentEvent",
    variables: { input },
    signal: options?.signal,
  });
  const preview = data.previewWorkerEmploymentEvent;
  return { ...preview, effects: preview.effects as EmploymentEffect[] };
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
