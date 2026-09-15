import {
  ApplyCreditMemoDocument,
  CreateInvoicesFromOrderDocument,
  CreateInvoicesFromShipmentsDocument,
  CreateMemoDocument,
  CreditMemoApplicationFieldsFragmentDoc,
  InvoiceArContextDocument,
  InvoiceArFieldsFragmentDoc,
  InvoiceDisputeFieldsFragmentDoc,
  InvoiceEdiSendPlanFieldsFragmentDoc,
  InvoiceRelatedFieldsFragmentDoc,
  OpenInvoiceDisputeDocument,
  ResolveInvoiceDisputeDocument,
  SendInvoiceEdiDocument,
  ShipmentInvoicesDocument,
  UnapplyCreditMemoApplicationDocument,
  VoidInvoiceDocument,
  WithdrawInvoiceDisputeDocument,
  type ApplyCreditMemoInput,
  type CreateInvoicesFromShipmentsMutation,
  type CreateMemoInput,
  type CreateMemoMutation,
  type CreditMemoApplicationFieldsFragment,
  type InvoiceArContextQuery,
  type InvoiceArFieldsFragment,
  type InvoiceDisputeFieldsFragment,
  type InvoiceEdiSendPlanFieldsFragment,
  type InvoiceRelatedFieldsFragment,
  type OpenInvoiceDisputeInput,
  type ResolveInvoiceDisputeInput,
  type SendInvoiceEdiMutation,
  type UnapplyCreditMemoApplicationInput,
  type VoidInvoiceInput,
  type VoidInvoiceMutation,
  type WithdrawInvoiceDisputeInput,
} from "@trenova/graphql/generated/graphql";
import { getFragmentData } from "@trenova/graphql/fragment-data";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

type RequestOptions = { signal?: AbortSignal };

export type CreateInvoicesResult =
  CreateInvoicesFromShipmentsMutation["createInvoicesFromShipments"];
export type CreatedInvoice = CreateInvoicesResult["primary"];
export type RelatedInvoice = InvoiceRelatedFieldsFragment;
export type InvoiceDisputeCase = InvoiceDisputeFieldsFragment;
export type InvoiceCreditApplication = CreditMemoApplicationFieldsFragment;
export type InvoicePaymentApplicationRow = InvoiceArFieldsFragment["paymentApplications"][number];
export type InvoiceLateChargeAssessment = InvoiceArFieldsFragment["lateChargeAssessments"][number];
export type InvoiceEdiPlan = InvoiceEdiSendPlanFieldsFragment;
export type VoidInvoiceResult = VoidInvoiceMutation["voidInvoice"];
export type CreatedMemo = CreateMemoMutation["createMemo"];
export type InvoiceEdiSendResult = SendInvoiceEdiMutation["sendInvoiceEdi"];

type ArContextBase = NonNullable<InvoiceArContextQuery["invoice"]>;

/**
 * The resolver-computed side of an invoice, unmasked into one flat shape: who
 * it bills on behalf of, its siblings, what is still owed, what settled it,
 * its disputes and late charges, and where its EDI 210 stands.
 */
export type InvoiceArContext = Omit<
  ArContextBase,
  "relatedInvoices" | " $fragmentRefs" | " $fragmentName"
> &
  Omit<
    InvoiceArFieldsFragment,
    | "relatedInvoices"
    | "referenceInvoice"
    | "creditApplications"
    | "disputes"
    | "openDispute"
    | "ediSendPlan"
    | " $fragmentRefs"
    | " $fragmentName"
  > & {
    relatedInvoices: RelatedInvoice[];
    referenceInvoice: RelatedInvoice | null;
    creditApplications: InvoiceCreditApplication[];
    disputes: InvoiceDisputeCase[];
    openDispute: InvoiceDisputeCase | null;
    ediSendPlan: InvoiceEdiPlan;
  };

/**
 * Bills the shipments, one invoice per payer of a split shipment. `offCycleReason`
 * is required only when a payer is on a periodic statement.
 */
export async function createInvoicesFromShipments(
  shipmentIds: string[],
  offCycleReason?: string,
): Promise<CreateInvoicesResult> {
  const data = await requestGraphQL({
    document: CreateInvoicesFromShipmentsDocument,
    operationName: "CreateInvoicesFromShipments",
    variables: { shipmentIds, offCycleReason },
  });
  return data.createInvoicesFromShipments;
}

export async function createInvoicesFromOrder(
  orderId: string,
  offCycleReason?: string,
): Promise<CreateInvoicesResult> {
  const data = await requestGraphQL({
    document: CreateInvoicesFromOrderDocument,
    operationName: "CreateInvoicesFromOrder",
    variables: { orderId, offCycleReason },
  });
  return data.createInvoicesFromOrder;
}

