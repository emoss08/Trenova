import { AmountDisplay } from "@trenova/shared/components/accounting/amount-display";
import { Badge } from "@trenova/shared/components/ui/badge";
import { formatUnixDate } from "@trenova/shared/lib/date";
import type { DriverExpenseRow } from "@trenova/shared/lib/graphql/driver-portal";
import { type ColumnDef } from "@tanstack/react-table";

type BadgeVariant = React.ComponentProps<typeof Badge>["variant"];

export const expenseStatusMeta: Record<string, { label: string; variant: BadgeVariant }> = {
  Pending: { label: "Pending", variant: "info" },
  Approved: { label: "Approved", variant: "active" },
  Rejected: { label: "Rejected", variant: "inactive" },
  Reimbursed: { label: "Reimbursed", variant: "active" },
  Cancelled: { label: "Cancelled", variant: "secondary" },
};

export function DriverExpenseStatusBadge({ status }: { status: string }) {
  const meta = expenseStatusMeta[status] ?? { label: status, variant: "secondary" as const };
  return <Badge variant={meta.variant}>{meta.label}</Badge>;
}

function workerName(row: DriverExpenseRow): string {
  if (!row.worker) return "—";
  return `${row.worker.firstName} ${row.worker.lastName}`.trim() || "—";
}

export function getColumns(): ColumnDef<DriverExpenseRow>[] {
  return [
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => <DriverExpenseStatusBadge status={row.original.status} />,
      size: 110,
      meta: { apiField: "status" },
    },
    {
      accessorKey: "worker",
      header: "Driver",
      cell: ({ row }) => <span className="text-sm font-medium">{workerName(row.original)}</span>,
      size: 160,
      meta: { apiField: "workerId", label: "Driver" },
    },
    {
      accessorKey: "amountMinor",
      header: "Amount",
      cell: ({ row }) => (
        <span className="text-sm font-medium tabular-nums">
          <AmountDisplay value={row.original.amountMinor} currency={row.original.currencyCode} />
        </span>
      ),
      size: 110,
      meta: { apiField: "amountMinor", label: "Amount" },
    },
    {
      accessorKey: "description",
      header: "Description",
      cell: ({ row }) => (
        <span className="line-clamp-2 max-w-96 text-xs text-muted-foreground">
          {row.original.description}
        </span>
      ),
      size: 300,
      meta: { apiField: "description" },
    },
    {
      accessorKey: "incurredDate",
      header: "Incurred",
      cell: ({ row }) => (
        <span className="text-xs">{formatUnixDate(row.original.incurredDate)}</span>
      ),
      size: 110,
      meta: { apiField: "incurredDate", label: "Incurred" },
    },
    {
      accessorKey: "receiptDocumentId",
      header: "Receipt",
      cell: ({ row }) =>
        row.original.receiptDocumentId ? (
          <Badge variant="secondary">Attached</Badge>
        ) : (
          <span className="text-xs text-muted-foreground">—</span>
        ),
      size: 90,
      meta: { apiField: "receiptDocumentId", label: "Receipt" },
    },
    {
      accessorKey: "createdAt",
      header: "Submitted",
      cell: ({ row }) => <span className="text-xs">{formatUnixDate(row.original.createdAt)}</span>,
      size: 110,
      meta: { apiField: "createdAt", label: "Submitted" },
    },
    {
      accessorKey: "reviewedAt",
      header: "Reviewed",
      cell: ({ row }) =>
        row.original.reviewedAt ? (
          <div className="flex flex-col">
            <span className="text-xs">{formatUnixDate(row.original.reviewedAt)}</span>
            {row.original.reviewedBy ? (
              <span className="text-xs text-muted-foreground">{row.original.reviewedBy.name}</span>
            ) : null}
          </div>
        ) : (
          <span className="text-xs text-muted-foreground">—</span>
        ),
      size: 130,
      meta: { apiField: "reviewedAt", label: "Reviewed" },
    },
  ];
}
