import {
  StoredMileageTableDocument,
  type StoredMileageTableQueryVariables,
} from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@/lib/graphql/data-table";
import type { StoredMileage } from "@/types/stored-mileage";

export const storedMileageTableGraphQLConfig = defineDataTableGraphQLConfig<
  StoredMileage,
  StoredMileageTableQueryVariables
>({
  document: StoredMileageTableDocument,
  operationName: "StoredMileageTable",
  connectionKey: "storedMileages",
});
