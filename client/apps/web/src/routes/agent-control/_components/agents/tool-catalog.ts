import { describeToolCall } from "@/components/assistant/tool-presentation";
import type { AutonomyTier, ToolCatalogEntry } from "@/types/assistant";
import { toTitleCase } from "@trenova/shared/lib/utils";
import { tierWithin } from "./agent-form-schema";

export const TIER_ORDER: readonly AutonomyTier[] = ["Propose", "ActWithApproval", "AutoExecute"];

export const TIER_LABEL: Record<AutonomyTier, string> = {
  Propose: "Propose",
  ActWithApproval: "Ask first",
  AutoExecute: "Automatic",
};

/**
 * Resources whose title-cased name reads wrong. Title case turns an
 * abbreviation into a word — worker_pto becomes "Worker Pto" — and this is the
 * label an administrator reads while deciding what an agent may touch.
 */
const RESOURCE_LABELS: Record<string, string> = {
  worker_pto: "Worker time off",
  hazardous_material: "Hazardous materials",
  bqi: "Billing queue",
};

export function resourceLabel(resource: string): string {
  const key = resource || "general";
  return RESOURCE_LABELS[key] ?? toTitleCase(key.replace(/[_-]+/g, " "));
}

export type ToolGroup = {
  resource: string;
  label: string;
  tools: ToolCatalogEntry[];
  /** How many of the group's tools are chosen, for the rail. */
  chosen: number;
};

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

/** The effective tier of a change tool: its own, capped by the ceiling. */
export function effectiveTier(
  tool: string,
  tiers: Record<string, AutonomyTier>,
  ceiling: AutonomyTier,
): AutonomyTier {
  const own = tiers[tool] ?? ceiling;
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
    if (tool.kind === "query") {
      summary.reads += 1;
      continue;
    }
    summary.changes += 1;
    summary.byTier[effectiveTier(name, tiers, ceiling)] += 1;
  }

  return summary;
}

/** The selection with one tool added or removed, and its tier dropped with it. */
export function toggleTool(
  selected: readonly string[],
  tiers: Record<string, AutonomyTier>,
  name: string,
  on: boolean,
): { selected: string[]; tiers: Record<string, AutonomyTier> } {
  if (on) {
    return {
      selected: selected.includes(name) ? [...selected] : [...selected, name],
      tiers,
    };
  }
  const { [name]: _dropped, ...rest } = tiers;
  return { selected: selected.filter((entry) => entry !== name), tiers: rest };
}
