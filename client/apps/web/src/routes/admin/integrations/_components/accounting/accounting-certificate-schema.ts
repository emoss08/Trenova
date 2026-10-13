import { normalizeNetSuiteAccountId } from "@/lib/accounting-sync";
import { translate } from "@trenova/shared/i18n/runtime";
import { z } from "zod";

const CERTIFICATE_ID_PATTERN = /^[A-Za-z0-9._~+/=-]+$/;
const MAX_CERTIFICATE_ID_LENGTH = 100;

const certificateIdSchema = z
  .string()
  .trim()
  .min(1, { error: () => translate("Certificate ID is required") })
  .max(MAX_CERTIFICATE_ID_LENGTH, {
    error: () => translate("Certificate ID cannot be longer than 100 characters"),
  })
  .regex(CERTIFICATE_ID_PATTERN, {
    error: () => translate("Copy the certificate ID exactly as NetSuite shows it, without spaces"),
  });

export const accountingCertificateConnectSchema = z.object({
  accountId: z
    .string()
    .trim()
    .min(1, { error: () => translate("Account ID is required") })
    .refine((value) => normalizeNetSuiteAccountId(value) !== null, {
      error: () => translate("Account ID looks like 1234567, or 1234567_SB1 for a sandbox"),
    }),
  certificateId: certificateIdSchema,
});

export type AccountingCertificateConnectValues = z.infer<typeof accountingCertificateConnectSchema>;

export const accountingCertificateRenewSchema = z.object({
  certificateId: certificateIdSchema,
});

export type AccountingCertificateRenewValues = z.infer<typeof accountingCertificateRenewSchema>;
