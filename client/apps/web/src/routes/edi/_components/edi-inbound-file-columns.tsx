import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { DataTablePlaceholder } from "@/components/data-table/_components/data-table-components";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { EDIInboundFileStatusBadge } from "@trenova/shared/components/status-badge";
import { Badge } from "@trenova/shared/components/ui/badge";
import { ediConnectionMethodChoices, ediInboundFileStatusChoices } from "@/lib/choices";
import type { EDIInboundFileRow } from "@/lib/graphql/edi-table";
import type { ColumnDef } from "@trenova/shared/types/data-table";

export function getInboundFileColumns(t: TranslateFn): ColumnDef<EDIInboundFileRow>[] {
  return [
    {
      accessorKey: "status",
      header: t("Status"),
      cell: ({ row }) => <EDIInboundFileStatusBadge status={row.original.status} />,
      size: 150,
      meta: {
        label: t("Status"),
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
      header: t("File"),
      cell: ({ row }) => (
        <div className="min-w-0">
          <div className="truncate font-medium">{row.original.fileName}</div>
          <div className="text-muted-foreground truncate text-xs">{row.original.remotePath}</div>
        </div>
      ),
      size: 280,
      meta: {
        label: t("File"),
        apiField: "fileName",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      id: "partner",
      header: t("Partner"),
      cell: ({ row }) =>
        row.original.partner?.name ? (
          <div className="min-w-0">
            <div className="truncate font-medium">{row.original.partner.name}</div>
            <div className="text-muted-foreground truncate text-xs">
              {row.original.partner.code}
            </div>
          </div>
        ) : (
          <DataTablePlaceholder />
        ),
      size: 220,
      meta: {
        label: t("Partner"),
        apiField: "ediPartnerId",
        filterable: false,
        sortable: false,
      },
    },
    {
      accessorKey: "method",
      header: t("Method"),
      cell: ({ row }) => <Badge variant="outline">{row.original.method}</Badge>,
      size: 110,
      meta: {
        label: t("Method"),
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
      header: t("Transactions"),
      cell: ({ row }) =>
        row.original.transactionCount > 0 ? (
          <Badge variant="secondary">{row.original.transactionCount}</Badge>
        ) : (
          <DataTablePlaceholder />
        ),
      size: 120,
      meta: {
        label: t("Transactions"),
        apiField: "transactionCount",
        filterable: false,
        sortable: true,
      },
    },
    {
      accessorKey: "interchangeControlNumber",
      header: t("Control Number"),
      cell: ({ row }) =>
        row.original.interchangeControlNumber ? (
          <span className="font-mono text-xs">{row.original.interchangeControlNumber}</span>
        ) : (
          <DataTablePlaceholder />
        ),
      size: 140,
      meta: {
        label: t("Control Number"),
        apiField: "interchangeControlNumber",
        filterable: true,
        sortable: false,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "receivedAt",
      header: t("Received"),
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.receivedAt} />,
      size: 180,
      meta: {
        label: t("Received"),
        apiField: "receivedAt",
        filterable: false,
        sortable: true,
        filterType: "date",
      },
    },
  ];
}
