import { DataTablePlaceholder } from "@/components/data-table/_components/data-table-components";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import {
  EDIMessageAckStatusBadge,
  EDIMessageDeliveryStatusBadge,
} from "@trenova/shared/components/status-badge";
import { Badge } from "@trenova/shared/components/ui/badge";
import {
  ediAckStatusChoices,
  ediDocumentDirectionChoices,
  ediMessageDeliveryStatusChoices,
  ediTransactionSetChoices,
} from "@/lib/choices";
import type { EDIMessage } from "@trenova/shared/types/edi";
import type { ColumnDef } from "@tanstack/react-table";

export function getMessageColumns(): ColumnDef<EDIMessage>[] {
  return [
    {
      accessorKey: "transactionSet",
      header: "Transaction",
      cell: ({ row }) => (
        <div className="flex items-center gap-2">
          <Badge variant="secondary">{row.original.transactionSet}</Badge>
          <Badge variant="outline">{row.original.direction}</Badge>
        </div>
      ),
      size: 170,
      meta: {
        label: "Transaction",
        apiField: "transactionSet",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: [...ediTransactionSetChoices],
        defaultFilterOperator: "eq",
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
      accessorKey: "direction",
      header: "Direction",
      cell: ({ row }) => row.original.direction,
      size: 120,
      meta: {
        label: "Direction",
        apiField: "direction",
        filterable: true,
        sortable: false,
        filterType: "select",
        filterOptions: [...ediDocumentDirectionChoices],
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "deliveryStatus",
      header: "Delivery",
      cell: ({ row }) => {
        if (row.original.direction === "Inbound") {
          return <Badge variant="outline">Received</Badge>;
        }
        if (!row.original.deliveryStatus) {
          return <DataTablePlaceholder text="Not queued" />;
        }
        return (
          <div className="flex items-center gap-1.5">
            <EDIMessageDeliveryStatusBadge status={row.original.deliveryStatus} />
            {row.original.deliveryAttempts > 0 && (
              <span className="text-xs text-muted-foreground">
                ×{row.original.deliveryAttempts}
              </span>
            )}
          </div>
        );
      },
      size: 170,
      meta: {
        label: "Delivery",
        apiField: "deliveryStatus",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: [...ediMessageDeliveryStatusChoices],
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "ackStatus",
      header: "Acknowledgment",
      cell: ({ row }) => <EDIMessageAckStatusBadge status={row.original.ackStatus} />,
      size: 150,
      meta: {
        label: "Acknowledgment",
        apiField: "ackStatus",
        filterable: true,
        sortable: false,
        filterType: "select",
        filterOptions: [...ediAckStatusChoices],
        defaultFilterOperator: "eq",
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
      accessorKey: "generatedAt",
      header: "Generated",
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.generatedAt} />,
      size: 180,
      meta: {
        label: "Generated",
        apiField: "generatedAt",
        filterable: false,
        sortable: true,
        filterType: "date",
      },
    },
  ];
}
