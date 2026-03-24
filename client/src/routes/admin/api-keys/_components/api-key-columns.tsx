import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { PermissionScopeBadge } from "@/components/status-badge";
import { Badge } from "@/components/ui/badge";
import type { ApiKey } from "@/types/api-key";
import type { ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<ApiKey>[] {
  return [
    {
      accessorKey: "name",
      header: "Name",
      cell: ({ row }) => (
        <div className="flex min-w-0 flex-col">
          <span className="truncate font-medium">{row.original.name}</span>
          <span className="truncate font-mono text-xs text-muted-foreground">
            {row.original.keyPrefix}
          </span>
        </div>
      ),
      meta: {
        label: "Name",
        apiField: "name",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
      size: 250,
    },
    {
      accessorKey: "description",
      header: "Description",
      cell: ({ row }) => (
        <span className="line-clamp-2 text-sm text-muted-foreground">
          {row.original.description || "No description"}
        </span>
      ),
      meta: {
        label: "Description",
        apiField: "description",
        filterable: true,
        sortable: false,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
      size: 280,
    },
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => (
        <Badge variant={row.original.status === "active" ? "active" : "inactive"}>
          {row.original.status}
        </Badge>
      ),
      meta: {
        label: "Status",
        apiField: "status",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "eq",
      },
      size: 120,
    },
    {
      accessorKey: "permissionScope",
      header: "Permissions",
      cell: ({ row }) => {
        console.info("scope", row.original);
        return <PermissionScopeBadge scope={row.original.permissionScope} />;
      },
      meta: {
        label: "Permissions",
        apiField: "permissionScope",
        filterable: false,
        sortable: false,
      },
      size: 140,
    },
    {
      accessorKey: "lastUsedAt",
      header: "Last Used",
      cell: ({ row }) =>
        row.original.lastUsedAt ? (
          <HoverCardTimestamp timestamp={row.original.lastUsedAt} />
        ) : (
          <span className="text-muted-foreground">Never</span>
        ),
      meta: {
        label: "Last Used",
        apiField: "lastUsedAt",
        filterable: false,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
      size: 180,
    },
    {
      accessorKey: "expiresAt",
      header: "Expires",
      cell: ({ row }) =>
        row.original.expiresAt ? (
          <HoverCardTimestamp timestamp={row.original.expiresAt} />
        ) : (
          <span className="text-muted-foreground">Does not expire</span>
        ),
      meta: {
        label: "Expires",
        apiField: "expiresAt",
        filterable: false,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
      size: 180,
    },
    {
      accessorKey: "updatedAt",
      header: "Updated",
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.updatedAt} />,
      meta: {
        label: "Updated",
        apiField: "updatedAt",
        filterable: false,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
      size: 180,
    },
  ];
}
