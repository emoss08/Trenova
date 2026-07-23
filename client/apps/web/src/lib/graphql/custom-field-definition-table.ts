import {
  CustomFieldDefinitionTableDocument,
  type CustomFieldDefinitionTableQueryVariables,
} from "@/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@/lib/graphql/data-table";
import type { CustomFieldDefinition } from "@/types/custom-field";

export const customFieldDefinitionTableGraphQLConfig =
  defineDataTableGraphQLConfig<
    CustomFieldDefinition,
    CustomFieldDefinitionTableQueryVariables
  >({
    document: CustomFieldDefinitionTableDocument,
    operationName: "CustomFieldDefinitionTable",
    connectionKey: "customFieldDefinitions",
  });
