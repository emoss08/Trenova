import type {
  FuelIndexFieldsFragment,
  FuelSurchargeProgramFieldsFragment,
} from "@trenova/graphql/generated/graphql";
import {
  FuelIndexTableDocument,
  FuelSurchargeProgramTableDocument,
} from "@trenova/graphql/generated/documents/fuel-surcharge/operations";
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
