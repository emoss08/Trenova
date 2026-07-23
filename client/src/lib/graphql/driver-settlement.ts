import {
  AddDriverSettlementAdjustmentDocument,
  AdjustEscrowAccountDocument,
  ApproveDriverSettlementDocument,
  AssignPayProfileToWorkerDocument,
  AttachPayEventsToSettlementDocument,
  BulkDriverSettlementActionDocument,
  CloseEscrowAccountDocument,
  DetachPayEventFromSettlementDocument,
  HoldDriverPayEventDocument,
  ReleaseDriverPayEventDocument,
  SettlementWorkspaceSummaryDocument,
  UnsettledWorkerSummariesDocument,
  CreatePayProfileDocument,
  CreatePayCodeDocument,
  CreateRecurringDeductionDocument,
  CreateRecurringEarningDocument,
  CurrentSettlementPeriodDocument,
  DriverPayEventTableDocument,
  DriverSettlementDetailDocument,
  DriverSettlementTableDocument,
  EffectiveWorkerPayAssignmentDocument,
  EndWorkerPayAssignmentDocument,
  EscrowAccountDetailDocument,
  EscrowAccountTableDocument,
  ExportSettlementBatchCsvDocument,
  GenerateDriverSettlementDocument,
  GenerateSettlementBatchDocument,
  IssuePayAdvanceDocument,
  MarkDriverSettlementPaidDocument,
  PayWorkerNowDocument,
  OpenEscrowAccountDocument,
  PayAdvanceTableDocument,
  PayCodeOptionsDocument,
  PayCodeTableDocument,
  PayProfileAssignmentsDocument,
  PayProfileDetailDocument,
  PayProfileOptionsDocument,
  PayProfileTableDocument,
  PostDriverSettlementDocument,
  UnsettledPayEventsDocument,
  PreviewDriverSettlementDocument,
  RecalculateDriverSettlementDocument,
  RecurringDeductionTableDocument,
  RecurringEarningTableDocument,
  RejectDriverSettlementDocument,
  RemoveDriverSettlementAdjustmentDocument,
  SettlementBatchTableDocument,
  SettlementControlDocument,
  SubmitDriverSettlementDocument,
  UpdatePayProfileDocument,
  UpdatePayCodeDocument,
  UpdateRecurringDeductionDocument,
  UpdateRecurringEarningDocument,
  UpdateSettlementControlDocument,
  VoidDriverSettlementDocument,
  WorkerEarningsSummaryDocument,
  WorkerPayAssignmentsDocument,
  WorkerYtdPaySummariesDocument,
  WriteOffPayAdvanceDocument,
  type AddSettlementAdjustmentInput,
  type AdjustEscrowAccountInput,
  type AssignPayProfileInput,
  type AttachPayEventsInput,
  type BulkSettlementActionInput,
  type DetachPayEventInput,
  type HoldPayEventInput,
  type SettlementWorkspaceSummaryQuery,
  type UnsettledWorkerSummariesQuery,
  type CreatePayProfileInput,
  type CreatePayCodeInput,
  type CreateRecurringDeductionInput,
  type CreateRecurringEarningInput,
  type DriverPayEventTableQuery,
  type DriverPayEventTableQueryVariables,
  type DriverSettlementActionInput,
  type DriverSettlementDetailQuery,
  type DriverSettlementTableQuery,
  type DriverSettlementTableQueryVariables,
  type EffectiveWorkerPayAssignmentQuery,
  type EndWorkerPayAssignmentInput,
  type EscrowAccountTableQuery,
  type EscrowAccountTableQueryVariables,
  type GenerateDriverSettlementInput,
  type GenerateSettlementBatchInput,
  type IssuePayAdvanceInput,
  type MarkDriverSettlementPaidInput,
  type PayWorkerNowInput,
  type UnsettledPayEventsQuery,
  type OpenEscrowAccountInput,
  type PayAdvanceTableQuery,
  type PayAdvanceTableQueryVariables,
  type PayProfileAssignmentsQuery,
  type PayProfileDetailQuery,
  type PayProfileTableQuery,
  type PayProfileTableQueryVariables,
  type PayeeClassification,
  type PayCodeOptionsQuery,
  type PayCodeTableQuery,
  type PayCodeTableQueryVariables,
  type RecurringDeductionTableQuery,
  type RecurringDeductionTableQueryVariables,
  type RecurringEarningTableQuery,
  type RecurringEarningTableQueryVariables,
  type RemoveSettlementAdjustmentInput,
  type SettlementBatchTableQuery,
  type SettlementBatchTableQueryVariables,
  type UpdatePayProfileInput,
  type UpdatePayCodeInput,
  type UpdateRecurringDeductionInput,
  type UpdateRecurringEarningInput,
  type UpdateSettlementControlInput,
  type WorkerPayAssignmentsQuery,
  type WriteOffPayAdvanceInput,
} from "@/graphql/generated/graphql";
import { requestGraphQL } from "@/lib/graphql";
import { defineDataTableGraphQLConfig } from "@/lib/graphql/data-table";

