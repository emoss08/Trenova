import {
  AcknowledgeFactoringNoaDocument,
  ActivateFactoringAgreementDocument,
  CancelFactoringSubmissionDocument,
  CreateFactoringAgreementDocument,
  DraftFactoringNoaDocument,
  EndFactoringAgreementDocument,
  FactoringAgreementFieldsFragmentDoc,
  FactoringAgreementOwnerDocument,
  FactoringAgreementsDocument,
  FactoringCompanyRecordDocument,
  FactoringInvoiceStateFieldsFragmentDoc,
  FactoringInvoiceStateTableDocument,
  FactoringNoaFieldsFragmentDoc,
  FactoringNoaOwnerDocument,
  FactoringNoasDocument,
  FactoringChargebackPlanDocument,
  FactoringChargebackReversalPlanDocument,
  FactoringFundingPlanDocument,
  FactoringPlannedMovementFieldsFragmentDoc,
  FactoringReleasePlanDocument,
  FactoringReserveBalanceDocument,
  FactoringReserveEntriesDocument,
  FactoringReserveEntryFieldsFragmentDoc,
  FactoringReserveMovementPlanFieldsFragmentDoc,
  RecordFactoringChargebackDocument,
  RecordFactoringFundingDocument,
  ReleaseFactoringReserveDocument,
  ReverseFactoringChargebackDocument,
  FactoringSubmissionDetailDocument,
  FactoringSubmissionFieldsFragmentDoc,
  FactoringSubmissionItemFieldsFragmentDoc,
  FactoringSubmissionPlanDocument,
  FactoringSubmissionTableDocument,
  FactoringSummaryDocument,
  FactoringTransportDocument,
  FactoringTransportFieldsFragmentDoc,
  InvoiceFactoringDocument,
  ReleaseFactoringNoaDocument,
  RequestMissingFactoringDocumentsDocument,
  RetryFactoringSubmissionDocument,
  SaveFactoringTransportDocument,
  SendFactoringNoaDocument,
  SetInvoiceFactoringOverrideDocument,
  SubmitInvoicesToFactorDocument,
  TestFactoringTransportDocument,
  UpdateFactoringAgreementDocument,
  type AcknowledgeFactoringNoaInput,
  type CancelFactoringSubmissionInput,
  type ChangeFactoringAgreementStatusInput,
  type DraftFactoringNoaInput,
  type FactoringAgreementFieldsFragment,
  type FactoringAgreementInput,
  type FactoringInvoiceStateFieldsFragment,
  type FactoringNoaFieldsFragment,
  type FactoringNoaStatus,
  type FactoringFundingPlanQuery,
  type FactoringPlannedMovementFieldsFragment,
  type FactoringReserveBalanceQuery,
  type FactoringReserveEntryFieldsFragment,
  type FactoringReserveMovementInput,
  type FactoringReserveMovementPlanFieldsFragment,
  type RecordFactoringFundingInput,
  type FactoringSubmissionFieldsFragment,
  type FactoringSubmissionItemFieldsFragment,
  type FactoringSubmissionPlanInput,
  type FactoringSubmissionPlanQuery,
  type FactoringSubmissionStatus,
  type FactoringSummaryQuery,
  type FactoringTransportFieldsFragment,
  type FactoringVerdict,
  type InvoiceFactoringQuery,
  type ReleaseFactoringNoaInput,
  type RequestMissingFactoringDocumentsInput,
  type RequestMissingFactoringDocumentsMutation,
  type RetryFactoringSubmissionInput,
  type SaveFactoringTransportInput,
  type SendFactoringNoaInput,
  type SetInvoiceFactoringOverrideInput,
  type UpdateFactoringAgreementInput,
} from "@trenova/graphql/generated/graphql";
import { getFragmentData, type FragmentType } from "@trenova/graphql/fragment-data";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";
import type { FactoringCompany } from "@/types/factoring-company";

type RequestOptions = { signal?: AbortSignal };

