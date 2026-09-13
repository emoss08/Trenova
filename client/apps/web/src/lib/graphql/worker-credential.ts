import {
  ArchiveWorkerCredentialDocument,
  ArchiveWorkerCredentialTypeDocument,
  AttachWorkerCredentialDocumentDocument,
  CreateWorkerCredentialDocument,
  CreateWorkerCredentialTypeDocument,
  CredentialExpiryForecastDocument,
  RestoreWorkerCredentialTypeDocument,
  UpdateWorkerCredentialDocument,
  UpdateWorkerCredentialTypeDocument,
  VerifyWorkerCredentialDocument,
  WorkerCredentialSummaryDocument,
  WorkerCredentialTypeTableDocument,
  WorkerCredentialsDocument,
  type ArchiveWorkerCredentialInput,
  type ArchiveWorkerCredentialMutation,
  type ArchiveWorkerCredentialMutationVariables,
  type ArchiveWorkerCredentialTypeMutation,
  type ArchiveWorkerCredentialTypeMutationVariables,
  type AttachWorkerCredentialDocumentInput,
  type AttachWorkerCredentialDocumentMutation,
  type AttachWorkerCredentialDocumentMutationVariables,
  type CreateWorkerCredentialMutation,
  type CreateWorkerCredentialMutationVariables,
  type CreateWorkerCredentialTypeMutation,
  type CreateWorkerCredentialTypeMutationVariables,
  type CredentialExpiryForecastQuery,
  type CredentialExpiryForecastQueryVariables,
  type RestoreWorkerCredentialTypeMutation,
  type RestoreWorkerCredentialTypeMutationVariables,
  type UpdateWorkerCredentialInput,
  type UpdateWorkerCredentialMutation,
  type UpdateWorkerCredentialMutationVariables,
  type UpdateWorkerCredentialTypeMutation,
  type UpdateWorkerCredentialTypeMutationVariables,
  type VerifyWorkerCredentialMutation,
  type VerifyWorkerCredentialMutationVariables,
  type WorkerCredentialFieldsFragment,
  type WorkerCredentialInput,
  type WorkerCredentialSummaryQuery,
  type WorkerCredentialSummaryQueryVariables,
  type WorkerCredentialTypeFieldsFragment,
  type WorkerCredentialTypeInput,
  type WorkerCredentialsQuery,
  type WorkerCredentialsQueryVariables,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";

export type WorkerCredentialTypeRow = WorkerCredentialTypeFieldsFragment;
export type WorkerCredentialRow = WorkerCredentialFieldsFragment;
export type WorkerCredentialSummary = WorkerCredentialSummaryQuery["workerCredentialSummary"];
export type WorkerCredentialSummaryItem = WorkerCredentialSummary["items"][number];
export type CredentialExpiryForecast = CredentialExpiryForecastQuery["credentialExpiryForecast"];
export type CredentialExpiryForecastItem = CredentialExpiryForecast["items"][number];

export const WORKER_CREDENTIAL_TYPE_LIST_KEY = "worker-credential-type-list";
export const WORKER_CREDENTIALS_KEY = "worker-credentials";
export const WORKER_CREDENTIAL_SUMMARY_KEY = "worker-credential-summary";
export const CREDENTIAL_EXPIRY_FORECAST_KEY = "credential-expiry-forecast";

export const workerCredentialTypeTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: WorkerCredentialTypeTableDocument,
  operationName: "WorkerCredentialTypeTable",
  connectionKey: "workerCredentialTypes",
});

export async function fetchWorkerCredentials(
  workerId: string,
  includeArchived = true,
  options?: { signal?: AbortSignal },
): Promise<WorkerCredentialRow[]> {
  const data = await requestGraphQL<WorkerCredentialsQuery, WorkerCredentialsQueryVariables>({
    document: WorkerCredentialsDocument,
    operationName: "WorkerCredentials",
    variables: { workerId, includeArchived },
    signal: options?.signal,
  });
  return data.workerCredentials as WorkerCredentialRow[];
}

export async function fetchWorkerCredentialSummary(
  workerId: string,
  options?: { signal?: AbortSignal },
): Promise<WorkerCredentialSummary> {
  const data = await requestGraphQL<
    WorkerCredentialSummaryQuery,
    WorkerCredentialSummaryQueryVariables
  >({
    document: WorkerCredentialSummaryDocument,
    operationName: "WorkerCredentialSummary",
    variables: { workerId },
    signal: options?.signal,
  });
  return data.workerCredentialSummary;
}