export type PayProfileRow = NonNullable<
  PayProfileTableQuery["payProfiles"]["edges"]
>[number]["node"];
export type PayProfileComponentRow = NonNullable<PayProfileRow["components"]>[number];
export type RecurringDeductionRow = NonNullable<
  RecurringDeductionTableQuery["recurringDeductions"]["edges"]
>[number]["node"];
export type PayCodeRow = NonNullable<PayCodeTableQuery["payCodes"]["edges"]>[number]["node"];
export type PayCodeOption = PayCodeOptionsQuery["payCodeOptions"][number];
export type RecurringEarningRow = NonNullable<
  RecurringEarningTableQuery["recurringEarnings"]["edges"]
>[number]["node"];
export type PayAdvanceRow = NonNullable<
  PayAdvanceTableQuery["payAdvances"]["edges"]
>[number]["node"];
export type EscrowAccountRow = NonNullable<
  EscrowAccountTableQuery["escrowAccounts"]["edges"]
>[number]["node"];
export type DriverSettlementRow = NonNullable<
  DriverSettlementTableQuery["driverSettlements"]["edges"]
>[number]["node"];
export type SettlementBatchRow = NonNullable<
  SettlementBatchTableQuery["settlementBatches"]["edges"]
>[number]["node"];
export type DriverPayEventRow = NonNullable<
  DriverPayEventTableQuery["driverPayEvents"]["edges"]
>[number]["node"];
export type DriverSettlementDetail = NonNullable<DriverSettlementDetailQuery["driverSettlement"]>;
export type DriverSettlementLineRow = NonNullable<DriverSettlementDetail["lines"]>[number];
export type WorkerPayAssignmentRow = NonNullable<
  WorkerPayAssignmentsQuery["workerPayAssignments"]
>[number];
export type EffectiveWorkerPayAssignment = NonNullable<
  EffectiveWorkerPayAssignmentQuery["effectiveWorkerPayAssignment"]
>;
export type PayProfileAssignmentRow = NonNullable<
  PayProfileAssignmentsQuery["payProfileAssignments"]
>[number];
export type PayProfileDetail = NonNullable<PayProfileDetailQuery["payProfile"]>;
export type SettlementWorkspaceSummary =
  SettlementWorkspaceSummaryQuery["settlementWorkspaceSummary"];
export type UnsettledWorkerSummary =
  UnsettledWorkerSummariesQuery["unsettledWorkerSummaries"][number];

export const payProfileTableGraphQLConfig = defineDataTableGraphQLConfig<
  PayProfileRow,
  PayProfileTableQueryVariables
>({
  document: PayProfileTableDocument,
  operationName: "PayProfileTable",
  connectionKey: "payProfiles",
});

export const recurringDeductionTableGraphQLConfig = defineDataTableGraphQLConfig<
  RecurringDeductionRow,
  RecurringDeductionTableQueryVariables
>({
  document: RecurringDeductionTableDocument,
  operationName: "RecurringDeductionTable",
  connectionKey: "recurringDeductions",
});

export const payCodeTableGraphQLConfig = defineDataTableGraphQLConfig<
  PayCodeRow,
  PayCodeTableQueryVariables
>({
  document: PayCodeTableDocument,
  operationName: "PayCodeTable",
  connectionKey: "payCodes",
});

export const recurringEarningTableGraphQLConfig = defineDataTableGraphQLConfig<
  RecurringEarningRow,
  RecurringEarningTableQueryVariables
>({
  document: RecurringEarningTableDocument,
  operationName: "RecurringEarningTable",
  connectionKey: "recurringEarnings",
});

export const payAdvanceTableGraphQLConfig = defineDataTableGraphQLConfig<
  PayAdvanceRow,
  PayAdvanceTableQueryVariables
