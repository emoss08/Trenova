import { DataTable } from "@/components/data-table/data-table";
import { statusChoices } from "@/lib/choices";
import { apiService } from "@/services/api";
import type { Customer } from "@/types/customer";
import type { DockAction } from "@/types/data-table";
import { Resource } from "@/types/permission";
import { useQueryClient } from "@tanstack/react-query";
import { CircleCheckIcon } from "lucide-react";
import { useCallback, useMemo } from "react";
import { toast } from "sonner";
import { getColumns } from "./customer-columns";
import { CustomerPanel } from "./customer-panel";

export default function CustomerTable() {
  const queryClient = useQueryClient();
  const columns = useMemo(() => getColumns(), []);

  const handleBulkStatusUpdate = useCallback(
    async (rows: Customer[], status: string) => {
      const ids = rows.map((r) => r.id);
      toast.promise(
        apiService.customerService.bulkUpdateStatus({
          customerIds: ids as string[],
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

  const dockActions = useMemo<DockAction<Customer>[]>(
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
    <DataTable<Customer>
      name="Customer"
      link="/customers/"
      queryKey="customer-list"
      exportModelName="customer"
      resource={Resource.Customer}
      columns={columns}
      dockActions={dockActions}
      TablePanel={CustomerPanel}
      enableRowSelection
      extraSearchParams={{
        includeState: true,
        includeBillingProfile: true,
        includeEmailProfile: true,
      }}
    />
  );
}
