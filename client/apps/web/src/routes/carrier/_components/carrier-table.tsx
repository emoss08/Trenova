import { useT } from "@trenova/shared/i18n/use-t";
import { DataTable } from "@/components/data-table/data-table";
import { carrierStatusChoices } from "@/lib/choices";
import { carrierTableGraphQLConfig, type CarrierRow } from "@/lib/graphql/carrier-table";
import { apiService } from "@/services/api";
import type { Carrier } from "@trenova/shared/types/carrier";
import type { DockAction } from "@trenova/shared/types/data-table";
import { Resource } from "@trenova/shared/types/permission";
import { useQueryClient } from "@tanstack/react-query";
import { CheckCircleIcon } from "@trenova/shared/components/icons";
import { useCallback, useMemo } from "react";
import { toast } from "sonner";
import { getColumns } from "./carrier-columns";
import { CarrierPanel } from "./carrier-panel";
import { translate } from "@trenova/shared/i18n/runtime";

export default function CarrierTable() {
  const t = useT();

  const queryClient = useQueryClient();
  const columns = useMemo(() => getColumns(t), [t]);

  const handleBulkStatusUpdate = useCallback(
    async (rows: CarrierRow[], status: string) => {
      const ids = rows.map((r) => r.id);
      toast.promise(
        apiService.carrierService.bulkUpdateStatus({
          carrierIds: ids,
          status: status as Carrier["status"],
        }),
        {
          loading: translate("Updating status..."),
          success: translate("Status updated successfully"),
          error: translate("Failed to update status"),
          finally: async () => {
            await queryClient.invalidateQueries({
              queryKey: ["carrier-list"],
              refetchType: "all",
            });
          },
        },
      );
    },
    [queryClient],
  );

  const dockActions = useMemo<DockAction<CarrierRow>[]>(
    () => [
      {
        id: "status-update",
        type: "select",
        label: t("Update status"),
        loadingLabel: t("Updating..."),
        icon: CheckCircleIcon,
        options: carrierStatusChoices,
        onSelect: handleBulkStatusUpdate,
        clearSelectionOnSuccess: true,
      },
    ],
    [handleBulkStatusUpdate, t],
  );

  return (
    <DataTable<CarrierRow>
      name="Carrier"
      emptyTitle={t("No carriers yet")}
      queryKey="carrier-list"
      resource={Resource.Carrier}
      columns={columns}
      dockActions={dockActions}
      TablePanel={CarrierPanel}
      enableRowSelection
      graphql={carrierTableGraphQLConfig}
    />
  );
}
