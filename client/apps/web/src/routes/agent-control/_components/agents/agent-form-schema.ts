import type { AgentAccess } from "@/lib/graphql/agent-access";
import type { AgentDefinitionRow } from "@/lib/graphql/agent-definition";
import {
  MAX_DELEGATES,
  autonomyTierSchema,
  saveAgentDefinitionRequestSchema,
  type AutonomyTier,
  type SaveAgentDefinitionRequest,
} from "@/types/assistant";
import { z } from "zod";
import { canDelegate, savedDelegates, type DelegateSummary } from "./delegates";

const TIER_RANK: Record<AutonomyTier, number> = { Propose: 0, ActWithApproval: 1, AutoExecute: 2 };

/** The most roles one request may grant an agent, as the server allows. */
export const MAX_ACCESS_ROLES = 100;

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
    /**
     * A limit box left empty is no limit: the field holds nothing for it, and
     * only a positive limit on a tool the agent holds is saved.
     */
    toolDailyLimits: z.record(z.string(), z.number().int().min(0).max(10000).nullish()).default({}),
    /** The agents this one may hand work to; the form always sends the whole list. */
    delegateIds: z
      .array(z.string())
      .max(MAX_DELEGATES, `An agent can ask at most ${MAX_DELEGATES} other agents`)
      .default([]),
    /**
     * Who may use it. Saved with the agent, but only when asked:
     * `toSaveRequest` leaves both out unless it is handed the access to send.
     */
    accessMode: z.enum(["Everyone", "Roles"]).default("Everyone"),
    /** The roles granted it, kept whatever the mode. */
    accessRoleIds: z
      .array(z.string())
      .max(MAX_ACCESS_ROLES, `An agent can be granted to at most ${MAX_ACCESS_ROLES} roles at once`)
      .default([]),
  })
  .superRefine((values, ctx) => {
    const seen = new Set<string>();
    for (const [index, id] of values.delegateIds.entries()) {
      if (seen.has(id)) {
        ctx.addIssue({
          code: "custom",
          path: ["delegateIds", index],
          message: "This agent is already on the list",
        });
      }
      seen.add(id);
    }

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

    // A limit or tier left behind by a tool the agent no longer holds has no
    // box to clear it from, so it is not an error: the save drops it.
    const selected = new Set(values.toolNames);
    for (const [tool, tier] of Object.entries(values.toolTiers)) {
      if (selected.has(tool) && !tierWithin(tier, values.autonomyCeiling)) {
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

/** A daily limit that caps anything: an empty box or a zero is no limit. */
function hasLimit(limit: number | null | undefined): limit is number {
  return typeof limit === "number" && limit > 0;
}

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
  memoryTokenBudget: null,
  contextProviders: [],
  outputMode: "Conversational",
  preferredProviderId: "",
  delegateIds: [],
  accessMode: "Everyone",
  accessRoleIds: [],
  version: 0,
};

/** Who may use the agent, as the form holds it. */
export function accessOf(
  values: Pick<AgentFormValues, "accessMode" | "accessRoleIds">,
): AgentAccess {
  return { mode: values.accessMode, roleIds: [...new Set(values.accessRoleIds)] };
}

/**
 * What goes over the wire: trigger fields that belong to other modes are
 * cleared so a switched agent does not carry a stale schedule, and tiers for
 * tools that were unselected go with them.
 *
 * Who may use the agent rides along only when `access` is given, and then in
 * the same transaction as the rest. Left out, the server keeps it as it is,
 * which is what a toggle or a prompt preview wants, and what a save that did
 * not change it wants too: re-sending it would need permission to update
 * roles, and would put back roles someone else changed in the meantime.
 */
export function toSaveRequest(
  values: AgentFormValues & { delegates?: unknown; accessRoles?: unknown },
  access?: AgentAccess,
): SaveAgentDefinitionRequest {
  // The allowlist's names and marks are for drawing it; only the ids are saved.
  const {
    delegates: _drawn,
    accessRoles: _granted,
    accessMode: _mode,
    accessRoleIds: _roleIds,
    ...form
  } = values;
  const selected = new Set(form.toolNames);
  const toolTiers = Object.fromEntries(
    Object.entries(form.toolTiers).filter(([tool]) => selected.has(tool)),
  );
  // A limit of zero is no limit, and a limit on a tool the agent no longer
  // holds is a leftover; neither goes over the wire.
  const toolDailyLimits = Object.fromEntries(
    Object.entries(form.toolDailyLimits).filter(
      (entry): entry is [string, number] => selected.has(entry[0]) && hasLimit(entry[1]),
    ),
  );
  const scheduled = form.triggerMode === "Scheduled";
  const event = form.triggerMode === "Event";
  const continuous = form.triggerMode === "Continuous";

  return {
    ...form,
    name: form.name.trim(),
    toolTiers,
    toolDailyLimits,
    cronExpression: scheduled ? form.cronExpression.trim() : "",
    cronTimezone: scheduled ? form.cronTimezone : "",
    eventKinds: event ? form.eventKinds : [],
    intervalSeconds: continuous ? form.intervalSeconds : 0,
    endsAt: continuous || scheduled ? form.endsAt : null,
    // Only an agent people talk to may ask others; the server refuses the
    // list on any other trigger, so a switched agent sends it empty.
    delegateIds: canDelegate(form.triggerMode) ? [...new Set(form.delegateIds)] : [],
    ...(access ? { accessMode: access.mode, accessRoleIds: [...new Set(access.roleIds)] } : {}),
  };
}

/** What the edit panel loads: the row's values plus what the panel header reads. */
export type AgentPanelRow = AgentFormValues & {
  id: string;
  updatedAt: number;
  systemKey: string;
  /** The allowlist as saved, with each agent's name and mark, for drawing it. */
  delegates: DelegateSummary[];
  /** The roles granted it as saved, by name, for drawing them. */
  accessRoles: { id: string; name: string }[];
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
  const delegates = savedDelegates(agent);

  const accessRoles = agent.accessRoles.map((role) => ({ id: role.id, name: role.name }));

  return {
    delegates,
    delegateIds: delegates.map((delegate) => delegate.id),
    accessMode: agent.accessMode,
    accessRoleIds: accessRoles.map((role) => role.id),
    accessRoles,
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
    memoryTokenBudget: agent.memoryTokenBudget ?? null,
    contextProviders: [...agent.contextProviders],
    outputMode: agent.outputMode,
    preferredProviderId: agent.preferredProviderId,
    version: agent.version,
  };
}
