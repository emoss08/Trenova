import {
  AutoBillCriteria,
  BillingCycleType,
  PaymentTerm,
  TransferCriteria,
} from "@/types/billing";
import { Status } from "@/types/common";
import { array, boolean, type InferType, mixed, object, string } from "yup";

export const emailProfileSchema = object({
  id: string().optional(),
  organizationId: string().nullable().optional(),
  businessUnitId: string().nullable().optional(),
  customerId: string().nullable().optional(),

  // Core Fields
  subject: string().optional(),
  comment: string().optional(),
  fromEmail: string().optional(),
  blindCopy: string().optional(),
  readReceipt: boolean(),
  attachmentName: string().optional(),
});

export const billingProfileSchema = object({
  id: string().optional(),
  organizationId: string().nullable().optional(),
  businessUnitId: string().nullable().optional(),
  customerId: string().nullable().optional(),

  // Core Fields
  billingCycleType: mixed<BillingCycleType>().oneOf(
    Object.values(BillingCycleType),
  ),
  documentTypeIds: array()
    .of(string())
    .required("Document Type IDs are required"),

  // Billing Control Overrides
  hasOverrides: boolean(),
  enforceCustomerBillingReq: boolean(),
  validateCustomerRates: boolean(),
  paymentTerm: mixed<PaymentTerm>().oneOf(Object.values(PaymentTerm)),
  autoTransfer: boolean(),
  transferCriteria: mixed<TransferCriteria>().oneOf(
    Object.values(TransferCriteria),
  ),
  autoMarkReadyToBill: boolean(),
  autoBill: boolean(),
  autoBillCriteria: mixed<AutoBillCriteria>().oneOf(
    Object.values(AutoBillCriteria),
  ),
  specialInstructions: string().optional(),
});

export const customerSchema = object({
  id: string().optional(),
  organizationId: string().nullable().optional(),
  businessUnitId: string().nullable().optional(),
  status: mixed<Status>()
    .required("Status is required")
    .oneOf(Object.values(Status)),
  name: string().required("Name is required"),
  code: string().required("Code is required"),
  description: string().optional(),
  addressLine1: string().required("Address line 1 is required"),
  addressLine2: string().optional(),
  city: string().required("City is required"),
  postalCode: string().required("Postal code is required"),
  stateId: string().required("State is required"),
  billingProfile: billingProfileSchema.optional(),
  emailProfile: emailProfileSchema.optional(),
});

export type CustomerSchema = InferType<typeof customerSchema>;
export type CustomerBillingProfileSchema = InferType<
  typeof billingProfileSchema
>;
