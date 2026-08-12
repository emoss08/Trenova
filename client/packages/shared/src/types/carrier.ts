import { z } from "zod";
import {
  decimalStringSchema,
  nullableIntegerSchema,
  nullableStringSchema,
  tenantInfoSchema,
} from "./helpers";

export const carrierStatusSchema = z.enum(["Active", "Inactive", "DoNotUse"]);
export type CarrierStatus = z.infer<typeof carrierStatusSchema>;

export const carrierTypeSchema = z.enum(["Common", "Contract", "Broker", "Exempt"]);
export type CarrierType = z.infer<typeof carrierTypeSchema>;

export const carrierComplianceStatusSchema = z.enum([
  "Pending",
  "Qualified",
  "Disqualified",
  "Expired",
]);
export type CarrierComplianceStatus = z.infer<typeof carrierComplianceStatusSchema>;

export const carrierSafetyRatingSchema = z.enum([
  "Satisfactory",
  "Conditional",
  "Unsatisfactory",
  "NotRated",
]);
export type CarrierSafetyRating = z.infer<typeof carrierSafetyRatingSchema>;

export const carrierTaxIdTypeSchema = z.enum(["EIN", "SSN"]);
export type CarrierTaxIdType = z.infer<typeof carrierTaxIdTypeSchema>;

export const carrierPaymentMethodSchema = z.enum(["Check", "ACHManual"]);
export type CarrierPaymentMethod = z.infer<typeof carrierPaymentMethodSchema>;

export const carrierInsurancePolicyTypeSchema = z.enum([
  "AutoLiability",
  "CargoLiability",
  "GeneralLiability",
  "WorkersComp",
  "Umbrella",
]);
export type CarrierInsurancePolicyType = z.infer<typeof carrierInsurancePolicyTypeSchema>;

export const carrierContactSchema = z
  .object({
    ...tenantInfoSchema.shape,
    carrierId: z.string().optional(),
    name: z
      .string()
      .min(1, { error: "Name is required" })
      .max(255, { error: "Name must be 255 characters or less" }),
    title: nullableStringSchema,
    email: nullableStringSchema,
    phone: nullableStringSchema,
    isPrimary: z.boolean().default(false),
    receivesRateConfirmations: z.boolean().default(false),
  })
  .superRefine((data, ctx) => {
    if (data.receivesRateConfirmations && !data.email) {
      ctx.addIssue({
        code: "custom",
        path: ["email"],
        message: "A rate confirmation recipient must have an email address",
      });
    }
  });

export type CarrierContact = z.infer<typeof carrierContactSchema>;

export const carrierInsurancePolicySchema = z
  .object({
    ...tenantInfoSchema.shape,
    carrierId: z.string().optional(),
    policyType: carrierInsurancePolicyTypeSchema,
    policyNumber: z
      .string()
      .min(1, { error: "Policy number is required" })
      .max(100, { error: "Policy number must be 100 characters or less" }),
    providerName: z
      .string()
      .min(1, { error: "Provider name is required" })
      .max(255, { error: "Provider name must be 255 characters or less" }),
    coverageAmount: decimalStringSchema,
    effectiveDate: z.number().int().positive({
      message: "Effective date is required",
    }),
    expirationDate: z.number().int().positive({
      message: "Expiration date is required",
    }),
    isVerified: z.boolean().default(false),
  })
  .superRefine((data, ctx) => {
    if (data.coverageAmount != null && data.coverageAmount < 0) {
      ctx.addIssue({
        code: "custom",
        path: ["coverageAmount"],
        message: "Coverage amount cannot be negative",
      });
    }
    if (data.expirationDate <= data.effectiveDate) {
      ctx.addIssue({
        code: "custom",
        path: ["expirationDate"],
        message: "Expiration date must be after the effective date",
      });
    }
  });

export type CarrierInsurancePolicy = z.infer<typeof carrierInsurancePolicySchema>;

export const carrierSchema = z
  .object({
    ...tenantInfoSchema.shape,
    stateId: nullableStringSchema,
    remitStateId: nullableStringSchema,
    status: carrierStatusSchema,
    code: z
      .string()
      .min(1, { error: "Code is required" })
      .max(10, { error: "Code must be 10 characters or less" }),
    name: z
      .string()
      .min(1, { error: "Name is required" })
      .max(255, { error: "Name must be 255 characters or less" }),
    dbaName: nullableStringSchema,
    carrierType: carrierTypeSchema,
    dotNumber: nullableStringSchema.refine(
      (value) => value == null || /^[0-9]{1,12}$/.test(value),
      { error: "DOT number must contain only digits (12 max)" },
    ),
    mcNumber: nullableStringSchema.refine((value) => value == null || /^[0-9]{1,12}$/.test(value), {
      error: "MC number must contain only digits (12 max)",
    }),
    scac: nullableStringSchema.refine((value) => value == null || /^[A-Z]{2,4}$/.test(value), {
      error: "SCAC must be 2-4 uppercase letters",
    }),
    complianceStatus: carrierComplianceStatusSchema,
    safetyRating: carrierSafetyRatingSchema,
    qualifiedAt: nullableIntegerSchema,
    disqualifiedReason: nullableStringSchema,
    taxId: nullableStringSchema,
    taxIdType: carrierTaxIdTypeSchema.nullish(),
    w9OnFile: z.boolean().default(false),
    is1099Eligible: z.boolean().default(false),
    paymentMethod: carrierPaymentMethodSchema,
    paymentTermDays: z
      .number()
      .int()
      .min(0, { error: "Payment term days cannot be negative" })
      .max(365, { error: "Payment term days cannot exceed 365" })
      .default(30),
    remitToName: nullableStringSchema,
    remitAddressLine1: nullableStringSchema,
    remitAddressLine2: nullableStringSchema,
    remitCity: nullableStringSchema,
    remitPostalCode: nullableStringSchema,
    addressLine1: nullableStringSchema,
    addressLine2: nullableStringSchema,
    city: nullableStringSchema,
    postalCode: nullableStringSchema,
    phone: nullableStringSchema,
    email: nullableStringSchema.refine(
      (value) => value == null || z.email().safeParse(value).success,
      { error: "Email must be a valid email address" },
    ),
    externalId: nullableStringSchema,
    notes: nullableStringSchema,
    contacts: z.array(carrierContactSchema).optional(),
    insurancePolicies: z.array(carrierInsurancePolicySchema).optional(),
  })
  .superRefine((data, ctx) => {
    if (data.complianceStatus === "Disqualified" && !data.disqualifiedReason) {
      ctx.addIssue({
        code: "custom",
        path: ["disqualifiedReason"],
        message: "A disqualified carrier must record the disqualification reason",
      });
    }
    if (data.taxId && !data.taxIdType) {
      ctx.addIssue({
        code: "custom",
        path: ["taxIdType"],
        message: "Tax ID type is required when a tax ID is provided",
      });
    }
  });

export type Carrier = z.infer<typeof carrierSchema>;

export const bulkUpdateCarrierStatusRequestSchema = z.object({
  carrierIds: z.array(z.string()),
  status: carrierStatusSchema,
});

export type BulkUpdateCarrierStatusRequest = z.infer<typeof bulkUpdateCarrierStatusRequestSchema>;

export const bulkUpdateCarrierStatusResponseSchema = z.array(carrierSchema);

export type BulkUpdateCarrierStatusResponse = z.infer<typeof bulkUpdateCarrierStatusResponseSchema>;
