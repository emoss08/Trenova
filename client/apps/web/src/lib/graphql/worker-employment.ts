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
  WorkerEmploymentEventsQuery,
  WorkerEmploymentEventsQueryVariables,
} from "@trenova/graphql/generated/graphql";
import {
  AmendWorkerEmploymentEventDocument,
  PreviewWorkerEmploymentEventDocument,
  RecordWorkerEmploymentEventDocument,
  WorkerEmploymentEventOptionsDocument,
  WorkerEmploymentEventsDocument,
} from "@trenova/graphql/generated/documents/worker-employment/operations";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

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
