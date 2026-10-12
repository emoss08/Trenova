import { ReportRunCard } from "@/components/assistant/report-run-card";
import type { AssistantArtifact } from "@/types/assistant";
import { useT } from "@trenova/shared/i18n/use-t";
import { useMemo } from "react";
import { reportRunFrom } from "../../artifact-payloads";
import { Notice } from "../artifact-body-parts";

/** A run the agent started, following the run itself. */
export function DeskReportRunBody({ artifact }: { artifact: AssistantArtifact }) {
  const t = useT();
  const run = useMemo(() => reportRunFrom(artifact), [artifact]);
  if (run === null) {
    return <Notice>{t("This run has no id to follow.")}</Notice>;
  }
  return (
    <div className="dk-ax-pad dk-ax-rep">
      <ReportRunCard run={run} />
    </div>
  );
}
