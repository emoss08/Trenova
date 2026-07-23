import {
  CommodityTableDocument,
  type CommodityTableQueryVariables,
} from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@/lib/graphql/data-table";
import type { Commodity } from "@/types/commodity";

export const commodityTableGraphQLConfig = defineDataTableGraphQLConfig<
  Commodity,
  CommodityTableQueryVariables
>({
  document: CommodityTableDocument,
  operationName: "CommodityTable",
  connectionKey: "commodities",
});
