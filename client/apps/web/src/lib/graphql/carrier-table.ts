import {
  CarrierTableDocument,
  type CarrierTableQueryVariables,
} from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { Carrier } from "@trenova/shared/types/carrier";

export const carrierTableGraphQLConfig = defineDataTableGraphQLConfig<
  Carrier,
  CarrierTableQueryVariables
>({
  document: CarrierTableDocument,
  operationName: "CarrierTable",
  connectionKey: "carriers",
});
