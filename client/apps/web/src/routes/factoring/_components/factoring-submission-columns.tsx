import { FactoringSubmissionStatusBadge } from "@/components/factoring/factoring-badges";
import { recordPath } from "@/config/record-links";
import type { FactoringSubmissionRow } from "@/lib/graphql/factoring";
import { FACTORING_TRANSPORT_METHOD_LABELS } from "@/lib/factoring";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTimeShort } from "@trenova/shared/lib/date";
import { formatMinor } from "@trenova/shared/lib/money";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { Link } from "react-router";

export function getFactoringSubmissionColumns(t: TranslateFn): ColumnDef<FactoringSubmissionRow>[] {
  return [
    {
      accessorKey: "number",
      header: t("Schedule"),
      cell: ({ row }) => (
        <Link
          to={recordPath("factoring_submission", row.original.id)}
          className="font-mono font-medium hover:underline"
        >
          {row.original.number}
        </Link>
      ),
      size: 150,
      minSize: 120,
      maxSize: 200,
      meta: { label: t("Schedule"), apiField: "number", sortable: true },
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
      accessorKey: "status",
      header: t("Status"),
      cell: ({ row }) => <FactoringSubmissionStatusBadge status={row.original.status} />,
      size: 140,
      minSize: 120,
      maxSize: 180,
      meta: { label: t("Status"), apiField: "status", sortable: true },
    },
    {
      accessorKey: "invoiceCount",
      header: t("Invoices"),
      cell: ({ row }) => <span className="tabular-nums">{row.original.invoiceCount}</span>,
      size: 100,
      minSize: 80,
      maxSize: 130,
      meta: { label: t("Invoices"), apiField: "invoiceCount", sortable: true },
    },
    {
      accessorKey: "faceMinor",
      header: t("Face value"),
      cell: ({ row }) => (
        <span className="tabular-nums">
          {formatMinor(row.original.faceMinor, row.original.currencyCode)}
        </span>
      ),
      size: 140,
      minSize: 120,
      maxSize: 180,
      meta: { label: t("Face value"), apiField: "faceMinor", sortable: true },
    },
    {
      accessorKey: "expectedAdvanceMinor",
      header: t("Expected advance"),
      cell: ({ row }) => (
        <span className="tabular-nums">
          {formatMinor(row.original.expectedAdvanceMinor, row.original.currencyCode)}
        </span>
      ),
      size: 150,
      minSize: 120,
      maxSize: 190,
      meta: { label: t("Expected advance"), apiField: "expectedAdvanceMinor", sortable: true },
    },
    {
      id: "delivery",
      header: t("Sent to"),
      cell: ({ row }) => (
        <span className="text-muted-foreground truncate">
          {FACTORING_TRANSPORT_METHOD_LABELS[row.original.transportMethod]} ·{" "}
          {row.original.transportTarget}
        </span>
      ),
      size: 260,
      minSize: 180,
      maxSize: 400,
      meta: { label: t("Sent to") },
    },
    {
      accessorKey: "createdAt",
      header: t("Submitted"),
      cell: ({ row }) => formatUnixDateTimeShort(row.original.createdAt),
      size: 160,
      minSize: 130,
      maxSize: 200,
      meta: {
        label: t("Submitted"),
        apiField: "createdAt",
        sortable: true,
        filterable: true,
        filterType: "date",
      },
    },
  ];
}
