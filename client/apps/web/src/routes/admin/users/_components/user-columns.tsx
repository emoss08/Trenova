/* eslint-disable react-refresh/only-export-components */
import { ResolvedUserAvatar } from "@/components/resolved-user-avatar";
import { EditableStatusBadge } from "@/components/editable-status-badge";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { statusChoices } from "@/lib/choices";
import { apiService } from "@/services/api";
import type { User } from "@/types/user";
import { useQueryClient } from "@tanstack/react-query";
import type { ColumnDef } from "@tanstack/react-table";
import { useCallback } from "react";

function UserStatusCell({ row }: { row: User }) {
  const queryClient = useQueryClient();

  const handleStatusChange = useCallback(
    async (newStatus: User["status"]) => {
      if (!row.id) return;
      await apiService.userService.patch(row.id, {
        status: newStatus,
      });

      await queryClient.invalidateQueries({
        queryKey: ["user-list"],
      });
    },
    [row.id, queryClient],
  );

  return (
    <EditableStatusBadge
      status={row.status}
      options={statusChoices}
      onStatusChange={handleStatusChange}
    />
  );
}

function UserNameCell({ user, isOnline }: { user: User; isOnline: boolean }) {
  return (
    <div className="flex items-center gap-3">
      <ResolvedUserAvatar
        userId={user.id}
        name={user.name}
        profilePicUrl={user.profilePicUrl}
        thumbnailUrl={user.thumbnailUrl}
        className="size-8 rounded-md bg-muted"
        imageClassName="rounded-md bg-muted"
        fallbackClassName="text-xs"
      />
      <div className="flex flex-col">
        <span className="flex items-center gap-2 font-medium">
          {user.name}
          <span
            className={`size-2 rounded-full ${
              isOnline ? "bg-green-500" : "bg-muted-foreground/40"
            }`}
            title={isOnline ? "Online" : "Offline"}
          />
        </span>
        <span className="text-xs text-muted-foreground">{user.emailAddress}</span>
      </div>
    </div>
  );
}

export function getColumns(
  onlineUserIDs: Set<string>,
): ColumnDef<User>[] {
  return [
    {
      accessorKey: "name",
      header: "User",
      cell: ({ row }) => (
        <UserNameCell
          user={row.original}
          isOnline={!!row.original.id && onlineUserIDs.has(row.original.id)}
        />
      ),
      meta: {
        label: "Name",
        apiField: "name",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
      size: 280,
      minSize: 200,
      maxSize: 400,
    },
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => <UserStatusCell row={row.original} />,
      meta: {
        label: "Status",
        apiField: "status",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: statusChoices,
        defaultFilterOperator: "eq",
      },
      size: 150,
      minSize: 120,
      maxSize: 200,
    },
    {
      accessorKey: "username",
      header: "Username",
      meta: {
        label: "Username",
        apiField: "username",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
      size: 150,
      minSize: 100,
      maxSize: 200,
    },
    {
      accessorKey: "lastLoginAt",
      header: "Last Login",
      cell: ({ row }) => {
        const lastLogin = row.original.lastLoginAt;
        if (!lastLogin) {
          return <span className="text-muted-foreground">Never</span>;
        }
        return <HoverCardTimestamp timestamp={lastLogin} />;
      },
      meta: {
        apiField: "lastLoginAt",
        filterable: false,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
      size: 180,
      minSize: 150,
      maxSize: 220,
    },
    {
      accessorKey: "createdAt",
      header: "Created At",
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.createdAt} />,
      meta: {
        apiField: "createdAt",
        filterable: false,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
      size: 180,
      minSize: 150,
      maxSize: 220,
    },
  ];
}
