import type {
  AdjustWorkerPtoBalanceInput,
  AdjustWorkerPtoBalanceMutation,
  AdjustWorkerPtoBalanceMutationVariables,
  ArchivePtoPolicyMutation,
  ArchivePtoPolicyMutationVariables,
  AssignWorkerPtoPolicyInput,
  AssignWorkerPtoPolicyMutation,
  AssignWorkerPtoPolicyMutationVariables,
  CreatePtoPolicyMutation,
  CreatePtoPolicyMutationVariables,
  EndWorkerPtoPolicyAssignmentInput,
  EndWorkerPtoPolicyAssignmentMutation,
  EndWorkerPtoPolicyAssignmentMutationVariables,
  PreviewWorkerPtoAccrualQuery,
  PreviewWorkerPtoAccrualQueryVariables,
  PtoAvailabilityInput,
  PtoBalanceSummaryQuery,
  PtoBalanceSummaryQueryVariables,
  PtoLiabilityReportQuery,
  PtoLiabilityReportQueryVariables,
  PtoPolicyInput,
  PtoPolicyAssignmentFieldsFragment,
  PtoPolicyFieldsFragment,
  PlannedPtoAccrualFieldsFragment,
  WorkerPtoBalanceFieldsFragment,
  WorkerPtoLedgerEntryFieldsFragment,
  RestorePtoPolicyMutation,
  RestorePtoPolicyMutationVariables,
  RunPtoAccrualInput,
  RunPtoAccrualMutation,
  RunPtoAccrualMutationVariables,
  UpdatePtoPolicyMutation,
  UpdatePtoPolicyMutationVariables,
  WorkerPtoAvailabilityQuery,
  WorkerPtoAvailabilityQueryVariables,
  WorkerPtoBalancesQuery,
  WorkerPtoBalancesQueryVariables,
  WorkerPtoLedgerInput,
  WorkerPtoLedgerQuery,
  WorkerPtoLedgerQueryVariables,
  WorkerPtoPolicyAssignmentsQuery,
  WorkerPtoPolicyAssignmentsQueryVariables,
  WorkerPtoQuery,
  WorkerPtoQueryVariables,
  WorkerPtoTableQuery,
  WorkerPtoTableQueryVariables,
} from "@trenova/graphql/generated/graphql";
import {
  WorkerPtoDocument,
  WorkerPtoTableDocument,
} from "@trenova/graphql/generated/documents/worker/page";
import type { WorkerPTO } from "@trenova/shared/types/worker";
import {
  AdjustWorkerPtoBalanceDocument,
  ArchivePtoPolicyDocument,
  AssignWorkerPtoPolicyDocument,
  CreatePtoPolicyDocument,
  EndWorkerPtoPolicyAssignmentDocument,
  PreviewWorkerPtoAccrualDocument,
  PtoBalanceSummaryDocument,
  PtoLiabilityReportDocument,
  PtoPolicyAssignmentFieldsFragmentDoc,
  PtoPolicyTableDocument,
  RestorePtoPolicyDocument,
  RunPtoAccrualDocument,
  UpdatePtoPolicyDocument,
  WorkerPtoAvailabilityDocument,
  WorkerPtoBalancesDocument,
  WorkerPtoLedgerDocument,
  WorkerPtoLedgerEntryFieldsFragmentDoc,
  WorkerPtoPolicyAssignmentsDocument,
} from "@trenova/graphql/generated/documents/pto-policy/operations";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import { unmaskConnection, type Connection } from "./connection";

export type PTOPolicyRow = PtoPolicyFieldsFragment;
export type PTOPolicyAssignment = PtoPolicyAssignmentFieldsFragment;
export type WorkerPTOBalanceView = WorkerPtoBalanceFieldsFragment;
export type WorkerPTOLedgerEntry = WorkerPtoLedgerEntryFieldsFragment;
export type PTOAvailability = WorkerPtoAvailabilityQuery["workerPtoAvailability"];
export type PTOBalanceSummary = PtoBalanceSummaryQuery["ptoBalanceSummary"];
export type PTOLiabilityReport = PtoLiabilityReportQuery["ptoLiabilityReport"];
export type PTOLiabilityRow = PTOLiabilityReport["rows"][number];
export type PlannedPTOAccrual = PlannedPtoAccrualFieldsFragment;
export type PTOAccrualRun = RunPtoAccrualMutation["runPtoAccrual"];

export const PTO_POLICY_LIST_KEY = "pto-policy-list";
export const WORKER_PTO_BALANCES_KEY = "worker-pto-balances";
export const WORKER_PTO_LEDGER_KEY = "worker-pto-ledger";
export const WORKER_PTO_ASSIGNMENTS_KEY = "worker-pto-assignments";
export const WORKER_PTO_REQUESTS_KEY = "worker-pto-requests";
export const PTO_BALANCE_SUMMARY_KEY = "pto-balance-summary";
export const PTO_LIABILITY_REPORT_KEY = "pto-liability-report";

