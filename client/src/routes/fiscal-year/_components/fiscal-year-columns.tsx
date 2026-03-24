import {
  DataTableColorColumn,
  DataTableDescription,
} from "@/components/data-table/_components/data-table-components";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { fiscalYearStatusChoices } from "@/lib/choices";
import { formatToUserTimezone } from "@/lib/date";
import type { FiscalYear } from "@/types/fiscal-year";
import type { ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<FiscalYear>[] {
  return [
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => {
        const choice = fiscalYearStatusChoices.find(
          (c) => c.value === row.original.status,
        );
        if (!choice) return row.original.status;
        return (
          <DataTableColorColumn text={choice.label} color={choice.color} />
        );
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
      header: "Year",
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
      header: "Name",
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
      header: "Date Range",
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
      header: "Description",
      cell: ({ row }) => (
        <DataTableDescription
          description={row.original.description}
          truncateLength={100}
        />
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
      header: "Created At",
      cell: ({ row }) => (
        <HoverCardTimestamp timestamp={row.original.createdAt} />
      ),
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
