import {
  BillingControlDocument,
  UpdateBillingControlDocument,
  type UpdateBillingControlInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { safeParse } from "@trenova/shared/lib/parse";
import { billingControlSchema, type BillingControl } from "@/types/billing-control";

export function toBillingControlInput(data: BillingControl): UpdateBillingControlInput {
  return {
    version: data.version ?? 0,
    defaultPaymentTerm: data.defaultPaymentTerm,
    defaultInvoiceTerms: data.defaultInvoiceTerms,
    defaultInvoiceFooter: data.defaultInvoiceFooter,
    showDueDateOnInvoice: data.showDueDateOnInvoice,
    showBalanceDueOnInvoice: data.showBalanceDueOnInvoice,
    readyToBillAssignmentMode: data.readyToBillAssignmentMode,
    billingQueueTransferMode: data.billingQueueTransferMode,
    billingQueueTransferSchedule: data.billingQueueTransferSchedule ?? "Continuous",
    billingQueueTransferBatchSize: data.billingQueueTransferBatchSize ?? 0,
    invoiceDraftCreationMode: data.invoiceDraftCreationMode,
    invoicePostingMode: data.invoicePostingMode,
    autoInvoiceBatchSize: data.autoInvoiceBatchSize ?? 0,
    notifyOnAutoInvoiceCreation: data.notifyOnAutoInvoiceCreation,
    shipmentBillingRequirementEnforcement: data.shipmentBillingRequirementEnforcement,
    rateValidationEnforcement: data.rateValidationEnforcement,
    billingExceptionDisposition: data.billingExceptionDisposition,
    notifyOnBillingExceptions: data.notifyOnBillingExceptions,
    rateVarianceTolerancePercent: String(data.rateVarianceTolerancePercent ?? 0),
    rateVarianceAutoResolutionMode: data.rateVarianceAutoResolutionMode,
    unratedShipmentDisposition: data.unratedShipmentDisposition,
    fallbackFormulaTemplateId: data.fallbackFormulaTemplateId || null,
    requireRateOverrideReason: data.requireRateOverrideReason,
    enforceMarginFloor: data.enforceMarginFloor,
    queueAutoApproveMode: data.queueAutoApproveMode,
    queueAutoApproveMaxAmount:
      data.queueAutoApproveMaxAmount == null ? null : String(data.queueAutoApproveMaxAmount),
    queueAgentReviewEnabled: data.queueAgentReviewEnabled,
    lateChargeAssessmentMode: data.lateChargeAssessmentMode,
    lateChargeMinimumAmount: String(data.lateChargeMinimumAmount ?? 0),
    paymentRemindersEnabled: data.paymentRemindersEnabled,
    paymentReminderDays: data.paymentReminderDays ?? [],
    monthlyStatementsEnabled: data.monthlyStatementsEnabled,
    monthlyStatementDay: data.monthlyStatementDay,
  };
}

export class BillingControlService {
  public async get(options?: { signal?: AbortSignal }) {
    const response = await requestGraphQL({
      document: BillingControlDocument,
      operationName: "BillingControl",
      signal: options?.signal,
    });

    return safeParse(billingControlSchema, response.billingControl, "Billing Control");
  }

  public async update(data: BillingControl) {
    const response = await requestGraphQL({
      document: UpdateBillingControlDocument,
      operationName: "UpdateBillingControl",
      variables: { input: toBillingControlInput(data) },
    });

    return safeParse(billingControlSchema, response.updateBillingControl, "Billing Control");
  }
}
