"use no memo";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Label } from "@trenova/shared/components/ui/label";
import { cn } from "@trenova/shared/lib/utils";
import {
  buildCsv,
  buildExportColumns,
  downloadCsv,
  EXPORT_MAX_ROWS,
  exportFilename,
  fetchAllRows,
  type ExportScope,
} from "@/lib/data-table-export";
import type { DataTableGraphQLConfig, DataTableQueryOptions } from "@trenova/shared/types/data-table";
import type { Table } from "@tanstack/react-table";
import { useRef, useState } from "react";
import { toast } from "sonner";

function ChoiceButton({
  selected,
  onClick,
  children,
}: {
  selected: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      role="radio"
      aria-checked={selected}
      onClick={onClick}
      className={cn(
        "flex flex-1 cursor-pointer flex-col items-start gap-0.5 rounded-md border px-3 py-2 text-left text-sm transition-colors",
        selected
          ? "border-primary bg-primary/5"
          : "border-border hover:border-muted-foreground/40 hover:bg-muted/40",
      )}
    >
      {children}
    </button>
  );
}

type DataTableExportDialogProps<TData extends Record<string, any>> = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  resource: string;
  table: Table<any>;
  graphql: DataTableGraphQLConfig<TData>;
  queryOptions: Omit<DataTableQueryOptions, "cursor">;
  currentPageRows: TData[];
  totalCount: number | null;
};

export default function DataTableExportDialog<TData extends Record<string, any>>({
  open,
  onOpenChange,
  resource,
  table,
  graphql,
  queryOptions,
  currentPageRows,
  totalCount,
}: DataTableExportDialogProps<TData>) {
  const [scope, setScope] = useState<ExportScope>("all");
  const [columnsMode, setColumnsMode] = useState<"visible" | "all">("visible");
  const [isExporting, setIsExporting] = useState(false);
  const [progress, setProgress] = useState<string | null>(null);
  const cancelledRef = useRef(false);

  const cappedTotal = totalCount != null ? Math.min(totalCount, EXPORT_MAX_ROWS) : null;

  const handleOpenChange = (nextOpen: boolean) => {
    if (!nextOpen) cancelledRef.current = true;
    onOpenChange(nextOpen);
  };

  const handleExport = async () => {
    const exportColumns = buildExportColumns(table.getAllLeafColumns(), columnsMode === "visible");
    if (exportColumns.length === 0) {
      toast.error("Nothing to export", {
        description: "No exportable columns are available.",
      });
      return;
    }

    cancelledRef.current = false;
    setIsExporting(true);
    setProgress(null);

    try {
      const rows =
        scope === "page"
          ? currentPageRows
          : await fetchAllRows<TData>({
              graphql,
              options: queryOptions,
              onProgress: ({ fetched, total }) =>
                setProgress(total != null ? `${fetched} of ${total} rows` : `${fetched} rows`),
              isCancelled: () => cancelledRef.current,
            });

      if (cancelledRef.current) return;

      downloadCsv(buildCsv(rows, exportColumns), exportFilename(resource));
      toast.success("Export complete", {
        description: `Exported ${rows.length} ${rows.length === 1 ? "row" : "rows"} to CSV.`,
      });
      onOpenChange(false);
    } catch (error) {
      toast.error("Export failed", {
        description: error instanceof Error ? error.message : "An unexpected error occurred.",
      });
    } finally {
      setIsExporting(false);
      setProgress(null);
    }
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-[420px]">
        <DialogHeader>
          <DialogTitle>Export to CSV</DialogTitle>
          <DialogDescription>
            Exports respect the current filters, sorting, and column layout.
          </DialogDescription>
        </DialogHeader>
        <div className="flex flex-col gap-4 pb-2">
          <div className="flex flex-col gap-2">
            <Label className="text-xs font-medium text-muted-foreground uppercase">Rows</Label>
            <div className="flex gap-2" role="radiogroup" aria-label="Export scope">
              <ChoiceButton selected={scope === "all"} onClick={() => setScope("all")}>
                <span className="font-medium">All matching</span>
                {cappedTotal != null && (
                  <span className="text-xs text-muted-foreground">
                    {cappedTotal.toLocaleString()} rows
                  </span>
                )}
              </ChoiceButton>
              <ChoiceButton selected={scope === "page"} onClick={() => setScope("page")}>
                <span className="font-medium">Current page</span>
                <span className="text-xs text-muted-foreground">{currentPageRows.length} rows</span>
              </ChoiceButton>
            </div>
            {totalCount != null && totalCount > EXPORT_MAX_ROWS && (
              <p className="text-xs text-muted-foreground">
                Exports are capped at {EXPORT_MAX_ROWS.toLocaleString()} rows. Narrow your filters
                to export a specific slice.
              </p>
            )}
          </div>
          <div className="flex flex-col gap-2">
            <Label className="text-xs font-medium text-muted-foreground uppercase">Columns</Label>
            <div className="flex gap-2" role="radiogroup" aria-label="Export columns">
              <ChoiceButton
                selected={columnsMode === "visible"}
                onClick={() => setColumnsMode("visible")}
              >
                <span className="font-medium">Visible columns</span>
                <span className="text-xs text-muted-foreground">Matches the table layout</span>
              </ChoiceButton>
              <ChoiceButton selected={columnsMode === "all"} onClick={() => setColumnsMode("all")}>
                <span className="font-medium">All columns</span>
                <span className="text-xs text-muted-foreground">Every exportable field</span>
              </ChoiceButton>
            </div>
          </div>
        </div>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => handleOpenChange(false)}>
            Cancel
          </Button>
          <Button
            type="button"
            onClick={handleExport}
            isLoading={isExporting}
            loadingText={progress ?? "Exporting..."}
          >
            Export
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
