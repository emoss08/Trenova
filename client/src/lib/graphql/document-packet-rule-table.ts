import {
  DocumentPacketRuleTableDocument,
  type DocumentPacketRuleTableQueryVariables,
} from "@/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@/lib/graphql/data-table";
import type { DocumentPacketRule } from "@/types/document-packet-rule";

export const documentPacketRuleTableGraphQLConfig = defineDataTableGraphQLConfig<
  DocumentPacketRule,
  DocumentPacketRuleTableQueryVariables
>({
  document: DocumentPacketRuleTableDocument,
  operationName: "DocumentPacketRuleTable",
  connectionKey: "documentPacketRules",
});
