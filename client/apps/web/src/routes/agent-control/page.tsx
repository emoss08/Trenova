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
  QUALITY_AGENT_PARAM,
  QUALITY_SUITE_RUN_PARAM,
  QUALITY_VIEW_PARAM,
  SAFETY_VIEW_PARAM,
  activityViewParser,
  aiControlTabParser,
  qualityAgentParser,
  qualitySuiteRunParser,
  qualityViewParser,
  safetyViewParser,
  type AIControlTab,
  type ActivityView,
  type QualityView,
  type RailView,
  type SafetyView,
} from "./ai-control-tabs";
import { ControlRail } from "./_components/control-rail";
import { buildRailItems, resolveRailView } from "./_components/rail-items";
import { useAIControlStats } from "./_components/overview/use-ai-control-stats";
import { extensionState } from "./_components/extensions/extension-roster";
import { useAIControlNavigation } from "./use-ai-control-navigation";
import { queries } from "@/lib/queries";

const OverviewTab = lazy(() => import("./_components/overview/overview-tab"));
const AgentsTab = lazy(() => import("./_components/agents/agents-tab"));
const ProvidersTab = lazy(() => import("./_components/providers/providers-tab"));
const ExtensionsTab = lazy(() => import("./_components/extensions/extensions-tab"));
const MemoryTab = lazy(() => import("./_components/memory/memory-tab"));
const SafetyTab = lazy(() => import("./_components/safety/safety-tab"));
const QualityTab = lazy(() => import("./_components/quality/quality-tab"));
const ActivityTab = lazy(() => import("./_components/activity/activity-tab"));

/**
 * One place for everything AI in the organization: where work goes
 * (providers), what it may do (agents), and what it did (activity). The
 * sections run down a rail that carries their counts, and the one open
 * section takes the rest of the width. A section with several tables lists
 * them under its row and shows one at a time, because every table on the
 * page keeps its paging, filters and open row in the same address keys.
 */
export function AgentControlPage() {
  const t = useT();
  const navigate = useAIControlNavigation();
  const [tab] = useQueryState(AI_CONTROL_TAB_PARAM, aiControlTabParser);
  const [activityView] = useQueryState(ACTIVITY_VIEW_PARAM, activityViewParser);
  const [safetyView] = useQueryState(SAFETY_VIEW_PARAM, safetyViewParser);
  const [qualityView] = useQueryState(QUALITY_VIEW_PARAM, qualityViewParser);
  const [qualityAgent] = useQueryState(QUALITY_AGENT_PARAM, qualityAgentParser);
  const [suiteRun] = useQueryState(QUALITY_SUITE_RUN_PARAM, qualitySuiteRunParser);

  const { allowed: canReadAgents } = usePermission(Resource.AgentDefinition, Operation.Read);
  const { allowed: canReadProviders } = usePermission(Resource.AIProvider, Operation.Read);
  const { allowed: canReadRuns } = usePermission(Resource.AgentRun, Operation.Read);
  const { allowed: canReadProposals } = usePermission(Resource.AgentProposal, Operation.Read);
  const { allowed: canReadExceptions } = usePermission(Resource.AgentException, Operation.Read);
  const { allowed: canReadMemory } = usePermission(Resource.AgentMemory, Operation.Read);
  const { allowed: canReadExtensions } = usePermission(Resource.AgentExtension, Operation.Read);
  const { allowed: canReadQuality } = usePermission(Resource.AgentEvalSuite, Operation.Read);
  const { allowed: canReadRatings } = usePermission(Resource.AgentFeedback, Operation.Read);

  const stats = useAIControlStats();
  const extensionsQuery = useQuery({
    ...queries.agentExtension.catalog(),
    enabled: canReadExtensions,
  });
  const qualityQuery = useQuery({
    ...queries.agentQuality.overview(),
    enabled: canReadQuality,
    staleTime: 60_000,
  });
  const qualityRegressions = qualityQuery.data?.openRegressions ?? 0;
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
              qualityRegressions,
            },
        {
          agents: canReadAgents,
          providers: canReadProviders,
          extensions: canReadExtensions,
          runs: canReadRuns,
          proposals: canReadProposals,
          exceptions: canReadExceptions,
          memory: canReadMemory,
          safety: canReadAgents,
          quality: canReadQuality,
          ratings: canReadRatings,
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
      qualityRegressions,
      canReadProposals,
      canReadProviders,
      canReadQuality,
      canReadRatings,
      canReadRuns,
      stats.counts,
      stats.isLoading,
      stats.providersEnabled,
      stats.providersTotal,
      t,
    ],
  );

  // A section the reader may not open falls back to the overview rather
  // than rendering nothing under a selected rail row, and a view the reader
  // may not open falls back to the section's first.
  const activeTab: AIControlTab = items.some((item) => item.tab === tab) ? tab : "overview";
  const activeItem = items.find((item) => item.tab === activeTab);
  const activeView = resolveRailView(
    activeItem,
    requestedView(activeTab, {
      activity: activityView,
      safety: safetyView,
      quality: qualityView ?? (suiteRun || qualityAgent ? "runs" : "agents"),
    }),
  );

  const select = useCallback(
    (next: AIControlTab, nextView?: RailView) => navigate({ tab: next, view: nextView }),
    [navigate],
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
          "Providers say where AI work goes, agents say what it may do, extensions add what they can reach, quality says how well they do it, and activity shows what it did.",
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
            {activeTab === "safety" && <SafetyTab view={activeView as SafetyView} />}
            {activeTab === "quality" && <QualityTab view={activeView as QualityView} />}
            {activeTab === "activity" && <ActivityTab view={activeView as ActivityView} />}
          </DataTableLazyComponent>
        </div>
      </div>
    </PageLayout>
  );
}

type RequestedViews = {
  activity: ActivityView;
  safety: SafetyView;
  quality: QualityView;
};

function requestedView(tab: AIControlTab, views: RequestedViews): RailView | null {
  switch (tab) {
    case "activity":
      return views.activity;
    case "safety":
      return views.safety;
    case "quality":
      return views.quality;
    default:
      return null;
  }
}
