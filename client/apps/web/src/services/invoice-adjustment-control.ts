import {
  InvoiceAdjustmentControlDocument,
  UpdateInvoiceAdjustmentControlDocument,
  type UpdateInvoiceAdjustmentControlInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  invoiceAdjustmentControlSchema,
  type InvoiceAdjustmentControl,
} from "@/types/invoice-adjustment-control";

export function toInvoiceAdjustmentControlInput(
  data: InvoiceAdjustmentControl,
): UpdateInvoiceAdjustmentControlInput {
  return {
    version: data.version ?? 0,
    partiallyPaidInvoiceAdjustmentPolicy: data.partiallyPaidInvoiceAdjustmentPolicy,
    paidInvoiceAdjustmentPolicy: data.paidInvoiceAdjustmentPolicy,
    disputedInvoiceAdjustmentPolicy: data.disputedInvoiceAdjustmentPolicy,
    adjustmentAccountingDatePolicy: data.adjustmentAccountingDatePolicy,
    closedPeriodAdjustmentPolicy: data.closedPeriodAdjustmentPolicy,
    adjustmentReasonRequirement: data.adjustmentReasonRequirement,
    standardAdjustmentApprovalPolicy: data.standardAdjustmentApprovalPolicy,
    standardAdjustmentApprovalThreshold: String(data.standardAdjustmentApprovalThreshold ?? 0),
    writeOffApprovalPolicy: data.writeOffApprovalPolicy,
    writeOffApprovalThreshold: String(data.writeOffApprovalThreshold ?? 0),
    requireSeparateApprover: data.requireSeparateApprover,
    rerateVarianceTolerancePercent: String(data.rerateVarianceTolerancePercent ?? 0),
    replacementInvoiceReviewPolicy: data.replacementInvoiceReviewPolicy,
    customerCreditBalancePolicy: data.customerCreditBalancePolicy,
    overCreditPolicy: data.overCreditPolicy,
    supersededInvoiceVisibilityPolicy: data.supersededInvoiceVisibilityPolicy,
  };
}

export class InvoiceAdjustmentControlService {
  public async get(options?: { signal?: AbortSignal }) {
    const response = await requestGraphQL({
      document: InvoiceAdjustmentControlDocument,
      operationName: "InvoiceAdjustmentControl",
      signal: options?.signal,
    });

    return safeParse(
      invoiceAdjustmentControlSchema,
      response.invoiceAdjustmentControl,
      "Invoice Adjustment Control",
    );
  }

  public async update(data: InvoiceAdjustmentControl) {
    const response = await requestGraphQL({
      document: UpdateInvoiceAdjustmentControlDocument,
      operationName: "UpdateInvoiceAdjustmentControl",
      variables: { input: toInvoiceAdjustmentControlInput(data) },
    });

    return safeParse(
      invoiceAdjustmentControlSchema,
      response.updateInvoiceAdjustmentControl,
      "Invoice Adjustment Control",
    );
  }
}