export type FactoringAgreement = FactoringAgreementFieldsFragment;
export type FactoringNoa = FactoringNoaFieldsFragment;
export type FactoringInvoiceState = FactoringInvoiceStateFieldsFragment;
export type FactoringCheck = FactoringInvoiceState["checks"][number];
export type FactoringSummary = FactoringSummaryQuery["factoringSummary"];
export type FactoringDocumentRequestResult =
  RequestMissingFactoringDocumentsMutation["requestMissingFactoringDocuments"];
export type InvoiceFactoring = Omit<
  NonNullable<InvoiceFactoringQuery["invoice"]>,
  "factoringState"
> & {
  factoringState: FactoringInvoiceState | null;
};

export type FactoringSubmission = FactoringSubmissionFieldsFragment;
export type FactoringSubmissionItem = FactoringSubmissionItemFieldsFragment;
export type FactoringSubmissionDetail = FactoringSubmission & { items: FactoringSubmissionItem[] };
export type FactoringSubmissionPlan = FactoringSubmissionPlanQuery["factoringSubmissionPlan"];
export type FactoringSubmissionPlanItem = FactoringSubmissionPlan["items"][number];
export type FactoringTransport = FactoringTransportFieldsFragment;
export type FactoringTransportTestResult = {
  passed: boolean;
  message: string;
  transport: FactoringTransport;
};

export type FactoringReserveEntry = FactoringReserveEntryFieldsFragment;
export type FactoringReserveBalance = FactoringReserveBalanceQuery["factoringReserveBalance"];
export type FactoringPlannedMovement = FactoringPlannedMovementFieldsFragment;
export type FactoringPlannedLine = FactoringPlannedMovement["lines"][number];
export type FactoringFundingPlan = Omit<
  FactoringFundingPlanQuery["factoringFundingPlan"],
  "movements"
> & { movements: FactoringPlannedMovement[] };
export type FactoringReserveMovementPlan = Omit<
  FactoringReserveMovementPlanFieldsFragment,
  "movement" | "reverses"
> & {
  movement: FactoringPlannedMovement | null;
  reverses: FactoringReserveEntry | null;
};

/** The movements a person books against one funded invoice on a schedule. */
export type FactoringReserveAction = "release" | "chargeback" | "reverse";

export const FACTORING_INVOICE_STATE_KEY = "factoring-invoice-state-list";
export const FACTORING_SUBMISSION_KEY = "factoring-submission-list";
export const FACTORING_REMITTANCE_KEY = "factoring-remittance-list";

/** The most ledger rows one invoice lists; an invoice moves a handful of times at most. */
export const FACTORING_INVOICE_ENTRY_LIMIT = 50;

/** The most recent movements a factor's reserve section lists. */
export const FACTORING_FACTOR_ENTRY_LIMIT = 25;

export type FactoringReserveEntryFilter =
  | { invoiceId: string; factoringCompanyId?: never }
  | { factoringCompanyId: string; invoiceId?: never };

export type FactoringReserveEntryPage = {
  entries: FactoringReserveEntry[];
  totalCount: number;
};

/** The readiness queue, one row per evaluated invoice, keyed by the invoice it describes. */
export function createFactoringInvoiceStateTableConfig(verdicts: FactoringVerdict[]) {
  return defineDataTableGraphQLConfig({
    document: FactoringInvoiceStateTableDocument,
    operationName: "FactoringInvoiceStateTable",
    connectionKey: "factoringInvoiceStates",
    extraVariables: { verdicts },
    mapNode: (node) => ({ ...node, id: node.invoiceId }),
  });
}

export type FactoringInvoiceStateRow = DataTableConfigRow<
  ReturnType<typeof createFactoringInvoiceStateTableConfig>
>;

/** Schedules of accounts sent to factors, newest first unless the table sorts otherwise. */
export function createFactoringSubmissionTableConfig(statuses: FactoringSubmissionStatus[] | null) {
  return defineDataTableGraphQLConfig({
    document: FactoringSubmissionTableDocument,
    operationName: "FactoringSubmissionTable",
    connectionKey: "factoringSubmissions",
    extraVariables: { statuses },
  });
}

export type FactoringSubmissionRow = DataTableConfigRow<
  ReturnType<typeof createFactoringSubmissionTableConfig>
>;

