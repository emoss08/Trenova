import { ReportRunCard } from "@/components/assistant/report-run-card";
import { ArtifactNotice } from "@/components/assistant/voice/artifact-chrome";
import { useT } from "@trenova/shared/i18n/use-t";
import type { AssistantArtifact } from "@/types/assistant";
import { useMemo } from "react";
import { reportRunFrom } from "./artifact-payloads";

/**
 * A run the assistant started. The card follows the run itself and mints
 * the download at the moment of the click, so what the pane shows is the
 * run's state now rather than what the tool said when it started.
 */
export function ReportRunArtifact({ artifact }: { artifact: AssistantArtifact }) {
  const t = useT();
  const run = useMemo(() => reportRunFrom(artifact), [artifact]);

  if (run === null) {
    return (
      <ArtifactNotice kind={artifact.kind}>{t("This run has no id to follow.")}</ArtifactNotice>
    );
  }

  return (
    <div className="p-3">
      <ReportRunCard run={run} />
    </div>
  );
}
