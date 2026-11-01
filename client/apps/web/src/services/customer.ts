import {
  BulkUpdateCustomerStatusDocument,
  CreateCustomerDocument,
  PatchCustomerDocument,
  UpdateCustomerDocument,
  type CustomerBillingProfileInput,
  type CustomerEmailProfileInput,
  type CustomerInput,
  type CustomerPatchInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type {
  BulkUpdateCustomerStatusRequest,
  Customer,
  CustomerBillingProfile,
  CustomerEmailProfile,
} from "@trenova/shared/types/customer";

function decimalInput(value: number | null | undefined): string | null {
  return value === null || value === undefined ? null : String(value);
}

function documentTypeIds(documentTypes: CustomerBillingProfile["documentTypes"]): string[] {
  return (documentTypes ?? [])
    .map((documentType: unknown) =>
      typeof documentType === "string"
        ? documentType
        : (documentType as { id?: string } | null)?.id,
    )
    .filter((id): id is string => !!id);
}

function toBillingProfileInput(profile: CustomerBillingProfile): CustomerBillingProfileInput {
  return {
    invoiceDelivery: profile.invoiceDelivery,
    billingCycle: profile.billingCycle,
    billingCycleAnchorDay: profile.billingCycleAnchorDay,
    billingCycleTimezone: profile.billingCycleTimezone,
    splitBy: profile.splitBy,
    sectionBy: profile.sectionBy,
    invoiceDetail: profile.invoiceDetail,
    minConsolidatedAmount: decimalInput(profile.minConsolidatedAmount),
    maxShipmentsPerInvoice: profile.maxShipmentsPerInvoice,
    paymentTerm: profile.paymentTerm,
    hasBillingControlOverrides: profile.hasBillingControlOverrides,
    creditLimit: decimalInput(profile.creditLimit),
    creditStatus: profile.creditStatus,
    enforceCreditLimit: profile.enforceCreditLimit,
    autoCreditHold: profile.autoCreditHold,
    creditHoldReason: profile.creditHoldReason,
    autoSendInvoiceOnGeneration: profile.autoSendInvoiceOnGeneration,
    sendPaymentReminders: profile.sendPaymentReminders,
    sendStatements: profile.sendStatements,
    emailInvoiceEnabled: profile.emailInvoiceEnabled,
    ediInvoiceEnabled: profile.ediInvoiceEnabled,
    invoiceNumberFormat: profile.invoiceNumberFormat,
    customerInvoicePrefix: profile.customerInvoicePrefix,
    invoiceCopies: profile.invoiceCopies,
    revenueAccountId: profile.revenueAccountId ?? null,
    arAccountId: profile.arAccountId ?? null,
    applyLateCharges: profile.applyLateCharges,
    lateChargeRate: decimalInput(profile.lateChargeRate),
    gracePeriodDays: profile.gracePeriodDays,
    taxExempt: profile.taxExempt,
    taxExemptNumber: profile.taxExemptNumber,
    enforceCustomerBillingReq: profile.enforceCustomerBillingReq,
    validateCustomerRates: profile.validateCustomerRates,
    autoTransfer: profile.autoTransfer,
    autoMarkReadyToBill: profile.autoMarkReadyToBill,
    autoApprove: profile.autoApprove,
    autoBill: profile.autoBill,
    countLateOnlyOnAppointmentStops: profile.countLateOnlyOnAppointmentStops,
    autoApplyAccessorials: profile.autoApplyAccessorials,
    billingCurrency: profile.billingCurrency,
    requirePONumber: profile.requirePONumber,
    requireBOLNumber: profile.requireBOLNumber,
    requireDeliveryNumber: profile.requireDeliveryNumber,
    defaultBillerId: profile.defaultBillerId ?? null,
    billingNotes: profile.billingNotes,
    fuelSurchargeMode: profile.fuelSurchargeMode,
    fuelSurchargeProgramId: profile.fuelSurchargeProgramId ?? null,
    factoringCompanyId: profile.factoringCompanyId ?? null,
    documentTypeIds: documentTypeIds(profile.documentTypes),
  };
}

function toEmailProfileInput(profile: CustomerEmailProfile): CustomerEmailProfileInput {
  return {
    subject: profile.subject,
    comment: profile.comment,
    toRecipients: profile.toRecipients,
    ccRecipients: profile.ccRecipients,
    bccRecipients: profile.bccRecipients,
    attachmentName: profile.attachmentName,
    readReceipt: profile.readReceipt,
    includeShipmentDetail: profile.includeShipmentDetail,
  };
}

export function toCustomerInput(data: Customer): CustomerInput {
  return {
    status: data.status,
    code: data.code,
    name: data.name,
    addressLine1: data.addressLine1,
    addressLine2: data.addressLine2 ?? null,
    city: data.city,
    stateId: data.stateId,
    postalCode: data.postalCode,
    longitude: data.longitude ?? null,
    latitude: data.latitude ?? null,
    placeId: data.placeId ?? null,
    externalId: data.externalId ?? null,
    dotNumber: data.dotNumber ?? null,
    mcNumber: data.mcNumber ?? null,
    brokerVettingEnabled: data.brokerVettingEnabled,
    checkCallIntervalHours: data.checkCallIntervalHours ?? null,
    etaRiskThresholdMinutes: data.etaRiskThresholdMinutes ?? null,
    allowConsolidation: data.allowConsolidation,
    exclusiveConsolidation: data.exclusiveConsolidation,
    consolidationPriority: data.consolidationPriority,
    statusUpdatePreference: data.statusUpdatePreference,
    statusUpdateRecipients: data.statusUpdateRecipients ?? null,
    billingProfile: data.billingProfile ? toBillingProfileInput(data.billingProfile) : null,
    emailProfile: data.emailProfile ? toEmailProfileInput(data.emailProfile) : null,
    version: data.version,
  };
}

export class CustomerService {
  public async create(data: Customer) {
    const response = await requestGraphQL({
      document: CreateCustomerDocument,
      operationName: "CreateCustomer",
      variables: { input: toCustomerInput(data) },
    });

    return response.createCustomer;
  }

  public async update(id: string, data: Customer) {
    const response = await requestGraphQL({
      document: UpdateCustomerDocument,
      operationName: "UpdateCustomer",
      variables: { id, input: toCustomerInput(data) },
    });

    return response.updateCustomer;
  }

  public async bulkUpdateStatus(request: BulkUpdateCustomerStatusRequest) {
    const response = await requestGraphQL({
      document: BulkUpdateCustomerStatusDocument,
      operationName: "BulkUpdateCustomerStatus",
      variables: { input: request },
    });

    return response.bulkUpdateCustomerStatus;
  }

  public async patch(id: string, input: CustomerPatchInput) {
    const response = await requestGraphQL({
      document: PatchCustomerDocument,
      operationName: "PatchCustomer",
      variables: { id, input },
    });

    return response.patchCustomer;
  }
}