export async function fetchFactoringSubmission(
  id: string,
  options?: RequestOptions,
): Promise<FactoringSubmissionDetail> {
  const data = await requestGraphQL({
    document: FactoringSubmissionDetailDocument,
    operationName: "FactoringSubmissionDetail",
    variables: { id },
    signal: options?.signal,
  });
  const { items, ...submission } = data.factoringSubmission;
  return {
    ...getFragmentData(FactoringSubmissionFieldsFragmentDoc, submission),
    items: items.map((item) => getFragmentData(FactoringSubmissionItemFieldsFragmentDoc, item)),
  };
}

export async function fetchFactoringSubmissionPlan(
  input: FactoringSubmissionPlanInput,
  options?: RequestOptions,
): Promise<FactoringSubmissionPlan> {
  const data = await requestGraphQL({
    document: FactoringSubmissionPlanDocument,
    operationName: "FactoringSubmissionPlan",
    variables: { input },
    signal: options?.signal,
  });
  return data.factoringSubmissionPlan;
}

export async function submitInvoicesToFactor(
  input: FactoringSubmissionPlanInput,
): Promise<FactoringSubmission> {
  const data = await requestGraphQL({
    document: SubmitInvoicesToFactorDocument,
    operationName: "SubmitInvoicesToFactor",
    variables: { input },
  });
  return getFragmentData(FactoringSubmissionFieldsFragmentDoc, data.submitInvoicesToFactor);
}

export async function retryFactoringSubmission(
  input: RetryFactoringSubmissionInput,
): Promise<FactoringSubmission> {
  const data = await requestGraphQL({
    document: RetryFactoringSubmissionDocument,
    operationName: "RetryFactoringSubmission",
    variables: { input },
  });
  return getFragmentData(FactoringSubmissionFieldsFragmentDoc, data.retryFactoringSubmission);
}

export async function cancelFactoringSubmission(
  input: CancelFactoringSubmissionInput,
): Promise<FactoringSubmission> {
  const data = await requestGraphQL({
    document: CancelFactoringSubmissionDocument,
    operationName: "CancelFactoringSubmission",
    variables: { input },
  });
  return getFragmentData(FactoringSubmissionFieldsFragmentDoc, data.cancelFactoringSubmission);
}

export async function fetchFactoringTransport(
  factoringCompanyId: string,
  options?: RequestOptions,
): Promise<FactoringTransport | null> {
  const data = await requestGraphQL({
    document: FactoringTransportDocument,
    operationName: "FactoringTransport",
    variables: { factoringCompanyId },
    signal: options?.signal,
  });
  return data.factoringTransport
    ? getFragmentData(FactoringTransportFieldsFragmentDoc, data.factoringTransport)
    : null;
}

export async function saveFactoringTransport(
  input: SaveFactoringTransportInput,
): Promise<FactoringTransport> {
  const data = await requestGraphQL({
    document: SaveFactoringTransportDocument,
    operationName: "SaveFactoringTransport",
    variables: { input },
  });
  return getFragmentData(FactoringTransportFieldsFragmentDoc, data.saveFactoringTransport);
}

export async function testFactoringTransport(
  factoringCompanyId: string,
): Promise<FactoringTransportTestResult> {
  const data = await requestGraphQL({
    document: TestFactoringTransportDocument,
    operationName: "TestFactoringTransport",
    variables: { factoringCompanyId },
  });
  const result = data.testFactoringTransport;
  return {
    passed: result.passed,
    message: result.message,
    transport: getFragmentData(FactoringTransportFieldsFragmentDoc, result.transport),
  };
}

export async function fetchFactoringSummary(options?: RequestOptions): Promise<FactoringSummary> {
  const data = await requestGraphQL({
    document: FactoringSummaryDocument,
    operationName: "FactoringSummary",
    signal: options?.signal,
  });
  return data.factoringSummary;
}

