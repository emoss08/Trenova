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
  "LoadMonitor",
  "ShipmentIntake",
  "CashApplication",
  "DetentionDesk",
  "CredentialDesk",
  "CustomerUpdateDesk",
  "CarrierRiskDesk",
  "IntakeDesk",
]);

export const autonomyTierSchema = z.enum(["Propose", "ActWithApproval", "AutoExecute"]);

export const triggerModeSchema = z.enum(["Chat", "Scheduled", "Event", "Continuous"]);

export const outputModeSchema = z.enum(["Conversational", "Report"]);

export const contextProviderSchema = z.enum([
  "Organization",
  "Clock",
  "User",
  "Page",
  "Tools",
  "Memory",
]);

export const messageRoleSchema = z.enum(["User", "Assistant", "Tool"]);

export const messageKindSchema = z.enum(["Message", "DecisionNote", "Delegated"]);

export const threadStatusSchema = z.enum(["Active", "Archived"]);

/**
 * Where a conversation began. A quick question from the palette (Ask) is
 * not listed until the person keeps it; every other origin is a
 * conversation from the start.
 */
export const threadOriginSchema = z.enum(["Panel", "Desk", "Ask", "Watchtower", "Briefing"]);

/** What an artifact is, which decides how the Desk renders it. */
export const artifactKindSchema = z.enum([
  "report_preview",
  "report_run",
  "email_draft",
  "plan",
  "entity_card",
  "table_view",
  "rate_explanation",
  "dashboard_ref",
  "briefing",
  "inbound_message",
  "run_diff",
  "document",
  "navigation",
]);

export const artifactStatusSchema = z.enum(["Pending", "Ready", "Failed", "Sent"]);

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

/** A decimal the server writes as a string, read as a number of dollars; null for no cap. */
const nullableMoney = z.preprocess(
  (value) => (value === null || value === undefined || value === "" ? null : Number(value)),
  z.number().min(0).nullable(),
);

const toolLimitsSchema = z.preprocess(
  (value) => value ?? {},
  z.record(z.string(), z.number().int().nonnegative()),
);

const toolTiersSchema = z.preprocess(
  (value) => value ?? {},
  z.record(z.string(), autonomyTierSchema),
);

/**
 * One row of the earned-autonomy ledger: how decisions on an agent's
 * proposals for one tool have gone. Rows exist only for tools that have had a
 * decision.
 */
export const toolTrustSchema = z.object({
  id: z.string(),
  organizationId: z.string(),
  businessUnitId: z.string(),
  agentDefinitionId: z.string(),
  toolName: z.string(),
  streak: z.number().default(0),
  approvals: z.number().default(0),
  modifications: z.number().default(0),
  rejections: z.number().default(0),
  executionFailures: z.number().default(0),
  earnedTier: nullableEnum(autonomyTierSchema),
  lastDecisionAt: z.number().nullish(),
  promotedAt: z.number().nullish(),
  demotedAt: z.number().nullish(),
  version: z.number().default(0),
  createdAt: z.number(),
  updatedAt: z.number(),
});

export const toolTrustListSchema = z.object({
  results: z.array(toolTrustSchema),
});

/** Where an agent stands against its caps this month and today. */
export const agentBudgetStatusSchema = z.object({
  monthStart: z.number(),
  dayStart: z.number(),
  spentUsd: z.string(),
  monthlyBudgetUsd: z.string().nullish(),
  monthCalls: z.number().default(0),
  unpricedCalls: z.number().default(0),
  runsToday: z.number().default(0),
  dailyRunLimit: z.number().default(0),
  tools: nullableList(
    z.object({
      tool: z.string(),
      used: z.number().default(0),
      limit: z.number().default(0),
    }),
  ),
  simulationMode: z.boolean().default(false),
});

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
  monthlyBudgetUsd: nullableMoney,
  dailyRunLimit: z.number().int().nonnegative().default(0),
  toolDailyLimits: toolLimitsSchema,
  simulationMode: z.boolean().default(false),
  contextProviders: nullableList(contextProviderSchema),
  outputMode: outputModeSchema.default("Conversational"),
  preferredProviderId: optionalIdSchema,
  /** The agents this one may hand a task to, in the order they were chosen. */
  delegateIds: nullableList(z.string()),
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

