import {
  LocationTableDocument,
  type LocationTableQueryVariables,
} from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { Location } from "@trenova/shared/types/location";

export const locationTableGraphQLConfig = defineDataTableGraphQLConfig<
  Location,
  LocationTableQueryVariables
>({
  document: LocationTableDocument,
  operationName: "LocationTable",
  connectionKey: "locations",
});
