import {
  HazardousMaterialTableDocument,
  type HazardousMaterialTableQueryVariables,
} from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { HazardousMaterial } from "@/types/hazardous-material";

export const hazardousMaterialTableGraphQLConfig = defineDataTableGraphQLConfig<
  HazardousMaterial,
  HazardousMaterialTableQueryVariables
>({
  document: HazardousMaterialTableDocument,
  operationName: "HazardousMaterialTable",
  connectionKey: "hazardousMaterials",
});
