import { useT } from "@trenova/shared/i18n/use-t";
import { DataTable } from "@/components/data-table/data-table";
import { usePermissionCheck } from "@/hooks/use-permission";
import { getFiscalYearActions, type FiscalYearAction } from "@/lib/fiscal-year-actions";
import { fiscalYearTableGraphQLConfig, type FiscalYearRow } from "@/lib/graphql/fiscal-year-table";
import type { RowAction, Row } from "@trenova/shared/types/data-table";
import { Resource, type OperationType } from "@trenova/shared/types/permission";
import { PlayIcon, RotateCcwIcon, XCircleIcon } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { FiscalYearActionDialog } from "./fiscal-year-alert-dialog-content";
import { getColumns } from "./fiscal-year-columns";
import { FiscalYearPanel } from "./fiscal-year-panel";

export default function FiscalYearTable() {
  const t = useT();
  const { check } = usePermissionCheck();

  const [selectedFiscalYear, setSelectedFiscalYear] = useState<FiscalYearRow | null>(null);
  const [yearAction, setYearAction] = useState<FiscalYearAction>("close");

  const handleYearAction = useCallback((fiscalYear: FiscalYearRow, action: FiscalYearAction) => {
    setSelectedFiscalYear(fiscalYear);
    setYearAction(action);
  }, []);

  const handleDialogClose = useCallback(() => setSelectedFiscalYear(null), []);

  const columns = useMemo(() => getColumns(t), [t]);

  const isHidden = useCallback(
    (row: Row<FiscalYearRow>, action: FiscalYearAction) =>
      !getFiscalYearActions({
        status: row.original.status,
        isCurrent: row.original.isCurrent,
        can: (operation: OperationType) => check(Resource.FiscalYear, operation),
      }).includes(action),
    [check],
  );

  const contextMenuActions = useMemo<RowAction<FiscalYearRow>[]>(
    () => [
      {
        id: "activate",
        label: t("Set as Current"),
        icon: PlayIcon,
        onClick: (row: Row<FiscalYearRow>) => handleYearAction(row.original, "activate"),
        hidden: (row: Row<FiscalYearRow>) => isHidden(row, "activate"),
      },
      {
        id: "close",
        label: t("Close Year"),
        icon: XCircleIcon,
        variant: "destructive",
        onClick: (row: Row<FiscalYearRow>) => handleYearAction(row.original, "close"),
        hidden: (row: Row<FiscalYearRow>) => isHidden(row, "close"),
      },
      {
        id: "reopen",
        label: t("Reopen Year"),
        icon: RotateCcwIcon,
        variant: "destructive",
        onClick: (row: Row<FiscalYearRow>) => handleYearAction(row.original, "reopen"),
        hidden: (row: Row<FiscalYearRow>) => isHidden(row, "reopen"),
      },
    ],
    [handleYearAction, isHidden, t],
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
        <FiscalYearActionDialog
          open
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
