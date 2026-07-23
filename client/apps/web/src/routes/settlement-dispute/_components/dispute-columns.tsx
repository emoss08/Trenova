import { AmountDisplay } from "@/components/accounting/amount-display";
import { Badge } from "@/components/ui/badge";
import type { SettlementDisputeRow } from "@/lib/graphql/driver-portal";
import { type ColumnDef } from "@tanstack/react-table";

type BadgeVariant = React.ComponentProps<typeof Badge>["variant"];

export const disputeStatusMeta: Record<string, { label: string; variant: BadgeVariant }> = {
  Open: { label: "Open", variant: "info" },
  InReview: { label: "In Review", variant: "warning" },
  Resolved: { label: "Resolved", variant: "active" },
  Denied: { label: "Denied", variant: "inactive" },
  Withdrawn: { label: "Withdrawn", variant: "secondary" },
};

export const disputeCategoryLabels: Record<string, string> = {
  MissingPay: "Missing Pay",
  IncorrectRate: "Incorrect Rate",
  IncorrectDeduction: "Incorrect Deduction",
  MissingReimbursement: "Missing Reimbursement",
  Other: "Other",
};

export function SettlementDisputeStatusBadge({ status }: { status: string }) {
  const meta = disputeStatusMeta[status] ?? { label: status, variant: "secondary" as const };
  return <Badge variant={meta.variant}>{meta.label}</Badge>;
}

function formatDate(unix: number): string {
  if (!unix) return "—";
  return new Date(unix * 1000).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

function workerName(row: SettlementDisputeRow): string {
  if (!row.worker) return "—";
  return `${row.worker.firstName} ${row.worker.lastName}`.trim() || "—";
}

export function getColumns(): ColumnDef<SettlementDisputeRow>[] {
  return [
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => <SettlementDisputeStatusBadge status={row.original.status} />,
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
      accessorKey: "category",
      header: "Category",
      cell: ({ row }) => (
        <span className="text-xs">
          {disputeCategoryLabels[row.original.category] ?? row.original.category}
        </span>
      ),
      size: 150,
      meta: { apiField: "category" },
    },
    {
      accessorKey: "settlement",
      header: "Settlement",
      cell: ({ row }) =>
        row.original.settlement ? (
          <div className="flex flex-col">
            <span className="font-mono text-xs font-medium">
              {row.original.settlement.settlementNumber}
            </span>
            <span className="text-xs text-muted-foreground">
              Net{" "}
              <AmountDisplay
                value={row.original.settlement.netPayMinor}
                currency={row.original.settlement.currencyCode}
              />
            </span>
          </div>
        ) : (
          <span className="text-xs text-muted-foreground">—</span>
        ),
      size: 150,
      meta: { apiField: "settlementId", label: "Settlement" },
    },
    {
      accessorKey: "description",
      header: "Description",
      cell: ({ row }) => (
        <span className="line-clamp-2 max-w-96 text-xs text-muted-foreground">
          {row.original.description}
        </span>
      ),
      size: 320,
      meta: { apiField: "description" },
    },
    {
      accessorKey: "createdAt",
      header: "Submitted",
      cell: ({ row }) => <span className="text-xs">{formatDate(row.original.createdAt)}</span>,
      size: 110,
      meta: { apiField: "createdAt", label: "Submitted" },
    },
    {
      accessorKey: "resolvedAt",
      header: "Resolved",
      cell: ({ row }) =>
        row.original.resolvedAt ? (
          <div className="flex flex-col">
            <span className="text-xs">{formatDate(row.original.resolvedAt)}</span>
            {row.original.resolvedBy ? (
              <span className="text-xs text-muted-foreground">{row.original.resolvedBy.name}</span>
            ) : null}
          </div>
        ) : (
          <span className="text-xs text-muted-foreground">—</span>
        ),
      size: 130,
      meta: { apiField: "resolvedAt", label: "Resolved" },
    },
  ];
}
