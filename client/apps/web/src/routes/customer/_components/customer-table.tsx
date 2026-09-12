import { useT } from "@trenova/shared/i18n/use-t";
import { DataTable } from "@/components/data-table/data-table";
import { statusChoices } from "@/lib/choices";
import { customerTableGraphQLConfig, type CustomerRow } from "@/lib/graphql/customer-table";
import { apiService } from "@/services/api";
import type { Customer } from "@trenova/shared/types/customer";
import type { DockAction } from "@trenova/shared/types/data-table";
import { Resource } from "@trenova/shared/types/permission";
import { useQueryClient } from "@tanstack/react-query";
import { CircleCheckIcon } from "lucide-react";
import { useCallback, useMemo } from "react";
import { toast } from "sonner";
import { getColumns } from "./customer-columns";
import { CustomerPanel } from "./customer-panel";

export default function CustomerTable() {
  const t = useT();

  const queryClient = useQueryClient();
  const columns = useMemo(() => getColumns(t), [t]);

  const handleBulkStatusUpdate = useCallback(
    async (rows: CustomerRow[], status: string) => {
      const ids = rows.map((r) => r.id);
      toast.promise(
        apiService.customerService.bulkUpdateStatus({
          customerIds: ids,
          status: status as Customer["status"],
        }),
        {
          loading: "Updating status...",
          success: "Status updated successfully",
          error: "Failed to update status",
          finally: async () => {
            await queryClient.invalidateQueries({
              queryKey: ["customer-list"],
              refetchType: "all",
            });
          },
        },
      );
    },
    [queryClient],
  );

  const dockActions = useMemo<DockAction<CustomerRow>[]>(
    () => [
      {
        id: "status-update",
        type: "select",
        label: t("Update Status"),
        loadingLabel: t("Updating..."),
        icon: CircleCheckIcon,
        options: statusChoices,
        onSelect: handleBulkStatusUpdate,
        clearSelectionOnSuccess: true,
      },
    ],
    [handleBulkStatusUpdate, t],
  );

  return (
    <DataTable<CustomerRow>
      name="Customer"
      queryKey="customer-list"
      resource={Resource.Customer}
      columns={columns}
      dockActions={dockActions}
      TablePanel={CustomerPanel}
      enableRowSelection
      graphql={customerTableGraphQLConfig}
    />
  );
}
