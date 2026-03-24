import { DataTable } from "@/components/data-table/data-table";
import { useOnlineUsers } from "@/hooks/use-online-users";
import { statusChoices } from "@/lib/choices";
import { apiService } from "@/services/api";
import type { DockAction, RowAction } from "@/types/data-table";
import { Resource } from "@/types/permission";
import type { User } from "@/types/user";
import { useQueryClient } from "@tanstack/react-query";
import type { Row } from "@tanstack/react-table";
import { CircleCheckIcon, LayersPlus } from "lucide-react";
import { useCallback, useMemo } from "react";
import { toast } from "sonner";
import { getColumns } from "./user-columns";
import { UserPanel } from "./user-panel";

export default function UserTable() {
  const { onlineUserIDs } = useOnlineUsers();
  const columns = useMemo(() => getColumns(onlineUserIDs), [onlineUserIDs]);
  const queryClient = useQueryClient();

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

  const handleAddMembership = useCallback((_row: Row<User>) => {
    // TODO: implement add membership dialog
  }, []);

  const contextMenuActions = useMemo<RowAction<User>[]>(
    () => [
      {
        id: "add-membership",
        label: "Add Membership",
        icon: LayersPlus,
        onClick: handleAddMembership,
        hidden: (row) => row.original.status === "Inactive",
      },
    ],
    [handleAddMembership],
  );

  return (
    <DataTable<User>
      name="User"
      link="/users/"
      queryKey="user-list"
      resource={Resource.User}
      columns={columns}
      exportModelName="User"
      extraSearchParams={{ includeMemberships: true }}
      enableRowSelection
      dockActions={dockActions}
      contextMenuActions={contextMenuActions}
      TablePanel={UserPanel}
    />
  );
}