/**
 * What a tool call did, as the server classifies it: read records, change one,
 * move the app to a page, find the tools for the job, put a result in front of
 * the person, or ask them something. Absent for a tool the server no longer
 * knows, and an effect this client has not heard of reads as absent, so a
 * newer server cannot break an older reader.
 */
export const toolEffectSchema = z.enum([
  "lookup",
  "change",
  "navigate",
  "discover",
  "present",
  "ask",
  "delegate",
]);

const optionalToolEffect = toolEffectSchema.optional().catch(undefined);

export const toolCatalogEntrySchema = z.object({
  name: z.string(),
  description: z.string(),
  parameters: z.record(z.string(), z.unknown()).nullish(),
  kind: z.enum(["query", "action"]),
  resource: z.string(),
  operation: z.string(),
  defaultAutonomyTier: z.string().optional().default(""),
  reversible: z.boolean().default(false),
  /** Held by every agent without being chosen: memory, escalation, review. */
  core: z.boolean().default(false),
  /** Tools this one takes its arguments from; the reads among them come with it. */
  prerequisites: z.array(z.string()).default([]),
  effect: optionalToolEffect,
  /** The extension the tool comes with, empty for Trenova's own tools. */
  extension: z.string().optional().default(""),
  /** The extension gives the tool to every agent, so it is never chosen. */
  grantedToEveryAgent: z.boolean().default(false),
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

/** The most agents one agent may hand work to, as the server enforces it. */
export const MAX_DELEGATES = 8;

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
  /** Dollars per calendar month across the agent's runs; null for no cap. */
  monthlyBudgetUsd: z.number().min(0).nullable().default(null),
  /** Runs per day; 0 for no cap. */
  dailyRunLimit: z.number().int().min(0).max(10000).default(0),
  /** Executions per tool per day, keyed by tool name; absent for no cap. */
  toolDailyLimits: z.record(z.string(), z.number().int().min(0).max(10000)).default({}),
  simulationMode: z.boolean().default(false),
  contextProviders: z.array(contextProviderSchema).default([]),
  outputMode: outputModeSchema.default("Conversational"),
  preferredProviderId: optionalIdSchema,
  /**
   * The agents this one may hand a task to. Absent keeps the saved list; a
   * list, empty or not, replaces it.
   */
  delegateIds: z.array(z.string()).max(MAX_DELEGATES).optional(),
  version: z.number().default(0),
});

export const toolCallRecordSchema = z.object({
  id: z.string(),
  name: z.string(),
  arguments: z.record(z.string(), z.unknown()).nullish(),
  effect: optionalToolEffect,
});

/** One filter as a table carries it, in the shape the list tools take. */
export const pageViewFilterSchema = z.object({
  field: z.string(),
  operator: z.string(),
  value: z.unknown().optional(),
});

export const pageViewSortSchema = z.object({
  field: z.string(),
  direction: z.string(),
});

/** One figure above a table, as the person read it. */
export const pageViewKpiSchema = z.object({
  label: z.string(),
  value: z.string(),
  sub: z.string().optional(),
});

/**
 * The table the page was showing: its filters, sort, selection and figures,
 * so "these rows" can be re-run as a query. Mirrors the server's PageView,
 * which bounds every list and validates the resource and the operators.
 */
export const pageViewSchema = z.object({
  resource: z.string(),
  query: z.string().optional(),
  fieldFilters: z.array(pageViewFilterSchema).optional(),
  filterGroups: z.array(z.object({ filters: z.array(pageViewFilterSchema) })).optional(),
  sort: z.array(pageViewSortSchema).optional(),
  selection: z.object({ count: z.number(), ids: z.array(z.string()).optional() }).optional(),
  kpis: z.array(pageViewKpiSchema).optional(),
  visibleColumns: z.array(z.string()).optional(),
  rowCount: z.number().nullish(),
});

/** What the person was looking at when they asked; mirrors the server's PageContext. */
export const pageContextSchema = z.object({
  path: z.string(),
  entityType: z.string().optional().default(""),
  entityId: z.string().optional().default(""),
  title: z.string().optional().default(""),
  view: pageViewSchema.nullish(),
});

