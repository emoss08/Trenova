import { z } from "zod";
import { driverTypeSchema } from "./worker";

export const credentialCategorySchema = z.enum([
  "License",
  "Medical",
  "Endorsement",
  "Security",
  "Certification",
  "Background",
  "Other",
]);
export type CredentialCategory = z.infer<typeof credentialCategorySchema>;

export const credentialStatusSchema = z.enum(["Active", "Archived"]);
export type CredentialStatus = z.infer<typeof credentialStatusSchema>;

export const credentialHealthSchema = z.enum(["Valid", "ExpiringSoon", "Expired", "Missing"]);
export type CredentialHealth = z.infer<typeof credentialHealthSchema>;

export const CREDENTIAL_CATEGORY_LABELS: Record<CredentialCategory, string> = {
  License: "License",
  Medical: "Medical",
  Endorsement: "Endorsement",
  Security: "Security",
  Certification: "Certification",
  Background: "Background check",
  Other: "Other",
};

export const CREDENTIAL_HEALTH_LABELS: Record<CredentialHealth, string> = {
  Valid: "Valid",
  ExpiringSoon: "Expiring soon",
  Expired: "Expired",
  Missing: "Missing",
};

const optionalTrimmed = (max: number, message: string) =>
  z
    .string()
    .nullable()
    .optional()
    .transform((value) => {
      const trimmed = value?.trim() ?? "";
      return trimmed === "" ? null : trimmed;
    })
    .pipe(z.string().max(max, { message }).nullable());

export const credentialTypeFormSchema = z.object({
  code: z
    .string()
    .trim()
    .min(1, { message: "Code is required" })
    .max(50, { message: "Code cannot exceed 50 characters" })
    .regex(/^[A-Za-z0-9_-]+$/, { message: "Letters, digits, dashes and underscores only" }),
  name: z
    .string()
    .trim()
    .min(1, { message: "Name is required" })
    .max(100, { message: "Name cannot exceed 100 characters" }),
  description: optionalTrimmed(1000, "Description cannot exceed 1000 characters"),
  category: credentialCategorySchema,
  status: z.enum(["Active", "Inactive"]),
  isRequired: z.boolean(),
  requiredForDriverTypes: z.array(driverTypeSchema),
  renewalWindowDays: z
    .number()
    .int()
    .min(0, { message: "Cannot be negative" })
    .max(365, { message: "Cannot exceed 365 days" }),
  validityMonths: z.number().int().min(1, { message: "Must be at least one month" }).nullable(),
  requiresNumber: z.boolean(),
  requiresDocument: z.boolean(),
});
export type CredentialTypeFormValues = z.infer<typeof credentialTypeFormSchema>;

export const credentialFormSchema = z
  .object({
    credentialTypeId: z.string().min(1, { message: "Choose a credential type" }),
    number: optionalTrimmed(100, "Number cannot exceed 100 characters"),
    issuingAuthority: optionalTrimmed(100, "Issuing authority cannot exceed 100 characters"),
    issuedAt: z.number().int().positive().nullable(),
    expiresAt: z.number().int().positive().nullable(),
    notes: optionalTrimmed(2000, "Notes cannot exceed 2000 characters"),
    requiresNumber: z.boolean(),
    requiresExpiry: z.boolean(),
  })
  .superRefine((values, ctx) => {
    if (values.requiresNumber && !values.number) {
      ctx.addIssue({
        code: "custom",
        path: ["number"],
        message: "This credential type requires a number",
      });
    }
    if (values.requiresExpiry && values.expiresAt == null) {
      ctx.addIssue({
        code: "custom",
        path: ["expiresAt"],
        message: "This credential type requires an expiry date",
      });
    }
    if (
      values.issuedAt != null &&
      values.expiresAt != null &&
      values.expiresAt <= values.issuedAt
    ) {
      ctx.addIssue({
        code: "custom",
        path: ["expiresAt"],
        message: "Expiry must be after the issue date",
      });
    }
  });
export type CredentialFormValues = z.infer<typeof credentialFormSchema>;

export const credentialArchiveSchema = z.object({
  reason: optionalTrimmed(255, "Reason cannot exceed 255 characters"),
});
export type CredentialArchiveValues = z.infer<typeof credentialArchiveSchema>;
