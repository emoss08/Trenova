import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { AmountDisplay } from "@trenova/shared/components/accounting/amount-display";
import { PayAdvanceStatusBadge } from "@trenova/shared/components/status-badge";
import { payAdvanceSourceChoices } from "@/lib/choices";
import type { PayAdvanceRow } from "@/lib/graphql/driver-settlement";
import type { PayAdvanceStatus } from "@trenova/shared/types/driver-pay";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";

function formatDate(unix: number): string {
  return formatUnixDateMedium(unix, { fallback: "—" });
}

export function getColumns(t: TranslateFn): ColumnDef<PayAdvanceRow>[] {
  return [
    {
      accessorKey: "status",
      header: t("Status"),
      cell: ({ row }) => <PayAdvanceStatusBadge status={row.original.status as PayAdvanceStatus} />,
      size: 140,
      meta: { apiField: "status", label: t("Status") },
    },
    {
      id: "worker",
      header: t("Driver"),
      cell: ({ row }) => (
        <span className="text-xs font-medium">
          {row.original.worker
            ? `${row.original.worker.firstName} ${row.original.worker.lastName}`.trim()
            : "—"}
        </span>
      ),
      meta: { label: t("Worker") },
      size: 180,
    },
    {
      accessorKey: "source",
      header: t("Source"),
      cell: ({ row }) => (
        <span className="text-xs">
          {payAdvanceSourceChoices.find((choice) => choice.value === row.original.source)?.label ??
            row.original.source}
        </span>
      ),
      size: 130,
      meta: { apiField: "source", label: t("Source") },
    },
    {
      accessorKey: "reference",
      header: t("Reference"),
      cell: ({ row }) => <span className="font-mono text-xs">{row.original.reference || "—"}</span>,
      size: 140,
      meta: { apiField: "reference", label: t("Reference") },
    },
    {
      accessorKey: "issuedDate",
      header: t("Issued"),
      cell: ({ row }) => <span className="text-xs">{formatDate(row.original.issuedDate)}</span>,
      size: 110,
      meta: { apiField: "issuedDate", label: t("Issued Date") },
    },
    {
      accessorKey: "amountMinor",
      header: () => <div className="text-right">{t("Amount")}</div>,
      cell: ({ row }) => (
        <div className="text-right">
          <AmountDisplay value={row.original.amountMinor} currency={row.original.currencyCode} />
        </div>
      ),
      size: 100,
      meta: { apiField: "amountMinor", label: t("Amount Minor") },
    },
    {
      accessorKey: "recoveredMinor",
      header: () => <div className="text-right">{t("Recovered")}</div>,
      cell: ({ row }) => (
        <div className="text-right">
          <AmountDisplay value={row.original.recoveredMinor} currency={row.original.currencyCode} />
        </div>
      ),
      size: 100,
      meta: { apiField: "recoveredMinor", label: t("Recovered Minor") },
    },
    {
      accessorKey: "outstandingMinor",
      header: () => <div className="text-right">{t("Outstanding")}</div>,
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
      meta: { apiField: "outstandingMinor", label: t("Outstanding Minor") },
    },
  ];
}
