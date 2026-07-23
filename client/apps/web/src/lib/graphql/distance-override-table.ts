import {
  DistanceOverrideTableDocument,
  type DistanceOverrideTableQueryVariables,
} from "@/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@/lib/graphql/data-table";
import type { DistanceOverride } from "@/types/distance-override";

export const distanceOverrideTableGraphQLConfig = defineDataTableGraphQLConfig<
  DistanceOverride,
  DistanceOverrideTableQueryVariables
>({
  document: DistanceOverrideTableDocument,
  operationName: "DistanceOverrideTable",
  connectionKey: "distanceOverrides",
});
