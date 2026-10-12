import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { useT } from "@trenova/shared/i18n/use-t";
import { useQueryStates } from "nuqs";
import { lazy, useCallback } from "react";
import {
  QUALITY_SHEET_PARAM,
  SWEEP_SETTINGS_PARAM,
  qualitySheetParser,
  sweepSettingsParser,
  type QualityView,
} from "../../ai-control-tabs";
import { Callout } from "../edit/callout";
import { Ic } from "../kit/ic";
import { AgentQualitySheet } from "./agent-quality-sheet";
import { EvalCasesTable } from "./cases";
import { QualityFigures } from "./quality-figures";
import { QualityHero } from "./quality-hero";
import { QUALITY_STALE_MS } from "./quality-model";
import { SweepSettingsEditor } from "./sweep-settings-editor";
import { Button } from "@trenova/shared/components/ui/button";

const qualityDialogParsers = {
  [QUALITY_SHEET_PARAM]: qualitySheetParser,
  [SWEEP_SETTINGS_PARAM]: sweepSettingsParser,
};

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
  const [{ [QUALITY_SHEET_PARAM]: openAgent, [SWEEP_SETTINGS_PARAM]: settingsOpen }, setDialogs] =
    useQueryStates(qualityDialogParsers);
  const setOpenAgent = useCallback(
    (agentId: string | null) => void setDialogs({ [QUALITY_SHEET_PARAM]: agentId }),
    [setDialogs],
  );
  const headed = view !== "extraction" && view !== "golden";

  const settingsButton = (
    <Button
      type="button"
      variant="outline"
      size="sm"
      onClick={() => void setDialogs({ [SWEEP_SETTINGS_PARAM]: true })}
    >
      <Ic n="gear" s={13} />
      {t("Sweep settings")}
    </Button>
  );

  return (
    <div className="tabp flex flex-col gap-1">
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
          open={settingsOpen === true}
          control={control.data}
          onClose={() => void setDialogs({ [SWEEP_SETTINGS_PARAM]: null })}
        />
      )}
    </div>
  );
}
