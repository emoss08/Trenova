import {
  ManualJournalTableDocument,
  type ManualJournalTableQueryVariables,
} from "@/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@/lib/graphql/data-table";
import type { ManualJournal } from "@/types/manual-journal";

export const manualJournalTableGraphQLConfig = defineDataTableGraphQLConfig<
  ManualJournal,
  ManualJournalTableQueryVariables
>({
  document: ManualJournalTableDocument,
  operationName: "ManualJournalTable",
  connectionKey: "manualJournals",
});
