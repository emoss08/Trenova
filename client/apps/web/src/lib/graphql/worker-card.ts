import {
  DeskWorkerCardDocument,
  DeskWorkerCredentialsDocument,
  DeskWorkerHosDocument,
  DeskWorkerStandingDocument,
  type DeskWorkerCardQuery,
  type DeskWorkerCredentialsQuery,
  type DeskWorkerHosQuery,
  type DeskWorkerStandingQuery,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type DeskWorker = NonNullable<DeskWorkerCardQuery["worker"]>;
export type DeskWorkerHos = {
  state: NonNullable<DeskWorkerHosQuery["workerHosState"]> | null;
  violations: DeskWorkerHosQuery["workerHosViolations"];
};
export type DeskWorkerStanding = NonNullable<DeskWorkerStandingQuery["workerOverview"]>;
export type DeskWorkerCredential =
  DeskWorkerCredentialsQuery["workerCredentialSummary"]["items"][number];

/**
 * One worker as the Desk's card reads it. A worker the server does not return
 * is an error rather than an empty card, so the card says it could not be read.
 */
export async function fetchDeskWorker(
  id: string,
  options?: { signal?: AbortSignal },
): Promise<DeskWorker> {
  const data = await requestGraphQL({
    document: DeskWorkerCardDocument,
    operationName: "DeskWorkerCard",
    variables: { id },
    signal: options?.signal,
  });
  if (!data.worker) {
    throw new Error("Worker not found");
  }
  return data.worker;
}

/** The driver's clocks as telematics last reported them, and their latest violations. */
export async function fetchWorkerHos(
  workerId: string,
  violations: number,
  options?: { signal?: AbortSignal },
): Promise<DeskWorkerHos> {
  const data = await requestGraphQL({
    document: DeskWorkerHosDocument,
    operationName: "DeskWorkerHos",
    variables: { workerId, violations },
    signal: options?.signal,
  });
  return { state: data.workerHosState ?? null, violations: data.workerHosViolations };
}

/** Whether the worker is in good standing, and what stands against them, worst first. */
export async function fetchWorkerStanding(
  workerId: string,
  options?: { signal?: AbortSignal },
): Promise<DeskWorkerStanding> {
  const data = await requestGraphQL({
    document: DeskWorkerStandingDocument,
    operationName: "DeskWorkerStanding",
    variables: { workerId },
    signal: options?.signal,
  });
  return data.workerOverview;
}

/** Every credential the worker holds or should, each with how it stands. */
export async function fetchWorkerCredentials(
  workerId: string,
  options?: { signal?: AbortSignal },
): Promise<DeskWorkerCredential[]> {
  const data = await requestGraphQL({
    document: DeskWorkerCredentialsDocument,
    operationName: "DeskWorkerCredentials",
    variables: { workerId },
    signal: options?.signal,
  });
  return data.workerCredentialSummary.items;
}