export async function fetchInvoiceFactoring(
  invoiceId: string,
  options?: RequestOptions,
): Promise<InvoiceFactoring | null> {
  const data = await requestGraphQL({
    document: InvoiceFactoringDocument,
    operationName: "InvoiceFactoring",
    variables: { id: invoiceId },
    signal: options?.signal,
  });
  if (!data.invoice) {
    return null;
  }
  const { factoringState, ...invoice } = data.invoice;
  return {
    ...invoice,
    factoringState: factoringState
      ? getFragmentData(FactoringInvoiceStateFieldsFragmentDoc, factoringState)
      : null,
  };
}

export async function setInvoiceFactoringOverride(
  input: SetInvoiceFactoringOverrideInput,
): Promise<FactoringInvoiceState> {
  const data = await requestGraphQL({
    document: SetInvoiceFactoringOverrideDocument,
    operationName: "SetInvoiceFactoringOverride",
    variables: { input },
  });
  return getFragmentData(FactoringInvoiceStateFieldsFragmentDoc, data.setInvoiceFactoringOverride);
}

export async function requestMissingFactoringDocuments(
  input: RequestMissingFactoringDocumentsInput,
): Promise<FactoringDocumentRequestResult> {
  const data = await requestGraphQL({
    document: RequestMissingFactoringDocumentsDocument,
    operationName: "RequestMissingFactoringDocuments",
    variables: { input },
  });
  return data.requestMissingFactoringDocuments;
}

export async function listFactoringAgreements(
  factoringCompanyId: string,
  options?: RequestOptions,
): Promise<FactoringAgreement[]> {
  const data = await requestGraphQL({
    document: FactoringAgreementsDocument,
    operationName: "FactoringAgreements",
    variables: { factoringCompanyId },
    signal: options?.signal,
  });
  return data.factoringAgreements.map((row) =>
    getFragmentData(FactoringAgreementFieldsFragmentDoc, row),
  );
}

export async function fetchFactoringAgreementOwner(
  id: string,
  options?: RequestOptions,
): Promise<string> {
  const data = await requestGraphQL({
    document: FactoringAgreementOwnerDocument,
    operationName: "FactoringAgreementOwner",
    variables: { id },
    signal: options?.signal,
  });
  return data.factoringAgreement.factoringCompanyId;
}

export async function createFactoringAgreement(
  input: FactoringAgreementInput,
): Promise<FactoringAgreement> {
  const data = await requestGraphQL({
    document: CreateFactoringAgreementDocument,
    operationName: "CreateFactoringAgreement",
    variables: { input },
  });
  return getFragmentData(FactoringAgreementFieldsFragmentDoc, data.createFactoringAgreement);
}

export async function updateFactoringAgreement(
  input: UpdateFactoringAgreementInput,
): Promise<FactoringAgreement> {
  const data = await requestGraphQL({
    document: UpdateFactoringAgreementDocument,
    operationName: "UpdateFactoringAgreement",
    variables: { input },
  });
  return getFragmentData(FactoringAgreementFieldsFragmentDoc, data.updateFactoringAgreement);
}

export async function activateFactoringAgreement(
  input: ChangeFactoringAgreementStatusInput,
): Promise<FactoringAgreement> {
  const data = await requestGraphQL({
    document: ActivateFactoringAgreementDocument,
    operationName: "ActivateFactoringAgreement",
    variables: { input },
  });
  return getFragmentData(FactoringAgreementFieldsFragmentDoc, data.activateFactoringAgreement);
}

export async function endFactoringAgreement(
  input: ChangeFactoringAgreementStatusInput,
): Promise<FactoringAgreement> {
  const data = await requestGraphQL({
    document: EndFactoringAgreementDocument,
    operationName: "EndFactoringAgreement",
    variables: { input },
  });
  return getFragmentData(FactoringAgreementFieldsFragmentDoc, data.endFactoringAgreement);
}

export type FactoringNoaFilter = {
  factoringCompanyId?: string;
  customerId?: string;
  statuses?: FactoringNoaStatus[];
};

/** The most notices one factor or customer lists at once; a factor has one per customer. */
export const FACTORING_NOA_PAGE_SIZE = 100;

