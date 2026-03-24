import { DataTable } from "@/components/data-table/data-table";
import type { AccessorialCharge } from "@/types/accessorial-charge";
import { Resource } from "@/types/permission";
import { useMemo } from "react";
import { getColumns } from "./accessorial-charge-columns";
import { AccessorialChargePanel } from "./accessorial-charge-panel";

export default function AccessorialChargeTable() {
  //   const queryClient = useQueryClient();
  const columns = useMemo(() => getColumns(), []);

  //   const handleBulkStatusUpdate = useCallback(
  //     async (rows: EquipmentType[], status: string) => {
  //       const ids = rows.map((r) => r.id);
  //       toast.promise(
  //         apiService.equipmentTypeService.bulkUpdateStatus({
  //           equipmentTypeIds: ids as string[],
  //           status: status as EquipmentType["status"],
  //         }),
  //         {
  //           loading: "Updating status...",
  //           success: "Status updated successfully",
  //           error: "Failed to update status",
  //           finally: async () => {
  //             await queryClient.invalidateQueries({
  //               queryKey: ["equipment-type-list"],
  //               refetchType: "all",
  //             });
  //           },
  //         },
  //       );
  //     },
  //     [queryClient],
  //   );

  //   const dockActions = useMemo<DockAction<EquipmentType>[]>(
  //     () => [
  //       {
  //         id: "status-update",
  //         type: "select",
  //         label: "Update Status",
  //         loadingLabel: "Updating...",
  //         icon: CircleCheckIcon,
  //         options: statusChoices,
  //         onSelect: handleBulkStatusUpdate,
  //         clearSelectionOnSuccess: true,
  //       },
  //     ],
  //     [handleBulkStatusUpdate],
  //   );

  return (
    <DataTable<AccessorialCharge>
      name="Accessorial Charge"
      link="/accessorial-charges/"
      queryKey="accessorial-charge-list"
      exportModelName="accessorial-charge"
      resource={Resource.AccessorialCharge}
      columns={columns}
      //   dockActions={dockActions}
      enableRowSelection
      TablePanel={AccessorialChargePanel}
    />
  );
}
