import type { AgentDefinitionRow } from "@/lib/graphql/agent-definition";
import type { AgentFilter } from "../../ai-control-tabs";
import type { AutonomyTier, ToolCatalogEntry } from "@/types/assistant";
import { summarizeSelection } from "./tool-catalog";

export type RosterCounts = Record<AgentFilter, number>;

/** An agent is waiting when it is on and something it proposed waits on a person. */
export function isWaiting(agent: AgentDefinitionRow): boolean {
  return agent.enabled && agent.pendingProposals > 0;
}

export function matchesFilter(agent: AgentDefinitionRow, filter: AgentFilter): boolean {
  switch (filter) {
    case "waiting":
      return isWaiting(agent);
    case "shadow":
      return agent.shadowMode;
    case "off":
      return !agent.enabled;
    default:
      return true;
  }
}

/** How many agents each filter would show, before the search narrows them. */
export function rosterCounts(agents: readonly AgentDefinitionRow[]): RosterCounts {
  const counts: RosterCounts = { all: agents.length, waiting: 0, shadow: 0, off: 0 };
  for (const agent of agents) {
    if (isWaiting(agent)) counts.waiting += 1;
    if (agent.shadowMode) counts.shadow += 1;
    if (!agent.enabled) counts.off += 1;
  }
  return counts;
}

/** The agents a filter and a search leave, the search reading name and description. */
export function filterRoster(
  agents: readonly AgentDefinitionRow[],
  filter: AgentFilter,
  query: string,
): AgentDefinitionRow[] {
  const needle = query.trim().toLowerCase();
  return agents.filter(
    (agent) =>
      matchesFilter(agent, filter) &&
      (needle === "" || `${agent.name} ${agent.description}`.toLowerCase().includes(needle)),
  );
}

/**
 * An agent's chosen tools split three ways: what only reads, what proposes, and what
 * acts (with a person's approval or on its own), each change at the tier it would run at.
 */
export function tierSplit(
  agent: Pick<AgentDefinitionRow, "toolNames" | "toolTiers" | "autonomyCeiling">,
  catalog: readonly ToolCatalogEntry[],
): [number, number, number] {
  const summary = summarizeSelection(
    agent.toolNames,
    catalog,
    (agent.toolTiers ?? {}) as Record<string, AutonomyTier>,
    agent.autonomyCeiling as AutonomyTier,
  );
  return [
    summary.reads,
    summary.byTier.Propose,
    summary.byTier.ActWithApproval + summary.byTier.AutoExecute,
  ];
}

export type RowFact =
  | { kind: "schedule"; nextRunAt: number }
  | { kind: "event"; label: string; more: number }
  | { kind: "roles"; names: string[] }
  | null;

/**
 * The one fact a row's second line starts with: when a scheduled agent runs next, what
 * wakes an event agent, or which roles may ask a chat agent that is not open to everyone.
 */
export function rowFact(
  agent: AgentDefinitionRow,
  eventLabel: (kind: string) => string,
): RowFact {
  if (agent.triggerMode === "Scheduled" || agent.triggerMode === "Continuous") {
    return agent.nextRunAt ? { kind: "schedule", nextRunAt: agent.nextRunAt } : null;
  }
  if (agent.triggerMode === "Event") {
    const [first, ...rest] = agent.eventKinds;
    return first ? { kind: "event", label: eventLabel(first), more: rest.length } : null;
  }
  if (agent.accessMode === "Roles") {
    const names = agent.accessRoles.map((role) => role.name);
    return names.length ? { kind: "roles", names } : null;
  }
  return null;
}

export type AgentMode = "live" | "shadow" | "sim";

export function agentMode(agent: Pick<AgentDefinitionRow, "shadowMode" | "simulationMode">): AgentMode {
  if (agent.simulationMode) return "sim";
  if (agent.shadowMode) return "shadow";
  return "live";
}

/** The two switches a mode sets on an agent. */
export function modeFlags(mode: AgentMode): { shadowMode: boolean; simulationMode: boolean } {
  return { shadowMode: mode === "shadow", simulationMode: mode === "sim" };
}

export type LeadingStreak = { toolName: string; streak: number };

/**
 * The tool with the longest run of clean approvals among those the agent still holds;
 * the first by name breaks a tie, and nothing leads until a streak has begun.
 */
export function leadingStreak(
  trust: readonly { toolName: string; streak: number }[],
  held: readonly string[],
): LeadingStreak | null {
  const holding = new Set(held);
  let best: LeadingStreak | null = null;
  for (const entry of trust) {
    if (!holding.has(entry.toolName) || entry.streak <= 0) {
      continue;
    }
    if (
      !best ||
      entry.streak > best.streak ||
      (entry.streak === best.streak && entry.toolName < best.toolName)
    ) {
      best = { toolName: entry.toolName, streak: entry.streak };
    }
  }
  return best;
}