/** A record the person named from the composer; mirrors the server's EntityRef. */
export const entityRefSchema = z.object({
  type: z.string(),
  id: z.string(),
  label: z.string().optional().default(""),
});

/** A file on a user turn: the document it became and enough to draw a chip. */
export const messageAttachmentSchema = z.object({
  documentId: z.string(),
  fileName: z.string(),
  contentType: z.string().optional(),
  fileSize: z.number().optional(),
});

/**
 * What the model thought before it answered. Only the readable part reaches
 * the panel; the provider's signed or encrypted continuation state stays on
 * the server's copy of the message.
 */
export const reasoningTraceSchema = z.object({
  text: z.string().optional().default(""),
});

/**
 * How a task handed to another agent ended. An ending this client has not
 * heard of reads as failed, which is the reading that claims the least.
 */
export const delegateStatusSchema = z.enum([
  "completed",
  "exhausted",
  "refused",
  "declined",
  "failed",
  "stopped",
]);

/**
 * The one record a write made or changed, as the app opens it: `entityType` is
 * a key of the record-link registry (`RECORD_LINKS`), `id` the record's id.
 */
export const recordRefSchema = z.object({
  entityType: z.string(),
  id: z.string(),
});

/**
 * What a write made, when its tool says: a past-tense verb, the kind in words,
 * the name, the ids, and — from a tool that names it — the record to link to.
 */
export const toolExecutionResultSchema = z.object({
  action: z.string().optional().default(""),
  kind: z.string().optional().default(""),
  name: z.string().optional().default(""),
  ids: z.preprocess((value) => value ?? {}, z.record(z.string(), z.string())),
  record: recordRefSchema.nullish().catch(null),
});

/** One write the other agent made or proposed on the task. */
export const delegateWriteSchema = z.object({
  toolName: z.string(),
  callId: z.string().optional().default(""),
  tier: autonomyTierSchema.optional().catch(undefined),
  summary: z.string().optional().default(""),
  result: toolExecutionResultSchema.nullish(),
  error: z.string().optional().default(""),
  simulated: z.boolean().optional().default(false),
});

/** Something the other agent kept beside the conversation. */
export const delegateDocumentSchema = z.object({
  id: z.string(),
  kind: z.string().optional().default(""),
  title: z.string().optional().default(""),
});

/**
 * The account of a task handed to another agent. The reader is shown it as
 * delegate_finished; the delegating agent reads the same object as the call's
 * result; and the call's saved result keeps it, bounded, as `delegateReport`.
 */
export const assistantDelegateFinishedEventSchema = z.object({
  delegateCallId: z.string(),
  agentId: z.string().optional().default(""),
  agentName: z.string().optional().default(""),
  /** The agent's mark; empty when the server did not say. */
  icon: z.string().optional().default(""),
  accent: z.string().optional().default(""),
  status: delegateStatusSchema.catch("failed"),
  reply: z.string().optional().default(""),
  reason: z.string().optional().default(""),
  made: nullableList(delegateWriteSchema),
  awaiting: nullableList(delegateWriteSchema),
  published: nullableList(delegateDocumentSchema),
  toolCallsUsed: z.number().int().nonnegative().optional().default(0),
  /** What a saved, bounded account left out of each list. */
  moreMade: z.number().int().nonnegative().optional().default(0),
  moreAwaiting: z.number().int().nonnegative().optional().default(0),
  morePublished: z.number().int().nonnegative().optional().default(0),
});

