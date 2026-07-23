import {
  LocationCategoryTableDocument,
  type LocationCategoryTableQueryVariables,
} from "@/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@/lib/graphql/data-table";
import type { LocationCategory } from "@/types/location-category";

export const locationCategoryTableGraphQLConfig = defineDataTableGraphQLConfig<
  LocationCategory,
  LocationCategoryTableQueryVariables
>({
  document: LocationCategoryTableDocument,
  operationName: "LocationCategoryTable",
  connectionKey: "locationCategories",
});
