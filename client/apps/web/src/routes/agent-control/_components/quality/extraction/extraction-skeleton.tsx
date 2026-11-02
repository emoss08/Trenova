import { KpiStripSkeleton } from "@/components/kpi/kpi-strip-skeleton";
import { DataTableSkeleton } from "@trenova/shared/components/data-table/data-table-skeleton";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useQueryState } from "nuqs";
import type { ReactNode } from "react";
import {
  EXTRACTION_VIEW_PARAM,
  extractionViewParser,
  type ExtractionView,
} from "../../../ai-control-tabs";

/** What the corrections table shows at first. */
const CORRECTION_COLUMNS = 7;

/** Field accuracy and corrections. */
export const EXTRACTION_FIGURES = 2;

type PanelSkeletonProps = {
  /** The title's width. */
  title: number;
  /** Whether the header carries a note at its far end. */
  hint?: boolean;
  children: ReactNode;
};

/** A section panel: its one-line header over the body it stands in for. */
function PanelSkeleton({ title, hint = false, children }: PanelSkeletonProps) {
  return (
    <div className="bg-card flex flex-col overflow-hidden rounded-lg border" aria-hidden>
      <div className="border-border flex min-h-9 items-center justify-between gap-2 border-b px-3 py-1.5">
        <Skeleton className="h-3.5" style={{ width: title }} />
        {hint && <Skeleton className="h-3 w-20" />}
      </div>
      {children}
    </div>
  );
}

/** A panel's table: the header row and the rows under it, each at the table's own height. */
export function PanelRowsSkeleton({ rows, columns = 4 }: { rows: number; columns?: number }) {
  return (
    <div className="flex flex-col" aria-busy="true">
      {Array.from({ length: rows + 1 }, (_, row) => (
        <div
          key={row}
          className={
            row === 0
              ? "border-border flex h-(--row-head-h) items-center gap-4 border-b px-3"
              : "border-border flex h-(--row-h) items-center gap-4 border-b px-3 last:border-b-0"
          }
        >
          {Array.from({ length: columns }, (_, column) => (
            <Skeleton
              key={column}
              className={column === 0 ? "h-3 flex-[2]" : row === 0 ? "h-3 flex-1" : "h-3.5 flex-1"}
            />
          ))}
        </div>
      ))}
    </div>
  );
}

/** Accuracy by field beside the comparisons by model and by document. */
export function AccuracyPanelSkeleton() {
  return (
    <div className="grid min-w-0 gap-4 xl:grid-cols-[2fr_1fr]" aria-busy="true">
      <PanelSkeleton title={112} hint>
        <PanelRowsSkeleton rows={8} />
      </PanelSkeleton>
      <div className="flex min-w-0 flex-col gap-4">
        <PanelSkeleton title={64}>
          <PanelRowsSkeleton rows={3} />
        </PanelSkeleton>
        <PanelSkeleton title={84}>
          <PanelRowsSkeleton rows={3} />
        </PanelSkeleton>
      </div>
    </div>
  );
}

/** The body under the view switch, for the view the address names. */
function ExtractionBodySkeleton({ view }: { view: ExtractionView }) {
  switch (view) {
    case "accuracy":
      return (
        <>
          <AccuracyPanelSkeleton />
          <PanelSkeleton title={176} hint>
            <PanelRowsSkeleton rows={3} columns={6} />
          </PanelSkeleton>
        </>
      );
    case "corrections":
      return <DataTableSkeleton columnCount={CORRECTION_COLUMNS} aria-hidden />;
  }
}

/**
 * Document extraction while its code or its accuracy is read: the figures, the view switch
 * and the view the address names, each laid out where the loaded one lands.
 */
export function ExtractionViewSkeleton() {
  const [view] = useQueryState(EXTRACTION_VIEW_PARAM, extractionViewParser);

  return (
    <div className="flex min-w-0 flex-col gap-4" aria-busy="true">
      <KpiStripSkeleton count={EXTRACTION_FIGURES} />
      <div className="w-full max-w-md" aria-hidden>
        <Skeleton className="h-7.5 w-full rounded-lg" />
      </div>
      <ExtractionBodySkeleton view={view} />
    </div>
  );
}
