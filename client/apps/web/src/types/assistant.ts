import { optionalIdSchema } from "@trenova/shared/types/helpers";
import { z } from "zod";
import { pageDraftEditSchema, pageDraftSchema } from "./page-draft";
import { translate } from "@trenova/shared/i18n/runtime";

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
  "LoadEntryCheck",
  "ServiceFailureDesk",
  "InsightAnalyst",
  "EDIDesk",
  "FormulaAssistant",
  "BooksKeeper",
  "SettlementsClerk",
  "Receivables",
  "MasterDataSteward",
  "WorkforceCoordinator",
  "FuelTaxClerk",
  "ReportAnalyst",
]);

export const autonomyTierSchema = z.enum(["Propose", "ActWithApproval", "AutoExecute"]);

/**
 * The most sensitive fields an agent's tools may read. Internal withholds
 * amounts, pay and other Restricted fields; Restricted shows them. A run a
 * person is in never reads past that person's own access.
 */
export const dataAccessCeilingSchema = z.enum(["Internal", "Restricted"]);
export type DataAccessCeiling = z.infer<typeof dataAccessCeilingSchema>;

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

export const messageKindSchema = z.enum([
  "Message",
  "DecisionNote",
  "Delegated",
  "Schedule",
  "Compaction",
  "Handoff",
  "HandoffBrief",
  "Steer",
  "WorldChange",
  "WaitNote",
]);

/**
 * How full a conversation's context window is, in estimated tokens by part,
 * measured when its last turn or compaction ended.
 */
export const contextUsageSchema = z.object({
  /** The agent's instructions, pinned facts, memories and tool definitions. */
  instructions: z.number().default(0),
  /** What was said, and the summary of any compacted stretch. */
  messages: z.number().default(0),
  toolResults: z.number().default(0),
  /** The text of attached files, read through the document tools. */
  files: z.number().default(0),
  /** How much a compaction would summarize: everything before the latest two turns. */
  compactable: z.number().default(0),
  /** The model's context window. */
  window: z.number().default(0),
  model: z.string().optional().default(""),
  measuredAt: z.number().default(0),
});

export type ContextUsage = z.infer<typeof contextUsageSchema>;

/** On a compaction summary: what it stands in for. */
export const compactionRecordSchema = z.object({
  /** The conversation compacted itself on nearing a full context. */
  auto: z.boolean().default(false),
  /** How many earlier messages the summary replaces. */
  summarized: z.number().default(0),
  through: z.number().default(0),
  /** Context use, in tokens, before and after. */
  before: z.number().default(0),
  after: z.number().default(0),
  /** What stayed in full: "recent" for the latest turns, "approvals" for decisions still waiting. */
  kept: z
    .array(z.string())
    .nullish()
    .transform((value) => value ?? []),
});

export type CompactionRecord = z.infer<typeof compactionRecordSchema>;

export const threadStatusSchema = z.enum(["Active", "Archived"]);

/**
 * Where a conversation began. A quick question from the palette (Ask) is
 * not listed until the person keeps it. An import or formula conversation
 * belongs to its page and is never listed; every other origin is a
 * conversation from the start.
 */
export const threadOriginSchema = z.enum([
  "Panel",
  "Desk",
  "Ask",
  "Watchtower",
  "Briefing",
  "Import",
  "Formula",
]);

/**
 * Why a reader may no longer ask a conversation's agent anything: it was
 * removed, it was turned off, it now runs on its own rather than in
 * conversation, or the reader may no longer use it.
 */
export const cannotContinueReasonSchema = z.enum([
  "AgentDeleted",
  "AgentDisabled",
  "AgentNotConversational",
  "NoAccess",
]);

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
  "draft_edit",
  "decision_request",
  "extraction",
]);

/**
 * The views of what a lookup returned: a table, a record card, a report
 * preview. The server keeps one only when the reply points to it, so the
 * Desk shows them where the reply names them rather than opening the
 * workspace on them while the reply is still being written.
 */
