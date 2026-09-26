import { DataTableDescription } from "@/components/data-table/_components/data-table-components";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import type { ExtractionEvalCaseRow } from "@/lib/graphql/extraction-eval";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { CaseStatusBadge } from "./extraction-badges";
import { CASE_STATUS, choicesOf, documentKindLabel } from "./extraction-model";

export function getCaseColumns(t: TranslateFn): ColumnDef<ExtractionEvalCaseRow>[] {
  return [
    {
      accessorKey: "title",
      header: t("Case"),
      cell: ({ row }) => (
        <DataTableDescription description={row.original.title} truncateLength={80} />
      ),
      size: 320,
      meta: {
        label: t("Case"),
        apiField: "title",
        filterable: true,
        sortable: true,
        filterType: "text",
      },
    },
    {
      accessorKey: "status",
      header: t("Status"),
      cell: ({ row }) => <CaseStatusBadge value={row.original.status} t={t} />,
      size: 120,
      meta: {
        label: t("Status"),
        apiField: "status",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: choicesOf(CASE_STATUS),
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "documentKind",
      header: t("Document"),
      cell: ({ row }) => documentKindLabel(row.original.documentKind, t),
      size: 170,
      meta: {
        label: t("Document"),
        apiField: "documentKind",
        filterable: true,
        sortable: true,
        filterType: "text",
      },
    },
    {
      accessorKey: "expectedFieldCount",
      header: t("Confirmed fields"),
      cell: ({ row }) => <span className="tabular-nums">{row.original.expectedFieldCount}</span>,
      size: 140,
      meta: {
        label: t("Confirmed fields"),
        apiField: "expectedFieldCount",
        filterable: true,
        sortable: true,
        filterType: "number",
      },
    },
    {
      accessorKey: "pageCount",
      header: t("Pages"),
      cell: ({ row }) => <span className="tabular-nums">{row.original.pageCount}</span>,
      size: 90,
      meta: {
        label: t("Pages"),
        apiField: "pageCount",
        filterable: true,
        sortable: true,
        filterType: "number",
      },
    },
    {
      accessorKey: "createdAt",
      header: t("Added"),
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.createdAt} />,
      size: 170,
      meta: {
        label: t("Added"),
        apiField: "createdAt",
        filterable: true,
        sortable: true,
        filterType: "date",
      },
    },
  ];
}
