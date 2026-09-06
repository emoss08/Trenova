import { LocationCategoryTableDocument } from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";

export const locationCategoryTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: LocationCategoryTableDocument,
  operationName: "LocationCategoryTable",
  connectionKey: "locationCategories",
});
