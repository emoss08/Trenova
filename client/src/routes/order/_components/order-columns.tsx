import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { OrderStatusBadge } from "@/components/status-badge";
import { orderStatusChoices } from "@/lib/choices";
import type { Order } from "@/types/order";
import { type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<Order>[] {
  return [
    {
      accessorKey: "orderNumber",
      header: "Order Number",
      cell: ({ row }) => (
        <span className="font-medium">{row.original.orderNumber}</span>
      ),
      meta: {
        apiField: "orderNumber",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => <OrderStatusBadge status={row.original.status} />,
      size: 120,
      minSize: 100,
      maxSize: 150,
      meta: {
        apiField: "status",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: orderStatusChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "poNumber",
      header: "PO Number",
      cell: ({ row }) => row.original.poNumber || "-",
      meta: {
        apiField: "poNumber",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "totalAmount",
      header: "Total Amount",
      cell: ({ row }) => row.original.totalAmount || "-",
      meta: {
        apiField: "totalAmount",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "eq",
      },
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
