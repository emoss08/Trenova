/* eslint-disable react-refresh/only-export-components */
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { ResolvedUserAvatar } from "@/components/resolved-user-avatar";
import { EditableStatusBadge } from "@/components/editable-status-badge";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { statusChoices } from "@/lib/choices";
import { apiService } from "@/services/api";
import type { UserRow } from "@/lib/graphql/user-table";
import type { User } from "@trenova/shared/types/user";
import { useQueryClient } from "@tanstack/react-query";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { useCallback } from "react";

function UserStatusCell({ row }: { row: UserRow }) {
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

function UserNameCell({ user, isOnline }: { user: UserRow; isOnline: boolean }) {
  return (
    <div className="flex items-center gap-3">
      <ResolvedUserAvatar
        userId={user.id}
        name={user.name}
        profilePicUrl={user.profilePicUrl}
        thumbnailUrl={user.thumbnailUrl}
        className="bg-muted size-8 rounded-md"
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
        <span className="text-muted-foreground text-xs">{user.emailAddress}</span>
      </div>
    </div>
  );
}

export function getColumns(onlineUserIDs: Set<string>, t: TranslateFn): ColumnDef<UserRow>[] {
  return [
    {
      accessorKey: "name",
      header: t("User"),
      cell: ({ row }) => (
        <UserNameCell
          user={row.original}
          isOnline={!!row.original.id && onlineUserIDs.has(row.original.id)}
        />
      ),
      meta: {
        label: t("Name"),
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
      header: t("Status"),
      cell: ({ row }) => <UserStatusCell row={row.original} />,
      meta: {
        label: t("Status"),
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
      header: t("Username"),
      meta: {
        label: t("Username"),
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
      header: t("Last Login"),
      cell: ({ row }) => {
        const lastLogin = row.original.lastLoginAt;
        if (!lastLogin) {
          return <span className="text-muted-foreground">{t("Never")}</span>;
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
      header: t("Created At"),
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
