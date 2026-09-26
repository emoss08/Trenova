import { recordPath } from "@/config/record-links";
import {
  JOURNAL_REVIEW_STATUSES,
  type JournalReviewLabels,
  type JournalReviewStatus,
} from "@/hooks/use-journal-review-labels";
import type { JournalReviewRow } from "@/lib/graphql/journal-review";
import { AmountDisplay } from "@trenova/shared/components/accounting/amount-display";
import { Badge } from "@trenova/shared/components/ui/badge";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { phaseTone } from "@trenova/shared/lib/status-phase";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { Link } from "react-router";
import { journalReviewPhase } from "./review-filters";

export function getJournalReviewColumns(
  t: TranslateFn,
  labels: JournalReviewLabels,
): ColumnDef<JournalReviewRow>[] {
  return [
    {
      accessorKey: "status",
      header: t("Status"),
      cell: ({ row }) => (
        <Badge variant={phaseTone(journalReviewPhase(row.original.status))}>
          {labels.status[row.original.status as JournalReviewStatus] ?? row.original.status}
        </Badge>
      ),
      size: 160,
      minSize: 130,
      maxSize: 190,
      meta: {
        label: t("Status"),
        apiField: "status",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: JOURNAL_REVIEW_STATUSES.map((value) => ({
          value,
          label: labels.status[value],
        })),
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "entryNumber",
      header: t("Entry"),
      cell: ({ row }) => (
        <Link
          to={recordPath("journal_entry", row.original.id)}
          onClick={(event) => event.stopPropagation()}
          className="font-mono text-xs hover:underline"
        >
          {row.original.entryNumber}
        </Link>
      ),
      size: 150,
      minSize: 120,
      maxSize: 190,
      meta: {
        label: t("Entry"),
        apiField: "entryNumber",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "accountingDate",
      header: t("Accounting date"),
      cell: ({ row }) => formatUnixDateMedium(row.original.accountingDate, { fallback: "—" }),
      size: 150,
      minSize: 120,
      maxSize: 180,
      meta: {
        label: t("Accounting date"),
        apiField: "accountingDate",
        filterable: true,
        sortable: true,
        filterType: "date",
      },
    },
    {
      accessorKey: "referenceType",
      header: t("Source"),
      cell: ({ row }) => labels.source(row.original.referenceType),
      size: 200,
      minSize: 150,
      maxSize: 240,
      meta: { label: t("Source"), apiField: "referenceType", sortable: true },
    },
    {
      accessorKey: "referenceNumber",
      header: t("Document"),
      cell: ({ row }) => <span className="font-mono text-xs">{row.original.referenceNumber}</span>,
      size: 160,
      minSize: 120,
      maxSize: 200,
      meta: {
        label: t("Document"),
        apiField: "referenceNumber",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "description",
      header: t("Description"),
      cell: ({ row }) => <span className="truncate">{row.original.description}</span>,
      size: 260,
      minSize: 180,
      maxSize: 400,
      meta: { label: t("Description"), apiField: "description" },
    },
    {
      accessorKey: "totalDebit",
      header: t("Amount"),
      cell: ({ row }) => <AmountDisplay value={row.original.totalDebit} />,
      size: 140,
      minSize: 110,
      maxSize: 170,
      meta: { label: t("Amount"), apiField: "totalDebit", sortable: true },
    },
  ];
}
