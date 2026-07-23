import {
  LocationTableDocument,
  type LocationTableQueryVariables,
} from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@/lib/graphql/data-table";
import type { Location } from "@/types/location";

export const locationTableGraphQLConfig = defineDataTableGraphQLConfig<
  Location,
  LocationTableQueryVariables
>({
  document: LocationTableDocument,
  operationName: "LocationTable",
  connectionKey: "locations",
});
