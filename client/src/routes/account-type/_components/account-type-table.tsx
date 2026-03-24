import { DataTable } from "@/components/data-table/data-table";
import { statusChoices } from "@/lib/choices";
import { apiService } from "@/services/api";
import type { DockAction } from "@/types/data-table";
import { Resource } from "@/types/permission";
import type { AccountType } from "@/types/account-type";
import { useQueryClient } from "@tanstack/react-query";
import { CircleCheckIcon } from "lucide-react";
import { useCallback, useMemo } from "react";
import { toast } from "sonner";
import { getColumns } from "./account-type-columns";
import { AccountTypePanel } from "./account-type-panel";

export default function AccountTypeTable() {
  const queryClient = useQueryClient();
  const columns = useMemo(() => getColumns(), []);

  const handleBulkStatusUpdate = useCallback(
    async (rows: AccountType[], status: string) => {
      const ids = rows.map((r) => r.id);
      toast.promise(
        apiService.accountTypeService.bulkUpdateStatus({
          accountTypeIds: ids as string[],
          status: status as AccountType["status"],
        }),
        {
          loading: "Updating status...",
          success: "Status updated successfully",
          error: "Failed to update status",
          finally: async () => {
            await queryClient.invalidateQueries({
              queryKey: ["account-type-list"],
              refetchType: "all",
            });
          },
        },
      );
    },
    [queryClient],
  );

  const dockActions = useMemo<DockAction<AccountType>[]>(
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
    <DataTable<AccountType>
      name="Account Type"
      link="/account-types/"
      queryKey="account-type-list"
      exportModelName="account-type"
      resource={Resource.AccountType}
      columns={columns}
      dockActions={dockActions}
      TablePanel={AccountTypePanel}
      enableRowSelection
    />
  );
}
