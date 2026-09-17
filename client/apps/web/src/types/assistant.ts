import { optionalIdSchema } from "@trenova/shared/types/helpers";
import { z } from "zod";

export const agentTemplateKindSchema = z.enum([
  "DispatchAssistant",
  "BillingAssistant",
  "ComplianceAssistant",
  "CustomerAssistant",
  "GeneralAssistant",
  "BillingException",
  "DispatchAssignment",
  "ImportAssistant",
]);

export const autonomyTierSchema = z.enum(["Propose", "ActWithApproval", "AutoExecute"]);

export const triggerModeSchema = z.enum(["Chat", "Scheduled", "Event", "Continuous"]);

export const outputModeSchema = z.enum(["Conversational", "Report"]);

export const contextProviderSchema = z.enum(["Organization", "Clock", "User", "Page", "Tools"]);

export const messageRoleSchema = z.enum(["User", "Assistant", "Tool"]);

export const threadStatusSchema = z.enum(["Active", "Archived"]);

/** Server-side `nullzero` arrays arrive as null when empty; every list here reads as []. */
const nullableList = <T extends z.ZodType>(item: T) =>
  z.preprocess((value) => value ?? [], z.array(item));

/**
 * An enum-typed Go string the server leaves unset. A plain string field
 * marshals as `""`, which is not one of the enum's values.
 *
 * This is a stricter cousin of `nullableEnumSchema` in shared/types/helpers: it
 * yields `T | null` rather than `T | null | undefined`, which these schemas rely
 * on, so the two are deliberately not the same function.
 */
const nullableEnum = <T extends z.ZodType>(item: T) =>
  z.preprocess((value) => (value === "" || value == null ? null : value), item.nullable());

const toolTiersSchema = z.preprocess(
  (value) => value ?? {},
  z.record(z.string(), autonomyTierSchema),
);

export const agentDefinitionSchema = z.object({
  id: z.string(),
  businessUnitId: z.string(),
  organizationId: z.string(),
  name: z.string(),
  description: z.string().optional().default(""),
  /** The starter this agent began from. It carries no restriction. */
  template: nullableEnum(agentTemplateKindSchema),
  /** Chosen icon name; empty means the client derives one. */
  icon: optionalIdSchema,
  /** Chosen accent name; empty means the client derives one. */
  accent: optionalIdSchema,
  /** Organization-authored instructions, placed after Trenova's safety preamble. */
  instructions: z.string().optional().default(""),
  guardrails: nullableList(z.string()),
  toolNames: nullableList(z.string()),
  toolTiers: toolTiersSchema,
  autonomyCeiling: autonomyTierSchema,
  enabled: z.boolean().default(false),
  shadowMode: z.boolean().default(false),
  decisionTimeoutSeconds: z.number().default(86400),
  triggerMode: triggerModeSchema.default("Chat"),
  cronExpression: z.string().optional().default(""),
  cronTimezone: z.string().optional().default(""),
  eventKinds: nullableList(z.string()),
  intervalSeconds: z.number().default(0),
  endsAt: z.number().nullish(),
  maxConcurrentRuns: z.number().default(1),
  runTimeoutSeconds: z.number().default(600),
  maxToolCalls: z.number().default(12),
  contextProviders: nullableList(contextProviderSchema),
  outputMode: outputModeSchema.default("Conversational"),
  preferredProviderId: optionalIdSchema,
  systemKey: z.string().optional().default(""),
  lastRunAt: z.number().nullish(),
  nextRunAt: z.number().nullish(),
  version: z.number().default(0),
  createdAt: z.number(),
  updatedAt: z.number(),
});

export const agentTemplateSchema = z.object({
  template: agentTemplateKindSchema,
  label: z.string(),
  description: z.string(),
  starterInstructions: z.string().optional().default(""),
  starterTools: nullableList(z.string()),
  starterTrigger: triggerModeSchema,
  starterEvents: nullableList(z.string()),
  starterCron: z.string().optional().default(""),
  starterCeiling: autonomyTierSchema,
  starterOutput: outputModeSchema,
  systemKey: z.string().optional().default(""),
  contextProviders: nullableList(contextProviderSchema),
});

export const agentTemplateListSchema = z.object({
  templates: z.array(agentTemplateSchema),
});

export const toolCatalogEntrySchema = z.object({
  name: z.string(),
  description: z.string(),
  parameters: z.record(z.string(), z.unknown()).nullish(),
  kind: z.enum(["query", "action"]),
  resource: z.string(),
  operation: z.string(),
  defaultAutonomyTier: z.string().optional().default(""),
  reversible: z.boolean().default(false),
});

export const toolCatalogSchema = z.object({
  tools: z.array(toolCatalogEntrySchema),
});

export const agentEventDescriptorSchema = z.object({
  kind: z.string(),
  subjectType: z.string(),
  label: z.string(),
  description: z.string(),
});

export const agentEventListSchema = z.object({
  events: z.array(agentEventDescriptorSchema),
});

