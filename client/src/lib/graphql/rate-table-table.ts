import {
  RateTableTableDocument,
  type RateTableTableQueryVariables,
} from "@/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@/lib/graphql/data-table";
import type { RateTableRow } from "@/types/rate-table";

export const rateTableTableGraphQLConfig = defineDataTableGraphQLConfig<
  RateTableRow,
  RateTableTableQueryVariables
>({
  document: RateTableTableDocument,
  operationName: "RateTableTable",
  connectionKey: "rateTables",
});
