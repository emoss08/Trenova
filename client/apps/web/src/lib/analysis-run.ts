import type { AnalysisView } from "@/components/assistant/analysis-result";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { BadgeAttrProps } from "@trenova/shared/lib/status-phase";
import type { AnalysisRun, AnalysisRunStatus } from "@/types/analysis";

/** How a script that showed something ended: ran clean, raised, hit a limit or could not run. */
export function analysisRunStatusAttrs(t: TranslateFn): Record<AnalysisRunStatus, BadgeAttrProps> {
  return {
    Succeeded: {
      phase: "complete",
      text: t("Done"),
      description: t("The script ran to the end"),
    },
    Errored: {
      phase: "attention",
      text: t("Script error"),
      description: t("The script raised an error; the agent can fix it and run it again"),
    },
    Stopped: {
      phase: "attention",
      text: t("Stopped"),
      description: t("The script reached a limit on its reads or memory and was stopped"),
    },
    TimedOut: {
      phase: "failed",
      text: t("Timed out"),
      description: t("The script ran out of time and was stopped"),
    },
    Failed: {
      phase: "failed",
      text: t("Failed"),
      description: t("The script could not be run"),
    },
  };
}

/** A stored run as the result view draws it. */
export function analysisViewOf(run: AnalysisRun): AnalysisView {
  return {
    runId: run.id,
    status: run.status,
    title: run.title || run.purpose,
    code: run.code,
    datasets: run.datasets.map((dataset) => ({
      name: dataset.name,
      source: dataset.source,
      sourceId: dataset.sourceId,
      label: dataset.label,
      rows: dataset.rows,
    })),
    output: run.output.trimEnd(),
    charts: run.charts,
    tables: run.tables,
    figures: run.figures,
    layout: run.layout,
    error:
      run.errorName !== ""
        ? { name: run.errorName, value: run.errorValue, traceback: run.traceback }
        : null,
    failure: run.failure,
    truncated: false,
  };
}
