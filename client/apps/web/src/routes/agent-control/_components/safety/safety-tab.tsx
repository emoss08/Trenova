import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { lazy } from "react";
import type { SafetyView } from "../../ai-control-tabs";
import { SafetyFigures } from "./safety-figures";

const ToolRulesTable = lazy(() => import("./tool-rules-table"));
const ByAgentView = lazy(() => import("./by-agent"));

/**
 * What the AI can do without a person, answered from the same policies the
 * runtime decides every call from: the figures counted on the server over
 * every tool and agent, then one table at a time, chosen on the rail: every
 * tool's rule, or what the agents someone picks make of the tools they hold.
 */
export default function SafetyTab({ view }: { view: SafetyView }) {
  return (
    <div className="flex min-w-0 flex-col gap-4">
      <SafetyFigures />
      <DataTableLazyComponent>
        {view === "rules" && <ToolRulesTable />}
        {view === "agents" && <ByAgentView />}
      </DataTableLazyComponent>
    </div>
  );
}
