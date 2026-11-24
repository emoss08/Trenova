import { describeToolCall } from "@/components/assistant/tool-presentation";
import { areaLabel, scopeAllows, toolArea, type AgentScopeLike } from "@/lib/agent-scope";
import type { AutonomyTier, ToolCatalogEntry } from "@/types/assistant";
import { resourceLabel } from "@/lib/resource-label";
import { tierWithin } from "./agent-form-schema";
import { defineLabels } from "@trenova/shared/i18n/labels";

export const TIER_ORDER: readonly AutonomyTier[] = ["Propose", "ActWithApproval", "AutoExecute"];

export const TIER_LABEL: Record<AutonomyTier, string> = defineLabels({
  Propose: "Propose",
  ActWithApproval: "Ask first",
  AutoExecute: "Automatic",
});

export type ToolGroup = {
  resource: string;
  label: string;
  tools: ToolCatalogEntry[];
  /** How many of the group's tools are chosen, for the rail. */
  chosen: number;
};

/**
 * The catalog split into the tools every agent holds and the ones an
 * administrator chooses. The server strips core tools from a saved selection,
 * and an extension turned on for every agent gives its tools to all of them,
 * so offering either as a choice would be a checkbox that does nothing.
 */
export function splitCoreTools(tools: readonly ToolCatalogEntry[]): {
  core: ToolCatalogEntry[];
  selectable: ToolCatalogEntry[];
} {
  const core: ToolCatalogEntry[] = [];
  const selectable: ToolCatalogEntry[] = [];
  for (const tool of tools) {
    (heldByEveryAgent(tool) ? core : selectable).push(tool);
  }
  return { core, selectable };
}

function heldByEveryAgent(tool: ToolCatalogEntry): boolean {
  return tool.core || tool.grantedToEveryAgent;
}

/**
 * The tools a scope gives an agent beyond those every agent holds, in catalog order: every one
 * its areas and capabilities allow, less those switched off.
 */
export function heldTools(
  scope: AgentScopeLike,
  tools: readonly ToolCatalogEntry[],
): ToolCatalogEntry[] {
  return tools.filter((tool) => !heldByEveryAgent(tool) && scopeAllows(scope, tool.name));
}

/** The areas a catalog's tools fall in, each once, alphabetical by what a person reads. */
export function areasOf(tools: readonly ToolCatalogEntry[]): string[] {
  return [...new Set(tools.map((tool) => toolArea(tool.name)))].sort((a, b) =>
    areaLabel(a).localeCompare(areaLabel(b)),
  );
}

/** The tier a tool runs at when no override is set: its own default, or the ceiling for a read. */
export function defaultTier(tool: ToolCatalogEntry | undefined): AutonomyTier | undefined {
  const tier = tool?.defaultAutonomyTier;
  return tier === "Propose" || tier === "ActWithApproval" || tier === "AutoExecute"
    ? tier
    : undefined;
}

export type ImpliedRead = {
  tool: ToolCatalogEntry;
  /** The chosen tools that take their arguments from it. */
  neededBy: ToolCatalogEntry[];
};

/**
 * The reads an agent holds because a chosen tool depends on them. The server
 * grants a read a held tool takes its arguments from — report_create_dashboard needs
 * report ids, which only report_list hands out — so the form shows it rather
 * than leaving an agent that looks unable to find one. Writes are never
 * granted this way, and a read chosen outright is not repeated.
 */
export function impliedReads(
  selected: readonly string[],
  tools: readonly ToolCatalogEntry[],
): ImpliedRead[] {
  const byName = new Map(tools.map((tool) => [tool.name, tool]));
  const chosen = new Set(selected);
  const implied = new Map<string, ImpliedRead>();

  for (const name of selected) {
    const tool = byName.get(name);
    if (!tool) {
      continue;
    }
    for (const prerequisite of tool.prerequisites) {
      const read = byName.get(prerequisite);
      if (!read || read.kind !== "query" || read.core || chosen.has(prerequisite)) {
        continue;
      }
      const entry = implied.get(prerequisite) ?? { tool: read, neededBy: [] };
      entry.neededBy.push(tool);
      implied.set(prerequisite, entry);
    }
  }

  return [...implied.values()];
}

