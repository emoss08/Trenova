import {
  ApproveInvoiceAdjustmentDocument,
  InvoiceAdjustmentApprovalDetailDocument,
  InvoiceAdjustmentApprovalsDocument,
  InvoiceAdjustmentOperationsSummaryDocument,
  InvoiceApprovalQueueItemFieldsFragmentDoc,
  RejectInvoiceAdjustmentDocument,
  type ApproveInvoiceAdjustmentMutation,
  type InvoiceAdjustmentApprovalDetailQuery,
  type InvoiceAdjustmentApprovalsQuery,
  type InvoiceAdjustmentKind,
  type InvoiceAdjustmentOperationsSummaryQuery,
  type InvoiceApprovalQueueItemFieldsFragment,
  type RejectInvoiceAdjustmentMutation,
} from "@trenova/graphql/generated/graphql";
import { getFragmentData } from "@trenova/graphql/fragment-data";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

type RequestOptions = { signal?: AbortSignal };

export type InvoiceApprovalQueueItem = InvoiceApprovalQueueItemFieldsFragment;
export type InvoiceApprovalDetail = NonNullable<
  InvoiceAdjustmentApprovalDetailQuery["invoiceAdjustment"]
>;
export type InvoiceAdjustmentOperationsSummary =
  InvoiceAdjustmentOperationsSummaryQuery["invoiceAdjustmentOperationsSummary"];
export type ApprovedInvoiceAdjustment =
  ApproveInvoiceAdjustmentMutation["approveInvoiceAdjustment"];
export type RejectedInvoiceAdjustment = RejectInvoiceAdjustmentMutation["rejectInvoiceAdjustment"];

export type InvoiceApprovalQueuePage = {
  items: InvoiceApprovalQueueItem[];
  pageInfo: InvoiceAdjustmentApprovalsQuery["invoiceAdjustmentApprovals"]["pageInfo"];
};

export type InvoiceApprovalQueueRequest = {
  first: number;
  after?: string | null;
  query: string;
  kind: InvoiceAdjustmentKind | null;
};

/**
 * One page of adjustments held for finance approval, newest submission first.
 * Paging is by cursor: pass the previous page's `endCursor` as `after`.
 */
export async function listInvoiceAdjustmentApprovals(
  req: InvoiceApprovalQueueRequest,
  options?: RequestOptions,
): Promise<InvoiceApprovalQueuePage> {
  const query = req.query.trim();
  const data = await requestGraphQL({
    document: InvoiceAdjustmentApprovalsDocument,
    operationName: "InvoiceAdjustmentApprovals",
    variables: {
      input: {
        first: req.first,
        after: req.after ?? undefined,
        query: query === "" ? undefined : query,
        kind: req.kind ?? undefined,
      },
    },
    signal: options?.signal,
  });
  const connection = data.invoiceAdjustmentApprovals;
  return {
    items: connection.edges.map((edge) =>
      getFragmentData(InvoiceApprovalQueueItemFieldsFragmentDoc, edge.node),
    ),
    pageInfo: connection.pageInfo,
  };
}

export async function fetchInvoiceApprovalDetail(
  id: string,
  options?: RequestOptions,
): Promise<InvoiceApprovalDetail | null> {
  const data = await requestGraphQL({
    document: InvoiceAdjustmentApprovalDetailDocument,
    operationName: "InvoiceAdjustmentApprovalDetail",
    variables: { id },
    signal: options?.signal,
  });
  return data.invoiceAdjustment;
}

export async function fetchInvoiceAdjustmentOperationsSummary(
  options?: RequestOptions,
): Promise<InvoiceAdjustmentOperationsSummary> {
  const data = await requestGraphQL({
    document: InvoiceAdjustmentOperationsSummaryDocument,
    operationName: "InvoiceAdjustmentOperationsSummary",
    signal: options?.signal,
  });
  return data.invoiceAdjustmentOperationsSummary;
}

/** Approves a pending adjustment, which executes it on the server. */
export async function approveInvoiceAdjustment(
  adjustmentId: string,
): Promise<ApprovedInvoiceAdjustment> {
  const data = await requestGraphQL({
    document: ApproveInvoiceAdjustmentDocument,
    operationName: "ApproveInvoiceAdjustment",
    variables: { adjustmentId },
  });
  return data.approveInvoiceAdjustment;
}

export async function rejectInvoiceAdjustment(input: {
  adjustmentId: string;
  reason: string;
}): Promise<RejectedInvoiceAdjustment> {
  const data = await requestGraphQL({
    document: RejectInvoiceAdjustmentDocument,
    operationName: "RejectInvoiceAdjustment",
    variables: { input },
  });
  return data.rejectInvoiceAdjustment;
}