>({
  document: PayAdvanceTableDocument,
  operationName: "PayAdvanceTable",
  connectionKey: "payAdvances",
});

export const escrowAccountTableGraphQLConfig = defineDataTableGraphQLConfig<
  EscrowAccountRow,
  EscrowAccountTableQueryVariables
>({
  document: EscrowAccountTableDocument,
  operationName: "EscrowAccountTable",
  connectionKey: "escrowAccounts",
});

export const driverSettlementTableGraphQLConfig = defineDataTableGraphQLConfig<
  DriverSettlementRow,
  DriverSettlementTableQueryVariables
>({
  document: DriverSettlementTableDocument,
  operationName: "DriverSettlementTable",
  connectionKey: "driverSettlements",
});

export const settlementBatchTableGraphQLConfig = defineDataTableGraphQLConfig<
  SettlementBatchRow,
  SettlementBatchTableQueryVariables
>({
  document: SettlementBatchTableDocument,
  operationName: "SettlementBatchTable",
  connectionKey: "settlementBatches",
});

export const driverPayEventTableGraphQLConfig = defineDataTableGraphQLConfig<
  DriverPayEventRow,
  DriverPayEventTableQueryVariables
>({
  document: DriverPayEventTableDocument,
  operationName: "DriverPayEventTable",
  connectionKey: "driverPayEvents",
});

export async function fetchPayProfileOptions(query?: string) {
  const data = await requestGraphQL({
    document: PayProfileOptionsDocument,
    operationName: "PayProfileOptions",
    variables: {
      input: {
        first: 50,
        query: query || undefined,
      },
    },
  });
  return (data.payProfiles.edges ?? []).map((edge) => edge.node);
}

export async function fetchEffectiveWorkerPayAssignment(workerId: string) {
  const data = await requestGraphQL({
    document: EffectiveWorkerPayAssignmentDocument,
    operationName: "EffectiveWorkerPayAssignment",
    variables: { workerId },
  });
  return data.effectiveWorkerPayAssignment;
}

export async function fetchPayProfileAssignments(payProfileId: string) {
  const data = await requestGraphQL({
    document: PayProfileAssignmentsDocument,
    operationName: "PayProfileAssignments",
    variables: { payProfileId },
  });
  return data.payProfileAssignments;
}

export async function fetchPayProfileDetail(id: string) {
  const data = await requestGraphQL({
    document: PayProfileDetailDocument,
    operationName: "PayProfileDetail",
    variables: { id },
  });
  return data.payProfile;
}

export async function fetchWorkerPayAssignments(workerId: string) {
  const data = await requestGraphQL({
    document: WorkerPayAssignmentsDocument,
    operationName: "WorkerPayAssignments",
    variables: { workerId },
  });
  return data.workerPayAssignments;
}

export async function fetchDriverSettlementDetail(id: string) {
  const data = await requestGraphQL({
    document: DriverSettlementDetailDocument,
    operationName: "DriverSettlementDetail",
    variables: { id },
  });
  return data.driverSettlement;
}

export async function fetchEscrowAccountDetail(id: string) {
  const data = await requestGraphQL({
    document: EscrowAccountDetailDocument,
    operationName: "EscrowAccountDetail",
    variables: { id },
  });
  return data.escrowAccount;
}

export async function fetchWorkerEarningsSummary(workerId: string) {
  const data = await requestGraphQL({
    document: WorkerEarningsSummaryDocument,
    operationName: "WorkerEarningsSummary",
    variables: { workerId },
  });
  return data.workerEarningsSummary;
}

export async function fetchWorkerYtdPaySummaries(
  year: number,
  classification?: PayeeClassification,
) {
  const data = await requestGraphQL({
    document: WorkerYtdPaySummariesDocument,
    operationName: "WorkerYtdPaySummaries",
    variables: { year, classification },
  });
  return data.workerYtdPaySummaries;
}

export async function fetchSettlementControl() {
  const data = await requestGraphQL({
    document: SettlementControlDocument,
    operationName: "SettlementControl",
  });
  return data.settlementControl;
}

export async function fetchCurrentSettlementPeriod() {
  const data = await requestGraphQL({
    document: CurrentSettlementPeriodDocument,
    operationName: "CurrentSettlementPeriod",
  });
  return data.currentSettlementPeriod;
}