export async function listFactoringNoas(
  filter: FactoringNoaFilter,
  options?: RequestOptions,
): Promise<FactoringNoa[]> {
  const data = await requestGraphQL({
    document: FactoringNoasDocument,
    operationName: "FactoringNoas",
    variables: {
      input: {
        table: {
          first: FACTORING_NOA_PAGE_SIZE,
          sort: [{ field: "createdAt", direction: "desc" }],
        },
        factoringCompanyId: filter.factoringCompanyId ?? null,
        customerId: filter.customerId ?? null,
        statuses: filter.statuses ?? null,
      },
    },
    signal: options?.signal,
  });
  return data.factoringNoas.edges.map((edge) =>
    getFragmentData(FactoringNoaFieldsFragmentDoc, edge.node),
  );
}

export async function fetchFactoringNoaOwner(
  id: string,
  options?: RequestOptions,
): Promise<string> {
  const data = await requestGraphQL({
    document: FactoringNoaOwnerDocument,
    operationName: "FactoringNoaOwner",
    variables: { id },
    signal: options?.signal,
  });
  return data.factoringNoa.factoringCompanyId;
}

export async function draftFactoringNoa(input: DraftFactoringNoaInput): Promise<FactoringNoa> {
  const data = await requestGraphQL({
    document: DraftFactoringNoaDocument,
    operationName: "DraftFactoringNoa",
    variables: { input },
  });
  return getFragmentData(FactoringNoaFieldsFragmentDoc, data.draftFactoringNoa);
}

export async function sendFactoringNoa(input: SendFactoringNoaInput): Promise<FactoringNoa> {
  const data = await requestGraphQL({
    document: SendFactoringNoaDocument,
    operationName: "SendFactoringNoa",
    variables: { input },
  });
  return getFragmentData(FactoringNoaFieldsFragmentDoc, data.sendFactoringNoa);
}

export async function acknowledgeFactoringNoa(
  input: AcknowledgeFactoringNoaInput,
): Promise<FactoringNoa> {
  const data = await requestGraphQL({
    document: AcknowledgeFactoringNoaDocument,
    operationName: "AcknowledgeFactoringNoa",
    variables: { input },
  });
  return getFragmentData(FactoringNoaFieldsFragmentDoc, data.acknowledgeFactoringNoa);
}

export async function releaseFactoringNoa(input: ReleaseFactoringNoaInput): Promise<FactoringNoa> {
  const data = await requestGraphQL({
    document: ReleaseFactoringNoaDocument,
    operationName: "ReleaseFactoringNoa",
    variables: { input },
  });
  return getFragmentData(FactoringNoaFieldsFragmentDoc, data.releaseFactoringNoa);
}

export async function fetchFactoringCompany(
  id: string,
  options?: RequestOptions,
): Promise<FactoringCompany | null> {
  const data = await requestGraphQL({
    document: FactoringCompanyRecordDocument,
    operationName: "FactoringCompanyRecord",
    variables: { id },
    signal: options?.signal,
  });
  return data.factoringCompany;
}

export async function fetchFactoringReserveBalance(
  factoringCompanyId: string,
  options?: RequestOptions,
): Promise<FactoringReserveBalance> {
  const data = await requestGraphQL({
    document: FactoringReserveBalanceDocument,
    operationName: "FactoringReserveBalance",
    variables: { factoringCompanyId },
    signal: options?.signal,
  });
  return data.factoringReserveBalance;
}

/**
 * Reserve movements for one invoice, oldest first so they read as its
 * history, or for one factor, newest first so the latest movements lead.
 */
export async function listFactoringReserveEntries(
  filter: FactoringReserveEntryFilter,
  options?: RequestOptions,
): Promise<FactoringReserveEntryPage> {
  const byInvoice = filter.invoiceId !== undefined;
  const data = await requestGraphQL({
    document: FactoringReserveEntriesDocument,
    operationName: "FactoringReserveEntries",
    variables: {
      input: {
        table: {
          first: byInvoice ? FACTORING_INVOICE_ENTRY_LIMIT : FACTORING_FACTOR_ENTRY_LIMIT,
          sort: [{ field: "createdAt", direction: byInvoice ? "asc" : "desc" }],
        },
        invoiceId: filter.invoiceId ?? null,
        factoringCompanyId: filter.factoringCompanyId ?? null,
      },
    },
    signal: options?.signal,
  });
  const entries = data.factoringReserveEntries.edges.map((edge) =>
    getFragmentData(FactoringReserveEntryFieldsFragmentDoc, edge.node),
  );
  return { entries, totalCount: data.factoringReserveEntries.totalCount ?? entries.length };
}

