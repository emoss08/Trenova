import { useAttentionSummary } from "@/hooks/use-attention";
import { useUserTimezone } from "@/hooks/use-user-timezone";
import {
  fetchWorkerRosterAttention,
  WORKER_ROSTER_ATTENTION_KEY,
} from "@/lib/graphql/worker-overview";
import { ar } from "@/lib/queries/ar";
import { shipmentBoard } from "@/lib/queries/shipment-board";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatCompactCurrency } from "@trenova/shared/lib/utils";
import { useQuery } from "@tanstack/react-query";
import type { PageGlanceKey } from "./route-command-data";

export type GlanceState = "neutral" | "warn" | "bad" | "ok";

export interface GlanceStat {
  label: string;
  value: string;
  state: GlanceState;
}

const GLANCE_STALE_TIME = 60_000;

function count(value: number | null | undefined): string | null {
  return typeof value === "number" && Number.isFinite(value) ? value.toLocaleString() : null;
}

function stat(
  label: string,
  value: string | null,
  state: GlanceState = "neutral",
): GlanceStat | null {
  return value === null ? null : { label, value, state };
}

function present(stats: (GlanceStat | null)[]): GlanceStat[] {
  return stats.filter((entry): entry is GlanceStat => entry !== null);
}

function attentionGlance(
  key: PageGlanceKey,
  summary: ReturnType<typeof useAttentionSummary>["data"],
  t: TranslateFn,
): GlanceStat[] {
  if (!summary) {
    return [];
  }
  switch (key) {
    case "service-failures":
      return present([
        stat(t("Unresolved"), count(summary.serviceFailures), warnWhenAny(summary.serviceFailures)),
      ]);
    case "billing-queue":
      return present([stat(t("In the queue"), count(summary.billingQueue))]);
    case "pending-approvals":
      return present([
        stat(
          t("Waiting on approval"),
          count(summary.pendingApprovals),
          warnWhenAny(summary.pendingApprovals),
        ),
      ]);
    case "reconciliation-exceptions":
      return present([
        stat(
          t("Open exceptions"),
          count(summary.reconciliationExceptions),
          (summary.reconciliationExceptions ?? 0) > 0 ? "bad" : "neutral",
        ),
      ]);
    case "edi":
      return present([
        stat(t("Needs attention"), count(summary.ediAttention), warnWhenAny(summary.ediAttention)),
      ]);
    case "desk":
      return present([
        stat(
          t("Decisions waiting"),
          count(summary.agentDecisions),
          warnWhenAny(summary.agentDecisions),
        ),
      ]);
    default:
      return [];
  }
}

function warnWhenAny(value: number | null | undefined): GlanceState {
  return (value ?? 0) > 0 ? "warn" : "neutral";
}

const ATTENTION_GLANCES: ReadonlySet<PageGlanceKey> = new Set([
  "service-failures",
  "billing-queue",
  "pending-approvals",
  "reconciliation-exceptions",
  "edi",
  "desk",
]);

/**
 * The live figures a page's preview shows. Each glance reads a query the page
 * itself depends on, so opening the page after the preview reuses what was
 * fetched. A glance the person cannot read, or one still loading, is empty,
 * and the preview hides its strip rather than show placeholder numbers.
 */
export function usePageGlance(key: PageGlanceKey | undefined): {
  stats: GlanceStat[];
  loading: boolean;
} {
  const t = useT();
  const timezone = useUserTimezone();

  const attention = useAttentionSummary();
  const shipments = useQuery({
    ...shipmentBoard.stageSummary({ timezone }),
    enabled: key === "shipments",
    staleTime: GLANCE_STALE_TIME,
  });
  const workers = useQuery({
    queryKey: [WORKER_ROSTER_ATTENTION_KEY],
    queryFn: ({ signal }) => fetchWorkerRosterAttention({ signal }),
    enabled: key === "workers",
    staleTime: GLANCE_STALE_TIME,
  });
  const receivables = useQuery({
    ...ar.dashboardKpis(),
    enabled: key === "receivables",
    staleTime: GLANCE_STALE_TIME,
  });

  if (!key) {
    return { stats: [], loading: false };
  }

  if (ATTENTION_GLANCES.has(key)) {
    return { stats: attentionGlance(key, attention.data, t), loading: attention.isLoading };
  }

  switch (key) {
    case "shipments": {
      const stages = new Map((shipments.data ?? []).map((row) => [row.stage, row.count]));
      return {
        stats: shipments.data
          ? present([
              stat(t("Moving"), count(stages.get("Moving") ?? 0)),
              stat(t("Late"), count(stages.get("Late") ?? 0), warnWhenAny(stages.get("Late"))),
              stat(t("Needs coverage"), count(stages.get("NeedsCoverage") ?? 0)),
            ])
          : [],
        loading: shipments.isLoading,
      };
    }
    case "workers": {
      const roster = workers.data;
      return {
        stats: roster
          ? present([
              stat(t("Active"), count(roster.activeWorkers)),
              stat(
                t("Expiring soon"),
                count(roster.expiringSoon),
                warnWhenAny(roster.expiringSoon),
              ),
              stat(
                t("Not compliant"),
                count(roster.nonCompliant),
                roster.nonCompliant > 0 ? "bad" : "neutral",
              ),
            ])
          : [],
        loading: workers.isLoading,
      };
    }
    case "receivables": {
      const overview = receivables.data?.overview;
      return {
        stats: overview
          ? present([
              stat(t("Open AR"), formatCompactCurrency(overview.totalOpenMinor / 100)),
              stat(
                t("Past due"),
                formatCompactCurrency(overview.overdueMinor / 100),
                overview.overdueMinor > 0 ? "warn" : "neutral",
              ),
            ])
          : [],
        loading: receivables.isLoading,
      };
    }
    default:
      return { stats: [], loading: false };
  }
}