/** The title an administrator reads for a tool, from the same words the chat uses. */
export function toolTitle(tool: ToolCatalogEntry): string {
  return describeToolCall(tool.name, null).title;
}

function matches(tool: ToolCatalogEntry, needle: string): boolean {
  if (needle === "") {
    return true;
  }
  return (
    tool.name.includes(needle) ||
    toolTitle(tool).toLowerCase().includes(needle) ||
    tool.description.toLowerCase().includes(needle) ||
    resourceLabel(tool.resource).toLowerCase().includes(needle)
  );
}

/**
 * The catalog grouped by what each tool touches, in alphabetical order of
 * group, reads before changes within a group. A query narrows the tools and
 * drops the groups it empties; the count of chosen tools on each group is
 * over the whole group, not the narrowed one, so the rail keeps saying what
 * the agent holds.
 */
export function groupToolsByResource(
  tools: readonly ToolCatalogEntry[],
  selected: readonly string[],
  query = "",
): ToolGroup[] {
  const needle = query.trim().toLowerCase();
  const chosen = new Set(selected);
  const byResource = new Map<string, { visible: ToolCatalogEntry[]; chosen: number }>();

  for (const tool of tools) {
    const key = tool.resource || "general";
    const group = byResource.get(key) ?? { visible: [], chosen: 0 };
    if (chosen.has(tool.name)) {
      group.chosen += 1;
    }
    if (matches(tool, needle)) {
      group.visible.push(tool);
    }
    byResource.set(key, group);
  }

  return [...byResource.entries()]
    .filter(([, group]) => group.visible.length > 0)
    .sort(([a], [b]) => resourceLabel(a).localeCompare(resourceLabel(b)))
    .map(([resource, group]) => ({
      resource,
      label: resourceLabel(resource),
      chosen: group.chosen,
      tools: [...group.visible].sort((a, b) =>
        a.kind === b.kind ? a.name.localeCompare(b.name) : a.kind === "query" ? -1 : 1,
      ),
    }));
}

export type SelectionSummary = {
  reads: number;
  changes: number;
  /** Change tools by the tier they will run at, after the ceiling. */
  byTier: Record<AutonomyTier, number>;
  /** Tools the agent names that the catalog no longer offers. */
  unknown: string[];
};

/**
 * The effective tier of a change tool: its override, else its default, else the ceiling, and
 * never past the ceiling.
 */
export function effectiveTier(
  tool: string,
  tiers: Record<string, AutonomyTier>,
  ceiling: AutonomyTier,
  fallback?: AutonomyTier,
): AutonomyTier {
  const own = tiers[tool] ?? fallback ?? ceiling;
  return tierWithin(own, ceiling) ? own : ceiling;
}

/**
 * What an agent's tools add up to, in the words the form shows above the
 * picker: how many reads, how many changes, and at what tiers the changes
 * would run.
 */
export function summarizeSelection(
  selected: readonly string[],
  tools: readonly ToolCatalogEntry[],
  tiers: Record<string, AutonomyTier>,
  ceiling: AutonomyTier,
): SelectionSummary {
  const byName = new Map(tools.map((tool) => [tool.name, tool]));
  const summary: SelectionSummary = {
    reads: 0,
    changes: 0,
    byTier: { Propose: 0, ActWithApproval: 0, AutoExecute: 0 },
    unknown: [],
  };

  for (const name of selected) {
    const tool = byName.get(name);
    if (!tool) {
      summary.unknown.push(name);
      continue;
    }
    if (heldByEveryAgent(tool)) {
      continue;
    }
    if (tool.kind === "query") {
      summary.reads += 1;
      continue;
    }
    summary.changes += 1;
    summary.byTier[effectiveTier(name, tiers, ceiling, defaultTier(tool))] += 1;
  }

  return summary;
}
