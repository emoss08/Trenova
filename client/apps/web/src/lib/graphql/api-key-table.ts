import {
  ApiKeyTableDocument,
  type ApiKeyTableQueryVariables,
} from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { ApiKey } from "@/types/api-key";

export const apiKeyTableGraphQLConfig = defineDataTableGraphQLConfig<
  ApiKey,
  ApiKeyTableQueryVariables
>({
  document: ApiKeyTableDocument,
  operationName: "ApiKeyTable",
  connectionKey: "apiKeys",
});
