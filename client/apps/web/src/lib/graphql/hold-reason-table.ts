import {
  HoldReasonTableDocument,
  type HoldReasonTableQueryVariables,
} from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@/lib/graphql/data-table";
import type { HoldReason } from "@/types/hold-reason";

export const holdReasonTableGraphQLConfig = defineDataTableGraphQLConfig<
  HoldReason,
  HoldReasonTableQueryVariables
>({
  document: HoldReasonTableDocument,
  operationName: "HoldReasonTable",
  connectionKey: "holdReasons",
});
