import { AnalysisResult } from "@/components/assistant/analysis-result";
import { agentExtension } from "@/lib/queries/agent-extension";
import type { AssistantArtifact } from "@/types/assistant";
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { analysisFrom } from "../../artifact-payloads";

/**
 * An analysis the agent ran: its figures, charts, tables and code. The agent is shown the first
 * rows of a long table; the card reads the whole run for the rest.
 */
export function DeskAnalysisBody({ artifact }: { artifact: AssistantArtifact }) {
  const analysis = useMemo(() => analysisFrom(artifact), [artifact]);
  const cutShort = analysis.tables.some((table) => table.rows.length < table.totalRows);
  const run = useQuery({
    ...agentExtension.analysisRun(analysis.runId),
    enabled: cutShort && analysis.runId !== "",
    staleTime: Number.POSITIVE_INFINITY,
  });
  const tables = run.data?.tables;
  const shown = useMemo(
    () => (tables && tables.length === analysis.tables.length ? { ...analysis, tables } : analysis),
    [analysis, tables],
  );

  return (
    <div className="dk-ax-pad">
      <AnalysisResult analysis={shown} />
    </div>
  );
}
