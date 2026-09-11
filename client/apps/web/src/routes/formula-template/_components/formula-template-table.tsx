"use no memo";
import { useT } from "@trenova/shared/i18n/use-t";
import { DataTable } from "@/components/data-table/data-table";
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
import { apiService } from "@/services/api";
import { pluralize } from "@trenova/shared/lib/utils";
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
import type { DockAction, RowAction, Row } from "@trenova/shared/types/data-table";
import { useQueryClient } from "@tanstack/react-query";
import { ArchiveIcon, CopyIcon, DownloadIcon, GitForkIcon, NetworkIcon } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { useNavigate } from "react-router";
import { toast } from "sonner";
import { ExportTemplateDialog } from "./export-template-dialog";
import { ForkLineageDialog } from "./fork-lineage-dialog";
import { ForkTemplateDialog } from "./fork-template-dialog";
import { getColumns } from "./formula-template-columns";
import { ImportTemplateDialog } from "./studio/import-template-dialog";

export default function FormulaTemplatesDataTable() {
  const t = useT();

  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [importDialogOpen, setImportDialogOpen] = useState(false);
  const [installDialogOpen, setInstallDialogOpen] = useState(false);
  const [isInstalling, setIsInstalling] = useState(false);

  const [isDuplicateDialogOpen, setIsDuplicateDialogOpen] = useState(false);
  const [pendingDuplicateRows, setPendingDuplicateRows] = useState<FormulaTemplateRow[]>([]);
  const [isDuplicating, setIsDuplicating] = useState(false);

  const [pendingArchiveRows, setPendingArchiveRows] = useState<FormulaTemplateRow[]>([]);
  const [isArchiving, setIsArchiving] = useState(false);

  const [exportDialogTemplate, setExportDialogTemplate] = useState<FormulaTemplateRow | null>(null);

  const [forkDialogTemplate, setForkDialogTemplate] = useState<FormulaTemplateRow | null>(null);
  const [lineageDialogTemplate, setLineageDialogTemplate] = useState<FormulaTemplateRow | null>(
    null,
  );

  const handleExportClick = useCallback((template: FormulaTemplateRow) => {
    setExportDialogTemplate(template);
  }, []);

  const handleInstallStandards = useCallback(async () => {
    setIsInstalling(true);
    await apiService.formulaTemplateService
      .installStandards()
      .then((result) => {
        if (result.installed.length === 0) {
          toast.info(t("Standard templates already installed"), {
            description: `All ${result.skipped.length} standard templates exist in your organization.`,
          });
        } else {
          toast.success(
            `Installed ${result.installed.length} standard ${pluralize(
              "template",
              result.installed.length,
            )}`,
            {
              description:
                result.skipped.length > 0
                  ? `${result.skipped.length} already existed and ${
                      result.skipped.length === 1 ? "was" : "were"
                    } skipped.`
                  : "The standard rating library is ready to use.",
            },
          );
        }
        setInstallDialogOpen(false);
      })
      .catch(() => {
        toast.error(t("Failed to install standard templates"));
      })
      .finally(async () => {
        setIsInstalling(false);
        await invalidateFormulaTemplate(queryClient);
      });
  }, [queryClient, t]);

  const handleDuplicate = useCallback(
    (row: Row<FormulaTemplateRow>) => {
      const id = row.original.id;
      if (!id) return;

      toast.promise(
        apiService.formulaTemplateService.bulkDuplicate({
          templateIds: [id],
        }),
        {
          loading: "Duplicating template...",
          success: "Template duplicated successfully",
          error: "Failed to duplicate template",
          finally: async () => {
            await invalidateFormulaTemplate(queryClient);
          },
        },
      );
    },
    [queryClient],
  );

  const columns = useMemo(() => getColumns(), []);

  const requestArchive = useCallback((templates: FormulaTemplateRow[]) => {
    const withIds = templates.filter((template) => template.id);
    if (withIds.length === 0) {
      toast.error(t("No formula templates selected"));
      return;
    }

    setPendingArchiveRows(withIds);
  }, [t]);

  const handleConfirmArchive = useCallback(async () => {
    const ids = pendingArchiveRows.flatMap((template) => (template.id ? [template.id] : []));
    if (ids.length === 0) return;

    setIsArchiving(true);
    await apiService.formulaTemplateService
      .bulkUpdateStatus({
        templateIds: ids,
        status: "Inactive",
      })
      .then(() => {
        toast.success(
          ids.length === 1 ? "Formula template archived" : "Formula templates archived",
        );
        setPendingArchiveRows([]);
      })
      .catch(() => {
        toast.error(
          ids.length === 1
            ? "Failed to archive formula template"
            : "Failed to archive formula templates",
        );
      })
      .finally(async () => {
        setIsArchiving(false);
        await invalidateFormulaTemplate(queryClient);
      });
  }, [pendingArchiveRows, queryClient]);

  const contextMenuActions = useMemo<RowAction<FormulaTemplateRow>[]>(
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
        id: "archive",
        label: "Archive",
        icon: ArchiveIcon,
        variant: "destructive",
        onClick: (row) => requestArchive([row.original]),
      },
    ],
    [handleDuplicate, handleExportClick, requestArchive],
  );

  const handleBulkExport = useCallback(async (rows: FormulaTemplateRow[]) => {
    try {
      const [templates, testCaseLists] = await Promise.all([
        Promise.all(rows.map((row) => apiService.formulaTemplateService.get(row.id))),
        Promise.all(rows.map((row) => apiService.formulaTemplateService.listTestCases(row.id))),
      ]);
      const testCasesByTemplateId = Object.fromEntries(
        rows.map((row, index) => [row.id, testCaseLists[index]]),
      );

      const exportData = buildBulkExport(templates, testCasesByTemplateId);
      const filename = getBulkExportFilename();
      downloadJson(exportData, filename);
      toast.success(`Exported ${rows.length} templates`, {
        description: filename,
      });
    } catch {
      toast.error(t("Export failed"), {
        description: t("Could not export the selected templates. Please try again."),
      });
    }
  }, [t]);

  const handleBulkDuplicate = useCallback((rows: FormulaTemplateRow[]) => {
    setPendingDuplicateRows(rows);
    setIsDuplicateDialogOpen(true);
  }, []);

  const handleConfirmDuplicate = useCallback(async () => {
    const ids = pendingDuplicateRows.map((r) => r.id);
    setIsDuplicating(true);
    await apiService.formulaTemplateService
      .bulkDuplicate({
        templateIds: ids,
      })
      .then(() => {
        toast.success(t("Templates duplicated successfully"));
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
        label: "Duplicate",
        icon: CopyIcon,
        onClick: (rows) => handleBulkDuplicate(rows),
      },
      {
        id: "export",
        label: "Export",
        icon: DownloadIcon,
        onClick: (rows) => void handleBulkExport(rows),
      },
      {
        id: "archive",
        label: "Archive",
        icon: ArchiveIcon,
        variant: "destructive",
        onClick: requestArchive,
        clearSelectionOnSuccess: true,
      },
    ],
    [handleBulkExport, handleBulkDuplicate, requestArchive],
  );

  const archiveCount = pendingArchiveRows.length;

  return (
    <>
      <DataTable<FormulaTemplateRow>
        name="Formula Template"
        queryKey="formula-template-list"
        graphql={formulaTemplateTableGraphQLConfig}
        columns={columns}
        enableRowSelection
        dockActions={dockActions}
        contextMenuActions={contextMenuActions}
        onAddRecord={() => void navigate(formulaTemplateRoutes.new)}
        onRowClick={(row) => {
          if (row.original.id) void navigate(formulaTemplateRoutes.edit(row.original.id));
        }}
        addRecordActions={[
          {
            id: "install-standards",
            label: "Install Standard Templates",
            description: "Add the vetted standard rating library (per mile, per CWT, ...).",
            onClick: () => setInstallDialogOpen(true),
          },
          {
            id: "import-templates",
            label: "Import Templates",
            description: "Import templates from an exported JSON file.",
            onClick: () => setImportDialogOpen(true),
          },
        ]}
      />
      <AlertDialog open={installDialogOpen} onOpenChange={setInstallDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle className="text-lg font-semibold">
              {t("Install standard templates?")}
            </AlertDialogTitle>
            <AlertDialogDescription>
              {t("This adds Trenova's vetted standard rating templates (Flat Rate, Per Mile, Per CWT, and more) to your organization as Active templates. Templates you already have are left untouched.")}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel variant="outline" size="default">
              {t("Cancel")}
            </AlertDialogCancel>
            <AlertDialogAction
              size="default"
              onClick={() => void handleInstallStandards()}
              disabled={isInstalling}
              isLoading={isInstalling}
            >
              {t("Install")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
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
          if (!open) setPendingArchiveRows([]);
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle className="text-lg font-semibold">
              {t("Archive {0} formula {1}?", archiveCount, pluralize("template", archiveCount))}
            </AlertDialogTitle>
            <AlertDialogDescription>
              {t("Archived templates are marked inactive and stop pricing new shipments. Rate agreements and shipments referencing them keep their history.")}
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
              {t("Archive")}
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
