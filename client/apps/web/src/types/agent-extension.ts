import { translate } from "@trenova/shared/i18n/runtime";
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
  /** What the extension gives agents; tools belong to it, so vendors of one are interchangeable. */
  capability: z.string(),
  capabilityLabel: z.string(),
  /** Another extension already on that gives agents the same capability, empty when none. */
  capabilityHeldBy: z.string().default(""),
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

/** What an extension's capability gives agents, as the server names it. */
export const AgentExtensionCapability = {
  WebResearch: "WebResearch",
  PortalAutomation: "PortalAutomation",
  DataAnalysis: "DataAnalysis",
} as const;

export const portalSiteSchema = z.object({
  id: z.string(),
  name: z.string(),
  purpose: z.string(),
  domains: z.array(z.string()),
  startUrl: z.string(),
  browserProfileId: z.string(),
  allowActions: z.boolean(),
  enabled: z.boolean(),
  hasSignIn: z.boolean(),
  hasAuthenticator: z.boolean().default(false),
  version: z.number().int(),
  createdAt: z.number().int(),
  updatedAt: z.number().int(),
});

export const portalSitesResponseSchema = z.object({
  items: z
    .array(portalSiteSchema)
    .nullish()
    .transform((value) => value ?? []),
});

export const portalRunStatusSchema = z.enum([
  "Running",
  "AwaitingCode",
  "Succeeded",
  "Failed",
  "Stopped",
  "TimedOut",
  "Strayed",
]);

export const portalRunSchema = z.object({
  id: z.string(),
  siteId: z.string(),
  siteName: z.string(),
  extensionType: z.string(),
  mode: z.enum(["Read", "Act"]),
  status: portalRunStatusSchema,
  instruction: z.string(),
  details: z
    .record(z.string(), z.string())
    .nullish()
    .transform((value) => value ?? {}),
  usedSignIn: z.boolean(),
  answer: z.string(),
  visitedUrls: z
    .array(z.string())
    .nullish()
    .transform((value) => value ?? []),
  offSiteUrls: z
    .array(z.string())
    .nullish()
    .transform((value) => value ?? []),
  steps: z.number().int(),
  costUsd: z.union([z.string(), z.number()]).transform((value) => Number(value) || 0),
  failure: z.string(),
  agentRunId: z.string().default(""),
  threadId: z.string().default(""),
  proposalId: z.string().default(""),
  requestedById: z.string().default(""),
  hasRecording: z.boolean(),
  /** How the portal asked for the sign-in code it sent, while the run waits on one. */
  codePrompt: z.string().default(""),
  /** When the browser stops waiting for the code; set only while the run waits. */
  codeExpiresAt: z.number().int().nullish(),
  /** How many codes a person has entered for the run. */
  codeAttempts: z.number().int().default(0),
  startedAt: z.number().int(),
  finishedAt: z.number().int().nullish(),
});

export const portalRunsResponseSchema = z.object({
  items: z
    .array(portalRunSchema)
    .nullish()
    .transform((value) => value ?? []),
  nextBefore: z.number().int(),
});

export const analysisRunStatusSchema = z.enum([
  "Running",
  "Succeeded",
  "Errored",
  "Failed",
  "TimedOut",
]);

export const analysisDatasetSchema = z.object({
  name: z.string(),
  source: z.enum(["ReportRun", "Attachment"]),
  sourceId: z.string(),
  label: z.string(),
  path: z.string(),
  rows: z.number().int(),
  bytes: z.number().int(),
});

export const analysisRunSchema = z.object({
  id: z.string(),
  extensionType: z.string(),
  status: analysisRunStatusSchema,
  purpose: z.string(),
  code: z.string(),
  datasets: z
    .array(analysisDatasetSchema)
    .nullish()
    .transform((value) => value ?? []),
  stdout: z.string(),
  stderr: z.string(),
  results: z
    .array(z.string())
    .nullish()
    .transform((value) => value ?? []),
  errorName: z.string(),
  errorValue: z.string(),
  traceback: z.string(),
  chartCount: z.number().int(),
  failure: z.string(),
  durationMs: z.number().int(),
  costUsd: z.union([z.string(), z.number()]).transform((value) => Number(value) || 0),
  agentRunId: z.string().default(""),
  threadId: z.string().default(""),
  requestedById: z.string().default(""),
  startedAt: z.number().int(),
  finishedAt: z.number().int().nullish(),
});

export const analysisRunsResponseSchema = z.object({
  items: z
    .array(analysisRunSchema)
    .nullish()
    .transform((value) => value ?? []),
  nextBefore: z.number().int(),
});

export const portalRecordingSchema = z.object({
  recordingUrl: z.string(),
  shareUrl: z.string(),
});

export const savePortalSiteRequestSchema = z.object({
  name: z.string(),
  purpose: z.string(),
  domains: z.array(z.string()),
  startUrl: z.string(),
  username: z.string(),
  password: z.string(),
  authenticatorKey: z.string(),
  clearSignIn: z.boolean(),
  clearAuthenticator: z.boolean(),
  browserProfileId: z.string(),
  allowActions: z.boolean(),
  enabled: z.boolean(),
  version: z.number().int(),
});

/** A sign-in code as a person types it from a text message or email; the server reads its shape. */
export const portalSignInCodeFormSchema = z.object({
  code: z
    .string()
    .trim()
    .min(1, { error: () => translate("Enter the code the portal sent you") }),
});

/** A site as the form edits it; a sign-in left blank keeps the stored one. */
export const portalSiteFormSchema = z.object({
  name: z
    .string()
    .trim()
    .min(1, { error: () => translate("Name the site") })
    .max(100, { error: () => translate("Keep the name to 100 characters") }),
  purpose: z
    .string()
    .trim()
    .max(500, { error: () => translate("Keep what it is for to 500 characters") }),
  domains: z
    .array(z.string().trim().min(1))
    .min(1, { error: () => translate("Add at least one address, such as portal.example.com") })
    .max(10, { error: () => translate("A site can have at most 10 addresses") }),
  startUrl: z
    .string()
    .trim()
    .min(1, { error: () => translate("Give the page the browser starts on") })
    .refine((value) => value.startsWith("https://"), {
      error: () => translate("The start page must be an https address"),
    }),
  username: z.string().trim(),
  password: z.string(),
  authenticatorKey: z.string(),
  clearSignIn: z.boolean(),
  clearAuthenticator: z.boolean(),
  browserProfileId: z
    .string()
    .trim()
    .max(100, { error: () => translate("Keep the profile to 100 characters") }),
  allowActions: z.boolean(),
  enabled: z.boolean(),
});

export type PortalSite = z.infer<typeof portalSiteSchema>;
export type PortalRun = z.infer<typeof portalRunSchema>;
export type PortalRunStatus = z.infer<typeof portalRunStatusSchema>;
export type PortalRunsResponse = z.infer<typeof portalRunsResponseSchema>;
export type PortalRecording = z.infer<typeof portalRecordingSchema>;
export type SavePortalSiteRequest = z.infer<typeof savePortalSiteRequestSchema>;
export type PortalSiteFormValues = z.infer<typeof portalSiteFormSchema>;
export type PortalSignInCodeFormValues = z.infer<typeof portalSignInCodeFormSchema>;
export type AnalysisRun = z.infer<typeof analysisRunSchema>;
export type AnalysisRunStatus = z.infer<typeof analysisRunStatusSchema>;
export type AnalysisRunsResponse = z.infer<typeof analysisRunsResponseSchema>;
