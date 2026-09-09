import { DataTable } from "@/components/data-table/data-table";
import {
  jurisdictionFilterOptions,
  useIftaJurisdictionOptions,
} from "@/components/fields/ifta-jurisdiction-select-field";
import { usePermission } from "@/hooks/use-permission";
import {
  IFTA_MILEAGE_ENTRY_LIST_KEY,
  iftaMileageEntryTableGraphQLConfig,
  type IftaMileageEntryRow,
} from "@/lib/graphql/ifta-jurisdiction-mileage";
import { EmptyTable } from "@trenova/shared/components/ui/empty-table";
import type { DataTableEmptyStateRenderProps, RowAction } from "@trenova/shared/types/data-table";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQueryClient } from "@tanstack/react-query";
import { Trash2Icon } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { DeleteIftaMileageEntryDialog } from "./delete-ifta-mileage-entry-dialog";
import { getColumns } from "./ifta-jurisdiction-mileage-columns";
import { IftaJurisdictionMileagePanel } from "./ifta-jurisdiction-mileage-panel";

const EMPTY_COLUMNS = [
  { label: "Travelled" },
  { label: "Tractor" },
  { label: "Jurisdiction" },
  { label: "Miles", numeric: true },
  { label: "Loaded" },
  { label: "Source" },
  { label: "Notes" },
] as const;

function IftaJurisdictionMileageEmpty({
  hasActiveFilters,
  onClearFilters,
}: DataTableEmptyStateRenderProps) {
  return (
    <EmptyTable
      className="py-10"
      title={hasActiveFilters ? "Nothing matches" : "No manual miles yet"}
      description={
        hasActiveFilters
          ? "No mileage entry fits the search and filters. Widen them, or clear them to see every one."
          : "Most miles come from shipment moves automatically. Enter miles here only for travel the system did not see."
      }
      columns={EMPTY_COLUMNS}
      onClearFilters={hasActiveFilters ? onClearFilters : undefined}
    />
  );
}

export default function IftaJurisdictionMileageTable() {
  const queryClient = useQueryClient();
  const { jurisdictions } = useIftaJurisdictionOptions();
  const columns = useMemo(
    () => getColumns(jurisdictionFilterOptions(jurisdictions)),
    [jurisdictions],
  );
  const { allowed: canDelete } = usePermission(Resource.IFTAJurisdictionMileage, Operation.Delete);
  const [deleting, setDeleting] = useState<IftaMileageEntryRow | null>(null);

  const invalidate = useCallback(async () => {
    await queryClient.invalidateQueries({
      queryKey: [IFTA_MILEAGE_ENTRY_LIST_KEY],
      refetchType: "all",
    });
  }, [queryClient]);

  const contextMenuActions = useMemo<RowAction<IftaMileageEntryRow>[]>(() => {
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
      <DataTable<IftaMileageEntryRow>
        name="Jurisdiction Mileage"
        queryKey={IFTA_MILEAGE_ENTRY_LIST_KEY}
        graphql={iftaMileageEntryTableGraphQLConfig}
        resource={Resource.IFTAJurisdictionMileage}
        columns={columns}
        contextMenuActions={contextMenuActions}
        TablePanel={IftaJurisdictionMileagePanel}
        renderEmptyState={(state) => <IftaJurisdictionMileageEmpty {...state} />}
      />
      <DeleteIftaMileageEntryDialog
        open={deleting !== null}
        onOpenChange={(open) => {
          if (!open) setDeleting(null);
        }}
        entry={deleting}
        onDeleted={invalidate}
      />
    </>
  );
}