export async function fetchPreviewDriverSettlement(
  workerId: string,
  periodStart?: number,
  periodEnd?: number,
) {
  const data = await requestGraphQL({
    document: PreviewDriverSettlementDocument,
    operationName: "PreviewDriverSettlement",
    variables: { workerId, periodStart, periodEnd },
  });
  return data.previewDriverSettlement;
}

export async function exportSettlementBatchCsv(batchId: string) {
  const data = await requestGraphQL({
    document: ExportSettlementBatchCsvDocument,
    operationName: "ExportSettlementBatchCsv",
    variables: { batchId },
  });
  return data.exportSettlementBatchCsv;
}

export async function createPayProfile(input: CreatePayProfileInput) {
  const data = await requestGraphQL({
    document: CreatePayProfileDocument,
    operationName: "CreatePayProfile",
    variables: { input },
  });
  return data.createPayProfile;
}

export async function updatePayProfile(input: UpdatePayProfileInput) {
  const data = await requestGraphQL({
    document: UpdatePayProfileDocument,
    operationName: "UpdatePayProfile",
    variables: { input },
  });
  return data.updatePayProfile;
}

export async function assignPayProfileToWorker(input: AssignPayProfileInput) {
  const data = await requestGraphQL({
    document: AssignPayProfileToWorkerDocument,
    operationName: "AssignPayProfileToWorker",
    variables: { input },
  });
  return data.assignPayProfileToWorker;
}

export async function endWorkerPayAssignment(input: EndWorkerPayAssignmentInput) {
  const data = await requestGraphQL({
    document: EndWorkerPayAssignmentDocument,
    operationName: "EndWorkerPayAssignment",
    variables: { input },
  });
  return data.endWorkerPayAssignment;
}

export async function createRecurringDeduction(input: CreateRecurringDeductionInput) {
  const data = await requestGraphQL({
    document: CreateRecurringDeductionDocument,
    operationName: "CreateRecurringDeduction",
    variables: { input },
  });
  return data.createRecurringDeduction;
}

export async function updateRecurringDeduction(input: UpdateRecurringDeductionInput) {
  const data = await requestGraphQL({
    document: UpdateRecurringDeductionDocument,
    operationName: "UpdateRecurringDeduction",
    variables: { input },
  });
  return data.updateRecurringDeduction;
}

export async function fetchPayCodeOptions(direction?: "Earning" | "Deduction") {
  const data = await requestGraphQL({
    document: PayCodeOptionsDocument,
    operationName: "PayCodeOptions",
    variables: { direction },
  });
  return data.payCodeOptions;
}

export async function createPayCode(input: CreatePayCodeInput) {
  const data = await requestGraphQL({
    document: CreatePayCodeDocument,
    operationName: "CreatePayCode",
    variables: { input },
  });
  return data.createPayCode;
}

export async function updatePayCode(input: UpdatePayCodeInput) {
  const data = await requestGraphQL({
    document: UpdatePayCodeDocument,
    operationName: "UpdatePayCode",
    variables: { input },
  });
  return data.updatePayCode;
}

export async function createRecurringEarning(input: CreateRecurringEarningInput) {
  const data = await requestGraphQL({
    document: CreateRecurringEarningDocument,
    operationName: "CreateRecurringEarning",
    variables: { input },
  });
  return data.createRecurringEarning;
}

export async function updateRecurringEarning(input: UpdateRecurringEarningInput) {
  const data = await requestGraphQL({
    document: UpdateRecurringEarningDocument,
    operationName: "UpdateRecurringEarning",
    variables: { input },
  });
  return data.updateRecurringEarning;
}

export async function issuePayAdvance(input: IssuePayAdvanceInput) {
  const data = await requestGraphQL({
    document: IssuePayAdvanceDocument,
    operationName: "IssuePayAdvance",
    variables: { input },
  });
  return data.issuePayAdvance;
}

export async function writeOffPayAdvance(input: WriteOffPayAdvanceInput) {
  const data = await requestGraphQL({
    document: WriteOffPayAdvanceDocument,
    operationName: "WriteOffPayAdvance",
    variables: { input },
  });
  return data.writeOffPayAdvance;
}

export async function openEscrowAccount(input: OpenEscrowAccountInput) {
  const data = await requestGraphQL({
    document: OpenEscrowAccountDocument,
    operationName: "OpenEscrowAccount",
    variables: { input },
  });
  return data.openEscrowAccount;
}

export async function adjustEscrowAccount(input: AdjustEscrowAccountInput) {
  const data = await requestGraphQL({
    document: AdjustEscrowAccountDocument,
    operationName: "AdjustEscrowAccount",
    variables: { input },
  });
  return data.adjustEscrowAccount;
}

