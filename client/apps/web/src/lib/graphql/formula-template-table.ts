import {
  FormulaTemplateTableDocument,
} from "@trenova/graphql/generated/documents/formula-template/table";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";

export const formulaTemplateTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: FormulaTemplateTableDocument,
  operationName: "FormulaTemplateTable",
  connectionKey: "formulaTemplates",
});

export type FormulaTemplateRow = DataTableConfigRow<typeof formulaTemplateTableGraphQLConfig>;
