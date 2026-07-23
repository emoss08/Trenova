import {
  JournalEntriesBySourceDocument,
  JournalEntryDetailDocument,
  JournalSourceByObjectDocument,
  type JournalEntriesBySourceQuery,
  type JournalEntryDetailQuery,
  type JournalSourceByObjectQuery,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@/lib/graphql";

export type JournalEntryDetail = NonNullable<JournalEntryDetailQuery["journalEntry"]>;
export type JournalEntryBySource =
  JournalEntriesBySourceQuery["journalEntriesBySource"][number];
export type JournalSourceInfo = NonNullable<
  JournalSourceByObjectQuery["journalSourceByObject"]
>;

export async function fetchJournalEntry(id: string) {
  const data = await requestGraphQL({
    document: JournalEntryDetailDocument,
    operationName: "JournalEntryDetail",
    variables: { id },
  });
  return data.journalEntry;
}

export async function fetchJournalEntriesBySource(sourceType: string, sourceId: string) {
  const data = await requestGraphQL({
    document: JournalEntriesBySourceDocument,
    operationName: "JournalEntriesBySource",
    variables: { sourceType, sourceId },
  });
  return data.journalEntriesBySource;
}

export async function fetchJournalSourceByObject(sourceType: string, sourceId: string) {
  const data = await requestGraphQL({
    document: JournalSourceByObjectDocument,
    operationName: "JournalSourceByObject",
    variables: { sourceType, sourceId },
  });
  return data.journalSourceByObject;
}
