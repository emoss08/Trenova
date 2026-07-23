import { DataTableDescription } from "@/components/data-table/_components/data-table-components";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { Badge } from "@trenova/shared/components/ui/badge";
import type { EmailProfile } from "@trenova/shared/types/email";
import { type ColumnDef } from "@tanstack/react-table";
import { emailProfileStatusChoices, emailProviderChoices } from "./email-profile-constants";

function StatusBadge({ status }: { status: EmailProfile["status"] }) {
  return <Badge variant={status === "Active" ? "active" : "inactive"}>{status}</Badge>;
}

export function getColumns(): ColumnDef<EmailProfile>[] {
  return [
    {
      accessorKey: "name",
      header: "Profile",
      cell: ({ row }) => (
        <div className="flex min-w-0 flex-col">
          <span className="truncate font-medium">{row.original.name}</span>
          <DataTableDescription description={row.original.description} truncateLength={70} />
        </div>
      ),
      size: 260,
      minSize: 220,
      meta: {
        label: "Profile",
        apiField: "name",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "senderEmail",
      header: "Sender",
      cell: ({ row }) => (
        <div className="flex min-w-0 flex-col">
          <span className="truncate">{row.original.senderName}</span>
          <span className="truncate text-xs text-muted-foreground">{row.original.senderEmail}</span>
        </div>
      ),
      size: 280,
      minSize: 240,
      meta: {
        label: "Sender",
        apiField: "senderEmail",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "provider",
      header: "Provider",
      size: 140,
      meta: {
        label: "Provider",
        apiField: "provider",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: emailProviderChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => <StatusBadge status={row.original.status} />,
      size: 120,
      meta: {
        label: "Status",
        apiField: "status",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: emailProfileStatusChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "replyToEmail",
      header: "Reply-To",
      cell: ({ row }) => (
        <span className="truncate text-muted-foreground">
          {row.original.replyToEmail || "Default sender"}
        </span>
      ),
      size: 240,
      meta: {
        label: "Reply-To",
        apiField: "replyToEmail",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "updatedAt",
      header: "Updated",
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.updatedAt} />,
      size: 180,
      meta: {
        label: "Updated",
        apiField: "updatedAt",
        filterable: false,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
    },
  ];
}
