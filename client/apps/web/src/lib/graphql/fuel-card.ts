import {
  CancelFuelCardDocument,
  CreateFuelCardDocument,
  FuelCardDocument,
  FuelCardTableDocument,
  UpdateFuelCardDocument,
  type CancelFuelCardInput,
  type CancelFuelCardMutation,
  type CancelFuelCardMutationVariables,
  type CreateFuelCardMutation,
  type CreateFuelCardMutationVariables,
  type FuelCardFieldsFragment,
  type FuelCardInput,
  type FuelCardQuery,
  type FuelCardQueryVariables,
  type UpdateFuelCardMutation,
  type UpdateFuelCardMutationVariables,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";

export type FuelCardRow = FuelCardFieldsFragment;

export const FUEL_CARD_LIST_KEY = "fuel-card-list";
export const FUEL_CARD_OPTIONS_KEY = "fuel-card-options";

export const fuelCardTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: FuelCardTableDocument,
  operationName: "FuelCardTable",
  connectionKey: "fuelCards",
});

export async function fetchFuelCard(
  id: string,
  options?: { signal?: AbortSignal },
): Promise<FuelCardRow> {
  const data = await requestGraphQL<FuelCardQuery, FuelCardQueryVariables>({
    document: FuelCardDocument,
    operationName: "FuelCard",
    variables: { id },
    signal: options?.signal,
  });
  return data.fuelCard as FuelCardRow;
}

export async function createFuelCard(input: FuelCardInput): Promise<FuelCardRow> {
  const data = await requestGraphQL<CreateFuelCardMutation, CreateFuelCardMutationVariables>({
    document: CreateFuelCardDocument,
    operationName: "CreateFuelCard",
    variables: { input },
  });
  return data.createFuelCard as FuelCardRow;
}

export async function updateFuelCard(
  id: string,
  version: number,
  input: FuelCardInput,
): Promise<FuelCardRow> {
  const data = await requestGraphQL<UpdateFuelCardMutation, UpdateFuelCardMutationVariables>({
    document: UpdateFuelCardDocument,
    operationName: "UpdateFuelCard",
    variables: { id, version, input },
  });
  return data.updateFuelCard as FuelCardRow;
}

export async function cancelFuelCard(input: CancelFuelCardInput): Promise<FuelCardRow> {
  const data = await requestGraphQL<CancelFuelCardMutation, CancelFuelCardMutationVariables>({
    document: CancelFuelCardDocument,
    operationName: "CancelFuelCard",
    variables: { input },
  });
  return data.cancelFuelCard as FuelCardRow;
}