export const assistantMessageSchema = z.object({
  id: z.string(),
  threadId: z.string(),
  sequence: z.number(),
  role: messageRoleSchema,
  /**
   * Message for what a person or the model wrote; DecisionNote for the input
   * of the turn that follows a decision, which the thread shows as a note;
   * Delegated for a step another agent took on a task this conversation's
   * agent handed it, which the thread shows under the call that handed it.
   */
  kind: messageKindSchema.catch("Message").default("Message"),
  /** On a Delegated message: the agent that took the step. */
  agentId: z.string().nullish(),
  /** On a Delegated message: the delegate_task call it answers. */
  delegateCallId: z.string().nullish(),
  /** On a Delegated message: the agent's name as the thread is served. */
  agentName: z.string().nullish(),
  /**
   * On a Delegated message: the agent's mark as the thread is served. The
   * icon is empty for an agent with none of its own, which draws its initials.
   */
  agentIcon: z.string().nullish(),
  agentAccent: z.string().nullish(),
  /**
   * On a delegate_task call's result: the account of the task, as
   * delegate_finished delivered it, bounded. Absent from a conversation saved
   * before it was kept, whose account is read back out of `content`.
   */
  delegateReport: assistantDelegateFinishedEventSchema.nullish().catch(null),
  content: z.string().optional().default(""),
  toolCalls: z.array(toolCallRecordSchema).nullish(),
  toolCallId: z.string().optional().default(""),
  toolName: z.string().optional().default(""),
  toolFailed: z.boolean().default(false),
  /** On a tool result: what the call did. */
  effect: optionalToolEffect,
  /**
   * On a tool result: one line the server wrote about it, in English — a
   * record's name, a page, a report, or a count. Never set on a failed call.
   */
  summary: z.string().optional(),
  /** Which guard layer decided this turn, kept so a refusal can be explained. */
  scopeStage: z.string().optional().default(""),
  scopeCategory: z.string().optional().default(""),
  scopeReason: z.string().optional().default(""),
  refused: z.boolean().default(false),
  pageContext: pageContextSchema.nullish(),
  attachments: z.array(messageAttachmentSchema).nullish(),
  mentions: z.array(entityRefSchema).nullish(),
  model: z.string().optional().default(""),
  inputTokens: z.number().default(0),
  outputTokens: z.number().default(0),
  reasoning: reasoningTraceSchema.nullish(),
  /** How long the model took, and what the turn cost where the provider is priced. */
  latencyMs: z.number().nullish(),
  costUsd: z.union([z.string(), z.number()]).nullish(),
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
  /** The model this conversation is set to. Empty means the org's own order. */
  preferredProviderId: optionalIdSchema,
  origin: threadOriginSchema.default("Panel"),
  pinned: z.boolean().default(false),
  /** The record the conversation was opened from, when it was. */
  subjectType: z.string().optional().default(""),
  subjectId: optionalIdSchema,
  /**
   * Whether the reader may still ask this conversation's agent anything. False
   * once they lose access to the agent or it is disabled; the conversation
   * stays readable.
   */
  canContinue: z.boolean().default(true),
  version: z.number().default(0),
  createdAt: z.number(),
  updatedAt: z.number(),
});

/**
 * What a turn produced besides words: the rows a preview returned, a run to
 * download, the email an agent wants to send, the plan it wants to carry
 * out, the record it looked up. The payload is kind-specific and read by the
 * renderer for that kind. A draft or a plan is a view over the proposal or
 * plan that carries the decision.
 */
export const assistantArtifactSchema = z.object({
  id: z.string(),
  threadId: z.string(),
  messageId: optionalIdSchema,
  runId: optionalIdSchema,
  proposalId: optionalIdSchema,
  planId: optionalIdSchema,
  kind: artifactKindSchema,
  status: artifactStatusSchema,
  title: z.string(),
  payload: z.preprocess((value) => value ?? {}, z.record(z.string(), z.unknown())),
  /** The tool call that produced it, so the transcript can point at it. */
  sourceToolCallId: z.string().optional().default(""),
  pinned: z.boolean().default(false),
  createdAt: z.number(),
  updatedAt: z.number(),
});

export const assistantArtifactListSchema = z.object({
  results: z.array(assistantArtifactSchema),
});

/** An artifact as a streamed turn announces it, before the pane reads it whole. */
export const assistantArtifactEventSchema = z.object({
  id: z.string(),
  kind: artifactKindSchema,
  status: artifactStatusSchema,
  title: z.string(),
  sourceToolCallId: z.string().optional().default(""),
  /** Where a navigation artifact moves the app; empty for every other kind. */
  path: z.string().optional().default(""),
});

