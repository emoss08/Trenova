"use no memo";
import { DataTable } from "@/components/data-table/data-table";
import { DuplicateAlertDialog } from "@/components/duplicate-alert-dialog";
import { statusChoices } from "@/lib/choices";
import {
  buildBulkExport,
  downloadJson,
  getBulkExportFilename,
} from "@/lib/formula-template-export";
import { apiService } from "@/services/api";
import type { DockAction, RowAction } from "@/types/data-table";
import type { FormulaTemplate } from "@/types/formula-template";
import { useQueryClient } from "@tanstack/react-query";
import type { Row } from "@tanstack/react-table";
import {
  CircleCheckIcon,
  CopyIcon,
  DownloadIcon,
  GitForkIcon,
  NetworkIcon,
  TrashIcon,
} from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { toast } from "sonner";
import { ExportTemplateDialog } from "./export-template-dialog";
import { ForkLineageDialog } from "./fork-lineage-dialog";
import { ForkTemplateDialog } from "./fork-template-dialog";
import { getColumns } from "./formula-template-columns";
import { FormulaTemplatePanel } from "./formula-template-panel";

export default function FormulaTemplatesDataTable() {
  const queryClient = useQueryClient();

  const [isDuplicateDialogOpen, setIsDuplicateDialogOpen] = useState(false);
  const [pendingDuplicateRows, setPendingDuplicateRows] = useState<
    FormulaTemplate[]
  >([]);
  const [isDuplicating, setIsDuplicating] = useState(false);

  const [exportDialogTemplate, setExportDialogTemplate] =
    useState<FormulaTemplate | null>(null);

  const [forkDialogTemplate, setForkDialogTemplate] =
    useState<FormulaTemplate | null>(null);
  const [lineageDialogTemplate, setLineageDialogTemplate] =
    useState<FormulaTemplate | null>(null);

  const handleExportClick = useCallback((template: FormulaTemplate) => {
    setExportDialogTemplate(template);
  }, []);

  const handleDuplicate = useCallback(
    (row: Row<FormulaTemplate>) => {
      const id = row.original.id;
      if (!id) return;

      toast.promise(
        apiService.formulaTemplateService.bulkDuplicate({
          templateIds: [id] as string[],
        }),
        {
          loading: "Duplicating template...",
          success: "Template duplicated successfully",
          error: "Failed to duplicate template",
          finally: async () => {
            await queryClient.invalidateQueries({
              queryKey: ["formula-template-list"],
              refetchType: "all",
            });
          },
        },
      );
    },
    [queryClient],
  );

  const columns = useMemo(() => getColumns(), []);

  const contextMenuActions = useMemo<RowAction<FormulaTemplate>[]>(
    () => [
      {
        id: "fork",
        label: "Fork Template",
        icon: GitForkIcon,
        group: { id: "fork", label: "Fork" },
        onClick: (row) => setForkDialogTemplate(row.original),
      },
      {
        id: "lineage",
        label: "View Lineage",
        icon: NetworkIcon,
        group: { id: "fork", label: "Fork" },
        onClick: (row) => setLineageDialogTemplate(row.original),
      },
      {
        id: "duplicate",
        label: "Duplicate",
        icon: CopyIcon,
        group: "actions",
        onClick: handleDuplicate,
      },
      {
        id: "export",
        label: "Export",
        icon: DownloadIcon,
        group: "actions",
        onClick: (row) => handleExportClick(row.original),
      },
      {
        id: "delete",
        label: "Delete",
        icon: TrashIcon,
        variant: "destructive",
        onClick: (row) => {
          console.log("Delete", row.original);
        },
      },
    ],
    [handleDuplicate, handleExportClick],
  );

  const handleBulkDelete = useCallback((rows: FormulaTemplate[]) => {
    const ids = rows.map((r) => r.id);
    console.log("Delete templates:", ids);
  }, []);

  const handleBulkExport = useCallback((rows: FormulaTemplate[]) => {
    const exportData = buildBulkExport(rows);
    const filename = getBulkExportFilename();
    downloadJson(exportData, filename);
    toast.success(`Exported ${rows.length} templates`, {
      description: filename,
    });
  }, []);

  const handleBulkDuplicate = useCallback((rows: FormulaTemplate[]) => {
    setPendingDuplicateRows(rows);
    setIsDuplicateDialogOpen(true);
  }, []);

  const handleConfirmDuplicate = useCallback(async () => {
    const ids = pendingDuplicateRows.map((r) => r.id);
    setIsDuplicating(true);
    await apiService.formulaTemplateService
      .bulkDuplicate({
        templateIds: ids as string[],
      })
      .then(() => {
        toast.success("Templates duplicated successfully");
        setIsDuplicateDialogOpen(false);
        setPendingDuplicateRows([]);
      })
      .catch(() => {
        toast.error("Failed to duplicate templates");
      })
      .finally(async () => {
        setIsDuplicating(false);
        await queryClient.invalidateQueries({
          queryKey: ["formula-template-list"],
          refetchType: "all",
        });
      });
  }, [pendingDuplicateRows, queryClient]);

  const handleBulkStatusUpdate = useCallback(
    async (rows: FormulaTemplate[], status: string) => {
      const ids = rows.map((r) => r.id);
      toast.promise(
        apiService.formulaTemplateService.bulkUpdateStatus({
          templateIds: ids as string[],
          status: status as FormulaTemplate["status"],
        }),
        {
          loading: "Updating status...",
          success: "Status updated successfully",
          error: "Failed to update status",
          finally: async () => {
            await queryClient.invalidateQueries({
              queryKey: ["formula-template-list"],
              refetchType: "all",
            });
          },
        },
      );
    },
    [queryClient],
  );

  const dockActions = useMemo<DockAction<FormulaTemplate>[]>(
    () => [
      {
        id: "status-update",
        type: "select",
        label: "Update Status",
        loadingLabel: "Updating...",
        icon: CircleCheckIcon,
        options: statusChoices,
        onSelect: handleBulkStatusUpdate,
        clearSelectionOnSuccess: true,
      },
      {
        id: "duplicate",
        label: "Duplicate",
        icon: CopyIcon,
        onClick: (rows) => handleBulkDuplicate(rows),
      },
      {
        id: "export",
        label: "Export",
        icon: DownloadIcon,
        onClick: handleBulkExport,
      },
      {
        id: "delete",
        label: "Delete",
        icon: TrashIcon,
        variant: "destructive",
        onClick: handleBulkDelete,
      },
    ],
    [
      handleBulkDelete,
      handleBulkExport,
      handleBulkDuplicate,
      handleBulkStatusUpdate,
    ],
  );

  return (
    <>
      <DataTable<FormulaTemplate>
        name="Formula Template"
        link="/formula-templates/"
        queryKey="formula-template-list"
        exportModelName="formula-template"
        columns={columns}
        enableRowSelection
        dockActions={dockActions}
        contextMenuActions={contextMenuActions}
        TablePanel={FormulaTemplatePanel}
      />
      <DuplicateAlertDialog
        open={isDuplicateDialogOpen}
        onOpenChange={setIsDuplicateDialogOpen}
        rowCount={pendingDuplicateRows.length}
        onConfirm={handleConfirmDuplicate}
        isLoading={isDuplicating}
      />
      <ExportTemplateDialog
        open={exportDialogTemplate !== null}
        onOpenChange={(open) => {
          if (!open) setExportDialogTemplate(null);
        }}
        template={exportDialogTemplate}
      />
      <ForkTemplateDialog
        open={forkDialogTemplate !== null}
        onOpenChange={(open) => {
          if (!open) setForkDialogTemplate(null);
        }}
        template={forkDialogTemplate}
        onForkSuccess={() => setForkDialogTemplate(null)}
      />
      <ForkLineageDialog
        open={lineageDialogTemplate !== null}
        onOpenChange={(open) => {
          if (!open) setLineageDialogTemplate(null);
        }}
        templateId={lineageDialogTemplate?.id}
        currentTemplateId={lineageDialogTemplate?.id}
      />
    </>
  );
}
