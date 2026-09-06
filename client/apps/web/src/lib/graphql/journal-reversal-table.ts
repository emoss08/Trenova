import { JournalReversalTableDocument } from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";

export const journalReversalTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: JournalReversalTableDocument,
  operationName: "JournalReversalTable",
  connectionKey: "journalReversals",
});

export type JournalReversalRow = DataTableConfigRow<typeof journalReversalTableGraphQLConfig>;
