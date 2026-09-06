import {
  WorkerOverviewDocument,
  WorkerRosterAttentionDocument,
  type WorkerOverviewQuery,
  type WorkerOverviewQueryVariables,
  type WorkerRosterAttentionQuery,
  type WorkerRosterAttentionQueryVariables,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type WorkerOverview = WorkerOverviewQuery["workerOverview"];
export type WorkerConcern = WorkerOverview["concerns"][number];
export type OverviewCredentials = NonNullable<WorkerOverview["credentials"]>;
export type OverviewTraining = NonNullable<WorkerOverview["training"]>;
export type OverviewSafety = NonNullable<WorkerOverview["safety"]>;
export type OverviewChecklist = NonNullable<WorkerOverview["checklist"]>;
export type OverviewPTOBalance = NonNullable<WorkerOverview["pto"]>[number];
export type OverviewReview = NonNullable<WorkerOverview["lastReview"]>;

export const WORKER_OVERVIEW_KEY = "worker-overview";

export async function fetchWorkerOverview(
  workerId: string,
  options?: { signal?: AbortSignal },
): Promise<WorkerOverview> {
  const data = await requestGraphQL<WorkerOverviewQuery, WorkerOverviewQueryVariables>({
    document: WorkerOverviewDocument,
    operationName: "WorkerOverview",
    variables: { workerId },
    signal: options?.signal,
  });
  return data.workerOverview;
}

export type WorkerRosterAttention = WorkerRosterAttentionQuery["workerRosterAttention"];

export const WORKER_ROSTER_ATTENTION_KEY = "worker-roster-attention";

export async function fetchWorkerRosterAttention(options?: {
  signal?: AbortSignal;
}): Promise<WorkerRosterAttention> {
  const data = await requestGraphQL<
    WorkerRosterAttentionQuery,
    WorkerRosterAttentionQueryVariables
  >({
    document: WorkerRosterAttentionDocument,
    operationName: "WorkerRosterAttention",
    variables: {},
    signal: options?.signal,
  });
  return data.workerRosterAttention;
}
