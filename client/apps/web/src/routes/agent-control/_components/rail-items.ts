import type { AIControlTab } from "../ai-control-tabs";

export type ActivityView = "runs" | "proposals" | "plans" | "evaluations" | "exceptions";

export type RailItem = {
  tab: AIControlTab;
  /** A count or a short status shown under the label; empty for none. */
  status: string;
  /** The status calls for attention: something is waiting on a person. */
  attention: boolean;
  /** Views under the item, shown when it is the active one. */
  children: { view: ActivityView; label: string }[];
};

export type RailCounts = {
  providersEnabled: number;
  providersTotal: number;
  agentsEnabled: number;
  agentsTotal: number;
  pendingProposals: number;
  runsLast24h: number;
  memoriesActive: number;
  extensionsOn: number;
  extensionsTotal: number;
};

export type RailPermissions = {
  agents: boolean;
  providers: boolean;
  extensions: boolean;
  runs: boolean;
  proposals: boolean;
  exceptions: boolean;
  memory: boolean;
};

type Translate = (text: string, ...args: (string | number)[]) => string;

/**
 * The rail's items in order, with what each one can say about itself
 * before it is opened: how many of the things are on, and whether any of
 * them is waiting on a person. An item the reader may not open is left out
 * rather than shown dead.
 */
export function buildRailItems(
  counts: RailCounts | undefined,
  permissions: RailPermissions,
  t: Translate,
): RailItem[] {
  const items: RailItem[] = [{ tab: "overview", status: "", attention: false, children: [] }];

  if (permissions.agents) {
    items.push({
      tab: "agents",
      status: counts ? t("{0} of {1} on", counts.agentsEnabled, counts.agentsTotal) : "",
      attention: false,
      children: [],
    });
  }

  if (permissions.providers) {
    items.push({
      tab: "providers",
      status: counts
        ? counts.providersTotal === 0
          ? t("None connected")
          : t("{0} of {1} on", counts.providersEnabled, counts.providersTotal)
        : "",
      attention: counts !== undefined && counts.providersEnabled === 0,
      children: [],
    });
  }

  if (permissions.extensions) {
    items.push({
      tab: "extensions",
      status: counts
        ? counts.extensionsOn === 0
          ? t("None on")
          : t("{0} of {1} on", counts.extensionsOn, counts.extensionsTotal)
        : "",
      attention: false,
      children: [],
    });
  }

  if (permissions.memory) {
    items.push({
      tab: "memory",
      status: counts
        ? counts.memoriesActive === 0
          ? t("Nothing recorded")
          : t("{0, plural, one {# active} other {# active}}", counts.memoriesActive)
        : "",
      attention: false,
      children: [],
    });
  }

  if (permissions.runs) {
    const pending = counts?.pendingProposals ?? 0;
    const children: RailItem["children"] = [{ view: "runs", label: t("Runs") }];
    if (permissions.proposals) {
      children.push({ view: "proposals", label: t("Proposals") });
      children.push({ view: "plans", label: t("Plans") });
    }
    children.push({ view: "evaluations", label: t("Evaluations") });
    if (permissions.exceptions) {
      children.push({ view: "exceptions", label: t("Exceptions") });
    }
    items.push({
      tab: "activity",
      status:
        pending > 0
          ? t("{0, plural, one {# awaiting decision} other {# awaiting decision}}", pending)
          : counts
            ? t("{0, plural, one {# run today} other {# runs today}}", counts.runsLast24h)
            : "",
      attention: pending > 0 && permissions.proposals,
      children,
    });
  }

  return items;
}
