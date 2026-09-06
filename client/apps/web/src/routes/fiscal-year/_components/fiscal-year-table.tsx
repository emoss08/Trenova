import { DataTable } from "@/components/data-table/data-table";
import { fiscalYearTableGraphQLConfig, type FiscalYearRow } from "@/lib/graphql/fiscal-year-table";
import { AlertDialog } from "@trenova/shared/components/ui/alert-dialog";
import type { RowAction, Row } from "@trenova/shared/types/data-table";
import { Resource } from "@trenova/shared/types/permission";
import { PlayIcon, XCircleIcon } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import {
  FiscalYearActivateAlertDialogContent,
  FiscalYearCloseAlertDialogContent,
} from "./fiscal-year-alert-dialog-content";
import { getColumns } from "./fiscal-year-columns";
import { FiscalYearPanel } from "./fiscal-year-panel";

export type FiscalYearAction = "activate" | "close" | "lock" | "unlock";

export default function FiscalYearTable() {
  const [selectedFiscalYear, setSelectedFiscalYear] = useState<FiscalYearRow | null>(null);
  const [yearAction, setYearAction] = useState<FiscalYearAction>("close");

  const handleYearAction = useCallback((fiscalYear: FiscalYearRow, action: FiscalYearAction) => {
    setSelectedFiscalYear(fiscalYear);
    setYearAction(action);
  }, []);

  const columns = useMemo(() => getColumns(), []);

  const contextMenuActions = useMemo<RowAction<FiscalYearRow>[]>(
    () => [
      {
        id: "activate",
        label: "Set as Current",
        icon: PlayIcon,
        onClick: (row: Row<FiscalYearRow>) => handleYearAction(row.original, "activate"),
        hidden: (row: Row<FiscalYearRow>) => row.original.isCurrent,
      },
      {
        id: "close",
        label: "Close Year",
        icon: XCircleIcon,
        variant: "destructive",
        onClick: (row: Row<FiscalYearRow>) => handleYearAction(row.original, "close"),
        hidden: (row: Row<FiscalYearRow>) => row.original.status !== "Open",
      },
    ],
    [handleYearAction],
  );

  return (
    <>
      <DataTable<FiscalYearRow>
        name="Fiscal Year"
        queryKey="fiscal-year-list"
        graphql={fiscalYearTableGraphQLConfig}
        resource={Resource.FiscalYear}
        columns={columns}
        contextMenuActions={contextMenuActions}
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
  record: FiscalYearRow;
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
  record?: FiscalYearRow;
}) {
  if (!record) return null;

  switch (action) {
    case "activate":
      return <FiscalYearActivateAlertDialogContent record={record} />;
    case "close":
      return <FiscalYearCloseAlertDialogContent record={record} />;
    default:
      return null;
  }
}
