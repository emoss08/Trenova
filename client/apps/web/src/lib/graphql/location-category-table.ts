import { LocationCategoryTableDocument } from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import { rootFields } from "@trenova/shared/lib/reference-access";

export const locationCategoryTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: LocationCategoryTableDocument,
  operationName: "LocationCategoryTable",
  connectionKey: "locationCategories",
  mapNode: (node) =>
    rootFields(node, "LocationCategory", ["allowsOvernight", "hasRestroom", "hasSecureParking"]),
});
