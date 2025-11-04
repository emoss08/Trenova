import { HoverCardTimestamp } from "@/components/data-table/_components/data-table-components";
import {
  FiscalPeriodStatusBadge,
  FiscalPeriodTypeBadge,
} from "@/components/status-badge";
import {
  fiscalPeriodStatusChoices,
  fiscalPeriodTypeChoices,
} from "@/lib/choices";
import { FiscalPeriodSchema } from "@/lib/schemas/fiscal-period-schema";
import { DashIcon } from "@radix-ui/react-icons";
import { type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<FiscalPeriodSchema>[] {
  return [
    {
      accessorKey: "periodNumber",
      header: "Period",
      cell: ({ row }) => {
        const { periodNumber } = row.original;
        return <p>{periodNumber || "-"}</p>;
      },
      meta: {
        apiField: "periodNumber",
        filterable: true,
        sortable: true,
        filterType: "number",
        defaultFilterOperator: "eq",
      },
      size: 100,
      minSize: 100,
      maxSize: 150,
    },
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => {
        const status = row.original.status;
        return <FiscalPeriodStatusBadge status={status} />;
      },
      size: 120,
      minSize: 100,
      maxSize: 150,
      meta: {
        apiField: "status",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: fiscalPeriodStatusChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "periodType",
      header: "Type",
      cell: ({ row }) => {
        const type = row.original.periodType;
        return <FiscalPeriodTypeBadge type={type} />;
      },
      size: 120,
      minSize: 100,
      maxSize: 150,
      meta: {
        apiField: "periodType",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: fiscalPeriodTypeChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "startDate",
      header: "Date Range",
      cell: ({ row }) => {
        const { startDate, endDate } = row.original;
        return (
          <div className="flex flex-row gap-0.5 cursor-default items-center justify-start">
            <HoverCardTimestamp
              timestamp={startDate}
              showTime={false}
              className="underline hover:no-underline decoration-dotted"
            />
            <DashIcon />
            <HoverCardTimestamp
              timestamp={endDate}
              showTime={false}
              className="underline hover:no-underline decoration-dotted"
            />
          </div>
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
