import { DataTable } from "@/components/data-table/data-table";
import {
  FiscalPeriodSchema,
  FiscalPeriodStatusSchema,
} from "@/lib/schemas/fiscal-period-schema";
import { useFiscalPeriodPermissions } from "@/types/_gen/permissions";
import { Resource } from "@/types/audit-entry";
import { ContextMenuAction } from "@/types/data-table";
import { useMemo, useState } from "react";
import { getColumns } from "./fiscal-period-columns";
import { CreateFiscalPeriodModal } from "./fiscal-period-create-modal";
import { EditFiscalPeriodModal } from "./fiscal-period-edit-modal";
import { FiscalPeriodStatusActions } from "./fiscal-period-status-actions";

type Action = "close" | "reopen" | "lock" | "unlock";

export default function FiscalPeriodsDataTable() {
  const columns = useMemo(() => getColumns(), []);
  const { canClose, canLock, canUnlock } = useFiscalPeriodPermissions();
  const [action, setAction] = useState<Action>("close");
  const [selectedPeriod, setSelectedPeriod] =
    useState<FiscalPeriodSchema | null>(null);

  const contextMenuActions: ContextMenuAction<FiscalPeriodSchema>[] = useMemo(
    () => [
      {
        id: "close",
        label: "Close Period",
        variant: "destructive",
        onClick: (row) => {
          setAction("close");
          setSelectedPeriod(row.original);
        },
        disabled: (row) =>
          !canClose ||
          row.original.status !== FiscalPeriodStatusSchema.enum.Open,
      },
      {
        id: "reopen",
        label: "Reopen Period",
        variant: "default",
        onClick: (row) => {
          setAction("reopen");
          setSelectedPeriod(row.original);
        },
        separator: "after",
        disabled: (row) =>
          !canClose ||
          row.original.status !== FiscalPeriodStatusSchema.enum.Closed,
      },
      {
        id: "lock",
        label: "Lock Period",
        onClick: (row) => {
          setAction("lock");
          setSelectedPeriod(row.original);
        },
        disabled: (row) =>
          !canLock ||
          row.original.status !== FiscalPeriodStatusSchema.enum.Closed,
      },
      {
        id: "unlock",
        label: "Unlock Period",
        onClick: (row) => {
          setAction("unlock");
          setSelectedPeriod(row.original);
        },
        disabled: (row) =>
          !canUnlock ||
          row.original.status !== FiscalPeriodStatusSchema.enum.Locked,
      },
    ],
    [canClose, canLock, canUnlock],
  );

  return (
    <>
      <DataTable<FiscalPeriodSchema>
        resource={Resource.FiscalPeriod}
        name="Fiscal Period"
        link="/fiscal-periods/"
        queryKey="fiscal-period-list"
        exportModelName="fiscal-period"
        TableModal={CreateFiscalPeriodModal}
        TableEditModal={EditFiscalPeriodModal}
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
      {selectedPeriod && (
        <FiscalPeriodStatusActions
          open={true}
          onOpenChange={(open) => !open && setSelectedPeriod(null)}
          record={selectedPeriod}
          action={action}
        />
      )}
    </>
  );
}
