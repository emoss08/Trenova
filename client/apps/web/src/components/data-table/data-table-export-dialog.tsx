"use no memo";
import { useT } from "@trenova/shared/i18n/use-t";
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
import {
  buildCsv,
  buildExportColumns,
  downloadCsv,
  EXPORT_MAX_ROWS,
  exportFilename,
  fetchAllRows,
  type ExportScope,
} from "@/lib/data-table-export";
import type {
  DataTableGraphQLSource,
  DataTableQueryOptions,
  Table,
} from "@trenova/shared/types/data-table";
import { useRef, useState } from "react";
import { toast } from "sonner";
import { ChoiceButton } from "@/components/choice-button";

type DataTableExportDialogProps<TData extends Record<string, any>> = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  resource: string;
  table: Table<TData>;
  graphql: DataTableGraphQLSource<TData>;
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
  const t = useT();

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
      toast.error(t("Nothing to export"), {
        description: t("No exportable columns are available."),
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
                setProgress(
                  total != null
                    ? t("{0} of {1, plural, one {# row} other {# rows}}", fetched, total)
                    : t("{0, plural, one {# row} other {# rows}}", fetched),
                ),
              isCancelled: () => cancelledRef.current,
            });

      if (cancelledRef.current) return;

      downloadCsv(buildCsv(rows, exportColumns), exportFilename(resource));
      toast.success(t("Export complete"), {
        description: t(
          "{0, plural, one {Exported # row to CSV.} other {Exported # rows to CSV.}}",
          rows.length,
        ),
      });
      onOpenChange(false);
    } catch (error) {
      toast.error(t("Export failed"), {
        description: error instanceof Error ? error.message : t("An unexpected error occurred."),
      });
    } finally {
      setIsExporting(false);
      setProgress(null);
    }
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("Export to CSV")}</DialogTitle>
          <DialogDescription>
            {t("Exports respect the current filters, sorting, and column layout.")}
          </DialogDescription>
        </DialogHeader>
        <div className="flex flex-col gap-4 pb-2">
          <div className="flex flex-col gap-2">
            <Label className="text-muted-foreground text-xs font-medium">{t("Rows")}</Label>
            <div className="flex gap-2" role="radiogroup" aria-label={t("Export scope")}>
              <ChoiceButton selected={scope === "all"} onClick={() => setScope("all")}>
                <span className="font-medium">{t("All matching")}</span>
                {cappedTotal != null && (
                  <span className="text-muted-foreground text-xs">
                    {t("{0} rows", cappedTotal.toLocaleString())}
                  </span>
                )}
              </ChoiceButton>
              <ChoiceButton selected={scope === "page"} onClick={() => setScope("page")}>
                <span className="font-medium">{t("Current page")}</span>
                <span className="text-muted-foreground text-xs">
                  {t("{0} rows", currentPageRows.length)}
                </span>
              </ChoiceButton>
            </div>
            {totalCount != null && totalCount > EXPORT_MAX_ROWS && (
              <p className="text-muted-foreground text-xs">
                {t(
                  "Exports are capped at {0} rows. Narrow your filters to export a specific slice.",
                  EXPORT_MAX_ROWS.toLocaleString(),
                )}
              </p>
            )}
          </div>
          <div className="flex flex-col gap-2">
            <Label className="text-muted-foreground text-xs font-medium">{t("Columns")}</Label>
            <div className="flex gap-2" role="radiogroup" aria-label={t("Export columns")}>
              <ChoiceButton
                selected={columnsMode === "visible"}
                onClick={() => setColumnsMode("visible")}
              >
                <span className="font-medium">{t("Visible columns")}</span>
                <span className="text-muted-foreground text-xs">
                  {t("Matches the table layout")}
                </span>
              </ChoiceButton>
              <ChoiceButton selected={columnsMode === "all"} onClick={() => setColumnsMode("all")}>
                <span className="font-medium">{t("All columns")}</span>
                <span className="text-muted-foreground text-xs">{t("Every exportable field")}</span>
              </ChoiceButton>
            </div>
          </div>
        </div>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => handleOpenChange(false)}>
            {t("Cancel")}
          </Button>
          <Button
            type="button"
            onClick={handleExport}
            isLoading={isExporting}
            loadingText={progress ?? "Exporting..."}
          >
            {t("Export")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
