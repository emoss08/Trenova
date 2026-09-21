import { useT } from "@trenova/shared/i18n/use-t";
import { SuspenseLoader } from "@trenova/shared/components/component-loader";
import { KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import { SectionPanel } from "@/components/section-panel";
import { formatLatency, formatTokens, formatUsd } from "@/lib/ai-usage-format";
import { lazy } from "react";
import type { ActivityView } from "../rail-items";
import { AIReadinessBanner } from "../ai-readiness-banner";
import { AgentsGlance } from "./agents-glance";
import { RecentFailures } from "./recent-failures";
import { useAIControlStats } from "./use-ai-control-stats";

const AgentControlForm = lazy(() => import("../agent-control-form"));

type OverviewTabProps = {
  onOpenProviders: () => void;
  onOpenAgents: () => void;
  onOpenActivity: (view: ActivityView) => void;
};

/**
 * The state of AI in the organization on one screen: whether it can work at
 * all, the figures for the week in one strip, then the switch that pauses
 * everything and the agents at a glance.
 */
export default function OverviewTab({
  onOpenProviders,
  onOpenAgents,
  onOpenActivity,
}: OverviewTabProps) {
  const t = useT();
  const stats = useAIControlStats();
  const counts = stats.counts;
  const usage = stats.usage;
  const days = stats.usageWindowDays;

  const unpriced = usage ? usage.calls - usage.pricedCalls : 0;
  const spend = usage && usage.pricedCalls > 0 ? (formatUsd(usage.costUsd) ?? "—") : "—";
  const spendSub =
    usage === undefined || usage.calls === 0
      ? undefined
      : usage.pricedCalls === 0
        ? t("No provider has a price yet")
        : unpriced > 0
          ? t("Partial: {0} of {1} calls unpriced", unpriced, usage.calls)
          : t("Every call priced");

  return (
    <div className="flex flex-col gap-4">
      <AIReadinessBanner onOpenProviders={onOpenProviders} />

      <KpiStrip minItemWidth="9.5rem">
        <KpiStripItem
          label={t("Providers on")}
          value={stats.isLoading ? "…" : `${stats.providersEnabled} / ${stats.providersTotal}`}
          tone={stats.providersEnabled > 0 ? "success" : "warning"}
          onClick={onOpenProviders}
        />
        <KpiStripItem
          label={t("Agents on")}
          value={
            stats.isLoading ? "…" : `${counts?.agentsEnabled ?? 0} / ${counts?.agentsTotal ?? 0}`
          }
          tone={counts && counts.agentsEnabled > 0 ? "success" : "muted"}
          onClick={onOpenAgents}
        />
        <KpiStripItem
          label={t("Awaiting a decision")}
          value={stats.isLoading ? "…" : (counts?.pendingProposals ?? 0)}
          tone={counts && counts.pendingProposals > 0 ? "warning" : "muted"}
          sub={t("Proposals a person has to decide")}
          onClick={() => onOpenActivity("proposals")}
        />
        <KpiStripItem
          label={t("Runs, last 24 hours")}
          value={stats.isLoading ? "…" : (counts?.runsLast24h ?? 0)}
          onClick={() => onOpenActivity("runs")}
        />
        <KpiStripItem
          label={t("Model calls, {0} days", days)}
          value={stats.usageLoading ? "…" : (usage?.calls ?? 0)}
          tone={usage && usage.failed > 0 ? "warning" : undefined}
          sub={usage && usage.failed > 0 ? t("{0} failed", usage.failed) : undefined}
        />
        <KpiStripItem
          label={t("Spend, {0} days", days)}
          value={stats.usageLoading ? "…" : spend}
          tone={usage && usage.calls > 0 && usage.pricedCalls < usage.calls ? "warning" : undefined}
          sub={spendSub}
          onClick={onOpenProviders}
        />
        <KpiStripItem
          label={t("Median response")}
          value={
            stats.usageLoading
              ? "…"
              : usage && usage.calls > 0
                ? formatLatency(usage.latencyP50Ms)
                : "—"
          }
          sub={
            usage && usage.calls > 0
              ? t("Slowest 5% took {0}+", formatLatency(usage.latencyP95Ms))
              : undefined
          }
        />
        <KpiStripItem
          label={t("Tokens, {0} days", days)}
          value={
            stats.usageLoading
              ? "…"
              : usage
                ? formatTokens(usage.inputTokens + usage.outputTokens)
                : "0"
          }
          sub={
            usage
              ? t(
                  "{0} in, {1} out",
                  formatTokens(usage.inputTokens),
                  formatTokens(usage.outputTokens),
                )
              : undefined
          }
        />
      </KpiStrip>

      <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(0,1.4fr)]">
        <SectionPanel
          title={t("Organization-wide")}
          help={t(
            "Applies to every agent in the organization, whatever its own configuration says.",
          )}
        >
          <div className="p-3">
            <SuspenseLoader>
              <AgentControlForm />
            </SuspenseLoader>
          </div>
        </SectionPanel>

        <AgentsGlance onOpenAgents={onOpenAgents} onOpenActivity={onOpenActivity} />
      </div>

      <RecentFailures failures={usage?.recentFailures ?? []} />
    </div>
  );
}
