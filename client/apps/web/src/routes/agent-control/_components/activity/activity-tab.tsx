import { lazyComponent } from "@/lib/lazy-component";
import { aiControl } from "@/lib/queries/ai-control";
import { useQuery } from "@tanstack/react-query";
import { DataTableLazyComponent, LazyComponent } from "@trenova/shared/components/error-boundary";
import { useT } from "@trenova/shared/i18n/use-t";
import { getStartOfDay } from "@trenova/shared/lib/date";
import { Activity, lazy, useMemo } from "react";
import { useAIControlNavigation } from "../../use-ai-control-navigation";
import { Callout } from "../edit/callout";
import type { ActivityView } from "../rail-items";
import { ActivityFigures } from "./activity-figures";
import { ActivityHero } from "./activity-hero";
import {
  ACTIVITY_STALE_MS,
  OPEN_EXCEPTION_FILTERS,
  PENDING_PROPOSAL_FILTERS,
  activityFacts,
  failedTodayFilters,
} from "./activity-model";
import { ACTIVITY_TABLE_COLUMNS, ActivityHeadSkeleton } from "./activity-skeleton";
import { AnalysisRunsSkeleton } from "./analysis-runs-skeleton";

const AgentRunTable = lazy(() => import("./agent-run-table"));
const AgentProposalTable = lazy(() => import("./agent-proposal-table"));
const AgentPlanTable = lazy(() => import("./agent-plan-table"));
const AgentEvaluationTable = lazy(() => import("./agent-evaluation-table"));
const AgentExceptionTable = lazy(() => import("./agent-exception-table"));
const analysisRuns = lazyComponent(
  () => import("./analysis-runs"),
  (module) => module.AnalysisRuns,
);

/**
 * What agents did: Nova's sentence on today's runs and what waits on a person, the
 * figures, and below them the one table the view switch picks: every run, the changes
 * they proposed, the plans that group several of them, the replays, and the cases they
 * could not resolve on their own, and the analyses their scripts showed people.
 */
export default function ActivityTab({ view }: { view: ActivityView }) {
  const t = useT();
  const navigate = useAIControlNavigation();
  const since = useMemo(() => getStartOfDay(), []);
  const summary = useQuery({
    ...aiControl.activity(since),
    staleTime: ACTIVITY_STALE_MS,
    refetchInterval: 60_000,
  });
  const facts = summary.data
    ? activityFacts(summary.data, Math.floor(summary.dataUpdatedAt / 1000))
    : null;

  return (
    <div className="tabp">
      {summary.isError ? (
        <Callout tone="d">
          {t("What agents did today could not be loaded. Try again shortly.")}
        </Callout>
      ) : facts ? (
        <>
          <ActivityHero
            facts={facts}
            onShowFailed={() =>
              navigate({ tab: "activity", view: "runs", fieldFilters: failedTodayFilters(since) })
            }
            onShowProposals={() =>
              navigate({
                tab: "activity",
                view: "proposals",
                fieldFilters: PENDING_PROPOSAL_FILTERS,
              })
            }
            onShowExceptions={() =>
              navigate({
                tab: "activity",
                view: "exceptions",
                fieldFilters: OPEN_EXCEPTION_FILTERS,
              })
            }
          />
          <ActivityFigures facts={facts} />
        </>
      ) : (
        <ActivityHeadSkeleton />
      )}
      {view === "analyses" ? (
        <LazyComponent fallback={<AnalysisRunsSkeleton />}>
          <analysisRuns.Component />
        </LazyComponent>
      ) : (
        <div className="flex flex-col gap-4 pt-2">
          <Activity mode={view === "runs" ? "visible" : "hidden"}>
            <DataTableLazyComponent columnCount={ACTIVITY_TABLE_COLUMNS[view]}>
              <AgentRunTable />
            </DataTableLazyComponent>
          </Activity>
          <Activity mode={view === "proposals" ? "visible" : "hidden"}>
            <DataTableLazyComponent columnCount={ACTIVITY_TABLE_COLUMNS[view]}>
              <AgentProposalTable />
            </DataTableLazyComponent>
          </Activity>
          <Activity mode={view === "plans" ? "visible" : "hidden"}>
            <DataTableLazyComponent columnCount={ACTIVITY_TABLE_COLUMNS[view]}>
              <AgentPlanTable />
            </DataTableLazyComponent>
          </Activity>
          <Activity mode={view === "evaluations" ? "visible" : "hidden"}>
            <DataTableLazyComponent columnCount={ACTIVITY_TABLE_COLUMNS[view]}>
              <AgentEvaluationTable />
            </DataTableLazyComponent>
          </Activity>
          <Activity mode={view === "exceptions" ? "visible" : "hidden"}>
            <DataTableLazyComponent columnCount={ACTIVITY_TABLE_COLUMNS[view]}>
              <AgentExceptionTable />
            </DataTableLazyComponent>
          </Activity>
        </div>
      )}
    </div>
  );
}
