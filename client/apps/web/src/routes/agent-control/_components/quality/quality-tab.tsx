import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { useT } from "@trenova/shared/i18n/use-t";
import { lazy, useState } from "react";
import type { QualityView } from "../../ai-control-tabs";
import { Callout } from "../edit/fields";
import { Ic } from "../kit/ic";
import { AgentQualitySheet } from "./agent-quality-sheet";
import { EvalCasesTable } from "./cases";
import { QualityFigures } from "./quality-figures";
import { QualityHero } from "./quality-hero";
import { QUALITY_STALE_MS } from "./quality-model";
import { SweepSettingsEditor } from "./sweep-settings-editor";

const AgentsTable = lazy(() => import("./agents-table"));
const SuiteRunsView = lazy(() => import("./suite-runs-view"));
const WorstRatedTable = lazy(() => import("./worst-rated"));
const ExtractionView = lazy(() => import("./extraction/extraction-view"));

/**
 * How well each agent is doing: what people think of its answers, how it scores against
 * its golden set every night, and whether that score fell after something about it
 * changed. Nova's sentence and the figures head the agent views; below them is the one
 * table the view switch picks. The sweep's settings open over the agents.
 */
export default function QualityTab({ view }: { view: QualityView }) {
  const t = useT();
  const overview = useQuery({ ...queries.agentQuality.overview(), staleTime: QUALITY_STALE_MS });
  const control = useQuery(queries.agentQuality.control());
  const [openAgent, setOpenAgent] = useState<string | null>(null);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const headed = view !== "extraction" && view !== "golden";

  const settingsButton = (
    <button type="button" className="btn" onClick={() => setSettingsOpen(true)}>
      <Ic n="gear" s={13} />
      {t("Sweep settings")}
    </button>
  );

  return (
    <div className="tabp">
      {headed &&
        (overview.isError ? (
          <Callout tone="d">
            {t("How well agents are doing could not be loaded. Try again shortly.")}
          </Callout>
        ) : (
          overview.data && (
            <>
              <QualityHero overview={overview.data} onOpenAgent={setOpenAgent} />
              <QualityFigures overview={overview.data} />
            </>
          )
        ))}
      <DataTableLazyComponent>
        {view === "agents" && <AgentsTable toolbar={settingsButton} />}
        {view === "runs" && <SuiteRunsView />}
        {view === "ratings" && <WorstRatedTable />}
        {view === "golden" && <EvalCasesTable />}
        {view === "extraction" && <ExtractionView />}
      </DataTableLazyComponent>
      <AgentQualitySheet
        agentId={openAgent}
        agentName={
          overview.data?.worstRegression?.agentDefinitionId === openAgent
            ? overview.data?.worstRegression?.agentName
            : undefined
        }
        onClose={() => setOpenAgent(null)}
      />
      {control.data && (
        <SweepSettingsEditor
          open={settingsOpen}
          control={control.data}
          onClose={() => setSettingsOpen(false)}
        />
      )}
    </div>
  );
}
