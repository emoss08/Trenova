import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Checkbox } from "@trenova/shared/components/ui/checkbox";
import { Input } from "@trenova/shared/components/ui/input";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { useDebounce } from "@/hooks/use-debounce";
import { useReportPreview } from "@/hooks/use-reports";
import { graphQLErrorMessage } from "@trenova/shared/lib/graphql";
import { cn } from "@trenova/shared/lib/utils";
import { irToInput } from "../../builder/_components/builder-state";
import { parseReportIR, type ReportIR, type ReportParameterDef } from "@/types/report";
import {
  ArrowLeftIcon,
  CircleAlertIcon,
  Columns3Icon,
  DownloadIcon,
  PencilIcon,
  SearchIcon,
} from "lucide-react";
import { useMemo, useState } from "react";
import { Link } from "react-router";
import { DrillThroughSheet, type DrillTarget } from "../../_components/drill-through-sheet";
import { buildDrillTarget } from "../../_components/drill-target";
import { ReportChart } from "../../_components/report-chart";
import {
  ParameterField,
  defaultParamValues,
  missingRequiredParams,
} from "../../_components/report-parameter-field";
import { ResultGrid, type GridSort } from "../../_components/result-grid";
import { RunReportDialog, type RunReportTarget } from "../../_components/run-report-dialog";

type ExploreViewProps = {
  name: string;
  description?: string;
  definition: unknown;
  target: RunReportTarget;
  defaultFormat: string;
  /** Present for saved reports, so Explore can hand off to the builder. */
  editHref?: string;
};

function ExploreEmpty({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex h-full flex-col items-center justify-center gap-3 p-6 text-center">
      {children}
    </div>
  );
}

