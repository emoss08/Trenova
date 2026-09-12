import { useT } from "@trenova/shared/i18n/use-t";
import { DataTable } from "@/components/data-table/data-table";
import { fiscalYearTableGraphQLConfig, type FiscalYearRow } from "@/lib/graphql/fiscal-year-table";
import { AlertDialog } from "@trenova/shared/components/ui/alert-dialog";
import type { RowAction, Row } from "@trenova/shared/types/data-table";
import { Resource } from "@trenova/shared/types/permission";
import { PlayIcon, RotateCcwIcon, XCircleIcon } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import {
  FiscalYearActivateAlertDialogContent,
  FiscalYearCloseAlertDialogContent,
  FiscalYearReopenAlertDialogContent,
} from "./fiscal-year-alert-dialog-content";
import { getColumns } from "./fiscal-year-columns";
import { FiscalYearPanel } from "./fiscal-year-panel";

export type FiscalYearAction = "activate" | "close" | "reopen";

export default function FiscalYearTable() {
  const t = useT();

  const [selectedFiscalYear, setSelectedFiscalYear] = useState<FiscalYearRow | null>(null);
  const [yearAction, setYearAction] = useState<FiscalYearAction>("close");

  const handleYearAction = useCallback((fiscalYear: FiscalYearRow, action: FiscalYearAction) => {
    setSelectedFiscalYear(fiscalYear);
    setYearAction(action);
  }, []);

  const handleDialogClose = useCallback(() => setSelectedFiscalYear(null), []);

  const columns = useMemo(() => getColumns(t), [t]);

  const contextMenuActions = useMemo<RowAction<FiscalYearRow>[]>(
    () => [
      {
        id: "activate",
        label: t("Set as Current"),
        icon: PlayIcon,
        onClick: (row: Row<FiscalYearRow>) => handleYearAction(row.original, "activate"),
        hidden: (row: Row<FiscalYearRow>) => row.original.isCurrent,
      },
      {
        id: "close",
        label: t("Close Year"),
        icon: XCircleIcon,
        variant: "destructive",
        onClick: (row: Row<FiscalYearRow>) => handleYearAction(row.original, "close"),
        hidden: (row: Row<FiscalYearRow>) => row.original.status !== "Open",
      },
      {
        id: "reopen",
        label: t("Reopen Year"),
        icon: RotateCcwIcon,
        variant: "destructive",
        onClick: (row: Row<FiscalYearRow>) => handleYearAction(row.original, "reopen"),
        hidden: (row: Row<FiscalYearRow>) => row.original.status !== "Closed",
      },
    ],
    [handleYearAction, t],
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
          onClose={handleDialogClose}
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
  onClose,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  record: FiscalYearRow;
  action: FiscalYearAction;
  onClose: () => void;
}) {
  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <FiscalYearDialogContent action={action} record={record} onClose={onClose} />
    </AlertDialog>
  );
}

function FiscalYearDialogContent({
  action,
  record,
  onClose,
}: {
  action: FiscalYearAction;
  record?: FiscalYearRow;
  onClose: () => void;
}) {
  if (!record) return null;

  switch (action) {
    case "activate":
      return <FiscalYearActivateAlertDialogContent record={record} onClose={onClose} />;
    case "close":
      return <FiscalYearCloseAlertDialogContent record={record} onClose={onClose} />;
    case "reopen":
      return <FiscalYearReopenAlertDialogContent record={record} onClose={onClose} />;
    default:
      return null;
  }
}
