import type { DataRetention } from "@/types/data-retention";
import { z } from "zod";
import { translate } from "@trenova/shared/i18n/runtime";

export const AI_FEEDBACK_RETENTION_MIN_DAYS = 30;
export const AI_FEEDBACK_RETENTION_DEFAULT_DAYS = 730;

/** Seven years, which is what most audits ask a record of AI decisions to be kept for. */
export const AI_AUDIT_RETENTION_DEFAULT_DAYS = 2555;
/** The server refuses less than a year. */
export const AI_AUDIT_RETENTION_MIN_DAYS = 365;

export const AI_CORRECTION_RETENTION_MIN_DAYS = 30;
export const AI_CORRECTION_RETENTION_DEFAULT_DAYS = 730;

/** The fewest days the server keeps each class of audit entry; it refuses less. */
export const AUDIT_ACCESS_RETENTION_MIN_DAYS = 90;
export const AUDIT_OPERATIONAL_RETENTION_MIN_DAYS = 730;
export const AUDIT_FINANCIAL_RETENTION_MIN_DAYS = 2555;
export const AUDIT_SECURITY_RETENTION_MIN_DAYS = 730;
/** The most days any audit entry is kept, after which its month's partition is dropped. */
export const AUDIT_RETENTION_MAX_DAYS = 3650;

/** What a window that kept everything starts at when someone gives it a limit. */
export const EDI_RETENTION_LIMIT_DAYS = 90;
export const AGENT_EVAL_CASE_RETENTION_LIMIT_DAYS = 365;

const DAY_SECONDS = 86_400;

/**
 * The moment a window reaches back to: what is older than it goes in the next
 * nightly purge. A window of 0 keeps everything, so there is none.
 */
export function retentionCutoff(days: number, now: number): number | null {
  if (!Number.isFinite(days) || days <= 0) {
    return null;
  }

  return now - days * DAY_SECONDS;
}

export const dataRetentionFormSchema = z.object({
  auditAccessRetentionDays: z
    .number()
    .int()
    .min(AUDIT_ACCESS_RETENTION_MIN_DAYS, {
      error: () => translate("Reads and downloads are kept for at least 90 days"),
    })
    .max(AUDIT_RETENTION_MAX_DAYS, {
      error: () => translate("Audit entries are kept for at most 3,650 days"),
    }),
  auditOperationalRetentionDays: z
    .number()
    .int()
    .min(AUDIT_OPERATIONAL_RETENTION_MIN_DAYS, {
      error: () => translate("Operational changes are kept for at least 730 days"),
    })
    .max(AUDIT_RETENTION_MAX_DAYS, {
      error: () => translate("Audit entries are kept for at most 3,650 days"),
    }),
  auditFinancialRetentionDays: z
    .number()
    .int()
    .min(AUDIT_FINANCIAL_RETENTION_MIN_DAYS, {
      error: () => translate("Financial changes are kept for at least 2,555 days"),
    })
    .max(AUDIT_RETENTION_MAX_DAYS, {
      error: () => translate("Audit entries are kept for at most 3,650 days"),
    }),
  auditSecurityRetentionDays: z
    .number()
    .int()
    .min(AUDIT_SECURITY_RETENTION_MIN_DAYS, {
      error: () => translate("Security entries are kept for at least 730 days"),
    })
    .max(AUDIT_RETENTION_MAX_DAYS, {
      error: () => translate("Audit entries are kept for at most 3,650 days"),
    }),
  ediInboundFileRetentionPeriod: z
    .number()
    .int()
    .min(0, { error: () => translate("EDI inbound file retention cannot be negative") }),
  ediMessageRetentionPeriod: z
    .number()
    .int()
    .min(0, { error: () => translate("EDI message retention cannot be negative") }),
  aiFeedbackRetentionPeriod: z
    .number()
    .int()
    .min(AI_FEEDBACK_RETENTION_MIN_DAYS, {
      error: () => translate("AI feedback retention must be at least 30 days"),
    }),
  agentEvalCaseRetentionPeriod: z
    .number()
    .int()
    .min(0, { error: () => translate("Agent evaluation case retention cannot be negative") }),
  aiAuditRetentionPeriod: z
    .number()
    .int({ error: () => translate("AI audit trail retention must be a whole number of days") })
    .min(AI_AUDIT_RETENTION_MIN_DAYS, {
      error: () => translate("AI audit trail retention must be at least 365 days"),
    }),
  aiCorrectionRetentionPeriod: z
    .number()
    .int({ error: () => translate("AI correction retention must be a whole number of days") })
    .min(AI_CORRECTION_RETENTION_MIN_DAYS, {
      error: () => translate("AI correction retention must be at least 30 days"),
    }),
});

