import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import {
  DataTableColorColumn,
  DataTableDescription,
} from "@/components/data-table/_components/data-table-components";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { fiscalYearStatusChoices } from "@/lib/choices";
import { formatToUserTimezone } from "@trenova/shared/lib/date";
import type { FiscalYearRow } from "@/lib/graphql/fiscal-year-table";
import type { ColumnDef } from "@trenova/shared/types/data-table";

export function getColumns(t: TranslateFn): ColumnDef<FiscalYearRow>[] {
  return [
    {
      accessorKey: "status",
      header: t("Status"),
      cell: ({ row }) => {
        const choice = fiscalYearStatusChoices.find((c) => c.value === row.original.status);
        if (!choice) return row.original.status;
        return <DataTableColorColumn text={t(choice.label)} color={choice.color} />;
      },
      size: 120,
      minSize: 100,
      maxSize: 150,
      meta: {
        apiField: "status",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: fiscalYearStatusChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "year",
      header: t("Year"),
      cell: ({ row }) => row.original.year,
      size: 80,
      meta: {
        apiField: "year",
        filterable: true,
        sortable: true,
        filterType: "number",
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "name",
      header: t("Name"),
      cell: ({ row }) => row.original.name,
      meta: {
        apiField: "name",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "startDate",
      header: t("Date Range"),
      cell: ({ row }) => {
        const { startDate, endDate } = row.original;
        if (!startDate || !endDate) return "-";
        return (
          <span className="font-mono text-xs whitespace-nowrap">
            {formatToUserTimezone(startDate, {
              showTime: false,
              showDate: true,
            })}{" "}
            -{" "}
            {formatToUserTimezone(endDate, {
              showTime: false,
              showDate: true,
            })}
          </span>
        );
      },
      size: 250,
      minSize: 200,
      maxSize: 300,
      meta: {
        apiField: "startDate",
        filterable: false,
        sortable: true,
      },
    },
    {
      accessorKey: "description",
      header: t("Description"),
      cell: ({ row }) => (
        <DataTableDescription description={t(row.original.description)} truncateLength={100} />
      ),
      size: 400,
      minSize: 300,
      maxSize: 500,
      meta: {
        apiField: "description",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "createdAt",
      header: t("Created At"),
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.createdAt} />,
      meta: {
        apiField: "createdAt",
        filterable: false,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
      size: 200,
      minSize: 200,
      maxSize: 250,
    },
  ];
}
