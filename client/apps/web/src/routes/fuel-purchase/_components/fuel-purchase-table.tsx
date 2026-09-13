import { useT } from "@trenova/shared/i18n/use-t";
import { DataTable } from "@/components/data-table/data-table";
import {
  jurisdictionFilterOptions,
  useIftaJurisdictionOptions,
} from "@/lib/ifta-jurisdiction-options";
import { usePermission } from "@/hooks/use-permission";
import {
  FUEL_PURCHASE_LIST_KEY,
  fuelPurchaseTableGraphQLConfig,
  type FuelPurchaseRow,
} from "@/lib/graphql/fuel-purchase";
import { Button } from "@trenova/shared/components/ui/button";
import { EmptyTable } from "@trenova/shared/components/ui/empty-table";
import type {
  AddRecordAction,
  DataTableEmptyStateRenderProps,
  RowAction,
} from "@trenova/shared/types/data-table";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQueryClient } from "@tanstack/react-query";
import { FileSpreadsheetIcon, Trash2Icon } from "lucide-react";
import { useQueryState } from "nuqs";
import { useCallback, useMemo, useState } from "react";
import { DeleteFuelPurchaseDialog } from "./delete-fuel-purchase-dialog";
import { getColumns } from "./fuel-purchase-columns";
import { FuelPurchasePanel } from "./fuel-purchase-panel";
import { FuelPurchaseImportDialog } from "./import/fuel-purchase-import-dialog";

export const IMPORT_QUERY_PARAM = "import";
export const IMPORT_QUERY_VALUE = "fuel-card";

const EMPTY_COLUMNS = [
  { label: "Purchased" },
  { label: "Tractor" },
  { label: "Jurisdiction" },
  { label: "Vendor" },
  { label: "Fuel" },
  { label: "Gallons", numeric: true },
  { label: "Amount", numeric: true },
] as const;

function FuelPurchasesEmpty({
  hasActiveFilters,
  onClearFilters,
  canImport,
  onImport,
}: DataTableEmptyStateRenderProps & { canImport: boolean; onImport: () => void }) {
  const t = useT();

  return (
    <EmptyTable
      className="py-10"
      title={hasActiveFilters ? "Nothing matches" : "No fuel purchases yet"}
      description={
        hasActiveFilters
          ? "No fuel purchase fits the search and filters. Widen them, or clear them to see every one."
          : "A purchase arrives here two ways: keyed by hand from a receipt, or imported from a fuel card statement, where every row that reads becomes one. Each one credits its jurisdiction on the quarterly IFTA return."
      }
      columns={EMPTY_COLUMNS}
      onClearFilters={hasActiveFilters ? onClearFilters : undefined}
      action={
        canImport ? (
          <Button variant="outline" size="sm" onClick={onImport}>
            <FileSpreadsheetIcon className="size-3.5" />
            {t("Import card statement")}
          </Button>
        ) : null
      }
    />
  );
}

export default function FuelPurchaseTable() {
  const t = useT();

  const queryClient = useQueryClient();
  const { jurisdictions } = useIftaJurisdictionOptions();
  const columns = useMemo(
    () => getColumns(jurisdictionFilterOptions(jurisdictions), t),
    [jurisdictions, t],
  );
  const { allowed: canDelete } = usePermission(Resource.FuelPurchase, Operation.Delete);
  const { allowed: canImport } = usePermission(Resource.FuelPurchaseImport, Operation.Create);
  const [importParam, setImportParam] = useQueryState(IMPORT_QUERY_PARAM);
  const [deleting, setDeleting] = useState<FuelPurchaseRow | null>(null);

  const invalidate = useCallback(async () => {
    await queryClient.invalidateQueries({ queryKey: [FUEL_PURCHASE_LIST_KEY], refetchType: "all" });
  }, [queryClient]);

  const openImport = useCallback(() => {
    void setImportParam(IMPORT_QUERY_VALUE);
  }, [setImportParam]);

  const closeImport = useCallback(() => {
    void setImportParam(null);
  }, [setImportParam]);

  const addRecordActions = useMemo<AddRecordAction[]>(() => {
    if (!canImport) return [];
    return [
      {
        id: "import-statement",
        label: t("Import card statement"),
        description: t(
          "Upload a Comdata, EFS or WEX statement and review it before it is recorded.",
        ),
        icon: FileSpreadsheetIcon,
        onClick: openImport,
      },
    ];
  }, [canImport, openImport, t]);

  const contextMenuActions = useMemo<RowAction<FuelPurchaseRow>[]>(() => {
    if (!canDelete) return [];
    return [
      {
        id: "delete",
        label: t("Delete"),
        icon: Trash2Icon,
        variant: "destructive",
        onClick: (row) => setDeleting(row.original),
      },
    ];
  }, [canDelete, t]);

  return (
    <>
      <DataTable<FuelPurchaseRow>
        name="Fuel Purchase"
        queryKey={FUEL_PURCHASE_LIST_KEY}
        graphql={fuelPurchaseTableGraphQLConfig}
        resource={Resource.FuelPurchase}
        columns={columns}
        addRecordActions={addRecordActions}
        contextMenuActions={contextMenuActions}
        TablePanel={FuelPurchasePanel}
        renderEmptyState={(state) => (
          <FuelPurchasesEmpty {...state} canImport={canImport} onImport={openImport} />
        )}
      />
      <DeleteFuelPurchaseDialog
        open={deleting !== null}
        onOpenChange={(open) => {
          if (!open) setDeleting(null);
        }}
        purchase={deleting}
        onDeleted={invalidate}
      />
      {canImport ? (
        <FuelPurchaseImportDialog
          open={importParam !== null}
          onOpenChange={(open) => {
            if (!open) closeImport();
          }}
        />
      ) : null}
    </>
  );
}