export const ptoPolicyTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: PtoPolicyTableDocument,
  operationName: "PtoPolicyTable",
  connectionKey: "ptoPolicies",
});

export async function createPtoPolicy(input: PtoPolicyInput): Promise<PTOPolicyRow> {
  const data = await requestGraphQL<CreatePtoPolicyMutation, CreatePtoPolicyMutationVariables>({
    document: CreatePtoPolicyDocument,
    operationName: "CreatePtoPolicy",
    variables: { input },
  });
  return data.createPtoPolicy as PTOPolicyRow;
}

export async function updatePtoPolicy(id: string, input: PtoPolicyInput): Promise<PTOPolicyRow> {
  const data = await requestGraphQL<UpdatePtoPolicyMutation, UpdatePtoPolicyMutationVariables>({
    document: UpdatePtoPolicyDocument,
    operationName: "UpdatePtoPolicy",
    variables: { id, input },
  });
  return data.updatePtoPolicy as PTOPolicyRow;
}

export async function archivePtoPolicy(id: string, version?: number): Promise<PTOPolicyRow> {
  const data = await requestGraphQL<ArchivePtoPolicyMutation, ArchivePtoPolicyMutationVariables>({
    document: ArchivePtoPolicyDocument,
    operationName: "ArchivePtoPolicy",
    variables: { id, version },
  });
  return data.archivePtoPolicy as PTOPolicyRow;
}

export async function restorePtoPolicy(id: string, version?: number): Promise<PTOPolicyRow> {
  const data = await requestGraphQL<RestorePtoPolicyMutation, RestorePtoPolicyMutationVariables>({
    document: RestorePtoPolicyDocument,
    operationName: "RestorePtoPolicy",
    variables: { id, version },
  });
  return data.restorePtoPolicy as PTOPolicyRow;
}

/** A page of the policies a worker has been on, newest first; the one in force has no end. */
export async function fetchWorkerPtoPolicyAssignments(
  args: { workerId: string; first?: number; after?: string | null },
  options?: { signal?: AbortSignal },
): Promise<Connection<PTOPolicyAssignment>> {
  const data = await requestGraphQL<
    WorkerPtoPolicyAssignmentsQuery,
    WorkerPtoPolicyAssignmentsQueryVariables
  >({
    document: WorkerPtoPolicyAssignmentsDocument,
    operationName: "WorkerPtoPolicyAssignments",
    variables: { workerId: args.workerId, first: args.first ?? null, after: args.after ?? null },
    signal: options?.signal,
  });
  return unmaskConnection(PtoPolicyAssignmentFieldsFragmentDoc, data.workerPtoPolicyAssignments);
}

/**
 * A page of a worker's time off requests, latest start first. The first page carries the total,
 * which the requests panel heads itself with.
 */
export async function fetchWorkerPtoRequests(
  args: { workerId: string; first?: number; after?: string | null },
  options?: { signal?: AbortSignal },
): Promise<Connection<WorkerPTO>> {
  const data = await requestGraphQL<WorkerPtoTableQuery, WorkerPtoTableQueryVariables>({
    document: WorkerPtoTableDocument,
    operationName: "WorkerPtoTable",
    variables: {
      input: {
        workerId: args.workerId,
        first: args.first,
        after: args.after ?? undefined,
        includeWorker: false,
        sort: [{ field: "startDate", direction: "desc" }],
      },
      includeTotalCount: !args.after,
    },
    signal: options?.signal,
  });
  return data.workerPTOEntries as unknown as Connection<WorkerPTO>;
}

/** One time off request by id: a deep link to a request outside the pages already read. */
export async function fetchWorkerPto(
  id: string,
  options?: { signal?: AbortSignal },
): Promise<WorkerPTO> {
  const data = await requestGraphQL<WorkerPtoQuery, WorkerPtoQueryVariables>({
    document: WorkerPtoDocument,
    operationName: "WorkerPto",
    variables: { id },
    signal: options?.signal,
  });
  return data.workerPTO as unknown as WorkerPTO;
}

export async function fetchWorkerPtoBalances(
  workerId: string,
  options?: { signal?: AbortSignal },
): Promise<WorkerPTOBalanceView[]> {
  const data = await requestGraphQL<WorkerPtoBalancesQuery, WorkerPtoBalancesQueryVariables>({
    document: WorkerPtoBalancesDocument,
    operationName: "WorkerPtoBalances",
    variables: { workerId },
    signal: options?.signal,
  });
  return data.workerPtoBalances as WorkerPTOBalanceView[];
}

