import { z } from "zod";

export const dataRetentionSchema = z.object({
  id: z.string().nullish(),
  organizationId: z.string(),
  businessUnitId: z.string(),
  auditAccessRetentionDays: z.number().default(90),
  auditOperationalRetentionDays: z.number().default(730),
  auditFinancialRetentionDays: z.number().default(2555),
  auditSecurityRetentionDays: z.number().default(730),
  ediInboundFileRetentionPeriod: z.number().default(0),
  ediMessageRetentionPeriod: z.number().default(0),
  agentEvalCaseRetentionPeriod: z.number().default(365),
  aiFeedbackRetentionPeriod: z.number().default(730),
  aiAuditRetentionPeriod: z.number().default(2555),
  aiCorrectionRetentionPeriod: z.number().default(730),
  version: z.number().default(0),
  createdAt: z.number().optional(),
  updatedAt: z.number().optional(),
});

export type DataRetention = z.infer<typeof dataRetentionSchema>;

export type UpdateDataRetentionRequest = {
  auditAccessRetentionDays: number;
  auditOperationalRetentionDays: number;
  auditFinancialRetentionDays: number;
  auditSecurityRetentionDays: number;
  ediInboundFileRetentionPeriod: number;
  ediMessageRetentionPeriod: number;
  agentEvalCaseRetentionPeriod: number;
  aiFeedbackRetentionPeriod: number;
  aiAuditRetentionPeriod: number;
  aiCorrectionRetentionPeriod: number;
};
