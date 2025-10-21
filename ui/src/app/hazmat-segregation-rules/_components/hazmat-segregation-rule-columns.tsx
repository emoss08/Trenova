import {
  DataTableDescription,
  HoverCardTimestamp,
} from "@/components/data-table/_components/data-table-components";
import { StatusBadge } from "@/components/status-badge";
import {
  hazardousClassChoices,
  segregationTypeChoices,
  statusChoices,
} from "@/lib/choices";
import { type HazmatSegregationRuleSchema } from "@/lib/schemas/hazmat-segregation-rule-schema";
import {
  HazardousClassChoiceProps,
  mapToHazardousClassChoice,
} from "@/types/hazardous-material";
import { type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<HazmatSegregationRuleSchema>[] {
  return [
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => {
        const status = row.original.status;
        return <StatusBadge status={status} />;
      },
      size: 120,
      minSize: 100,
      maxSize: 150,
      meta: {
        apiField: "status",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: statusChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "name",
      header: "Name",
      cell: ({ row }) => <p>{row.original.name}</p>,
      meta: {
        apiField: "name",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "classA",
      header: "Class A",
      enableResizing: true,
      cell: ({ row }) =>
        mapToHazardousClassChoice(
          row.original.classA as HazardousClassChoiceProps,
        ),
      meta: {
        apiField: "classA",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: hazardousClassChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "classB",
      header: "Class B",
      enableResizing: true,
      cell: ({ row }) =>
        mapToHazardousClassChoice(
          row.original.classB as HazardousClassChoiceProps,
        ),
      meta: {
        apiField: "classB",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: hazardousClassChoices,
        defaultFilterOperator: "eq",
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
      accessorKey: "segregationType",
      header: "Segregation Type",
      meta: {
        apiField: "segregationType",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: segregationTypeChoices,
        defaultFilterOperator: "eq",
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
