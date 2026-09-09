import {
  CreateFuelPurchaseDocument,
  DeleteFuelPurchaseDocument,
  FuelPurchaseDocument,
  FuelPurchaseTableDocument,
  UpdateFuelPurchaseDocument,
  type CreateFuelPurchaseMutation,
  type CreateFuelPurchaseMutationVariables,
  type DeleteFuelPurchaseMutation,
  type DeleteFuelPurchaseMutationVariables,
  type FuelPurchaseFieldsFragment,
  type FuelPurchaseInput,
  type FuelPurchaseQuery,
  type FuelPurchaseQueryVariables,
  type UpdateFuelPurchaseMutation,
  type UpdateFuelPurchaseMutationVariables,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";

export type FuelPurchaseRow = FuelPurchaseFieldsFragment;

export const FUEL_PURCHASE_LIST_KEY = "fuel-purchase-list";

export const fuelPurchaseTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: FuelPurchaseTableDocument,
  operationName: "FuelPurchaseTable",
  connectionKey: "fuelPurchases",
});

export async function fetchFuelPurchase(
  id: string,
  options?: { signal?: AbortSignal },
): Promise<FuelPurchaseRow> {
  const data = await requestGraphQL<FuelPurchaseQuery, FuelPurchaseQueryVariables>({
    document: FuelPurchaseDocument,
    operationName: "FuelPurchase",
    variables: { id },
    signal: options?.signal,
  });
  return data.fuelPurchase as FuelPurchaseRow;
}

export async function createFuelPurchase(input: FuelPurchaseInput): Promise<FuelPurchaseRow> {
  const data = await requestGraphQL<
    CreateFuelPurchaseMutation,
    CreateFuelPurchaseMutationVariables
  >({
    document: CreateFuelPurchaseDocument,
    operationName: "CreateFuelPurchase",
    variables: { input },
  });
  return data.createFuelPurchase as FuelPurchaseRow;
}

export async function updateFuelPurchase(
  id: string,
  version: number,
  input: FuelPurchaseInput,
): Promise<FuelPurchaseRow> {
  const data = await requestGraphQL<
    UpdateFuelPurchaseMutation,
    UpdateFuelPurchaseMutationVariables
  >({
    document: UpdateFuelPurchaseDocument,
    operationName: "UpdateFuelPurchase",
    variables: { id, version, input },
  });
  return data.updateFuelPurchase as FuelPurchaseRow;
}

export async function deleteFuelPurchase(id: string, version: number): Promise<boolean> {
  const data = await requestGraphQL<
    DeleteFuelPurchaseMutation,
    DeleteFuelPurchaseMutationVariables
  >({
    document: DeleteFuelPurchaseDocument,
    operationName: "DeleteFuelPurchase",
    variables: { id, version },
  });
  return data.deleteFuelPurchase;
}
