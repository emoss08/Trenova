import {
  CommitFuelPurchaseImportDocument,
  FuelPurchaseImportTableDocument,
  ResolveFuelPurchaseImportRowsDocument,
  CreateFuelPurchaseImportDocument,
  DiscardFuelPurchaseImportDocument,
  FuelPurchaseImportDocument,
  FuelPurchaseImportRowsDocument,
  FuelPurchaseImportTemplateDocument,
  StageFuelPurchaseImportDocument,
  type CommitFuelPurchaseImportMutation,
  type CommitFuelPurchaseImportMutationVariables,
  type ResolveFuelPurchaseImportRowsMutation,
  type ResolveFuelPurchaseImportRowsMutationVariables,
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
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";

export type FuelPurchaseImportBatch = FuelPurchaseImportBatchFieldsFragment;
export type FuelPurchaseImportRow = FuelPurchaseImportRowFieldsFragment;
export type FuelPurchaseImportRowsPage = {
  edges: Array<{ node: FuelPurchaseImportRow; cursor: string }>;
  totalCount: number | null;
  pageInfo: { hasNextPage: boolean; endCursor: string | null };
};
export type FuelImportTemplate = FuelPurchaseImportTemplateQuery["fuelPurchaseImportTemplate"];

export const FUEL_PURCHASE_IMPORT_KEY = "fuel-purchase-import";
export const FUEL_FEED_RUN_LIST_KEY = "fuel-feed-run-list";

/**
 * Every run a connected fuel card feed has opened, newest first. An upload is a
 * different thing entirely — somebody chose that file — so this view is narrowed
 * to Feed, and the two never mix.
 */
export const fuelFeedRunTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: FuelPurchaseImportTableDocument,
  operationName: "FuelPurchaseImportTable",
  connectionKey: "fuelPurchaseImports",
  inputExtraVariables: { origin: "Feed" },
});
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

export type FuelPurchaseImportResolveResult =
  ResolveFuelPurchaseImportRowsMutation["resolveFuelPurchaseImportRows"];

/**
 * Works the rows an import is still holding out again against the organization's
 * current cards, tractors and jurisdictions. This is what posts the rows a feed
 * could not place, once the card it discovered has been assigned.
 */
export async function resolveFuelPurchaseImportRows(
  id: string,
  version: number,
): Promise<FuelPurchaseImportResolveResult> {
  const data = await requestGraphQL<
    ResolveFuelPurchaseImportRowsMutation,
    ResolveFuelPurchaseImportRowsMutationVariables
  >({
    document: ResolveFuelPurchaseImportRowsDocument,
    operationName: "ResolveFuelPurchaseImportRows",
    variables: { id, version },
  });
  return data.resolveFuelPurchaseImportRows;
}
