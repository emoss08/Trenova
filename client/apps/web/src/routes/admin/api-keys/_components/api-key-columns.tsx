import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { PermissionScopeBadge } from "@trenova/shared/components/status-badge";
import { Badge } from "@trenova/shared/components/ui/badge";
import type { ApiKeyRow } from "@/lib/graphql/api-key-table";
import type { ColumnDef } from "@trenova/shared/types/data-table";

export function getColumns(t: TranslateFn): ColumnDef<ApiKeyRow>[] {
  return [
    {
      accessorKey: "name",
      header: t("Name"),
      cell: ({ row }) => (
        <div className="flex min-w-0 flex-col">
          <span className="truncate font-medium">{row.original.name}</span>
          <span className="text-muted-foreground truncate font-mono text-xs">
            {row.original.keyPrefix}
          </span>
        </div>
      ),
      meta: {
        label: t("Name"),
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
      header: t("Description"),
      cell: ({ row }) => (
        <span className="text-muted-foreground line-clamp-2 text-sm">
          {row.original.description || t("No description")}
        </span>
      ),
      meta: {
        label: t("Description"),
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
      header: t("Status"),
      cell: ({ row }) => (
        <Badge variant={row.original.status === "active" ? "active" : "inactive"}>
          {row.original.status}
        </Badge>
      ),
      meta: {
        label: t("Status"),
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
      header: t("Permissions"),
      cell: ({ row }) => {
        console.info("scope", row.original);
        return <PermissionScopeBadge scope={row.original.permissionScope} />;
      },
      meta: {
        label: t("Permissions"),
        apiField: "permissionScope",
        filterable: false,
        sortable: false,
      },
      size: 140,
    },
    {
      accessorKey: "lastUsedAt",
      header: t("Last Used"),
      cell: ({ row }) =>
        row.original.lastUsedAt ? (
          <HoverCardTimestamp timestamp={row.original.lastUsedAt} />
        ) : (
          <span className="text-muted-foreground">{t("Never")}</span>
        ),
      meta: {
        label: t("Last Used"),
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
      header: t("Expires"),
      cell: ({ row }) =>
        row.original.expiresAt ? (
          <HoverCardTimestamp timestamp={row.original.expiresAt} />
        ) : (
          <span className="text-muted-foreground">{t("Does not expire")}</span>
        ),
      meta: {
        label: t("Expires"),
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
      header: t("Updated"),
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.updatedAt} />,
      meta: {
        label: t("Updated"),
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
