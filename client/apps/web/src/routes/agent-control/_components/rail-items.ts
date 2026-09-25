import type { AIControlTab, RailView } from "../ai-control-tabs";
import { retrievalRailStatus, type RetrievalRailState } from "./retrieval/retrieval-model";
import type { AiAuditVerificationStatus } from "@trenova/graphql/generated/graphql";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";

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
  pendingProposals: number;
  runsLast24h: number;
  memoriesActive: number;
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
        ? counts.memoriesActive === 0
          ? t("Nothing recorded")
          : t("{0, plural, one {# active} other {# active}}", counts.memoriesActive)
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
        ...(permissions.ratings
          ? [{ view: "ratings" as const, label: t("Worst-rated answers") }]
          : []),
        { view: "golden", label: t("Golden set") },
        { view: "settings", label: t("Settings") },
      ],
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
