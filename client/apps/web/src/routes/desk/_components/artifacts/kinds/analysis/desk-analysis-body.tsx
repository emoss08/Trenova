import { AnalysisResult } from "@/components/assistant/analysis-result";
import type { AssistantArtifact } from "@/types/assistant";
import { useMemo } from "react";
import { analysisFrom } from "../../artifact-payloads";

/** An analysis the agent ran: its figures, charts and code. */
export function DeskAnalysisBody({ artifact }: { artifact: AssistantArtifact }) {
  const analysis = useMemo(() => analysisFrom(artifact), [artifact]);

  return (
    <div className="dk-ax-pad">
      <AnalysisResult analysis={analysis} />
    </div>
  );
}
