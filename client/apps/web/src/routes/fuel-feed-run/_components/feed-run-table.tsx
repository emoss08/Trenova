import { useT } from "@trenova/shared/i18n/use-t";
import { DataTable } from "@/components/data-table/data-table";
import { usePermission } from "@/hooks/use-permission";
import {
  FUEL_FEED_RUN_LIST_KEY,
  fuelFeedRunTableGraphQLConfig,
  type FuelPurchaseImportBatch,
} from "@/lib/graphql/fuel-purchase-import";
import { useQueryClient } from "@tanstack/react-query";
import { EmptyTable } from "@trenova/shared/components/ui/empty-table";
import type { DataTableEmptyStateRenderProps, RowAction } from "@trenova/shared/types/data-table";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { ListChecksIcon } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { FeedRunDetailDialog } from "./feed-run-detail-dialog";
import { getColumns } from "./feed-run-columns";

const EMPTY_COLUMNS = [
  { label: "Provider" },
  { label: "Read" },
  { label: "Rows", numeric: true },
  { label: "Posted", numeric: true },
  { label: "Held", numeric: true },
  { label: "Ran" },
] as const;

function FeedRunsEmpty({ hasActiveFilters, onClearFilters }: DataTableEmptyStateRenderProps) {
  return (
    <EmptyTable
      className="py-10"
      title={hasActiveFilters ? "Nothing matches" : "No feed has run yet"}
      description={
        hasActiveFilters
          ? "No run fits the search and filters. Widen them, or clear them to see every one."
          : "Connect WEX, Comdata or Ramp under Integrations and Trenova reads your transactions on a schedule. Each read appears here with what it posted and what it is still holding."
      }
      columns={EMPTY_COLUMNS}
      onClearFilters={hasActiveFilters ? onClearFilters : undefined}
    />
  );
}

export default function FeedRunTable() {
  const t = useT();

  const queryClient = useQueryClient();
  const columns = useMemo(() => getColumns(t), [t]);
  const { allowed: canImport } = usePermission(Resource.FuelPurchaseImport, Operation.Import);
  const [reviewing, setReviewing] = useState<FuelPurchaseImportBatch | null>(null);

  const invalidate = useCallback(async () => {
    await queryClient.invalidateQueries({
      queryKey: [FUEL_FEED_RUN_LIST_KEY],
      refetchType: "all",
    });
  }, [queryClient]);

  const contextMenuActions = useMemo<RowAction<FuelPurchaseImportBatch>[]>(() => {
    if (!canImport) {
      return [];
    }

    return [
      {
        id: "review",
        label: t("Review rows"),
        icon: ListChecksIcon,
        onClick: (row) => setReviewing(row.original),
      },
    ];
  }, [canImport, t]);

  return (
    <>
      <DataTable<FuelPurchaseImportBatch>
        name="Feed Run"
        queryKey={FUEL_FEED_RUN_LIST_KEY}
        graphql={fuelFeedRunTableGraphQLConfig}
        resource={Resource.FuelPurchaseImport}
        columns={columns}
        contextMenuActions={contextMenuActions}
        onRowClick={(row) => setReviewing(row.original)}
        // A run is opened by a schedule, never by a person, so there is nothing
        // to add here.
        enableCreateAction={false}
        renderEmptyState={(state) => <FeedRunsEmpty {...state} />}
      />
      <FeedRunDetailDialog
        open={reviewing !== null}
        onOpenChange={(open) => {
          if (!open) {
            setReviewing(null);
            void invalidate();
          }
        }}
        batch={reviewing}
      />
    </>
  );
}
