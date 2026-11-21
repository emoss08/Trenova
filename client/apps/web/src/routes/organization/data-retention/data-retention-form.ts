import type { DataRetention } from "@/types/data-retention";
import { z } from "zod";
import { translate } from "@trenova/shared/i18n/runtime";

export const AI_FEEDBACK_RETENTION_MIN_DAYS = 30;
export const AI_FEEDBACK_RETENTION_DEFAULT_DAYS = 730;

/** Seven years, which is what most audits ask a record of AI decisions to be kept for. */
export const AI_AUDIT_RETENTION_DEFAULT_DAYS = 2555;
/** The server refuses less than a year. */
export const AI_AUDIT_RETENTION_MIN_DAYS = 365;

/** Three years after employment ends, the least 49 CFR 391.51 lets a driver qualification file be kept; the server refuses less. */
export const DRIVER_QUALIFICATION_RETENTION_MIN_DAYS = 1095;
export const DRIVER_QUALIFICATION_RETENTION_DEFAULT_DAYS = 1095;

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

/**
 * A period in days. An emptied field holds null, which is refused with its own words rather
 * than as a value of the wrong type; every period has to hold a number for the save to go.
 */
function days(empty: () => string) {
  return z.number({ error: empty }).int({ error: () => translate("Enter a whole number of days") });
}

/**
 * The server's rules for each period (`dataretention.DataRetention.Validate`). None depends on
 * another setting: the server's one conditional rule, a feedback or correction period of 1 to
 * 29 days, cannot arise because the page never offers 0 for either (0 there means the default).
 * A period with a "keep everything" switch holds 0 while the switch is off.
 */
export const dataRetentionFormSchema = z.object({
  auditAccessRetentionDays: days(() => translate("Enter how many days to keep reads and downloads"))
    .min(AUDIT_ACCESS_RETENTION_MIN_DAYS, {
      error: () => translate("Reads and downloads are kept for at least 90 days"),
    })
    .max(AUDIT_RETENTION_MAX_DAYS, {
      error: () => translate("Audit entries are kept for at most 3,650 days"),
    }),
  auditOperationalRetentionDays: days(() =>
    translate("Enter how many days to keep operational changes"),
  )
    .min(AUDIT_OPERATIONAL_RETENTION_MIN_DAYS, {
      error: () => translate("Operational changes are kept for at least 730 days"),
    })
    .max(AUDIT_RETENTION_MAX_DAYS, {
      error: () => translate("Audit entries are kept for at most 3,650 days"),
    }),
  auditFinancialRetentionDays: days(() =>
    translate("Enter how many days to keep financial changes"),
  )
    .min(AUDIT_FINANCIAL_RETENTION_MIN_DAYS, {
      error: () => translate("Financial changes are kept for at least 2,555 days (seven years)"),
    })
    .max(AUDIT_RETENTION_MAX_DAYS, {
      error: () => translate("Audit entries are kept for at most 3,650 days"),
    }),
  auditSecurityRetentionDays: days(() => translate("Enter how many days to keep security entries"))
    .min(AUDIT_SECURITY_RETENTION_MIN_DAYS, {
      error: () => translate("Security entries are kept for at least 730 days"),
    })
    .max(AUDIT_RETENTION_MAX_DAYS, {
      error: () => translate("Audit entries are kept for at most 3,650 days"),
    }),
  ediInboundFileRetentionPeriod: days(() =>
    translate("Enter how many days to keep inbound file contents"),
  ).min(0, { error: () => translate("A period cannot be negative") }),
  ediMessageRetentionPeriod: days(() =>
    translate("Enter how many days to keep message payloads"),
  ).min(0, { error: () => translate("A period cannot be negative") }),
  driverQualificationRetentionPeriod: days(() =>
    translate("Enter how many days to keep driver qualification files"),
  ).min(DRIVER_QUALIFICATION_RETENTION_MIN_DAYS, {
    error: () =>
      translate(
        "Driver qualification files must be kept for at least 1,095 days (three years) after employment ends",
      ),
  }),
  aiFeedbackRetentionPeriod: days(() => translate("Enter how many days to keep AI feedback")).min(
    AI_FEEDBACK_RETENTION_MIN_DAYS,
    { error: () => translate("AI feedback is kept for at least 30 days") },
  ),
  agentEvalCaseRetentionPeriod: days(() =>
    translate("Enter how many days to keep evaluation cases"),
  ).min(0, { error: () => translate("A period cannot be negative") }),
  aiAuditRetentionPeriod: days(() =>
    translate("Enter how many days to keep the AI audit trail"),
  ).min(AI_AUDIT_RETENTION_MIN_DAYS, {
    error: () => translate("The AI audit trail is kept for at least 365 days"),
  }),
  aiCorrectionRetentionPeriod: days(() =>
    translate("Enter how many days to keep AI corrections"),
  ).min(AI_CORRECTION_RETENTION_MIN_DAYS, {
    error: () => translate("AI corrections are kept for at least 30 days"),
  }),
});

export type DataRetentionFormValues = z.infer<typeof dataRetentionFormSchema>;

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
    driverQualificationRetentionPeriod:
      data.driverQualificationRetentionPeriod > 0
        ? data.driverQualificationRetentionPeriod
        : DRIVER_QUALIFICATION_RETENTION_DEFAULT_DAYS,
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