export function ExploreView({
  name,
  description,
  definition,
  target,
  defaultFormat,
  editHref,
}: ExploreViewProps) {
  const ir = useMemo<ReportIR | null>(() => parseReportIR(definition), [definition]);
  const parameters: ReportParameterDef[] = ir?.parameters ?? [];

  const [paramValues, setParamValues] = useState<Record<string, unknown>>(() =>
    defaultParamValues(parameters),
  );
  const [quickFilter, setQuickFilter] = useState("");
  const [sort, setSort] = useState<GridSort | null>(null);
  const [hidden, setHidden] = useState<ReadonlySet<string>>(new Set());
  const [view, setView] = useState("table");
  const [drillTarget, setDrillTarget] = useState<DrillTarget | null>(null);
  const [runOpen, setRunOpen] = useState(false);

  const debouncedParams = useDebounce(paramValues, 400);
  const debouncedFilter = useDebounce(quickFilter, 200);

  const missing = missingRequiredParams(parameters, paramValues);
  const previewInput = useMemo(
    () => (ir && missing.length === 0 ? irToInput(ir) : null),
    [ir, missing.length],
  );
  const preview = useReportPreview(previewInput, debouncedParams);

  const rows = Array.isArray(preview.data?.rows) ? (preview.data.rows as unknown[][]) : [];
  const totals = Array.isArray(preview.data?.totals) ? (preview.data.totals as unknown[]) : null;
  const columns = preview.data?.columns ?? [];
  const charts = ir?.charts ?? [];
  const activeChart = charts.find((chart) => chart.id === view);

  const toggleColumn = (columnId: string, visible: boolean) => {
    setHidden((prev) => {
      const next = new Set(prev);
      if (visible) {
        next.delete(columnId);
      } else {
        next.add(columnId);
      }
      return next;
    });
  };

  if (!ir) {
    return (
      <ExploreEmpty>
        <CircleAlertIcon className="size-5 text-destructive" />
        <p className="text-sm">This report&apos;s definition could not be read.</p>
      </ExploreEmpty>
    );
  }

  return (
    <div className="flex h-full min-h-0 flex-col">
      <header className="flex h-12 shrink-0 items-center gap-2 border-b border-border px-3">
        <Button
          variant="ghost"
          size="icon"
          className="size-7"
          render={<Link to="/reports" />}
          aria-label="Back to reports"
        >
          <ArrowLeftIcon className="size-4" />
        </Button>
        <div className="flex min-w-0 flex-col">
          <span className="truncate text-sm font-medium">{name}</span>
          {description && (
            <span className="truncate text-2xs text-muted-foreground">{description}</span>
          )}
        </div>
        <div className="flex-1" />
        {editHref && (
          <Button variant="outline" size="sm" className="h-7" render={<Link to={editHref} />}>
            <PencilIcon className="size-3.5" />
            Edit
          </Button>
        )}
        <Button size="sm" className="h-7" onClick={() => setRunOpen(true)}>
          <DownloadIcon className="size-3.5" />
          Export
        </Button>
      </header>

      {parameters.length > 0 && (
        <div className="flex flex-wrap items-end gap-3 border-b border-border bg-muted/20 px-3 py-2">
          {parameters.map((param) => (
            <div key={param.name} className="w-48">
              <ParameterField
                param={param}
                value={paramValues[param.name]}
                compact
                onChange={(value) => setParamValues((prev) => ({ ...prev, [param.name]: value }))}
              />
            </div>
          ))}
        </div>
      )}

      <div className="flex h-9 shrink-0 items-center gap-2 border-b border-border px-3">
        <button
          type="button"
          onClick={() => setView("table")}
          className={cn(
            "text-2xs font-medium tracking-wide uppercase transition-colors",
            view === "table" ? "text-foreground" : "text-muted-foreground hover:text-foreground",
          )}
        >
          Table
        </button>
        {charts.map((chart) => (
          <button
            key={chart.id}
            type="button"
            onClick={() => setView(chart.id)}
            className={cn(
              "max-w-32 truncate text-2xs font-medium tracking-wide uppercase transition-colors",
              view === chart.id ? "text-foreground" : "text-muted-foreground hover:text-foreground",
            )}
          >
            {chart.title || chart.type}
          </button>
        ))}
        <Badge variant="secondary" className="tabular-nums">
          {rows.length} row{rows.length === 1 ? "" : "s"}
        </Badge>
        {preview.data?.truncated && (
          <span className="rounded-sm bg-amber-500/10 px-1.5 py-px text-2xs text-amber-600 dark:text-amber-400">
            row limit reached
          </span>
        )}
        <div className="flex-1" />
        {preview.isFetching && <Spinner className="size-3.5 text-muted-foreground" />}
        {view === "table" && (
          <>
            <Input
              className="h-7 w-56 text-xs"
              placeholder="Filter these rows..."
              leftElement={<SearchIcon className="size-3.5 text-muted-foreground" />}
              value={quickFilter}
              onChange={(event) => setQuickFilter(event.target.value)}
            />
            <Popover>
              <PopoverTrigger
                render={
                  <Button variant="outline" size="sm" className="h-7">
                    <Columns3Icon className="size-3.5" />
                    Columns
                  </Button>
                }
              />
              <PopoverContent className="max-h-80 w-56 overflow-y-auto p-2" align="end">
                <div className="flex flex-col gap-1">
                  {columns.map((column) => (
                    <label key={column.id} className="flex items-center gap-2 text-xs">
                      <Checkbox
                        checked={!hidden.has(column.id)}
                        onCheckedChange={(checked) => toggleColumn(column.id, Boolean(checked))}
                      />
                      <span className="truncate">{column.label}</span>
                    </label>
                  ))}
                </div>
              </PopoverContent>
            </Popover>
          </>
        )}
      </div>

      <div className="flex min-h-0 flex-1 flex-col">
        {missing.length > 0 ? (
          <ExploreEmpty>
            <p className="text-sm font-medium">This report needs a value first</p>
            <p className="max-w-sm text-xs text-muted-foreground">
              Fill in {missing.map((param) => param.label || param.name).join(", ")} above to load
              the results.
            </p>
          </ExploreEmpty>
        ) : preview.isError ? (
          <ExploreEmpty>
            <CircleAlertIcon className="size-5 text-destructive" />
            <p className="max-w-lg text-xs whitespace-pre-wrap text-muted-foreground">
              {graphQLErrorMessage(preview.error, "This report could not be run")}
            </p>
          </ExploreEmpty>
        ) : preview.isPending ? (
          <div className="flex flex-col gap-1.5 p-4">
            {Array.from({ length: 14 }, (_, i) => (
              <Skeleton key={i} className="h-6" style={{ opacity: 1 - i * 0.06 }} />
            ))}
          </div>
        ) : activeChart ? (
          <div className="flex min-h-0 flex-1 flex-col p-6">
            <ReportChart
              chart={activeChart}
              columns={columns}
              rows={rows}
              className="min-h-0 flex-1"
            />
          </div>
        ) : (
          <ResultGrid
            columns={columns}
            rows={rows}
            totals={totals}
            loading={preview.isFetching}
            hiddenColumnIds={hidden}
            sort={sort}
            onSortChange={setSort}
            quickFilter={debouncedFilter}
            onCellClick={(rowIndex, columnIndex) => {
              const drill = buildDrillTarget(ir, columns, rows[rowIndex], columnIndex);
              if (drill) setDrillTarget(drill);
            }}
          />
        )}
      </div>

      <DrillThroughSheet
        open={drillTarget !== null}
        onOpenChange={(open) => {
          if (!open) setDrillTarget(null);
        }}
        definition={previewInput}
        params={debouncedParams}
        target={drillTarget}
      />

      <RunReportDialog
        open={runOpen}
        onOpenChange={setRunOpen}
        target={target}
        reportName={name}
        defaultFormat={defaultFormat}
        parameters={parameters}
        initialParams={paramValues}
      />
    </div>
  );
}
