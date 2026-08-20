import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import type { ReportIrInput, ReportPreview } from "@/lib/graphql/reports";
import { cn } from "@trenova/shared/lib/utils";
import type { ReportIR } from "@/types/report";
import { CircleAlertIcon, Table2Icon } from "lucide-react";
import { m } from "motion/react";
import { useState } from "react";
import { DrillThroughSheet, type DrillTarget } from "../../_components/drill-through-sheet";
import { buildDrillTarget } from "../../_components/drill-target";
import { ReportChart } from "../../_components/report-chart";
import { ResultGrid, type GridSort } from "../../_components/result-grid";

type PreviewGridProps = {
  ir: ReportIR;
  previewInput: ReportIrInput | null;
  previewParams?: Record<string, unknown>;
  preview: ReportPreview | undefined;
  loading: boolean;
  error: string | null;
  ready: boolean;
};

function CenteredState({ children }: { children: React.ReactNode }) {
  return (
    <m.div
      initial={{ opacity: 0, y: 6 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.25, ease: "easeOut" }}
      className="flex h-full flex-col items-center justify-center gap-3 p-6"
    >
      {children}
    </m.div>
  );
}

export function PreviewGrid({
  ir,
  previewInput,
  previewParams,
  preview,
  loading,
  error,
  ready,
}: PreviewGridProps) {
  const [view, setView] = useState("table");
  const [sort, setSort] = useState<GridSort | null>(null);
  const [drillTarget, setDrillTarget] = useState<DrillTarget | null>(null);

  const rows = Array.isArray(preview?.rows) ? (preview.rows as unknown[][]) : [];
  const totals = Array.isArray(preview?.totals) ? (preview.totals as unknown[]) : null;
  const charts = ir.charts ?? [];

  if (!ready) {
    return (
      <CenteredState>
        <div className="bg-muted flex size-10 items-center justify-center rounded-lg">
          <Table2Icon className="text-muted-foreground size-5" strokeWidth={1.75} />
        </div>
        <div className="text-center">
          <p className="text-sm font-medium">Live preview</p>
          <p className="text-muted-foreground max-w-xs text-xs">
            Add fields from the catalog on the left — the preview updates as you build.
          </p>
        </div>
      </CenteredState>
    );
  }

  if (error) {
    return (
      <CenteredState>
        <div className="bg-destructive/10 flex size-10 items-center justify-center rounded-lg">
          <CircleAlertIcon className="text-destructive size-5" strokeWidth={1.75} />
        </div>
        <div className="text-center">
          <p className="text-sm font-medium">The preview couldn&apos;t be compiled</p>
          <p className="text-muted-foreground max-w-lg text-xs whitespace-pre-wrap">{error}</p>
        </div>
      </CenteredState>
    );
  }

  if (loading && !preview) {
    return (
      <div className="flex flex-col gap-1.5 p-4">
        {Array.from({ length: 10 }, (_, i) => (
          <Skeleton key={i} className="h-6" style={{ opacity: 1 - i * 0.09 }} />
        ))}
      </div>
    );
  }

  if (!preview) return null;

  const activeChart = charts.find((chart) => chart.id === view);

  return (
    <div className="flex h-full min-h-0 flex-col">
      <div className="border-border flex h-8 shrink-0 items-center gap-2 border-b px-3">
        <button
          type="button"
          onClick={() => setView("table")}
          className={cn(
            "text-2xs font-medium tracking-wide uppercase transition-colors",
            view === "table" ? "text-foreground" : "text-muted-foreground hover:text-foreground",
          )}
        >
          Preview
        </button>
        {charts.map((chart) => (
          <button
            key={chart.id}
            type="button"
            onClick={() => setView(chart.id)}
            className={cn(
              "text-2xs max-w-32 truncate font-medium tracking-wide uppercase transition-colors",
              view === chart.id ? "text-foreground" : "text-muted-foreground hover:text-foreground",
            )}
          >
            {chart.title || chart.type}
          </button>
        ))}
        <span className="bg-muted text-2xs text-muted-foreground rounded-sm px-1.5 py-px tabular-nums">
          {rows.length} row{rows.length === 1 ? "" : "s"}
        </span>
        {preview.truncated && (
          <span className="text-2xs rounded-sm bg-amber-500/10 px-1.5 py-px text-amber-600 dark:text-amber-400">
            first 100 shown
          </span>
        )}
        <div className="flex-1" />
        {loading && <Spinner className="text-muted-foreground size-3.5" />}
      </div>

      {activeChart ? (
        <div className="flex min-h-0 flex-1 flex-col p-4">
          <ReportChart
            chart={activeChart}
            columns={preview.columns}
            rows={rows}
            className="min-h-0 flex-1"
          />
        </div>
      ) : (
        <ResultGrid
          columns={preview.columns}
          rows={rows}
          totals={totals}
          loading={loading}
          sort={sort}
          onSortChange={setSort}
          onCellClick={(rowIndex, columnIndex) => {
            const target = buildDrillTarget(ir, preview.columns, rows[rowIndex], columnIndex);
            if (target) setDrillTarget(target);
          }}
          emptyMessage="The report compiled but returned no rows for the preview window."
        />
      )}

      <DrillThroughSheet
        open={drillTarget !== null}
        onOpenChange={(open) => {
          if (!open) setDrillTarget(null);
        }}
        definition={previewInput}
        params={previewParams}
        target={drillTarget}
      />
    </div>
  );
}
