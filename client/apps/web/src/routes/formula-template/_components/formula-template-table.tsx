"use no memo";
import { useT } from "@trenova/shared/i18n/use-t";
import { DataTable } from "@/components/data-table/data-table";
import { DEFAULT_CREATE_ACTION_ID } from "@/lib/data-table";
import { DuplicateAlertDialog } from "@/components/duplicate-alert-dialog";
import {
  formulaTemplateTableGraphQLConfig,
  type FormulaTemplateRow,
} from "@/lib/graphql/formula-template-table";
import {
  buildBulkExport,
  downloadJson,
  getBulkExportFilename,
} from "@/lib/formula-template-export";
import { formulaTemplateRoutes, importLandingRoute } from "@/lib/formula-template-routes";
import { invalidateFormulaTemplate } from "@/lib/queries/formula-template";
import { formulaTemplateService } from "@/services/formula-template";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@trenova/shared/components/ui/alert-dialog";
import type { AddRecordAction, DockAction, RowAction, Row } from "@trenova/shared/types/data-table";
import { useQueryClient } from "@tanstack/react-query";
import {
  ArchiveIcon,
  Copy01Icon,
  Dataflow04Icon,
  Download01Icon,
  FilePlus02Icon,
  FileUploadIcon,
  GitBranch02Icon,
} from "@trenova/shared/components/icons";
import { useCallback, useMemo, useState } from "react";
import { useNavigate } from "react-router";
import { toast } from "@trenova/shared/components/ui/toast";
import { archiveRefusal } from "@/lib/archive-refusal";
import { ExportTemplateDialog } from "./export-template-dialog";
import { ForkLineageDialog } from "./fork-lineage-dialog";
import { ForkTemplateDialog } from "./fork-template-dialog";
import { getColumns } from "./formula-template-columns";
import { ImportTemplateDialog } from "./studio/import-template-dialog";
import { translate } from "@trenova/shared/i18n/runtime";

