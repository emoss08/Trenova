import {
  FormulaTemplateTableDocument,
  type FormulaTemplateTableQueryVariables,
} from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@/lib/graphql/data-table";
import type { FormulaTemplate } from "@/types/formula-template";

export const formulaTemplateTableGraphQLConfig = defineDataTableGraphQLConfig<
  FormulaTemplate,
  FormulaTemplateTableQueryVariables
>({
  document: FormulaTemplateTableDocument,
  operationName: "FormulaTemplateTable",
  connectionKey: "formulaTemplates",
});
