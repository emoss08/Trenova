import { z } from "zod";

export const dataRetentionSchema = z.object({
  id: z.string().optional(),
  organizationId: z.string(),
  businessUnitId: z.string(),
  auditRetentionPeriod: z.number(),
  ediInboundFileRetentionPeriod: z.number().default(0),
  ediMessageRetentionPeriod: z.number().default(0),
  version: z.number().default(0),
  createdAt: z.number().optional(),
  updatedAt: z.number().optional(),
});

export type DataRetention = z.infer<typeof dataRetentionSchema>;

export type UpdateDataRetentionRequest = {
  auditRetentionPeriod: number;
  ediInboundFileRetentionPeriod: number;
  ediMessageRetentionPeriod: number;
};
