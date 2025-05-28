import { DataTableColumnHeaderWithTooltip } from "@/components/data-table/_components/data-table-column-header";
import { UserAvatar } from "@/components/nav-user";
import { generateDateTimeStringFromUnixTimestamp } from "@/lib/date";
import { type AuditEntry } from "@/types/audit-entry";
import { type ColumnDef } from "@tanstack/react-table";
import {
  AuditEntryActionBadge,
  AuditEntryResourceBadge,
} from "./audit-column-components";

export function getColumns(): ColumnDef<AuditEntry>[] {
  return [
    {
      accessorKey: "resourceId",
      header: ({ column }) => (
        <DataTableColumnHeaderWithTooltip
          column={column}
          title="Resource ID"
          tooltipContent="The ID of the resource that was affected."
        />
      ),
    },
    {
      accessorKey: "comment",
      header: "Description",
      cell: ({ row }) => {
        const { comment } = row.original;

        return <p>{comment}</p>;
      },
    },
    {
      accessorKey: "resource",
      header: ({ column }) => (
        <DataTableColumnHeaderWithTooltip
          column={column}
          title="Resource"
          tooltipContent="The resource that was affected."
        />
      ),
      cell: ({ row }) => {
        const { resource } = row.original;

        return <AuditEntryResourceBadge withDot={false} resource={resource} />;
      },
    },
    {
      accessorKey: "action",
      header: ({ column }) => (
        <DataTableColumnHeaderWithTooltip
          column={column}
          title="Action"
          tooltipContent="The action that was performed on the resource."
        />
      ),
      cell: ({ row }) => {
        const entry = row.original;

        return <AuditEntryActionBadge withDot={false} action={entry.action} />;
      },
    },
    {
      accessorKey: "timestamp",
      header: ({ column }) => (
        <DataTableColumnHeaderWithTooltip
          column={column}
          title="Timestamp"
          tooltipContent="The timestamp of when the action was performed."
        />
      ),
      cell: ({ row }) => {
        const entry = row.original;

        return (
          <p>{generateDateTimeStringFromUnixTimestamp(entry.timestamp)}</p>
        );
      },
    },
    {
      accessorKey: "user",
      header: ({ column }) => (
        <DataTableColumnHeaderWithTooltip
          column={column}
          title="User"
          tooltipContent="The user that performed the action."
        />
      ),
      cell: ({ row }) => {
        const { user } = row.original;

        return <UserAvatar user={user} />;
      },
    },
  ];
}
