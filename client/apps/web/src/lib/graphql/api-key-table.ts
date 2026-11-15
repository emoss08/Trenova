import { ApiKeyTableDocument } from "@trenova/graphql/generated/documents/api-key/table";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";

export const apiKeyTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: ApiKeyTableDocument,
  operationName: "ApiKeyTable",
  connectionKey: "apiKeys",
});

export type ApiKeyRow = DataTableConfigRow<typeof apiKeyTableGraphQLConfig>;
