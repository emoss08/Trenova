import {
  FuelIndexTableDocument,
  FuelSurchargeProgramTableDocument,
  type FuelIndexFieldsFragment,
  type FuelSurchargeProgramFieldsFragment,
} from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";

export type FuelIndexRow = FuelIndexFieldsFragment;
export type FuelSurchargeProgramRow = FuelSurchargeProgramFieldsFragment;

export const fuelIndexTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: FuelIndexTableDocument,
  operationName: "FuelIndexTable",
  connectionKey: "fuelIndexes",
});

export const fuelSurchargeProgramTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: FuelSurchargeProgramTableDocument,
  operationName: "FuelSurchargeProgramTable",
  connectionKey: "fuelSurchargePrograms",
});
