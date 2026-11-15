import {
  HazardousMaterialTableDocument,
} from "@trenova/graphql/generated/documents/hazardous-material/table";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";

export const hazardousMaterialTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: HazardousMaterialTableDocument,
  operationName: "HazardousMaterialTable",
  connectionKey: "hazardousMaterials",
});