export async function fetchCredentialExpiryForecast(
  days = 30,
  limit = 100,
  options?: { signal?: AbortSignal },
): Promise<CredentialExpiryForecast> {
  const data = await requestGraphQL<
    CredentialExpiryForecastQuery,
    CredentialExpiryForecastQueryVariables
  >({
    document: CredentialExpiryForecastDocument,
    operationName: "CredentialExpiryForecast",
    variables: { days, limit },
    signal: options?.signal,
  });
  return data.credentialExpiryForecast;
}

export async function createWorkerCredentialType(
  input: WorkerCredentialTypeInput,
): Promise<WorkerCredentialTypeRow> {
  const data = await requestGraphQL<
    CreateWorkerCredentialTypeMutation,
    CreateWorkerCredentialTypeMutationVariables
  >({
    document: CreateWorkerCredentialTypeDocument,
    operationName: "CreateWorkerCredentialType",
    variables: { input },
  });
  return data.createWorkerCredentialType as WorkerCredentialTypeRow;
}

export async function updateWorkerCredentialType(
  id: string,
  input: WorkerCredentialTypeInput,
): Promise<WorkerCredentialTypeRow> {
  const data = await requestGraphQL<
    UpdateWorkerCredentialTypeMutation,
    UpdateWorkerCredentialTypeMutationVariables
  >({
    document: UpdateWorkerCredentialTypeDocument,
    operationName: "UpdateWorkerCredentialType",
    variables: { id, input },
  });
  return data.updateWorkerCredentialType as WorkerCredentialTypeRow;
}

export async function archiveWorkerCredentialType(
  id: string,
  version?: number,
): Promise<WorkerCredentialTypeRow> {
  const data = await requestGraphQL<
    ArchiveWorkerCredentialTypeMutation,
    ArchiveWorkerCredentialTypeMutationVariables
  >({
    document: ArchiveWorkerCredentialTypeDocument,
    operationName: "ArchiveWorkerCredentialType",
    variables: { id, version },
  });
  return data.archiveWorkerCredentialType as WorkerCredentialTypeRow;
}

export async function restoreWorkerCredentialType(
  id: string,
  version?: number,
): Promise<WorkerCredentialTypeRow> {
  const data = await requestGraphQL<
    RestoreWorkerCredentialTypeMutation,
    RestoreWorkerCredentialTypeMutationVariables
  >({
    document: RestoreWorkerCredentialTypeDocument,
    operationName: "RestoreWorkerCredentialType",
    variables: { id, version },
  });
  return data.restoreWorkerCredentialType as WorkerCredentialTypeRow;
}

export async function createWorkerCredential(
  input: WorkerCredentialInput,
): Promise<WorkerCredentialRow> {
  const data = await requestGraphQL<
    CreateWorkerCredentialMutation,
    CreateWorkerCredentialMutationVariables
  >({
    document: CreateWorkerCredentialDocument,
    operationName: "CreateWorkerCredential",
    variables: { input },
  });
  return data.createWorkerCredential as WorkerCredentialRow;
}

export async function updateWorkerCredential(
  input: UpdateWorkerCredentialInput,
): Promise<WorkerCredentialRow> {
  const data = await requestGraphQL<
    UpdateWorkerCredentialMutation,
    UpdateWorkerCredentialMutationVariables
  >({
    document: UpdateWorkerCredentialDocument,
    operationName: "UpdateWorkerCredential",
    variables: { input },
  });
  return data.updateWorkerCredential as WorkerCredentialRow;
}

export async function verifyWorkerCredential(
  id: string,
  version?: number,
): Promise<WorkerCredentialRow> {
  const data = await requestGraphQL<
    VerifyWorkerCredentialMutation,
    VerifyWorkerCredentialMutationVariables
  >({
    document: VerifyWorkerCredentialDocument,
    operationName: "VerifyWorkerCredential",
    variables: { id, version },
  });
  return data.verifyWorkerCredential as WorkerCredentialRow;
}

export async function archiveWorkerCredential(
  input: ArchiveWorkerCredentialInput,
): Promise<WorkerCredentialRow> {
  const data = await requestGraphQL<
    ArchiveWorkerCredentialMutation,
    ArchiveWorkerCredentialMutationVariables
  >({
    document: ArchiveWorkerCredentialDocument,
    operationName: "ArchiveWorkerCredential",
    variables: { input },
  });
  return data.archiveWorkerCredential as WorkerCredentialRow;
}

export async function attachWorkerCredentialDocument(
  input: AttachWorkerCredentialDocumentInput,
): Promise<WorkerCredentialRow> {
  const data = await requestGraphQL<
    AttachWorkerCredentialDocumentMutation,
    AttachWorkerCredentialDocumentMutationVariables
  >({
    document: AttachWorkerCredentialDocumentDocument,
    operationName: "AttachWorkerCredentialDocument",
    variables: { input },
  });
  return data.attachWorkerCredentialDocument as WorkerCredentialRow;
}
