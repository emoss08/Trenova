import { DataTableColumnHeaderWithTooltip } from "@/components/data-table/_components/data-table-column-header";
import { createEntityColumn } from "@/components/data-table/_components/data-table-column-helpers";
import { UserAvatar } from "@/components/nav-user";
import { Checkbox } from "@/components/ui/checkbox";
import { generateDateTimeStringFromUnixTimestamp } from "@/lib/date";
import { type AuditEntry } from "@/types/audit-entry";
import { createColumnHelper, type ColumnDef } from "@tanstack/react-table";
import {
  AuditEntryActionBadge,
  AuditEntryResourceBadge,
} from "./audit-column-components";

export function getColumns(): ColumnDef<AuditEntry>[] {
  const columnHelper = createColumnHelper<AuditEntry>();

  return [
    {
      id: "select",
      header: ({ table }) => {
        const isAllSelected = table.getIsAllPageRowsSelected();
        const isSomeSelected = table.getIsSomePageRowsSelected();

        return (
          <Checkbox
            data-slot="select-all"
            checked={isAllSelected || (isSomeSelected && "indeterminate")}
            onCheckedChange={(checked) =>
              table.toggleAllPageRowsSelected(!!checked)
            }
            aria-label="Select all"
          />
        );
      },
      cell: ({ row }) => (
        <Checkbox
          data-slot="select-row"
          checked={row.getIsSelected()}
          onCheckedChange={(checked) => row.toggleSelected(!!checked)}
          aria-label="Select row"
        />
      ),
      size: 50,
      enableSorting: false,
      enableHiding: false,
    },
    createEntityColumn(columnHelper, "resourceId", {
      accessorKey: "resourceId",
      getHeaderText: "Resource ID",
      getId: (auditEntry) => auditEntry.id,
      getDisplayText: (auditEntry) => auditEntry.resourceId || "-",
    }),
    {
      id: "comment",
      accessorKey: "comment",
      header: ({ column }) => (
        <DataTableColumnHeaderWithTooltip
          column={column}
          title="Description"
          tooltipContent="The description of the audit log."
        />
      ),
    },
    {
      id: "resource",
      header: ({ column }) => (
        <DataTableColumnHeaderWithTooltip
          column={column}
          title="Resource"
          tooltipContent="The resource that was affected."
        />
      ),
      cell: ({ row }) => {
        const entry = row.original;

        return (
          <AuditEntryResourceBadge withDot={false} resource={entry.resource} />
        );
      },
    },
    {
      id: "action",
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
      id: "timestamp",
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
      id: "user",
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
