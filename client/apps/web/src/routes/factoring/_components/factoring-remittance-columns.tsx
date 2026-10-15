import { FactoringRemittanceStatusBadge } from "@/components/factoring/factoring-badges";
import { recordPath } from "@/config/record-links";
import { FACTORING_REMITTANCE_LINE_KIND_LABELS } from "@/lib/factoring-remittance";
import type { FactoringRemittanceRow } from "@/lib/graphql/factoring-remittance";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatUnixDate, formatUnixDateTimeShort } from "@trenova/shared/lib/date";
import { formatCurrency } from "@trenova/shared/lib/utils";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { Link } from "react-router";

export function getFactoringRemittanceColumns(t: TranslateFn): ColumnDef<FactoringRemittanceRow>[] {
  return [
    {
      accessorKey: "fileName",
      header: t("Remittance"),
      cell: ({ row }) => (
        <Link
          to={recordPath("factoring_remittance", row.original.id)}
          className="font-medium hover:underline"
        >
          {row.original.fileName || t("Waiting for its file")}
        </Link>
      ),
      size: 240,
      minSize: 160,
      maxSize: 360,
      meta: { label: t("Remittance"), apiField: "fileName", sortable: true },
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
      cell: ({ row }) => <FactoringRemittanceStatusBadge status={row.original.status} />,
      size: 150,
      minSize: 120,
      maxSize: 190,
      meta: { label: t("Status"), apiField: "status", sortable: true },
    },
    {
      accessorKey: "defaultKind",
      header: t("Report of"),
      cell: ({ row }) => FACTORING_REMITTANCE_LINE_KIND_LABELS[row.original.defaultKind],
      size: 140,
      minSize: 110,
      maxSize: 180,
      meta: { label: t("Report of"), apiField: "defaultKind", sortable: true },
    },
    {
      accessorKey: "lineCount",
      header: t("Lines"),
      cell: ({ row }) => <span className="tabular-nums">{row.original.lineCount}</span>,
      size: 90,
      minSize: 70,
      maxSize: 120,
      meta: { label: t("Lines"), apiField: "lineCount", sortable: true },
    },
    {
      accessorKey: "exceptionCount",
      header: t("To resolve"),
      cell: ({ row }) => <span className="tabular-nums">{row.original.exceptionCount}</span>,
      size: 110,
      minSize: 90,
      maxSize: 140,
      meta: { label: t("To resolve"), apiField: "exceptionCount", sortable: true },
    },
    {
      accessorKey: "totalCashMinor",
      header: t("Cash"),
      cell: ({ row }) => (
        <span className="tabular-nums">
          {formatCurrency(row.original.totalCashMinor / 100, row.original.currencyCode)}
        </span>
      ),
      size: 140,
      minSize: 110,
      maxSize: 180,
      meta: { label: t("Cash"), apiField: "totalCashMinor", sortable: true },
    },
    {
      accessorKey: "remittedOn",
      header: t("Remitted"),
      cell: ({ row }) => (row.original.remittedOn ? formatUnixDate(row.original.remittedOn) : ""),
      size: 130,
      minSize: 110,
      maxSize: 160,
      meta: { label: t("Remitted"), apiField: "remittedOn", sortable: true },
    },
    {
      accessorKey: "createdAt",
      header: t("Imported"),
      cell: ({ row }) => formatUnixDateTimeShort(row.original.createdAt),
      size: 160,
      minSize: 130,
      maxSize: 200,
      meta: { label: t("Imported"), apiField: "createdAt", sortable: true },
    },
  ];
}
