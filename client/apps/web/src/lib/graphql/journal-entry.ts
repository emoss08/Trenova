import {
  JournalEntriesBySourceDocument,
  JournalEntryDetailDocument,
  JournalSourceByObjectDocument,
  type JournalEntriesBySourceQuery,
  type JournalEntryDetailQuery,
  type JournalSourceByObjectQuery,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type JournalEntryDetail = NonNullable<JournalEntryDetailQuery["journalEntry"]>;
export type JournalEntryBySource = JournalEntriesBySourceQuery["journalEntriesBySource"][number];
export type JournalSourceInfo = NonNullable<JournalSourceByObjectQuery["journalSourceByObject"]>;

export async function fetchJournalEntry(id: string, options?: { signal?: AbortSignal }) {
  const data = await requestGraphQL({
    document: JournalEntryDetailDocument,
    operationName: "JournalEntryDetail",
    variables: { id },
    signal: options?.signal,
  });
  return data.journalEntry;
}

export async function fetchJournalEntriesBySource(
  sourceType: string,
  sourceId: string,
  options?: { signal?: AbortSignal },
) {
  const data = await requestGraphQL({
    document: JournalEntriesBySourceDocument,
    operationName: "JournalEntriesBySource",
    variables: { sourceType, sourceId },
    signal: options?.signal,
  });
  return data.journalEntriesBySource;
}

export async function fetchJournalSourceByObject(
  sourceType: string,
  sourceId: string,
  options?: { signal?: AbortSignal },
) {
  const data = await requestGraphQL({
    document: JournalSourceByObjectDocument,
    operationName: "JournalSourceByObject",
    variables: { sourceType, sourceId },
    signal: options?.signal,
  });
  return data.journalSourceByObject;
}
