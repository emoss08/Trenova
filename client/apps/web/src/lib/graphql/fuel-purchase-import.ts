import {
  CommitFuelPurchaseImportDocument,
  CreateFuelPurchaseImportDocument,
  DiscardFuelPurchaseImportDocument,
  FuelPurchaseImportDocument,
  FuelPurchaseImportRowsDocument,
  FuelPurchaseImportTemplateDocument,
  StageFuelPurchaseImportDocument,
  type CommitFuelPurchaseImportMutation,
  type CommitFuelPurchaseImportMutationVariables,
  type CreateFuelPurchaseImportInput,
  type CreateFuelPurchaseImportMutation,
  type CreateFuelPurchaseImportMutationVariables,
  type DiscardFuelPurchaseImportMutation,
  type DiscardFuelPurchaseImportMutationVariables,
  type FuelCardProvider,
  type FuelPurchaseImportBatchFieldsFragment,
  type FuelPurchaseImportQuery,
  type FuelPurchaseImportQueryVariables,
  type FuelPurchaseImportRowFieldsFragment,
  type FuelPurchaseImportRowsInput,
  type FuelPurchaseImportRowsQuery,
  type FuelPurchaseImportRowsQueryVariables,
  type FuelPurchaseImportTemplateQuery,
  type FuelPurchaseImportTemplateQueryVariables,
  type StageFuelPurchaseImportInput,
  type StageFuelPurchaseImportMutation,
  type StageFuelPurchaseImportMutationVariables,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type FuelPurchaseImportBatch = FuelPurchaseImportBatchFieldsFragment;
export type FuelPurchaseImportRow = FuelPurchaseImportRowFieldsFragment;
export type FuelPurchaseImportRowsPage = {
  edges: Array<{ node: FuelPurchaseImportRow; cursor: string }>;
  totalCount: number | null;
  pageInfo: { hasNextPage: boolean; endCursor: string | null };
};
export type FuelImportTemplate = FuelPurchaseImportTemplateQuery["fuelPurchaseImportTemplate"];

export const FUEL_PURCHASE_IMPORT_KEY = "fuel-purchase-import";
export const FUEL_PURCHASE_IMPORT_ROWS_KEY = "fuel-purchase-import-rows";

export async function fetchFuelPurchaseImport(
  id: string,
  options?: { signal?: AbortSignal },
): Promise<FuelPurchaseImportBatch> {
  const data = await requestGraphQL<FuelPurchaseImportQuery, FuelPurchaseImportQueryVariables>({
    document: FuelPurchaseImportDocument,
    operationName: "FuelPurchaseImport",
    variables: { id },
    signal: options?.signal,
  });
  return data.fuelPurchaseImport as FuelPurchaseImportBatch;
}

export async function fetchFuelPurchaseImportRows(
  id: string,
  input: FuelPurchaseImportRowsInput,
  options?: { signal?: AbortSignal },
): Promise<FuelPurchaseImportRowsPage> {
  const data = await requestGraphQL<
    FuelPurchaseImportRowsQuery,
    FuelPurchaseImportRowsQueryVariables
  >({
    document: FuelPurchaseImportRowsDocument,
    operationName: "FuelPurchaseImportRows",
    variables: { id, input },
    signal: options?.signal,
  });
  return data.fuelPurchaseImport.rows as FuelPurchaseImportRowsPage;
}

export async function fetchFuelPurchaseImportTemplate(
  provider: FuelCardProvider,
  options?: { signal?: AbortSignal },
): Promise<FuelImportTemplate> {
  const data = await requestGraphQL<
    FuelPurchaseImportTemplateQuery,
    FuelPurchaseImportTemplateQueryVariables
  >({
    document: FuelPurchaseImportTemplateDocument,
    operationName: "FuelPurchaseImportTemplate",
    variables: { provider },
    signal: options?.signal,
  });
  return data.fuelPurchaseImportTemplate;
}

export async function createFuelPurchaseImport(
  input: CreateFuelPurchaseImportInput,
): Promise<FuelPurchaseImportBatch> {
  const data = await requestGraphQL<
    CreateFuelPurchaseImportMutation,
    CreateFuelPurchaseImportMutationVariables
  >({
    document: CreateFuelPurchaseImportDocument,
    operationName: "CreateFuelPurchaseImport",
    variables: { input },
  });
  return data.createFuelPurchaseImport as FuelPurchaseImportBatch;
}

export async function stageFuelPurchaseImport(
  input: StageFuelPurchaseImportInput,
): Promise<FuelPurchaseImportBatch> {
  const data = await requestGraphQL<
    StageFuelPurchaseImportMutation,
    StageFuelPurchaseImportMutationVariables
  >({
    document: StageFuelPurchaseImportDocument,
    operationName: "StageFuelPurchaseImport",
    variables: { input },
  });
  return data.stageFuelPurchaseImport as FuelPurchaseImportBatch;
}

export async function commitFuelPurchaseImport(
  id: string,
  version: number,
): Promise<FuelPurchaseImportBatch> {
  const data = await requestGraphQL<
    CommitFuelPurchaseImportMutation,
    CommitFuelPurchaseImportMutationVariables
  >({
    document: CommitFuelPurchaseImportDocument,
    operationName: "CommitFuelPurchaseImport",
    variables: { id, version },
  });
  return data.commitFuelPurchaseImport as FuelPurchaseImportBatch;
}

export async function discardFuelPurchaseImport(
  id: string,
  version: number,
  reason?: string,
): Promise<FuelPurchaseImportBatch> {
  const data = await requestGraphQL<
    DiscardFuelPurchaseImportMutation,
    DiscardFuelPurchaseImportMutationVariables
  >({
    document: DiscardFuelPurchaseImportDocument,
    operationName: "DiscardFuelPurchaseImport",
    variables: { id, version, reason: reason ?? null },
  });
  return data.discardFuelPurchaseImport as FuelPurchaseImportBatch;
}
