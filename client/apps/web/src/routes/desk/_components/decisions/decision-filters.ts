import { humanizeToolName } from "@/components/assistant/proposal-state";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { parseAsString } from "nuqs";
import type { PendingDecisionFilter } from "./use-pending-decisions";

/** The group a plan is filed under, beside the groups named after one kind of write. */
export const PLAN_GROUP = "plan";

/** A kind of change in the reader's words: what the write does, or "Plans". */
export function changeLabel(t: TranslateFn, tool: string): string {
  return tool === PLAN_GROUP ? t("Plans") : humanizeToolName(tool);
}

/**
 * What the Decisions lens keeps in the address beside its lens: the words
 * searched for, which are the page's and stay from lens to lens (`q`, as
 * the other lenses keep them), one agent and one kind of change.
 */
export const DECISION_ADDRESS = {
  q: parseAsString.withDefault(""),
  agent: parseAsString.withDefault(""),
  change: parseAsString.withDefault(""),
};

/** What the queue is narrowed to; an empty value is every agent, every kind of change, any words. */
export type DecisionFilters = { search: string; agentId: string; tool: string };

export const NO_DECISION_FILTERS: DecisionFilters = { search: "", agentId: "", tool: "" };

export function decisionFiltersFromAddress(address: {
  q: string;
  agent: string;
  change: string;
}): DecisionFilters {
  return {
    search: address.q.trim(),
    agentId: address.agent.trim(),
    tool: address.change.trim(),
  };
}

/** The address the filters are kept as; a filter that is off leaves the address. */
export function decisionAddressOf(filters: DecisionFilters) {
  return {
    q: filters.search.trim() || null,
    agent: filters.agentId || null,
    change: filters.tool || null,
  };
}

/** How many filters are on. The search is not one: it has its own box. */
export function decisionFilterCount(filters: DecisionFilters): number {
  return (filters.agentId === "" ? 0 : 1) + (filters.tool === "" ? 0 : 1);
}

/** Whether the queue shows less than what waits. */
export function isDecisionNarrowed(filters: DecisionFilters): boolean {
  return filters.search !== "" || decisionFilterCount(filters) > 0;
}

/** What the queue is asked for. */
export function pendingFilter(filters: DecisionFilters): PendingDecisionFilter {
  return {
    agentDefinitionId: filters.agentId || undefined,
    toolName: filters.tool || undefined,
    search: filters.search || undefined,
  };
}

/** Who asks and what for, each with how much of it waits: what the filter offers. */
export type DecisionFacets = {
  agents: readonly { id: string; name: string; count: number }[];
  tools: readonly { tool: string; count: number }[];
};

export const NO_DECISION_FACETS: DecisionFacets = { agents: [], tools: [] };

export type DecisionFilterToken =
  | { key: "agent"; type: "agent"; label: string }
  | { key: "change"; type: "change"; label: string };

/**
 * One token per filter that is on. An agent with nothing waiting is no longer
 * in the summary, so it is named as "One agent" rather than by its id.
 */
export function decisionFilterTokens(
  t: TranslateFn,
  filters: DecisionFilters,
  facets: DecisionFacets,
): DecisionFilterToken[] {
  const tokens: DecisionFilterToken[] = [];
  if (filters.agentId !== "") {
    tokens.push({
      key: "agent",
      type: "agent",
      label: facets.agents.find((agent) => agent.id === filters.agentId)?.name || t("One agent"),
    });
  }
  if (filters.tool !== "") {
    tokens.push({ key: "change", type: "change", label: changeLabel(t, filters.tool) });
  }

  return tokens;
}
