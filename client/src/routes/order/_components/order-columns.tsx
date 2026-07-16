import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { OrderStatusBadge } from "@/components/status-badge";
import { orderStatusChoices } from "@/lib/choices";
import { formatCurrency } from "@/lib/utils";
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
      id: "customer",
      accessorKey: "customer",
      header: "Customer",
      cell: ({ row }) => {
        const customer = row.original.customer;
        if (!customer) return "-";
        return (
          <div className="flex flex-col">
            <span className="truncate">{customer.name}</span>
            <span className="text-2xs text-muted-foreground">{customer.code}</span>
          </div>
        );
      },
      meta: {
        apiField: "customerId",
        filterable: false,
        sortable: false,
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
      accessorKey: "bol",
      header: "BOL",
      cell: ({ row }) => row.original.bol || "-",
      meta: {
        apiField: "bol",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "totalAmount",
      header: () => <div className="text-right">Total</div>,
      cell: ({ row }) => {
        const { totalAmount, currencyCode } = row.original;
        if (totalAmount == null) return <div className="text-right">-</div>;
        return (
          <div className="text-right tabular-nums">
            {formatCurrency(Number(totalAmount), currencyCode || "USD")}
          </div>
        );
      },
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
