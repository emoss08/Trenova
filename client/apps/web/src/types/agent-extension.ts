import { z } from "zod";
import { configFieldSpecSchema, configFieldValueSchema } from "@/types/integration";

export const agentExtensionAvailabilitySchema = z.enum(["AllAgents", "SelectedAgents"]);

export const agentExtensionToolSchema = z.object({
  name: z.string(),
  label: z.string(),
  description: z.string(),
});

export const agentExtensionUsageSchema = z.object({
  requestsToday: z.number().int(),
  requestsThisMonth: z.number().int(),
  failuresThisMonth: z.number().int(),
  costThisMonthUsd: z.union([z.string(), z.number()]).transform((value) => Number(value) || 0),
});

export const agentExtensionCatalogItemSchema = z.object({
  type: z.string(),
  name: z.string(),
  vendor: z.string(),
  summary: z.string(),
  description: z.string(),
  category: z.string(),
  categoryLabel: z.string(),
  brandDomain: z.string(),
  docsUrl: z.string(),
  websiteUrl: z.string(),
  pricingUrl: z.string(),
  capabilities: z.array(z.string()),
  tools: z.array(agentExtensionToolSchema),
  dataNotice: z.string(),
  featured: z.boolean(),
  sortOrder: z.number().int(),
  releasedAt: z.number().int(),
  enabled: z.boolean(),
  configured: z.boolean(),
  availability: agentExtensionAvailabilitySchema,
  configSpec: z
    .array(configFieldSpecSchema)
    .nullish()
    .transform((value) => value ?? []),
  supportsTestConnect: z.boolean(),
  dailyRequestLimit: z.number().int(),
  usage: agentExtensionUsageSchema,
  enabledAt: z.number().int().nullish(),
  updatedAt: z.number().int(),
  version: z.number().int(),
});

export const agentExtensionCatalogResponseSchema = z.object({
  items: z.array(agentExtensionCatalogItemSchema),
  categories: z.array(z.object({ value: z.string(), label: z.string() })),
});

export const agentExtensionConfigResponseSchema = z.object({
  type: z.string(),
  enabled: z.boolean(),
  availability: agentExtensionAvailabilitySchema,
  fields: z.array(configFieldValueSchema),
  spec: z.array(configFieldSpecSchema),
  version: z.number().int(),
  updatedAt: z.number().int(),
});

export const updateAgentExtensionRequestSchema = z.object({
  enabled: z.boolean(),
  availability: agentExtensionAvailabilitySchema,
  configuration: z.record(z.string(), z.string()),
  version: z.number().int(),
});

export const agentExtensionTestResponseSchema = z.object({
  type: z.string(),
  success: z.boolean(),
  checkedAt: z.number().int(),
  latencyMs: z.number().int(),
  message: z.string(),
});

export type AgentExtensionAvailability = z.infer<typeof agentExtensionAvailabilitySchema>;
export type AgentExtensionCatalogItem = z.infer<typeof agentExtensionCatalogItemSchema>;
export type AgentExtensionCatalogResponse = z.infer<typeof agentExtensionCatalogResponseSchema>;
export type AgentExtensionConfigResponse = z.infer<typeof agentExtensionConfigResponseSchema>;
export type UpdateAgentExtensionRequest = z.infer<typeof updateAgentExtensionRequestSchema>;
