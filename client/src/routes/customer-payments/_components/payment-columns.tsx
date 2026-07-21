import { AmountDisplay } from "@/components/accounting/amount-display";
import { PlainCustomerPaymentStatusBadge } from "@/components/status-badge";
import type { CustomerPaymentRow } from "@/lib/graphql/customer-payment";
import type { CustomerPaymentStatus } from "@/types/customer-payment";
import { type ColumnDef } from "@tanstack/react-table";

function formatDate(unix: number): string {
  if (!unix) return "—";
  return new Date(unix * 1000).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

export function getColumns(): ColumnDef<CustomerPaymentRow>[] {
  return [
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => (
        <PlainCustomerPaymentStatusBadge
          status={row.original.status as CustomerPaymentStatus}
        />
      ),
      size: 100,
      meta: { apiField: "status" },
    },
    {
      accessorKey: "referenceNumber",
      header: "Reference",
      cell: ({ row }) => (
        <span className="font-mono text-xs font-medium">
          {row.original.referenceNumber || "—"}
        </span>
      ),
      size: 140,
      meta: { apiField: "referenceNumber" },
    },
    {
      id: "customer",
      header: "Customer",
      cell: ({ row }) => (
        <span className="text-xs font-medium">
          {row.original.customer
            ? `${row.original.customer.code} - ${row.original.customer.name}`
            : "—"}
        </span>
      ),
      size: 220,
    },
    {
      accessorKey: "paymentMethod",
      header: "Method",
      cell: ({ row }) => <span className="text-xs">{row.original.paymentMethod}</span>,
      size: 90,
      meta: { apiField: "paymentMethod" },
    },
    {
      accessorKey: "paymentDate",
      header: "Payment Date",
      cell: ({ row }) => (
        <span className="text-xs">{formatDate(row.original.paymentDate)}</span>
      ),
      size: 120,
      meta: { apiField: "paymentDate" },
    },
    {
      accessorKey: "accountingDate",
      header: "Accounting Date",
      cell: ({ row }) => (
        <span className="text-xs">{formatDate(row.original.accountingDate)}</span>
      ),
      size: 120,
      meta: { apiField: "accountingDate" },
    },
    {
      accessorKey: "amountMinor",
      header: () => <div className="text-right">Amount</div>,
      cell: ({ row }) => (
        <div className="text-right">
          <AmountDisplay value={row.original.amountMinor} className="text-xs font-semibold" />
        </div>
      ),
      size: 110,
      meta: { apiField: "amountMinor" },
    },
    {
      accessorKey: "appliedAmountMinor",
      header: () => <div className="text-right">Applied</div>,
      cell: ({ row }) => (
        <div className="text-right">
          <AmountDisplay
            value={row.original.appliedAmountMinor}
            className="text-xs text-muted-foreground"
          />
        </div>
      ),
      size: 110,
      meta: { apiField: "appliedAmountMinor" },
    },
    {
      accessorKey: "unappliedAmountMinor",
      header: () => <div className="text-right">Unapplied</div>,
      cell: ({ row }) => (
        <div className="text-right">
          <AmountDisplay
            value={row.original.unappliedAmountMinor}
            className={
              row.original.unappliedAmountMinor > 0
                ? "text-xs font-medium text-amber-600 dark:text-amber-400"
                : "text-xs text-muted-foreground/60"
            }
          />
        </div>
      ),
      size: 110,
      meta: { apiField: "unappliedAmountMinor" },
    },
    {
      id: "applications",
      header: "Invoices",
      cell: ({ row }) => (
        <span className="text-xs text-muted-foreground tabular-nums">
          {row.original.applications?.length ?? 0}
        </span>
      ),
      size: 70,
    },
  ];
}
