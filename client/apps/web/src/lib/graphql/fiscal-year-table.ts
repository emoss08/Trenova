import {
  FiscalYearTableDocument,
  type FiscalYearTableQueryVariables,
} from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { FiscalYear } from "@/types/fiscal-year";

export const fiscalYearTableGraphQLConfig = defineDataTableGraphQLConfig<
  FiscalYear,
  FiscalYearTableQueryVariables
>({
  document: FiscalYearTableDocument,
  operationName: "FiscalYearTable",
  connectionKey: "fiscalYears",
});
