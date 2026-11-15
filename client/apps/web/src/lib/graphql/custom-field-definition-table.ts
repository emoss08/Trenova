import {
  CustomFieldDefinitionTableDocument,
} from "@trenova/graphql/generated/documents/custom-field/table";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";

export const customFieldDefinitionTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: CustomFieldDefinitionTableDocument,
  operationName: "CustomFieldDefinitionTable",
  connectionKey: "customFieldDefinitions",
});

export type CustomFieldDefinitionRow = DataTableConfigRow<
  typeof customFieldDefinitionTableGraphQLConfig
>;
