import { AnalysisChartSkeleton } from "@/components/assistant/analysis-chart-skeleton";
import { KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { analysisRunStatusAttrs } from "@/lib/analysis-run";
import { formatAnalysisValue, isNumericKind } from "@/lib/analysis-values";
import { lazyComponent } from "@/lib/lazy-component";
import type {
  AnalysisBlock,
  AnalysisChart,
  AnalysisFigure,
  AnalysisRunStatus,
  AnalysisTable,
} from "@/types/analysis";
import {
  CheckIcon,
  ChevronRightIcon,
  CodeSnippet02Icon,
  Copy01Icon,
  Database01Icon,
  TerminalSquareIcon,
  type IconComponent,
} from "@trenova/shared/components/icons";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
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
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import { useT } from "@trenova/shared/i18n/use-t";
import { phaseTone } from "@trenova/shared/lib/status-phase";
import { cn } from "@trenova/shared/lib/utils";
import { useState, type ReactNode } from "react";

export type AnalysisDatasetView = {
  name: string;
  source: "ReportRun" | "Attachment" | "ReportDefinition";
  sourceId: string;
  label: string;
  rows: number;
};

/** An analysis as a person reads it, from the Desk's artifact or the stored run. */
export type AnalysisView = {
  runId: string;
  status: AnalysisRunStatus;
  title: string;
  code: string;
  datasets: AnalysisDatasetView[];
  output: string;
  charts: AnalysisChart[];
  tables: AnalysisTable[];
  figures: AnalysisFigure[];
  layout: AnalysisBlock[];
  error: { name: string; value: string; traceback: string } | null;
  failure: string;
  truncated: boolean;
};

const CODE_LANGUAGE = "python";

const analysisChart = lazyComponent(
  () => import("./analysis-chart"),
  (module) => module.AnalysisChartView,
);

// Each line of code is numbered by a CSS counter in a gutter that stays put while the code
// scrolls sideways, so the numbers are never selected or copied with the code.
const NUMBERED_LINE_CLASS =
  "before:sticky before:left-0 before:inline-block before:w-12 before:bg-sunken before:pr-4 before:text-right before:text-foreground-subtle before:tabular-nums before:select-none before:content-[counter(line)] before:[counter-increment:line]";

/**
 * The charts and tables in the order the script showed them. Without that order, its charts
 * come first, then its tables.
 */
function blocksOf(analysis: AnalysisView): AnalysisBlock[] {
  if (analysis.layout.length > 0) {
    return analysis.layout;
  }

  return [
    ...analysis.charts.map((_, index) => ({ kind: "chart" as const, index })),
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
  const shown = analysis.figures.length + blocks.length > 0;

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

      {blocks.map((block) => {
        if (block.kind === "chart") {
          const chart = analysis.charts[block.index];
          return chart ? (
            <LazyComponent
              key={`chart-${block.index}`}
              fallback={<AnalysisChartSkeleton title={chart.title || t("Chart")} />}
            >
              <analysisChart.Component chart={chart} />
            </LazyComponent>
          ) : null;
        }
        const table = analysis.tables[block.index];
        return table ? <AnalysisTableView key={`table-${block.index}`} table={table} /> : null;
      })}

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

      <AnalysisWorkings analysis={analysis} outputOpen={!shown} />
    </div>
  );
}

/** A table the analysis showed, its values read as their kinds and numbers aligned right. */
/** A table an analysis showed, its first rows and how many it holds. */
export function AnalysisTableView({ table }: { table: AnalysisTable }) {
  const t = useT();
  const shown = table.rows.length;

  return (
    <section aria-label={table.title || t("Table")} className="flex flex-col gap-1.5">
      {table.title !== "" && <h4 className="text-sm font-medium">{table.title}</h4>}
      <div className="rounded-surface scrollbar-overlay max-h-96 overflow-auto border">
        <Table className="text-xs">
          <TableHeader className="bg-sunken sticky top-0">
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
    <div className="text-foreground-muted flex flex-wrap items-center gap-1.5 text-xs">
      <span className="inline-flex items-center gap-1">
        <Database01Icon className="size-3.5" aria-hidden />
        {t("From")}
      </span>
      <ul aria-label={t("Datasets read")} className="flex min-w-0 flex-wrap gap-1.5">
        {datasets.map((dataset) => (
          <li
            key={dataset.name}
            className="bg-sunken text-foreground rounded-badge max-w-full truncate border px-1.5 py-0.5"
            title={dataset.name}
          >
            {dataset.rows > 0
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

/**
 * How the analysis got its answer, framed as one panel beneath the findings: what it printed
 * and the code that ran, each folded under its own header with a way to copy it. Both scroll
 * inside the panel, so a long line never widens the card.
 */
function AnalysisWorkings({
  analysis,
  outputOpen,
}: {
  analysis: AnalysisView;
  outputOpen: boolean;
}) {
  const t = useT();
  const code = analysis.code.trimEnd();
  if (analysis.output === "" && code === "") {
    return null;
  }

  return (
    <div className="rounded-surface divide-y overflow-hidden border">
      {analysis.output !== "" && (
        <WorkingsSection
          icon={TerminalSquareIcon}
          label={t("What it printed")}
          copyText={analysis.output}
          defaultOpen={outputOpen}
        >
          <pre className="scrollbar-overlay text-foreground max-h-72 overflow-auto px-4 py-3 font-mono text-xs leading-relaxed whitespace-pre">
            {analysis.output}
          </pre>
          {analysis.truncated && (
            <p className="text-foreground-muted border-t px-4 py-2 text-xs">
              {t("Only the end of what it printed is kept here.")}
            </p>
          )}
        </WorkingsSection>
      )}
      {code !== "" && (
        <WorkingsSection
          icon={CodeSnippet02Icon}
          label={t("Code")}
          meta={
            <>
              <span>{CODE_LANGUAGE}</span>
              <span aria-hidden>·</span>
              <span>{t("{0, plural, one {# line} other {# lines}}", code.split("\n").length)}</span>
            </>
          }
          copyText={code}
          defaultOpen={false}
        >
          <div className="scrollbar-overlay max-h-96 overflow-auto">
            <HighlightedCode
              code={code}
              language={CODE_LANGUAGE}
              className="w-max min-w-full py-3 pr-4 [counter-reset:line]"
              lineClassName={NUMBERED_LINE_CLASS}
            />
          </div>
        </WorkingsSection>
      )}
    </div>
  );
}

function WorkingsSection({
  icon: Icon,
  label,
  meta,
  copyText,
  defaultOpen,
  children,
}: {
  icon: IconComponent;
  label: string;
  meta?: ReactNode;
  copyText: string;
  defaultOpen: boolean;
  children: ReactNode;
}) {
  const t = useT();
  const [open, setOpen] = useState(defaultOpen);
  const { copy, isCopied } = useCopyToClipboard();

  return (
    <Collapsible open={open} onOpenChange={setOpen}>
      <div className="flex h-9 items-center gap-2 pr-1.5 pl-2">
        <CollapsibleTrigger className="ui-focus-ring text-foreground-muted hover:text-foreground flex min-w-0 flex-1 items-center gap-2 self-stretch rounded-sm pl-1 text-left text-xs">
          <ChevronRightIcon
            className={cn(
              "text-foreground-subtle size-3.5 shrink-0 transition-transform",
              open && "rotate-90",
            )}
            aria-hidden
          />
          <Icon className="size-3.5 shrink-0" aria-hidden />
          <span className="text-foreground shrink-0 font-medium">{label}</span>
          {meta && (
            <span className="text-foreground-subtle flex min-w-0 items-center gap-1.5 truncate font-mono">
              {meta}
            </span>
          )}
        </CollapsibleTrigger>
        {open && (
          <Button
            variant="ghost"
            size="xs"
            className="text-foreground-muted shrink-0"
            onClick={() => void copy(copyText, { timeout: 1500 })}
          >
            {isCopied ? <CheckIcon aria-hidden /> : <Copy01Icon aria-hidden />}
            {isCopied ? t("Copied") : t("Copy")}
          </Button>
        )}
      </div>
      <CollapsibleContent className="bg-sunken border-t">{children}</CollapsibleContent>
    </Collapsible>
  );
}
