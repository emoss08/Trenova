import { usePermission } from "@/hooks/use-permission";
import { downloadCsv, exportFilename } from "@/lib/data-table-export";
import type { AgentAnalytics, AIAnalyticsInput } from "@/lib/graphql/ai-analytics";
import { queries } from "@/lib/queries";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { resolveUserTimezone } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import type { FieldFilter } from "@trenova/shared/types/data-table";
import type { AiAnalyticsWindow, AiTask } from "@trenova/graphql/generated/graphql";
import { keepPreviousData, useQuery, useQueryClient } from "@tanstack/react-query";
import { useQueryState } from "nuqs";
import { useCallback, useMemo, useState } from "react";
import {
  ANALYTICS_AGENT_PARAM,
  ANALYTICS_COMPARE_PARAM,
  ANALYTICS_SERIES_PARAM,
  ANALYTICS_TASK_PARAM,
  ANALYTICS_TOOL_PARAM,
  ANALYTICS_WINDOW_PARAM,
  analyticsAgentParser,
  analyticsCompareParser,
  analyticsSeriesParser,
  analyticsTaskParser,
  analyticsToolParser,
  analyticsWindowParser,
} from "../../ai-control-tabs";
import { useAIControlNavigation } from "../../use-ai-control-navigation";
import { Ic } from "../kit/ic";
import { AnalyticsFigures } from "./analytics-figures";
import { AnalyticsScope } from "./analytics-scope";
import { AnalyticsSummary } from "./analytics-summary";
import {
  analyticsCsv,
  groupKeyOf,
  LEADERBOARD,
  sameModelPair,
  seriesColors,
  WINDOW_PHRASES,
} from "./analytics-model";
import { BoardCard } from "./board-card";
import { ComparingChips } from "./comparing-chips";
import { DecisionsCard } from "./decisions-card";
import { HighlightProvider } from "./highlight";
import { OverTimeCard } from "./over-time-card";
import { ProviderHealthCard } from "./provider-health-card";
import { RunTimeCard } from "./run-time-card";
import { SameModelCard } from "./same-model-card";
import { SeriesSheet } from "./series-sheet";
import { TaskFitCard } from "./task-fit-card";
import { ToolCallsCard } from "./tool-calls-card";
import { ToolFailureSheet } from "./tool-failure-sheet";
import { TradeoffsCard } from "./tradeoffs-card";
import { WhoseMistakeCard } from "./whose-mistake-card";
import "./analytics.css";

/** The figures are worked out every few minutes; reading them again sooner shows nothing new. */
const ANALYTICS_STALE_MS = 60_000;
const ANALYTICS_REFETCH_MS = 5 * 60_000;

/**
 * Which model, on which provider, should do which job: every model or provider the
 * organization runs, compared on completion, speed, tool calls, cost and judgment, each
 * finding linked to the setting that acts on it. Every chart reads one query, so they agree.
 */
