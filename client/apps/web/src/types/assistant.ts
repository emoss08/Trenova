import { z } from "zod";

export const agentKindSchema = z.enum([
  "DispatchAssistant",
  "BillingAssistant",
  "ComplianceAssistant",
  "CustomerAssistant",
  "GeneralAssistant",
]);

export const autonomyTierSchema = z.enum(["Propose", "ActWithApproval", "AutoExecute"]);

export const messageRoleSchema = z.enum(["User", "Assistant", "Tool"]);

export const threadStatusSchema = z.enum(["Active", "Archived"]);

export const agentDefinitionSchema = z.object({
  id: z.string(),
  businessUnitId: z.string(),
  organizationId: z.string(),
  name: z.string(),
  description: z.string().optional().default(""),
  kind: agentKindSchema,
  /** Organization guidance. Delivered to the model as data, never as instruction. */
  focus: z.string().optional().default(""),
  toolNames: z.array(z.string()).optional().default([]),
  autonomyCeiling: autonomyTierSchema,
  enabled: z.boolean().default(false),
  version: z.number().default(0),
  createdAt: z.number(),
  updatedAt: z.number(),
});

export const agentToolDescriptorSchema = z.object({
  name: z.string(),
  description: z.string(),
  parameters: z.record(z.string(), z.unknown()).optional(),
  autonomyTier: autonomyTierSchema.optional(),
});

export const agentTemplateSchema = z.object({
  kind: agentKindSchema,
  label: z.string(),
  description: z.string(),
  mutatingAllowed: z.boolean().default(false),
  availableTools: z.array(agentToolDescriptorSchema).optional().default([]),
});

export const agentTemplateListSchema = z.object({
  templates: z.array(agentTemplateSchema),
});

export const saveAgentDefinitionRequestSchema = z.object({
  name: z.string().trim().min(1, "Name is required"),
  description: z.string().optional().default(""),
  kind: agentKindSchema,
  focus: z.string().max(2000, "Focus cannot be longer than 2000 characters").optional().default(""),
  toolNames: z.array(z.string()).default([]),
  autonomyCeiling: autonomyTierSchema,
  enabled: z.boolean().default(true),
  version: z.number().default(0),
});

export const toolCallRecordSchema = z.object({
  id: z.string(),
  name: z.string(),
  arguments: z.record(z.string(), z.unknown()).optional(),
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
  sourceMessageId: z.string().optional().default(""),
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

export type AgentKind = z.infer<typeof agentKindSchema>;
export type AutonomyTier = z.infer<typeof autonomyTierSchema>;
export type AgentDefinition = z.infer<typeof agentDefinitionSchema>;
export type AgentTemplate = z.infer<typeof agentTemplateSchema>;
export type AgentToolDescriptor = z.infer<typeof agentToolDescriptorSchema>;
export type SaveAgentDefinitionRequest = z.infer<typeof saveAgentDefinitionRequestSchema>;
export type AssistantThread = z.infer<typeof assistantThreadSchema>;
export type AssistantMessage = z.infer<typeof assistantMessageSchema>;
export type SendMessageResult = z.infer<typeof sendMessageResultSchema>;
export type AssistantProposal = z.infer<typeof assistantProposalSchema>;
export type ProposalStatus = z.infer<typeof proposalStatusSchema>;
export type ProposalDecision = z.infer<typeof proposalDecisionSchema>;
