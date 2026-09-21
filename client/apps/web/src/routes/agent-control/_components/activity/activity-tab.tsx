import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { lazy } from "react";
import type { ActivityView } from "../rail-items";

const AgentRunTable = lazy(() => import("./agent-run-table"));
const AgentProposalTable = lazy(() => import("./agent-proposal-table"));
const AgentPlanTable = lazy(() => import("./agent-plan-table"));
const AgentEvaluationTable = lazy(() => import("./agent-evaluation-table"));
const AgentExceptionTable = lazy(() => import("./agent-exception-table"));

/**
 * What agents did: every run, the changes they proposed, the plans that
 * group several of them, and the cases they could not resolve on their own.
 * Which one is showing is chosen on the rail, so the tab is only the table.
 */
export default function ActivityTab({ view }: { view: ActivityView }) {
  return (
    <DataTableLazyComponent>
      {view === "runs" && <AgentRunTable />}
      {view === "proposals" && <AgentProposalTable />}
      {view === "plans" && <AgentPlanTable />}
      {view === "evaluations" && <AgentEvaluationTable />}
      {view === "exceptions" && <AgentExceptionTable />}
    </DataTableLazyComponent>
  );
}