export async function closeEscrowAccount(accountId: string) {
  const data = await requestGraphQL({
    document: CloseEscrowAccountDocument,
    operationName: "CloseEscrowAccount",
    variables: { accountId },
  });
  return data.closeEscrowAccount;
}

export async function generateSettlementBatch(input: GenerateSettlementBatchInput) {
  const data = await requestGraphQL({
    document: GenerateSettlementBatchDocument,
    operationName: "GenerateSettlementBatch",
    variables: { input },
  });
  return data.generateSettlementBatch;
}

export async function generateDriverSettlement(input: GenerateDriverSettlementInput) {
  const data = await requestGraphQL({
    document: GenerateDriverSettlementDocument,
    operationName: "GenerateDriverSettlement",
    variables: { input },
  });
  return data.generateDriverSettlement;
}

export async function submitDriverSettlement(input: DriverSettlementActionInput) {
  const data = await requestGraphQL({
    document: SubmitDriverSettlementDocument,
    operationName: "SubmitDriverSettlement",
    variables: { input },
  });
  return data.submitDriverSettlement;
}

export async function approveDriverSettlement(input: DriverSettlementActionInput) {
  const data = await requestGraphQL({
    document: ApproveDriverSettlementDocument,
    operationName: "ApproveDriverSettlement",
    variables: { input },
  });
  return data.approveDriverSettlement;
}

export async function rejectDriverSettlement(input: DriverSettlementActionInput) {
  const data = await requestGraphQL({
    document: RejectDriverSettlementDocument,
    operationName: "RejectDriverSettlement",
    variables: { input },
  });
  return data.rejectDriverSettlement;
}

export async function postDriverSettlement(input: DriverSettlementActionInput) {
  const data = await requestGraphQL({
    document: PostDriverSettlementDocument,
    operationName: "PostDriverSettlement",
    variables: { input },
  });
  return data.postDriverSettlement;
}

export async function markDriverSettlementPaid(input: MarkDriverSettlementPaidInput) {
  const data = await requestGraphQL({
    document: MarkDriverSettlementPaidDocument,
    operationName: "MarkDriverSettlementPaid",
    variables: { input },
  });
  return data.markDriverSettlementPaid;
}

export async function voidDriverSettlement(input: DriverSettlementActionInput) {
  const data = await requestGraphQL({
    document: VoidDriverSettlementDocument,
    operationName: "VoidDriverSettlement",
    variables: { input },
  });
  return data.voidDriverSettlement;
}

export async function recalculateDriverSettlement(input: DriverSettlementActionInput) {
  const data = await requestGraphQL({
    document: RecalculateDriverSettlementDocument,
    operationName: "RecalculateDriverSettlement",
    variables: { input },
  });
  return data.recalculateDriverSettlement;
}

export async function addDriverSettlementAdjustment(input: AddSettlementAdjustmentInput) {
  const data = await requestGraphQL({
    document: AddDriverSettlementAdjustmentDocument,
    operationName: "AddDriverSettlementAdjustment",
    variables: { input },
  });
  return data.addDriverSettlementAdjustment;
}

export async function removeDriverSettlementAdjustment(input: RemoveSettlementAdjustmentInput) {
  const data = await requestGraphQL({
    document: RemoveDriverSettlementAdjustmentDocument,
    operationName: "RemoveDriverSettlementAdjustment",
    variables: { input },
  });
  return data.removeDriverSettlementAdjustment;
}

export async function fetchSettlementWorkspaceSummary(periodStart?: number, periodEnd?: number) {
  const data = await requestGraphQL({
    document: SettlementWorkspaceSummaryDocument,
    operationName: "SettlementWorkspaceSummary",
    variables: { periodStart, periodEnd },
  });
  return data.settlementWorkspaceSummary;
}