export const LOOKUP_ARTIFACT_KINDS: ReadonlySet<string> = new Set([
  "table_view",
  "entity_card",
  "report_preview",
  "rate_explanation",
  "run_diff",
  "extraction",
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
  dataAccessCeiling: dataAccessCeilingSchema.default("Internal"),
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
  /** Tokens of recorded memory one prompt may carry; absent for the default. */
  memoryTokenBudget: z.number().int().nullish(),
  /** The agent no longer looks back over its work to keep what it learned. */
  learningOff: z.boolean().default(false),
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
  starterDataAccess: dataAccessCeilingSchema.default("Internal"),
  starterOutput: outputModeSchema,
  /** Runs a day the starter suggests; 0 is no cap. */
  starterDailyRunLimit: z.number().int().nonnegative().default(0),
  /** The starter asks to begin in shadow mode: its runs are recorded, not acted on. */
  starterShadow: z.boolean().default(false),
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

/**
 * How the runtime judged a tool call: it ran, was proposed or simulated, or
 * was turned away before it could run — not permitted, arguments not
 * accepted, a repeat, past the turn's budget — or it failed. A verdict this
 * client has not heard of reads as absent, which draws the plain failed state.
 */
export const toolVerdictSchema = z.enum([
  "ran",
  "proposed",
  "simulated",
  "denied",
  "invalid",
  "duplicate",
  "over_budget",
  "failed",
  "unknown",
]);

export type ToolVerdict = z.infer<typeof toolVerdictSchema>;

const optionalToolVerdict = toolVerdictSchema.optional().catch(undefined);

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

/**
 * How much recorded memory one prompt may carry, in approximate tokens, as the
 * server bounds it (`agentdefinition.MinMemoryTokenBudget` and its siblings).
 * An agent with no budget set uses the default.
 */
export const MEMORY_TOKEN_BUDGET = { min: 1000, max: 16000, default: 6000 } as const;

export const saveAgentDefinitionRequestSchema = z.object({
  name: z
    .string()
    .trim()
    .min(1, { error: () => translate("Name is required") }),
  description: z.string().optional().default(""),
  template: agentTemplateKindSchema.nullable().default(null),
  icon: z.string().optional().default(""),
  accent: z.string().optional().default(""),
  instructions: z
    .string()
    .max(20000, { error: () => translate("Instructions cannot be longer than 20000 characters") })
    .optional()
    .default(""),
  guardrails: z.array(z.string()).default([]),
  toolNames: z.array(z.string()).default([]),
  toolTiers: z.record(z.string(), autonomyTierSchema).default({}),
  autonomyCeiling: autonomyTierSchema,
  dataAccessCeiling: dataAccessCeilingSchema.default("Internal"),
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
  /** Tokens of recorded memory one prompt may carry; null for the default. */
  memoryTokenBudget: z
    .number()
    .int({ error: () => translate("Use a whole number of tokens") })
    .min(MEMORY_TOKEN_BUDGET.min, {
      error: () =>
        translate("Memory in the prompt must be at least {0} tokens", MEMORY_TOKEN_BUDGET.min),
    })
    .max(MEMORY_TOKEN_BUDGET.max, {
      error: () =>
        translate("Memory in the prompt can be at most {0} tokens", MEMORY_TOKEN_BUDGET.max),
    })
    .nullable()
    .default(null),
  /** The agent no longer looks back over its work to keep what it learned. */
  learningOff: z.boolean().default(false),
  contextProviders: z.array(contextProviderSchema).default([]),
  outputMode: outputModeSchema.default("Conversational"),
  preferredProviderId: optionalIdSchema,
  /**
   * The agents this one may hand a task to. Absent keeps the saved list; a
   * list, empty or not, replaces it.
   */
  delegateIds: z.array(z.string()).max(MAX_DELEGATES).optional(),
  /**
   * Who may use the agent, set in the same transaction as the save. Both
   * absent keeps who may use it; both present replaces it. Changing it needs
   * permission to update roles as well as agents.
   */
  accessMode: z.enum(["Everyone", "Roles"]).optional(),
  accessRoleIds: z.array(z.string()).optional(),
  version: z.number().default(0),
});

/** Why the agent took a step, in its own words: what it saw, why, and what it passed over. */
export const stepRationaleSchema = z.object({
  saw: z.string().optional().default(""),
  because: z.string().optional().default(""),
  insteadOf: z.string().optional().default(""),
});
export type StepRationale = z.infer<typeof stepRationaleSchema>;

export const toolCallRecordSchema = z.object({
  id: z.string(),
  name: z.string(),
  arguments: z.record(z.string(), z.unknown()).nullish(),
  effect: optionalToolEffect,
  why: stepRationaleSchema.nullish(),
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
  /** The unsaved work of the page a page-bound conversation belongs to. */
  draft: pageDraftSchema.nullish().catch(null),
});

/** The kinds of record the composer's @ search can be narrowed to. */
export const mentionSearchTypes = [
  "all",
  "shipment",
  "customer",
  "invoice",
  "worker",
  "carrier",
] as const;
export type MentionSearchType = (typeof mentionSearchTypes)[number];

/** A record the @ search offers. */
export const mentionCandidateSchema = z.object({
  type: z.string(),
  id: z.string(),
  label: z.string(),
  subtitle: z.string().optional().default(""),
});

export const mentionCandidateListSchema = z.object({
  results: z.array(mentionCandidateSchema).default([]),
});

export type MentionCandidateRecord = z.infer<typeof mentionCandidateSchema>;

/** The kinds of record a list can page through one at a time. */
export type MentionPageKind = "shipment" | "invoice" | "invoice_dispute";

/** One page of the records of one kind, and whether another follows. */
export const mentionPageSchema = z.object({
  results: z
    .array(mentionCandidateSchema)
    .nullish()
    .transform((results) => results ?? []),
  hasMore: z.boolean().default(false),
});

export type MentionPage = z.infer<typeof mentionPageSchema>;

/** Where a conversation's agent and its asker stand against their usage caps. */
export const threadBudgetSchema = z.object({
  agentName: z.string().optional().default(""),
  spentUsd: z.string().optional().default(""),
  limitUsd: z.string().optional().default(""),
  share: z.number().optional().default(0),
  near: z.boolean().optional().default(false),
  monthStart: z.number().optional().default(0),
  resetsAt: z.number().optional().default(0),
  runsToday: z.number().optional().default(0),
  dailyRunLimit: z.number().optional().default(0),
  budgetUsed: z.boolean().optional().default(false),
  dailyUsed: z.boolean().optional().default(false),
  dayResetsAt: z.number().optional().default(0),
  /** Who turned the agent off, and when; empty while it is on. */
  disabledBy: z.string().optional().default(""),
  disabledAt: z.number().optional().default(0),
  person: z
    .object({ used: z.number(), limit: z.number(), resetsAt: z.number() })
    .nullable()
    .optional()
    .default(null),
});

export type ThreadBudget = z.infer<typeof threadBudgetSchema>;

/** What the Desk's search palette looks through. */
export const deskSearchKinds = ["all", "chat", "msg", "art", "dec"] as const;
export type DeskSearchKind = (typeof deskSearchKinds)[number];

/** One conversation, message, artifact or decision the search palette found. */
export const deskSearchResultSchema = z.object({
  kind: z.enum(["chat", "msg", "art", "dec"]),
  id: z.string(),
  threadId: z.string(),
  agentId: z.string().optional().default(""),
  title: z.string().optional().default(""),
  threadTitle: z.string().optional().default(""),
  /** On an artifact result: what kind it is; absent when the server names one this client does not know. */
  artifactKind: artifactKindSchema.optional().catch(undefined),
  status: z.string().optional().default(""),
  at: z.number().optional().default(0),
});

export const deskSearchResultListSchema = z.object({
  results: z.array(deskSearchResultSchema).default([]),
});

export type DeskSearchResult = z.infer<typeof deskSearchResultSchema>;

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
  poorlyRead: z.boolean().optional(),
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
  /** On a write over many records: how many it was asked to change. */
  total: z.number().optional(),
  /** On a write over many records: each one that did not go through, and why. */
  failed: z
    .array(
      z.object({
        id: z.string(),
        label: z.string().optional().default(""),
        reason: z.string().optional().default(""),
      }),
    )
    .nullish(),
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
  /** The proposal the write was filed as, which names its card. */
  proposalId: z.string().optional().default(""),
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

/** One model a failed reply was asked of, and what happened. */
export const failedProviderSchema = z.object({
  name: z.string().optional().default(""),
  model: z.string().optional().default(""),
  vendor: z.string().optional().default(""),
  /** Overloaded, Timed out, Unavailable, Failed, or Not set up. */
  status: z.string().optional().default(""),
  detail: z.string().optional().default(""),
});

export type FailedProvider = z.infer<typeof failedProviderSchema>;

/** Why a saved reply did not finish. */
export const replyFailureSchema = z.object({
  kind: z.enum(["no_model", "interrupted", "stopped", "before_start"]).catch("before_start"),
  providers: z
    .array(failedProviderSchema)
    .nullish()
    .transform((value) => value ?? []),
});

export type ReplyFailure = z.infer<typeof replyFailureSchema>;

/** The model a reply was asked of first, when another one answered. */
export const providerFallbackSchema = z.object({
  providerId: z.string().optional().default(""),
  name: z.string().optional().default(""),
  model: z.string().optional().default(""),
  status: z.string().optional().default(""),
});

/** Another memory a note points to, as it reads now. */
export const memoryNoteLinkSchema = z.object({
  id: z.string(),
  content: z.string(),
  status: z.string(),
});

/**
 * A memory a reply used or saved, as the person reading the conversation may
 * see it. Scope is Organization, User (just them) or Role (their team).
 */
export const memoryNoteSchema = z.object({
  id: z.string(),
  content: z.string(),
  /** Instruction, Fact, Correction or Procedure. */
  kind: z.string().optional().default(""),
  scope: z.string(),
  roleId: z.string().nullish(),
  roleName: z.string().nullish(),
  /** Active, Paused, Retired once forgotten, Suggested while an offer waits, Dismissed once turned down. */
  status: z.string(),
  /** User, Agent, Decision, Feedback or Reflection: how it was recorded. */
  source: z.string().optional().default(""),
  sourceTitle: z.string().nullish(),
  createdAt: z.number(),
  version: z.number(),
  /** The reader may change, pause and forget it. */
  editable: z.boolean().default(false),
  /** Why it was kept, in the words of the agent that kept it. */
  reason: z.string().nullish(),
  /** The memory it replaced, when the reader can see it. */
  replaces: memoryNoteLinkSchema.nullish().catch(null),
  /** The newest memory that replaced it, when the reader can see it. */
  replacedBy: memoryNoteLinkSchema.nullish().catch(null),
});

/** A memory the turn kept through remember, or offered to keep when the person asked to be asked. */
export const savedMemorySchema = z.object({
  id: z.string(),
  callId: z.string().optional().default(""),
  pending: z.boolean().default(false),
});

/** A pinned artifact a hand-off carried: the copy in the new conversation and its source. */
export const handoffArtifactSchema = z.object({
  id: z.string(),
  sourceId: z.string().optional().default(""),
  title: z.string().optional().default(""),
  kind: z.string().optional().default(""),
});

/**
 * A conversation a person took to another agent, and what went with it. Set
 * on the Handoff card left where it was handed off and on the HandoffBrief
 * that opens the new conversation.
 */
export const handoffSchema = z.object({
  fromThreadId: z.string(),
  toThreadId: z.string(),
  fromAgentId: z.string().optional().default(""),
  fromAgentName: z.string().optional().default(""),
  toAgentId: z.string().optional().default(""),
  toAgentName: z.string().optional().default(""),
  summary: z.string().optional().default(""),
  facts: z
    .array(z.string())
    .nullish()
    .transform((value) => value ?? []),
  artifacts: z
    .array(handoffArtifactSchema)
    .nullish()
    .transform((value) => value ?? []),
  at: z.number().optional().default(0),
});

export type AssistantHandoff = z.infer<typeof handoffSchema>;

/** A record that changed elsewhere while a reply was being written. */
export const watchedRecordChangeSchema = z.object({
  recordId: z.string(),
  resource: z.string().optional().default(""),
  label: z.string().optional().default(""),
  action: z.string().optional().default("updated"),
  fields: z
    .array(z.string())
    .nullish()
    .transform((fields) => fields ?? []),
  actorType: z.string().optional().default(""),
  actorUserId: z.string().optional().default(""),
  at: z.number().optional().default(0),
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
   * agent handed it, which the thread shows under the call that handed it;
   * Schedule for a request the person scheduled, drawn as its schedule card.
   */
  kind: messageKindSchema.catch("Message").default("Message"),
  /** On a Schedule message: the schedule it made, which may since be deleted. */
  scheduleId: z.string().nullish(),
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
  /** On a Handoff or HandoffBrief message: what the hand-off carried. */
  handoff: handoffSchema.nullish().catch(null),
  content: z.string().optional().default(""),
  /** On a Compaction message: what the summary stands in for. */
  compaction: compactionRecordSchema.nullish().catch(null),
  /** On a WorldChange notice: the records that changed while the reply was written. */
  worldChanges: z.array(watchedRecordChangeSchema).nullish().catch(null),
  toolCalls: z.array(toolCallRecordSchema).nullish(),
  toolCallId: z.string().optional().default(""),
  toolName: z.string().optional().default(""),
  toolFailed: z.boolean().default(false),
  /** On a tool result: how the runtime judged the call; absent from a result saved before it was kept. */
  toolVerdict: optionalToolVerdict,
  /**
   * On a find_tools result: the person's other agents it named as holding what
   * this agent could not call, which the hand-off menu offers first.
   */
  handOffAgents: z.array(z.string()).nullish().catch(null),
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
  /** The model that wrote the reply. */
  providerId: z.string().nullish(),
  /** The reply broke off partway; what arrived is kept. */
  truncated: z.boolean().optional(),
  /** Another model answered because this one did not. */
  fallbackFrom: providerFallbackSchema.nullish().catch(null),
  /** Set on a reply that is only a closing note: why it did not finish. */
  failure: replyFailureSchema.nullish().catch(null),
  inputTokens: z.number().default(0),
  outputTokens: z.number().default(0),
  reasoning: reasoningTraceSchema.nullish(),
  /** How long the model took, and what the turn cost where the provider is priced. */
  latencyMs: z.number().nullish(),
  costUsd: z.union([z.string(), z.number()]).nullish(),
  /** On a turn's last reply: the memories the turn used, in the order it used them. */
  usedMemoryIds: z.array(z.string()).nullish().catch(null),
  /** On a turn's last reply: what the turn kept, or offered to keep. */
  savedMemories: z.array(savedMemorySchema).nullish().catch(null),
  /** Each of those memories as the reader may see them; one moved out of reach is left out. */
  memories: z.array(memoryNoteSchema).nullish().catch(null),
  createdAt: z.number(),
});

export const threadAttentionSchema = z.object({
  pendingDecisions: z.number().default(0),
  lastTurnFailed: z.boolean().default(false),
  unread: z.boolean().default(false),
});

export type ThreadAttention = z.infer<typeof threadAttentionSchema>;

/**
 * Where a case stands: worked out by the server from the record it is about,
 * its open waits and its snooze each time it is read.
 */
export const caseStateSchema = z.enum(["Working", "Waiting", "Snoozed", "Settled"]);
export const caseWaitingOnSchema = z.enum(["Carrier", "Customer", "Reply", "Event"]);
export const snoozeAnchorSchema = z.enum(["Time", "Appointment", "ETA"]);
export const caseSubjectTypeSchema = z.enum(["Shipment", "Invoice", "InvoiceDispute"]);

export const caseRecordSchema = z.object({
  type: caseSubjectTypeSchema,
  id: z.string(),
  /** The record's own number: a PRO, an invoice number. */
  label: z.string().optional().default(""),
  status: z.string(),
  closed: z.boolean().default(false),
  /** How it closed: Invoiced, Canceled, Paid, Voided, or the dispute's resolution. */
  closedAs: z.string().optional().default(""),
  /** The invoice a dispute is on, where the dispute is opened. */
  invoiceId: z.string().optional().default(""),
  customerId: z.string().optional().default(""),
  carrierIds: z
    .array(z.string())
    .nullish()
    .transform((ids) => ids ?? []),
});

export const caseSummarySchema = z.object({
  state: caseStateSchema,
  waitingOn: caseWaitingOnSchema.optional().catch(undefined),
  openWaits: z.number().default(0),
  nextWaitDue: z.number().nullish(),
  snoozedUntil: z.number().nullish(),
  snoozeAnchor: snoozeAnchorSchema.optional().catch(undefined),
  record: caseRecordSchema,
});

export const checklistItemStateSchema = z.enum(["Done", "Blocked", "Pending", "NotNeeded"]);

export const checklistItemSchema = z.object({
  key: z.string(),
  state: checklistItemStateSchema,
  at: z.number().nullish(),
  /** The checks behind a blocked item, which the Desk words. */
  codes: z
    .array(z.string())
    .nullish()
    .transform((codes) => codes ?? []),
  /** The records involved, shown as they are: a document type, a carrier. */
  names: z
    .array(z.string())
    .nullish()
    .transform((names) => names ?? []),
  count: z.number().optional().default(0),
  step: z.string().optional().default(""),
  /** Shown and ticked, but never keeps the record from being ready. */
  optional: z.boolean().optional().default(false),
  /** A step the organization added: its own name, button and request to the agent. */
  label: z.string().optional().default(""),
  stepLabel: z.string().optional().default(""),
  prompt: z.string().optional().default(""),
  /** A person ticks it on the case. */
  manual: z.boolean().optional().default(false),
  tickedBy: z.string().optional().default(""),
});

export const caseChecklistSchema = z.object({
  kind: z.enum(["ReadyToBill", "ReadyToClose"]),
  ready: z.boolean(),
  items: z.array(checklistItemSchema),
  next: z.string().optional().default(""),
});

export const casePartySchema = z.object({
  kind: z.enum(["Carrier", "Customer"]),
  id: z.string(),
  name: z.string(),
});

/**
 * Who can take a step: the conversation's agent, another agent the person
 * may use (handed the step from this conversation, its work shown in the
 * thread), or the person, on the page where it is done.
 */
export const stepAbilitySchema = z.object({
  via: z.enum(["Agent", "Ask", "Person"]).catch("Agent"),
  agentId: z.string().optional().default(""),
  agentName: z.string().optional().default(""),
});

export const caseViewSchema = z.object({
  summary: caseSummarySchema,
  /** By step; a step it leaves out is the conversation's agent's to try. */
  abilities: z
    .record(z.string(), stepAbilitySchema)
    .nullish()
    .transform((abilities) => abilities ?? {}),
  checklist: caseChecklistSchema.nullish(),
  parties: z
    .array(casePartySchema)
    .nullish()
    .transform((parties) => parties ?? []),
});

/** What a write to a case answers with. */
export const caseBindingSchema = z.object({
  threadId: z.string(),
  subjectType: z.string().optional().default(""),
  subjectId: optionalIdSchema,
  snoozedUntil: z.number().nullish(),
  snoozeAnchor: snoozeAnchorSchema.optional().catch(undefined),
  case: caseSummarySchema.optional(),
});

export type CaseState = z.infer<typeof caseStateSchema>;
export type CaseWaitingOn = z.infer<typeof caseWaitingOnSchema>;
export type SnoozeAnchor = z.infer<typeof snoozeAnchorSchema>;
export type CaseSubjectType = z.infer<typeof caseSubjectTypeSchema>;
export type CaseRecord = z.infer<typeof caseRecordSchema>;
export type CaseSummary = z.infer<typeof caseSummarySchema>;
export type ChecklistItem = z.infer<typeof checklistItemSchema>;
export type ChecklistItemState = z.infer<typeof checklistItemStateSchema>;
export type CaseChecklist = z.infer<typeof caseChecklistSchema>;
export type CaseParty = z.infer<typeof casePartySchema>;
export type CaseView = z.infer<typeof caseViewSchema>;
export type StepAbility = z.infer<typeof stepAbilitySchema>;
export type CaseBinding = z.infer<typeof caseBindingSchema>;

export const assistantThreadSchema = z.object({
  id: z.string(),
  businessUnitId: z.string(),
  organizationId: z.string(),
  userId: z.string(),
  agentDefinitionId: z.string(),
  title: z.string().optional().default(""),
  status: threadStatusSchema,
  lastMessageAt: z.number().default(0),
  /** When the owner last looked at the conversation. */
  lastReadAt: z.number().optional(),
  /** What the conversation is waiting on; only the thread list carries it. */
  attention: threadAttentionSchema.optional(),
  /** The model this conversation is set to. Empty means the org's own order. */
  preferredProviderId: optionalIdSchema,
  origin: threadOriginSchema.default("Panel"),
  pinned: z.boolean().default(false),
  /**
   * What the person pinned for the agents to keep in mind for the whole
   * conversation, in the order they pinned them.
   */
  pinnedFacts: z.array(z.string()).nullish(),
  /** The record the conversation was opened from, when it was. */
  subjectType: z.string().optional().default(""),
  subjectId: optionalIdSchema,
  /** A case's snooze: until when, and what it follows. */
  snoozedUntil: z.number().nullish(),
  snoozeAnchor: snoozeAnchorSchema.optional().catch(undefined),
  /** Where the case stands, when the conversation is about a case's record. */
  case: caseSummarySchema.optional(),
  /**
   * Whether the reader may still ask this conversation's agent anything. False
   * once they lose access to the agent or it is disabled; the conversation
   * stays readable.
   */
  canContinue: z.boolean().default(true),
  /**
   * Why `canContinue` is false; absent while it is true. A reason this build
   * does not know reads as absent, so the notice falls back to a plain one.
   */
  cannotContinueReason: cannotContinueReasonSchema.optional().catch(undefined),
  /**
   * When the conversation first read text written outside the organization,
   * such as the document an import is about. From then on every change the
   * agent proposes waits for a person.
   */
  taintedAt: z.number().nullish(),
  /** How full the context was after the last turn or compaction; absent before the first. */
  contextUsage: contextUsageSchema.nullish().catch(null),
  /** The conversation no longer compacts itself on nearing a full context. */
  autoCompactOff: z.boolean().optional(),

  /** The conversation this one was handed off from. */
  handedFromThreadId: z.string().nullish(),
  version: z.number().default(0),
  createdAt: z.number(),
  updatedAt: z.number(),
});

/**
 * The agent a page's conversation is with. System agents like the import and
 * formula assistants are left out of every chat picker, so the page is told
 * who it is talking to when it opens the conversation.
 */
export const pageAgentSchema = z.object({
  id: z.string(),
  name: z.string(),
  description: z.string().optional().default(""),
  template: agentTemplateKindSchema.nullish().catch(null),
  icon: z.string().optional().default(""),
  accent: z.string().optional().default(""),
  systemKey: z.string().optional().default(""),
  toolNames: z.preprocess((value) => value ?? [], z.array(z.string())),
  starters: z.preprocess(
    (value) => value ?? [],
    z.array(z.object({ label: z.string(), prompt: z.string() })),
  ),
});

/** An import or formula page's own conversation, and the agent it is with. */
export const pageThreadSchema = z.object({
  thread: assistantThreadSchema,
  agent: pageAgentSchema,
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
  /** The first artifact of the lineage this one is a later version of; empty for the first. */
  lineageId: optionalIdSchema,
  /** This artifact's version within its lineage, from 1. */
  lineageSeq: z.number().optional().default(1),
  /** The lineage's name in a link, the same for every version. */
  slug: z.string().optional().default(""),
  /** The question asked in the turn that made it. */
  turn: z.string().optional().default(""),
  createdAt: z.number(),
  updatedAt: z.number(),
});

export const assistantArtifactListSchema = z.object({
  results: z.array(assistantArtifactSchema),
});

/** How many of a conversation's lineages match, in all, pinned and per family. */
export const artifactCountsSchema = z.object({
  all: z.number().default(0),
  pinned: z.number().default(0),
  families: z.preprocess((value) => value ?? {}, z.record(z.string(), z.number())),
});

/** One page of a conversation's artifacts by lineage, every version of each. */
export const assistantArtifactPageSchema = z.object({
  results: z.array(assistantArtifactSchema),
  total: z.number().default(0),
  nextCursor: z.string().optional().default(""),
  counts: artifactCountsSchema.default({ all: 0, pinned: 0, families: {} }),
});

export const documentRewriteSchema = z.object({ text: z.string() });

/** An artifact as a streamed turn announces it, before the pane reads it whole. */
export const assistantArtifactEventSchema = z.object({
  id: z.string(),
  kind: artifactKindSchema,
  status: artifactStatusSchema,
  title: z.string(),
  sourceToolCallId: z.string().optional().default(""),
  /** Where a navigation artifact moves the app; empty for every other kind. */
  path: z.string().optional().default(""),
  /** The change a draft_edit artifact hands the page; absent for every other kind. */
  draft: pageDraftEditSchema.nullish().catch(null),
});

/**
 * An artifact the turn withdrew: a record card a later read of the same tool
 * folded into one table. The server has deleted it, so a reader drops it from
 * what the turn produced rather than leaving it beside the table that
 * replaced it.
 */
export const assistantArtifactRemovedEventSchema = z.object({
  id: z.string(),
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
  /** The company behind the endpoint, read from where it points; empty when nobody publishes it. */
  vendor: z.string().optional(),
  /** The reasoning effort the endpoint is asked for. */
  reasoning: z.string().optional(),
  /** The endpoint failed its last connection test. */
  unavailable: z.boolean().optional(),
});

export const assistantProviderListSchema = z.object({
  results: z.array(assistantProviderOptionSchema).default([]),
});

/**
 * One page of the person's conversations, pinned first and then newest
 * first. The cursor continues below the page's last row; empty when nothing
 * follows.
 */
export const assistantThreadPageSchema = z.object({
  items: z.array(assistantThreadSchema),
  nextCursor: z.string().default(""),
});

/**
 * What started a turn: the person, the application reporting a decision, a
 * request the person scheduled coming round, or the conversation being
 * compacted.
 */
export const turnOriginSchema = z.enum([
  "Person",
  "DecisionFollowUp",
  "Scheduled",
  "Compaction",
  "WaitResolved",
]);

/**
 * A request the person asked to have repeated in a conversation. `cadence` is
 * when, as the card shows it ("Every weekday · 7:30 AM"); the run times are
 * Unix seconds.
 */
/**
 * A message the person left for a conversation while its agent was working:
 * sent in order as each reply ends, or read into the reply under way at its
 * next step when it steers.
 */
export const queuedMessageSchema = z.object({
  id: z.string(),
  threadId: z.string(),
  content: z.string(),
  request: z
    .object({
      mentions: z
        .array(entityRefSchema)
        .nullish()
        .transform((mentions) => mentions ?? []),
      attachmentDocumentIds: z
        .array(z.string())
        .nullish()
        .transform((ids) => ids ?? []),
    })
    .nullish()
    .transform((request) => request ?? { mentions: [], attachmentDocumentIds: [] }),
  position: z.number(),
  steer: z.boolean().optional().default(false),
  version: z.number().optional().default(0),
  createdAt: z.number().optional().default(0),
});

export const agentWaitKindSchema = z.enum([
  "Time",
  "StopArrival",
  "StopDeparture",
  "Reply",
  "AppointmentNear",
  "FreeTimeEnding",
  "HOSDriveBelow",
]);

export const agentWaitStatusSchema = z.enum(["Waiting", "Met", "TimedOut", "Cancelled", "Failed"]);

/**
 * Work the agent parked until something happens: the conversation picks it up
 * as a new turn when the wait is met or runs out.
 */
export const agentWaitSchema = z.object({
  id: z.string(),
  kind: agentWaitKindSchema,
  description: z.string(),
  /** What the agent said it would do when the wait ends. */
  nextStep: z.string().optional().default(""),
  threadId: z.string().optional().default(""),
  status: agentWaitStatusSchema,
  dueAt: z.number().nullish(),
  expiresAt: z.number(),
  resolvedAt: z.number().nullish(),
  outcome: z.string().optional().default(""),
  resumedTurnId: z.string().optional().default(""),
  createdAt: z.number().optional().default(0),
});

export const agentWaitListSchema = z.object({
  items: z
    .array(agentWaitSchema)
    .nullish()
    .transform((items) => items ?? []),
});

export type AgentWait = z.infer<typeof agentWaitSchema>;
export type AgentWaitKind = z.infer<typeof agentWaitKindSchema>;
export type AgentWaitList = z.infer<typeof agentWaitListSchema>;

export const queuedMessageListSchema = z.object({
  items: z
    .array(queuedMessageSchema)
    .nullish()
    .transform((items) => items ?? []),
});

export type QueuedMessage = z.infer<typeof queuedMessageSchema>;
export type QueuedMessageList = z.infer<typeof queuedMessageListSchema>;

export const conversationScheduleSchema = z.object({
  id: z.string(),
  threadId: z.string(),
  userId: z.string(),
  prompt: z.string(),
  cadence: z.string(),
  cronExpression: z.string().optional().default(""),
  timezone: z.string().optional().default("UTC"),
  enabled: z.boolean(),
  lastRunAt: z.number().nullish(),
  nextRunAt: z.number().nullish(),
  lastTurnId: z.string().nullish(),
  createdAt: z.number(),
});

export const conversationScheduleListSchema = z.object({
  items: nullableList(conversationScheduleSchema),
  total: z.number().default(0),
});

/** A schedule just made, and the message that draws its card. */
export const createdScheduleSchema = z.object({
  schedule: conversationScheduleSchema,
  message: assistantMessageSchema,
});

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
  /** Approved, and waiting out the few seconds in which it can be undone. */
  "Approving",
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
  /** Approved, and waiting out the few seconds in which it can be undone. */
  "Approving",
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
  "RecordSubset",
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
  /**
   * The parameter that names the record the write is about. A change may
   * alter what is done to that record, never which record it is, so the
   * form shows it without letting it be edited; the server refuses a
   * retargeted approval either way.
   */
  readOnly: z.boolean().optional(),
  /**
   * For a RecordSubset field, the permission resource its ids belong to. A
   * person may drop ids from the proposed set but never add one; the server
   * refuses a widened set.
   */
  resource: z.string().nullish(),
  /**
   * For a RecordSubset field, every record the agent proposed, in the order
   * proposed, named by its label (its id when the record is gone or the
   * reader may not read it). Absent for every other kind.
   */
  choices: z.array(z.object({ id: z.string(), label: z.string() })).nullish(),
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
  /** What the run made; for a write over many records, the ones that did not go through. */
  executionResult: toolExecutionResultSchema.nullish().catch(null),
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
   * The values a person changed and saved but has not yet approved with, keyed
   * by parameter, such as the rewording of a drafted message. Kept on the
   * server so the edit survives a reload; absent once decided.
   */
  pendingModifications: z.record(z.string(), z.unknown()).nullish(),
  /**
   * The agent that proposed it: the conversation's own, or another agent it
   * handed a task to. Absent from a server that does not say.
   */
  agentId: z.string().nullish(),
  agentName: z.string().nullish(),
  /** When the agent proposed it, in Unix seconds; 0 from a server that does not say. */
  createdAt: z
    .number()
    .nullish()
    .transform((value) => value ?? 0),
  /** When a person decided it, in Unix seconds; absent while it waits. */
  decidedAt: z.number().nullish(),
  /** Who decided it; empty while it waits. */
  decidedByUserId: optionalIdSchema,
  /** What the person told the agent with the decision; empty when they said nothing. */
  decisionNote: z
    .string()
    .nullish()
    .transform((value) => value ?? ""),
});

