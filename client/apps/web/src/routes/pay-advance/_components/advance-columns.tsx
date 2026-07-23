import { AmountDisplay } from "@trenova/shared/components/accounting/amount-display";
import { PayAdvanceStatusBadge } from "@trenova/shared/components/status-badge";
import { payAdvanceSourceChoices } from "@/lib/choices";
import type { PayAdvanceRow } from "@/lib/graphql/driver-settlement";
import type { PayAdvanceStatus } from "@trenova/shared/types/driver-pay";
import { type ColumnDef } from "@tanstack/react-table";

function formatDate(unix: number): string {
  if (!unix) return "—";
  return new Date(unix * 1000).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

export function getColumns(): ColumnDef<PayAdvanceRow>[] {
  return [
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => <PayAdvanceStatusBadge status={row.original.status as PayAdvanceStatus} />,
      size: 140,
      meta: { apiField: "status", label: "Status" },
    },
    {
      id: "worker",
      header: "Driver",
      cell: ({ row }) => (
        <span className="text-xs font-medium">
          {row.original.worker
            ? `${row.original.worker.firstName} ${row.original.worker.lastName}`.trim()
            : "—"}
        </span>
      ),
      meta: { label: "Worker" },
      size: 180,
    },
    {
      accessorKey: "source",
      header: "Source",
      cell: ({ row }) => (
        <span className="text-xs">
          {payAdvanceSourceChoices.find((choice) => choice.value === row.original.source)?.label ??
            row.original.source}
        </span>
      ),
      size: 130,
      meta: { apiField: "source", label: "Source" },
    },
    {
      accessorKey: "reference",
      header: "Reference",
      cell: ({ row }) => <span className="font-mono text-xs">{row.original.reference || "—"}</span>,
      size: 140,
      meta: { apiField: "reference", label: "Reference" },
    },
    {
      accessorKey: "issuedDate",
      header: "Issued",
      cell: ({ row }) => <span className="text-xs">{formatDate(row.original.issuedDate)}</span>,
      size: 110,
      meta: { apiField: "issuedDate", label: "Issued Date" },
    },
    {
      accessorKey: "amountMinor",
      header: () => <div className="text-right">Amount</div>,
      cell: ({ row }) => (
        <div className="text-right">
          <AmountDisplay value={row.original.amountMinor} currency={row.original.currencyCode} />
        </div>
      ),
      size: 100,
      meta: { apiField: "amountMinor", label: "Amount Minor" },
    },
    {
      accessorKey: "recoveredMinor",
      header: () => <div className="text-right">Recovered</div>,
      cell: ({ row }) => (
        <div className="text-right">
          <AmountDisplay value={row.original.recoveredMinor} currency={row.original.currencyCode} />
        </div>
      ),
      size: 100,
      meta: { apiField: "recoveredMinor", label: "Recovered Minor" },
    },
    {
      accessorKey: "outstandingMinor",
      header: () => <div className="text-right">Outstanding</div>,
      cell: ({ row }) => (
        <div className="text-right font-medium">
          <AmountDisplay
            value={row.original.outstandingMinor}
            variant={row.original.outstandingMinor > 0 ? "negative" : "neutral"}
            currency={row.original.currencyCode}
          />
        </div>
      ),
      size: 110,
      meta: { apiField: "outstandingMinor", label: "Outstanding Minor" },
    },
  ];
}