function unmaskArContext(invoice: ArContextBase): InvoiceArContext {
  const ar = getFragmentData(InvoiceArFieldsFragmentDoc, invoice);
  return {
    ...invoice,
    ...ar,
    relatedInvoices: invoice.relatedInvoices.map((row) =>
      getFragmentData(InvoiceRelatedFieldsFragmentDoc, row),
    ),
    referenceInvoice: ar.referenceInvoice
      ? getFragmentData(InvoiceRelatedFieldsFragmentDoc, ar.referenceInvoice)
      : null,
    creditApplications: ar.creditApplications.map((row) =>
      getFragmentData(CreditMemoApplicationFieldsFragmentDoc, row),
    ),
    disputes: ar.disputes.map((row) => getFragmentData(InvoiceDisputeFieldsFragmentDoc, row)),
    openDispute: ar.openDispute
      ? getFragmentData(InvoiceDisputeFieldsFragmentDoc, ar.openDispute)
      : null,
    ediSendPlan: getFragmentData(InvoiceEdiSendPlanFieldsFragmentDoc, ar.ediSendPlan),
  };
}

/**
 * The resolver-computed side of an invoice. Read beside the REST detail rather
 * than folded into it, so the detail's own shape stays what the panel expects.
 */
export async function fetchInvoiceArContext(
  id: string,
  options?: RequestOptions,
): Promise<InvoiceArContext | null> {
  const data = await requestGraphQL({
    document: InvoiceArContextDocument,
    operationName: "InvoiceArContext",
    variables: { id },
    signal: options?.signal,
  });
  return data.invoice ? unmaskArContext(data.invoice) : null;
}

export async function fetchInvoicesByShipment(
  shipmentId: string,
  options?: RequestOptions,
): Promise<RelatedInvoice[]> {
  const data = await requestGraphQL({
    document: ShipmentInvoicesDocument,
    operationName: "ShipmentInvoices",
    variables: {
      input: {
        first: 50,
        fieldFilters: [{ field: "shipmentId", operator: "eq", value: shipmentId }],
      },
    },
    signal: options?.signal,
  });
  return data.invoices.edges.map((edge) =>
    getFragmentData(InvoiceRelatedFieldsFragmentDoc, edge.node),
  );
}

/**
 * Voids an invoice. A draft is voided at once; a posted one goes through a
 * full-reversal adjustment, and `pendingApproval` says whether an approver
 * still has to sign it off before the invoice reads Voided.
 */
export async function voidInvoice(input: VoidInvoiceInput): Promise<VoidInvoiceResult> {
  const data = await requestGraphQL({
    document: VoidInvoiceDocument,
    operationName: "VoidInvoice",
    variables: { input },
  });
  return data.voidInvoice;
}

export async function createMemo(input: CreateMemoInput): Promise<CreatedMemo> {
  const data = await requestGraphQL({
    document: CreateMemoDocument,
    operationName: "CreateMemo",
    variables: { input },
  });
  return data.createMemo;
}

export async function sendInvoiceEdi(
  invoiceId: string,
  force = false,
): Promise<InvoiceEdiSendResult> {
  const data = await requestGraphQL({
    document: SendInvoiceEdiDocument,
    operationName: "SendInvoiceEdi",
    variables: { invoiceId, force },
  });
  return data.sendInvoiceEdi;
}

export async function applyCreditMemo(
  input: ApplyCreditMemoInput,
): Promise<InvoiceCreditApplication[]> {
  const data = await requestGraphQL({
    document: ApplyCreditMemoDocument,
    operationName: "ApplyCreditMemo",
    variables: { input },
  });
  return data.applyCreditMemo.map((row) =>
    getFragmentData(CreditMemoApplicationFieldsFragmentDoc, row),
  );
}

export async function unapplyCreditMemoApplication(
  input: UnapplyCreditMemoApplicationInput,
): Promise<InvoiceCreditApplication> {
  const data = await requestGraphQL({
    document: UnapplyCreditMemoApplicationDocument,
    operationName: "UnapplyCreditMemoApplication",
    variables: { input },
  });
  return getFragmentData(CreditMemoApplicationFieldsFragmentDoc, data.unapplyCreditMemoApplication);
}

export async function openInvoiceDispute(
  input: OpenInvoiceDisputeInput,
): Promise<InvoiceDisputeCase> {
  const data = await requestGraphQL({
    document: OpenInvoiceDisputeDocument,
    operationName: "OpenInvoiceDispute",
    variables: { input },
  });
  return getFragmentData(InvoiceDisputeFieldsFragmentDoc, data.openInvoiceDispute);
}

export async function resolveInvoiceDispute(
  input: ResolveInvoiceDisputeInput,
): Promise<InvoiceDisputeCase> {
  const data = await requestGraphQL({
    document: ResolveInvoiceDisputeDocument,
    operationName: "ResolveInvoiceDispute",
    variables: { input },
  });
  return getFragmentData(InvoiceDisputeFieldsFragmentDoc, data.resolveInvoiceDispute);
}

export async function withdrawInvoiceDispute(
  input: WithdrawInvoiceDisputeInput,
): Promise<InvoiceDisputeCase> {
  const data = await requestGraphQL({
    document: WithdrawInvoiceDisputeDocument,
    operationName: "WithdrawInvoiceDispute",
    variables: { input },
  });
  return getFragmentData(InvoiceDisputeFieldsFragmentDoc, data.withdrawInvoiceDispute);
}
