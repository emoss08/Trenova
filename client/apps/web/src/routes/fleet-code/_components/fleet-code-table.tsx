import { useT } from "@trenova/shared/i18n/use-t";
import { DataTable } from "@/components/data-table/data-table";
import { fleetCodeTableGraphQLConfig, type FleetCodeRow } from "@/lib/graphql/fleet-code-table";
import { statusChoices } from "@/lib/choices";
import { apiService } from "@/services/api";
import type { DockAction } from "@trenova/shared/types/data-table";
import type { FleetCode } from "@trenova/shared/types/fleet-code";
import { Resource } from "@trenova/shared/types/permission";
import { useQueryClient } from "@tanstack/react-query";
import { CheckCircleIcon } from "@trenova/shared/components/icons";
import { useCallback, useMemo } from "react";
import { toast } from "sonner";
import { getColumns } from "./fleet-code-columns";
import { FleetCodePanel } from "./fleet-code-panel";
import { translate } from "@trenova/shared/i18n/runtime";

export default function FleetCodeTable() {
  const t = useT();

  const queryClient = useQueryClient();
  const columns = useMemo(() => getColumns(t), [t]);

  const handleBulkStatusUpdate = useCallback(
    async (rows: FleetCodeRow[], status: string) => {
      const ids = rows.map((r) => r.id);
      toast.promise(
        apiService.fleetCodeService.bulkUpdateStatus({
          fleetCodeIds: ids,
          status: status as FleetCode["status"],
        }),
        {
          loading: translate("Updating status..."),
          success: translate("Status updated successfully"),
          error: translate("Failed to update status"),
          finally: async () => {
            await queryClient.invalidateQueries({
              queryKey: ["fleet-code-list"],
              refetchType: "all",
            });
          },
        },
      );
    },
    [queryClient],
  );

  const dockActions = useMemo<DockAction<FleetCodeRow>[]>(
    () => [
      {
        id: "status-update",
        type: "select",
        label: t("Update status"),
        loadingLabel: t("Updating..."),
        icon: CheckCircleIcon,
        options: statusChoices,
        onSelect: handleBulkStatusUpdate,
        clearSelectionOnSuccess: true,
      },
    ],
    [handleBulkStatusUpdate, t],
  );

  return (
    <DataTable<FleetCodeRow>
      name="Fleet Code"
      emptyTitle={t("No fleet codes yet")}
      queryKey="fleet-code-list"
      graphql={fleetCodeTableGraphQLConfig}
      resource={Resource.FleetCode}
      columns={columns}
      dockActions={dockActions}
      enableRowSelection
      TablePanel={FleetCodePanel}
    />
  );
}
