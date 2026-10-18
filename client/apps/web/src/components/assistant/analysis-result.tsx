import { AnalysisChartImage } from "@/components/assistant/analysis-chart-image";
import { KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import { analysisRunStatusAttrs } from "@/lib/analysis-run";
import { formatAnalysisValue, isNumericKind } from "@/lib/analysis-values";
import type {
  AnalysisBlock,
  AnalysisFigure,
  AnalysisRunStatus,
  AnalysisTable,
} from "@/types/agent-extension";
import { ChevronDownIcon } from "@trenova/shared/components/icons";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@trenova/shared/components/ui/collapsible";
import { HighlightedCode } from "@trenova/shared/components/ui/highlighted-code";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@trenova/shared/components/ui/table";
import { useT } from "@trenova/shared/i18n/use-t";
import { phaseTone } from "@trenova/shared/lib/status-phase";
import { cn } from "@trenova/shared/lib/utils";

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
  title: string;
  code: string;
  datasets: AnalysisDatasetView[];
  output: string;
  results: string[];
  charts: AnalysisChartView[];
  tables: AnalysisTable[];
  figures: AnalysisFigure[];
  layout: AnalysisBlock[];
  error: { name: string; value: string; traceback: string } | null;
  failure: string;
  truncated: boolean;
};

/**
 * The charts and tables in the order the analysis showed them. A run kept before the order was
 * recorded shows its charts, then its tables.
 */
function blocksOf(analysis: AnalysisView): AnalysisBlock[] {
  if (analysis.layout.length > 0) {
    return analysis.layout;
  }

  return [
    ...analysis.charts.map((chart) => ({ kind: "chart" as const, index: chart.position })),
    ...analysis.tables.map((_, index) => ({ kind: "table" as const, index })),
  ];
}

/**
 * What an analysis found, in the order a person reads it: the headline figures, then each chart
 * and table as the analysis showed them, what it could not do, and where the data came from.
 * What it printed and the code that ran stay folded beneath for whoever wants them.
 */
export function AnalysisResult({ analysis }: { analysis: AnalysisView }) {
  const t = useT();
  const status = analysisRunStatusAttrs(t)[analysis.status];
  const blocks = blocksOf(analysis);
  const shown = analysis.figures.length + blocks.length + analysis.results.length > 0;
  const titleOf = new Map(analysis.charts.map((chart) => [chart.position, chart.title]));

  return (
    <div className="flex flex-col gap-5">
      {analysis.status !== "Succeeded" && (
        <div className="flex flex-wrap items-center gap-2">
          <Badge variant={phaseTone(status.phase)} title={status.description}>
            {status.text}
          </Badge>
          <span className="text-muted-foreground text-xs">{status.description}</span>
        </div>
      )}

      {analysis.figures.length > 0 && (
        <KpiStrip minItemWidth="9rem" aria-label={t("Headline figures")}>
          {analysis.figures.map((figure, index) => (
            <KpiStripItem
              key={`${index}-${figure.label}`}
              label={figure.label}
              value={formatAnalysisValue(figure.value, figure.kind, t, figure.currency)}
              sub={figure.note || undefined}
            />
          ))}
        </KpiStrip>
      )}

      {analysis.runId !== "" &&
        blocks.map((block) => {
          if (block.kind === "chart") {
            return (
              <AnalysisChartImage
                key={`chart-${block.index}`}
                runId={analysis.runId}
                position={block.index}
                title={titleOf.get(block.index) ?? ""}
              />
            );
          }
          const table = analysis.tables[block.index];
          return table ? <AnalysisTableView key={`table-${block.index}`} table={table} /> : null;
        })}

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

      <DatasetsRead datasets={analysis.datasets} />

      {(analysis.output !== "" || analysis.code !== "") && (
        <div className="flex flex-col gap-2 border-t pt-3">
          {analysis.output !== "" && (
            <Fold label={t("What it printed")} defaultOpen={!shown}>
              <pre className="bg-muted/40 max-h-72 overflow-auto rounded-md border p-2 font-mono text-xs whitespace-pre">
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
              <HighlightedCode
                code={analysis.code}
                language="python"
                className="max-h-96 text-xs"
              />
            </Fold>
          )}
        </div>
      )}
    </div>
  );
}

/** A table the analysis showed, its values read as their kinds and numbers aligned right. */
function AnalysisTableView({ table }: { table: AnalysisTable }) {
  const t = useT();
  const shown = table.rows.length;

  return (
    <section aria-label={table.title || t("Table")} className="flex flex-col gap-1.5">
      {table.title !== "" && <h4 className="text-sm font-medium">{table.title}</h4>}
      <div className="max-h-96 overflow-auto rounded-md border">
        <Table className="text-xs">
          <TableHeader className="bg-muted/40 sticky top-0">
            <TableRow>
              {table.columns.map((column, index) => (
                <TableHead
                  key={`${index}-${column.name}`}
                  className={cn(
                    "text-foreground-subtle h-8 whitespace-nowrap",
                    isNumericKind(column.kind) && "text-right",
                  )}
                >
                  {column.name}
                </TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {table.rows.map((row, rowIndex) => (
              <TableRow key={rowIndex}>
                {table.columns.map((column, index) => (
                  <TableCell
                    key={`${rowIndex}-${index}`}
                    className={cn(
                      "py-1.5 whitespace-nowrap",
                      isNumericKind(column.kind) && "text-right tabular-nums",
                    )}
                  >
                    {formatAnalysisValue(row[index], column.kind, t, column.currency)}
                  </TableCell>
                ))}
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
      {table.totalRows > shown && (
        <p className="text-muted-foreground text-xs">
          {t("Showing {0} of {1, plural, one {# row} other {# rows}}", shown, table.totalRows)}
        </p>
      )}
    </section>
  );
}

function DatasetsRead({ datasets }: { datasets: AnalysisDatasetView[] }) {
  const t = useT();
  if (datasets.length === 0) {
    return null;
  }

  return (
    <div className="text-muted-foreground flex flex-wrap items-center gap-1.5 text-xs">
      <span>{t("From")}</span>
      <ul aria-label={t("Datasets read")} className="flex flex-wrap gap-1.5">
        {datasets.map((dataset) => (
          <li
            key={dataset.name}
            className="bg-muted/50 text-foreground rounded-badge px-1.5 py-0.5"
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
    </div>
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
      <CollapsibleTrigger className="ui-focus-ring group text-muted-foreground hover:text-foreground flex w-fit items-center gap-1 rounded-sm text-xs font-medium">
        <ChevronDownIcon className="size-3.5 transition-transform group-data-[panel-open]:rotate-180" />
        {label}
      </CollapsibleTrigger>
      <CollapsibleContent>{children}</CollapsibleContent>
    </Collapsible>
  );
}
