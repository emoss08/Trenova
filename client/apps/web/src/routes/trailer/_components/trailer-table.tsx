import { useT } from "@trenova/shared/i18n/use-t";
import { DataTable } from "@/components/data-table/data-table";
import { equipmentStatusChoices } from "@/lib/choices";
import { apiService } from "@/services/api";
import type { DockAction, RowAction } from "@trenova/shared/types/data-table";
import { Resource } from "@trenova/shared/types/permission";
import type { Trailer } from "@/types/trailer";
import { useQueryClient } from "@tanstack/react-query";
import { CircleCheckIcon, MapPinIcon } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { toast } from "sonner";
import { equipmentTableGraphQLConfigs, type TrailerRow } from "@/lib/graphql/equipment-table";
import { LocateTrailerDialog } from "./locate-trailer-dialog";
import { getColumns } from "./trailer-columns";
import { TrailerPanel } from "./trailer-panel";

export default function Table() {
  const t = useT();

  const queryClient = useQueryClient();
  const columns = useMemo(() => getColumns(t), [t]);
  const [locateTrailerId, setLocateTrailerId] = useState<string | null>(null);

  const handleBulkStatusUpdate = useCallback(
    async (rows: TrailerRow[], status: string) => {
      const ids = rows.map((r) => r.id);
      toast.promise(
        apiService.trailerService.bulkUpdateStatus({
          trailerIds: ids as string[],
          status: status as Trailer["status"],
        }),
        {
          loading: "Updating status...",
          success: "Status updated successfully",
          error: "Failed to update status",
          finally: async () => {
            await queryClient.invalidateQueries({
              queryKey: ["trailer-list"],
              refetchType: "all",
            });
          },
        },
      );
    },
    [queryClient],
  );

  const dockActions = useMemo<DockAction<TrailerRow>[]>(
    () => [
      {
        id: "status-update",
        type: "select",
        label: t("Update Status"),
        loadingLabel: t("Updating..."),
        icon: CircleCheckIcon,
        options: equipmentStatusChoices,
        onSelect: handleBulkStatusUpdate,
        clearSelectionOnSuccess: true,
      },
    ],
    [handleBulkStatusUpdate, t],
  );

  const contextMenuActions = useMemo<RowAction<TrailerRow>[]>(
    () => [
      {
        id: "locate",
        label: t("Locate Trailer"),
        icon: MapPinIcon,
        onClick: (row) => setLocateTrailerId(row.original.id ?? null),
      },
    ],
    [t],
  );

  return (
    <>
      <DataTable<TrailerRow>
        name="Trailer"
        queryKey="trailer-list"
        graphql={equipmentTableGraphQLConfigs.trailer}
        resource={Resource.Trailer}
        columns={columns}
        dockActions={dockActions}
        contextMenuActions={contextMenuActions}
        enableRowSelection
        TablePanel={TrailerPanel}
      />
      {locateTrailerId && (
        <LocateTrailerDialog
          open={!!locateTrailerId}
          onOpenChange={(nextOpen) => {
            if (!nextOpen) setLocateTrailerId(null);
          }}
          trailerId={locateTrailerId}
          onLocated={() => setLocateTrailerId(null)}
        />
      )}
    </>
  );
}
