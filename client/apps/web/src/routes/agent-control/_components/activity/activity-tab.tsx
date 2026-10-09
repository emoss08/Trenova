import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { useT } from "@trenova/shared/i18n/use-t";
import { getStartOfDay } from "@trenova/shared/lib/date";
import { lazy, useMemo } from "react";
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

const AgentRunTable = lazy(() => import("./agent-run-table"));
const AgentProposalTable = lazy(() => import("./agent-proposal-table"));
const AgentPlanTable = lazy(() => import("./agent-plan-table"));
const AgentEvaluationTable = lazy(() => import("./agent-evaluation-table"));
const AgentExceptionTable = lazy(() => import("./agent-exception-table"));

/**
 * What agents did: Nova's sentence on today's runs and what waits on a person, the
 * figures, and below them the one table the view switch picks: every run, the changes
 * they proposed, the plans that group several of them, the replays, and the cases they
 * could not resolve on their own.
 */
export default function ActivityTab({ view }: { view: ActivityView }) {
  const t = useT();
  const navigate = useAIControlNavigation();
  const since = useMemo(() => getStartOfDay(), []);
  const summary = useQuery({
    ...queries.aiControl.activity(since),
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
      ) : (
        facts && (
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
        )
      )}
      <DataTableLazyComponent>
        {view === "runs" && <AgentRunTable />}
        {view === "proposals" && <AgentProposalTable />}
        {view === "plans" && <AgentPlanTable />}
        {view === "evaluations" && <AgentEvaluationTable />}
        {view === "exceptions" && <AgentExceptionTable />}
      </DataTableLazyComponent>
    </div>
  );
}
