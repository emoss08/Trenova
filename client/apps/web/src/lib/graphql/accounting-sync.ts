import { getFragmentData } from "@trenova/graphql/fragment-data";
import {
  AccountingConnectionFieldsFragmentDoc,
  AccountingMappingFieldsFragmentDoc,
  AccountingMappingSummaryDocument,
  AccountingMappingsDocument,
  AccountingReferenceObjectFieldsFragmentDoc,
  AccountingReferenceObjectsDocument,
  AccountingSyncStatusDocument,
  CheckAccountingConnectionDocument,
  ClearAccountingMappingDocument,
  CompleteAccountingAuthorizationDocument,
  CompleteAccountingSetupDocument,
  ConfirmAccountingMappingsDocument,
  CreateAccountingReferenceRecordDocument,
  DisconnectAccountingSystemDocument,
  RefreshAccountingReferenceDataDocument,
  RejectAccountingMappingDocument,
  SetAccountingMappingDocument,
  StartAccountingAuthorizationDocument,
  type AccountingConnectionFieldsFragment,
  type AccountingMappingFieldsFragment,
  type AccountingMappingFilterInput,
  type AccountingMappingTargetType,
  type AccountingReferenceKind,
  type AccountingReferenceObjectFieldsFragment,
  type AccountingSystem,
  type CompleteAccountingAuthorizationInput,
  type CreateAccountingReferenceRecordInput,
  type SetAccountingMappingInput,
  type StartAccountingAuthorizationMutation,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type AccountingConnection = AccountingConnectionFieldsFragment;
export type AccountingMapping = AccountingMappingFieldsFragment;
export type AccountingReferenceObject = AccountingReferenceObjectFieldsFragment;
export type AccountingMappingGroup = {
  targetType: AccountingMappingTargetType;
  unmatched: number;
  proposed: number;
  confirmed: number;
};
export type AccountingMappingSummary = {
  integrationType: AccountingSystem;
  providerName: string;
  connection: AccountingConnection | null;
  groups: AccountingMappingGroup[];
  requiredTotal: number;
  requiredConfirmed: number;
  canCompleteSetup: boolean;
};
export type AccountingMappingPage = {
  mappings: AccountingMapping[];
  hasNextPage: boolean;
  endCursor: string | null;
};
export type AccountingReferenceSearch = {
  integrationType: AccountingSystem;
  kind: AccountingReferenceKind;
  query: string;
};
export type AccountingAuthorizationStart =
  StartAccountingAuthorizationMutation["startAccountingAuthorization"];
export type AccountingSyncStatus = {
  integrationType: AccountingSystem;
  providerName: string;
  available: boolean;
  connection: AccountingConnection | null;
};

type RequestOptions = { signal?: AbortSignal };

export async function fetchAccountingSyncStatus(
  integrationType: AccountingSystem,
  options?: RequestOptions,
): Promise<AccountingSyncStatus> {
  const data = await requestGraphQL({
    document: AccountingSyncStatusDocument,
    operationName: "AccountingSyncStatus",
    variables: { integrationType },
    signal: options?.signal,
  });
  const status = data.accountingSyncStatus;
  return {
    integrationType: status.integrationType,
    providerName: status.providerName,
    available: status.available,
    connection: status.connection
      ? getFragmentData(AccountingConnectionFieldsFragmentDoc, status.connection)
      : null,
  };
}

export async function startAccountingAuthorization(
  integrationType: AccountingSystem,
): Promise<AccountingAuthorizationStart> {
  const data = await requestGraphQL({
    document: StartAccountingAuthorizationDocument,
    operationName: "StartAccountingAuthorization",
    variables: { integrationType },
  });
  return data.startAccountingAuthorization;
}

export async function completeAccountingAuthorization(
  input: CompleteAccountingAuthorizationInput,
): Promise<AccountingConnection> {
  const data = await requestGraphQL({
    document: CompleteAccountingAuthorizationDocument,
    operationName: "CompleteAccountingAuthorization",
    variables: { input },
  });
  return getFragmentData(
    AccountingConnectionFieldsFragmentDoc,
    data.completeAccountingAuthorization,
  );
}

export async function disconnectAccountingSystem(
  integrationType: AccountingSystem,
): Promise<AccountingConnection> {
  const data = await requestGraphQL({
    document: DisconnectAccountingSystemDocument,
    operationName: "DisconnectAccountingSystem",
    variables: { integrationType },
  });
  return getFragmentData(AccountingConnectionFieldsFragmentDoc, data.disconnectAccountingSystem);
}

export async function checkAccountingConnection(
  integrationType: AccountingSystem,
): Promise<AccountingConnection> {
  const data = await requestGraphQL({
    document: CheckAccountingConnectionDocument,
    operationName: "CheckAccountingConnection",
    variables: { integrationType },
  });
  return getFragmentData(AccountingConnectionFieldsFragmentDoc, data.checkAccountingConnection);
}

export async function fetchAccountingMappingSummary(
  integrationType: AccountingSystem,
  options?: RequestOptions,
): Promise<AccountingMappingSummary> {
  const data = await requestGraphQL({
    document: AccountingMappingSummaryDocument,
    operationName: "AccountingMappingSummary",
    variables: { integrationType },
    signal: options?.signal,
  });
  const summary = data.accountingMappingSummary;
  return {
    integrationType: summary.integrationType,
    providerName: summary.providerName,
    connection: summary.connection
      ? getFragmentData(AccountingConnectionFieldsFragmentDoc, summary.connection)
      : null,
    groups: summary.groups,
    requiredTotal: summary.requiredTotal,
    requiredConfirmed: summary.requiredConfirmed,
    canCompleteSetup: summary.canCompleteSetup,
  };
}

export async function fetchAccountingMappings(
  params: {
    integrationType: AccountingSystem;
    filter: AccountingMappingFilterInput;
    first: number;
    after?: string | null;
  },
  options?: RequestOptions,
): Promise<AccountingMappingPage> {
  const data = await requestGraphQL({
    document: AccountingMappingsDocument,
    operationName: "AccountingMappings",
    variables: {
      integrationType: params.integrationType,
      filter: params.filter,
      first: params.first,
      after: params.after ?? null,
    },
    signal: options?.signal,
  });
  const connection = data.accountingMappings;
  return {
    mappings: connection.edges.map((edge) =>
      getFragmentData(AccountingMappingFieldsFragmentDoc, edge.node),
    ),
    hasNextPage: connection.pageInfo.hasNextPage,
    endCursor: connection.pageInfo.endCursor ?? null,
  };
}

export async function searchAccountingReferenceObjects(
  search: AccountingReferenceSearch,
  options?: RequestOptions,
): Promise<AccountingReferenceObject[]> {
  const data = await requestGraphQL({
    document: AccountingReferenceObjectsDocument,
    operationName: "AccountingReferenceObjects",
    variables: {
      integrationType: search.integrationType,
      kind: search.kind,
      query: search.query,
      usableOnly: true,
      limit: 25,
    },
    signal: options?.signal,
  });
  return data.accountingReferenceObjects.map((ref) =>
    getFragmentData(AccountingReferenceObjectFieldsFragmentDoc, ref),
  );
}

export async function confirmAccountingMappings(ids: string[]): Promise<AccountingMapping[]> {
  const data = await requestGraphQL({
    document: ConfirmAccountingMappingsDocument,
    operationName: "ConfirmAccountingMappings",
    variables: { ids },
  });
  return data.confirmAccountingMappings.map((row) =>
    getFragmentData(AccountingMappingFieldsFragmentDoc, row),
  );
}

export async function rejectAccountingMapping(id: string): Promise<AccountingMapping> {
  const data = await requestGraphQL({
    document: RejectAccountingMappingDocument,
    operationName: "RejectAccountingMapping",
    variables: { id },
  });
  return getFragmentData(AccountingMappingFieldsFragmentDoc, data.rejectAccountingMapping);
}

export async function setAccountingMapping(
  input: SetAccountingMappingInput,
): Promise<AccountingMapping> {
  const data = await requestGraphQL({
    document: SetAccountingMappingDocument,
    operationName: "SetAccountingMapping",
    variables: { input },
  });
  return getFragmentData(AccountingMappingFieldsFragmentDoc, data.setAccountingMapping);
}

export async function clearAccountingMapping(id: string): Promise<AccountingMapping> {
  const data = await requestGraphQL({
    document: ClearAccountingMappingDocument,
    operationName: "ClearAccountingMapping",
    variables: { id },
  });
  return getFragmentData(AccountingMappingFieldsFragmentDoc, data.clearAccountingMapping);
}

export async function createAccountingReferenceRecord(
  input: CreateAccountingReferenceRecordInput,
): Promise<AccountingMapping> {
  const data = await requestGraphQL({
    document: CreateAccountingReferenceRecordDocument,
    operationName: "CreateAccountingReferenceRecord",
    variables: { input },
  });
  return getFragmentData(AccountingMappingFieldsFragmentDoc, data.createAccountingReferenceRecord);
}

export async function refreshAccountingReferenceData(
  integrationType: AccountingSystem,
): Promise<AccountingConnection> {
  const data = await requestGraphQL({
    document: RefreshAccountingReferenceDataDocument,
    operationName: "RefreshAccountingReferenceData",
    variables: { integrationType },
  });
  return getFragmentData(
    AccountingConnectionFieldsFragmentDoc,
    data.refreshAccountingReferenceData,
  );
}

export async function completeAccountingSetup(
  integrationType: AccountingSystem,
): Promise<AccountingConnection> {
  const data = await requestGraphQL({
    document: CompleteAccountingSetupDocument,
    operationName: "CompleteAccountingSetup",
    variables: { integrationType },
  });
  return getFragmentData(AccountingConnectionFieldsFragmentDoc, data.completeAccountingSetup);
}