export const assistantProposalListSchema = z.object({
  results: z.array(assistantProposalSchema),
});

/** What a pending proposal holds as changed once a person saved their edits. */
export const proposalEditsSchema = z.object({
  proposalId: z.string(),
  pendingModifications: z.record(z.string(), z.unknown()).nullish(),
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
  /** Who decided the plan; empty while it waits. */
  decidedByUserId: optionalIdSchema,
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
  why: stepRationaleSchema.nullish(),
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
  verdict: optionalToolVerdict,
  /** The person's other agents a find_tools answer named as holding what this agent lacks. */
  handOffAgents: z.array(z.string()).nullish().catch(null),
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

/** The usage cap that turned a question away, when one did. */
export const turnLimitSchema = z.object({
  kind: z.enum(["monthly_budget", "daily_runs", "person_allowance"]),
  used: z.string().optional().default(""),
  limit: z.string().optional().default(""),
  resetsAt: z.number().optional().default(0),
});

export const assistantErrorEventSchema = z.object({
  message: z.string(),
  /** no_model_answered when every model failed, no_provider when none is set up. */
  code: z.string().optional(),
  /** The models asked, when every one of them failed. */
  providers: z.array(failedProviderSchema).nullish(),
  /** Set when a usage cap turned the question away. */
  limit: turnLimitSchema.nullish().catch(null),
});

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
  /** How many times the model is asked in all, for "attempt 2 of 3". */
  maxAttempts: z.number().int().nonnegative().optional(),
});

