import { FactoringSubmissionStatusBadge } from "@/components/factoring/factoring-badges";
import { recordPath } from "@/config/record-links";
import type { FactoringInvoiceStateRow } from "@/lib/graphql/factoring";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTimeShort } from "@trenova/shared/lib/date";
import { formatCurrency } from "@trenova/shared/lib/utils";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { Link } from "react-router";
import type { FactoringQueueTab } from "../use-factoring-state";

export function getFactoringInvoiceColumns(
  t: TranslateFn,
  tab: FactoringQueueTab,
): ColumnDef<FactoringInvoiceStateRow>[] {
  const columns: ColumnDef<FactoringInvoiceStateRow>[] = [
    {
      id: "invoice",
      header: t("Invoice #"),
      cell: ({ row }) =>
        row.original.invoice ? (
          <Link
            to={recordPath("invoice", row.original.invoiceId)}
            className="font-mono font-medium hover:underline"
            onClick={(event) => event.stopPropagation()}
          >
            {row.original.invoice.number}
          </Link>
        ) : (
          <span className="text-muted-foreground">{t("Unavailable")}</span>
        ),
      size: 160,
      minSize: 130,
      maxSize: 220,
      meta: { label: t("Invoice #") },
    },
    {
      id: "customer",
      header: t("Customer"),
      cell: ({ row }) => row.original.customer?.name ?? row.original.invoice?.billToName ?? "",
      size: 220,
      minSize: 160,
      maxSize: 320,
      meta: { label: t("Customer") },
    },
    {
      id: "factor",
      header: t("Factor"),
      cell: ({ row }) => row.original.factoringCompany?.name ?? "",
      size: 200,
      minSize: 150,
      maxSize: 300,
      meta: { label: t("Factor") },
    },
    {
      accessorKey: "openBalanceMinor",
      header: t("Open balance"),
      cell: ({ row }) => (
        <span className="tabular-nums">
          {formatCurrency(
            row.original.openBalanceMinor / 100,
            row.original.invoice?.currencyCode ?? "USD",
          )}
        </span>
      ),
      size: 140,
      minSize: 120,
      maxSize: 180,
      meta: {
        label: t("Open balance"),
        apiField: "openBalanceMinor",
        sortable: true,
      },
    },
  ];

  if (tab === "ready") {
    columns.push({
      accessorKey: "readySince",
      header: t("Ready since"),
      cell: ({ row }) =>
        row.original.readySince ? formatUnixDateTimeShort(row.original.readySince) : "",
      size: 160,
      minSize: 130,
      maxSize: 200,
      meta: {
        label: t("Ready since"),
        apiField: "readySince",
        sortable: true,
        filterable: true,
        filterType: "date",
      },
    });
  } else if (tab === "submitted") {
    columns.push(
      {
        id: "submission",
        header: t("Schedule"),
        cell: ({ row }) =>
          row.original.submission ? (
            <Link
              to={recordPath("factoring_submission", row.original.submission.submissionId)}
              className="font-mono font-medium hover:underline"
              onClick={(event) => event.stopPropagation()}
            >
              {row.original.submission.submissionNumber}
            </Link>
          ) : (
            <span className="text-muted-foreground">{t("Unavailable")}</span>
          ),
        size: 160,
        minSize: 130,
        maxSize: 220,
        meta: { label: t("Schedule") },
      },
      {
        id: "submissionStatus",
        header: t("Delivery"),
        cell: ({ row }) =>
          row.original.submission ? (
            <FactoringSubmissionStatusBadge status={row.original.submission.submissionStatus} />
          ) : null,
        size: 150,
        minSize: 120,
        maxSize: 200,
        meta: { label: t("Delivery") },
      },
    );
  } else {
    columns.push({
      accessorKey: "failedChecks",
      header: t("Failed checks"),
      cell: ({ row }) => <span className="tabular-nums">{row.original.failedChecks}</span>,
      size: 120,
      minSize: 100,
      maxSize: 150,
      meta: {
        label: t("Failed checks"),
        apiField: "failedChecks",
        sortable: true,
        filterable: true,
        filterType: "number",
      },
    });
  }

  columns.push({
    accessorKey: "evaluatedAt",
    header: t("Checked"),
    cell: ({ row }) => formatUnixDateTimeShort(row.original.evaluatedAt),
    size: 160,
    minSize: 130,
    maxSize: 200,
    meta: {
      label: t("Checked"),
      apiField: "evaluatedAt",
      sortable: true,
    },
  });

  return columns;
}
