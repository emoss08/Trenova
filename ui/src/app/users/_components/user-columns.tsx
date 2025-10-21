import { HoverCardTimestamp } from "@/components/data-table/_components/data-table-components";
import { StatusBadge } from "@/components/status-badge";
import { Badge } from "@/components/ui/badge";
import { LazyImage } from "@/components/ui/image";
import { statusChoices } from "@/lib/choices";
import type { RoleSchema, UserSchema } from "@/lib/schemas/user-schema";
import { RoleType } from "@/types/roles-permissions";
import { type ColumnDef } from "@tanstack/react-table";

export function getUserColumns(): ColumnDef<UserSchema>[] {
  return [
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => {
        const { status } = row.original;
        return <StatusBadge status={status} />;
      },
      size: 120,
      minSize: 100,
      maxSize: 150,
      meta: {
        apiField: "status",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: statusChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      id: "fullName",
      accessorKey: "name",
      header: "Full Name",
      cell: ({ row }) => {
        const { profilePicUrl, name, username } = row.original;
        return (
          <div className="flex items-center gap-2">
            <LazyImage
              src={profilePicUrl || `https://avatar.vercel.sh/${name}.svg`}
              alt={name}
              className="size-6 rounded-full"
            />
            <div className="flex flex-col text-left leading-tight">
              <span className="text-sm font-medium">{name}</span>
              <span className="text-2xs text-muted-foreground">{username}</span>
            </div>
          </div>
        );
      },
      size: 200,
      minSize: 200,
      maxSize: 250,
      meta: {
        apiField: "name",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "emailAddress",
      header: "Email",
      cell: ({ row }) => {
        const { emailAddress } = row.original;
        return (
          <a
            href={`mailto:${emailAddress}`}
            className="text-sm text-muted-foreground underline hover:text-foreground"
          >
            {emailAddress}
          </a>
        );
      },
      meta: {
        apiField: "emailAddress",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
      size: 200,
      minSize: 200,
      maxSize: 250,
    },
    {
      id: "lastLoginAt",
      accessorKey: "lastLoginAt",
      header: "Last Login",
      cell: ({ row }) => {
        const { lastLoginAt } = row.original;
        return <HoverCardTimestamp timestamp={lastLoginAt} />;
      },
      meta: {
        apiField: "lastLoginAt",
        filterable: false,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
      size: 200,
      minSize: 200,
      maxSize: 250,
    },
    {
      accessorKey: "createdAt",
      header: "Created At",
      cell: ({ row }) => {
        return <HoverCardTimestamp timestamp={row.original.createdAt} />;
      },
      meta: {
        apiField: "createdAt",
        filterable: false,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
      size: 200,
      minSize: 200,
      maxSize: 250,
    },
  ];
}

export function getRoleColumns(): ColumnDef<RoleSchema>[] {
  return [
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => {
        const { status } = row.original;
        return <StatusBadge status={status} />;
      },
    },
    {
      accessorKey: "roleType",
      header: "Type",
      cell: ({ row }) => {
        const { roleType } = row.original;
        return (
          <Badge variant={roleType === RoleType.System ? "indigo" : "orange"}>
            {roleType}
          </Badge>
        );
      },
    },
    {
      accessorKey: "name",
      header: "Name",
    },
    {
      accessorKey: "description",
      header: "Description",
    },
    {
      accessorKey: "createdAt",
      header: "Created At",
      cell: ({ row }) => {
        return <HoverCardTimestamp timestamp={row.original.createdAt} />;
      },
      meta: {
        apiField: "createdAt",
        filterable: false,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
      size: 200,
      minSize: 200,
      maxSize: 250,
    },
  ];
}