export default function AnalyticsTab() {
  const t = useT();
  const go = useAIControlNavigation();
  const queryClient = useQueryClient();
  const timezone = useMemo(() => resolveUserTimezone(), []);
  const [range, setRange] = useQueryState(ANALYTICS_WINDOW_PARAM, analyticsWindowParser);
  const [compareBy, setCompareBy] = useQueryState(ANALYTICS_COMPARE_PARAM, analyticsCompareParser);
  const [agentId, setAgentId] = useQueryState(ANALYTICS_AGENT_PARAM, analyticsAgentParser);
  const [task, setTask] = useQueryState(ANALYTICS_TASK_PARAM, analyticsTaskParser);
  const [openSeries, setOpenSeries] = useQueryState(ANALYTICS_SERIES_PARAM, analyticsSeriesParser);
  const [openTool, setOpenTool] = useQueryState(ANALYTICS_TOOL_PARAM, analyticsToolParser);
  const [hidden, setHidden] = useState<readonly string[]>([]);

  const { allowed: canReadAgents } = usePermission(Resource.AgentDefinition, Operation.Read);
  const { allowed: canRoute } = usePermission(Resource.AIProvider, Operation.Update);

  const inputFor = useCallback(
    (window: AiAnalyticsWindow): AIAnalyticsInput => ({
      window,
      compareBy,
      agentDefinitionId: agentId ?? null,
      task: (task as AiTask | null) ?? null,
      timezone,
    }),
    [agentId, compareBy, task, timezone],
  );
  const input = useMemo(() => inputFor(range), [inputFor, range]);

  const analyticsQuery = useQuery({
    ...queries.aiAnalytics.models(input),
    staleTime: ANALYTICS_STALE_MS,
    refetchInterval: ANALYTICS_REFETCH_MS,
    placeholderData: keepPreviousData,
  });
  const providersQuery = useQuery(queries.aiProvider.list());
  const agentsQuery = useQuery({ ...queries.assistant.agents(false), enabled: canReadAgents });
  const catalogQuery = useQuery({ ...queries.aiProvider.catalog(), staleTime: Infinity });

  const prefetchWindow = useCallback(
    (window: AiAnalyticsWindow) => {
      void queryClient.prefetchQuery({
        ...queries.aiAnalytics.models(inputFor(window)),
        staleTime: ANALYTICS_STALE_MS,
      });
    },
    [inputFor, queryClient],
  );

  const analytics = analyticsQuery.data;
  const colors = useMemo(() => seriesColors(analytics?.series ?? []), [analytics?.series]);
  const deploymentColors = useMemo(
    () =>
      seriesColors(
        (analytics?.deployments ?? []).map((deployment) => ({
          key: deployment.providerId ?? deployment.key,
          colorSlot: deployment.colorSlot,
        })),
      ),
    [analytics?.deployments],
  );
  const visible = useMemo(
    () => (analytics?.series ?? []).filter((series) => !hidden.includes(series.key)),
    [analytics?.series, hidden],
  );
  const visibleDeployments = useMemo(
    () =>
      (analytics?.deployments ?? []).filter(
        (deployment) => !hidden.includes(groupKeyOf(compareBy, deployment)),
      ),
    [analytics?.deployments, compareBy, hidden],
  );
  const pair = useMemo(() => sameModelPair(visibleDeployments), [visibleDeployments]);
  const taskLabels = useMemo(
    () => new Map((catalogQuery.data?.tasks ?? []).map((item) => [item.task, item.label] as const)),
    [catalogQuery.data?.tasks],
  );

  const toggleSeries = useCallback(
    (key: string) =>
      setHidden((current) => {
        if (current.includes(key)) {
          return current.filter((item) => item !== key);
        }
        const shown = (analytics?.series.length ?? 0) - current.length;
        return shown <= 1 ? current : [...current, key];
      }),
    [analytics?.series.length],
  );
  const openSeriesSheet = useCallback(
    (key: string) => {
      void setOpenTool(null);
      void setOpenSeries(key);
    },
    [setOpenSeries, setOpenTool],
  );
  const openToolSheet = useCallback(
    (seriesKey: string, toolName: string) => {
      void setOpenSeries(seriesKey);
      void setOpenTool(toolName);
    },
    [setOpenSeries, setOpenTool],
  );
  const closeSheet = useCallback(() => {
    void setOpenSeries(null);
    void setOpenTool(null);
  }, [setOpenSeries, setOpenTool]);
  const openProvider = useCallback(
    (providerId: string) => go({ tab: "providers", panel: { mode: "edit", entityId: providerId } }),
    [go],
  );
  const exportCsv = useCallback(
    (data: AgentAnalytics) =>
      downloadCsv(
        analyticsCsv(data, t),
        exportFilename(`ai-analytics-${data.window.toLowerCase()}`),
      ),
    [t],
  );

  if (providersQuery.data && providersQuery.data.length === 0) {
    return (
      <div className="an-tab">
        <div className="empty tall">
          <b>{t("Nothing to measure yet")}</b>
          <span>
            {t("Analytics fills in once a provider is connected and agents start running.")}
          </span>
          <Button
            type="button"
            size="sm"
            className="mt-3"
            onClick={() => go({ tab: "providers", panel: { mode: "create" } })}
          >
            <Ic n="plus" s={12} />
            {t("Connect a provider")}
          </Button>
        </div>
      </div>
    );
  }

  const scope = (
    <AnalyticsScope
      range={range}
      onRange={(next) => void setRange(next)}
      onPrefetch={prefetchWindow}
      compareBy={compareBy}
      onCompareBy={(next) => {
        setHidden([]);
        void setCompareBy(next);
      }}
      agentId={agentId}
      onAgent={(next) => void setAgentId(next)}
      agents={canReadAgents ? (agentsQuery.data ?? []) : []}
      task={(task as AiTask | null) ?? null}
      onTask={(next) => void setTask(next)}
      taskLabels={taskLabels}
    />
  );
  const sheetSeries = analytics?.series.find((series) => series.key === openSeries) ?? null;
  const agentFilter: FieldFilter[] = agentId
    ? [{ field: "agentDefinitionId", operator: "eq", value: agentId }]
    : [];
  const loading = analyticsQuery.isLoading || !analytics;
  const empty = !loading && analytics.totals.runs === 0;

  return (
    <HighlightProvider>
      <div className={cn("an-tab", analyticsQuery.isPlaceholderData && "is-stale")}>
        <AnalyticsSummary
          analytics={analytics}
          loading={loading}
          onExport={analytics && !empty ? () => exportCsv(analytics) : undefined}
          onOpenSeries={openSeriesSheet}
          onOpenProvider={openProvider}
        />
        {scope}

        {loading ? (
          <AnalyticsSkeleton />
        ) : empty ? (
          <div className="empty">
            <b>
              {analytics.refreshedAt === null ? t("Working out the analytics") : t("No runs match")}
            </b>
            <span>
              {analytics.refreshedAt === null
                ? t("The last 30 days of runs are being read. They appear within a few minutes.")
                : t(
                    "Nothing ran for this agent and task in {0}.",
                    t(WINDOW_PHRASES[analytics.window].label),
                  )}
            </span>
          </div>
        ) : (
          <>
            <AnalyticsFigures totals={analytics.totals} />
            <ComparingChips
              series={analytics.series}
              colors={colors}
              compareBy={compareBy}
              hidden={hidden}
              onToggle={toggleSeries}
              onShowAll={() => setHidden([])}
            />

            <h2 className="an-h">{t("Leaderboard")}</h2>
            <div className="an-g3">
              {LEADERBOARD.map((metric) => (
                <BoardCard
                  key={`${metric}-${compareBy}`}
                  metric={metric}
                  series={visible}
                  colors={colors}
                  onOpen={openSeriesSheet}
                />
              ))}
            </div>

            <h2 className="an-h">{t("Trade-offs")}</h2>
            <div className={cn("an-g2", !(compareBy === "Model" && pair) && "is-one")}>
              <TradeoffsCard
                series={visible}
                colors={colors}
                compareBy={compareBy}
                onOpen={openSeriesSheet}
              />
              {compareBy === "Model" && pair && (
                <SameModelCard
                  pair={pair}
                  colors={deploymentColors}
                  routing={providersQuery.data ?? []}
                  canRoute={canRoute}
                />
              )}
            </div>

            <h2 className="an-h">{t("Run time")}</h2>
            <RunTimeCard
              series={visible}
              colors={colors}
              note={analytics.runTimeNote}
              onOpen={openSeriesSheet}
              onOpenProvider={openProvider}
            />
            <OverTimeCard
              key={`${compareBy}-${analytics.window}`}
              analytics={analytics}
              series={visible}
              colors={colors}
            />

            <h2 className="an-h">{t("Tool calls")}</h2>
            <div className="an-g2 is-tools">
              <ToolCallsCard
                tools={analytics.tools}
                series={visible}
                colors={colors}
                onOpenCell={openToolSheet}
              />
              <WhoseMistakeCard series={visible} colors={colors} />
            </div>

            <h2 className="an-h">{t("Routing and judgment")}</h2>
            {!task && (
              <TaskFitCard
                tasks={analytics.tasks}
                series={visible}
                colors={colors}
                compareBy={compareBy}
                taskLabels={taskLabels}
                canRoute={canRoute}
              />
            )}
            <div className="an-g2 is-health">
              <DecisionsCard series={visible} colors={colors} onOpen={openSeriesSheet} />
              <ProviderHealthCard
                providers={analytics.providers}
                hidden={hidden}
                compareBy={compareBy}
                until={analytics.until}
                onOpenProvider={openProvider}
              />
            </div>
          </>
        )}

        <SeriesSheet
          open={Boolean(sheetSeries) && !openTool}
          series={sheetSeries}
          tools={analytics?.tools ?? []}
          until={analytics?.until ?? 0}
          compareBy={compareBy}
          onOpenProvider={openProvider}
          onOpenRuns={() => go({ tab: "activity", view: "runs", fieldFilters: agentFilter })}
          onClose={closeSheet}
        />
        <ToolFailureSheet
          open={Boolean(openSeries && openTool)}
          input={input}
          series={sheetSeries}
          seriesKey={openSeries}
          toolName={openTool}
          onOpenToolRules={() => go({ tab: "safety", view: "rules" })}
          onOpenFailedRuns={() =>
            go({
              tab: "activity",
              view: "runs",
              fieldFilters: [...agentFilter, { field: "status", operator: "eq", value: "Failed" }],
            })
          }
          onClose={closeSheet}
        />
      </div>
    </HighlightProvider>
  );
}

function AnalyticsSkeleton() {
  return (
    <div className="flex flex-col gap-4 pt-4" aria-busy>
      <Skeleton className="h-20 w-full" />
      <div className="an-g3">
        <Skeleton className="h-72 w-full" />
        <Skeleton className="h-72 w-full" />
        <Skeleton className="h-72 w-full" />
      </div>
      <Skeleton className="h-96 w-full" />
    </div>
  );
}
