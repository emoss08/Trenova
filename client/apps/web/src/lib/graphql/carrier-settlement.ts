import {
  AcceptCarrierInvoiceMatchDocument,
  AcceptCarrierInvoiceMatchWithVarianceDocument,
  AddCarrierSettlementAdjustmentDocument,
  ApproveCarrierSettlementDocument,
  CarrierCostEventTableDocument,
  CarrierInvoiceMatchesDocument,
  CarrierLedgerEntriesDocument,
  CarrierSettlementBatchDetailDocument,
  CarrierSettlementBatchTableDocument,
  CarrierSettlementControlDocument,
  CarrierSettlementDetailDocument,
  CarrierSettlementTableDocument,
  CarrierSettlementWorkspaceSummaryDocument,
  CreateCarrierInvoiceMatchDocument,
  CurrentCarrierSettlementPeriodDocument,
  EdiCarrierInvoicesDocument,
  ExportCarrierSettlementBatchCsvDocument,
  GenerateCarrierSettlementBatchDocument,
  LinkEdiCarrierInvoiceToCarrierDocument,
  MarkCarrierSettlementPaidDocument,
  PostCarrierSettlementDocument,
  RecalculateCarrierSettlementDocument,
  RejectCarrierInvoiceMatchDocument,
  RejectCarrierSettlementDocument,
  RemoveCarrierSettlementAdjustmentDocument,
  SubmitCarrierSettlementDocument,
  SuggestCarrierForEdiInvoiceDocument,
  UpdateCarrierSettlementControlDocument,
  VoidCarrierSettlementDocument,
  type AddCarrierSettlementAdjustmentInput,
  type CarrierCostEventTableQuery,
  type CarrierCostEventTableQueryVariables,
  type CarrierInvoiceMatchActionInput,
  type CarrierInvoiceMatchesQuery,
  type CarrierInvoiceMatchStatus,
  type CarrierLedgerEntriesQuery,
  type CarrierSettlementActionInput,
  type CarrierSettlementBatchDetailQuery,
  type CarrierSettlementBatchTableQuery,
  type CarrierSettlementBatchTableQueryVariables,
  type CarrierSettlementControlQuery,
  type CarrierSettlementDetailQuery,
  type CarrierSettlementTableQuery,
  type CarrierSettlementTableQueryVariables,
  type CarrierSettlementWorkspaceSummaryQuery,
  type CreateCarrierInvoiceMatchInput,
  type EdiCarrierInvoicesQuery,
  type GenerateCarrierSettlementBatchInput,
  type MarkCarrierSettlementPaidInput,
  type RemoveCarrierSettlementAdjustmentInput,
  type SuggestCarrierForEdiInvoiceQuery,
  type UpdateCarrierSettlementControlInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { QueryClient } from "@tanstack/react-query";

/**
 * Invalidates every query the carrier settlement workspace renders — the queue
 * and summary, the selected detail, the history table, and the context rail
 * (cost events, recent settlements, AP ledger). Any surface that mutates a
 * carrier settlement must use this so the rail refreshes with the money.
 */
export function invalidateCarrierWorkspace(queryClient: QueryClient) {
  const prefixes = [
    "carrier-settlement-workspace-summary",
    "carrier-settlement-workspace-settlements",
    "carrier-settlement-detail",
    "carrier-settlement-list",
    "carrier-cost-event-list",
    "carrier-pending-cost-events",
    "carrier-recent-settlements",
    "carrier-ledger-entries",
  ];
  for (const prefix of prefixes) {
    void queryClient.invalidateQueries({ queryKey: [prefix] });
  }
}

export type CarrierSettlementRow = NonNullable<
  CarrierSettlementTableQuery["carrierSettlements"]["edges"]
>[number]["node"];
export type CarrierSettlementBatchRow = NonNullable<
  CarrierSettlementBatchTableQuery["carrierSettlementBatches"]["edges"]
>[number]["node"];
export type CarrierCostEventRow = NonNullable<
  CarrierCostEventTableQuery["carrierCostEvents"]["edges"]
>[number]["node"];
export type CarrierSettlementDetail = NonNullable<
  CarrierSettlementDetailQuery["carrierSettlement"]
>;
export type CarrierSettlementLineRow = NonNullable<CarrierSettlementDetail["lines"]>[number];
export type CarrierSettlementBatchDetail = NonNullable<
  CarrierSettlementBatchDetailQuery["carrierSettlementBatch"]
>;
export type CarrierSettlementControl = CarrierSettlementControlQuery["carrierSettlementControl"];
export type CarrierSettlementWorkspaceSummary =
  CarrierSettlementWorkspaceSummaryQuery["carrierSettlementWorkspaceSummary"];
export type CarrierLedgerEntryRow = CarrierLedgerEntriesQuery["carrierLedgerEntries"][number];
export type CarrierInvoiceMatchRow =
  CarrierInvoiceMatchesQuery["carrierInvoiceMatches"]["items"][number];
export type EdiCarrierInvoiceRow = EdiCarrierInvoicesQuery["ediCarrierInvoices"]["items"][number];
export type SuggestedCarrier = NonNullable<
  SuggestCarrierForEdiInvoiceQuery["suggestCarrierForEdiInvoice"]
>;

export const carrierSettlementTableGraphQLConfig = defineDataTableGraphQLConfig<
  CarrierSettlementRow,
  CarrierSettlementTableQueryVariables
>({
  document: CarrierSettlementTableDocument,
  operationName: "CarrierSettlementTable",
  connectionKey: "carrierSettlements",
});

export const carrierSettlementBatchTableGraphQLConfig = defineDataTableGraphQLConfig<
  CarrierSettlementBatchRow,
  CarrierSettlementBatchTableQueryVariables
>({
  document: CarrierSettlementBatchTableDocument,
  operationName: "CarrierSettlementBatchTable",
  connectionKey: "carrierSettlementBatches",
});

export const carrierCostEventTableGraphQLConfig = defineDataTableGraphQLConfig<
  CarrierCostEventRow,
  CarrierCostEventTableQueryVariables
>({
  document: CarrierCostEventTableDocument,
  operationName: "CarrierCostEventTable",
  connectionKey: "carrierCostEvents",
});

export async function fetchCarrierSettlementDetail(id: string) {
  const data = await requestGraphQL({
    document: CarrierSettlementDetailDocument,
    operationName: "CarrierSettlementDetail",
    variables: { id },
  });
  return data.carrierSettlement;
}

export async function fetchCarrierSettlementBatchDetail(id: string) {
  const data = await requestGraphQL({
    document: CarrierSettlementBatchDetailDocument,
    operationName: "CarrierSettlementBatchDetail",
    variables: { id },
  });
  return data.carrierSettlementBatch;
}

export async function fetchCarrierSettlementControl() {
  const data = await requestGraphQL({
    document: CarrierSettlementControlDocument,
    operationName: "CarrierSettlementControl",
  });
  return data.carrierSettlementControl;
}

export async function fetchCurrentCarrierSettlementPeriod() {
  const data = await requestGraphQL({
    document: CurrentCarrierSettlementPeriodDocument,
    operationName: "CurrentCarrierSettlementPeriod",
  });
  return data.currentCarrierSettlementPeriod;
}

export async function fetchCarrierSettlementWorkspaceSummary(
  periodStart?: number,
  periodEnd?: number,
) {
  const data = await requestGraphQL({
    document: CarrierSettlementWorkspaceSummaryDocument,
    operationName: "CarrierSettlementWorkspaceSummary",
    variables: { periodStart, periodEnd },
  });
  return data.carrierSettlementWorkspaceSummary;
}

export async function fetchWorkspaceCarrierSettlements(periodStart: number, periodEnd: number) {
  const data = await requestGraphQL({
    document: CarrierSettlementTableDocument,
    operationName: "CarrierSettlementTable",
    variables: {
      input: {
        first: 200,
        fieldFilters: [
          { field: "periodStart", operator: "eq", value: periodStart },
          { field: "periodEnd", operator: "eq", value: periodEnd },
        ],
        sort: [{ field: "createdAt", direction: "asc" }],
      },
    },
  });
  return (data.carrierSettlements.edges ?? []).map((edge) => edge.node);
}

export async function fetchCarrierRecentSettlements(carrierId: string, limit = 5) {
  const data = await requestGraphQL({
    document: CarrierSettlementTableDocument,
    operationName: "CarrierSettlementTable",
    variables: {
      input: {
        first: limit,
        fieldFilters: [{ field: "carrierId", operator: "eq", value: carrierId }],
        sort: [{ field: "createdAt", direction: "desc" }],
      },
    },
  });
  return (data.carrierSettlements.edges ?? []).map((edge) => edge.node);
}

export async function fetchCarrierPendingCostEvents(carrierId: string) {
  const data = await requestGraphQL({
    document: CarrierCostEventTableDocument,
    operationName: "CarrierCostEventTable",
    variables: {
      input: {
        first: 100,
        fieldFilters: [
          { field: "carrierId", operator: "eq", value: carrierId },
          { field: "status", operator: "eq", value: "Pending" },
        ],
        sort: [{ field: "eventDate", direction: "desc" }],
      },
    },
  });
  return (data.carrierCostEvents.edges ?? []).map((edge) => edge.node);
}

export async function fetchCarrierLedgerEntries(carrierId: string, limit = 100) {
  const data = await requestGraphQL({
    document: CarrierLedgerEntriesDocument,
    operationName: "CarrierLedgerEntries",
    variables: { carrierId, limit },
  });
  return data.carrierLedgerEntries;
}

export async function fetchCarrierInvoiceMatches(params?: {
  status?: CarrierInvoiceMatchStatus;
  carrierId?: string;
  limit?: number;
  offset?: number;
}) {
  const data = await requestGraphQL({
    document: CarrierInvoiceMatchesDocument,
    operationName: "CarrierInvoiceMatches",
    variables: {
      status: params?.status,
      carrierId: params?.carrierId,
      limit: params?.limit ?? 100,
      offset: params?.offset,
    },
  });
  return data.carrierInvoiceMatches;
}

export async function fetchEdiCarrierInvoices(params?: {
  reconciliationStatus?: string;
  limit?: number;
  offset?: number;
}) {
  const data = await requestGraphQL({
    document: EdiCarrierInvoicesDocument,
    operationName: "EdiCarrierInvoices",
    variables: {
      reconciliationStatus: params?.reconciliationStatus,
      limit: params?.limit ?? 100,
      offset: params?.offset,
    },
  });
  return data.ediCarrierInvoices;
}

export async function suggestCarrierForEdiInvoice(invoiceId: string) {
  const data = await requestGraphQL({
    document: SuggestCarrierForEdiInvoiceDocument,
    operationName: "SuggestCarrierForEdiInvoice",
    variables: { invoiceId },
  });
  return data.suggestCarrierForEdiInvoice;
}

export async function exportCarrierSettlementBatchCsv(batchId: string) {
  const data = await requestGraphQL({
    document: ExportCarrierSettlementBatchCsvDocument,
    operationName: "ExportCarrierSettlementBatchCsv",
    variables: { batchId },
  });
  return data.exportCarrierSettlementBatchCsv;
}

export async function generateCarrierSettlementBatch(input: GenerateCarrierSettlementBatchInput) {
  const data = await requestGraphQL({
    document: GenerateCarrierSettlementBatchDocument,
    operationName: "GenerateCarrierSettlementBatch",
    variables: { input },
  });
  return data.generateCarrierSettlementBatch;
}

export async function submitCarrierSettlement(input: CarrierSettlementActionInput) {
  const data = await requestGraphQL({
    document: SubmitCarrierSettlementDocument,
    operationName: "SubmitCarrierSettlement",
    variables: { input },
  });
  return data.submitCarrierSettlement;
}

export async function approveCarrierSettlement(input: CarrierSettlementActionInput) {
  const data = await requestGraphQL({
    document: ApproveCarrierSettlementDocument,
    operationName: "ApproveCarrierSettlement",
    variables: { input },
  });
  return data.approveCarrierSettlement;
}

export async function rejectCarrierSettlement(input: CarrierSettlementActionInput) {
  const data = await requestGraphQL({
    document: RejectCarrierSettlementDocument,
    operationName: "RejectCarrierSettlement",
    variables: { input },
  });
  return data.rejectCarrierSettlement;
}

export async function postCarrierSettlement(input: CarrierSettlementActionInput) {
  const data = await requestGraphQL({
    document: PostCarrierSettlementDocument,
    operationName: "PostCarrierSettlement",
    variables: { input },
  });
  return data.postCarrierSettlement;
}

export async function markCarrierSettlementPaid(input: MarkCarrierSettlementPaidInput) {
  const data = await requestGraphQL({
    document: MarkCarrierSettlementPaidDocument,
    operationName: "MarkCarrierSettlementPaid",
    variables: { input },
  });
  return data.markCarrierSettlementPaid;
}

export async function voidCarrierSettlement(input: CarrierSettlementActionInput) {
  const data = await requestGraphQL({
    document: VoidCarrierSettlementDocument,
    operationName: "VoidCarrierSettlement",
    variables: { input },
  });
  return data.voidCarrierSettlement;
}

export async function recalculateCarrierSettlement(input: CarrierSettlementActionInput) {
  const data = await requestGraphQL({
    document: RecalculateCarrierSettlementDocument,
    operationName: "RecalculateCarrierSettlement",
    variables: { input },
  });
  return data.recalculateCarrierSettlement;
}

export async function addCarrierSettlementAdjustment(input: AddCarrierSettlementAdjustmentInput) {
  const data = await requestGraphQL({
    document: AddCarrierSettlementAdjustmentDocument,
    operationName: "AddCarrierSettlementAdjustment",
    variables: { input },
  });
  return data.addCarrierSettlementAdjustment;
}

export async function removeCarrierSettlementAdjustment(
  input: RemoveCarrierSettlementAdjustmentInput,
) {
  const data = await requestGraphQL({
    document: RemoveCarrierSettlementAdjustmentDocument,
    operationName: "RemoveCarrierSettlementAdjustment",
    variables: { input },
  });
  return data.removeCarrierSettlementAdjustment;
}

export async function updateCarrierSettlementControl(input: UpdateCarrierSettlementControlInput) {
  const data = await requestGraphQL({
    document: UpdateCarrierSettlementControlDocument,
    operationName: "UpdateCarrierSettlementControl",
    variables: { input },
  });
  return data.updateCarrierSettlementControl;
}

export async function linkEdiCarrierInvoiceToCarrier(invoiceId: string, carrierId: string) {
  const data = await requestGraphQL({
    document: LinkEdiCarrierInvoiceToCarrierDocument,
    operationName: "LinkEdiCarrierInvoiceToCarrier",
    variables: { invoiceId, carrierId },
  });
  return data.linkEdiCarrierInvoiceToCarrier;
}

export async function createCarrierInvoiceMatch(input: CreateCarrierInvoiceMatchInput) {
  const data = await requestGraphQL({
    document: CreateCarrierInvoiceMatchDocument,
    operationName: "CreateCarrierInvoiceMatch",
    variables: { input },
  });
  return data.createCarrierInvoiceMatch;
}

export async function acceptCarrierInvoiceMatch(input: CarrierInvoiceMatchActionInput) {
  const data = await requestGraphQL({
    document: AcceptCarrierInvoiceMatchDocument,
    operationName: "AcceptCarrierInvoiceMatch",
    variables: { input },
  });
  return data.acceptCarrierInvoiceMatch;
}

export async function acceptCarrierInvoiceMatchWithVariance(input: CarrierInvoiceMatchActionInput) {
  const data = await requestGraphQL({
    document: AcceptCarrierInvoiceMatchWithVarianceDocument,
    operationName: "AcceptCarrierInvoiceMatchWithVariance",
    variables: { input },
  });
  return data.acceptCarrierInvoiceMatchWithVariance;
}

export async function rejectCarrierInvoiceMatch(input: CarrierInvoiceMatchActionInput) {
  const data = await requestGraphQL({
    document: RejectCarrierInvoiceMatchDocument,
    operationName: "RejectCarrierInvoiceMatch",
    variables: { input },
  });
  return data.rejectCarrierInvoiceMatch;
}