/**
 * One entry in the model picker.
 *
 * A provider record also holds the endpoint and an encrypted key; neither is
 * here, because this list is readable by anyone who may use the assistant
 * while the records themselves are not.
 */
export const assistantProviderOptionSchema = z.object({
  id: z.string(),
  name: z.string(),
  kind: z.string(),
  model: z.string(),
  trusted: z.boolean().default(false),
});

export const assistantProviderListSchema = z.object({
  results: z.array(assistantProviderOptionSchema).default([]),
});

export const assistantThreadListSchema = z.object({
  items: z.array(assistantThreadSchema),
  total: z.number().default(0),
});

/** What started a turn: the person, or the application reporting a decision. */
export const turnOriginSchema = z.enum(["Person", "DecisionFollowUp"]);

/**
 * A reply the person's assistant is still writing, in any of their
 * conversations. It keeps going whether or not anything is reading it, so
 * this is what the launcher and the conversation lists say is under way.
 */
export const assistantLiveTurnSchema = z.object({
  turnId: z.string(),
  threadId: z.string(),
  threadTitle: z
    .string()
    .nullish()
    .transform((value) => value ?? ""),
  origin: turnOriginSchema,
  /** Unix seconds. */
  startedAt: z.number(),
});

export const assistantLiveTurnListSchema = z.object({
  items: nullableList(assistantLiveTurnSchema),
});

/**
 * One page of a thread in reading order. `hasMore` says a page exists above
 * the first message here; `total` is the whole thread's length and `limit` is
 * where the server stops accepting turns, so the client can say so first.
 */
export const assistantMessagePageSchema = z.object({
  results: z.array(assistantMessageSchema),
  hasMore: z.boolean().default(false),
  total: z.number().int().nonnegative().default(0),
  limit: z.number().int().nonnegative().default(0),
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
  "Skipped",
  "Simulated",
]);

/** What a write would have changed, produced instead of the write for an agent in simulation. */
export const toolSimulationSchema = z.object({
  summary: z.string().optional().default(""),
  changes: nullableList(
    z.object({
      field: z.string(),
      from: z.string().optional().default(""),
      to: z.string().optional().default(""),
    }),
  ),
  previewed: z.boolean().default(false),
});

export const proposalDecisionSchema = z.enum(["Accepted", "Rejected", "Modified"]);

/** A plan is decided whole: there is no accepting it with changes. */
export const planDecisionSchema = z.enum(["Accepted", "Rejected"]);

export const planStatusSchema = z.enum([
  "Pending",
  "Approved",
  "Completed",
  "Failed",
  "Rejected",
  "Expired",
]);

/**
 * Which switch is holding a proposal: the organization-wide pause on the AI
 * Control overview, or the shadow switch on the agent that made it.
 */
export const proposalHoldReasonSchema = z.enum(["OrganizationPaused", "AgentShadow"]);

export const proposalHoldSchema = z.object({
  reason: proposalHoldReasonSchema,
  agentName: z.string().optional().default(""),
});

/** The control a parameter takes when a person edits it, from the tool's schema. */
export const proposalFieldKindSchema = z.enum([
  "Text",
  "Multiline",
  "Integer",
  "Number",
  "Boolean",
  "Choice",
  "List",
  "JSON",
]);

/**
 * One parameter of a proposal's tool as a person may edit it before
 * approving: its name and label, the control it takes, and the bounds the
 * tool's schema puts on it. Derived server-side from the schema, so the form
 * follows the tool rather than a hand-kept list.
 */