export type DataRetentionFormValues = z.infer<typeof dataRetentionFormSchema>;

export const DATA_RETENTION_FORM_DEFAULTS: DataRetentionFormValues = {
  auditAccessRetentionDays: AUDIT_ACCESS_RETENTION_MIN_DAYS,
  auditOperationalRetentionDays: AUDIT_OPERATIONAL_RETENTION_MIN_DAYS,
  auditFinancialRetentionDays: AUDIT_FINANCIAL_RETENTION_MIN_DAYS,
  auditSecurityRetentionDays: AUDIT_SECURITY_RETENTION_MIN_DAYS,
  ediInboundFileRetentionPeriod: 0,
  ediMessageRetentionPeriod: 0,
  aiFeedbackRetentionPeriod: AI_FEEDBACK_RETENTION_DEFAULT_DAYS,
  agentEvalCaseRetentionPeriod: 365,
  aiAuditRetentionPeriod: AI_AUDIT_RETENTION_DEFAULT_DAYS,
  aiCorrectionRetentionPeriod: AI_CORRECTION_RETENTION_DEFAULT_DAYS,
};

/**
 * The form's values for a saved setting. A window saved before its field
 * existed reads back as zero; the form offers the server's default for it
 * rather than a value the server would refuse.
 */
export function dataRetentionFormValues(data: DataRetention): DataRetentionFormValues {
  return {
    auditAccessRetentionDays: Math.max(
      data.auditAccessRetentionDays,
      AUDIT_ACCESS_RETENTION_MIN_DAYS,
    ),
    auditOperationalRetentionDays: Math.max(
      data.auditOperationalRetentionDays,
      AUDIT_OPERATIONAL_RETENTION_MIN_DAYS,
    ),
    auditFinancialRetentionDays: Math.max(
      data.auditFinancialRetentionDays,
      AUDIT_FINANCIAL_RETENTION_MIN_DAYS,
    ),
    auditSecurityRetentionDays: Math.max(
      data.auditSecurityRetentionDays,
      AUDIT_SECURITY_RETENTION_MIN_DAYS,
    ),
    ediInboundFileRetentionPeriod: data.ediInboundFileRetentionPeriod,
    ediMessageRetentionPeriod: data.ediMessageRetentionPeriod,
    aiFeedbackRetentionPeriod:
      data.aiFeedbackRetentionPeriod > 0
        ? data.aiFeedbackRetentionPeriod
        : AI_FEEDBACK_RETENTION_DEFAULT_DAYS,
    agentEvalCaseRetentionPeriod: data.agentEvalCaseRetentionPeriod,
    aiAuditRetentionPeriod:
      data.aiAuditRetentionPeriod > 0
        ? data.aiAuditRetentionPeriod
        : AI_AUDIT_RETENTION_DEFAULT_DAYS,
    aiCorrectionRetentionPeriod:
      data.aiCorrectionRetentionPeriod > 0
        ? data.aiCorrectionRetentionPeriod
        : AI_CORRECTION_RETENTION_DEFAULT_DAYS,
  };
}
