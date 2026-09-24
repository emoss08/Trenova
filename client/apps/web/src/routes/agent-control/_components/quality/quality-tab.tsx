import { SectionPanel } from "@/components/section-panel";
import { usePermission } from "@/hooks/use-permission";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { parseAsString, useQueryState } from "nuqs";
import { useCallback } from "react";
import { AgentQualitySheet } from "./agent-quality-sheet";
import { AgentsPanel } from "./agents-panel";
import { EvalCasesTable } from "./cases";
import { QualityFigures } from "./quality-figures";
import { SettingsPanel } from "./settings-panel";
import { WorstRatedPanel } from "./worst-rated";

export const QUALITY_AGENT_PARAM = "agent";
export const QUALITY_SUITE_RUN_PARAM = "suiteRun";

const agentParser = parseAsString.withOptions({ history: "push", shallow: true });
const suiteRunParser = parseAsString.withOptions({ history: "replace", shallow: true });

/**
 * How well each agent is doing: what people think of its answers, how it
 * scores against its golden set every night, and whether that score fell
 * after something about it changed. The golden set and the sweep's settings
 * sit underneath, so the cases and the budget are edited where their effect
 * is read.
 */
export default function QualityTab() {
  const t = useT();
  const [agentId, setAgentId] = useQueryState(QUALITY_AGENT_PARAM, agentParser);
  const [suiteRunId, setSuiteRunId] = useQueryState(QUALITY_SUITE_RUN_PARAM, suiteRunParser);
  const { allowed: canReadRatings } = usePermission(Resource.AgentFeedback, Operation.Read);

  const openAgent = useCallback(
    (id: string) => {
      void setSuiteRunId(null);
      void setAgentId(id);
    },
    [setAgentId, setSuiteRunId],
  );
  const close = useCallback(() => {
    void setSuiteRunId(null);
    void setAgentId(null);
  }, [setAgentId, setSuiteRunId]);
  const selectRun = useCallback((id: string | null) => void setSuiteRunId(id), [setSuiteRunId]);

  return (
    <div className="flex min-w-0 flex-col gap-4">
      <QualityFigures />
      <AgentsPanel onOpen={openAgent} />
      {canReadRatings ? <WorstRatedPanel /> : null}
      <SectionPanel
        title={t("Golden set")}
        help={t(
          "The cases every agent is replayed against. Decided proposals arrive as candidates on their own; activate the ones worth keeping, write cases by hand, and quarantine a case that has stopped being fair.",
        )}
      >
        <EvalCasesTable />
      </SectionPanel>
      <SettingsPanel />
      <AgentQualitySheet
        agentId={agentId}
        suiteRunId={suiteRunId}
        onClose={close}
        onSelectRun={selectRun}
      />
    </div>
  );
}
