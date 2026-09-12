import type { TranslateFn } from "@trenova/shared/i18n/use-t";
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
import type { EDIMessageRow } from "@/lib/graphql/edi-table";
import type { ColumnDef } from "@trenova/shared/types/data-table";

export function getMessageColumns(t: TranslateFn): ColumnDef<EDIMessageRow>[] {
  return [
    {
      accessorKey: "transactionSet",
      header: t("Transaction"),
      cell: ({ row }) => (
        <div className="flex items-center gap-2">
          <Badge variant="secondary">{row.original.transactionSet}</Badge>
          <Badge variant="outline">{row.original.direction}</Badge>
        </div>
      ),
      size: 170,
      meta: {
        label: t("Transaction"),
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
      accessorKey: "direction",
      header: t("Direction"),
      cell: ({ row }) => row.original.direction,
      size: 120,
      meta: {
        label: t("Direction"),
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
      header: t("Delivery"),
      cell: ({ row }) => {
        if (row.original.direction === "Inbound") {
          return <Badge variant="outline">{t("Received")}</Badge>;
        }
        if (!row.original.deliveryStatus) {
          return <DataTablePlaceholder text={t("Not queued")} />;
        }
        return (
          <div className="flex items-center gap-1.5">
            <EDIMessageDeliveryStatusBadge status={row.original.deliveryStatus} />
            {row.original.deliveryAttempts > 0 && (
              <span className="text-muted-foreground text-xs">
                ×{row.original.deliveryAttempts}
              </span>
            )}
          </div>
        );
      },
      size: 170,
      meta: {
        label: t("Delivery"),
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
      header: t("Acknowledgment"),
      cell: ({ row }) => <EDIMessageAckStatusBadge status={row.original.ackStatus} />,
      size: 150,
      meta: {
        label: t("Acknowledgment"),
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
      accessorKey: "generatedAt",
      header: t("Generated"),
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.generatedAt} />,
      size: 180,
      meta: {
        label: t("Generated"),
        apiField: "generatedAt",
        filterable: false,
        sortable: true,
        filterType: "date",
      },
    },
  ];
}
