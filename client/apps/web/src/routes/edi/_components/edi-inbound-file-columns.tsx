import { DataTablePlaceholder } from "@/components/data-table/_components/data-table-components";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { EDIInboundFileStatusBadge } from "@trenova/shared/components/status-badge";
import { Badge } from "@trenova/shared/components/ui/badge";
import { ediConnectionMethodChoices, ediInboundFileStatusChoices } from "@/lib/choices";
import type { EDIInboundFile } from "@trenova/shared/types/edi";
import type { ColumnDef } from "@tanstack/react-table";

export function getInboundFileColumns(): ColumnDef<EDIInboundFile>[] {
  return [
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => <EDIInboundFileStatusBadge status={row.original.status} />,
      size: 150,
      meta: {
        label: "Status",
        apiField: "status",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: [...ediInboundFileStatusChoices],
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "fileName",
      header: "File",
      cell: ({ row }) => (
        <div className="min-w-0">
          <div className="truncate font-medium">{row.original.fileName}</div>
          <div className="truncate text-xs text-muted-foreground">
            {row.original.remotePath}
          </div>
        </div>
      ),
      size: 280,
      meta: {
        label: "File",
        apiField: "fileName",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      id: "partner",
      header: "Partner",
      cell: ({ row }) =>
        row.original.partner?.name ? (
          <div className="min-w-0">
            <div className="truncate font-medium">{row.original.partner.name}</div>
            <div className="truncate text-xs text-muted-foreground">
              {row.original.partner.code}
            </div>
          </div>
        ) : (
          <DataTablePlaceholder />
        ),
      size: 220,
      meta: {
        label: "Partner",
        apiField: "ediPartnerId",
        filterable: false,
        sortable: false,
      },
    },
    {
      accessorKey: "method",
      header: "Method",
      cell: ({ row }) => <Badge variant="outline">{row.original.method}</Badge>,
      size: 110,
      meta: {
        label: "Method",
        apiField: "method",
        filterable: true,
        sortable: false,
        filterType: "select",
        filterOptions: [...ediConnectionMethodChoices],
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "transactionCount",
      header: "Transactions",
      cell: ({ row }) =>
        row.original.transactionCount > 0 ? (
          <Badge variant="secondary">{row.original.transactionCount}</Badge>
        ) : (
          <DataTablePlaceholder />
        ),
      size: 120,
      meta: {
        label: "Transactions",
        apiField: "transactionCount",
        filterable: false,
        sortable: true,
      },
    },
    {
      accessorKey: "interchangeControlNumber",
      header: "Control Number",
      cell: ({ row }) =>
        row.original.interchangeControlNumber ? (
          <span className="font-mono text-xs">{row.original.interchangeControlNumber}</span>
        ) : (
          <DataTablePlaceholder />
        ),
      size: 140,
      meta: {
        label: "Control Number",
        apiField: "interchangeControlNumber",
        filterable: true,
        sortable: false,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "receivedAt",
      header: "Received",
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.receivedAt} />,
      size: 180,
      meta: {
        label: "Received",
        apiField: "receivedAt",
        filterable: false,
        sortable: true,
        filterType: "date",
      },
    },
  ];
}
