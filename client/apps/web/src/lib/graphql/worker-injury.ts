import type {
  CertifyOshaSummaryMutation,
  CertifyOshaSummaryMutationVariables,
  DeleteWorkerInjuryMutation,
  DeleteWorkerInjuryMutationVariables,
  OshaLogQuery,
  OshaLogQueryVariables,
  OshaSummariesQuery,
  OshaSummariesQueryVariables,
  RecordWorkerInjuryInput,
  RecordWorkerInjuryMutation,
  RecordWorkerInjuryMutationVariables,
  SaveOshaSummaryInput,
  SaveOshaSummaryMutation,
  SaveOshaSummaryMutationVariables,
  UncertifyOshaSummaryMutation,
  UncertifyOshaSummaryMutationVariables,
  UpdateWorkerInjuryInput,
  UpdateWorkerInjuryMutation,
  UpdateWorkerInjuryMutationVariables,
  WorkerInjuriesQuery,
  WorkerInjuriesQueryVariables,
  WorkerInjuryQuery,
  WorkerInjuryQueryVariables,
} from "@trenova/graphql/generated/graphql";
import {
  CertifyOshaSummaryDocument,
  DeleteWorkerInjuryDocument,
  OshaLogDocument,
  OshaSummariesDocument,
  RecordWorkerInjuryDocument,
  SaveOshaSummaryDocument,
  UncertifyOshaSummaryDocument,
  UpdateWorkerInjuryDocument,
  WorkerInjuriesDocument,
  WorkerInjuryDocument,
} from "@trenova/graphql/generated/documents/worker-injury/operations";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type { Connection } from "./connection";

export type WorkerInjuryRow = WorkerInjuriesQuery["workerInjuries"]["edges"][number]["node"];
export type OshaLog = OshaLogQuery["oshaLog"];
export type OshaLogCase = OshaLog["cases"][number];
export type OshaSummary = NonNullable<OshaLog["summary"]>;
export type OshaSummaryRow = OshaSummariesQuery["oshaSummaries"][number];

export const WORKER_INJURIES_KEY = "worker-injuries";
export const OSHA_LOG_KEY = "osha-log";
export const OSHA_SUMMARIES_KEY = "osha-summaries";

/** A page of a worker's injury and illness cases, newest first, as the server's connection. */
export async function fetchWorkerInjuries(
  variables: WorkerInjuriesQueryVariables,
  options?: { signal?: AbortSignal },
): Promise<Connection<WorkerInjuryRow>> {
  const data = await requestGraphQL<WorkerInjuriesQuery, WorkerInjuriesQueryVariables>({
    document: WorkerInjuriesDocument,
    operationName: "WorkerInjuries",
    variables: { ...variables, includeTotalCount: !variables.after },
    signal: options?.signal,
  });
  return data.workerInjuries as Connection<WorkerInjuryRow>;
}

/** One injury or illness case by id: a deep link to a case outside the pages already read. */
export async function fetchWorkerInjury(
  id: string,
  options?: { signal?: AbortSignal },
): Promise<WorkerInjuryRow> {
  const data = await requestGraphQL<WorkerInjuryQuery, WorkerInjuryQueryVariables>({
    document: WorkerInjuryDocument,
    operationName: "WorkerInjury",
    variables: { id },
    signal: options?.signal,
  });
  return data.workerInjury;
}

export async function fetchOshaLog(
  year?: number,
  options?: { signal?: AbortSignal },
): Promise<OshaLog> {
  const data = await requestGraphQL<OshaLogQuery, OshaLogQueryVariables>({
    document: OshaLogDocument,
    operationName: "OshaLog",
    variables: { year: year ?? null },
    signal: options?.signal,
  });
  return data.oshaLog;
}

export async function fetchOshaSummaries(options?: {
  signal?: AbortSignal;
}): Promise<OshaSummaryRow[]> {
  const data = await requestGraphQL<OshaSummariesQuery, OshaSummariesQueryVariables>({
    document: OshaSummariesDocument,
    operationName: "OshaSummaries",
    variables: {},
    signal: options?.signal,
  });
  return data.oshaSummaries;
}

export async function recordWorkerInjury(input: RecordWorkerInjuryInput) {
  const data = await requestGraphQL<
    RecordWorkerInjuryMutation,
    RecordWorkerInjuryMutationVariables
  >({
    document: RecordWorkerInjuryDocument,
    operationName: "RecordWorkerInjury",
    variables: { input },
  });
  return data.recordWorkerInjury;
}

export async function updateWorkerInjury(input: UpdateWorkerInjuryInput) {
  const data = await requestGraphQL<
    UpdateWorkerInjuryMutation,
    UpdateWorkerInjuryMutationVariables
  >({
    document: UpdateWorkerInjuryDocument,
    operationName: "UpdateWorkerInjury",
    variables: { input },
  });
  return data.updateWorkerInjury;
}

export async function deleteWorkerInjury(id: string): Promise<boolean> {
  const data = await requestGraphQL<
    DeleteWorkerInjuryMutation,
    DeleteWorkerInjuryMutationVariables
  >({
    document: DeleteWorkerInjuryDocument,
    operationName: "DeleteWorkerInjury",
    variables: { id },
  });
  return data.deleteWorkerInjury;
}

export async function saveOshaSummary(input: SaveOshaSummaryInput) {
  const data = await requestGraphQL<SaveOshaSummaryMutation, SaveOshaSummaryMutationVariables>({
    document: SaveOshaSummaryDocument,
    operationName: "SaveOshaSummary",
    variables: { input },
  });
  return data.saveOshaSummary;
}

export async function certifyOshaSummary(year: number) {
  const data = await requestGraphQL<
    CertifyOshaSummaryMutation,
    CertifyOshaSummaryMutationVariables
  >({
    document: CertifyOshaSummaryDocument,
    operationName: "CertifyOshaSummary",
    variables: { year },
  });
  return data.certifyOshaSummary;
}

export async function uncertifyOshaSummary(year: number) {
  const data = await requestGraphQL<
    UncertifyOshaSummaryMutation,
    UncertifyOshaSummaryMutationVariables
  >({
    document: UncertifyOshaSummaryDocument,
    operationName: "UncertifyOshaSummary",
    variables: { year },
  });
  return data.uncertifyOshaSummary;
}