export const previewPromptResponseSchema = z.object({
  prompt: z.string(),
});

export const saveAgentDefinitionRequestSchema = z.object({
  name: z.string().trim().min(1, "Name is required"),
  description: z.string().optional().default(""),
  template: agentTemplateKindSchema.nullable().default(null),
  icon: z.string().optional().default(""),
  accent: z.string().optional().default(""),
  instructions: z
    .string()
    .max(20000, "Instructions cannot be longer than 20000 characters")
    .optional()
    .default(""),
  guardrails: z.array(z.string()).default([]),
  toolNames: z.array(z.string()).default([]),
  toolTiers: z.record(z.string(), autonomyTierSchema).default({}),
  autonomyCeiling: autonomyTierSchema,
  enabled: z.boolean().default(true),
  shadowMode: z.boolean().default(false),
  decisionTimeoutSeconds: z.number().min(60).default(86400),
  triggerMode: triggerModeSchema.default("Chat"),
  cronExpression: z.string().optional().default(""),
  cronTimezone: z.string().optional().default(""),
  eventKinds: z.array(z.string()).default([]),
  intervalSeconds: z.number().min(0).default(0),
  endsAt: z.number().nullable().default(null),
  maxConcurrentRuns: z.number().min(1).max(10).default(1),
  runTimeoutSeconds: z.number().min(60).max(3600).default(600),
  maxToolCalls: z.number().min(1).max(64).default(12),
  contextProviders: z.array(contextProviderSchema).default([]),
  outputMode: outputModeSchema.default("Conversational"),
  preferredProviderId: optionalIdSchema,
  version: z.number().default(0),
});

export const toolCallRecordSchema = z.object({
  id: z.string(),
  name: z.string(),
  arguments: z.record(z.string(), z.unknown()).nullish(),
});

/** What the person was looking at when they asked; mirrors the server's PageContext. */
export const pageContextSchema = z.object({
  path: z.string(),
  entityType: z.string().optional().default(""),
  entityId: z.string().optional().default(""),
  title: z.string().optional().default(""),
});

export const assistantMessageSchema = z.object({
  id: z.string(),
  threadId: z.string(),
  sequence: z.number(),
  role: messageRoleSchema,
  content: z.string().optional().default(""),
  toolCalls: z.array(toolCallRecordSchema).nullish(),
  toolCallId: z.string().optional().default(""),
  toolName: z.string().optional().default(""),
  toolFailed: z.boolean().default(false),
  /** Which guard layer decided this turn, kept so a refusal can be explained. */
  scopeStage: z.string().optional().default(""),
  scopeCategory: z.string().optional().default(""),
  scopeReason: z.string().optional().default(""),
  refused: z.boolean().default(false),
  pageContext: pageContextSchema.nullish(),
  model: z.string().optional().default(""),
  inputTokens: z.number().default(0),
  outputTokens: z.number().default(0),
  createdAt: z.number(),
});

export const assistantThreadSchema = z.object({
  id: z.string(),
  businessUnitId: z.string(),
  organizationId: z.string(),
  userId: z.string(),
  agentDefinitionId: z.string(),
  title: z.string().optional().default(""),
  status: threadStatusSchema,
  lastMessageAt: z.number().default(0),
  version: z.number().default(0),
  createdAt: z.number(),
  updatedAt: z.number(),
});

export const assistantThreadListSchema = z.object({
  items: z.array(assistantThreadSchema),
  total: z.number().default(0),
});

export const assistantMessageListSchema = z.object({
  results: z.array(assistantMessageSchema),
});

export const proposalStatusSchema = z.enum([
  "Pending",
  "Accepted",
  "Rejected",
  "Modified",
  "Expired",
  "Superseded",
  "Executed",
  "ExecutionFailed",
]);

export const proposalDecisionSchema = z.enum(["Accepted", "Rejected", "Modified"]);

/**
 * A proposal is a write the assistant asked for and has not made. It is a
 * persisted record, so it survives a refresh and is decided through the same
 * endpoint as any other agent's proposal.
 */
export const assistantProposalSchema = z.object({
  id: z.string(),
  runId: z.string().optional().default(""),
  toolName: z.string(),
  arguments: z.record(z.string(), z.unknown()).nullish(),
  rationale: z.string().optional().default(""),
  autonomyTier: autonomyTierSchema,
  status: proposalStatusSchema,
  sourceMessageId: optionalIdSchema,
  /** The model's own estimate, 0 to 1. */
  confidence: z.number().min(0).max(1).nullish(),
  /** Set once an approved proposal has actually run. */
  executedAt: z.number().nullish(),
  /** Why an approved proposal failed to run, shown instead of a success state. */
  executionError: z.string().optional().default(""),
});

export const assistantProposalListSchema = z.object({
  results: z.array(assistantProposalSchema),
});