/** A page of the worker's ledger as the connection the server returns, newest first as asked. */
export async function fetchWorkerPtoLedger(
  input: WorkerPtoLedgerInput,
  options?: { signal?: AbortSignal },
): Promise<Connection<WorkerPTOLedgerEntry>> {
  const data = await requestGraphQL<WorkerPtoLedgerQuery, WorkerPtoLedgerQueryVariables>({
    document: WorkerPtoLedgerDocument,
    operationName: "WorkerPtoLedger",
    variables: { input },
    signal: options?.signal,
  });
  return unmaskConnection(WorkerPtoLedgerEntryFieldsFragmentDoc, data.workerPtoLedger);
}

export async function fetchWorkerPtoAvailability(
  input: PtoAvailabilityInput,
  options?: { signal?: AbortSignal },
): Promise<PTOAvailability> {
  const data = await requestGraphQL<
    WorkerPtoAvailabilityQuery,
    WorkerPtoAvailabilityQueryVariables
  >({
    document: WorkerPtoAvailabilityDocument,
    operationName: "WorkerPtoAvailability",
    variables: { input },
    signal: options?.signal,
  });
  return data.workerPtoAvailability;
}

export async function fetchPreviewWorkerPtoAccrual(
  workerId: string,
  asOf?: number,
  options?: { signal?: AbortSignal },
): Promise<PlannedPTOAccrual[]> {
  const data = await requestGraphQL<
    PreviewWorkerPtoAccrualQuery,
    PreviewWorkerPtoAccrualQueryVariables
  >({
    document: PreviewWorkerPtoAccrualDocument,
    operationName: "PreviewWorkerPtoAccrual",
    variables: { workerId, asOf },
    signal: options?.signal,
  });
  return data.previewWorkerPtoAccrual as PlannedPTOAccrual[];
}

export async function fetchPtoBalanceSummary(options?: {
  signal?: AbortSignal;
}): Promise<PTOBalanceSummary> {
  const data = await requestGraphQL<PtoBalanceSummaryQuery, PtoBalanceSummaryQueryVariables>({
    document: PtoBalanceSummaryDocument,
    operationName: "PtoBalanceSummary",
    signal: options?.signal,
  });
  return data.ptoBalanceSummary;
}

export async function fetchPtoLiabilityReport(
  asOf?: number,
  options?: { signal?: AbortSignal },
): Promise<PTOLiabilityReport> {
  const data = await requestGraphQL<PtoLiabilityReportQuery, PtoLiabilityReportQueryVariables>({
    document: PtoLiabilityReportDocument,
    operationName: "PtoLiabilityReport",
    variables: { asOf },
    signal: options?.signal,
  });
  return data.ptoLiabilityReport;
}

export type AssignWorkerPtoPolicyResult = {
  assignments: PTOPolicyAssignment[];
  failures: AssignWorkerPtoPolicyMutation["assignWorkerPtoPolicy"]["failures"];
};

export async function assignWorkerPtoPolicy(
  input: AssignWorkerPtoPolicyInput,
): Promise<AssignWorkerPtoPolicyResult> {
  const data = await requestGraphQL<
    AssignWorkerPtoPolicyMutation,
    AssignWorkerPtoPolicyMutationVariables
  >({
    document: AssignWorkerPtoPolicyDocument,
    operationName: "AssignWorkerPtoPolicy",
    variables: { input },
  });
  return {
    assignments: data.assignWorkerPtoPolicy.assignments as PTOPolicyAssignment[],
    failures: data.assignWorkerPtoPolicy.failures,
  };
}

export async function endWorkerPtoPolicyAssignment(
  input: EndWorkerPtoPolicyAssignmentInput,
): Promise<PTOPolicyAssignment> {
  const data = await requestGraphQL<
    EndWorkerPtoPolicyAssignmentMutation,
    EndWorkerPtoPolicyAssignmentMutationVariables
  >({
    document: EndWorkerPtoPolicyAssignmentDocument,
    operationName: "EndWorkerPtoPolicyAssignment",
    variables: { input },
  });
  return data.endWorkerPtoPolicyAssignment as PTOPolicyAssignment;
}

export async function adjustWorkerPtoBalance(
  input: AdjustWorkerPtoBalanceInput,
): Promise<WorkerPTOLedgerEntry> {
  const data = await requestGraphQL<
    AdjustWorkerPtoBalanceMutation,
    AdjustWorkerPtoBalanceMutationVariables
  >({
    document: AdjustWorkerPtoBalanceDocument,
    operationName: "AdjustWorkerPtoBalance",
    variables: { input },
  });
  return data.adjustWorkerPtoBalance as WorkerPTOLedgerEntry;
}

export async function runPtoAccrual(input: RunPtoAccrualInput): Promise<PTOAccrualRun> {
  const data = await requestGraphQL<RunPtoAccrualMutation, RunPtoAccrualMutationVariables>({
    document: RunPtoAccrualDocument,
    operationName: "RunPtoAccrual",
    variables: { input },
  });
  return data.runPtoAccrual;
}
