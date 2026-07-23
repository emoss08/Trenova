import {
  DistanceProfileTableDocument,
  type DistanceProfileTableQueryVariables,
} from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DistanceProfile } from "@/types/distance-profile";

export const distanceProfileTableGraphQLConfig = defineDataTableGraphQLConfig<
  DistanceProfile,
  DistanceProfileTableQueryVariables
>({
  document: DistanceProfileTableDocument,
  operationName: "DistanceProfileTable",
  connectionKey: "distanceProfiles",
});