export default function FormulaTemplatesDataTable() {
  const t = useT();

  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [importDialogOpen, setImportDialogOpen] = useState(false);

  const [isDuplicateDialogOpen, setIsDuplicateDialogOpen] = useState(false);
  const [pendingDuplicateRows, setPendingDuplicateRows] = useState<FormulaTemplateRow[]>([]);
  const [isDuplicating, setIsDuplicating] = useState(false);

  const [pendingArchiveRows, setPendingArchiveRows] = useState<FormulaTemplateRow[]>([]);
  const [archiveInUse, setArchiveInUse] = useState<string | null>(null);
  const [isArchiving, setIsArchiving] = useState(false);

  const [exportDialogTemplate, setExportDialogTemplate] = useState<FormulaTemplateRow | null>(null);

  const [forkDialogTemplate, setForkDialogTemplate] = useState<FormulaTemplateRow | null>(null);
  const [lineageDialogTemplate, setLineageDialogTemplate] = useState<FormulaTemplateRow | null>(
    null,
  );

  const handleExportClick = useCallback((template: FormulaTemplateRow) => {
    setExportDialogTemplate(template);
  }, []);

  const handleDuplicate = useCallback(
    (row: Row<FormulaTemplateRow>) => {
      const id = row.original.id;
      if (!id) return;

      toast.promise(
        formulaTemplateService.bulkDuplicate({
          templateIds: [id],
        }),
        {
          loading: translate("Duplicating template..."),
          success: translate("Template duplicated successfully"),
          error: translate("Failed to duplicate template"),
          finally: async () => {
            await invalidateFormulaTemplate(queryClient);
          },
        },
      );
    },
    [queryClient],
  );

  const columns = useMemo(() => getColumns(t), [t]);

  const requestArchive = useCallback(
    (templates: FormulaTemplateRow[]) => {
      const withIds = templates.filter((template) => template.id);
      if (withIds.length === 0) {
        toast.error(t("No formula templates selected"));
        return;
      }

      setArchiveInUse(null);
      setPendingArchiveRows(withIds);
    },
    [t],
  );

  const closeArchiveDialog = useCallback(() => {
    setPendingArchiveRows([]);
    setArchiveInUse(null);
  }, []);

  const handleConfirmArchive = useCallback(async () => {
    const ids = pendingArchiveRows.flatMap((template) => (template.id ? [template.id] : []));
    if (ids.length === 0) return;

    setIsArchiving(true);
    await formulaTemplateService
      .bulkUpdateStatus({
        templateIds: ids,
        status: "Inactive",
        confirmInUse: archiveInUse !== null,
      })
      .then(() => {
        closeArchiveDialog();
      })
      .catch((error: unknown) => {
        const refusal = archiveRefusal(error);
        if (refusal?.kind === "needsConfirmation") {
          setArchiveInUse(refusal.message);
          return;
        }
        if (refusal?.kind === "forbidden") {
          toast.error(refusal.message);
          return;
        }
        toast.error(
          ids.length === 1
            ? translate("Failed to archive formula template")
            : translate("Failed to archive formula templates"),
        );
      })
      .finally(async () => {
        setIsArchiving(false);
        await invalidateFormulaTemplate(queryClient);
      });
  }, [archiveInUse, closeArchiveDialog, pendingArchiveRows, queryClient]);

  const contextMenuActions = useMemo<RowAction<FormulaTemplateRow>[]>(
    () => [
      {
        id: "fork",
        label: t("Fork template"),
        icon: GitBranch02Icon,
        group: { id: "fork", label: t("Fork") },
        onClick: (row) => setForkDialogTemplate(row.original),
      },
      {
        id: "lineage",
        label: t("View lineage"),
        icon: Dataflow04Icon,
        group: { id: "fork", label: t("Fork") },
        onClick: (row) => setLineageDialogTemplate(row.original),
      },
      {
        id: "duplicate",
        label: t("Duplicate"),
        icon: Copy01Icon,
        group: "actions",
        onClick: handleDuplicate,
      },
      {
        id: "export",
        label: t("Export"),
        icon: Download01Icon,
        group: "actions",
        onClick: (row) => handleExportClick(row.original),
      },
      {
        id: "archive",
        label: t("Archive"),
        icon: ArchiveIcon,
        variant: "destructive",
        onClick: (row) => requestArchive([row.original]),
      },
    ],
    [handleDuplicate, handleExportClick, requestArchive, t],
  );

  const handleBulkExport = useCallback(
    async (rows: FormulaTemplateRow[]) => {
      try {
        const templates = await Promise.all(rows.map((row) => formulaTemplateService.get(row.id)));
        const exportData = buildBulkExport(templates);
        const filename = getBulkExportFilename();
        downloadJson(exportData, filename);
      } catch {
        toast.error(t("Export failed"), {
          description: t("Could not export the selected templates. Please try again."),
        });
      }
    },
    [t],
  );

  const handleBulkDuplicate = useCallback((rows: FormulaTemplateRow[]) => {
    setPendingDuplicateRows(rows);
    setIsDuplicateDialogOpen(true);
  }, []);

  const handleConfirmDuplicate = useCallback(async () => {
    const ids = pendingDuplicateRows.map((r) => r.id);
    setIsDuplicating(true);
    await formulaTemplateService
      .bulkDuplicate({
        templateIds: ids,
      })
      .then(() => {
        setIsDuplicateDialogOpen(false);
        setPendingDuplicateRows([]);
      })
      .catch(() => {
        toast.error(t("Failed to duplicate templates"));
      })
      .finally(async () => {
        setIsDuplicating(false);
        await invalidateFormulaTemplate(queryClient);
      });
  }, [pendingDuplicateRows, queryClient, t]);

  const dockActions = useMemo<DockAction<FormulaTemplateRow>[]>(
    () => [
      {
        id: "duplicate",
        label: t("Duplicate"),
        icon: Copy01Icon,
        onClick: (rows) => handleBulkDuplicate(rows),
      },
      {
        id: "export",
        label: t("Export"),
        icon: Download01Icon,
        onClick: (rows) => void handleBulkExport(rows),
      },
      {
        id: "archive",
        label: t("Archive"),
        icon: ArchiveIcon,
        variant: "destructive",
        onClick: requestArchive,
        clearSelectionOnSuccess: true,
      },
    ],
    [handleBulkExport, handleBulkDuplicate, requestArchive, t],
  );

  const addRecordActions = useMemo<AddRecordAction[]>(
    () => [
      {
        id: DEFAULT_CREATE_ACTION_ID,
        label: t("New formula template"),
        description: t("Write a rating formula from a blank template in the studio."),
        icon: FilePlus02Icon,
        onClick: () => void navigate(formulaTemplateRoutes.new),
      },
      {
        id: "import-templates",
        label: t("Import templates"),
        description: t("Bring in templates from a JSON file exported from Trenova."),
        icon: FileUploadIcon,
        onClick: () => setImportDialogOpen(true),
      },
    ],
    [navigate, t],
  );

  const archiveCount = pendingArchiveRows.length;

  return (
    <>
      <DataTable<FormulaTemplateRow>
        name="Formula Template"
        emptyTitle={t("No formula templates yet")}
        queryKey="formula-template-list"
        graphql={formulaTemplateTableGraphQLConfig}
        columns={columns}
        enableRowSelection
        enableSelectionExport={false}
        dockActions={dockActions}
        contextMenuActions={contextMenuActions}
        onAddRecord={() => void navigate(formulaTemplateRoutes.new)}
        onRowClick={(row) => {
          if (row.original.id) void navigate(formulaTemplateRoutes.edit(row.original.id));
        }}
        addRecordActions={addRecordActions}
      />
      <ImportTemplateDialog
        open={importDialogOpen}
        onOpenChange={setImportDialogOpen}
        onImported={(response) => void navigate(importLandingRoute(response))}
      />
      <DuplicateAlertDialog
        open={isDuplicateDialogOpen}
        onOpenChange={setIsDuplicateDialogOpen}
        rowCount={pendingDuplicateRows.length}
        onConfirm={handleConfirmDuplicate}
        isLoading={isDuplicating}
      />
      <AlertDialog
        open={archiveCount > 0}
        onOpenChange={(open) => {
          if (!open) closeArchiveDialog();
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle className="text-lg font-semibold">
              {t(
                "{0, plural, one {Archive # formula template?} other {Archive # formula templates?}}",
                archiveCount,
              )}
            </AlertDialogTitle>
            <AlertDialogDescription>
              {archiveInUse ??
                t(
                  "Archived templates are marked inactive and stop pricing new shipments. Rate agreements and shipments referencing them keep their history.",
                )}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel variant="outline" size="default">
              {t("Cancel")}
            </AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              size="default"
              onClick={() => void handleConfirmArchive()}
              disabled={isArchiving}
              isLoading={isArchiving}
            >
              {archiveInUse === null ? t("Archive") : t("Archive anyway")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
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
        onForkSuccess={(forked) => {
          setForkDialogTemplate(null);
          if (forked.id) void navigate(formulaTemplateRoutes.edit(forked.id));
        }}
      />
      <ForkLineageDialog
        open={lineageDialogTemplate !== null}
        onOpenChange={(open) => {
          if (!open) setLineageDialogTemplate(null);
        }}
        templateId={lineageDialogTemplate?.id}
        currentTemplateId={lineageDialogTemplate?.id}
        onNavigateToTemplate={(templateId) => {
          setLineageDialogTemplate(null);
          if (templateId) void navigate(formulaTemplateRoutes.edit(templateId));
        }}
      />
    </>
  );
}
