import { getFragmentData } from "@trenova/graphql/fragment-data";
import {
  AccountingConnectionFieldsFragmentDoc,
  AccountingSyncStatusDocument,
  CheckAccountingConnectionDocument,
  CompleteAccountingAuthorizationDocument,
  DisconnectAccountingSystemDocument,
  StartAccountingAuthorizationDocument,
  type AccountingConnectionFieldsFragment,
  type AccountingSystem,
  type CompleteAccountingAuthorizationInput,
  type StartAccountingAuthorizationMutation,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type AccountingConnection = AccountingConnectionFieldsFragment;
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
