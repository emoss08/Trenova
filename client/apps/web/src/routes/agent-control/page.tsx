import { PageLayout } from "@/components/navigation/sidebar-layout";
import { usePermission } from "@/hooks/use-permission";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQuery } from "@tanstack/react-query";
import { useQueryState } from "nuqs";
import { lazy, useCallback, useMemo } from "react";
import {
  ACTIVITY_VIEW_PARAM,
  AI_CONTROL_TAB_PARAM,
  activityViewParser,
  aiControlTabParser,
  type AIControlTab,
} from "./ai-control-tabs";
import { ControlRail } from "./_components/control-rail";
import { buildRailItems, type ActivityView } from "./_components/rail-items";
import { useAIControlStats } from "./_components/overview/use-ai-control-stats";
import { extensionState } from "./_components/extensions/extension-roster";
import { queries } from "@/lib/queries";

const OverviewTab = lazy(() => import("./_components/overview/overview-tab"));
const AgentsTab = lazy(() => import("./_components/agents/agents-tab"));
const ProvidersTab = lazy(() => import("./_components/providers/providers-tab"));
const ExtensionsTab = lazy(() => import("./_components/extensions/extensions-tab"));
const MemoryTab = lazy(() => import("./_components/memory/memory-tab"));
const ActivityTab = lazy(() => import("./_components/activity/activity-tab"));

/**
 * One place for everything AI in the organization: where work goes
 * (providers), what it may do (agents), and what it did (activity). The
 * sections run down a rail that carries their counts, and the one open
 * section takes the rest of the width.
 */
export function AgentControlPage() {
  const t = useT();
  const [tab, setTab] = useQueryState(AI_CONTROL_TAB_PARAM, aiControlTabParser);
  const [view, setView] = useQueryState(ACTIVITY_VIEW_PARAM, activityViewParser);

  const { allowed: canReadAgents } = usePermission(Resource.AgentDefinition, Operation.Read);
  const { allowed: canReadProviders } = usePermission(Resource.AIProvider, Operation.Read);
  const { allowed: canReadRuns } = usePermission(Resource.AgentRun, Operation.Read);
  const { allowed: canReadProposals } = usePermission(Resource.AgentProposal, Operation.Read);
  const { allowed: canReadExceptions } = usePermission(Resource.AgentException, Operation.Read);
  const { allowed: canReadMemory } = usePermission(Resource.AgentMemory, Operation.Read);
  const { allowed: canReadExtensions } = usePermission(Resource.AgentExtension, Operation.Read);

  const stats = useAIControlStats();
  const extensionsQuery = useQuery({
    ...queries.agentExtension.catalog(),
    enabled: canReadExtensions,
  });
  const extensionItems = extensionsQuery.data?.items;
  const extensionsOn = useMemo(
    () => extensionItems?.filter((item) => extensionState(item) === "on").length ?? 0,
    [extensionItems],
  );
  const items = useMemo(
    () =>
      buildRailItems(
        stats.isLoading
          ? undefined
          : {
              providersEnabled: stats.providersEnabled,
              providersTotal: stats.providersTotal,
              agentsEnabled: stats.counts?.agentsEnabled ?? 0,
              agentsTotal: stats.counts?.agentsTotal ?? 0,
              pendingProposals: stats.counts?.pendingProposals ?? 0,
              runsLast24h: stats.counts?.runsLast24h ?? 0,
              memoriesActive: stats.counts?.memoriesActive ?? 0,
              extensionsOn,
              extensionsTotal: extensionItems?.length ?? 0,
            },
        {
          agents: canReadAgents,
          providers: canReadProviders,
          extensions: canReadExtensions,
          runs: canReadRuns,
          proposals: canReadProposals,
          exceptions: canReadExceptions,
          memory: canReadMemory,
        },
        t,
      ),
    [
      canReadAgents,
      canReadExceptions,
      canReadExtensions,
      extensionItems?.length,
      extensionsOn,
      canReadMemory,
      canReadProposals,
      canReadProviders,
      canReadRuns,
      stats.counts,
      stats.isLoading,
      stats.providersEnabled,
      stats.providersTotal,
      t,
    ],
  );

  // A section the reader may not open falls back to the overview rather
  // than rendering nothing under a selected rail row.
  const activeTab: AIControlTab = items.some((item) => item.tab === tab) ? tab : "overview";
  const activeView: ActivityView =
    (items.find((item) => item.tab === "activity")?.children.some((c) => c.view === view) ?? false)
      ? view
      : "runs";

  const select = useCallback(
    (next: AIControlTab, nextView?: ActivityView) => {
      void setTab(next);
      if (nextView) {
        void setView(nextView);
      }
    },
    [setTab, setView],
  );
  const openProviders = useCallback(() => select("providers"), [select]);
  const openAgents = useCallback(() => select("agents"), [select]);
  const openActivity = useCallback(
    (nextView: ActivityView) => select("activity", nextView),
    [select],
  );

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("AI control"),
        description: t(
          "Providers say where AI work goes, agents say what it may do, extensions add what they can reach, and activity shows what it did.",
        ),
      }}
    >
      <div className="grid min-w-0 gap-4 md:grid-cols-[13.5rem_minmax(0,1fr)] md:gap-6">
        <ControlRail items={items} active={activeTab} activeView={activeView} onSelect={select} />

        <div className="min-w-0">
          <DataTableLazyComponent>
            {activeTab === "overview" && (
              <OverviewTab
                onOpenProviders={openProviders}
                onOpenAgents={openAgents}
                onOpenActivity={openActivity}
              />
            )}
            {activeTab === "agents" && <AgentsTab />}
            {activeTab === "providers" && <ProvidersTab />}
            {activeTab === "extensions" && <ExtensionsTab />}
            {activeTab === "memory" && <MemoryTab />}
            {activeTab === "activity" && <ActivityTab view={activeView} />}
          </DataTableLazyComponent>
        </div>
      </div>
    </PageLayout>
  );
}
