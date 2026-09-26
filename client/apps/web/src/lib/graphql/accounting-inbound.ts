import { getFragmentData } from "@trenova/graphql/fragment-data";
import {
  AccountingInboundApplyPreviewDocument,
  AccountingInboundChangeDocument,
  AccountingInboundChangeFieldsFragmentDoc,
  AccountingInboundOverviewDocument,
  ApplyAccountingInboundChangeDocument,
  IgnoreAccountingInboundChangeDocument,
  type AccountingInboundApplyPreviewQuery,
  type AccountingInboundChangeFieldsFragment,
  type AccountingInboundOverviewQuery,
  type AccountingSystem,
  type IgnoreAccountingInboundChangeInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type AccountingInboundChange = AccountingInboundChangeFieldsFragment;
export type AccountingInboundOverview = AccountingInboundOverviewQuery["accountingInboundOverview"];
export type AccountingInboundApplyPreview =
  AccountingInboundApplyPreviewQuery["accountingInboundApplyPreview"];

type RequestOptions = { signal?: AbortSignal };

export async function fetchAccountingInboundOverview(
  integrationType: AccountingSystem,
  options?: RequestOptions,
): Promise<AccountingInboundOverview> {
  const data = await requestGraphQL({
    document: AccountingInboundOverviewDocument,
    operationName: "AccountingInboundOverview",
    variables: { integrationType },
    signal: options?.signal,
  });
  return data.accountingInboundOverview;
}

export async function fetchAccountingInboundChange(
  id: string,
  options?: RequestOptions,
): Promise<AccountingInboundChange> {
  const data = await requestGraphQL({
    document: AccountingInboundChangeDocument,
    operationName: "AccountingInboundChange",
    variables: { id },
    signal: options?.signal,
  });
  return getFragmentData(AccountingInboundChangeFieldsFragmentDoc, data.accountingInboundChange);
}

export async function fetchAccountingInboundApplyPreview(
  id: string,
  options?: RequestOptions,
): Promise<AccountingInboundApplyPreview> {
  const data = await requestGraphQL({
    document: AccountingInboundApplyPreviewDocument,
    operationName: "AccountingInboundApplyPreview",
    variables: { id },
    signal: options?.signal,
  });
  return data.accountingInboundApplyPreview;
}

export async function applyAccountingInboundChange(id: string): Promise<AccountingInboundChange> {
  const data = await requestGraphQL({
    document: ApplyAccountingInboundChangeDocument,
    operationName: "ApplyAccountingInboundChange",
    variables: { id },
  });
  return getFragmentData(
    AccountingInboundChangeFieldsFragmentDoc,
    data.applyAccountingInboundChange,
  );
}

export async function ignoreAccountingInboundChange(
  input: IgnoreAccountingInboundChangeInput,
): Promise<AccountingInboundChange> {
  const data = await requestGraphQL({
    document: IgnoreAccountingInboundChangeDocument,
    operationName: "IgnoreAccountingInboundChange",
    variables: { input },
  });
  return getFragmentData(
    AccountingInboundChangeFieldsFragmentDoc,
    data.ignoreAccountingInboundChange,
  );
}
