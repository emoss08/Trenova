import {
  DataTableDescription,
  HoverCardTimestamp,
} from "@/components/data-table/_components/data-table-components";
import { FiscalYearStatusBadge } from "@/components/status-badge";
import { fiscalYearStatusChoices } from "@/lib/choices";
import { toDate } from "@/lib/date";
import { FiscalYearSchema } from "@/lib/schemas/fiscal-year-schema";
import { type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<FiscalYearSchema>[] {
  return [
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => {
        const status = row.original.status;
        return <FiscalYearStatusBadge status={status} />;
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
      accessorKey: "startDate",
      header: "Name",
      cell: ({ row }) => {
        const { startDate, endDate } = row.original;
        return (
          <p>
            {toDate(startDate)?.toLocaleDateString()} -{" "}
            {toDate(endDate)?.toLocaleDateString()}
          </p>
        );
      },
      meta: {
        apiField: "name",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
      size: 200,
      minSize: 200,
      maxSize: 250,
    },
    {
      accessorKey: "year",
      header: "Year",
      cell: ({ row }) => {
        const { year } = row.original;
        return <p>{year}</p>;
      },
      meta: {
        apiField: "year",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
      size: 100,
      minSize: 100,
      maxSize: 150,
    },
    {
      accessorKey: "name",
      header: "Name",
      cell: ({ row }) => {
        const { name } = row.original;
        return <p>{name}</p>;
      },
      meta: {
        apiField: "name",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
      size: 200,
      minSize: 200,
      maxSize: 250,
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
      cell: ({ row }) => {
        return <HoverCardTimestamp timestamp={row.original.createdAt} />;
      },
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
