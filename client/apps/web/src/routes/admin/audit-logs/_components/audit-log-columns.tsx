import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { ResolvedUserAvatar } from "@/components/resolved-user-avatar";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import type { AuditEntryRow } from "@/lib/graphql/audit-log-table";
import { Resource } from "@trenova/shared/types/permission";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { auditOperationFilterOptions, operationLabel, resourceLabel } from "./audit-log-formatters";

const auditResourceFilterOptions = Object.values(Resource).map((value) => ({
  value,
  label: resourceLabel(value),
}));

export function getColumns(t: TranslateFn): ColumnDef<AuditEntryRow>[] {
  return [
    {
      accessorKey: "resourceId",
      header: t("Resource ID"),
      size: 220,
      minSize: 180,
      maxSize: 260,
      meta: {
        label: t("Resource ID"),
        apiField: "resourceId",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "comment",
      header: t("Description"),
      cell: ({ row }) => row.original.comment || "-",
      size: 360,
      minSize: 300,
      maxSize: 480,
      meta: {
        label: t("Description"),
        apiField: "comment",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "resource",
      header: t("Resource"),
      cell: ({ row }) => resourceLabel(row.original.resource),
      size: 170,
      minSize: 140,
      maxSize: 220,
      meta: {
        label: t("Resource"),
        apiField: "resource",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: auditResourceFilterOptions,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "operation",
      header: t("Action"),
      cell: ({ row }) => operationLabel(row.original.operation),
      size: 150,
      minSize: 120,
      maxSize: 180,
      meta: {
        label: t("Action"),
        apiField: "operation",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: auditOperationFilterOptions,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "timestamp",
      header: t("Timestamp"),
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.timestamp} />,
      size: 200,
      minSize: 170,
      maxSize: 240,
      meta: {
        label: t("Timestamp"),
        apiField: "timestamp",
        filterable: true,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
    },
    {
      accessorKey: "user",
      header: t("User"),
      cell: ({ row }) => {
        const user = row.original.user;
        const name = user?.name || "Unknown User";
        const email = user?.emailAddress || "No email";

        return (
          <div className="flex items-center gap-3">
            <ResolvedUserAvatar
              userId={user?.id}
              name={user?.name}
              profilePicUrl={user?.profilePicUrl}
              thumbnailUrl={user?.thumbnailUrl}
              className="bg-muted size-8 rounded-md"
              imageClassName="rounded-md bg-muted"
              fallbackClassName="text-xs"
              alt={name}
            />
            <div className="flex flex-col">
              <span className="font-medium">{name}</span>
              <span className="text-muted-foreground text-xs">{email}</span>
            </div>
          </div>
        );
      },
      size: 260,
      minSize: 220,
      maxSize: 320,
      meta: {
        label: t("User"),
        apiField: "user.name",
        filterable: false,
        sortable: false,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
  ];
}
