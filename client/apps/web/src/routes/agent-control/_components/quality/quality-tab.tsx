import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { lazy } from "react";
import type { QualityView } from "../../ai-control-tabs";
import { EvalCasesTable } from "./cases";
import { QualityFigures } from "./quality-figures";
import { SettingsPanel } from "./settings-panel";

const AgentsTable = lazy(() => import("./agents-table"));
const SuiteRunsView = lazy(() => import("./suite-runs-view"));
const WorstRatedTable = lazy(() => import("./worst-rated"));
const ExtractionView = lazy(() => import("./extraction/extraction-view"));

/**
 * How well each agent is doing: what people think of its answers, how it
 * scores against its golden set every night, and whether that score fell
 * after something about it changed. The figures head every view; below them
 * is the one table chosen on the rail, or the sweep's settings.
 */
export default function QualityTab({ view }: { view: QualityView }) {
  return (
    <div className="flex min-w-0 flex-col gap-4">
      {view === "settings" || view === "extraction" ? null : <QualityFigures />}
      <DataTableLazyComponent>
        {view === "agents" && <AgentsTable />}
        {view === "runs" && <SuiteRunsView />}
        {view === "ratings" && <WorstRatedTable />}
        {view === "golden" && <EvalCasesTable />}
        {view === "extraction" && <ExtractionView />}
        {view === "settings" && <SettingsPanel />}
      </DataTableLazyComponent>
    </div>
  );
}
