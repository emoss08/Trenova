import { CommodityTableDocument } from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";

export const commodityTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: CommodityTableDocument,
  operationName: "CommodityTable",
  connectionKey: "commodities",
});

export type CommodityRow = DataTableConfigRow<typeof commodityTableGraphQLConfig>;