export async function fetchFactoringFundingPlan(
  input: RecordFactoringFundingInput,
  options?: RequestOptions,
): Promise<FactoringFundingPlan> {
  const data = await requestGraphQL({
    document: FactoringFundingPlanDocument,
    operationName: "FactoringFundingPlan",
    variables: { input },
    signal: options?.signal,
  });
  const { movements, ...plan } = data.factoringFundingPlan;
  return {
    ...plan,
    movements: movements.map((movement) =>
      getFragmentData(FactoringPlannedMovementFieldsFragmentDoc, movement),
    ),
  };
}

function readMovementPlan(
  raw: FragmentType<typeof FactoringReserveMovementPlanFieldsFragmentDoc>,
): FactoringReserveMovementPlan {
  const { movement, reverses, ...plan } = getFragmentData(
    FactoringReserveMovementPlanFieldsFragmentDoc,
    raw,
  );
  return {
    ...plan,
    movement: movement
      ? getFragmentData(FactoringPlannedMovementFieldsFragmentDoc, movement)
      : null,
    reverses: reverses ? getFragmentData(FactoringReserveEntryFieldsFragmentDoc, reverses) : null,
  };
}

export async function fetchFactoringReserveMovementPlan(
  action: FactoringReserveAction,
  input: FactoringReserveMovementInput,
  options?: RequestOptions,
): Promise<FactoringReserveMovementPlan> {
  switch (action) {
    case "release": {
      const data = await requestGraphQL({
        document: FactoringReleasePlanDocument,
        operationName: "FactoringReleasePlan",
        variables: { input },
        signal: options?.signal,
      });
      return readMovementPlan(data.factoringReleasePlan);
    }
    case "chargeback": {
      const data = await requestGraphQL({
        document: FactoringChargebackPlanDocument,
        operationName: "FactoringChargebackPlan",
        variables: { input },
        signal: options?.signal,
      });
      return readMovementPlan(data.factoringChargebackPlan);
    }
    case "reverse": {
      const data = await requestGraphQL({
        document: FactoringChargebackReversalPlanDocument,
        operationName: "FactoringChargebackReversalPlan",
        variables: { input },
        signal: options?.signal,
      });
      return readMovementPlan(data.factoringChargebackReversalPlan);
    }
  }
}

export async function recordFactoringFunding(
  input: RecordFactoringFundingInput,
): Promise<FactoringSubmission> {
  const data = await requestGraphQL({
    document: RecordFactoringFundingDocument,
    operationName: "RecordFactoringFunding",
    variables: { input },
  });
  return getFragmentData(FactoringSubmissionFieldsFragmentDoc, data.recordFactoringFunding);
}

export async function bookFactoringReserveMovement(
  action: FactoringReserveAction,
  input: FactoringReserveMovementInput,
): Promise<FactoringSubmission> {
  switch (action) {
    case "release": {
      const data = await requestGraphQL({
        document: ReleaseFactoringReserveDocument,
        operationName: "ReleaseFactoringReserve",
        variables: { input },
      });
      return getFragmentData(FactoringSubmissionFieldsFragmentDoc, data.releaseFactoringReserve);
    }
    case "chargeback": {
      const data = await requestGraphQL({
        document: RecordFactoringChargebackDocument,
        operationName: "RecordFactoringChargeback",
        variables: { input },
      });
      return getFragmentData(FactoringSubmissionFieldsFragmentDoc, data.recordFactoringChargeback);
    }
    case "reverse": {
      const data = await requestGraphQL({
        document: ReverseFactoringChargebackDocument,
        operationName: "ReverseFactoringChargeback",
        variables: { input },
      });
      return getFragmentData(FactoringSubmissionFieldsFragmentDoc, data.reverseFactoringChargeback);
    }
  }
}