/** The other agent's reply died partway and is starting over; the reply being shown is untouched. */
export const assistantDelegateRetryingEventSchema = assistantRetryingEventSchema.extend({
  agentId: z.string().optional().default(""),
  delegateCallId: z.string(),
});

/**
 * The reply the turn recorded, sent in place of the one that streamed when the
 * runtime corrected it afterwards (ids taken out, a reprinted table pointed to,
 * a code block left out). The model was not asked again, so it is no retry.
 */
export const assistantReplyReplacedEventSchema = z.object({
  text: z.string(),
  reason: z.string().optional().default(""),
});

/** The other agent's corrected reply; the reply being shown is untouched. */
export const assistantDelegateReplyReplacedEventSchema = assistantReplyReplacedEventSchema.extend({
  agentId: z.string().optional().default(""),
  delegateCallId: z.string(),
});

/** How full the conversation's context is, sent as a turn is saved. */
export const assistantContextEventSchema = z.object({
  threadId: z.string(),
  usage: contextUsageSchema,
  autoCompactOff: z.boolean().optional().default(false),
});

/**
 * Where a compaction stands. From a turn that set one off, it names the
 * compaction's own turn to follow; on that turn's stream it opens, and
 * finished or cancelled ends it.
 */
export const assistantCompactionEventSchema = z.object({
  turnId: z.string(),
  threadId: z.string(),
  auto: z.boolean().optional().default(false),
  before: z.number().optional().default(0),
  after: z.number().optional().default(0),
  /** The summary, once saved. */
  message: assistantMessageSchema.nullish(),
  usage: contextUsageSchema.nullish(),
  autoCompactOff: z.boolean().optional().default(false),
});

