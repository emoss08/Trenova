import { HoldReasonTableDocument } from "@trenova/graphql/generated/documents/hold-reason/table";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import { isRecord } from "@trenova/shared/lib/utils";

export const holdReasonTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: HoldReasonTableDocument,
  operationName: "HoldReasonTable",
  connectionKey: "holdReasons",
  mapNode: (node) => ({
    ...node,
    externalMap: isRecord(node.externalMap) ? node.externalMap : null,
  }),
});
