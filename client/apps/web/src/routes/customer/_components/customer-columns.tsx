import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { EditableStatusBadge } from "@/components/editable-status-badge";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { statusChoices } from "@/lib/choices";
import { apiService } from "@/services/api";
import type { CustomerRow } from "@/lib/graphql/customer-table";
import type { Customer } from "@trenova/shared/types/customer";
import { useQueryClient } from "@tanstack/react-query";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { useCallback } from "react";
import { SeriesCell } from "@/components/data-table/cells/series-cell";
import type { DataTableSeriesSpec } from "@/lib/data-table-series";

const WEEKLY_SHIPMENTS: DataTableSeriesSpec = {
  resource: "shipment",
  groupField: "customerId",
  dateField: "createdAt",
  interval: "week",
  periods: 12,
};

const MONTHLY_REVENUE: DataTableSeriesSpec = {
  resource: "shipment",
  groupField: "customerId",
  dateField: "createdAt",
  valueField: "totalChargeAmount",
  interval: "month",
  periods: 6,
};

/** Charted columns cost a request, so they wait until someone shows them. */
export const CUSTOMER_HIDDEN_COLUMNS: Record<string, boolean> = {
  shipmentTrend: false,
  revenueTrend: false,
};

function CustomerStatusCell({ row }: { row: CustomerRow }) {
  const queryClient = useQueryClient();

  const handleStatusChange = useCallback(
    async (newStatus: Customer["status"]) => {
      if (!row.id) return;
      await apiService.customerService.patch(row.id, {
        status: newStatus,
      });

      await queryClient.invalidateQueries({
        queryKey: ["customer-list"],
      });
    },
    [row.id, queryClient],
  );

  return (
    <EditableStatusBadge
      status={row.status}
      options={statusChoices}
      onStatusChange={handleStatusChange}
    />
  );
}

export function getColumns(t: TranslateFn): ColumnDef<CustomerRow>[] {
  return [
    {
      accessorKey: "status",
      header: t("Status"),
      cell: ({ row }) => <CustomerStatusCell row={row.original} />,
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
      accessorKey: "code",
      header: t("Code"),
      cell: ({ row }) => <span className="font-medium">{row.original.code}</span>,
      size: 120,
      minSize: 80,
      maxSize: 150,
      meta: {
        apiField: "code",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "name",
      header: t("Name"),
      cell: ({ row }) => <span>{row.original.name}</span>,
      meta: {
        apiField: "name",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "city",
      header: t("City"),
      cell: ({ row }) => <span>{row.original.city || "—"}</span>,
      size: 150,
      minSize: 100,
      maxSize: 200,
      meta: {
        apiField: "city",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "postalCode",
      header: t("Postal code"),
      cell: ({ row }) => <span>{row.original.postalCode || "—"}</span>,
      size: 120,
      minSize: 80,
      maxSize: 150,
      meta: {
        apiField: "postalCode",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "createdAt",
      header: t("Created At"),
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
    {
      id: "shipmentTrend",
      header: t("Shipments, 12 weeks"),
      accessorFn: () => null,
      cell: ({ row }) => <SeriesCell spec={WEEKLY_SHIPMENTS} id={row.original.id} />,
      size: 170,
      minSize: 140,
      maxSize: 220,
      meta: {
        label: t("Shipments, 12 weeks"),
        sortable: false,
        filterable: false,
        exportable: false,
      },
    },
    {
      id: "revenueTrend",
      header: t("Revenue, 6 months"),
      accessorFn: () => null,
      cell: ({ row }) => (
        <SeriesCell spec={MONTHLY_REVENUE} id={row.original.id} format="money" />
      ),
      size: 190,
      minSize: 150,
      maxSize: 240,
      meta: {
        label: t("Revenue, 6 months"),
        sortable: false,
        filterable: false,
        exportable: false,
      },
    },
  ];
}
