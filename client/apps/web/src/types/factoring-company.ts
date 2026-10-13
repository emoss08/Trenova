import { translate } from "@trenova/shared/i18n/runtime";
import { statusSchema, tenantInfoSchema } from "@trenova/shared/types/helpers";
import { z } from "zod";

/** The longest remittance instructions or notice of assignment a factor may carry. */
export const MAX_FACTOR_TEXT = 2000;

const optionalText = (max: number, message: () => string) =>
  z.string().max(max, { error: message }).nullish();

export const factoringCompanySchema = z.object({
  ...tenantInfoSchema.shape,
  status: statusSchema.default("Active"),
  code: z
    .string()
    .trim()
    .min(1, { error: () => translate("Code is required") })
    .max(10, { error: () => translate("Code must be at most 10 characters") }),
  name: z
    .string()
    .trim()
    .min(1, { error: () => translate("Name is required") })
    .max(255, { error: () => translate("Name must be at most 255 characters") }),
  addressLine1: z
    .string()
    .trim()
    .min(1, { error: () => translate("Address line 1 is required") })
    .max(150, { error: () => translate("Address line 1 must be at most 150 characters") }),
  addressLine2: optionalText(150, () => translate("Address line 2 must be at most 150 characters")),
  city: z
    .string()
    .trim()
    .min(1, { error: () => translate("City is required") })
    .max(100, { error: () => translate("City must be at most 100 characters") }),
  stateId: z.string().min(1, { error: () => translate("State is required") }),
  postalCode: z
    .string()
    .trim()
    .min(1, { error: () => translate("Postal code is required") }),
  email: z
    .union([z.literal(""), z.email({ error: () => translate("Enter a valid email address") })])
    .nullish(),
  phoneNumber: optionalText(30, () => translate("Phone number must be at most 30 characters")),
  remittanceInstructions: optionalText(MAX_FACTOR_TEXT, () =>
    translate("Remittance instructions must be at most 2,000 characters"),
  ),
  noticeOfAssignment: optionalText(MAX_FACTOR_TEXT, () =>
    translate("The notice of assignment must be at most 2,000 characters"),
  ),
  notes: optionalText(MAX_FACTOR_TEXT, () => translate("Notes must be at most 2,000 characters")),
  nonRecourse: z.boolean().default(false),
});

export type FactoringCompany = z.infer<typeof factoringCompanySchema>;
