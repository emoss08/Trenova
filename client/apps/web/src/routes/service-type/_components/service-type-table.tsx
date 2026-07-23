import { DataTable } from "@/components/data-table/data-table";
import { serviceTypeTableGraphQLConfig } from "@/lib/graphql/service-type-table";
import { statusChoices } from "@/lib/choices";
import { apiService } from "@/services/api";
import type { DockAction } from "@trenova/shared/types/data-table";
import { Resource } from "@trenova/shared/types/permission";
import type { ServiceType } from "@/types/service-type";
import { useQueryClient } from "@tanstack/react-query";
import { CircleCheckIcon } from "lucide-react";
import { useCallback, useMemo } from "react";
import { toast } from "sonner";
import { getColumns } from "./service-type-columns";
import { ServiceTypePanel } from "./service-type-panel";

export default function EquipmentTypeTable() {
  const queryClient = useQueryClient();
  const columns = useMemo(() => getColumns(), []);

  const handleBulkStatusUpdate = useCallback(
    async (rows: ServiceType[], status: string) => {
      const ids = rows.map((r) => r.id);
      toast.promise(
        apiService.serviceTypeService.bulkUpdateStatus({
          serviceTypeIds: ids as string[],
          status: status as ServiceType["status"],
        }),
        {
          loading: "Updating status...",
          success: "Status updated successfully",
          error: "Failed to update status",
          finally: async () => {
            await queryClient.invalidateQueries({
              queryKey: ["service-type-list"],
              refetchType: "all",
            });
          },
        },
      );
    },
    [queryClient],
  );

  const dockActions = useMemo<DockAction<ServiceType>[]>(
    () => [
      {
        id: "status-update",
        type: "select",
        label: "Update Status",
        loadingLabel: "Updating...",
        icon: CircleCheckIcon,
        options: statusChoices,
        onSelect: handleBulkStatusUpdate,
        clearSelectionOnSuccess: true,
      },
    ],
    [handleBulkStatusUpdate],
  );

  return (
    <DataTable<ServiceType>
      name="Service Type"
      queryKey="service-type-list"
      graphql={serviceTypeTableGraphQLConfig}
      resource={Resource.ServiceType}
      columns={columns}
      dockActions={dockActions}
      TablePanel={ServiceTypePanel}
      enableRowSelection
    />
  );
}
