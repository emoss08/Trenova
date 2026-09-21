import { useT } from "@trenova/shared/i18n/use-t";
import { SuspenseLoader } from "@trenova/shared/components/component-loader";
import { ActivityIcon, BotIcon, CoinsIcon, InboxIcon, PlugZapIcon, TimerIcon } from "lucide-react";
import { formatLatency, formatTokens, formatUsd } from "@/lib/ai-usage-format";
import { useQueryState } from "nuqs";
import { lazy } from "react";
import { AI_CONTROL_TAB_PARAM, aiControlTabParser } from "../../ai-control-tabs";
import { AIReadinessBanner } from "../ai-readiness-banner";
import { StatCard } from "./stat-card";
import { useAIControlStats } from "./use-ai-control-stats";

const AgentControlForm = lazy(() => import("../agent-control-form"));

type OverviewTabProps = {
  onOpenProviders: () => void;
};

export default function OverviewTab({ onOpenProviders }: OverviewTabProps) {
  const t = useT();
  const [, setTab] = useQueryState(AI_CONTROL_TAB_PARAM, aiControlTabParser);
  const stats = useAIControlStats();

  const counts = stats.counts;

  return (
    <div className="flex flex-col gap-6">
      <AIReadinessBanner onOpenProviders={onOpenProviders} />

      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <StatCard
          icon={PlugZapIcon}
          label={t("Providers ready")}
          value={`${stats.providersEnabled} / ${stats.providersTotal}`}
          hint={
            stats.providersTotal === 0
              ? t("None connected yet")
              : t("Enabled endpoints work can route to")
          }
          tone={stats.providersEnabled > 0 ? "success" : "warning"}
          isLoading={stats.isLoading}
          onClick={onOpenProviders}
        />
        <StatCard
          icon={BotIcon}
          label={t("Agents enabled")}
          value={`${counts?.agentsEnabled ?? 0} / ${counts?.agentsTotal ?? 0}`}
          hint={t("Chat, scheduled, event-driven and continuous agents")}
          tone={counts && counts.agentsEnabled > 0 ? "success" : "default"}
          isLoading={stats.isLoading}
          onClick={() => void setTab("agents")}
        />
        <StatCard
          icon={InboxIcon}
          label={t("Awaiting a decision")}
          value={counts?.pendingProposals ?? 0}
          hint={t("Proposals a person still has to approve or reject")}
          tone={counts && counts.pendingProposals > 0 ? "warning" : "default"}
          isLoading={stats.isLoading}
          onClick={() => void setTab("activity")}
        />
        <StatCard
          icon={ActivityIcon}
          label={t("Runs in the last 24 hours")}
          value={counts?.runsLast24h ?? 0}
          hint={t("Every agent run, whatever started it")}
          isLoading={stats.isLoading}
          onClick={() => void setTab("activity")}
        />
      </div>

      <UsageTiles stats={stats} onOpenProviders={onOpenProviders} />

      <section className="flex flex-col gap-3">
        <div>
          <h2 className="text-base font-semibold">{t("Organization-wide controls")}</h2>
          <p className="text-muted-foreground max-w-prose text-sm">
            {t(
              "Settings that apply to every agent in the organization, whatever its own configuration says.",
            )}
          </p>
        </div>
        <SuspenseLoader>
          <AgentControlForm />
        </SuspenseLoader>
      </section>
    </div>
  );
}

/**
 * What the models did this week: calls, what they consumed, what it cost
 * where the price is known, and how long a person waited.
 *
 * Cost is labelled partial when some calls were unpriced. Summing them as
 * zero would understate spend by exactly the providers nobody got round to
 * pricing, which are the ones whose spend is a surprise.
 */
function UsageTiles({
  stats,
  onOpenProviders,
}: {
  stats: ReturnType<typeof useAIControlStats>;
  onOpenProviders: () => void;
}) {
  const t = useT();
  const usage = stats.usage;
  const days = stats.usageWindowDays;

  const cost = formatUsd(usage?.costUsd);
  const unpriced = usage ? usage.calls - usage.pricedCalls : 0;
  const costHint =
    usage === undefined
      ? ""
      : usage.pricedCalls === 0
        ? t("No provider has a price yet")
        : unpriced > 0
          ? t("Partial: {0} of {1} calls had no price", unpriced, usage.calls)
          : t("Every call was priced");

  return (
    <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
      <StatCard
        icon={ActivityIcon}
        label={t("Model calls, last {0} days", days)}
        value={usage?.calls ?? 0}
        hint={
          usage && usage.failed > 0
            ? t("{0} failed", usage.failed)
            : t("Every attempt, whichever provider answered")
        }
        tone={usage && usage.failed > 0 ? "warning" : "default"}
        isLoading={stats.usageLoading}
      />
      <StatCard
        icon={CoinsIcon}
        label={t("Spend, last {0} days", days)}
        value={usage && usage.pricedCalls > 0 ? (cost ?? "—") : "—"}
        hint={costHint}
        tone={usage && usage.calls > 0 && usage.pricedCalls < usage.calls ? "warning" : "default"}
        isLoading={stats.usageLoading}
        onClick={onOpenProviders}
      />
      <StatCard
        icon={TimerIcon}
        label={t("Median response")}
        value={usage && usage.calls > 0 ? formatLatency(usage.latencyP50Ms) : "—"}
        hint={
          usage && usage.calls > 0
            ? t("Slowest 5% took {0} or more", formatLatency(usage.latencyP95Ms))
            : t("Over successful calls")
        }
        isLoading={stats.usageLoading}
      />
      <StatCard
        icon={BotIcon}
        label={t("Tokens, last {0} days", days)}
        value={usage ? formatTokens(usage.inputTokens + usage.outputTokens) : "0"}
        hint={
          usage
            ? t(
                "{0} in, {1} out",
                formatTokens(usage.inputTokens),
                formatTokens(usage.outputTokens),
              )
            : ""
        }
        isLoading={stats.usageLoading}
      />
    </div>
  );
}
