import { z } from "zod";

export const dataRetentionSchema = z.object({
  id: z.string().nullish(),
  organizationId: z.string(),
  businessUnitId: z.string(),
  auditRetentionPeriod: z.number(),
  ediInboundFileRetentionPeriod: z.number().default(0),
  ediMessageRetentionPeriod: z.number().default(0),
  agentEvalCaseRetentionPeriod: z.number().default(365),
  aiFeedbackRetentionPeriod: z.number().default(730),
  aiAuditRetentionPeriod: z.number().default(2555),
  version: z.number().default(0),
  createdAt: z.number().optional(),
  updatedAt: z.number().optional(),
});

export type DataRetention = z.infer<typeof dataRetentionSchema>;

export type UpdateDataRetentionRequest = {
  auditRetentionPeriod: number;
  ediInboundFileRetentionPeriod: number;
  ediMessageRetentionPeriod: number;
  agentEvalCaseRetentionPeriod: number;
  aiFeedbackRetentionPeriod: number;
  aiAuditRetentionPeriod: number;
};
