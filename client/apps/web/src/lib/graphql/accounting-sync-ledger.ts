import { getFragmentData } from "@trenova/graphql/fragment-data";
import {
  AccountingBackfillFieldsFragmentDoc,
  AccountingBackfillsDocument,
  AccountingConnectionFieldsFragmentDoc,
  AccountingSyncAttemptsDocument,
  AccountingSyncObjectStatesDocument,
  AccountingSyncRecordDocument,
  AccountingSyncRecordFieldsFragmentDoc,
  AccountingSyncSummaryDocument,
  ChangeAccountingBackfillDocument,
  EnableAccountingSyncDocument,
  PauseAccountingSyncDocument,
  RedateAccountingSyncDocument,
  ReleaseAccountingSyncDocument,
  RequestAccountingBackfillDocument,
  ResumeAccountingSyncDocument,
  RetryAccountingSyncDocument,
  SkipAccountingSyncDocument,
  UpdateAccountingSyncSettingsDocument,
  type AccountingBackfillFieldsFragment,
  type AccountingSyncAttemptsQuery,
  type AccountingSyncRecordFieldsFragment,
  type AccountingSyncSummaryQuery,
  type AccountingSystem,
  type ChangeAccountingBackfillInput,
  type EnableAccountingSyncInput,
  type PauseAccountingSyncInput,
  type ReleaseAccountingSyncInput,
  type RequestAccountingBackfillInput,
  type RetryAccountingSyncInput,
  type SkipAccountingSyncInput,
  type UpdateAccountingSyncSettingsInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type { AccountingConnection } from "./accounting-sync";

export type AccountingSyncRecord = AccountingSyncRecordFieldsFragment;
export type AccountingBackfill = AccountingBackfillFieldsFragment;
export type AccountingSyncAttempt = AccountingSyncAttemptsQuery["accountingSyncAttempts"][number];
type SummaryData = AccountingSyncSummaryQuery["accountingSyncSummary"];
export type AccountingSyncStatusCount = SummaryData["counts"][number];
export type AccountingSyncAttentionGroup = SummaryData["attention"][number];
export type AccountingSyncSummary = {
  integrationType: AccountingSystem;
  providerName: string;
  connection: AccountingConnection | null;
  counts: AccountingSyncStatusCount[];
  attention: AccountingSyncAttentionGroup[];
  activeBackfill: AccountingBackfill | null;
};
export type AccountingSyncObjectState = {
  objectType: AccountingSyncRecord["objectType"];
  objectId: string;
  providerName: string;
  record: AccountingSyncRecord;
};

type RequestOptions = { signal?: AbortSignal };

export async function fetchAccountingSyncSummary(
  integrationType: AccountingSystem,
  options?: RequestOptions,
): Promise<AccountingSyncSummary> {
  const data = await requestGraphQL({
    document: AccountingSyncSummaryDocument,
    operationName: "AccountingSyncSummary",
    variables: { integrationType },
    signal: options?.signal,
  });
  const summary = data.accountingSyncSummary;
  return {
    integrationType: summary.integrationType,
    providerName: summary.providerName,
    connection: summary.connection
      ? getFragmentData(AccountingConnectionFieldsFragmentDoc, summary.connection)
      : null,
    counts: summary.counts,
    attention: summary.attention,
    activeBackfill: summary.activeBackfill
      ? getFragmentData(AccountingBackfillFieldsFragmentDoc, summary.activeBackfill)
      : null,
  };
}

export async function fetchAccountingSyncRecord(
  id: string,
  options?: RequestOptions,
): Promise<AccountingSyncRecord> {
  const data = await requestGraphQL({
    document: AccountingSyncRecordDocument,
    operationName: "AccountingSyncRecord",
    variables: { id },
    signal: options?.signal,
  });
  return getFragmentData(AccountingSyncRecordFieldsFragmentDoc, data.accountingSyncRecord);
}

export async function fetchAccountingSyncAttempts(
  recordId: string,
  options?: RequestOptions,
): Promise<AccountingSyncAttempt[]> {
  const data = await requestGraphQL({
    document: AccountingSyncAttemptsDocument,
    operationName: "AccountingSyncAttempts",
    variables: { recordId },
    signal: options?.signal,
  });
  return data.accountingSyncAttempts;
}

export async function fetchAccountingSyncObjectStates(
  objectIds: readonly string[],
  options?: RequestOptions,
): Promise<AccountingSyncObjectState[]> {
  if (objectIds.length === 0) {
    return [];
  }
  const data = await requestGraphQL({
    document: AccountingSyncObjectStatesDocument,
    operationName: "AccountingSyncObjectStates",
    variables: { objectIds: [...objectIds] },
    signal: options?.signal,
  });
  return data.accountingSyncObjectStates.map((state) => ({
    objectType: state.objectType,
    objectId: state.objectId,
    providerName: state.providerName,
    record: getFragmentData(AccountingSyncRecordFieldsFragmentDoc, state.record),
  }));
}

export async function fetchAccountingBackfills(
  integrationType: AccountingSystem,
  options?: RequestOptions,
): Promise<AccountingBackfill[]> {
  const data = await requestGraphQL({
    document: AccountingBackfillsDocument,
    operationName: "AccountingBackfills",
    variables: { integrationType },
    signal: options?.signal,
  });
  return data.accountingBackfills.map((backfill) =>
    getFragmentData(AccountingBackfillFieldsFragmentDoc, backfill),
  );
}

export async function enableAccountingSync(
  input: EnableAccountingSyncInput,
): Promise<AccountingConnection> {
  const data = await requestGraphQL({
    document: EnableAccountingSyncDocument,
    operationName: "EnableAccountingSync",
    variables: { input },
  });
  return getFragmentData(AccountingConnectionFieldsFragmentDoc, data.enableAccountingSync);
}

export async function updateAccountingSyncSettings(
  input: UpdateAccountingSyncSettingsInput,
): Promise<AccountingConnection> {
  const data = await requestGraphQL({
    document: UpdateAccountingSyncSettingsDocument,
    operationName: "UpdateAccountingSyncSettings",
    variables: { input },
  });
  return getFragmentData(AccountingConnectionFieldsFragmentDoc, data.updateAccountingSyncSettings);
}

export async function pauseAccountingSync(
  input: PauseAccountingSyncInput,
): Promise<AccountingConnection> {
  const data = await requestGraphQL({
    document: PauseAccountingSyncDocument,
    operationName: "PauseAccountingSync",
    variables: { input },
  });
  return getFragmentData(AccountingConnectionFieldsFragmentDoc, data.pauseAccountingSync);
}

export async function resumeAccountingSync(
  integrationType: AccountingSystem,
): Promise<AccountingConnection> {
  const data = await requestGraphQL({
    document: ResumeAccountingSyncDocument,
    operationName: "ResumeAccountingSync",
    variables: { integrationType },
  });
  return getFragmentData(AccountingConnectionFieldsFragmentDoc, data.resumeAccountingSync);
}

export async function retryAccountingSync(input: RetryAccountingSyncInput): Promise<number> {
  const data = await requestGraphQL({
    document: RetryAccountingSyncDocument,
    operationName: "RetryAccountingSync",
    variables: { input },
  });
  return data.retryAccountingSync.affected;
}

export async function releaseAccountingSync(input: ReleaseAccountingSyncInput): Promise<number> {
  const data = await requestGraphQL({
    document: ReleaseAccountingSyncDocument,
    operationName: "ReleaseAccountingSync",
    variables: { input },
  });
  return data.releaseAccountingSync.affected;
}

export async function skipAccountingSync(
  input: SkipAccountingSyncInput,
): Promise<AccountingSyncRecord> {
  const data = await requestGraphQL({
    document: SkipAccountingSyncDocument,
    operationName: "SkipAccountingSync",
    variables: { input },
  });
  return getFragmentData(AccountingSyncRecordFieldsFragmentDoc, data.skipAccountingSync);
}

export async function redateAccountingSync(id: string): Promise<AccountingSyncRecord> {
  const data = await requestGraphQL({
    document: RedateAccountingSyncDocument,
    operationName: "RedateAccountingSync",
    variables: { id },
  });
  return getFragmentData(AccountingSyncRecordFieldsFragmentDoc, data.redateAccountingSync);
}

export async function requestAccountingBackfill(
  input: RequestAccountingBackfillInput,
): Promise<AccountingBackfill> {
  const data = await requestGraphQL({
    document: RequestAccountingBackfillDocument,
    operationName: "RequestAccountingBackfill",
    variables: { input },
  });
  return getFragmentData(AccountingBackfillFieldsFragmentDoc, data.requestAccountingBackfill);
}

export async function changeAccountingBackfill(
  input: ChangeAccountingBackfillInput,
): Promise<AccountingBackfill> {
  const data = await requestGraphQL({
    document: ChangeAccountingBackfillDocument,
    operationName: "ChangeAccountingBackfill",
    variables: { input },
  });
  return getFragmentData(AccountingBackfillFieldsFragmentDoc, data.changeAccountingBackfill);
}
