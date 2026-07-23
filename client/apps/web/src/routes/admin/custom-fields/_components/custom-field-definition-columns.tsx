import { DataTableColorColumn } from "@/components/data-table/_components/data-table-components";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { Badge, type BadgeVariant } from "@/components/ui/badge";
import { fieldTypeChoices } from "@/lib/choices";
import type { CustomFieldDefinition, FieldType } from "@/types/custom-field";
import type { ColumnDef } from "@tanstack/react-table";

const fieldTypeBadgeVariants: Record<FieldType, BadgeVariant> = {
  text: "info",
  number: "teal",
  date: "purple",
  boolean: "warning",
  select: "orange",
  multiSelect: "pink",
};

export function getColumns(): ColumnDef<CustomFieldDefinition>[] {
  return [
    {
      accessorKey: "label",
      header: "Label",
      cell: ({ row }) => {
        const { color, label } = row.original;
        return <DataTableColorColumn text={label} color={color} />;
      },
      meta: {
        label: "Label",
        apiField: "label",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "name",
      header: "Name",
      meta: {
        label: "Name",
        apiField: "name",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "resourceType",
      header: "Resource Type",
      cell: ({ row }) => (
        <Badge variant="outline" className="capitalize">
          {row.original.resourceType}
        </Badge>
      ),
      meta: {
        label: "Resource Type",
        apiField: "resourceType",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "fieldType",
      header: "Field Type",
      cell: ({ row }) => {
        const fieldType = row.original.fieldType;
        const choice = fieldTypeChoices.find((c) => c.value === fieldType);
        const variant = fieldTypeBadgeVariants[fieldType] || "default";
        return <Badge variant={variant}>{choice?.label || fieldType}</Badge>;
      },
      meta: {
        label: "Field Type",
        apiField: "fieldType",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: fieldTypeChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "isRequired",
      header: "Required",
      cell: ({ row }) => (
        <Badge variant={row.original.isRequired ? "active" : "inactive"}>
          {row.original.isRequired ? "Yes" : "No"}
        </Badge>
      ),
      size: 100,
    },
    {
      accessorKey: "isActive",
      header: "Active",
      cell: ({ row }) => (
        <Badge variant={row.original.isActive ? "active" : "inactive"}>
          {row.original.isActive ? "Active" : "Inactive"}
        </Badge>
      ),
      size: 100,
    },
    {
      accessorKey: "createdAt",
      header: "Created At",
      cell: ({ row }) => (
        <HoverCardTimestamp timestamp={row.original.createdAt} />
      ),
      meta: {
        label: "Created At",
        apiField: "createdAt",
        filterable: false,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
      size: 200,
    },
  ];
}
