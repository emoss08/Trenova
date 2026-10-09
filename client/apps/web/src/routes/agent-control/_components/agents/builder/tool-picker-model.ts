import type { AgentEgressClass, AgentToolPolicy } from "@/lib/graphql/agent-safety";
import type { AutonomyTier, ToolCatalogEntry } from "@/types/assistant";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { groupToolsByResource, type ToolGroup } from "../tool-catalog";

/** Which tools the picker shows: every one, the reads, or the changes. */
export type KindFilter = "all" | "read" | "act";

export type PickerFilter = {
  query: string;
  /** One record's tools, or "all". */
  group: string;
  kind: KindFilter;
  /** Only the tools the agent already holds. */
  chosenOnly: boolean;
};

/** How much of a set of tools is chosen, for a tri-state checkbox. */
export type CheckState = "none" | "some" | "all";

export type PickerRow =
  | {
      kind: "group";
      key: string;
      resource: string;
      label: string;
      /** The tools the header's checkbox acts on: those shown under it. */
      tools: ToolCatalogEntry[];
      state: CheckState;
    }
  | { kind: "tool"; key: string; tool: ToolCatalogEntry };

function kindMatches(tool: ToolCatalogEntry, kind: KindFilter): boolean {
  return kind === "all" || (kind === "read" ? tool.kind === "query" : tool.kind === "action");
}

/**
 * The groups the picker shows under a filter. The search, the record, the kind and
 * "chosen only" narrow the tools and drop the groups they empty; each group's chosen
 * count stays over the whole group, so the rail keeps saying what the agent holds.
 */
export function pickerGroups(
  tools: readonly ToolCatalogEntry[],
  selected: readonly string[],
  filter: PickerFilter,
): ToolGroup[] {
  const chosen = new Set(selected);
  return groupToolsByResource(tools, selected, filter.query)
    .filter((group) => filter.group === "all" || group.resource === filter.group)
    .map((group) => ({
      ...group,
      tools: group.tools.filter(
        (tool) => kindMatches(tool, filter.kind) && (!filter.chosenOnly || chosen.has(tool.name)),
      ),
    }))
    .filter((group) => group.tools.length > 0);
}

export function checkState(names: readonly string[], chosen: ReadonlySet<string>): CheckState {
  let held = 0;
  for (const name of names) {
    if (chosen.has(name)) held += 1;
  }
  return held === 0 ? "none" : held === names.length ? "all" : "some";
}

/** The groups as one list, a header before each group's tools, for a virtual list. */
export function pickerRows(groups: readonly ToolGroup[], chosen: ReadonlySet<string>): PickerRow[] {
  const rows: PickerRow[] = [];
  for (const group of groups) {
    rows.push({
      kind: "group",
      key: `group:${group.resource}`,
      resource: group.resource,
      label: group.label,
      tools: group.tools,
      state: checkState(
        group.tools.map((tool) => tool.name),
        chosen,
      ),
    });
    for (const tool of group.tools) {
      rows.push({ kind: "tool", key: `tool:${tool.name}`, tool });
    }
  }
  return rows;
}

/**
 * The selection with many tools added or removed at once. A change tool newly added
 * starts at the tier `tierFor` gives it; one already held keeps its own. A removed tool
 * takes its tier with it, and tools not named are left exactly as they were.
 */
export function setTools(
  selected: readonly string[],
  tiers: Readonly<Record<string, AutonomyTier>>,
  tools: readonly ToolCatalogEntry[],
  on: boolean,
  tierFor: (tool: ToolCatalogEntry) => AutonomyTier,
): { selected: string[]; tiers: Record<string, AutonomyTier> } {
  const held = new Set(selected);
  if (on) {
    const nextSelected = [...selected];
    const nextTiers = { ...tiers };
    for (const tool of tools) {
      if (held.has(tool.name)) continue;
      held.add(tool.name);
      nextSelected.push(tool.name);
      if (tool.kind === "action") nextTiers[tool.name] = tierFor(tool);
    }
    return { selected: nextSelected, tiers: nextTiers };
  }

  const dropped = new Set(tools.map((tool) => tool.name));
  return {
    selected: selected.filter((name) => !dropped.has(name)),
    tiers: Object.fromEntries(Object.entries(tiers).filter(([name]) => !dropped.has(name))),
  };
}

/** How far each class of work reaches, nearest first. */
const EGRESS_ORDER: readonly AgentEgressClass[] = [
  "None",
  "Personal",
  "Internal",
  "CustomerVisible",
  "DriverVisible",
  "ExternalRecipient",
  "Money",
];

/** The furthest any of a tool's work reaches. */
export function furthest(rule: AgentToolPolicy | undefined): AgentEgressClass {
  if (!rule) return "None";
  return rule.egress.reduce<AgentEgressClass>(
    (far, egress) => (EGRESS_ORDER.indexOf(egress) > EGRESS_ORDER.indexOf(far) ? egress : far),
    "None",
  );
}

export function egressLabel(egress: AgentEgressClass, t: TranslateFn): string {
  switch (egress) {
    case "None":
      return t("Reads only");
    case "Personal":
      return t("Own records");
    case "Internal":
      return t("Internal");
    case "CustomerVisible":
      return t("Customer");
    case "DriverVisible":
      return t("Driver");
    case "ExternalRecipient":
      return t("Outside recipient");
    case "Money":
      return t("Money");
  }
}

export function tierName(tier: AutonomyTier, t: TranslateFn): string {
  switch (tier) {
    case "Propose":
      return t("Propose");
    case "ActWithApproval":
      return t("Ask first");
    case "AutoExecute":
      return t("Automatic");
  }
}
