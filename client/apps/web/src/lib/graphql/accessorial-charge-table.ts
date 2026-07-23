import {
  AccessorialChargeTableDocument,
  type AccessorialChargeTableQueryVariables,
} from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { AccessorialCharge } from "@trenova/shared/types/accessorial-charge";

export const accessorialChargeTableGraphQLConfig = defineDataTableGraphQLConfig<
  AccessorialCharge,
  AccessorialChargeTableQueryVariables
>({
  document: AccessorialChargeTableDocument,
  operationName: "AccessorialChargeTable",
  connectionKey: "accessorialCharges",
});
