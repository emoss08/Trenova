import { DataTable } from "@/components/data-table/data-table";
import { usePermission } from "@/hooks/use-permission";
import {
  FUEL_CARD_LIST_KEY,
  FUEL_CARD_OPTIONS_KEY,
  fuelCardTableGraphQLConfig,
  UNASSIGNED_FUEL_CARD_LIST_KEY,
  unassignedFuelCardTableGraphQLConfig,
  type FuelCardRow,
} from "@/lib/graphql/fuel-card";
import { EmptyTable } from "@trenova/shared/components/ui/empty-table";
import type { DataTableEmptyStateRenderProps, RowAction } from "@trenova/shared/types/data-table";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQueryClient } from "@tanstack/react-query";
import { BanIcon, LinkIcon } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { AssignFuelCardDialog } from "./assign-fuel-card-dialog";
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

function UnassignedFuelCardsEmpty({
  hasActiveFilters,
  onClearFilters,
}: DataTableEmptyStateRenderProps) {
  return (
    <EmptyTable
      className="py-10"
      title={hasActiveFilters ? "Nothing matches" : "Nothing waiting to be assigned"}
      description={
        hasActiveFilters
          ? "No unassigned card fits the search and filters. Widen them, or clear them to see every one."
          : "Cards land here when a connected fuel card feed sees a transaction on a card nobody had registered. Assign one to a tractor or driver and its purchases start matching on their own."
      }
      columns={EMPTY_COLUMNS}
      onClearFilters={hasActiveFilters ? onClearFilters : undefined}
    />
  );
}

export default function FuelCardTable({ unassignedOnly = false }: { unassignedOnly?: boolean }) {
  const queryClient = useQueryClient();
  const columns = useMemo(() => getColumns(), []);
  const { allowed: canCancel } = usePermission(Resource.FuelCard, Operation.Update);
  const { allowed: canAssign } = usePermission(Resource.FuelCard, Operation.Update);
  const [cancelling, setCancelling] = useState<FuelCardRow | null>(null);
  const [assigning, setAssigning] = useState<FuelCardRow | null>(null);

  const invalidate = useCallback(async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: [FUEL_CARD_LIST_KEY], refetchType: "all" }),
      queryClient.invalidateQueries({
        queryKey: [UNASSIGNED_FUEL_CARD_LIST_KEY],
        refetchType: "all",
      }),
      queryClient.invalidateQueries({ queryKey: [FUEL_CARD_OPTIONS_KEY], refetchType: "all" }),
    ]);
  }, [queryClient]);

  const contextMenuActions = useMemo<RowAction<FuelCardRow>[]>(() => {
    const actions: RowAction<FuelCardRow>[] = [];

    if (canAssign) {
      actions.push({
        id: "assign",
        label: "Assign card",
        icon: LinkIcon,
        hidden: (row) => row.original.status === "Cancelled",
        onClick: (row) => setAssigning(row.original),
      });
    }

    if (canCancel) {
      actions.push({
        id: "cancel",
        label: "Cancel card",
        icon: BanIcon,
        variant: "destructive",
        hidden: (row) => row.original.status === "Cancelled",
        onClick: (row) => setCancelling(row.original),
      });
    }

    return actions;
  }, [canAssign, canCancel]);

  return (
    <>
      <DataTable<FuelCardRow>
        name={unassignedOnly ? "Unassigned Fuel Card" : "Fuel Card"}
        queryKey={unassignedOnly ? UNASSIGNED_FUEL_CARD_LIST_KEY : FUEL_CARD_LIST_KEY}
        graphql={unassignedOnly ? unassignedFuelCardTableGraphQLConfig : fuelCardTableGraphQLConfig}
        resource={Resource.FuelCard}
        columns={columns}
        contextMenuActions={contextMenuActions}
        TablePanel={FuelCardPanel}
        // A card only lands here because a feed saw a transaction on one nobody
        // had registered. Anything somebody adds by hand they assign as they go,
        // so offering "add" on this view would only invite a card that instantly
        // does not belong on it.
        enableCreateAction={!unassignedOnly}
        renderEmptyState={(state) =>
          unassignedOnly ? <UnassignedFuelCardsEmpty {...state} /> : <FuelCardsEmpty {...state} />
        }
      />
      <AssignFuelCardDialog
        open={assigning !== null}
        onOpenChange={(open) => {
          if (!open) setAssigning(null);
        }}
        card={assigning}
        onAssigned={invalidate}
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
