import { AnalysisChartImage } from "@/components/assistant/analysis-chart-image";
import { analysisRunStatusAttrs } from "@/lib/analysis-run";
import type { AnalysisRunStatus } from "@/types/agent-extension";
import { ChevronDownIcon } from "@trenova/shared/components/icons";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@trenova/shared/components/ui/collapsible";
import { HighlightedCode } from "@trenova/shared/components/ui/highlighted-code";
import { useT } from "@trenova/shared/i18n/use-t";
import { phaseTone } from "@trenova/shared/lib/status-phase";

export type AnalysisDatasetView = {
  name: string;
  path: string;
  source: "ReportRun" | "Attachment";
  label: string;
  rows: number;
};

export type AnalysisChartView = { position: number; title: string };

/** An analysis as a person reads it, from the Desk's artifact or the stored run. */
export type AnalysisView = {
  runId: string;
  status: AnalysisRunStatus;
  code: string;
  datasets: AnalysisDatasetView[];
  output: string;
  results: string[];
  charts: AnalysisChartView[];
  error: { name: string; value: string; traceback: string } | null;
  failure: string;
  truncated: boolean;
};

/**
 * What an analysis worked out, in the order a person reads it: the charts, the
 * figures it returned, the error it raised, what it printed, the data it read,
 * and last the code.
 */
export function AnalysisResult({ analysis }: { analysis: AnalysisView }) {
  const t = useT();
  const status = analysisRunStatusAttrs(t)[analysis.status];

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center gap-2">
        <Badge variant={phaseTone(status.phase)} title={status.description}>
          {status.text}
        </Badge>
        <DatasetsRead datasets={analysis.datasets} />
      </div>

      {analysis.runId !== "" && analysis.charts.length > 0 && (
        <div className="grid gap-3">
          {analysis.charts.map((chart) => (
            <AnalysisChartImage
              key={chart.position}
              runId={analysis.runId}
              position={chart.position}
              title={chart.title}
            />
          ))}
        </div>
      )}

      {analysis.results.length > 0 && (
        <section aria-label={t("Results")} className="flex flex-col gap-1.5">
          {analysis.results.map((result, index) => (
            <pre
              key={`${index}-${result.slice(0, 16)}`}
              className="bg-muted/40 overflow-x-auto rounded-md border p-2 font-mono text-xs whitespace-pre-wrap"
            >
              {result}
            </pre>
          ))}
        </section>
      )}

      {analysis.error && (
        <Alert size="sm" variant="destructive">
          <AlertTitle>
            {analysis.error.value
              ? t("{0}: {1}", analysis.error.name, analysis.error.value)
              : analysis.error.name}
          </AlertTitle>
          {analysis.error.traceback && (
            <AlertDescription>
              <pre className="max-h-48 overflow-auto font-mono text-xs whitespace-pre-wrap">
                {analysis.error.traceback}
              </pre>
            </AlertDescription>
          )}
        </Alert>
      )}

      {analysis.failure !== "" && !analysis.error && (
        <Alert size="sm" variant="destructive">
          <AlertDescription>{analysis.failure}</AlertDescription>
        </Alert>
      )}

      {analysis.output !== "" && (
        <Fold label={t("What it printed")} defaultOpen={analysis.charts.length === 0}>
          <pre className="max-h-72 overflow-auto font-mono text-xs whitespace-pre-wrap">
            {analysis.output}
          </pre>
          {analysis.truncated && (
            <p className="text-muted-foreground mt-1 text-xs">
              {t("Only the end of what it printed is kept here.")}
            </p>
          )}
        </Fold>
      )}

      {analysis.code !== "" && (
        <Fold label={t("Code")} defaultOpen={false}>
          <HighlightedCode code={analysis.code} language="python" className="max-h-96 text-xs" />
        </Fold>
      )}
    </div>
  );
}

function DatasetsRead({ datasets }: { datasets: AnalysisDatasetView[] }) {
  const t = useT();
  if (datasets.length === 0) {
    return <span className="text-muted-foreground text-xs">{t("Read no datasets")}</span>;
  }

  return (
    <ul aria-label={t("Datasets read")} className="flex flex-wrap gap-1.5">
      {datasets.map((dataset) => (
        <li
          key={dataset.name}
          className="bg-muted/50 rounded-badge px-1.5 py-0.5 text-xs"
          title={dataset.path}
        >
          {dataset.source === "ReportRun" && dataset.rows > 0
            ? t(
                "{0} · {1, plural, one {# row} other {# rows}}",
                dataset.label || dataset.name,
                dataset.rows,
              )
            : dataset.label || dataset.name}
        </li>
      ))}
    </ul>
  );
}

function Fold({
  label,
  defaultOpen,
  children,
}: {
  label: string;
  defaultOpen: boolean;
  children: React.ReactNode;
}) {
  return (
    <Collapsible defaultOpen={defaultOpen} className="flex flex-col gap-1.5">
      <CollapsibleTrigger
        className="ui-focus-ring group text-muted-foreground hover:text-foreground flex w-fit items-center gap-1 rounded-sm text-xs font-medium"
      >
        <ChevronDownIcon
          className="size-3.5 transition-transform group-data-[panel-open]:rotate-180"
        />
        {label}
      </CollapsibleTrigger>
      <CollapsibleContent>{children}</CollapsibleContent>
    </Collapsible>
  );
}
