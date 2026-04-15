import { z } from "zod";

export const configFieldSpecSchema = z.object({
  key: z.string(),
  label: z.string(),
  type: z.enum(["string", "url", "password"]),
  required: z.boolean(),
  sensitive: z.boolean(),
  placeholder: z.string().optional(),
  helpText: z.string().optional(),
  default: z.string().optional(),
});

export const configFieldValueSchema = z.object({
  key: z.string(),
  value: z.string().optional(),
  hasValue: z.boolean(),
});

export const integrationConfigResponseSchema = z.object({
  type: z.string(),
  enabled: z.boolean(),
  fields: z.array(configFieldValueSchema),
  spec: z.array(configFieldSpecSchema),
  updatedAt: z.number().int(),
});

export const updateIntegrationConfigRequestSchema = z.object({
  enabled: z.boolean(),
  configuration: z.record(z.string(), z.string()),
});

export const integrationCatalogLinkSchema = z.object({
  kind: z.enum(["docs", "website", "support", "api"]),
  label: z.string(),
  url: z.string(),
});

export const integrationCatalogStatusSchema = z.object({
  connection: z.enum(["connected", "disconnected"]),
  connectionLabel: z.string(),
  configuration: z.enum(["configured", "needs_setup"]),
  configurationLabel: z.string(),
});

export const integrationCatalogItemSchema = z.object({
  type: z.string(),
  name: z.string(),
  description: z.string(),
  category: z.string(),
  categoryLabel: z.string(),
  logoUrl: z.string(),
  logoLightUrl: z.string().optional(),
  logoDarkUrl: z.string().optional(),
  docsUrl: z.string().optional(),
  websiteUrl: z.string().optional(),
  links: z.array(integrationCatalogLinkSchema),
  color: z.string(),
  glowFrom: z.string().optional(),
  glowTo: z.string().optional(),
  featured: z.boolean(),
  sortOrder: z.number().int(),
  primaryActionLabel: z.string(),
  enabled: z.boolean(),
  configured: z.boolean(),
  status: integrationCatalogStatusSchema,
  configSpec: z.array(configFieldSpecSchema).optional(),
  supportsTestConnect: z.boolean().optional(),
});

export const integrationCatalogResponseSchema = z.object({
  items: z.array(integrationCatalogItemSchema),
});

export type ConfigFieldSpec = z.infer<typeof configFieldSpecSchema>;
export type ConfigFieldValue = z.infer<typeof configFieldValueSchema>;
export type IntegrationConfigResponse = z.infer<typeof integrationConfigResponseSchema>;
export type UpdateIntegrationConfigRequest = z.infer<typeof updateIntegrationConfigRequestSchema>;
export type IntegrationCatalogItem = z.infer<typeof integrationCatalogItemSchema>;
export type IntegrationCatalogResponse = z.infer<typeof integrationCatalogResponseSchema>;
