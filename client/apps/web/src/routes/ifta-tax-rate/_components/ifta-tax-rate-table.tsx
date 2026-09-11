import { useT } from "@trenova/shared/i18n/use-t";
import { DataTable } from "@/components/data-table/data-table";
import {
  jurisdictionFilterOptions,
  useIftaJurisdictionOptions,
} from "@/components/fields/ifta-jurisdiction-select-field";
import { usePermission } from "@/hooks/use-permission";
import {
  IFTA_TAX_RATE_LIST_KEY,
  iftaTaxRateTableGraphQLConfig,
  type IftaTaxRateRow,
} from "@/lib/graphql/ifta-tax-rate";
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
import { useCallback, useMemo, useState } from "react";
import { DeleteIftaTaxRateDialog } from "./delete-ifta-tax-rate-dialog";
import { getColumns } from "./ifta-tax-rate-columns";
import { IftaTaxRatePanel } from "./ifta-tax-rate-panel";
import { ImportIftaTaxRatesDialog } from "./import-ifta-tax-rates-dialog";

const EMPTY_COLUMNS = [
  { label: "Year", numeric: true },
  { label: "Quarter" },
  { label: "Jurisdiction" },
  { label: "Fuel" },
  { label: "Rate / gal", numeric: true },
  { label: "Surcharge / gal", numeric: true },
] as const;

function IftaTaxRatesEmpty({
  hasActiveFilters,
  onClearFilters,
  canImport,
  onImport,
}: DataTableEmptyStateRenderProps & { canImport: boolean; onImport: () => void }) {
  const t = useT();

  return (
    <EmptyTable
      className="py-10"
      title={hasActiveFilters ? "Nothing matches" : "No tax rates yet"}
      description={
        hasActiveFilters
          ? "No rate fits the search and filters. Widen them, or clear them to see every one."
          : "A rate arrives here when the quarter's matrix is imported from a CSV or a single rate is published by hand. Until a jurisdiction has one, every return for that quarter reports it as missing and cannot be finalized."
      }
      columns={EMPTY_COLUMNS}
      onClearFilters={hasActiveFilters ? onClearFilters : undefined}
      action={
        canImport ? (
          <Button variant="outline" size="sm" onClick={onImport}>
            <FileSpreadsheetIcon className="size-3.5" />
            {t("Import rates")}
          </Button>
        ) : null
      }
    />
  );
}

export default function IftaTaxRateTable() {
  const queryClient = useQueryClient();
  const { jurisdictions } = useIftaJurisdictionOptions();
  const columns = useMemo(
    () => getColumns(jurisdictionFilterOptions(jurisdictions)),
    [jurisdictions],
  );
  const { allowed: canImport } = usePermission(Resource.IFTATaxRate, Operation.Create);
  const { allowed: canDelete } = usePermission(Resource.IFTATaxRate, Operation.Delete);
  const [importOpen, setImportOpen] = useState(false);
  const [deleting, setDeleting] = useState<IftaTaxRateRow | null>(null);

  const invalidate = useCallback(async () => {
    await queryClient.invalidateQueries({ queryKey: [IFTA_TAX_RATE_LIST_KEY], refetchType: "all" });
  }, [queryClient]);

  const openImport = useCallback(() => setImportOpen(true), []);

  const addRecordActions = useMemo<AddRecordAction[]>(() => {
    if (!canImport) return [];
    return [
      {
        id: "import-rates",
        label: "Import rates",
        description: "Read a CSV of the quarter's matrix and publish every row that checks out.",
        icon: FileSpreadsheetIcon,
        onClick: openImport,
      },
    ];
  }, [canImport, openImport]);

  const contextMenuActions = useMemo<RowAction<IftaTaxRateRow>[]>(() => {
    if (!canDelete) return [];
    return [
      {
        id: "delete",
        label: "Delete",
        icon: Trash2Icon,
        variant: "destructive",
        onClick: (row) => setDeleting(row.original),
      },
    ];
  }, [canDelete]);

  return (
    <>
      <DataTable<IftaTaxRateRow>
        name="IFTA Tax Rate"
        queryKey={IFTA_TAX_RATE_LIST_KEY}
        graphql={iftaTaxRateTableGraphQLConfig}
        resource={Resource.IFTATaxRate}
        columns={columns}
        addRecordActions={addRecordActions}
        contextMenuActions={contextMenuActions}
        TablePanel={IftaTaxRatePanel}
        renderEmptyState={(state) => (
          <IftaTaxRatesEmpty {...state} canImport={canImport} onImport={openImport} />
        )}
      />
      <DeleteIftaTaxRateDialog
        open={deleting !== null}
        onOpenChange={(open) => {
          if (!open) setDeleting(null);
        }}
        rate={deleting}
        onDeleted={invalidate}
      />
      {canImport ? (
        <ImportIftaTaxRatesDialog open={importOpen} onOpenChange={setImportOpen} />
      ) : null}
    </>
  );
}
