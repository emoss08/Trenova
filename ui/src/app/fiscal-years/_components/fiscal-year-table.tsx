import { DataTable } from "@/components/data-table/data-table";
import { AlertDialog } from "@/components/ui/alert-dialog";
import {
  FiscalYearSchema,
  FiscalYearStatusSchema,
} from "@/lib/schemas/fiscal-year-schema";
import { useFiscalYearPermissions } from "@/types/_gen/permissions";
import { Resource } from "@/types/audit-entry";
import { ContextMenuAction } from "@/types/data-table";
import { useMemo, useState } from "react";
import {
  FiscalYearCloseAlertDialogContent,
  FiscalYearLockAlertDialogContent,
  FiscalYearUnlockAlertDialogContent,
} from "./fiscal-year-alert-dialog-content";
import { getColumns } from "./fiscal-year-columns";
import { CreateFiscalYearModal } from "./fiscal-year-create-modal";
import { EditFiscalYearModal } from "./fiscal-year-edit-modal";

type Action = "close" | "lock" | "unlock";

export default function FiscalYearsDataTable() {
  const columns = useMemo(() => getColumns(), []);
  const { canActivate, canClose, canLock, canUnlock } =
    useFiscalYearPermissions();
  const [action, setAction] = useState<Action>("close");
  const [selectedYear, setSelectedYear] = useState<FiscalYearSchema | null>(
    null,
  );

  const contextMenuActions: ContextMenuAction<FiscalYearSchema>[] = useMemo(
    () => [
      {
        id: "activate",
        label: "Set as Current Year",
        onClick: (row) => {
          console.log(row);
        },
        disabled: (row) =>
          !canActivate ||
          row.original.isCurrent ||
          row.original.status === FiscalYearStatusSchema.enum.Locked,
      },
      {
        id: "close",
        label: "Close Fiscal Year",
        variant: "destructive",
        onClick: (row) => {
          setAction("close");
          setSelectedYear(row.original);
        },
        separator: "after",
        disabled: (row) =>
          !canClose || row.original.status !== FiscalYearStatusSchema.enum.Open,
      },
      {
        id: "lock",
        label: "Lock Fiscal Year",
        onClick: (row) => {
          setAction("lock");
          setSelectedYear(row.original);
        },
        disabled: (row) =>
          !canLock ||
          row.original.status !== FiscalYearStatusSchema.enum.Closed,
      },
      {
        id: "unlock",
        label: "Unlock Fiscal Year",
        onClick: (row) => {
          setAction("unlock");
          setSelectedYear(row.original);
        },
        disabled: (row) =>
          !canUnlock ||
          row.original.status !== FiscalYearStatusSchema.enum.Locked,
      },
    ],
    [canActivate, canClose, canLock, canUnlock],
  );

  return (
    <>
      <DataTable<FiscalYearSchema>
        resource={Resource.FiscalYear}
        name="Fiscal Year"
        link="/fiscal-years/"
        queryKey="fiscal-year-list"
        exportModelName="fiscal-year"
        TableModal={CreateFiscalYearModal}
        TableEditModal={EditFiscalYearModal}
        columns={columns}
        config={{
          enableFiltering: true,
          enableSorting: true,
          enableMultiSort: true,
          maxFilters: 5,
          maxSorts: 3,
          searchDebounce: 300,
          showFilterUI: true,
          showSortUI: true,
        }}
        useEnhancedBackend={true}
        contextMenuActions={contextMenuActions}
      />
      {selectedYear && (
        <CloseAlertDialog
          open={true}
          onOpenChange={() => setSelectedYear(null)}
          record={selectedYear}
          action={action}
        />
      )}
    </>
  );
}

function CloseAlertDialog({
  open,
  onOpenChange,
  record,
  action,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  record?: FiscalYearSchema;
  action: Action;
}) {
  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <FiscalYearDialogContent action={action} record={record} />
    </AlertDialog>
  );
}

function FiscalYearDialogContent({
  action,
  record,
}: {
  action: Action;
  record?: FiscalYearSchema;
}) {
  if (!record) return null;

  switch (action) {
    case "close":
      return <FiscalYearCloseAlertDialogContent record={record} />;
    case "lock":
      return <FiscalYearLockAlertDialogContent record={record} />;
    case "unlock":
      return <FiscalYearUnlockAlertDialogContent record={record} />;
    default:
      return null;
  }
}
