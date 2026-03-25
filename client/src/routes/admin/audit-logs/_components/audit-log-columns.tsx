import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import type { AuditEntry } from "@/types/audit-entry";
import { Resource } from "@/types/permission";
import type { ColumnDef } from "@tanstack/react-table";
import {
  auditOperationFilterOptions,
  operationLabel,
  resourceLabel,
  userInitials,
} from "./audit-log-formatters";

const auditResourceFilterOptions = Object.values(Resource).map((value) => ({
  value,
  label: resourceLabel(value),
}));

export function getColumns(): ColumnDef<AuditEntry>[] {
  return [
    {
      accessorKey: "resourceId",
      header: "Resource ID",
      size: 220,
      minSize: 180,
      maxSize: 260,
      meta: {
        label: "Resource ID",
        apiField: "resourceId",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "comment",
      header: "Description",
      cell: ({ row }) => row.original.comment || "-",
      size: 360,
      minSize: 300,
      maxSize: 480,
      meta: {
        label: "Description",
        apiField: "comment",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "resource",
      header: "Resource",
      cell: ({ row }) => resourceLabel(row.original.resource),
      size: 170,
      minSize: 140,
      maxSize: 220,
      meta: {
        label: "Resource",
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
      header: "Action",
      cell: ({ row }) => operationLabel(row.original.operation),
      size: 150,
      minSize: 120,
      maxSize: 180,
      meta: {
        label: "Action",
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
      header: "Timestamp",
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.timestamp} />,
      size: 200,
      minSize: 170,
      maxSize: 240,
      meta: {
        label: "Timestamp",
        apiField: "timestamp",
        filterable: true,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
    },
    {
      accessorKey: "user",
      header: "User",
      cell: ({ row }) => {
        const user = row.original.user;
        const name = user?.name || "Unknown User";
        const email = user?.emailAddress || "No email";

        return (
          <div className="flex items-center gap-3">
            <Avatar className="size-8 rounded-md bg-muted">
              <AvatarImage
                className="rounded-md bg-muted"
                src={user?.thumbnailUrl || user?.profilePicUrl}
                alt={name}
              />
              <AvatarFallback className="text-xs">{userInitials(user?.name)}</AvatarFallback>
            </Avatar>
            <div className="flex flex-col">
              <span className="font-medium">{name}</span>
              <span className="text-xs text-muted-foreground">{email}</span>
            </div>
          </div>
        );
      },
      size: 260,
      minSize: 220,
      maxSize: 320,
      meta: {
        label: "User",
        apiField: "user.name",
        filterable: false,
        sortable: false,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
  ];
}
