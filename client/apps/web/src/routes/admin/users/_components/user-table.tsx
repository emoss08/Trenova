import { DataTable } from "@/components/data-table/data-table";
import { panelSearchParamsParser } from "@/hooks/data-table/use-data-table-state";
import { useOnlineUsers } from "@/hooks/use-online-users";
import { userTableGraphQLConfig } from "@/lib/graphql/user-table";
import { statusChoices } from "@/lib/choices";
import { apiService } from "@/services/api";
import type { DockAction, RowAction } from "@trenova/shared/types/data-table";
import { Resource } from "@trenova/shared/types/permission";
import type { User } from "@trenova/shared/types/user";
import { useQueryClient } from "@tanstack/react-query";
import type { Row } from "@tanstack/react-table";
import { CircleCheckIcon, LayersPlus } from "lucide-react";
import { useQueryStates } from "nuqs";
import { useCallback, useMemo } from "react";
import { toast } from "sonner";
import { getColumns } from "./user-columns";
import { UserPanel } from "./user-panel";

export default function UserTable() {
  const { onlineUserIDs } = useOnlineUsers();
  const columns = useMemo(() => getColumns(onlineUserIDs), [onlineUserIDs]);
  const queryClient = useQueryClient();
  const [, setPanelSearchParams] = useQueryStates(panelSearchParamsParser);

  const handleBulkStatusUpdate = useCallback(
    async (rows: User[], status: string) => {
      const ids = rows.map((r) => r.id);

      toast.promise(
        apiService.userService.bulkUpdateStatus({
          userIds: ids as string[],
          status: status as User["status"],
        }),
        {
          loading: "Updating status...",
          success: "Status updated successfully",
          error: "Failed to update status",
          finally: async () => {
            await queryClient.invalidateQueries({
              queryKey: ["user-list"],
              refetchType: "all",
            });
          },
        },
      );
    },
    [queryClient],
  );

  const dockActions = useMemo<DockAction<User>[]>(
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

  const handleManageMemberships = useCallback(
    (row: Row<User>) => {
      const userId = row.original.id;
      if (!userId) return;

      void setPanelSearchParams({
        panelType: "edit",
        panelEntityId: userId,
      });
    },
    [setPanelSearchParams],
  );

  const contextMenuActions = useMemo<RowAction<User>[]>(
    () => [
      {
        id: "manage-memberships",
        label: "Manage Memberships",
        icon: LayersPlus,
        onClick: handleManageMemberships,
        hidden: (row) => row.original.status === "Inactive",
      },
    ],
    [handleManageMemberships],
  );

  return (
    <DataTable<User>
      name="User"
      queryKey="user-list"
      graphql={userTableGraphQLConfig}
      resource={Resource.User}
      columns={columns}
      enableRowSelection
      dockActions={dockActions}
      contextMenuActions={contextMenuActions}
      TablePanel={UserPanel}
    />
  );
}