export type AssistantCompactionEvent = z.infer<typeof assistantCompactionEventSchema>;

/** Every memory the turn has used so far; each event repeats the whole list. */
export const assistantMemoryUsedEventSchema = z.object({
  ids: z
    .array(z.string())
    .nullish()
    .transform((ids) => ids ?? []),
});

/** Something the person said while the reply was being written, read at its next step. */
export const assistantSteeredEventSchema = z.object({
  id: z.string(),
  content: z.string(),
  mentions: z
    .array(entityRefSchema)
    .nullish()
    .transform((mentions) => mentions ?? []),
});

/** Records the reply was working with changed elsewhere while it worked. */
export const assistantWorldChangedEventSchema = z.object({
  changes: z
    .array(watchedRecordChangeSchema)
    .nullish()
    .transform((changes) => changes ?? []),
});

/** The turn the conversation's queue started once this one was saved. */
export const assistantNextTurnEventSchema = z.object({
  turnId: z.string(),
  threadId: z.string(),
  queuedId: z.string().optional().default(""),
  input: z.string().optional().default(""),
});

export type WatchedRecordChange = z.infer<typeof watchedRecordChangeSchema>;
export type AssistantSteeredEvent = z.infer<typeof assistantSteeredEventSchema>;
export type AssistantNextTurnEvent = z.infer<typeof assistantNextTurnEventSchema>;

