import { DataTable } from "@/components/data-table/data-table";
import { equipmentStatusChoices } from "@/lib/choices";
import { apiService } from "@/services/api";
import type { DockAction, RowAction } from "@/types/data-table";
import { Resource } from "@/types/permission";
import type { Trailer } from "@/types/trailer";
import { useQueryClient } from "@tanstack/react-query";
import { CircleCheckIcon, MapPinIcon } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { toast } from "sonner";
import { LocateTrailerDialog } from "./locate-trailer-dialog";
import { getColumns } from "./trailer-columns";
import { TrailerPanel } from "./trailer-panel";

export default function Table() {
  const queryClient = useQueryClient();
  const columns = useMemo(() => getColumns(), []);
  const [locateTrailerId, setLocateTrailerId] = useState<string | null>(null);

  const handleBulkStatusUpdate = useCallback(
    async (rows: Trailer[], status: string) => {
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

  const dockActions = useMemo<DockAction<Trailer>[]>(
    () => [
      {
        id: "status-update",
        type: "select",
        label: "Update Status",
        loadingLabel: "Updating...",
        icon: CircleCheckIcon,
        options: equipmentStatusChoices,
        onSelect: handleBulkStatusUpdate,
        clearSelectionOnSuccess: true,
      },
    ],
    [handleBulkStatusUpdate],
  );

  const contextMenuActions = useMemo<RowAction<Trailer>[]>(
    () => [
      {
        id: "locate",
        label: "Locate Trailer",
        icon: MapPinIcon,
        onClick: (row) => setLocateTrailerId(row.original.id ?? null),
      },
    ],
    [],
  );

  return (
    <>
      <DataTable<Trailer>
        name="Trailer"
        link="/trailers/"
        queryKey="trailer-list"
        exportModelName="trailer"
        resource={Resource.Trailer}
        columns={columns}
        dockActions={dockActions}
        contextMenuActions={contextMenuActions}
        enableRowSelection
        TablePanel={TrailerPanel}
        extraSearchParams={{
          includeFleetDetails: true,
          includeEquipmentDetails: true,
        }}
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
