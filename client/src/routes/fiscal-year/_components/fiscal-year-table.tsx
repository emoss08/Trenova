import { DataTable } from "@/components/data-table/data-table";
import { AlertDialog } from "@/components/ui/alert-dialog";
import type { RowAction } from "@/types/data-table";
import type { FiscalYear } from "@/types/fiscal-year";
import { Resource } from "@/types/permission";
import type { Row } from "@tanstack/react-table";
import { LockIcon, PlayIcon, UnlockIcon, XCircleIcon } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import {
  FiscalYearActivateAlertDialogContent,
  FiscalYearCloseAlertDialogContent,
  FiscalYearLockAlertDialogContent,
  FiscalYearUnlockAlertDialogContent,
} from "./fiscal-year-alert-dialog-content";
import { getColumns } from "./fiscal-year-columns";
import { FiscalYearPanel } from "./fiscal-year-panel";

export type FiscalYearAction = "activate" | "close" | "lock" | "unlock";

export default function FiscalYearTable() {
  const [selectedFiscalYear, setSelectedFiscalYear] =
    useState<FiscalYear | null>(null);
  const [yearAction, setYearAction] = useState<FiscalYearAction>("close");

  const handleYearAction = useCallback(
    (fiscalYear: FiscalYear, action: FiscalYearAction) => {
      setSelectedFiscalYear(fiscalYear);
      setYearAction(action);
    },
    [],
  );

  const columns = useMemo(() => getColumns(), []);

  const contextMenuActions = useMemo<RowAction<FiscalYear>[]>(
    () => [
      {
        id: "activate",
        label: "Set as Current",
        icon: PlayIcon,
        onClick: (row: Row<FiscalYear>) =>
          handleYearAction(row.original, "activate"),
        hidden: (row: Row<FiscalYear>) =>
          row.original.isCurrent || row.original.status === "Locked",
      },
      {
        id: "close",
        label: "Close Year",
        icon: XCircleIcon,
        variant: "destructive",
        onClick: (row: Row<FiscalYear>) =>
          handleYearAction(row.original, "close"),
        hidden: (row: Row<FiscalYear>) => row.original.status !== "Open",
      },
      {
        id: "lock",
        label: "Lock Year",
        icon: LockIcon,
        onClick: (row: Row<FiscalYear>) =>
          handleYearAction(row.original, "lock"),
        hidden: (row: Row<FiscalYear>) => row.original.status !== "Closed",
      },
      {
        id: "unlock",
        label: "Unlock Year",
        icon: UnlockIcon,
        onClick: (row: Row<FiscalYear>) =>
          handleYearAction(row.original, "unlock"),
        hidden: (row: Row<FiscalYear>) => row.original.status !== "Locked",
      },
    ],
    [handleYearAction],
  );

  return (
    <>
      <DataTable<FiscalYear>
        name="Fiscal Year"
        link="/fiscal-years/"
        queryKey="fiscal-year-list"
        exportModelName="fiscal-year"
        resource={Resource.FiscalYear}
        columns={columns}
        contextMenuActions={contextMenuActions}
        extraSearchParams={{
          includePeriods: true,
        }}
        TablePanel={FiscalYearPanel}
      />
      {selectedFiscalYear && (
        <CloseAlertDialog
          open={true}
          onOpenChange={(open) => {
            if (!open) setSelectedFiscalYear(null);
          }}
          record={selectedFiscalYear}
          action={yearAction}
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
  record: FiscalYear;
  action: FiscalYearAction;
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
  action: FiscalYearAction;
  record?: FiscalYear;
}) {
  if (!record) return null;

  switch (action) {
    case "activate":
      return <FiscalYearActivateAlertDialogContent record={record} />;
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
