import {
  AssignFuelCardDocument,
  CancelFuelCardDocument,
  CreateFuelCardDocument,
  FuelCardDocument,
  FuelCardTableDocument,
  SyncFuelCardFeedDocument,
  UpdateFuelCardDocument,
  type AssignFuelCardInput,
  type AssignFuelCardMutation,
  type AssignFuelCardMutationVariables,
  type CancelFuelCardInput,
  type CancelFuelCardMutation,
  type CancelFuelCardMutationVariables,
  type CreateFuelCardMutation,
  type CreateFuelCardMutationVariables,
  type FuelCardFieldsFragment,
  type FuelCardInput,
  type FuelCardQuery,
  type FuelCardQueryVariables,
  type FuelCardProvider,
  type SyncFuelCardFeedMutation,
  type SyncFuelCardFeedMutationVariables,
  type UpdateFuelCardMutation,
  type UpdateFuelCardMutationVariables,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";

export type FuelCardRow = FuelCardFieldsFragment;

export const FUEL_CARD_LIST_KEY = "fuel-card-list";
export const FUEL_CARD_OPTIONS_KEY = "fuel-card-options";
export const UNASSIGNED_FUEL_CARD_LIST_KEY = "unassigned-fuel-card-list";

export const fuelCardTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: FuelCardTableDocument,
  operationName: "FuelCardTable",
  connectionKey: "fuelCards",
});

/**
 * The same table narrowed to cards with neither a tractor nor a driver on them.
 * A feed creates cards in this state the first time it sees a transaction on
 * one, and they cannot match a purchase until somebody assigns them.
 */
export const unassignedFuelCardTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: FuelCardTableDocument,
  operationName: "FuelCardTable",
  connectionKey: "fuelCards",
  inputExtraVariables: { unassignedOnly: true },
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

export async function assignFuelCard(input: AssignFuelCardInput): Promise<FuelCardRow> {
  const data = await requestGraphQL<AssignFuelCardMutation, AssignFuelCardMutationVariables>({
    document: AssignFuelCardDocument,
    operationName: "AssignFuelCard",
    variables: { input },
  });
  return data.assignFuelCard as FuelCardRow;
}

export type FuelCardSyncResult = SyncFuelCardFeedMutation["syncFuelCardFeed"];

export async function syncFuelCardFeed(provider: FuelCardProvider): Promise<FuelCardSyncResult> {
  const data = await requestGraphQL<SyncFuelCardFeedMutation, SyncFuelCardFeedMutationVariables>({
    document: SyncFuelCardFeedDocument,
    operationName: "SyncFuelCardFeed",
    variables: { provider },
  });
  return data.syncFuelCardFeed;
}
