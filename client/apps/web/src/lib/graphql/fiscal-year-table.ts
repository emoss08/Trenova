import { FiscalYearTableDocument } from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";

export const fiscalYearTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: FiscalYearTableDocument,
  operationName: "FiscalYearTable",
  connectionKey: "fiscalYears",
});

export type FiscalYearRow = DataTableConfigRow<typeof fiscalYearTableGraphQLConfig>;
