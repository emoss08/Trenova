import {
  AccessorialChargeTableDocument,
} from "@trenova/graphql/generated/documents/accessorial-charge/table";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";

export const accessorialChargeTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: AccessorialChargeTableDocument,
  operationName: "AccessorialChargeTable",
  connectionKey: "accessorialCharges",
});

export type AccessorialChargeRow = DataTableConfigRow<typeof accessorialChargeTableGraphQLConfig>;
