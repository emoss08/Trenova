import type { AnalysisView } from "@/components/assistant/analysis-result";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { BadgeAttrProps } from "@trenova/shared/lib/status-phase";
import type { AnalysisRun, AnalysisRunStatus } from "@/types/agent-extension";

/** How an analysis ended: ran clean, the code raised, or the sandbox could not finish it. */
export function analysisRunStatusAttrs(t: TranslateFn): Record<AnalysisRunStatus, BadgeAttrProps> {
  return {
    Running: {
      phase: "active",
      text: t("Running"),
      description: t("The code is running in the sandbox"),
    },
    Succeeded: {
      phase: "complete",
      text: t("Done"),
      description: t("The code ran to the end"),
    },
    Errored: {
      phase: "attention",
      text: t("Code error"),
      description: t("The code raised an error; the agent can fix it and run it again"),
    },
    Failed: {
      phase: "failed",
      text: t("Failed"),
      description: t("The sandbox could not run the code"),
    },
    TimedOut: {
      phase: "failed",
      text: t("Timed out"),
      description: t("The code ran out of time and was stopped"),
    },
  };
}

/**
 * A stored run as the result view draws it. The history does not carry chart
 * titles, so each chart is named by its place; what was printed to stderr
 * follows stdout, as the agent read it.
 */
export function analysisViewOf(run: AnalysisRun): AnalysisView {
  const stdout = run.stdout.trim();
  const stderr = run.stderr.trim();
  // i18n-ignore: the stream's own name, as the agent read it, not interface text
  const marked = `[stderr]\n${stderr}`;
  const output = stderr === "" ? stdout : [stdout, marked].filter(Boolean).join("\n");

  return {
    runId: run.id,
    status: run.status,
    title: run.title || run.purpose,
    code: run.code,
    datasets: run.datasets.map((dataset) => ({
      name: dataset.name,
      path: dataset.path,
      source: dataset.source,
      label: dataset.label,
      rows: dataset.rows,
    })),
    output,
    results: run.results,
    tables: run.tables,
    figures: run.figures,
    layout: run.layout,
    charts: Array.from({ length: run.chartCount }, (_, index) => ({
      position: index + 1,
      title: "",
    })),
    error:
      run.errorName !== ""
        ? { name: run.errorName, value: run.errorValue, traceback: run.traceback }
        : null,
    failure: run.failure,
    truncated: false,
  };
}
