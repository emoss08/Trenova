import { DataTable } from "@/components/data-table/data-table";
import { usePermission } from "@/hooks/use-permission";
import {
  FUEL_CARD_LIST_KEY,
  FUEL_CARD_OPTIONS_KEY,
  fuelCardTableGraphQLConfig,
  type FuelCardRow,
} from "@/lib/graphql/fuel-card";
import { EmptyTable } from "@trenova/shared/components/ui/empty-table";
import type { DataTableEmptyStateRenderProps, RowAction } from "@trenova/shared/types/data-table";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQueryClient } from "@tanstack/react-query";
import { BanIcon } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { CancelFuelCardDialog } from "./cancel-fuel-card-dialog";
import { getColumns } from "./fuel-card-columns";
import { FuelCardPanel } from "./fuel-card-panel";

const EMPTY_COLUMNS = [
  { label: "Status" },
  { label: "Label" },
  { label: "Provider" },
  { label: "Card" },
  { label: "Driver" },
  { label: "Tractor" },
  { label: "Expires", numeric: true },
] as const;

function FuelCardsEmpty({ hasActiveFilters, onClearFilters }: DataTableEmptyStateRenderProps) {
  return (
    <EmptyTable
      className="py-10"
      title={hasActiveFilters ? "Nothing matches" : "No fuel cards yet"}
      description={
        hasActiveFilters
          ? "No fuel card fits the search and filters. Widen them, or clear them to see every one."
          : "A card appears here once it is registered, by hand or from the provider's card list. Register the cards drivers carry before importing a statement, so its rows can be matched to a driver and unit."
      }
      columns={EMPTY_COLUMNS}
      onClearFilters={hasActiveFilters ? onClearFilters : undefined}
    />
  );
}

export default function FuelCardTable() {
  const queryClient = useQueryClient();
  const columns = useMemo(() => getColumns(), []);
  const { allowed: canCancel } = usePermission(Resource.FuelCard, Operation.Update);
  const [cancelling, setCancelling] = useState<FuelCardRow | null>(null);

  const invalidate = useCallback(async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: [FUEL_CARD_LIST_KEY], refetchType: "all" }),
      queryClient.invalidateQueries({ queryKey: [FUEL_CARD_OPTIONS_KEY], refetchType: "all" }),
    ]);
  }, [queryClient]);

  const contextMenuActions = useMemo<RowAction<FuelCardRow>[]>(() => {
    if (!canCancel) return [];
    return [
      {
        id: "cancel",
        label: "Cancel card",
        icon: BanIcon,
        variant: "destructive",
        hidden: (row) => row.original.status === "Cancelled",
        onClick: (row) => setCancelling(row.original),
      },
    ];
  }, [canCancel]);

  return (
    <>
      <DataTable<FuelCardRow>
        name="Fuel Card"
        queryKey={FUEL_CARD_LIST_KEY}
        graphql={fuelCardTableGraphQLConfig}
        resource={Resource.FuelCard}
        columns={columns}
        contextMenuActions={contextMenuActions}
        TablePanel={FuelCardPanel}
        renderEmptyState={(state) => <FuelCardsEmpty {...state} />}
      />
      <CancelFuelCardDialog
        open={cancelling !== null}
        onOpenChange={(open) => {
          if (!open) setCancelling(null);
        }}
        card={cancelling}
        onCancelled={invalidate}
      />
    </>
  );
}
