import { getFragmentData } from "@trenova/graphql/fragment-data";
import {
  AccountingDriftFindingDocument,
  AccountingDriftFindingFieldsFragmentDoc,
  AccountingDriftFixPreviewDocument,
  AccountingDriftOverviewDocument,
  AccountingDriftOverviewFieldsFragmentDoc,
  CheckAccountingDriftDocument,
  DismissAccountingDriftDocument,
  ResolveAccountingDriftDocument,
  type AccountingDriftFindingFieldsFragment,
  type AccountingDriftFixPreviewQuery,
  type AccountingDriftOverviewFieldsFragment,
  type AccountingSystem,
  type DismissAccountingDriftInput,
  type ResolveAccountingDriftInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type AccountingDriftFinding = AccountingDriftFindingFieldsFragment;
export type AccountingDriftOverview = AccountingDriftOverviewFieldsFragment;
export type AccountingDriftFixPreview = AccountingDriftFixPreviewQuery["accountingDriftFixPreview"];

type RequestOptions = { signal?: AbortSignal };

export async function fetchAccountingDriftOverview(
  integrationType: AccountingSystem,
  options?: RequestOptions,
): Promise<AccountingDriftOverview> {
  const data = await requestGraphQL({
    document: AccountingDriftOverviewDocument,
    operationName: "AccountingDriftOverview",
    variables: { integrationType },
    signal: options?.signal,
  });
  return getFragmentData(AccountingDriftOverviewFieldsFragmentDoc, data.accountingDriftOverview);
}

export async function fetchAccountingDriftFinding(
  id: string,
  options?: RequestOptions,
): Promise<AccountingDriftFinding> {
  const data = await requestGraphQL({
    document: AccountingDriftFindingDocument,
    operationName: "AccountingDriftFinding",
    variables: { id },
    signal: options?.signal,
  });
  return getFragmentData(AccountingDriftFindingFieldsFragmentDoc, data.accountingDriftFinding);
}

export async function fetchAccountingDriftFixPreview(
  input: ResolveAccountingDriftInput,
  options?: RequestOptions,
): Promise<AccountingDriftFixPreview> {
  const data = await requestGraphQL({
    document: AccountingDriftFixPreviewDocument,
    operationName: "AccountingDriftFixPreview",
    variables: { input },
    signal: options?.signal,
  });
  return data.accountingDriftFixPreview;
}

export async function resolveAccountingDrift(
  input: ResolveAccountingDriftInput,
): Promise<AccountingDriftFinding> {
  const data = await requestGraphQL({
    document: ResolveAccountingDriftDocument,
    operationName: "ResolveAccountingDrift",
    variables: { input },
  });
  return getFragmentData(AccountingDriftFindingFieldsFragmentDoc, data.resolveAccountingDrift);
}

export async function dismissAccountingDrift(
  input: DismissAccountingDriftInput,
): Promise<AccountingDriftFinding> {
  const data = await requestGraphQL({
    document: DismissAccountingDriftDocument,
    operationName: "DismissAccountingDrift",
    variables: { input },
  });
  return getFragmentData(AccountingDriftFindingFieldsFragmentDoc, data.dismissAccountingDrift);
}

export async function checkAccountingDrift(
  integrationType: AccountingSystem,
): Promise<AccountingDriftOverview> {
  const data = await requestGraphQL({
    document: CheckAccountingDriftDocument,
    operationName: "CheckAccountingDrift",
    variables: { integrationType },
  });
  return getFragmentData(AccountingDriftOverviewFieldsFragmentDoc, data.checkAccountingDrift);
}
