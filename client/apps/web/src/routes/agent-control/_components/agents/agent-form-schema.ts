import type { AgentDefinitionRow } from "@/lib/graphql/agent-definition";
import {
  autonomyTierSchema,
  saveAgentDefinitionRequestSchema,
  type AutonomyTier,
  type SaveAgentDefinitionRequest,
} from "@/types/assistant";
import { z } from "zod";

const TIER_RANK: Record<AutonomyTier, number> = { Propose: 0, ActWithApproval: 1, AutoExecute: 2 };

export function tierWithin(tier: AutonomyTier, ceiling: AutonomyTier): boolean {
  return TIER_RANK[tier] <= TIER_RANK[ceiling];
}

/**
 * The save contract with the checks the server would make, so the panel
 * points at the field instead of showing a server error after submit.
 */
export const agentFormSchema = saveAgentDefinitionRequestSchema
  .extend({
    toolTiers: z.record(z.string(), autonomyTierSchema).default({}),
  })
  .superRefine((values, ctx) => {
    if (values.triggerMode === "Scheduled" && values.cronExpression.trim() === "") {
      ctx.addIssue({
        code: "custom",
        path: ["cronExpression"],
        message: "A schedule is required for a scheduled agent",
      });
    }
    if (values.triggerMode === "Event" && values.eventKinds.length === 0) {
      ctx.addIssue({
        code: "custom",
        path: ["eventKinds"],
        message: "Choose at least one event that starts this agent",
      });
    }
    if (values.triggerMode === "Continuous" && values.intervalSeconds < 60) {
      ctx.addIssue({
        code: "custom",
        path: ["intervalSeconds"],
        message: "A continuous agent runs at most once a minute",
      });
    }

    const selected = new Set(values.toolNames);
    for (const tool of Object.keys(values.toolDailyLimits)) {
      if (!selected.has(tool)) {
        ctx.addIssue({
          code: "custom",
          path: ["toolDailyLimits"],
          message: `${tool} has a daily limit but is not one of this agent's tools`,
        });
        break;
      }
    }
    for (const [tool, tier] of Object.entries(values.toolTiers)) {
      if (!selected.has(tool)) {
        ctx.addIssue({
          code: "custom",
          path: ["toolTiers"],
          message: `${tool} has a tier but is not one of this agent's tools`,
        });
        break;
      }
      if (!tierWithin(tier, values.autonomyCeiling)) {
        ctx.addIssue({
          code: "custom",
          path: ["toolTiers"],
          message: `${tool} cannot act above the agent's ceiling`,
        });
        break;
      }
    }
  });

export type AgentFormValues = z.infer<typeof agentFormSchema>;

export const agentFormDefaults: AgentFormValues = {
  name: "",
  description: "",
  template: null,
  icon: "",
  accent: "",
  instructions: "",
  guardrails: [],
  toolNames: [],
  toolTiers: {},
  autonomyCeiling: "Propose",
  enabled: true,
  shadowMode: false,
  decisionTimeoutSeconds: 86400,
  triggerMode: "Chat",
  cronExpression: "",
  cronTimezone: "",
  eventKinds: [],
  intervalSeconds: 0,
  endsAt: null,
  maxConcurrentRuns: 1,
  runTimeoutSeconds: 600,
  maxToolCalls: 12,
  monthlyBudgetUsd: null,
  dailyRunLimit: 0,
  toolDailyLimits: {},
  simulationMode: false,
  contextProviders: [],
  outputMode: "Conversational",
  preferredProviderId: "",
  version: 0,
};

/**
 * What goes over the wire: trigger fields that belong to other modes are
 * cleared so a switched agent does not carry a stale schedule, and tiers for
 * tools that were unselected go with them.
 */
export function toSaveRequest(values: AgentFormValues): SaveAgentDefinitionRequest {
  const selected = new Set(values.toolNames);
  const toolTiers = Object.fromEntries(
    Object.entries(values.toolTiers).filter(([tool]) => selected.has(tool)),
  );
  // A limit of zero is no limit, and a limit on a tool the agent no longer
  // holds is a leftover; neither goes over the wire.
  const toolDailyLimits = Object.fromEntries(
    Object.entries(values.toolDailyLimits).filter(
      ([tool, limit]) => selected.has(tool) && limit > 0,
    ),
  );
  const scheduled = values.triggerMode === "Scheduled";
  const event = values.triggerMode === "Event";
  const continuous = values.triggerMode === "Continuous";

  return {
    ...values,
    name: values.name.trim(),
    toolTiers,
    toolDailyLimits,
    cronExpression: scheduled ? values.cronExpression.trim() : "",
    cronTimezone: scheduled ? values.cronTimezone : "",
    eventKinds: event ? values.eventKinds : [],
    intervalSeconds: continuous ? values.intervalSeconds : 0,
    endsAt: continuous || scheduled ? values.endsAt : null,
  };
}

/** What the edit panel loads: the row's values plus what the panel header reads. */
export type AgentPanelRow = AgentFormValues & {
  id: string;
  updatedAt: number;
  systemKey: string;
};

function limitsOf(value: unknown): Record<string, number> {
  if (!value || typeof value !== "object") return {};
  const parsed = z.record(z.string(), z.number().int().nonnegative()).safeParse(value);
  return parsed.success ? parsed.data : {};
}

/** The server writes a decimal as a string; the form edits dollars as a number. */
function moneyOf(value: string | null | undefined): number | null {
  if (value === null || value === undefined || value === "") return null;
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : null;
}

function tiersOf(value: unknown): Record<string, AutonomyTier> {
  if (!value || typeof value !== "object") return {};
  const parsed = z.record(z.string(), autonomyTierSchema).safeParse(value);
  return parsed.success ? parsed.data : {};
}

export function toAgentPanelRow(agent: AgentDefinitionRow): AgentPanelRow {
  return {
    id: agent.id,
    updatedAt: agent.updatedAt,
    systemKey: agent.systemKey,
    name: agent.name,
    description: agent.description,
    template: agent.template ?? null,
    icon: agent.icon,
    accent: agent.accent,
    instructions: agent.instructions,
    guardrails: [...agent.guardrails],
    toolNames: [...agent.toolNames],
    toolTiers: tiersOf(agent.toolTiers),
    autonomyCeiling: agent.autonomyCeiling,
    enabled: agent.enabled,
    shadowMode: agent.shadowMode,
    decisionTimeoutSeconds: agent.decisionTimeoutSeconds,
    triggerMode: agent.triggerMode,
    cronExpression: agent.cronExpression,
    cronTimezone: agent.cronTimezone,
    eventKinds: [...agent.eventKinds],
    intervalSeconds: agent.intervalSeconds,
    endsAt: agent.endsAt ?? null,
    maxConcurrentRuns: agent.maxConcurrentRuns,
    runTimeoutSeconds: agent.runTimeoutSeconds,
    maxToolCalls: agent.maxToolCalls,
    monthlyBudgetUsd: moneyOf(agent.monthlyBudgetUsd),
    dailyRunLimit: agent.dailyRunLimit,
    toolDailyLimits: limitsOf(agent.toolDailyLimits),
    simulationMode: agent.simulationMode,
    contextProviders: [...agent.contextProviders],
    outputMode: agent.outputMode,
    preferredProviderId: agent.preferredProviderId,
    version: agent.version,
  };
}
