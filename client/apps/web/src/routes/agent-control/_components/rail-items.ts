import type { AIControlTab, RailView } from "../ai-control-tabs";
import { retrievalRailStatus, type RetrievalRailState } from "./retrieval/retrieval-model";
import type { AiAuditVerificationStatus } from "@trenova/graphql/generated/graphql";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatRowTotal, type RowTotal } from "@trenova/shared/lib/row-total";

export type {
  ActivityView,
  AuditView,
  QualityView,
  RailView,
  SafetyView,
} from "../ai-control-tabs";

export type RailItem = {
  tab: AIControlTab;
  /** A count or a short status shown under the label; empty for none. */
  status: string;
  /** The status calls for attention: something is waiting on a person. */
  attention: boolean;
  /** Views under the item, shown when it is the active one. */
  children: { view: RailView; label: string }[];
};

export type RailCounts = {
  providersEnabled: number;
  providersTotal: number;
  agentsEnabled: number;
  agentsTotal: number;
  pendingProposals: RowTotal;
  runsLast24h: RowTotal;
  memoriesActive: RowTotal;
  extensionsOn: number;
  extensionsTotal: number;
  /** Agents whose latest scored suite run regressed. */
  qualityRegressions: number;
  /** Where search by meaning stands; null until it has been read. */
  retrieval: RetrievalRailState | null;
  /** What the audit trail's last verification found; null until it has been checked or read. */
  auditVerification: AiAuditVerificationStatus | null;
};

export type RailPermissions = {
  agents: boolean;
  providers: boolean;
  extensions: boolean;
  runs: boolean;
  proposals: boolean;
  exceptions: boolean;
  memory: boolean;
  /** The index is the providers' work, so it is read under the right to read providers. */
  retrieval: boolean;
  /** Reading what agents may do on their own is reading agents. */
  safety: boolean;
  /** How well agents are doing is read under the golden set's right. */
  quality: boolean;
  /** The answers people rated down are read under the right to read agent feedback. */
  ratings: boolean;
  /** The audit trail has its own right; reading runs does not grant it. */
  audit: boolean;
};

/**
 * The rail's items in order, with what each one can say about itself
 * before it is opened: how many of the things are on, and whether any of
 * them is waiting on a person. An item the reader may not open is left out
 * rather than shown dead.
 */
