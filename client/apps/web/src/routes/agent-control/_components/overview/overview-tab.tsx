import { useT } from "@trenova/shared/i18n/use-t";
import { SuspenseLoader } from "@trenova/shared/components/component-loader";
import { ActivityIcon, BotIcon, InboxIcon, PlugZapIcon } from "lucide-react";
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
