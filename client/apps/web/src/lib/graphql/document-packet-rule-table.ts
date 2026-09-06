import { DocumentPacketRuleTableDocument } from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";

export const documentPacketRuleTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: DocumentPacketRuleTableDocument,
  operationName: "DocumentPacketRuleTable",
  connectionKey: "documentPacketRules",
});

export type DocumentPacketRuleRow = DataTableConfigRow<typeof documentPacketRuleTableGraphQLConfig>;
