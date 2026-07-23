import {
  AccessorialChargeTableDocument,
  type AccessorialChargeTableQueryVariables,
} from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@/lib/graphql/data-table";
import type { AccessorialCharge } from "@/types/accessorial-charge";

export const accessorialChargeTableGraphQLConfig = defineDataTableGraphQLConfig<
  AccessorialCharge,
  AccessorialChargeTableQueryVariables
>({
  document: AccessorialChargeTableDocument,
  operationName: "AccessorialChargeTable",
  connectionKey: "accessorialCharges",
});
