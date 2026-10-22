"use no memo";
import { DataTable } from "@/components/data-table/data-table";
import { DEFAULT_CREATE_ACTION_ID } from "@/lib/data-table";
import { formulaLibraryRoutes } from "@/lib/formula-library-routes";
import {
  formulaLibraryTableGraphQLConfig,
  type FormulaLibraryRow,
} from "@/lib/graphql/formula-library";
import { invalidateFormulaLibrary } from "@/lib/queries/formula-library";
import { apiService } from "@/services/api";
import { useQueryClient } from "@tanstack/react-query";
import { ArchiveIcon, Copy01Icon, FilePlus02Icon } from "@trenova/shared/components/icons";
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
import { toast } from "@trenova/shared/components/ui/toast";
import { useT } from "@trenova/shared/i18n/use-t";
import type { AddRecordAction, DockAction, RowAction } from "@trenova/shared/types/data-table";
import { useCallback, useMemo, useState } from "react";
import { useNavigate } from "react-router";
import { archiveLibraries, type HeldArchive } from "./archive-libraries";
import { getColumns } from "./formula-library-columns";

export default function FormulaLibrariesDataTable() {
  const t = useT();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const columns = useMemo(() => getColumns(t), [t]);

  const [pendingArchive, setPendingArchive] = useState<FormulaLibraryRow[]>([]);
  const [held, setHeld] = useState<HeldArchive<FormulaLibraryRow>[]>([]);
  const [isArchiving, setIsArchiving] = useState(false);

  const duplicate = useCallback(
    async (rows: FormulaLibraryRow[]) => {
      const ids = rows.flatMap((row) => (row.id ? [row.id] : []));
      if (ids.length === 0) return;
      try {
        const copies = await apiService.formulaLibraryService.duplicate(ids);
        toast.success(
          t(
            "{0, plural, one {Duplicated # library as a draft} other {Duplicated # libraries as drafts}}",
            copies.length,
          ),
        );
        const [only] = copies;
        if (copies.length === 1 && only?.id) void navigate(formulaLibraryRoutes.edit(only.id));
      } catch {
        toast.error(t("The libraries could not be duplicated"));
      } finally {
        await invalidateFormulaLibrary(queryClient);
      }
    },
    [navigate, queryClient, t],
  );

  const closeArchive = useCallback(() => {
    setPendingArchive([]);
    setHeld([]);
  }, []);

  // Each library is archived on its own. One templates still import comes back asking for a
  // confirmation in the server's words; those wait in the dialog for "Archive anyway".
  const archive = useCallback(async () => {
    const confirming = held.length > 0;
    const rows = confirming ? held.map((entry) => entry.item) : pendingArchive;
    setIsArchiving(true);
    const outcome = await archiveLibraries(rows, confirming, (row, confirmInUse) =>
      apiService.formulaLibraryService.archive(row.id, { confirmInUse }),
    );
    setIsArchiving(false);
    await invalidateFormulaLibrary(queryClient);
    for (const message of outcome.forbidden) toast.error(message);
    if (outcome.archived > 0) {
      toast.success(
        t("{0, plural, one {Archived # library} other {Archived # libraries}}", outcome.archived),
      );
    }
    if (outcome.failed > outcome.forbidden.length) {
      toast.error(
        t(
          "{0, plural, one {# library was not archived} other {# libraries were not archived}}",
          outcome.failed - outcome.forbidden.length,
        ),
      );
    }
    const nextHeld = outcome.held;
    if (nextHeld.length > 0) {
      setHeld(nextHeld);
      return;
    }
    closeArchive();
  }, [closeArchive, held, pendingArchive, queryClient, t]);

  const contextMenuActions = useMemo<RowAction<FormulaLibraryRow>[]>(
    () => [
      {
        id: "duplicate",
        label: t("Duplicate"),
        icon: Copy01Icon,
        group: "actions",
        onClick: (row) => void duplicate([row.original]),
      },
      {
        id: "archive",
        label: t("Archive"),
        icon: ArchiveIcon,
        variant: "destructive",
        onClick: (row) => setPendingArchive([row.original]),
      },
    ],
    [duplicate, t],
  );

  const dockActions = useMemo<DockAction<FormulaLibraryRow>[]>(
    () => [
      {
        id: "duplicate",
        label: t("Duplicate"),
        icon: Copy01Icon,
        onClick: (rows) => void duplicate(rows),
      },
      {
        id: "archive",
        label: t("Archive"),
        icon: ArchiveIcon,
        variant: "destructive",
        onClick: (rows) => setPendingArchive(rows),
        clearSelectionOnSuccess: true,
      },
    ],
    [duplicate, t],
  );

  const addRecordActions = useMemo<AddRecordAction[]>(
    () => [
      {
        id: DEFAULT_CREATE_ACTION_ID,
        label: t("New formula library"),
        description: t("Write functions and constants your formula templates import."),
        icon: FilePlus02Icon,
        onClick: () => void navigate(formulaLibraryRoutes.new),
      },
    ],
    [navigate, t],
  );

  const archiveCount = held.length > 0 ? held.length : pendingArchive.length;

  return (
    <>
      <DataTable<FormulaLibraryRow>
        name="Formula Library"
        emptyTitle={t("No formula libraries yet")}
        queryKey="formula-library-list"
        graphql={formulaLibraryTableGraphQLConfig}
        columns={columns}
        enableRowSelection
        enableSelectionExport={false}
        dockActions={dockActions}
        contextMenuActions={contextMenuActions}
        onAddRecord={() => void navigate(formulaLibraryRoutes.new)}
        onRowClick={(row) => {
          if (row.original.id) void navigate(formulaLibraryRoutes.edit(row.original.id));
        }}
        addRecordActions={addRecordActions}
      />
      <AlertDialog
        open={archiveCount > 0}
        onOpenChange={(open) => {
          if (!open) closeArchive();
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle className="text-lg font-semibold">
              {t(
                "{0, plural, one {Archive # formula library?} other {Archive # formula libraries?}}",
                archiveCount,
              )}
            </AlertDialogTitle>
            <AlertDialogDescription>
              {held.length > 0
                ? held.map((entry) => entry.message).join(" ")
                : t(
                    "New template versions can no longer import an archived library. Template versions that already import it keep the version they pinned.",
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
              onClick={(event) => {
                event.preventDefault();
                void archive();
              }}
              disabled={isArchiving}
              isLoading={isArchiving}
            >
              {held.length > 0 ? t("Archive anyway") : t("Archive")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}
