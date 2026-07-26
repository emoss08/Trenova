import { Badge } from "@trenova/shared/components/ui/badge";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@trenova/shared/components/ui/sheet";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useReportDrillThrough } from "@/hooks/use-reports";
import { graphQLErrorMessage } from "@trenova/shared/lib/graphql";
import type { ReportIrInput } from "@/lib/graphql/reports";
import { CircleAlertIcon } from "lucide-react";
import { useMemo } from "react";
import { ResultGrid } from "./result-grid";

/** Identifies the aggregate cell that was clicked. */
export type DrillTarget = {
  /** Grouping column values that identify the row, keyed by column id. */
  dimensions: { columnId: string; value: unknown }[];
  /** The output column that was clicked, so the drill inherits its scope. */
  columnId?: string;
  /** What the cell said, shown as context in the sheet. */
  columnLabel?: string;
  rowLabel?: string;
};

type DrillThroughSheetProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  definition: ReportIrInput | null;
  params?: Record<string, unknown>;
  target: DrillTarget | null;
};

const DRILL_LIMIT = 200;

export function DrillThroughSheet({
  open,
  onOpenChange,
  definition,
  params,
  target,
}: DrillThroughSheetProps) {
  const input = useMemo(() => {
    if (!open || !definition || !target) return null;
    return {
      definition,
      params,
      dimensions: target.dimensions,
      columnId: target.columnId,
      limit: DRILL_LIMIT,
    };
  }, [open, definition, params, target]);

  const drill = useReportDrillThrough(input);
  const rows = Array.isArray(drill.data?.rows) ? (drill.data.rows as unknown[][]) : [];

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent
        side="right"
        className="w-full sm:max-w-3xl"
        aria-describedby="drill-description"
      >
        <SheetHeader>
          <SheetTitle>Records behind this number</SheetTitle>
          <SheetDescription id="drill-description">
            {target?.rowLabel && target?.columnLabel
              ? `${target.columnLabel} for ${target.rowLabel}`
              : "The individual records this aggregate was built from."}
          </SheetDescription>
        </SheetHeader>

        <div className="flex min-h-0 flex-1 flex-col px-4 pb-4">
          {drill.isError ? (
            <div className="flex flex-1 flex-col items-center justify-center gap-2 text-center">
              <CircleAlertIcon className="size-5 text-destructive" />
              <p className="max-w-md text-xs whitespace-pre-wrap text-muted-foreground">
                {graphQLErrorMessage(drill.error, "These records could not be loaded")}
              </p>
            </div>
          ) : drill.isPending ? (
            <div className="flex flex-col gap-1.5 py-2">
              {Array.from({ length: 12 }, (_, i) => (
                <Skeleton key={i} className="h-6" style={{ opacity: 1 - i * 0.07 }} />
              ))}
            </div>
          ) : (
            <>
              <div className="flex h-8 shrink-0 items-center gap-2">
                <Badge variant="secondary">
                  {rows.length} record{rows.length === 1 ? "" : "s"}
                </Badge>
                {drill.data?.truncated && (
                  <span className="text-2xs text-amber-600 dark:text-amber-400">
                    first {DRILL_LIMIT} shown
                  </span>
                )}
              </div>
              <div className="flex min-h-0 flex-1 flex-col overflow-hidden rounded-md border border-border">
                <ResultGrid
                  columns={drill.data?.columns ?? []}
                  rows={rows}
                  density="compact"
                  emptyMessage="No records matched this cell."
                />
              </div>
            </>
          )}
        </div>
      </SheetContent>
    </Sheet>
  );
}
