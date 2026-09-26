import { getFragmentData } from "@trenova/graphql/fragment-data";
import {
  ApproveJournalEntriesDocument,
  JournalReviewResultFieldsFragmentDoc,
  JournalReviewSummaryDocument,
  JournalReviewTableDocument,
  PostJournalEntriesDocument,
  type JournalReviewResultFieldsFragment,
  type JournalReviewSummaryQuery,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";

export const JOURNAL_REVIEW_TABLE_KEY = "journal-review";

export type JournalReviewSummary = JournalReviewSummaryQuery["journalReviewSummary"];
export type JournalReviewResult = JournalReviewResultFieldsFragment;

type RequestOptions = { signal?: AbortSignal };

export const journalReviewTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: JournalReviewTableDocument,
  operationName: "JournalReviewTable",
  connectionKey: "journalReviewTable",
});

export type JournalReviewRow = DataTableConfigRow<typeof journalReviewTableGraphQLConfig>;

export async function fetchJournalReviewSummary(
  options?: RequestOptions,
): Promise<JournalReviewSummary> {
  const data = await requestGraphQL({
    document: JournalReviewSummaryDocument,
    operationName: "JournalReviewSummary",
    variables: {},
    signal: options?.signal,
  });
  return data.journalReviewSummary;
}

export async function approveJournalEntries(entryIds: string[]): Promise<JournalReviewResult> {
  const data = await requestGraphQL({
    document: ApproveJournalEntriesDocument,
    operationName: "ApproveJournalEntries",
    variables: { input: { entryIds } },
  });
  return getFragmentData(JournalReviewResultFieldsFragmentDoc, data.approveJournalEntries);
}

export async function postJournalEntries(entryIds: string[]): Promise<JournalReviewResult> {
  const data = await requestGraphQL({
    document: PostJournalEntriesDocument,
    operationName: "PostJournalEntries",
    variables: { input: { entryIds } },
  });
  return getFragmentData(JournalReviewResultFieldsFragmentDoc, data.postJournalEntries);
}
