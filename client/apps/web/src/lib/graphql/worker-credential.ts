import type {
  ArchiveWorkerCredentialInput,
  ArchiveWorkerCredentialMutation,
  ArchiveWorkerCredentialMutationVariables,
  ArchiveWorkerCredentialTypeMutation,
  ArchiveWorkerCredentialTypeMutationVariables,
  AttachWorkerCredentialDocumentInput,
  AttachWorkerCredentialDocumentMutation,
  AttachWorkerCredentialDocumentMutationVariables,
  CreateWorkerCredentialMutation,
  CreateWorkerCredentialMutationVariables,
  CreateWorkerCredentialTypeMutation,
  CreateWorkerCredentialTypeMutationVariables,
  CredentialExpiryForecastQuery,
  CredentialExpiryForecastQueryVariables,
  RestoreWorkerCredentialTypeMutation,
  RestoreWorkerCredentialTypeMutationVariables,
  UpdateWorkerCredentialInput,
  UpdateWorkerCredentialMutation,
  UpdateWorkerCredentialMutationVariables,
  UpdateWorkerCredentialTypeMutation,
  UpdateWorkerCredentialTypeMutationVariables,
  VerifyWorkerCredentialMutation,
  VerifyWorkerCredentialMutationVariables,
  WorkerCredentialFieldsFragment,
  WorkerCredentialInput,
  WorkerCredentialSummaryQuery,
  WorkerCredentialSummaryQueryVariables,
  WorkerCredentialTypeFieldsFragment,
  WorkerCredentialTypeInput,
  WorkerCredentialsQuery,
  WorkerCredentialsQueryVariables,
} from "@trenova/graphql/generated/graphql";
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
  WorkerCredentialTypeChoicesDocument,
  WorkerCredentialTypeTableDocument,
  WorkerCredentialsDocument,
} from "@trenova/graphql/generated/documents/worker-credential/operations";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { rootFields, type RootFields } from "@trenova/shared/lib/reference-access";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";

/*
 * A credential type is the root record of every read that returns this fragment, so the fields
 * the schema marks nullable for where a type is linked from a worker's credentials
 * (docs/engineering/reference-views.md) are its own here and never withheld.
 */
const CREDENTIAL_TYPE_ROOT_FIELDS = [
  "activeCredentialCount",
  "requiredForDriverTypes",
  "status",
] as const;

export type WorkerCredentialTypeRow = RootFields<
  WorkerCredentialTypeFieldsFragment,
  (typeof CREDENTIAL_TYPE_ROOT_FIELDS)[number]
>;
export type WorkerCredentialRow = WorkerCredentialFieldsFragment;
type RawCredentialSummary = WorkerCredentialSummaryQuery["workerCredentialSummary"];
type RawCredentialSummaryItem = RawCredentialSummary["items"][number];
/**
 * A credential slot. The server answers its type null only when the organization's plan leaves
 * out Safety & Compliance, and a slot with no type is no slot, so it is left out.
 */
export type WorkerCredentialSummaryItem = Omit<RawCredentialSummaryItem, "credentialType"> & {
  credentialType: NonNullable<RawCredentialSummaryItem["credentialType"]>;
};
export type WorkerCredentialSummary = Omit<RawCredentialSummary, "items"> & {
  items: WorkerCredentialSummaryItem[];
};

/** The credential slots whose type the server answered (see `WorkerCredentialSummaryItem`). */
export function typedCredentialSlots<T extends { credentialType: unknown }>(
  items: readonly T[],
): (T & { credentialType: NonNullable<T["credentialType"]> })[] {
  return items.filter(
    (item): item is T & { credentialType: NonNullable<T["credentialType"]> } =>
      item.credentialType != null,
  );
}
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
  mapNode: (node): WorkerCredentialTypeRow =>
    rootFields(node, "WorkerCredentialType", CREDENTIAL_TYPE_ROOT_FIELDS),
});

/**
 * One worker's credentials, current and archived, newest first, capped by the server at 100. The
 * current ones are also in the summary, one per type.
 */
/**
 * A worker's credentials, archived ones included unless asked otherwise, capped by the server at
 * the first 100; with `credentialTypeCode`, only those of that type (a credential's versions).
 */
export async function fetchWorkerCredentials(
  variables: { workerId: string; includeArchived?: boolean; credentialTypeCode?: string },
  options?: { signal?: AbortSignal },
): Promise<WorkerCredentialRow[]> {
  const data = await requestGraphQL<WorkerCredentialsQuery, WorkerCredentialsQueryVariables>({
    document: WorkerCredentialsDocument,
    operationName: "WorkerCredentials",
    variables: {
      workerId: variables.workerId,
      includeArchived: variables.includeArchived ?? true,
      credentialTypeCode: variables.credentialTypeCode,
    },
    signal: options?.signal,
  });
  return data.workerCredentials as WorkerCredentialRow[];
}

export type WorkerCredentialTypeChoice = {
  id: string;
  code: string;
  name: string;
  isRequired: boolean;
};

export async function fetchWorkerCredentialTypeChoices(options?: {
  signal?: AbortSignal;
}): Promise<WorkerCredentialTypeChoice[]> {
  const data = await requestGraphQL({
    document: WorkerCredentialTypeChoicesDocument,
    operationName: "WorkerCredentialTypeChoices",
    variables: {},
    signal: options?.signal,
  });
  return data.activeWorkerCredentialTypes;
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
  return {
    ...data.workerCredentialSummary,
    items: typedCredentialSlots(data.workerCredentialSummary.items),
  };
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