export type AssistantStreamEvent =
  | { event: "accepted"; data: z.infer<typeof assistantAcceptedEventSchema> }
  | { event: "refused"; data: z.infer<typeof assistantRefusedEventSchema> }
  | { event: "delta"; data: z.infer<typeof assistantDeltaEventSchema> }
  | { event: "reasoning"; data: z.infer<typeof assistantReasoningEventSchema> }
  | { event: "message"; data: z.infer<typeof assistantMessageEventSchema> }
  | { event: "tool_started"; data: z.infer<typeof assistantToolStartedEventSchema> }
  | { event: "tool_finished"; data: z.infer<typeof assistantToolFinishedEventSchema> }
  | { event: "retrying"; data: z.infer<typeof assistantRetryingEventSchema> }
  | { event: "reply_replaced"; data: z.infer<typeof assistantReplyReplacedEventSchema> }
  | { event: "delegate_started"; data: z.infer<typeof assistantDelegateStartedEventSchema> }
  | { event: "delegate_delta"; data: z.infer<typeof assistantDelegateTextEventSchema> }
  | { event: "delegate_reasoning"; data: z.infer<typeof assistantDelegateTextEventSchema> }
  | { event: "delegate_retrying"; data: z.infer<typeof assistantDelegateRetryingEventSchema> }
  | {
      event: "delegate_reply_replaced";
      data: z.infer<typeof assistantDelegateReplyReplacedEventSchema>;
    }
  | { event: "delegate_finished"; data: z.infer<typeof assistantDelegateFinishedEventSchema> }
  | { event: "artifact"; data: z.infer<typeof assistantArtifactEventSchema> }
  | { event: "artifact_removed"; data: z.infer<typeof assistantArtifactRemovedEventSchema> }
  | { event: "memory_used"; data: z.infer<typeof assistantMemoryUsedEventSchema> }
  | { event: "memory_saved"; data: z.infer<typeof savedMemorySchema> }
  | { event: "thread"; data: AssistantThread }
  /**
   * The saved turn, or null when the ending was rebuilt from the turn's record
   * because its stream could not supply one: the reader then reads the
   * conversation rather than trusting what it has on screen.
   */
  | { event: "done"; data: SendMessageResult | null }
  | { event: "error"; data: z.infer<typeof assistantErrorEventSchema> }
  | { event: "context"; data: z.infer<typeof assistantContextEventSchema> }
  | { event: "compaction_started"; data: AssistantCompactionEvent }
  | { event: "compaction_finished"; data: AssistantCompactionEvent }
  | { event: "compaction_cancelled"; data: AssistantCompactionEvent }
  | { event: "steered"; data: AssistantSteeredEvent }
  | { event: "world_changed"; data: z.infer<typeof assistantWorldChangedEventSchema> }
  | { event: "next_turn"; data: AssistantNextTurnEvent };

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
    case "reply_replaced":
      return { event, data: assistantReplyReplacedEventSchema.parse(data) };
    case "delegate_started":
      return { event, data: assistantDelegateStartedEventSchema.parse(data) };
    case "delegate_delta":
    case "delegate_reasoning":
      return { event, data: assistantDelegateTextEventSchema.parse(data) };
    case "delegate_retrying":
      return { event, data: assistantDelegateRetryingEventSchema.parse(data) };
    case "delegate_reply_replaced":
      return { event, data: assistantDelegateReplyReplacedEventSchema.parse(data) };
    case "delegate_finished":
      return { event, data: assistantDelegateFinishedEventSchema.parse(data) };
    case "artifact":
      return { event, data: assistantArtifactEventSchema.parse(data) };
    case "artifact_removed":
      return { event, data: assistantArtifactRemovedEventSchema.parse(data) };
    case "memory_used":
      return { event, data: assistantMemoryUsedEventSchema.parse(data) };
    case "memory_saved":
      return { event, data: savedMemorySchema.parse(data) };
    case "thread":
      return { event, data: assistantThreadSchema.parse(data) };
    case "done":
      return {
        event,
        data: isReplayedEnding(data) ? null : sendMessageResultSchema.parse(data),
      };
    case "error":
      return { event, data: assistantErrorEventSchema.parse(data) };
    case "context":
      return { event, data: assistantContextEventSchema.parse(data) };
    case "compaction_started":
    case "compaction_finished":
    case "compaction_cancelled":
      return { event, data: assistantCompactionEventSchema.parse(data) };
    case "steered":
      return { event, data: assistantSteeredEventSchema.parse(data) };
    case "world_changed":
      return { event, data: assistantWorldChangedEventSchema.parse(data) };
    case "next_turn":
      return { event, data: assistantNextTurnEventSchema.parse(data) };
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
export type AssistantThreadPage = z.infer<typeof assistantThreadPageSchema>;
export type CannotContinueReason = z.infer<typeof cannotContinueReasonSchema>;
export type ThreadOrigin = z.infer<typeof threadOriginSchema>;
export type TurnOrigin = z.infer<typeof turnOriginSchema>;
export type ConversationSchedule = z.infer<typeof conversationScheduleSchema>;
export type ConversationScheduleList = z.infer<typeof conversationScheduleListSchema>;
export type CreatedSchedule = z.infer<typeof createdScheduleSchema>;
export type AssistantLiveTurn = z.infer<typeof assistantLiveTurnSchema>;
export type AssistantLiveTurnList = z.infer<typeof assistantLiveTurnListSchema>;
export type AssistantArtifact = z.infer<typeof assistantArtifactSchema>;
export type AssistantArtifactPage = z.infer<typeof assistantArtifactPageSchema>;
export type ArtifactKind = z.infer<typeof artifactKindSchema>;
export type ArtifactStatus = z.infer<typeof artifactStatusSchema>;
export type AssistantArtifactEvent = z.infer<typeof assistantArtifactEventSchema>;
export type PageAgent = z.infer<typeof pageAgentSchema>;
export type PageThread = z.infer<typeof pageThreadSchema>;
export type AssistantProviderOption = z.infer<typeof assistantProviderOptionSchema>;
export type AssistantMessage = z.infer<typeof assistantMessageSchema>;
export type MemoryNote = z.infer<typeof memoryNoteSchema>;
export type SavedMemory = z.infer<typeof savedMemorySchema>;
export type AssistantMessagePage = z.infer<typeof assistantMessagePageSchema>;
export type AssistantPageContext = z.infer<typeof pageContextSchema>;

/** Where a conversation is being had: the Desk, or the assistant over a page. */
export type AssistantSurface = "Desk" | "Assistant";
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

export const handoffResultSchema = z.object({
  thread: assistantThreadSchema,
  message: assistantMessageSchema.nullish(),
});

export type HandoffResult = z.infer<typeof handoffResultSchema>;
