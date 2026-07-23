import { z } from "zod";
import { createLimitOffsetResponse } from "./server";

export const apiKeyStatusSchema = z.enum(["active"]);

export const apiKeyPermissionInputSchema = z.object({
  resource: z.string(),
  operations: z.array(z.string()).min(1),
  dataScope: z.string().default("organization"),
});

export const apiKeySchema = z.object({
  id: z.string(),
  businessUnitId: z.string(),
  organizationId: z.string(),
  name: z.string(),
  description: z.string().optional().default(""),
  keyPrefix: z.string(),
  status: z.string(),
  expiresAt: z.number().optional().default(0),
  lastUsedAt: z.number().optional().default(0),
  createdAt: z.number(),
  updatedAt: z.number(),
  permissionScope: z.string().optional().default("restricted"),
  permissions: z.array(apiKeyPermissionInputSchema).optional().default([]),
});

export const apiKeySecretSchema = apiKeySchema.extend({
  token: z.string(),
});

export const apiKeyListSchema = createLimitOffsetResponse(apiKeySchema);

export const createApiKeyRequestSchema = z.object({
  name: z.string().trim().min(1),
  description: z.string().optional().default(""),
  expiresAt: z.number().optional().default(0),
  permissions: z.array(apiKeyPermissionInputSchema).min(1),
});

export const updateApiKeyRequestSchema = createApiKeyRequestSchema;

export type ApiKey = z.infer<typeof apiKeySchema>;
export type ApiKeySecret = z.infer<typeof apiKeySecretSchema>;
export type ApiKeyList = z.infer<typeof apiKeyListSchema>;
export type CreateApiKeyRequest = z.infer<typeof createApiKeyRequestSchema>;
export type UpdateApiKeyRequest = z.infer<typeof updateApiKeyRequestSchema>;
export type ApiKeyPermissionInput = z.infer<typeof apiKeyPermissionInputSchema>;
