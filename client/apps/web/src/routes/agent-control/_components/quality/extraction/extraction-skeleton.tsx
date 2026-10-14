import { KpiStripSkeleton } from "@/components/kpi/kpi-strip-skeleton";
import { usePermission } from "@/hooks/use-permission";
import { DataTableSkeleton } from "@trenova/shared/components/data-table/data-table-skeleton";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQueryState } from "nuqs";
import type { ReactNode } from "react";
import {
  EXTRACTION_VIEW_PARAM,
  extractionViewParser,
  type ExtractionView,
} from "../../../ai-control-tabs";

/** What each extraction table shows at first. */
const TABLE_COLUMNS = { corrections: 7, cases: 5, runs: 8, shadow: 7 } as const;

/** Field accuracy, corrections, the evaluation set and the latest run. */
export const EXTRACTION_FIGURES = 4;
/** The candidate's figures beside production's, in the shadow and the rollout. */
export const COMPARISON_FIGURES = 5;

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

/** The candidate's settings panel, its figures and its field comparison. */
function ComparisonSkeleton({ guards }: { guards: boolean }) {
  return (
    <>
      <PanelSkeleton title={104}>
        <Skeleton className="m-3 h-10" />
      </PanelSkeleton>
      <KpiStripSkeleton count={COMPARISON_FIGURES} />
      {guards && (
        <PanelSkeleton title={56}>
          <PanelRowsSkeleton rows={2} />
        </PanelSkeleton>
      )}
      <PanelSkeleton title={112} hint>
        <PanelRowsSkeleton rows={4} />
      </PanelSkeleton>
    </>
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
    case "shadow":
      return (
        <>
          <ComparisonSkeleton guards={false} />
          <DataTableSkeleton columnCount={TABLE_COLUMNS.shadow} aria-hidden />
        </>
      );
    case "rollout":
      return <ComparisonSkeleton guards />;
    default:
      return <DataTableSkeleton columnCount={TABLE_COLUMNS[view]} aria-hidden />;
  }
}

/**
 * Document extraction while its code or its accuracy is read: the four figures, the view
 * switch and the new run beside it, and the view the address names, each laid out where
 * the loaded one lands.
 */
export function ExtractionViewSkeleton() {
  const [view] = useQueryState(EXTRACTION_VIEW_PARAM, extractionViewParser);
  const { allowed: canCreate } = usePermission(Resource.AgentEvalSuite, Operation.Create);

  return (
    <div className="flex min-w-0 flex-col gap-4" aria-busy="true">
      <KpiStripSkeleton count={EXTRACTION_FIGURES} />
      <div className="flex flex-wrap items-center justify-between gap-3" aria-hidden>
        <div className="w-full max-w-2xl">
          <Skeleton className="h-7.5 w-full rounded-lg" />
        </div>
        {canCreate && <Skeleton className="h-7 w-40" />}
      </div>
      <ExtractionBodySkeleton view={view} />
    </div>
  );
}