export async function fetchWorkspaceSettlements(periodStart: number, periodEnd: number) {
  const data = await requestGraphQL({
    document: DriverSettlementTableDocument,
    operationName: "DriverSettlementTable",
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
  return (data.driverSettlements.edges ?? []).map((edge) => edge.node);
}

export async function fetchWorkerUnsettledPayEvents(workerId: string) {
  const data = await requestGraphQL({
    document: DriverPayEventTableDocument,
    operationName: "DriverPayEventTable",
    variables: {
      input: {
        first: 100,
        fieldFilters: [
          { field: "workerId", operator: "eq", value: workerId },
          { field: "status", operator: "eq", value: "Accrued" },
        ],
        sort: [{ field: "eventDate", direction: "desc" }],
      },
    },
  });
  return (data.driverPayEvents.edges ?? []).map((edge) => edge.node);
}

export async function fetchWorkerRecurringDeductions(workerId: string) {
  const data = await requestGraphQL({
    document: RecurringDeductionTableDocument,
    operationName: "RecurringDeductionTable",
    variables: {
      input: {
        first: 50,
        fieldFilters: [{ field: "workerId", operator: "eq", value: workerId }],
        sort: [{ field: "createdAt", direction: "desc" }],
      },
    },
  });
  return (data.recurringDeductions.edges ?? []).map((edge) => edge.node);
}

export async function fetchUnsettledWorkerSummaries(periodStart?: number, periodEnd?: number) {
  const data = await requestGraphQL({
    document: UnsettledWorkerSummariesDocument,
    operationName: "UnsettledWorkerSummaries",
    variables: { periodStart, periodEnd },
  });
  return data.unsettledWorkerSummaries;
}

export async function fetchWorkerRecurringEarnings(workerId: string) {
  const data = await requestGraphQL({
    document: RecurringEarningTableDocument,
    operationName: "RecurringEarningTable",
    variables: {
      input: {
        first: 50,
        fieldFilters: [{ field: "workerId", operator: "eq", value: workerId }],
        sort: [{ field: "createdAt", direction: "desc" }],
      },
    },
  });
  return (data.recurringEarnings.edges ?? []).map((edge) => edge.node);
}

export async function fetchWorkerPayAdvances(workerId: string) {
  const data = await requestGraphQL({
    document: PayAdvanceTableDocument,
    operationName: "PayAdvanceTable",
    variables: {
      input: {
        first: 50,
        fieldFilters: [{ field: "workerId", operator: "eq", value: workerId }],
        sort: [{ field: "issuedDate", direction: "desc" }],
      },
    },
  });
  return (data.payAdvances.edges ?? []).map((edge) => edge.node);
}

export async function holdDriverPayEvent(input: HoldPayEventInput) {
  const data = await requestGraphQL({
    document: HoldDriverPayEventDocument,
    operationName: "HoldDriverPayEvent",
    variables: { input },
  });
  return data.holdDriverPayEvent;
}

export async function releaseDriverPayEvent(payEventId: string) {
  const data = await requestGraphQL({
    document: ReleaseDriverPayEventDocument,
    operationName: "ReleaseDriverPayEvent",
    variables: { payEventId },
  });
  return data.releaseDriverPayEvent;
}

export async function attachPayEventsToSettlement(input: AttachPayEventsInput) {
  const data = await requestGraphQL({
    document: AttachPayEventsToSettlementDocument,
    operationName: "AttachPayEventsToSettlement",
    variables: { input },
  });
  return data.attachPayEventsToSettlement;
}

export async function detachPayEventFromSettlement(input: DetachPayEventInput) {
  const data = await requestGraphQL({
    document: DetachPayEventFromSettlementDocument,
    operationName: "DetachPayEventFromSettlement",
    variables: { input },
  });
  return data.detachPayEventFromSettlement;
}

export async function bulkDriverSettlementAction(input: BulkSettlementActionInput) {
  const data = await requestGraphQL({
    document: BulkDriverSettlementActionDocument,
    operationName: "BulkDriverSettlementAction",
    variables: { input },
  });
  return data.bulkDriverSettlementAction;
}

export async function updateSettlementControl(input: UpdateSettlementControlInput) {
  const data = await requestGraphQL({
    document: UpdateSettlementControlDocument,
    operationName: "UpdateSettlementControl",
    variables: { input },
  });
  return data.updateSettlementControl;
}

export type UnsettledPayEvent = UnsettledPayEventsQuery["unsettledPayEvents"][number];

export async function fetchUnsettledPayEvents(workerId: string) {
  const data = await requestGraphQL({
    document: UnsettledPayEventsDocument,
    operationName: "UnsettledPayEvents",
    variables: { workerId },
  });
  return data.unsettledPayEvents;
}

export async function payWorkerNow(input: PayWorkerNowInput) {
  const data = await requestGraphQL({
    document: PayWorkerNowDocument,
    operationName: "PayWorkerNow",
    variables: { input },
  });
  return data.payWorkerNow;
}
