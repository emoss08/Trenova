/* eslint-disable react-refresh/only-export-components */
import { DataTableDescription } from "@/components/data-table/_components/data-table-components";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { Badge } from "@/components/ui/badge";
import { fieldSensitivityChoices } from "@/lib/choices";
import type { FieldSensitivity, Role } from "@/types/role";
import { type ColumnDef } from "@tanstack/react-table";

function SensitivityBadge({ sensitivity }: { sensitivity: FieldSensitivity }) {
  const choice = fieldSensitivityChoices.find((c) => c.value === sensitivity);
  return (
    <Badge variant={choice?.variant ?? "info"} className="font-normal">
      {choice?.label ?? sensitivity}
    </Badge>
  );
}

export function getColumns(): ColumnDef<Role>[] {
  return [
    {
      accessorKey: "name",
      header: "Name",
      cell: ({ row }) => (
        <div className="flex flex-col gap-1">
          <span className="font-medium">{row.original.name}</span>
        </div>
      ),
      meta: {
        label: "Name",
        apiField: "name",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
      size: 250,
      minSize: 180,
      maxSize: 350,
    },
    {
      accessorKey: "description",
      header: "Description",
      cell: ({ row }) => (
        <DataTableDescription
          description={row.original.description}
          truncateLength={60}
        />
      ),
      meta: {
        label: "Description",
        apiField: "description",
        filterable: true,
        sortable: false,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
      size: 300,
      minSize: 200,
      maxSize: 450,
    },
    {
      accessorKey: "maxSensitivity",
      header: "Max Sensitivity",
      cell: ({ row }) => (
        <SensitivityBadge sensitivity={row.original.maxSensitivity} />
      ),
      meta: {
        label: "Max Sensitivity",
        apiField: "maxSensitivity",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: fieldSensitivityChoices,
        defaultFilterOperator: "eq",
      },
      size: 150,
      minSize: 120,
      maxSize: 180,
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
      size: 180,
      minSize: 150,
      maxSize: 220,
    },
  ];
}
