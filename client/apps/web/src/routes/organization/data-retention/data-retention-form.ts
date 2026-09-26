import type { DataRetention } from "@/types/data-retention";
import { z } from "zod";

export const AI_FEEDBACK_RETENTION_MIN_DAYS = 30;
export const AI_FEEDBACK_RETENTION_DEFAULT_DAYS = 730;

/** Seven years, which is what most audits ask a record of AI decisions to be kept for. */
export const AI_AUDIT_RETENTION_DEFAULT_DAYS = 2555;
/** The server refuses less than a year. */
export const AI_AUDIT_RETENTION_MIN_DAYS = 365;

export const AI_CORRECTION_RETENTION_MIN_DAYS = 30;
export const AI_CORRECTION_RETENTION_DEFAULT_DAYS = 730;

export const dataRetentionFormSchema = z.object({
  auditRetentionPeriod: z.number().int().min(1, "Audit retention must be at least 1 day"),
  ediInboundFileRetentionPeriod: z
    .number()
    .int()
    .min(0, "EDI inbound file retention cannot be negative"),
  ediMessageRetentionPeriod: z.number().int().min(0, "EDI message retention cannot be negative"),
  aiFeedbackRetentionPeriod: z
    .number()
    .int()
    .min(AI_FEEDBACK_RETENTION_MIN_DAYS, "AI feedback retention must be at least 30 days"),
  agentEvalCaseRetentionPeriod: z
    .number()
    .int()
    .min(0, "Agent evaluation case retention cannot be negative"),
  aiAuditRetentionPeriod: z
    .number()
    .int("AI audit trail retention must be a whole number of days")
    .min(AI_AUDIT_RETENTION_MIN_DAYS, "AI audit trail retention must be at least 365 days"),
  aiCorrectionRetentionPeriod: z
    .number()
    .int("AI correction retention must be a whole number of days")
    .min(AI_CORRECTION_RETENTION_MIN_DAYS, "AI correction retention must be at least 30 days"),
});

export type DataRetentionFormValues = z.infer<typeof dataRetentionFormSchema>;

export const DATA_RETENTION_FORM_DEFAULTS: DataRetentionFormValues = {
  auditRetentionPeriod: 120,
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
    auditRetentionPeriod: data.auditRetentionPeriod,
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