export const proposalFieldSchema = z.object({
  name: z.string(),
  label: z.string(),
  description: z.string().optional().default(""),
  kind: proposalFieldKindSchema,
  required: z.boolean().default(false),
  options: nullableList(z.string()),
  minimum: z.number().nullish(),
  maximum: z.number().nullish(),
  maxLength: z.number().int().nullish(),
});

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
  /** When a pending proposal stops being decidable; 0 for one made before expiry existed. */
  expiresAt: z.number().nullish().default(0),
  /**
   * Set while a shadow switch keeps the proposal from being decided. The server
   * refuses a decision while it is set, so the card shows the reason instead of
   * buttons. `agentName` is set when the switch is the agent's own.
   */
  hold: proposalHoldSchema.nullish(),
  /** Set when the proposal is one step of a plan; the plan is decided, not the step. */
  planId: optionalIdSchema,
  /** The step's position in its plan, from 1; 0 for a proposal outside any plan. */
  planStep: z.number().int().nonnegative().default(0),
  /** Set when the write was previewed instead of made, because the agent was in simulation. */
  simulatedAt: z.number().nullish(),
  simulation: toolSimulationSchema.nullish(),
  /** What a person may edit before approving; empty once decided. */
  fields: nullableList(proposalFieldSchema),
  /** The values the approver changed before approving, keyed by parameter. */
  modifications: z.record(z.string(), z.unknown()).nullish(),
  /**
   * The agent that proposed it: the conversation's own, or another agent it
   * handed a task to. Absent from a server that does not say.
   */
  agentId: z.string().nullish(),
  agentName: z.string().nullish(),
});

export const assistantProposalListSchema = z.object({
  results: z.array(assistantProposalSchema),
});

/**
 * Several writes one run asked for, decided together and run in the order the
 * agent asked. Its steps are the proposals that carry its id.
 */
export const assistantPlanSchema = z.object({
  id: z.string(),
  runId: z.string().optional().default(""),
  title: z.string(),
  summary: z.string().optional().default(""),
  status: planStatusSchema,
  stepCount: z.number().int().nonnegative(),
  completedSteps: z.number().int().nonnegative().default(0),
  /** The step whose write failed and stopped the plan, from 1. */
  failedStep: z.number().int().nullish(),
  failureError: z.string().optional().default(""),
  decidedAt: z.number().nullish(),
  /** When a pending plan stops being decidable. */
  expiresAt: z.number().nullish().default(0),
  hold: proposalHoldSchema.nullish(),
  createdAt: z.number(),
  /** The agent whose proposals the plan groups. */
  agentId: z.string().nullish(),
  agentName: z.string().nullish(),
});

