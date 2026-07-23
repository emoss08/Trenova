import {
  JournalReversalTableDocument,
  type JournalReversalTableQueryVariables,
} from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@/lib/graphql/data-table";
import type { JournalReversal } from "@/types/journal-reversal";

export const journalReversalTableGraphQLConfig = defineDataTableGraphQLConfig<
  JournalReversal,
  JournalReversalTableQueryVariables
>({
  document: JournalReversalTableDocument,
  operationName: "JournalReversalTable",
  connectionKey: "journalReversals",
});
