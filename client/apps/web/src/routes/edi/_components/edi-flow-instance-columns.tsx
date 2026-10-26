import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { DataTablePlaceholder } from "@/components/data-table/_components/data-table-components";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { FlowInstanceStatusBadge } from "@trenova/shared/components/status-badge";
import type { EDIFlowInstanceRow } from "@/lib/graphql/edi-table";
import type { ColumnDef } from "@trenova/shared/types/data-table";

export function getFlowInstanceColumns(t: TranslateFn): ColumnDef<EDIFlowInstanceRow>[] {
  return [
    {
      accessorKey: "status",
      header: t("Status"),
      cell: ({ row }) => <FlowInstanceStatusBadge status={row.original.status} />,
      size: 130,
      meta: {
        label: t("Status"),
        apiField: "status",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: [
          { value: "Open", label: t("Open") },
          { value: "Held", label: t("Held") },
          { value: "Escalated", label: t("Escalated") },
          { value: "Closed", label: t("Closed") },
        ],
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
        sortable: false,
        filterable: true,
        filterType: "record",
        filterRecord: "EDI_PARTNER",
      },
    },
    {
      accessorKey: "key",
      header: t("Key"),
      cell: ({ row }) => <span className="font-mono">{row.original.key}</span>,
      size: 200,
      meta: {
        label: t("Key"),
        apiField: "key",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "flow",
      header: t("Flow"),
      cell: ({ row }) => <span className="font-mono">{row.original.flow}</span>,
      size: 220,
      meta: {
        label: t("Flow"),
        apiField: "flow",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "reason",
      header: t("Why it waits"),
      cell: ({ row }) =>
        row.original.reason ? (
          <span className="truncate">{row.original.reason}</span>
        ) : (
          <DataTablePlaceholder />
        ),
      size: 280,
      meta: { label: t("Why it waits"), filterable: false, sortable: false },
    },
    {
      accessorKey: "wakeAt",
      header: t("Next deadline"),
      cell: ({ row }) =>
        row.original.wakeAt ? (
          <HoverCardTimestamp timestamp={row.original.wakeAt} />
        ) : (
          <DataTablePlaceholder />
        ),
      size: 170,
      meta: { label: t("Next deadline"), apiField: "wakeAt", filterable: false, sortable: true },
    },
    {
      accessorKey: "lastEventAt",
      header: t("Last event"),
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.lastEventAt} />,
      size: 170,
      meta: {
        label: t("Last event"),
        apiField: "lastEventAt",
        filterable: false,
        sortable: true,
      },
    },
  ];
}