export const assistantPlanListSchema = z.object({
  results: z.array(assistantPlanSchema),
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
  /** What the turn produced besides words, in the order it produced them. */
  artifacts: z.array(assistantArtifactSchema).nullish(),
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

export const assistantReasoningEventSchema = z.object({ text: z.string() });

/**
 * Set on an event of another agent's turn, on a task the conversation's agent
 * handed it: which agent, and the delegate_task call the event belongs under.
 * Both are absent on the conversation's own events.
 */
const delegateScopeShape = {
  agentId: z.string().optional(),
  delegateCallId: z.string().optional(),
};

export const assistantMessageEventSchema = z.object({
  content: z.string().optional().default(""),
  toolCalls: z.array(toolCallRecordSchema).nullish(),
  model: z.string().optional().default(""),
  ...delegateScopeShape,
});

export const assistantToolStartedEventSchema = z.object({
  callId: z.string(),
  name: z.string(),
  arguments: z.record(z.string(), z.unknown()).nullish(),
  effect: optionalToolEffect,
  ...delegateScopeShape,
});

export const assistantToolFinishedEventSchema = z.object({
  callId: z.string(),
  name: z.string(),
  failed: z.boolean().default(false),
  proposed: z.boolean().default(false),
  content: z.string().optional().default(""),
  effect: optionalToolEffect,
  summary: z.string().optional(),
  ...delegateScopeShape,
});

/** The conversation's agent handed a task to another agent. */
export const assistantDelegateStartedEventSchema = z.object({
  delegateCallId: z.string(),
  agentId: z.string(),
  agentName: z.string().optional().default(""),
  icon: z.string().optional().default(""),
  accent: z.string().optional().default(""),
  task: z.string().optional().default(""),
});

/** A piece of the other agent's reply or thinking, apart from the reply being shown. */
export const assistantDelegateTextEventSchema = z.object({
  agentId: z.string().optional().default(""),
  delegateCallId: z.string(),
  text: z.string(),
});

export const assistantErrorEventSchema = z.object({ message: z.string() });

/**
 * The model died partway through its reply and the turn is starting over,
 * on another model when one is configured. Whatever streamed before it is
 * withdrawn; what follows is the whole reply.
 */
/**
 * Why a turn is being retried: a reply that died partway starting over on
 * another provider, or a busy provider being asked again after a wait.
 */
export const retryKindSchema = z.enum(["restart", "busy"]);

export const assistantRetryingEventSchema = z.object({
  attempt: z.number().int().nonnegative(),
  provider: z.string().optional().default(""),
  reason: z.string().optional().default(""),
  kind: retryKindSchema.optional().default("restart"),
  /** How long the router is waiting before a busy retry, in seconds. */
  waitSeconds: z.number().int().nonnegative().optional().default(0),
});

/** The other agent's reply died partway and is starting over; the reply being shown is untouched. */
export const assistantDelegateRetryingEventSchema = assistantRetryingEventSchema.extend({
  agentId: z.string().optional().default(""),
  delegateCallId: z.string(),
});

export type AssistantStreamEvent =
  | { event: "accepted"; data: z.infer<typeof assistantAcceptedEventSchema> }
  | { event: "refused"; data: z.infer<typeof assistantRefusedEventSchema> }
  | { event: "delta"; data: z.infer<typeof assistantDeltaEventSchema> }
  | { event: "reasoning"; data: z.infer<typeof assistantReasoningEventSchema> }
  | { event: "message"; data: z.infer<typeof assistantMessageEventSchema> }
  | { event: "tool_started"; data: z.infer<typeof assistantToolStartedEventSchema> }
  | { event: "tool_finished"; data: z.infer<typeof assistantToolFinishedEventSchema> }
  | { event: "retrying"; data: z.infer<typeof assistantRetryingEventSchema> }
  | { event: "delegate_started"; data: z.infer<typeof assistantDelegateStartedEventSchema> }
  | { event: "delegate_delta"; data: z.infer<typeof assistantDelegateTextEventSchema> }
  | { event: "delegate_reasoning"; data: z.infer<typeof assistantDelegateTextEventSchema> }
  | { event: "delegate_retrying"; data: z.infer<typeof assistantDelegateRetryingEventSchema> }
  | { event: "delegate_finished"; data: z.infer<typeof assistantDelegateFinishedEventSchema> }
  | { event: "artifact"; data: z.infer<typeof assistantArtifactEventSchema> }
  | { event: "thread"; data: AssistantThread }
  /**
   * The saved turn, or null when the ending was rebuilt from the turn's record
   * because its stream could not supply one: the reader then reads the
   * conversation rather than trusting what it has on screen.
   */
  | { event: "done"; data: SendMessageResult | null }
  | { event: "error"; data: z.infer<typeof assistantErrorEventSchema> };

/**
 * An ending the server rebuilt from a turn's record (`replay: true`) rather
 * than forwarded from the turn: it names the turn and how it ended, not what
 * was said, so it is not a saved result and must not be parsed as one.
 */
function isReplayedEnding(data: unknown): boolean {
  return (
    typeof data === "object" && data !== null && (data as { replay?: unknown }).replay === true
  );
}

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
    case "reasoning":
      return { event, data: assistantReasoningEventSchema.parse(data) };
    case "message":
      return { event, data: assistantMessageEventSchema.parse(data) };
    case "tool_started":
      return { event, data: assistantToolStartedEventSchema.parse(data) };
    case "tool_finished":
      return { event, data: assistantToolFinishedEventSchema.parse(data) };
    case "retrying":
      return { event, data: assistantRetryingEventSchema.parse(data) };
    case "delegate_started":
      return { event, data: assistantDelegateStartedEventSchema.parse(data) };
    case "delegate_delta":
    case "delegate_reasoning":
      return { event, data: assistantDelegateTextEventSchema.parse(data) };
    case "delegate_retrying":
      return { event, data: assistantDelegateRetryingEventSchema.parse(data) };
    case "delegate_finished":
      return { event, data: assistantDelegateFinishedEventSchema.parse(data) };
    case "artifact":
      return { event, data: assistantArtifactEventSchema.parse(data) };
    case "thread":
      return { event, data: assistantThreadSchema.parse(data) };
    case "done":
      return {
        event,
        data: isReplayedEnding(data) ? null : sendMessageResultSchema.parse(data),
      };
    case "error":
      return { event, data: assistantErrorEventSchema.parse(data) };
    default:
      return null;
  }
}

