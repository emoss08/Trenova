import {
  ManualJournalTableDocument,
} from "@trenova/graphql/generated/documents/manual-journal/table";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";

export const manualJournalTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: ManualJournalTableDocument,
  operationName: "ManualJournalTable",
  connectionKey: "manualJournals",
});

export type ManualJournalRow = DataTableConfigRow<typeof manualJournalTableGraphQLConfig>;