export const sendMessageResultSchema = z.object({
  thread: assistantThreadSchema,
  messages: z.array(assistantMessageSchema),
  reply: z.string().optional().default(""),
  refused: z.boolean().default(false),
  proposals: z.array(assistantProposalSchema).nullish(),
  /**
   * The turn proposed a write that could not be saved for approval. Nothing ran,
   * but there is nothing to approve either, so the client must say so rather than
   * offer a decision the server cannot honor.
   */
  proposalsUnrecorded: z.boolean().default(false),
});

/**
 * The events a streamed turn emits, in the shape the server sends them. The
 * union is discriminated on the SSE event name so a reducer can switch on it
 * without a second lookup.
 */
export const assistantAcceptedEventSchema = z.object({
  content: z.string(),
  scopeStage: z.string().optional().default(""),
  scopeCategory: z.string().optional().default(""),
});

export const assistantRefusedEventSchema = z.object({
  message: z.string(),
  stage: z.string().optional().default(""),
  category: z.string().optional().default(""),
  reason: z.string().optional().default(""),
});

export const assistantDeltaEventSchema = z.object({ text: z.string() });

export const assistantMessageEventSchema = z.object({
  content: z.string().optional().default(""),
  toolCalls: z.array(toolCallRecordSchema).nullish(),
  model: z.string().optional().default(""),
});

export const assistantToolStartedEventSchema = z.object({
  callId: z.string(),
  name: z.string(),
  arguments: z.record(z.string(), z.unknown()).nullish(),
});

export const assistantToolFinishedEventSchema = z.object({
  callId: z.string(),
  name: z.string(),
  failed: z.boolean().default(false),
  proposed: z.boolean().default(false),
  content: z.string().optional().default(""),
});

export const assistantErrorEventSchema = z.object({ message: z.string() });

export type AssistantStreamEvent =
  | { event: "accepted"; data: z.infer<typeof assistantAcceptedEventSchema> }
  | { event: "refused"; data: z.infer<typeof assistantRefusedEventSchema> }
  | { event: "delta"; data: z.infer<typeof assistantDeltaEventSchema> }
  | { event: "message"; data: z.infer<typeof assistantMessageEventSchema> }
  | { event: "tool_started"; data: z.infer<typeof assistantToolStartedEventSchema> }
  | { event: "tool_finished"; data: z.infer<typeof assistantToolFinishedEventSchema> }
  | { event: "done"; data: SendMessageResult }
  | { event: "error"; data: z.infer<typeof assistantErrorEventSchema> };

/**
 * Parses one raw SSE frame into a typed event. An event name this client does
 * not know returns null so a newer server can add events without breaking an
 * older reader; a known event with a malformed body throws, because that is a
 * contract violation rather than an extension.
 */
export function parseAssistantStreamEvent(event: string, raw: string): AssistantStreamEvent | null {
  const data: unknown = raw === "" ? {} : JSON.parse(raw);
  switch (event) {
    case "accepted":
      return { event, data: assistantAcceptedEventSchema.parse(data) };
    case "refused":
      return { event, data: assistantRefusedEventSchema.parse(data) };
    case "delta":
      return { event, data: assistantDeltaEventSchema.parse(data) };
    case "message":
      return { event, data: assistantMessageEventSchema.parse(data) };
    case "tool_started":
      return { event, data: assistantToolStartedEventSchema.parse(data) };
    case "tool_finished":
      return { event, data: assistantToolFinishedEventSchema.parse(data) };
    case "done":
      return { event, data: sendMessageResultSchema.parse(data) };
    case "error":
      return { event, data: assistantErrorEventSchema.parse(data) };
    default:
      return null;
  }
}

export type AgentTemplateKind = z.infer<typeof agentTemplateKindSchema>;
export type AutonomyTier = z.infer<typeof autonomyTierSchema>;
export type TriggerMode = z.infer<typeof triggerModeSchema>;
export type OutputMode = z.infer<typeof outputModeSchema>;
export type ContextProvider = z.infer<typeof contextProviderSchema>;
export type AgentDefinition = z.infer<typeof agentDefinitionSchema>;
export type AgentTemplate = z.infer<typeof agentTemplateSchema>;
export type ToolCatalogEntry = z.infer<typeof toolCatalogEntrySchema>;
export type AgentEventDescriptor = z.infer<typeof agentEventDescriptorSchema>;
export type SaveAgentDefinitionRequest = z.infer<typeof saveAgentDefinitionRequestSchema>;
export type AssistantThread = z.infer<typeof assistantThreadSchema>;
export type AssistantMessage = z.infer<typeof assistantMessageSchema>;
export type AssistantPageContext = z.infer<typeof pageContextSchema>;
export type SendMessageResult = z.infer<typeof sendMessageResultSchema>;
export type AssistantProposal = z.infer<typeof assistantProposalSchema>;
export type ProposalStatus = z.infer<typeof proposalStatusSchema>;
export type ProposalDecision = z.infer<typeof proposalDecisionSchema>;
