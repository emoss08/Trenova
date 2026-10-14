import {
  AcknowledgeFactoringNoaDocument,
  ActivateFactoringAgreementDocument,
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
  FactoringSummaryDocument,
  InvoiceFactoringDocument,
  ReleaseFactoringNoaDocument,
  RequestMissingFactoringDocumentsDocument,
  SendFactoringNoaDocument,
  SetInvoiceFactoringOverrideDocument,
  UpdateFactoringAgreementDocument,
  type AcknowledgeFactoringNoaInput,
  type ChangeFactoringAgreementStatusInput,
  type DraftFactoringNoaInput,
  type FactoringAgreementFieldsFragment,
  type FactoringAgreementInput,
  type FactoringInvoiceStateFieldsFragment,
  type FactoringNoaFieldsFragment,
  type FactoringNoaStatus,
  type FactoringSummaryQuery,
  type FactoringVerdict,
  type InvoiceFactoringQuery,
  type ReleaseFactoringNoaInput,
  type RequestMissingFactoringDocumentsInput,
  type RequestMissingFactoringDocumentsMutation,
  type SendFactoringNoaInput,
  type SetInvoiceFactoringOverrideInput,
  type UpdateFactoringAgreementInput,
} from "@trenova/graphql/generated/graphql";
import { getFragmentData } from "@trenova/graphql/fragment-data";
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

export const FACTORING_INVOICE_STATE_KEY = "factoring-invoice-state-list";

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