export type AgentTemplateKind = z.infer<typeof agentTemplateKindSchema>;
export type AutonomyTier = z.infer<typeof autonomyTierSchema>;
export type ToolEffect = z.infer<typeof toolEffectSchema>;
export type TriggerMode = z.infer<typeof triggerModeSchema>;
export type OutputMode = z.infer<typeof outputModeSchema>;
export type ContextProvider = z.infer<typeof contextProviderSchema>;
export type AgentDefinition = z.infer<typeof agentDefinitionSchema>;
export type AgentTemplate = z.infer<typeof agentTemplateSchema>;
export type ToolCatalogEntry = z.infer<typeof toolCatalogEntrySchema>;
export type ToolTrust = z.infer<typeof toolTrustSchema>;
export type AgentBudgetStatus = z.infer<typeof agentBudgetStatusSchema>;
export type ToolSimulation = z.infer<typeof toolSimulationSchema>;
export type AgentEventDescriptor = z.infer<typeof agentEventDescriptorSchema>;
export type SaveAgentDefinitionRequest = z.infer<typeof saveAgentDefinitionRequestSchema>;
export type AssistantThread = z.infer<typeof assistantThreadSchema>;
export type ThreadOrigin = z.infer<typeof threadOriginSchema>;
export type TurnOrigin = z.infer<typeof turnOriginSchema>;
export type AssistantLiveTurn = z.infer<typeof assistantLiveTurnSchema>;
export type AssistantLiveTurnList = z.infer<typeof assistantLiveTurnListSchema>;
export type AssistantArtifact = z.infer<typeof assistantArtifactSchema>;
export type ArtifactKind = z.infer<typeof artifactKindSchema>;
export type ArtifactStatus = z.infer<typeof artifactStatusSchema>;
export type AssistantArtifactEvent = z.infer<typeof assistantArtifactEventSchema>;
export type AssistantProviderOption = z.infer<typeof assistantProviderOptionSchema>;
export type AssistantMessage = z.infer<typeof assistantMessageSchema>;
export type AssistantMessagePage = z.infer<typeof assistantMessagePageSchema>;
export type AssistantPageContext = z.infer<typeof pageContextSchema>;
export type AssistantPageView = z.infer<typeof pageViewSchema>;
export type AssistantPageViewFilter = z.infer<typeof pageViewFilterSchema>;
export type AssistantPageViewKpi = z.infer<typeof pageViewKpiSchema>;
export type AssistantEntityRef = z.infer<typeof entityRefSchema>;
export type AssistantMessageAttachment = z.infer<typeof messageAttachmentSchema>;
export type SendMessageResult = z.infer<typeof sendMessageResultSchema>;
export type AssistantProposal = z.infer<typeof assistantProposalSchema>;
export type ProposalStatus = z.infer<typeof proposalStatusSchema>;
export type ProposalDecision = z.infer<typeof proposalDecisionSchema>;
export type ProposalField = z.infer<typeof proposalFieldSchema>;
export type RetryKind = z.infer<typeof retryKindSchema>;
export type MessageKind = z.infer<typeof messageKindSchema>;
export type DelegateStatus = z.infer<typeof delegateStatusSchema>;
export type DelegateWrite = z.infer<typeof delegateWriteSchema>;
export type DelegateDocument = z.infer<typeof delegateDocumentSchema>;
export type DelegateReport = z.infer<typeof assistantDelegateFinishedEventSchema>;
export type ToolExecutionResult = z.infer<typeof toolExecutionResultSchema>;
export type RecordRef = z.infer<typeof recordRefSchema>;
export type ProposalFieldKind = z.infer<typeof proposalFieldKindSchema>;
export type ProposalHold = z.infer<typeof proposalHoldSchema>;
export type AssistantPlan = z.infer<typeof assistantPlanSchema>;
export type PlanStatus = z.infer<typeof planStatusSchema>;
export type PlanDecision = z.infer<typeof planDecisionSchema>;