export function buildRailItems(
  counts: RailCounts | undefined,
  permissions: RailPermissions,
  t: TranslateFn,
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
        ? counts.memoriesActive.count === 0
          ? t("Nothing recorded")
          : counts.memoriesActive.exact
            ? t("{0, plural, one {# active} other {# active}}", counts.memoriesActive.count)
            : t("About {0} active", formatRowTotal(counts.memoriesActive.count, false))
        : "",
      attention: false,
      children: [],
    });
  }

  if (permissions.retrieval) {
    const state = counts?.retrieval ?? null;
    const summary = state ? retrievalRailStatus(state, t) : { status: "", attention: false };
    items.push({
      tab: "retrieval",
      status: summary.status,
      attention: summary.attention,
      children: [],
    });
  }

  if (permissions.safety) {
    items.push({
      tab: "safety",
      status: "",
      attention: false,
      children: [
        { view: "rules", label: t("Tool rules") },
        { view: "agents", label: t("By agent") },
      ],
    });
  }

  if (permissions.quality) {
    const regressed = counts?.qualityRegressions ?? 0;
    items.push({
      tab: "quality",
      status:
        regressed > 0
          ? t("{0, plural, one {# agent regressed} other {# agents regressed}}", regressed)
          : "",
      attention: regressed > 0,
      children: [
        { view: "agents", label: t("Agents") },
        { view: "runs", label: t("Suite runs") },
        ...(permissions.ratings ? [{ view: "ratings" as const, label: t("Worst rated") }] : []),
        { view: "golden", label: t("Golden set") },
        { view: "extraction", label: t("Document extraction") },
      ],
    });
  }

  if (permissions.runs) {
    items.push({ tab: "analytics", status: "", attention: false, children: [] });
  }

  if (permissions.runs) {
    const pending = counts?.pendingProposals.count ?? 0;
    const pendingExact = counts?.pendingProposals.exact ?? true;
    const children: RailItem["children"] = [{ view: "runs", label: t("Runs") }];
    if (permissions.proposals) {
      children.push({
        view: "proposals",
        label:
          pending === 0
            ? t("Proposals")
            : pendingExact
              ? t("Proposals · {0}", pending)
              : t("Proposals · about {0}", formatRowTotal(pending, false)),
      });
      children.push({ view: "plans", label: t("Plans") });
    }
    children.push({ view: "evaluations", label: t("Evaluations") });
    if (permissions.exceptions) {
      children.push({ view: "exceptions", label: t("Exceptions") });
    }
    children.push({ view: "analyses", label: t("Analyses") });
    items.push({
      tab: "activity",
      status:
        pending > 0
          ? pendingExact
            ? t("{0, plural, one {# awaiting decision} other {# awaiting decision}}", pending)
            : t("About {0} awaiting decision", formatRowTotal(pending, false))
          : counts
            ? counts.runsLast24h.exact
              ? t("{0, plural, one {# run today} other {# runs today}}", counts.runsLast24h.count)
              : t("About {0} runs today", formatRowTotal(counts.runsLast24h.count, false))
            : "",
      attention: pending > 0 && permissions.proposals,
      children,
    });
  }

  if (permissions.audit) {
    const verification = counts?.auditVerification ?? null;
    items.push({
      tab: "audit",
      status:
        verification === "Mismatch"
          ? t("Failed verification")
          : verification === "KeyMissing"
            ? t("Signing key missing")
            : "",
      attention: verification === "Mismatch" || verification === "KeyMissing",
      children: [
        { view: "trail", label: t("Trail") },
        { view: "exports", label: t("Exports") },
      ],
    });
  }

  return items;
}

/**
 * The view to show under the active row: the one asked for when the row
 * offers it, the row's first otherwise, and none for a row without views.
 */
export function resolveRailView(
  item: RailItem | undefined,
  requested: RailView | null,
): RailView | null {
  if (!item || item.children.length === 0) {
    return null;
  }

  return item.children.some((child) => child.view === requested)
    ? requested
    : item.children[0].view;
}

export type TabCount = { text: string; tone?: "w" | "d" };

/**
 * What each tab says beside its name: how many agents and providers are on,
 * what is new or waiting, and whether search runs on words alone. Nothing
 * is shown for a tab with nothing to say.
 */
export function tabCounts(
  counts: RailCounts | undefined,
  t: TranslateFn,
): Partial<Record<AIControlTab, TabCount | null>> {
  if (!counts) {
    return {};
  }
  return {
    agents: { text: `${counts.agentsEnabled}/${counts.agentsTotal}` },
    providers:
      counts.providersTotal > 0
        ? { text: `${counts.providersEnabled}/${counts.providersTotal}` }
        : null,
    extensions: counts.extensionsOn > 0 ? { text: t("{0} on", counts.extensionsOn) } : null,
    memory:
      counts.memoriesActive.count === 0
        ? null
        : {
            text: counts.memoriesActive.exact
              ? formatRowTotal(counts.memoriesActive.count, true)
              : t("about {0}", formatRowTotal(counts.memoriesActive.count, false)),
          },
    retrieval:
      counts.retrieval && !counts.retrieval.available ? { text: t("words only"), tone: "w" } : null,
    quality:
      counts.qualityRegressions > 0
        ? { text: t("{0} regressed", counts.qualityRegressions), tone: "d" }
        : null,
    activity: {
      text: counts.pendingProposals.exact
        ? t("{0} waiting", counts.pendingProposals.count)
        : t("About {0} waiting", formatRowTotal(counts.pendingProposals.count, false)),
      tone: "w",
    },
  };
}
