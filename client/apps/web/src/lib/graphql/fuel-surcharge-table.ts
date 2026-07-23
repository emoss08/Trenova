import {
  FuelIndexTableDocument,
  FuelSurchargeProgramTableDocument,
  type FuelIndexTableQueryVariables,
  type FuelIndexFieldsFragment,
  type FuelSurchargeProgramFieldsFragment,
  type FuelSurchargeProgramTableQueryVariables,
} from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@/lib/graphql/data-table";

export type FuelIndexRow = FuelIndexFieldsFragment;
export type FuelSurchargeProgramRow = FuelSurchargeProgramFieldsFragment;

export const fuelIndexTableGraphQLConfig = defineDataTableGraphQLConfig<
  FuelIndexRow,
  FuelIndexTableQueryVariables
>({
  document: FuelIndexTableDocument,
  operationName: "FuelIndexTable",
  connectionKey: "fuelIndexes",
});

export const fuelSurchargeProgramTableGraphQLConfig = defineDataTableGraphQLConfig<
  FuelSurchargeProgramRow,
  FuelSurchargeProgramTableQueryVariables
>({
  document: FuelSurchargeProgramTableDocument,
  operationName: "FuelSurchargeProgramTable",
  connectionKey: "fuelSurchargePrograms",
});
